package provisioner

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/batch/v1"
)

const (
	// Run job with only one attempt to avoid possibility of waking up workspace before retry.
	backupRestoreBackoffLimit int32 = 0
)

// ErrJobBackoffLimitReached indicates that job failed all possible attempts and there is no reason for retrying.
var ErrJobBackoffLimitReached = errors.New("job reached backoff limit")

type BackupOperator struct {
	jobTTLSecondsAfterFinish *int32
	backupRestoreImage       string
	awsRegion                string
}

func NewBackupOperator(image, region string, jobTTLSeconds int32) *BackupOperator {
	jobTTL := &jobTTLSeconds
	if jobTTLSeconds < 0 {
		jobTTL = nil
	}

	return &BackupOperator{
		jobTTLSecondsAfterFinish: jobTTL,
		backupRestoreImage:       image,
		awsRegion:                region,
	}
}

func (o BackupOperator) TriggerBackup(
	jobsClient v1.JobInterface,
	backupMetadata *model.BackupMetadata,
	installation *model.Installation,
	fileStoreCfg *model.FilestoreConfig,
	dbSecret string,
	logger log.FieldLogger) (*model.S3DataResidence, error) {

	storageObjectKey := backupObjectKey(backupMetadata.ID, backupMetadata.RequestAt)

	envVars := o.prepareEnvs(fileStoreCfg.URL, fileStoreCfg.Bucket, storageObjectKey, fileStoreCfg.Secret, dbSecret)

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

	// TODO: should we wait for backup job to start running?

	dataResidence := &model.S3DataResidence{
		Region:    o.awsRegion,
		Bucket:    fileStoreCfg.Bucket,
		URL:       fileStoreCfg.URL,
		ObjectKey: storageObjectKey,
	}

	return dataResidence, nil
}

func (o BackupOperator) CheckBackupStatus(jobsClient v1.JobInterface, backupMetadata *model.BackupMetadata, logger log.FieldLogger) (int64, error) {
	ctx := context.Background()
	job, err := jobsClient.Get(ctx, jobName("backup", backupMetadata.ID), metav1.GetOptions{})
	if err != nil {
		return -1, errors.Wrap(err, "failed to get backup job")
	}

	if job.Status.Succeeded > 0 {
		logger.Info("Backup finished with success")
		return o.extractStartTime(job, logger), nil
	}

	if job.Status.Failed > 0 {
		logger.Warnf("Backup job failed %d times", job.Status.Failed)
	}

	if job.Status.Active > 0 {
		logger.Info("Backup job is still running")
		return -1, nil
	}

	if job.Status.Failed == 0 {
		logger.Info("Backup job not started yet")
		return -1, nil
	}

	backoffLimit := getInt32(job.Spec.BackoffLimit)
	if job.Status.Failed > backoffLimit {
		logger.Error("Backup job reached backoff limit")
		return -1, ErrJobBackoffLimitReached
	}

	logger.Infof("Backup job waiting for retry, will be retried at most %d more times", backoffLimit+1-job.Status.Failed)
	return -1, nil
}

func (o BackupOperator) extractStartTime(job *batchv1.Job, logger log.FieldLogger) int64 {
	if job.Status.StartTime != nil {
		return asMillis(*job.Status.StartTime)
	}

	logger.Warn("failed to get job start time, using creation timestamp")
	return asMillis(job.CreationTimestamp)
}

func (o BackupOperator) createBackupRestoreJob(backupId, namespace, action string, envs []corev1.EnvVar) *batchv1.Job {
	backoff := backupRestoreBackoffLimit

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName(action, backupId),
			Namespace: namespace,
			Labels:    map[string]string{"app": "backup-restore"},
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
			BackoffLimit:            &backoff,
			TTLSecondsAfterFinished: o.jobTTLSecondsAfterFinish,
		},
	}

	return job
}

func (o BackupOperator) createBackupRestoreContainer(action string, envs []corev1.EnvVar) corev1.Container {
	return corev1.Container{
		Name:  "backup-restore",
		Image: o.backupRestoreImage,
		Args:  []string{action},
		Env:   envs,
		Resources: corev1.ResourceRequirements{
			// TODO: memory?
			Limits: corev1.ResourceList{
				corev1.ResourceEphemeralStorage: resource.MustParse("15Gi"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceEphemeralStorage: resource.MustParse("2Gi"),
			},
		},
	}
}

func (o BackupOperator) prepareEnvs(storageEndpoint, bucket, objectKey, fileStoreSecret, dbSecret string) []corev1.EnvVar {
	envs := []corev1.EnvVar{
		{
			Name:  "BRT_STORAGE_REGION",
			Value: o.awsRegion,
		},
		{
			Name:  "BRT_STORAGE_BUCKET",
			Value: bucket,
		},
		{
			Name:  "BRT_STORAGE_ENDPOINT",
			Value: storageEndpoint, // TODO: this also needs to change for Bifrost
		},
		{
			Name:  "BRT_STORAGE_OBJECT_KEY",
			Value: objectKey,
		},
		{
			Name:      "BRT_DATABASE",
			ValueFrom: envSourceFromSecret(dbSecret, "DB_CONNECTION_STRING"),
		},
		{
			Name:      "BRT_STORAGE_ACCESS_KEY",
			ValueFrom: envSourceFromSecret(fileStoreSecret, "accesskey"),
		},
		{
			Name:      "BRT_STORAGE_SECRET_KEY",
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
			Name:  "BRT_STORAGE_TLS",
			Value: strconv.FormatBool(false),
		},
		{
			Name:  "BRT_STORAGE_TYPE",
			Value: "bifrost",
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
