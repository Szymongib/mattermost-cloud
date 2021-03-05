package provisioner

import (
	"context"
	"fmt"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"testing"
)

func TestOperator_TriggerBackup(t *testing.T) {
	fileStoreSecret := "file-store-secret"
	fileStoreCfg := &model.FilestoreConfig{
		URL:    "filestore.com",
		Bucket: "plastic",
		Secret: fileStoreSecret,
	}
	databaseSecret := "database-secret"

	backupMeta := &model.BackupMetadata{
		ID:             "backup-meta-1",
		InstallationID: "installation-1",
		State:          model.BackupStateBackupRequested,
		RequestAt:      1,
	}

	operator := Operator{
		provisioner:        nil,
		backupRestoreImage: "mattermost/backup-restore:test",
		awsRegion:          "us",
	}

	for _, testCase := range []struct {
		description          string
		installation         *model.Installation
		expectedFileStoreURL string
		extraEnvs            map[string]string
	}{
		{
			description:          "s3 installation",
			installation:         &model.Installation{ID: "installation-1", Filestore: model.InstallationFilestoreMultiTenantAwsS3},
			expectedFileStoreURL: "filestore.com",
		},
		{
			description:          "bifrost installation",
			installation:         &model.Installation{ID: "installation-1", Filestore: model.InstallationFilestoreBifrost},
			expectedFileStoreURL: "bifrost.bifrost:80",
			extraEnvs: map[string]string{
				"BRT_STORAGE_TLS":  "false",
				"BRT_STORAGE_TYPE": "bifrost",
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			k8sClient := fake.NewSimpleClientset()
			jobClinet := k8sClient.BatchV1().Jobs("installation-1")

			dataRes, err := operator.triggerBackup(
				jobClinet,
				backupMeta,
				testCase.installation,
				fileStoreCfg,
				fileStoreSecret,
				databaseSecret,
				logrus.New())
			require.NoError(t, err)

			assert.Equal(t, "filestore.com", dataRes.URL)
			assert.Equal(t, "us", dataRes.Region)
			assert.Equal(t, "plastic", dataRes.Bucket)
			assert.Equal(t, "backup-backup-meta-1", dataRes.ObjectKey)

			createdJob, err := jobClinet.Get(context.Background(), "database-backup-backup-meta-1", v1.GetOptions{})
			require.NoError(t, err)

			assert.Equal(t, "backup-restore", createdJob.Labels["app"])
			assert.Equal(t, "installation-1", createdJob.Namespace)
			assert.Equal(t, backupRestoreBackoffLimit, *createdJob.Spec.BackoffLimit)

			podTemplate := createdJob.Spec.Template
			assert.Equal(t, "backup-restore", podTemplate.Labels["app"])

			envs := createdJob.Spec.Template.Spec.Containers[0].Env
			assertEnvVarEqual(t, "BRT_STORAGE_REGION", "us", envs)
			assertEnvVarEqual(t, "BRT_STORAGE_BUCKET", "plastic", envs)
			assertEnvVarEqual(t, "BRT_STORAGE_ENDPOINT", testCase.expectedFileStoreURL, envs)
			assertEnvVarEqual(t, "BRT_STORAGE_OBJECT_KEY", "backup-backup-meta-1", envs)
			assertEnvVarFromSecret(t, "BRT_STORAGE_ACCESS_KEY", fileStoreSecret, "accesskey", envs)
			assertEnvVarFromSecret(t, "BRT_STORAGE_SECRET_KEY", fileStoreSecret, "secretkey", envs)
			assertEnvVarFromSecret(t, "BRT_DATABASE", databaseSecret, "DB_CONNECTION_STRING", envs)

			for k, v := range testCase.extraEnvs {
				assertEnvVarEqual(t, k, v, envs)
			}
		})
	}

	t.Run("succeed if job already exists", func(t *testing.T) {
		existing := &batchv1.Job{
			ObjectMeta: v1.ObjectMeta{Name: "database-backup-backup-meta-1", Namespace: "installation-1"},
		}
		k8sClient := fake.NewSimpleClientset(existing)
		jobClinet := k8sClient.BatchV1().Jobs("installation-1")

		installation := &model.Installation{ID: "installation-1", Filestore: model.InstallationFilestoreMultiTenantAwsS3}

		dataRes, err := operator.triggerBackup(
			jobClinet,
			backupMeta,
			installation,
			fileStoreCfg,
			fileStoreSecret,
			databaseSecret,
			logrus.New())
		require.NoError(t, err)

		assert.Equal(t, "filestore.com", dataRes.URL)
		assert.Equal(t, "us", dataRes.Region)
		assert.Equal(t, "plastic", dataRes.Bucket)
		assert.Equal(t, "backup-backup-meta-1", dataRes.ObjectKey)
	})
}

