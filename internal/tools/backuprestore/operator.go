package backuprestore

import (
	"context"
	"fmt"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	backupRestoreBackoffLimit int32 = 5
)

type Operator struct {
	kubeClient *k8s.KubeClient

	backupRestoreImage string
	awsRegion string
}

func (o Operator) TriggerBackup(installationID, storageEndpoint, fileStoreSecret, dbSecret string) error {

	// TODO: fetch Mattermost to get all the config?

	envVars := o.prepareEnvs(installationID, storageEndpoint, fileStoreSecret, dbSecret)

	job := o.createBackupRestoreJob(installationID, "backup", envVars)

	jobsClient := o.kubeClient.Clientset.BatchV1().Jobs(installationID)

	job, err := jobsClient.Create(context.Background(), job, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create backup job")
	}

	// take timestamp from job


	return nil
}

func (o Operator) CheckBackupStatus(installationID, storageEndpoint, fileStoreSecret, dbSecret string) error {

	// TODO: fetch Mattermost to get all the config?

	envVars := o.prepareEnvs(installationID, storageEndpoint, fileStoreSecret, dbSecret)

	job := o.createBackupRestoreJob(installationID, "backup", envVars)

	jobsClient := o.kubeClient.Clientset.BatchV1().Jobs(installationID)

	job, err := jobsClient.Create(context.Background(), job, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create backup job")
	}

	// take timestamp from job


	return nil
}



func (o Operator) createBackupRestoreJob(installationID, action string, envs []corev1.EnvVar) (*batchv1.Job) {
	backoff := backupRestoreBackoffLimit

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("database-%s"),
			Namespace: installationID,
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
				},
			},
			BackoffLimit: &backoff,
		},
	}

	return job
}

func (o Operator) createBackupRestoreContainer(action string, envs []corev1.EnvVar) corev1.Container {
	return corev1.Container{
		Name:                     "backup-restore",
		Image:                    o.backupRestoreImage,
		Command:                  []string{"backup-restore-tool", action},
		Args:                     []string{"storage-type", "bifrost"},
		Env: 		envs,
	}
}

func (o Operator) prepareEnvs(installationID, storageEndpoint, fileStoreSecret, dbSecret string) ([]corev1.EnvVar) {
	return []corev1.EnvVar{
		{
			Name: "BRT_STORAGE_REGION",
			Value: o.awsRegion,
		},
		{
			Name: "BRT_STORAGE_BUCKET",
			Value: installationID, // TODO: make sure it is correct
		},
		{
			Name: "BRT_STORAGE_ENDPOINT",
			Value: storageEndpoint,
		},
		{
			Name: "BRT_STORAGE_TLS",
			Value: "false", // TODO: should I do it based on storage endpoint?
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
