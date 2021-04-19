package supervisor_test

import (
	"fmt"
	"github.com/mattermost/mattermost-cloud/internal/provisioner"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/internal/tools/utils"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pborman/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

type mockDatabase struct {}

func (m *mockDatabase) Provision(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	panic("implement me")
}

func (m *mockDatabase) Teardown(store model.InstallationDatabaseStoreInterface, keepData bool, logger log.FieldLogger) error {
	panic("implement me")
}

func (m *mockDatabase) Snapshot(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	panic("implement me")
}

func (m *mockDatabase) GenerateDatabaseSecret(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*corev1.Secret, error) {
	panic("implement me")
}

func (m *mockDatabase) RefreshResourceMetadata(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	panic("implement me")
}

func (m *mockDatabase) MigrateOut(store model.InstallationDatabaseStoreInterface, dbMigration *model.DBMigrationOperation, logger log.FieldLogger) error {
	return nil
}

func (m *mockDatabase) MigrateTo(store model.InstallationDatabaseStoreInterface, dbMigration *model.DBMigrationOperation, logger log.FieldLogger) error {
	return nil
}

type mockResourceUtil struct {}

func (m *mockResourceUtil) 	GetDatabase(installationID, dbType string) model.Database {
	return &mockDatabase{}
}

type mockMigrationProvisioner struct {}
func (m *mockMigrationProvisioner) 	ClusterInstallationProvisioner(version string) provisioner.ClusterInstallationProvisioner {
	return &mockInstallationProvisioner{}
}

// TODO: Do method tests

func TestDBMigrationSupervisor_Supervise(t *testing.T) {

	t.Run("trigger backup", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		installation, _ := setupMigrationRequiredResources(t, sqlStore)

		migrationOp := &model.DBMigrationOperation{
			InstallationID:          installation.ID,
			State:                   model.DBMigrationStateRequested,
		}

		err := sqlStore.CreateInstallationDBMigration(migrationOp)
		require.NoError(t, err)

		dbMigrationSupervisor := supervisor.NewInstallationDBMigrationSupervisor(sqlStore, &mockAWS{}, &utils.ResourceUtil{}, "instanceID", nil, logger)
		dbMigrationSupervisor.Supervise(migrationOp)

		// Assert
		migrationOp, err = sqlStore.GetInstallationDBMigration(migrationOp.ID)
		require.NoError(t, err)
		assert.Equal(t, model.DBMigrationStateInstallationBackupInProgress, migrationOp.State)
		assert.NotEmpty(t, migrationOp.BackupID)

		backup, err := sqlStore.GetInstallationBackup(migrationOp.BackupID)
		require.NoError(t, err)
		require.NotNil(t, backup)
		assert.Equal(t, model.InstallationBackupStateBackupRequested, backup.State)
		assert.Equal(t, installation.ID, backup.InstallationID)
	})

	t.Run("wait for installation backup", func(t *testing.T) {
		for _, testCase := range []struct{
			description string
			backupState model.InstallationBackupState
			expectedState model.DBMigrationOperationState
		}{
			{
				description:   "when backup requested",
				backupState: model.InstallationBackupStateBackupRequested,
				expectedState: model.DBMigrationStateInstallationBackupInProgress,
			},
			{
				description:   "when backup in progress",
				backupState: model.InstallationBackupStateBackupInProgress,
				expectedState: model.DBMigrationStateInstallationBackupInProgress,
			},
			{
				description:   "when backup succeeded",
				backupState: model.InstallationBackupStateBackupSucceeded,
				expectedState: model.DBMigrationStateDatabaseSwitch,
			},
			{
				description:   "when backup failed",
				backupState: model.InstallationBackupStateBackupFailed,
				expectedState: model.DBMigrationStateFailing,
			},
		} {
			t.Run(testCase.description, func(t *testing.T) {
				logger := testlib.MakeLogger(t)
				sqlStore := store.MakeTestSQLStore(t, logger)
				defer store.CloseConnection(t, sqlStore)

				installation, _ := setupMigrationRequiredResources(t, sqlStore)

				backup := &model.InstallationBackup{
					InstallationID: installation.ID,
					State: testCase.backupState,
				}
				err := sqlStore.CreateInstallationBackup(backup)
				require.NoError(t, err)

				migrationOp := &model.DBMigrationOperation{
					InstallationID:          installation.ID,
					State:                   model.DBMigrationStateInstallationBackupInProgress,
					BackupID: backup.ID,
				}

				err = sqlStore.CreateInstallationDBMigration(migrationOp)
				require.NoError(t, err)

				dbMigrationSupervisor := supervisor.NewInstallationDBMigrationSupervisor(sqlStore, &mockAWS{}, &utils.ResourceUtil{}, "instanceID",  nil,logger)
				dbMigrationSupervisor.Supervise(migrationOp)

				// Assert
				migrationOp, err = sqlStore.GetInstallationDBMigration(migrationOp.ID)
				require.NoError(t, err)
				assert.Equal(t, testCase.expectedState, migrationOp.State)
			})
		}
	})

	t.Run("switch database", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		installation, _ := setupMigrationRequiredResources(t, sqlStore)

		migrationOp := &model.DBMigrationOperation{
			InstallationID:          installation.ID,
			State:                   model.DBMigrationStateDatabaseSwitch,
			SourceDatabase:          model.InstallationDatabaseMultiTenantRDSPostgres,
			DestinationDatabase:     model.InstallationDatabaseSingleTenantRDSPostgres,
			SourceMultiTenant: &model.MultiTenantDBMigrationData{DatabaseID: "source-id"},
			DestinationMultiTenant: &model.MultiTenantDBMigrationData{DatabaseID: "destination-id"},
		}

		err := sqlStore.CreateInstallationDBMigration(migrationOp)
		require.NoError(t, err)

		dbMigrationSupervisor := supervisor.NewInstallationDBMigrationSupervisor(sqlStore, &mockAWS{}, &mockResourceUtil{}, "instanceID",  nil,logger)
		dbMigrationSupervisor.Supervise(migrationOp)

		// Assert
		migrationOp, err = sqlStore.GetInstallationDBMigration(migrationOp.ID)
		require.NoError(t, err)
		assert.Equal(t, model.DBMigrationStateRefreshSecrets, migrationOp.State)

		installation, err = sqlStore.GetInstallation(installation.ID, false, false)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationDatabaseSingleTenantRDSPostgres, installation.Database)
	})

	t.Run("refresh secrets", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		installation, _ := setupMigrationRequiredResources(t, sqlStore)

		migrationOp := &model.DBMigrationOperation{
			InstallationID:          installation.ID,
			State:                   model.DBMigrationStateRefreshSecrets,
		}

		err := sqlStore.CreateInstallationDBMigration(migrationOp)
		require.NoError(t, err)

		dbMigrationSupervisor := supervisor.NewInstallationDBMigrationSupervisor(sqlStore, &mockAWS{}, &mockResourceUtil{}, "instanceID",  &mockMigrationProvisioner{},logger)
		dbMigrationSupervisor.Supervise(migrationOp)

		// Assert
		migrationOp, err = sqlStore.GetInstallationDBMigration(migrationOp.ID)
		require.NoError(t, err)
		assert.Equal(t, model.DBMigrationStateTriggerRestoration, migrationOp.State)
	})

	t.Run("trigger restoration", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		installation, _ := setupMigrationRequiredResources(t, sqlStore)

		backup := &model.InstallationBackup{
			InstallationID: installation.ID,
			State: model.InstallationBackupStateBackupSucceeded,
		}
		err := sqlStore.CreateInstallationBackup(backup)
		require.NoError(t, err)

		migrationOp := &model.DBMigrationOperation{
			InstallationID:          installation.ID,
			State:                   model.DBMigrationStateTriggerRestoration,
			BackupID: backup.ID,
		}

		err = sqlStore.CreateInstallationDBMigration(migrationOp)
		require.NoError(t, err)

		dbMigrationSupervisor := supervisor.NewInstallationDBMigrationSupervisor(sqlStore, &mockAWS{}, &mockResourceUtil{}, "instanceID",  nil,logger)
		dbMigrationSupervisor.Supervise(migrationOp)

		// Assert
		migrationOp, err = sqlStore.GetInstallationDBMigration(migrationOp.ID)
		require.NoError(t, err)
		assert.Equal(t, model.DBMigrationStateRestorationInProgress, migrationOp.State)
		assert.NotEmpty(t, migrationOp.InstallationDBRestorationOperationID)

		restorationOp, err := sqlStore.GetInstallationDBRestoration(migrationOp.InstallationDBRestorationOperationID)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationDBRestorationStateRequested, restorationOp.State)
		assert.Equal(t, installation.ID, restorationOp.InstallationID)
		assert.Equal(t, backup.ID, restorationOp.BackupID)
	})

	t.Run("trigger restoration - fail if no backup", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		installation, _ := setupMigrationRequiredResources(t, sqlStore)

		migrationOp := &model.DBMigrationOperation{
			InstallationID:          installation.ID,
			State:                   model.DBMigrationStateTriggerRestoration,
		}

		err := sqlStore.CreateInstallationDBMigration(migrationOp)
		require.NoError(t, err)

		dbMigrationSupervisor := supervisor.NewInstallationDBMigrationSupervisor(sqlStore, &mockAWS{}, &mockResourceUtil{}, "instanceID",  nil,logger)
		dbMigrationSupervisor.Supervise(migrationOp)

		// Assert
		migrationOp, err = sqlStore.GetInstallationDBMigration(migrationOp.ID)
		require.NoError(t, err)
		assert.Equal(t, model.DBMigrationStateFailing, migrationOp.State)
	})

	t.Run("wait for installation restoration", func(t *testing.T) {
		for _, testCase := range []struct{
			description        string
			restorationOpState model.InstallationDBRestorationState
			expectedState      model.DBMigrationOperationState
		}{
			{
				description:        "when restoration requested",
				restorationOpState: model.InstallationDBRestorationStateRequested,
				expectedState:      model.DBMigrationStateRestorationInProgress,
			},
			{
				description:        "when restoration in progress",
				restorationOpState: model.InstallationDBRestorationStateInProgress,
				expectedState:      model.DBMigrationStateRestorationInProgress,
			},
			{
				description:        "when restoration finalizing",
				restorationOpState: model.InstallationDBRestorationStateFinalizing,
				expectedState:      model.DBMigrationStateRestorationInProgress,
			},
			{
				description:        "when restoration succeeded",
				restorationOpState: model.InstallationDBRestorationStateSucceeded,
				expectedState:      model.DbMigrationStateFinalizing,
			},
			{
				description:        "when restoration failed",
				restorationOpState: model.InstallationDBRestorationStateFailed,
				expectedState:      model.DBMigrationStateFailing,
			},
			{
				description:        "when restoration invalid",
				restorationOpState: model.InstallationDBRestorationStateInvalid,
				expectedState:      model.DBMigrationStateFailing,
			},
		} {
			t.Run(testCase.description, func(t *testing.T) {
				logger := testlib.MakeLogger(t)
				sqlStore := store.MakeTestSQLStore(t, logger)
				defer store.CloseConnection(t, sqlStore)

				installation, _ := setupMigrationRequiredResources(t, sqlStore)

				restorationOp := &model.InstallationDBRestorationOperation{
					InstallationID: installation.ID,
					State: testCase.restorationOpState,
				}
				err := sqlStore.CreateInstallationDBRestoration(restorationOp)
				require.NoError(t, err)

				migrationOp := &model.DBMigrationOperation{
					InstallationID:          installation.ID,
					State:                   model.DBMigrationStateRestorationInProgress,
					InstallationDBRestorationOperationID: restorationOp.ID,
				}

				err = sqlStore.CreateInstallationDBMigration(migrationOp)
				require.NoError(t, err)

				dbMigrationSupervisor := supervisor.NewInstallationDBMigrationSupervisor(sqlStore, &mockAWS{}, &utils.ResourceUtil{}, "instanceID",  nil,logger)
				dbMigrationSupervisor.Supervise(migrationOp)

				// Assert
				migrationOp, err = sqlStore.GetInstallationDBMigration(migrationOp.ID)
				require.NoError(t, err)
				assert.Equal(t, testCase.expectedState, migrationOp.State)
			})
		}
	})


	t.Run("finalizing migration", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		installation, _ := setupMigrationRequiredResources(t, sqlStore)

		migrationOp := &model.DBMigrationOperation{
			InstallationID:          installation.ID,
			State:                   model.DbMigrationStateFinalizing,
		}

		err := sqlStore.CreateInstallationDBMigration(migrationOp)
		require.NoError(t, err)

		dbMigrationSupervisor := supervisor.NewInstallationDBMigrationSupervisor(sqlStore, &mockAWS{}, &mockResourceUtil{}, "instanceID",  nil,logger)
		dbMigrationSupervisor.Supervise(migrationOp)

		// Assert
		migrationOp, err = sqlStore.GetInstallationDBMigration(migrationOp.ID)
		require.NoError(t, err)
		assert.Equal(t, model.DBMigrationStateSucceeded, migrationOp.State)
		assert.True(t, migrationOp.CompleteAt > 0)

		installation, err = sqlStore.GetInstallation(installation.ID, false, false)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationStateHibernating, installation.State)
	})


	t.Run("failing migration", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		installation, _ := setupMigrationRequiredResources(t, sqlStore)

		migrationOp := &model.DBMigrationOperation{
			InstallationID:          installation.ID,
			State:                   model.DBMigrationStateFailing,
		}

		err := sqlStore.CreateInstallationDBMigration(migrationOp)
		require.NoError(t, err)

		dbMigrationSupervisor := supervisor.NewInstallationDBMigrationSupervisor(sqlStore, &mockAWS{}, &mockResourceUtil{}, "instanceID",  nil,logger)
		dbMigrationSupervisor.Supervise(migrationOp)

		// Assert
		migrationOp, err = sqlStore.GetInstallationDBMigration(migrationOp.ID)
		require.NoError(t, err)
		assert.Equal(t, model.DBMigrationStateFailed, migrationOp.State)

		installation, err = sqlStore.GetInstallation(installation.ID, false, false)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationStateDBMigrationFailed, installation.State)
	})


	//
	//
	//t.Run("finalizing restoration", func(t *testing.T) {
	//	logger := testlib.MakeLogger(t)
	//	sqlStore := store.MakeTestSQLStore(t, logger)
	//	defer store.CloseConnection(t, sqlStore)
	//
	//	mockRestoreOp := &mockRestoreProvisioner{}
	//
	//	installation, clusterInstallation, backup := setupRestoreRequiredResources(t, sqlStore)
	//
	//	restorationOp := &model.InstallationDBRestorationOperation{
	//		InstallationID:          installation.ID,
	//		BackupID:                backup.ID,
	//		State:                   model.InstallationDBRestorationStateFinalizing,
	//		ClusterInstallationID: clusterInstallation.ID,
	//		TargetInstallationState: model.InstallationStateHibernating,
	//	}
	//	err := sqlStore.CreateInstallationDBRestoration(restorationOp)
	//	require.NoError(t, err)
	//
	//	backupSupervisor := supervisor.NewInstallationDBRestorationSupervisor(sqlStore, &mockAWS{}, mockRestoreOp, "instanceID", logger)
	//	backupSupervisor.Supervise(restorationOp)
	//
	//	// Assert
	//	restorationOp, err = sqlStore.GetInstallationDBRestoration(restorationOp.ID)
	//	require.NoError(t, err)
	//	assert.Equal(t, model.InstallationDBRestorationStateSucceeded, restorationOp.State)
	//
	//	installation, err = sqlStore.GetInstallation(installation.ID, false,false)
	//	require.NoError(t, err)
	//	assert.Equal(t, model.InstallationStateHibernating, installation.State)
	//})
	//
	//
	////t.Run("do not trigger backup if installation not hibernated", func(t *testing.T) {
	////	logger := testlib.MakeLogger(t)
	////	sqlStore := store.MakeTestSQLStore(t, logger)
	////	mockBackupOp := &mockBackupProvisioner{}
	////
	////	installation, _ := setupBackupRequiredResources(t, sqlStore)
	////	installation.State = model.InstallationStateStable
	////	err := sqlStore.UpdateInstallationState(installation)
	////	require.NoError(t, err)
	////
	////	backupMeta := &model.InstallationBackup{
	////		InstallationID: installation.ID,
	////		State:          model.InstallationBackupStateBackupRequested,
	////	}
	////	err = sqlStore.CreateInstallationBackup(backupMeta)
	////	require.NoError(t, err)
	////
	////	backupSupervisor := supervisor.NewBackupSupervisor(sqlStore, mockBackupOp, &mockAWS{}, "instanceID", logger)
	////	backupSupervisor.Supervise(backupMeta)
	////
	////	// Assert
	////	backupMeta, err = sqlStore.GetInstallationBackup(backupMeta.ID)
	////	require.NoError(t, err)
	////	assert.Equal(t, model.InstallationBackupStateBackupRequested, backupMeta.State)
	////})
	////
	////t.Run("set backup as failed if installation deleted", func(t *testing.T) {
	////	logger := testlib.MakeLogger(t)
	////	sqlStore := store.MakeTestSQLStore(t, logger)
	////	mockBackupOp := &mockBackupProvisioner{}
	////
	////	backupMeta := &model.InstallationBackup{
	////		InstallationID: "deleted-installation-id",
	////		State:          model.InstallationBackupStateBackupRequested,
	////	}
	////	err := sqlStore.CreateInstallationBackup(backupMeta)
	////	require.NoError(t, err)
	////
	////	backupSupervisor := supervisor.NewBackupSupervisor(sqlStore, mockBackupOp, &mockAWS{}, "instanceID", logger)
	////	backupSupervisor.Supervise(backupMeta)
	////
	////	// Assert
	////	backupMeta, err = sqlStore.GetInstallationBackup(backupMeta.ID)
	////	require.NoError(t, err)
	////	assert.Equal(t, model.InstallationBackupStateBackupFailed, backupMeta.State)
	////})
	//
	////
	////t.Run("cleanup backup", func(t *testing.T) {
	////	logger := testlib.MakeLogger(t)
	////	sqlStore := store.MakeTestSQLStore(t, logger)
	////	mockBackupOp := &mockBackupProvisioner{}
	////
	////	installation, clusterInstallation := setupBackupRequiredResources(t, sqlStore)
	////
	////	backup := &model.InstallationBackup{
	////		InstallationID:        installation.ID,
	////		ClusterInstallationID: clusterInstallation.ID,
	////		State:                 model.InstallationBackupStateDeletionRequested,
	////	}
	////	err := sqlStore.CreateInstallationBackup(backup)
	////	require.NoError(t, err)
	////
	////	backup.DataResidence = &model.S3DataResidence{
	////		Region:     "us-east",
	////		URL:        aws.S3URL,
	////		Bucket:     "my-bucket",
	////		PathPrefix: installation.ID,
	////		ObjectKey:  "backup-123",
	////	}
	////	err = sqlStore.UpdateInstallationBackupSchedulingData(backup)
	////	require.NoError(t, err)
	////
	////	backupSupervisor := supervisor.NewBackupSupervisor(sqlStore, mockBackupOp, &mockAWS{}, "instanceID", logger)
	////	backupSupervisor.Supervise(backup)
	////
	////	// Assert
	////	backup, err = sqlStore.GetInstallationBackup(backup.ID)
	////	require.NoError(t, err)
	////	assert.Equal(t, model.InstallationBackupStateDeleted, backup.State)
	////	assert.NotEqualValues(t, 0, backup.DeleteAt)
	////})
	////
	////t.Run("full backup lifecycle", func(t *testing.T) {
	////	logger := testlib.MakeLogger(t)
	////	sqlStore := store.MakeTestSQLStore(t, logger)
	////	mockBackupOp := &mockBackupProvisioner{}
	////
	////	installation, clusterInstallation := setupBackupRequiredResources(t, sqlStore)
	////
	////	backup := &model.InstallationBackup{
	////		InstallationID:        installation.ID,
	////		ClusterInstallationID: clusterInstallation.ID,
	////		State:                 model.InstallationBackupStateBackupRequested,
	////	}
	////	err := sqlStore.CreateInstallationBackup(backup)
	////	require.NoError(t, err)
	////
	////	// Requested -> InProgress
	////	backupSupervisor := supervisor.NewBackupSupervisor(sqlStore, mockBackupOp, &mockAWS{}, "instanceID", logger)
	////	backupSupervisor.Supervise(backup)
	////
	////	backup, err = sqlStore.GetInstallationBackup(backup.ID)
	////	require.NoError(t, err)
	////	assert.Equal(t, model.InstallationBackupStateBackupInProgress, backup.State)
	////	assert.Equal(t, clusterInstallation.ID, backup.ClusterInstallationID)
	////
	////	// In progress -> Succeeded
	////	mockBackupOp.BackupStartTime = 100
	////	backupSupervisor.Supervise(backup)
	////
	////	backup, err = sqlStore.GetInstallationBackup(backup.ID)
	////	require.NoError(t, err)
	////	assert.Equal(t, model.InstallationBackupStateBackupSucceeded, backup.State)
	////
	////	// Deletion requested -> Deleted
	////	backup.State = model.InstallationBackupStateDeletionRequested
	////	err = sqlStore.UpdateInstallationBackupState(backup)
	////	require.NoError(t, err)
	////
	////	backupSupervisor.Supervise(backup)
	////
	////	backup, err = sqlStore.GetInstallationBackup(backup.ID)
	////	require.NoError(t, err)
	////	assert.Equal(t, model.InstallationBackupStateDeleted, backup.State)
	////	assert.NotEqualValues(t, 0, backup.DeleteAt)
	////})
}

