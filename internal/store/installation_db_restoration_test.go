package store

import (
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestInstallationDBRestoration(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation := setupBasicInstallation(t, sqlStore)

	dbRestoration := &model.InstallationDBRestorationOperation{
		InstallationID:        installation.ID,
		BackupID:              "test",
		State:                 model.InstallationDBRestorationStateRequested,
		ClusterInstallationID: "",
		CompleteAt:            0,
	}

	err := sqlStore.CreateInstallationDBRestoration(dbRestoration)
	require.NoError(t, err)
	assert.NotEmpty(t, dbRestoration.ID)

	fetchedRestoration, err := sqlStore.GetInstallationDBRestoration(dbRestoration.ID)
	require.NoError(t, err)
	assert.Equal(t, dbRestoration, fetchedRestoration)

	t.Run("unknown restoration", func(t *testing.T) {
		fetchedRestoration, err = sqlStore.GetInstallationDBRestoration("unknown")
		require.NoError(t, err)
		assert.Nil(t, fetchedRestoration)
	})
}

func TestGetInstallationDBRestorations(t *testing.T) {
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

	dbRestorations := []*model.InstallationDBRestorationOperation{
		{InstallationID: installation1.ID, State: model.InstallationDBRestorationStateRequested, ClusterInstallationID: clusterInstallation.ID},
		{InstallationID: installation1.ID, State: model.InstallationDBRestorationStateInProgress, ClusterInstallationID: clusterInstallation.ID},
		{InstallationID: installation1.ID, State: model.InstallationDBRestorationStateFailed},
		{InstallationID: installation2.ID, State: model.InstallationDBRestorationStateRequested},
		{InstallationID: installation2.ID, State: model.InstallationDBRestorationStateInProgress},
	}

	for i := range dbRestorations {
		err := sqlStore.CreateInstallationDBRestoration(dbRestorations[i])
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond) // Ensure RequestAt is different for all installations.
	}

	for _, testCase := range []struct {
		description string
		filter      *model.InstallationDBRestorationFilter
		fetchedIds  []string
	}{
		{
			description: "fetch all",
			filter:      &model.InstallationDBRestorationFilter{Paging: model.AllPagesNotDeleted()},
			fetchedIds:  []string{dbRestorations[4].ID, dbRestorations[3].ID, dbRestorations[2].ID, dbRestorations[1].ID, dbRestorations[0].ID},
		},
		{
			description: "fetch all for installation 1",
			filter:      &model.InstallationDBRestorationFilter{InstallationID: installation1.ID, Paging: model.AllPagesNotDeleted()},
			fetchedIds:  []string{dbRestorations[2].ID, dbRestorations[1].ID, dbRestorations[0].ID},
		},
		{
			description: "fetch all for cluster installation ",
			filter:      &model.InstallationDBRestorationFilter{ClusterInstallationID: clusterInstallation.ID, Paging: model.AllPagesNotDeleted()},
			fetchedIds:  []string{dbRestorations[1].ID, dbRestorations[0].ID},
		},
		{
			description: "fetch requested installations",
			filter:      &model.InstallationDBRestorationFilter{States: []model.InstallationDBRestorationState{model.InstallationDBRestorationStateRequested}, Paging: model.AllPagesNotDeleted()},
			fetchedIds:  []string{dbRestorations[3].ID, dbRestorations[0].ID},
		},
		{
			description: "fetch with IDs",
			filter:      &model.InstallationDBRestorationFilter{IDs: []string{dbRestorations[0].ID, dbRestorations[3].ID, dbRestorations[4].ID}, Paging: model.AllPagesNotDeleted()},
			fetchedIds:  []string{dbRestorations[4].ID, dbRestorations[3].ID, dbRestorations[0].ID},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			fetchedBackups, err := sqlStore.GetInstallationDBRestorations(testCase.filter)
			require.NoError(t, err)
			assert.Equal(t, len(testCase.fetchedIds), len(fetchedBackups))

			for i, b := range fetchedBackups {
				assert.Equal(t, testCase.fetchedIds[i], b.ID)
			}
		})
	}
}

func TestGetUnlockedInstallationDBRestorationsPendingWork(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation := setupBasicInstallation(t, sqlStore)

	dbRestoration1 := &model.InstallationDBRestorationOperation{
		InstallationID: installation.ID,
		State:          model.InstallationDBRestorationStateRequested,
	}

	err := sqlStore.CreateInstallationDBRestoration(dbRestoration1)
	require.NoError(t, err)
	assert.NotEmpty(t, dbRestoration1.ID)

	dbRestoration2 := &model.InstallationDBRestorationOperation{
		InstallationID: installation.ID,
		State:          model.InstallationDBRestorationStateSucceeded,
	}

	err = sqlStore.CreateInstallationDBRestoration(dbRestoration2)
	require.NoError(t, err)
	assert.NotEmpty(t, dbRestoration1.ID)

	backupsMeta, err := sqlStore.GetUnlockedInstallationDBRestorationsPendingWork()
	require.NoError(t, err)
	assert.Equal(t, 1, len(backupsMeta))
	assert.Equal(t, dbRestoration1.ID, backupsMeta[0].ID)

	locaked, err := sqlStore.LockInstallationDBRestoration(dbRestoration1.ID, "abc")
	require.NoError(t, err)
	assert.True(t, locaked)

	backupsMeta, err = sqlStore.GetUnlockedInstallationDBRestorationsPendingWork()
	require.NoError(t, err)
	assert.Equal(t, 0, len(backupsMeta))
}

func TestUpdateInstallationDBRestoration(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation := setupBasicInstallation(t, sqlStore)

	dbRestoration := &model.InstallationDBRestorationOperation{
		InstallationID: installation.ID,
		State:          model.InstallationDBRestorationStateRequested,
	}

	err := sqlStore.CreateInstallationDBRestoration(dbRestoration)
	require.NoError(t, err)
	assert.NotEmpty(t, dbRestoration.ID)

	t.Run("update state only", func(t *testing.T) {
		dbRestoration.State = model.InstallationDBRestorationStateSucceeded
		dbRestoration.CompleteAt = -1

		err = sqlStore.UpdateInstallationDBRestorationState(dbRestoration)
		require.NoError(t, err)

		fetched, err := sqlStore.GetInstallationDBRestoration(dbRestoration.ID)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationDBRestorationStateSucceeded, fetched.State)
		assert.Equal(t, int64(0), fetched.CompleteAt)         // Assert complete time not updated
		assert.Equal(t, "", fetched.ClusterInstallationID) // Assert CI ID not updated
	})

	t.Run("full update", func(t *testing.T) {
		dbRestoration.ClusterInstallationID = "test"
		dbRestoration.CompleteAt = 100
		dbRestoration.State = model.InstallationDBRestorationStateFailed
		err = sqlStore.UpdateInstallationDBRestoration(dbRestoration)
		require.NoError(t, err)

		fetched, err := sqlStore.GetInstallationDBRestoration(dbRestoration.ID)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationDBRestorationStateFailed, fetched.State)
		assert.Equal(t, "test", fetched.ClusterInstallationID)
		assert.Equal(t, int64(100), fetched.CompleteAt)
	})
}
