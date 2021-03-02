package provisioner

import (
	"context"
	"fmt"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/batch/v1"
	"strconv"
	"time"
)

const (
	backupRestoreBackoffLimit int32 = 5
)

// ErrJobBackoffLimitReached indicates that job failed all possible attempts and there is no reason for retrying.
var ErrJobBackoffLimitReached = errors.New("job reached backoff limit")

type Operator struct {
	provisioner *KopsProvisioner

	backupRestoreImage string
	awsRegion string
}

// TODO: create this from KopsProvisioner and do not pass Provisioner?
func NewBackupOperator(provisioner *KopsProvisioner, image string, region string) *Operator {
	return &Operator{
		provisioner:        provisioner,
		backupRestoreImage: image,
		awsRegion:          region,
	}
}

func (o Operator) TriggerBackup(backupMetadata *model.BackupMetadata, cluster *model.Cluster, installation *model.Installation) (*model.S3DataResidence, error) {
	logger := o.provisioner.logger.WithFields(log.Fields{
		"cluster":      cluster.ID,
		"installation": installation.ID,
		"backup":       backupMetadata.ID,
	})
	logger.Info("Triggering backup for installation")

	k8sClient, invalidateCache, err := o.provisioner.k8sClient(cluster.ProvisionerMetadataKops.Name, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create k8s client")
	}
	defer invalidateCache(err)

	filestoreCfg, filestoreSecret, err := o.provisioner.resourceUtil.GetFilestore(installation).
		GenerateFilestoreSpecAndSecret(o.provisioner.store, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get files store configuration for installation")
	}
	// Backup is not supported for local MinIO storage, therefore this should not happen
	if filestoreCfg == nil || filestoreSecret == nil {
		return nil, errors.New("file store secret and config cannot be empty for backup")
	}
	dbSecret, err := o.provisioner.resourceUtil.GetDatabase(installation).GenerateDatabaseSecret(o.provisioner.store, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database configuration")
	}
	// Backup is not supported for local MySQL, therefore this should not happen
	if dbSecret == nil {
		return nil, errors.New("database secret cannot be empty for backup")
	}

	// TODO: fetch Mattermost to get all the config? Or do it by getting all resources from provi stuff?

	//name := makeClusterInstallationName(clusterInstallation)
	//mmClient := k8sClient.MattermostClientsetV1Beta.MattermostV1beta1().Mattermosts(clusterInstallation.InstallationID)
	//
	//mattermostCR, err := mmClient.Get(ctx, name, metav1.GetOptions{})
	//if err != nil {
	//	return errors.Wrap(err, "failed to get Mattermost CR")
	//}
	jobsClient := k8sClient.Clientset.BatchV1().Jobs(installation.ID)

	return o.triggerBackup(jobsClient, backupMetadata, installation, filestoreCfg, filestoreSecret.Name, dbSecret.Name, logger)
}

func (o Operator) triggerBackup(
	jobsClient v1.JobInterface,
	backupMetadata *model.BackupMetadata,
	installation *model.Installation,
	fileStoreCfg *model.FilestoreConfig,
	fileStoreSecret, dbSecret string,
	logger log.FieldLogger,) (*model.S3DataResidence, error) {

	storageObjectKey := backupObjectKey(backupMetadata.ID, backupMetadata.RequestAt)

	envVars := o.prepareEnvs(fileStoreCfg.URL, fileStoreCfg.Bucket, storageObjectKey, fileStoreSecret, dbSecret)

	if installation.Filestore == model.InstallationFilestoreBifrost {
		bifrostEnv := bifrostEnvs(envVars)
		envVars = append(envVars, bifrostEnv...)
	}

	job := o.createBackupRestoreJob(backupMetadata.ID, installation.ID, "backup", envVars)

	ctx := context.Background()
	job, err := jobsClient.Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		if !k8sErrors.IsAlreadyExists(err) {
			return nil, errors.Wrap(err, "failed to create backup job")
		}
		logger.Warn("Backup job already exists")
	}

	dataResidence := &model.S3DataResidence{
		Region: o.awsRegion,
		Bucket: fileStoreCfg.Bucket,
		URL:    fileStoreCfg.URL,
		ObjectKey: storageObjectKey,
	}

	return dataResidence, nil
}

// CheckBackupStatus checks status of running backup job,
// returns job start time, when the job finished or -1 if it is still running.
func (o Operator) CheckBackupStatus(backupMetadata *model.BackupMetadata, cluster *model.Cluster) (int64, error) {
	logger := o.provisioner.logger.WithFields(log.Fields{
		"cluster":      cluster.ID,
		"installation": backupMetadata.InstallationID,
		"backup":       backupMetadata.ID,
	})
	logger.Info("Checking backup status for installation")

	k8sClient, invalidateCache, err := o.provisioner.k8sClient(cluster.ProvisionerMetadataKops.Name, logger)
	if err != nil {
		return -1, errors.Wrap(err, "failed to create k8s client")
	}
	defer invalidateCache(err)

	jobsClient := k8sClient.Clientset.BatchV1().Jobs(backupMetadata.InstallationID)

	return o.checkBackupStatus(jobsClient, backupMetadata, logger)
}