func TestOperator_CheckBackupStatus(t *testing.T) {
	backupMeta := &model.BackupMetadata{
		ID:             "backup-meta-1",
		InstallationID: "installation-1",
		State:          model.BackupStateBackupRequested,
		RequestAt:      1,
	}

	k8sClient := fake.NewSimpleClientset()
	jobClinet := k8sClient.BatchV1().Jobs("installation-1")

	operator := Operator{
		provisioner:        nil,
		backupRestoreImage: "mattermost/backup-restore:test",
		awsRegion:          "us",
	}

	t.Run("error when job does not exists", func(t *testing.T) {
		_, err := operator.checkBackupStatus(jobClinet, backupMeta, logrus.New())
		require.Error(t, err)
	})

	job := &batchv1.Job{
		ObjectMeta: v1.ObjectMeta{Name: "database-backup-backup-meta-1"},
		Spec:       batchv1.JobSpec{},
		Status:     batchv1.JobStatus{},
	}
	var err error
	job, err = jobClinet.Create(context.Background(), job, v1.CreateOptions{})
	require.NoError(t, err)

	t.Run("return -1 start time if not finished", func(t *testing.T) {
		startTime, err := operator.checkBackupStatus(jobClinet, backupMeta, logrus.New())
		require.NoError(t, err)
		assert.Equal(t, int64(-1), startTime)
	})

	job.Status.Failed = backupRestoreBackoffLimit
	job, err = jobClinet.Update(context.Background(), job, v1.UpdateOptions{})
	require.NoError(t, err)

	t.Run("ErrJobBackoffLimitReached when failed enough times", func(t *testing.T) {
		_, err = operator.checkBackupStatus(jobClinet, backupMeta, logrus.New())
		require.Error(t, err)
		assert.Equal(t, ErrJobBackoffLimitReached, err)
	})

	expectedStartTime := v1.Now()
	job.Status.Succeeded = 1
	job.Status.StartTime = &expectedStartTime
	job, err = jobClinet.Update(context.Background(), job, v1.UpdateOptions{})
	require.NoError(t, err)

	t.Run("return start time when succeeded", func(t *testing.T) {
		startTime, err := operator.checkBackupStatus(jobClinet, backupMeta, logrus.New())
		require.NoError(t, err)
		assert.Equal(t, asMillis(expectedStartTime), startTime)
	})
}

func assertEnvVarEqual(t *testing.T, name, val string, env []corev1.EnvVar) {
	for _, e := range env {
		if e.Name == name {
			assert.Equal(t, e.Value, val)
			return
		}
	}

	assert.Fail(t, fmt.Sprintf("failed to find env var %s", name))
}

func assertEnvVarFromSecret(t *testing.T, name, secret, key string, env []corev1.EnvVar) {
	for _, e := range env {
		if e.Name == name {
			valFrom := e.ValueFrom
			require.NotNil(t, valFrom)
			require.NotNil(t, valFrom.SecretKeyRef)
			assert.Equal(t, secret, valFrom.SecretKeyRef.Name)
			assert.Equal(t, key, valFrom.SecretKeyRef.Key)
			return
		}
	}

	assert.Fail(t, fmt.Sprintf("failed to find env var %s", name))
}
