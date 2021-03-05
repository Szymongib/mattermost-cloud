package store

import (
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestIsBackupRunning(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation := setupBasicInstallation(t, sqlStore)

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

	installation := setupBasicInstallation(t, sqlStore)

	metadata := &model.BackupMetadata{
		InstallationID: installation.ID,
		State:          model.BackupStateBackupRequested,
	}

	err := sqlStore.CreateBackupMetadata(metadata)
	require.NoError(t, err)
	assert.NotEmpty(t, metadata.ID)
}

func TestGetBackupMetadata(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation1 := setupBasicInstallation(t, sqlStore)
	installation2 := setupBasicInstallation(t, sqlStore)

	metadata1 := &model.BackupMetadata{
		InstallationID: installation1.ID,
		State:          model.BackupStateBackupRequested,
	}
	err := sqlStore.CreateBackupMetadata(metadata1)
	require.NoError(t, err)

	metadata2 := &model.BackupMetadata{
		InstallationID: installation2.ID,
		State:          model.BackupStateBackupRequested,
	}
	err = sqlStore.CreateBackupMetadata(metadata2)
	require.NoError(t, err)

	for _, testCase := range []struct {
		description    string
		id             string
		installationID string
		expectedMeta   *model.BackupMetadata
	}{
		{
			description:    "fetch with ID and Installation ID",
			id:             metadata1.ID,
			installationID: installation1.ID,
			expectedMeta:   metadata1,
		},
		{
			description:    "fetch with ID only",
			id:             metadata1.ID,
			installationID: "",
			expectedMeta:   metadata1,
		},
		{
			description:    "fail to fetch if no match",
			id:             metadata1.ID,
			installationID: installation2.ID,
			expectedMeta:   nil,
		},
		{
			description:    "fail to fetch with empty id",
			id:             "",
			installationID: installation2.ID,
			expectedMeta:   nil,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			fetchedMetadata, err := sqlStore.GetBackupMetadata(testCase.id, testCase.installationID)
			require.NoError(t, err)
			assert.Equal(t, testCase.expectedMeta, fetchedMetadata)
		})
	}
}

func TestGetBackupsMetadata(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation1 := setupBasicInstallation(t, sqlStore)
	installation2 := setupBasicInstallation(t, sqlStore)
	clusterInstallation := &model.ClusterInstallation{
		InstallationID: installation1.ID,
	}
	err := sqlStore.CreateClusterInstallation(clusterInstallation)
	require.NoError(t, err)

	backupsMeta := []*model.BackupMetadata{
		{InstallationID: installation1.ID, State: model.BackupStateBackupRequested, ClusterInstallationID: clusterInstallation.ID},
		{InstallationID: installation1.ID, State: model.BackupStateBackupInProgress, ClusterInstallationID: clusterInstallation.ID},
		{InstallationID: installation1.ID, State: model.BackupStateBackupFailed},
		{InstallationID: installation2.ID, State: model.BackupStateBackupRequested},
		{InstallationID: installation2.ID, State: model.BackupStateBackupInProgress},
	}

	for i := range backupsMeta {
		err := sqlStore.CreateBackupMetadata(backupsMeta[i])
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond) // Ensure RequestAt is different for all installations.
	}

	err = sqlStore.DeleteBackupMetadata(backupsMeta[2])
	require.NoError(t, err)

	for _, testCase := range []struct {
		description string
		filter      *model.BackupMetadataFilter
		fetchedIds  []string
	}{
		{
			description: "fetch all not deleted",
			filter:      &model.BackupMetadataFilter{PerPage: model.AllPerPage},
			fetchedIds:  []string{backupsMeta[4].ID, backupsMeta[3].ID, backupsMeta[1].ID, backupsMeta[0].ID},
		},
		{
			description: "fetch all for installation 1",
			filter:      &model.BackupMetadataFilter{InstallationID: installation1.ID, IncludeDeleted: true, PerPage: model.AllPerPage},
			fetchedIds:  []string{backupsMeta[2].ID, backupsMeta[1].ID, backupsMeta[0].ID},
		},
		{
			description: "fetch all for cluster installation ",
			filter:      &model.BackupMetadataFilter{ClusterInstallationID: clusterInstallation.ID, PerPage: model.AllPerPage},
			fetchedIds:  []string{backupsMeta[1].ID, backupsMeta[0].ID},
		},
		{
			description: "fetch for installation 1 without deleted",
			filter:      &model.BackupMetadataFilter{InstallationID: installation1.ID, IncludeDeleted: false, PerPage: model.AllPerPage},
			fetchedIds:  []string{backupsMeta[1].ID, backupsMeta[0].ID},
		},
		{
			description: "fetch requested installations",
			filter:      &model.BackupMetadataFilter{State: model.BackupStateBackupRequested, PerPage: model.AllPerPage},
			fetchedIds:  []string{backupsMeta[3].ID, backupsMeta[0].ID},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			fetchedMetadatas, err := sqlStore.GetBackupsMetadata(testCase.filter)
			require.NoError(t, err)
			assert.Equal(t, len(testCase.fetchedIds), len(fetchedMetadatas))

			for i, meta := range fetchedMetadatas {
				assert.Equal(t, testCase.fetchedIds[i], meta.ID)
			}
		})
	}
}

func TestGetUnlockedBackupMetadataPendingWork(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation := setupBasicInstallation(t, sqlStore)

	metadata1 := &model.BackupMetadata{
		InstallationID: installation.ID,
		State:          model.BackupStateBackupRequested,
	}

	err := sqlStore.CreateBackupMetadata(metadata1)
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

	installation := setupBasicInstallation(t, sqlStore)

	metadata := &model.BackupMetadata{
		InstallationID: installation.ID,
		State:          model.BackupStateBackupRequested,
	}

	err := sqlStore.CreateBackupMetadata(metadata)
	require.NoError(t, err)
	assert.NotEmpty(t, metadata.ID)

	t.Run("update state only", func(t *testing.T) {
		metadata.State = model.BackupStateBackupSucceeded
		metadata.StartAt = -1

		err = sqlStore.UpdateBackupMetadataState(metadata)
		require.NoError(t, err)

		fetched, err := sqlStore.GetBackupMetadata(metadata.ID, "")
		require.NoError(t, err)
		assert.Equal(t, model.BackupStateBackupSucceeded, fetched.State)
		assert.Equal(t, int64(0), fetched.StartAt)         // Assert start time not updated
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

		fetched, err := sqlStore.GetBackupMetadata(metadata.ID, "")
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

		fetched, err := sqlStore.GetBackupMetadata(metadata.ID, "")
		require.NoError(t, err)
		assert.Equal(t, startTime, fetched.StartAt)
		assert.Equal(t, originalCIId, fetched.ClusterInstallationID) // Assert ClusterInstallationID not updated
	})
}

func setupBasicInstallation(t *testing.T, sqlStore *SQLStore) *model.Installation {
	installation := &model.Installation{
		State: model.InstallationStateStable,
	}

	err := sqlStore.CreateInstallation(installation, nil)
	require.NoError(t, err)

	return installation
}