func (o Operator) checkBackupStatus(jobsClient v1.JobInterface, backupMetadata *model.BackupMetadata, logger log.FieldLogger) (int64, error) {
	ctx := context.Background()
	job, err := jobsClient.Get(ctx, jobName("backup", backupMetadata.ID), metav1.GetOptions{})
	if err != nil {
		return -1, errors.Wrap(err, "failed to get backup job")
	}

	// TODO: move it to function - finalizeBackup
	if job.Status.Succeeded > 0 {
		// TODO: what else on success? Cleanup?

		logger.Info("Backup finished with success")

		var startTime int64
		if job.Status.StartTime == nil {
			logger.Warn("failed to get job start time, using creation timestamp")
			startTime = asMillis(job.CreationTimestamp)
		} else {
			startTime = asMillis(*job.Status.StartTime)
		}

		return startTime, nil
	}

	if job.Status.Failed > 0 {
		logger.Warnf("Backup job failed %d times", job.Status.Failed)
	}

	if job.Status.Active > 0 {
		logger.Info("Backup job is still running")
		return  -1, nil
	}

	if job.Status.Failed == 0 {
		logger.Info("Backup job not running yet")
		return  -1, nil
	}

	backoffLimit :=  getInt32(job.Spec.BackoffLimit)
	if job.Status.Failed >= backoffLimit {
		logger.Error("Backup job reached backoff limit")
		return  -1, ErrJobBackoffLimitReached
	}

	logger.Infof("Backup job waiting for retry, will be retried at most %d more times", backoffLimit-job.Status.Failed)
	return  -1, nil
}

func (o Operator) createBackupRestoreJob(backupId, namespace, action string, envs []corev1.EnvVar) (*batchv1.Job) {
	backoff := backupRestoreBackoffLimit

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName(action, backupId),
			Namespace: namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "backup-restore"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						o.createBackupRestoreContainer(action, envs),
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
			BackoffLimit: &backoff,
			//TTLSecondsAfterFinished: int32Ptr(), // TODO: add job TTL?
		},
	}

	return job
}

func (o Operator) createBackupRestoreContainer(action string, envs []corev1.EnvVar) corev1.Container {
	return corev1.Container{
		Name:                     "backup-restore",
		Image:                    o.backupRestoreImage,
		Args:                     []string{action},
		Env: 		envs,
	}
}

func (o Operator) prepareEnvs(storageEndpoint, bucket, objectKey, fileStoreSecret, dbSecret string) []corev1.EnvVar {
	envs := []corev1.EnvVar{
		{
			Name: "BRT_STORAGE_REGION",
			Value: o.awsRegion,
		},
		{
			Name: "BRT_STORAGE_BUCKET",
			Value: bucket,
		},
		{
			Name: "BRT_STORAGE_ENDPOINT",
			Value: storageEndpoint, // TODO: this also needs to change for Bifrost
		},
		{
			Name: "BRT_STORAGE_OBJECT_KEY",
			Value: objectKey,
		},
		{
			Name: "BRT_DATABASE",
			ValueFrom: envSourceFromSecret(dbSecret, "DB_CONNECTION_STRING"),
		},
		{
			Name: "BRT_STORAGE_ACCESS_KEY",
			ValueFrom: envSourceFromSecret(fileStoreSecret, "accesskey"),
		},
		{
			Name: "BRT_STORAGE_SECRET_KEY",
			ValueFrom: envSourceFromSecret(fileStoreSecret, "secretkey"),
		},
	}

	return envs
}

func bifrostEnvs(envs []corev1.EnvVar) []corev1.EnvVar {
	// TODO: remove this part if you align with gabe, or change it?
	for i, e := range envs {
		if e.Name == "BRT_STORAGE_ENDPOINT" {
			envs[i].Value = "bifrost.bifrost:80"
			break
		}
	}

	return []corev1.EnvVar{
		{
			Name: "BRT_STORAGE_TLS",
			Value: strconv.FormatBool(false),
		},
		{
			Name:      "BRT_STORAGE_TYPE",
			Value:     "bifrost",
		},
	}
}

func envSourceFromSecret(secretName, key string) *corev1.EnvVarSource {
	return &corev1.EnvVarSource{
		SecretKeyRef: &corev1.SecretKeySelector{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: secretName,
			},
			Key: key,
		},
	}
}

func backupObjectKey(id string, timestamp int64) string {
	return fmt.Sprintf("backup-%s", id) // TODO: use time here?
}

func jobName(action, id string) string {
	return fmt.Sprintf("database-%s-%s", action, id)
}

func getInt32(i32 *int32) int32 {
	if i32 == nil {
		return 0
	}
	return *i32
}

// asMillis returns time.Time as milliseconds.
func asMillis(t metav1.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}
