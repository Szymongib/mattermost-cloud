package api_test

import (
	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/internal/api"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/internal/testutil"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http/httptest"
	"testing"
	"time"
)

func TestInstallationBackup(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})

	ts := httptest.NewServer(router)
	client := model.NewClient(ts.URL)
	installation1, err := client.CreateInstallation(
		&model.CreateInstallationRequest{
			OwnerID:   "owner",
			Version:   "version",
			DNS:       "dns1.example.com",
			Affinity:  model.InstallationAffinityMultiTenant,
			Database:  model.InstallationDatabaseMultiTenantRDSPostgres,
			Filestore: model.InstallationFilestoreBifrost,
		})
	require.NoError(t, err)

	t.Run("fail for not hibernated installation1", func(t *testing.T) {
		_, err = client.RequestInstallationBackup(installation1.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})

	installation1.State = model.InstallationStateHibernating
	err = sqlStore.UpdateInstallation(installation1.Installation)
	require.NoError(t, err)

	backupMeta, err := client.RequestInstallationBackup(installation1.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, backupMeta.ID)

	t.Run("fail to request multiple backups for same installation1", func(t *testing.T) {
		_, err = client.RequestInstallationBackup(installation1.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "400")
	})

	t.Run("can request backup for different installation", func(t *testing.T) {
		installation2, err := client.CreateInstallation(
			&model.CreateInstallationRequest{
				OwnerID:   "owner",
				Version:   "version",
				DNS:       "dns2.example.com",
				Affinity:  model.InstallationAffinityMultiTenant,
				Database:  model.InstallationDatabaseMultiTenantRDSPostgres,
				Filestore: model.InstallationFilestoreBifrost,
			})
		require.NoError(t, err)

		installation2.State = model.InstallationStateHibernating
		err = sqlStore.UpdateInstallation(installation2.Installation)
		require.NoError(t, err)

		backupMeta2, err := client.RequestInstallationBackup(installation2.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, backupMeta2.ID)
	})
}

func TestGetInstallationBackups(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})

	ts := httptest.NewServer(router)
	client := model.NewClient(ts.URL)

	installation1 := testutil.CreateBackupCompatibleInstallation(t, sqlStore)
	installation2 := testutil.CreateBackupCompatibleInstallation(t, sqlStore)

	backupMeta := []*model.InstallationBackup{
		{
			InstallationID: installation1.ID,
			State:          model.InstallationBackupStateBackupRequested,
		},
		{
			InstallationID: installation1.ID,
			State:          model.InstallationBackupStateBackupFailed,
		},
		{
			InstallationID: installation2.ID,
			State:          model.InstallationBackupStateBackupRequested,
		},
		{
			InstallationID:        installation2.ID,
			State:                 model.InstallationBackupStateBackupRequested,
			ClusterInstallationID: "ci1",
		},
		{
			InstallationID:        installation2.ID,
			State:                 model.InstallationBackupStateBackupSucceeded,
			ClusterInstallationID: "ci1",
		},
	}

	for i := range backupMeta {
		err := sqlStore.CreateInstallationBackup(backupMeta[i])
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond)
	}
	deletedMeta := &model.InstallationBackup{InstallationID: "deleted"}
	err := sqlStore.CreateInstallationBackup(deletedMeta)
	require.NoError(t, err)
	err = sqlStore.DeleteBackup(deletedMeta.ID)
	require.NoError(t, err)
	deletedMeta, err = sqlStore.GetInstallationBackup(deletedMeta.ID)

	for _, testCase := range []struct {
		description string
		filter      model.GetInstallationBackupsRequest
		found       []*model.InstallationBackup
	}{
		{
			description: "all",
			filter:      model.GetInstallationBackupsRequest{PerPage: model.AllPerPage, IncludeDeleted: true},
			found:       append(backupMeta, deletedMeta),
		},
		{
			description: "all not deleted",
			filter:      model.GetInstallationBackupsRequest{PerPage: model.AllPerPage, IncludeDeleted: false},
			found:       backupMeta,
		},
		{
			description: "1 per page",
			filter:      model.GetInstallationBackupsRequest{PerPage: 1},
			found:       []*model.InstallationBackup{backupMeta[4]},
		},
		{
			description: "2nd page",
			filter:      model.GetInstallationBackupsRequest{PerPage: 1, Page: 1},
			found:       []*model.InstallationBackup{backupMeta[3]},
		},
		{
			description: "filter by installation ID",
			filter:      model.GetInstallationBackupsRequest{PerPage: model.AllPerPage, InstallationID: installation1.ID},
			found:       []*model.InstallationBackup{backupMeta[0], backupMeta[1]},
		},
		{
			description: "filter by cluster installation ID",
			filter:      model.GetInstallationBackupsRequest{PerPage: model.AllPerPage, ClusterInstallationID: "ci1"},
			found:       []*model.InstallationBackup{backupMeta[3], backupMeta[4]},
		},
		{
			description: "filter by state",
			filter:      model.GetInstallationBackupsRequest{PerPage: model.AllPerPage, State: string(model.InstallationBackupStateBackupRequested)},
			found:       []*model.InstallationBackup{backupMeta[0], backupMeta[2], backupMeta[3]},
		},
		{
			description: "no results",
			filter:      model.GetInstallationBackupsRequest{PerPage: model.AllPerPage, InstallationID: "no-existent"},
			found:       []*model.InstallationBackup{},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {

			backups, err := client.GetInstallationBackups(&testCase.filter)
			require.NoError(t, err)
			require.Equal(t, len(testCase.found), len(backups))

			for i := 0; i < len(testCase.found); i++ {
				assert.Equal(t, testCase.found[i], backups[len(testCase.found)-1-i])
			}

		})
	}
}

func TestGetInstallationBackup(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})

	ts := httptest.NewServer(router)
	client := model.NewClient(ts.URL)

	installation1 := testutil.CreateBackupCompatibleInstallation(t, sqlStore)

	backupMeta, err := client.RequestInstallationBackup(installation1.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, backupMeta.ID)

	fetchedMeta, err := client.GetInstallationBackup(backupMeta.ID)
	require.NoError(t, err)
	assert.Equal(t, backupMeta, fetchedMeta)

	t.Run("return 404 if backup not found", func(t *testing.T) {
		_, err = client.GetInstallationBackup("not-real")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "404")
	})
}

func TestBackupAPILock(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})

	ts := httptest.NewServer(router)
	client := model.NewClient(ts.URL)

	installation1 := testutil.CreateBackupCompatibleInstallation(t, sqlStore)

	backupMeta, err := client.RequestInstallationBackup(installation1.ID)
	require.NoError(t, err)
	assert.NotEmpty(t, backupMeta.ID)

	err = client.LockAPIForBackup(backupMeta.ID)
	require.NoError(t, err)
	fetchedMeta, err := client.GetInstallationBackup(backupMeta.ID)
	require.NoError(t, err)
	assert.True(t, fetchedMeta.APISecurityLock)

	err = client.UnlockAPIForBackup(backupMeta.ID)
	require.NoError(t, err)
	fetchedMeta, err = client.GetInstallationBackup(backupMeta.ID)
	require.NoError(t, err)
	assert.False(t, fetchedMeta.APISecurityLock)
}
