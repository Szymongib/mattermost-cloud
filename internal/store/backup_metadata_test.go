package store

import (
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIsBackupRunning(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation := &model.Installation{
		State:     model.InstallationStateStable,
	}

	err := sqlStore.CreateInstallation(installation, nil)
	require.NoError(t, err)

	running, err := sqlStore.IsBackupRunning(installation.ID)
	require.NoError(t, err)
	require.False(t, running)

	metadata := &model.BackupMetadata{
		InstallationID: installation.ID,
		State:          model.BackupStateBackupRequested,
	}

	err = sqlStore.CreateBackupMetadata(metadata)
	require.NoError(t, err)

	running, err = sqlStore.IsBackupRunning(installation.ID)
	require.NoError(t, err)
	require.True(t, running)

	// TODO: extend test with one stable and then one in progress
}

func TestCreateBackupMetadata(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation := &model.Installation{
		State:     model.InstallationStateStable,
	}

	err := sqlStore.CreateInstallation(installation, nil)
	require.NoError(t, err)

	metadata := &model.BackupMetadata{
		InstallationID: installation.ID,
		State:          model.BackupStateBackupRequested,
	}

	err = sqlStore.CreateBackupMetadata(metadata)
	require.NoError(t, err)
	assert.NotEmpty(t, metadata.ID)

	t.Run("fail to create backup metadata for installation when other is requested", func(t *testing.T) {
		newMetadata := &model.BackupMetadata{
			InstallationID: installation.ID,
			State:          model.BackupStateBackupRequested,
		}

		err = sqlStore.CreateBackupMetadata(newMetadata)
		require.Error(t, err)
	})
}

func TestGetUnlockedBackupMetadataPendingWork(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation := &model.Installation{
		State:     model.InstallationStateStable,
	}

	err := sqlStore.CreateInstallation(installation, nil)
	require.NoError(t, err)

	metadata1 := &model.BackupMetadata{
		InstallationID: installation.ID,
		State:          model.BackupStateBackupRequested,
	}

	err = sqlStore.CreateBackupMetadata(metadata1)
	require.NoError(t, err)
	assert.NotEmpty(t, metadata1.ID)

	metadata2 := &model.BackupMetadata{
		InstallationID: installation.ID,
		State:          model.BackupStateBackupSucceeded,
	}

	err = sqlStore.CreateBackupMetadata(metadata2)
	require.NoError(t, err)
	assert.NotEmpty(t, metadata1.ID)

	backupsMeta, err := sqlStore.GetUnlockedBackupMetadataPendingWork()
	require.NoError(t, err)
	assert.Equal(t, 1, len(backupsMeta))
	assert.Equal(t, metadata1.ID, backupsMeta[0].ID)

	locaked, err := sqlStore.LockBackupMetadata(metadata1.ID, "abc")
	require.NoError(t, err)
	assert.True(t, locaked)

	backupsMeta, err = sqlStore.GetUnlockedBackupMetadataPendingWork()
	require.NoError(t, err)
	assert.Equal(t, 0, len(backupsMeta))
}

func TestUpdateBackupMetadata(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation := &model.Installation{
		State:     model.InstallationStateStable,
	}

	err := sqlStore.CreateInstallation(installation, nil)
	require.NoError(t, err)

	metadata := &model.BackupMetadata{
		InstallationID: installation.ID,
		State:          model.BackupStateBackupRequested,
	}

	err = sqlStore.CreateBackupMetadata(metadata)
	require.NoError(t, err)
	assert.NotEmpty(t, metadata.ID)

	t.Run("update state only", func(t *testing.T) {
		metadata.State = model.BackupStateBackupSucceeded
		metadata.StartAt = -1

		err = sqlStore.UpdateBackupMetadataState(metadata)
		require.NoError(t, err)

		fetched, err := sqlStore.GetBackupMetadata(metadata.ID)
		require.NoError(t, err)
		assert.Equal(t, model.BackupStateBackupSucceeded, fetched.State)
		assert.Equal(t, int64(0), fetched.StartAt) // Assert start time not updated
		assert.Equal(t, "", fetched.ClusterInstallationID) // Assert CI ID not updated
	})

	t.Run("update data residency only", func(t *testing.T) {
		updatedResidence := &model.S3DataResidence{URL: "s3.amazon.com"}
		clusterInstallationID := "cluster-installation-1"

		metadata.StartAt = -1
		metadata.DataResidence = updatedResidence
		metadata.ClusterInstallationID = clusterInstallationID

		err = sqlStore.UpdateBackupSchedulingData(metadata)
		require.NoError(t, err)

		fetched, err := sqlStore.GetBackupMetadata(metadata.ID)
		require.NoError(t, err)
		assert.Equal(t, updatedResidence, fetched.DataResidence)
		assert.Equal(t, clusterInstallationID, fetched.ClusterInstallationID)
		assert.Equal(t, int64(0), fetched.StartAt) // Assert start time not updated
	})

	t.Run("update start time", func(t *testing.T) {
		var startTime int64 = 10000
		originalCIId := metadata.ClusterInstallationID

		metadata.StartAt = startTime
		metadata.ClusterInstallationID = "modified-ci-id"

		err = sqlStore.UpdateBackupStartTime(metadata)
		require.NoError(t, err)

		fetched, err := sqlStore.GetBackupMetadata(metadata.ID)
		require.NoError(t, err)
		assert.Equal(t, startTime, fetched.StartAt)
		assert.Equal(t, originalCIId, fetched.ClusterInstallationID) // Assert ClusterInstallationID not updated
	})
}