func setupMigrationRequiredResources(t *testing.T, sqlStore *store.SQLStore) (*model.Installation, *model.ClusterInstallation) {
	installation := &model.Installation{
		Database:  model.InstallationDatabaseMultiTenantRDSPostgres,
		Filestore: model.InstallationFilestoreBifrost,
		State:     model.InstallationStateDBMigrationInProgress,
		DNS:       fmt.Sprintf("dns-%s", uuid.NewRandom().String()[:6]),
	}
	err := sqlStore.CreateInstallation(installation, nil)
	require.NoError(t, err)

	cluster := &model.Cluster{}
	err = sqlStore.CreateCluster(cluster, nil)
	require.NoError(t, err)

	clusterInstallation := &model.ClusterInstallation{InstallationID: installation.ID, ClusterID: cluster.ID}
	err = sqlStore.CreateClusterInstallation(clusterInstallation)
	require.NoError(t, err)

	return installation, clusterInstallation
}

func setupMultiTenantDBsForMigration(t *testing.T, sqlStore *store.SQLStore, installation *model.Installation) (*model.MultitenantDatabase, *model.MultitenantDatabase) {
	db1 := &model.MultitenantDatabase{
		ID:                    "database-1",
		DatabaseType:          model.DatabaseEngineTypePostgres,
		Installations: 		[]string{installation.ID},
	}
	err := sqlStore.CreateMultitenantDatabase(db1)
	require.NoError(t, err)

	db2 := &model.MultitenantDatabase{
		ID:                    "database-2",
		DatabaseType:          model.DatabaseEngineTypePostgres,
	}
	err = sqlStore.CreateMultitenantDatabase(db2)
	require.NoError(t, err)

	return db1, db2
}
