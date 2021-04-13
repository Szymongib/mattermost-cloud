package supervisor_test

import (
	"github.com/mattermost/mattermost-cloud/internal/provisioner"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

//type mockBackupStore struct {
//	BackupMetadata        *model.InstallationBackup
//	BackupMetadataPending []*model.InstallationBackup
//	Cluster               *model.Cluster
//	Installation          *model.Installation
//	ClusterInstallations  []*model.ClusterInstallation
//	UnlockChan            chan interface{}
//
//	UpdateBackupMetadataCalls int
//}
//
//func (s mockBackupStore) GetUnlockedInstallationBackupPendingWork() ([]*model.InstallationBackup, error) {
//	return s.BackupMetadataPending, nil
//}
//
//func (s mockBackupStore) GetInstallationBackup(id string) (*model.InstallationBackup, error) {
//	return s.BackupMetadataPending[0], nil
//}
//
//func (s *mockBackupStore) UpdateInstallationBackupState(backupMeta *model.InstallationBackup) error {
//	s.UpdateBackupMetadataCalls++
//	return nil
//}
//
//func (s *mockBackupStore) UpdateInstallationBackupSchedulingData(backupMeta *model.InstallationBackup) error {
//	s.UpdateBackupMetadataCalls++
//	return nil
//}
//
//func (s mockBackupStore) UpdateInstallationBackupStartTime(backupMeta *model.InstallationBackup) error {
//	panic("implement me")
//}
//
//func (s mockBackupStore) DeleteInstallationBackup(backupID string) error {
//	panic("implement me")
//}
//
//func (s mockBackupStore) LockInstallationBackups(backupIDs []string, lockerID string) (bool, error) {
//	return true, nil
//}
//
//func (s *mockBackupStore) UnlockInstallationBackups(backupIDs []string, lockerID string, force bool) (bool, error) {
//	if s.UnlockChan != nil {
//		close(s.UnlockChan)
//	}
//	return true, nil
//}
//
//func (s mockBackupStore) GetInstallation(installationID string, includeGroupConfig, includeGroupConfigOverrides bool) (*model.Installation, error) {
//	return s.Installation, nil
//}
//
//func (s mockBackupStore) LockInstallation(installationID, lockerID string) (bool, error) {
//	return true, nil
//}
//
//func (s mockBackupStore) UnlockInstallation(installationID, lockerID string, force bool) (bool, error) {
//	return true, nil
//}
//
//func (s mockBackupStore) GetClusterInstallations(filter *model.ClusterInstallationFilter) ([]*model.ClusterInstallation, error) {
//	return s.ClusterInstallations, nil
//}
//
//func (s mockBackupStore) GetClusterInstallation(clusterInstallationID string) (*model.ClusterInstallation, error) {
//	return s.ClusterInstallations[0], nil
//}
//
//func (s mockBackupStore) LockClusterInstallations(clusterInstallationID []string, lockerID string) (bool, error) {
//	return true, nil
//}
//
//func (s mockBackupStore) UnlockClusterInstallations(clusterInstallationID []string, lockerID string, force bool) (bool, error) {
//	return true, nil
//}
//
//func (s mockBackupStore) GetCluster(id string) (*model.Cluster, error) {
//	return s.Cluster, nil
//}
//
//func (s mockBackupStore) GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error) {
//	return nil, nil
//}

type mockRestoreProvisioner struct {
	RestoreCompleteTime int64
	err             error
}

func (p *mockRestoreProvisioner) TriggerRestore(installation *model.Installation, backup *model.InstallationBackup, cluster *model.Cluster) error {
	return p.err
}

func (p *mockRestoreProvisioner) CheckRestoreStatus(backupMeta *model.InstallationBackup, cluster *model.Cluster) (int64, error) {
	return p.RestoreCompleteTime, p.err
}

func (p *mockRestoreProvisioner) CleanupRestoreJob(backup *model.InstallationBackup, cluster *model.Cluster) error {
	return p.err
}


//func TestInstallationDBRestorationSupervisor_Do(t *testing.T) {
//	t.Run("no backup pending work", func(t *testing.T) {
//		logger := testlib.MakeLogger(t)
//		mockStore := &mockBackupStore{}
//		mockBackupOp := &mockBackupProvisioner{}
//
//		backupSupervisor := supervisor.NewBackupSupervisor(mockStore, mockBackupOp, &mockAWS{}, "instanceID", logger)
//		err := backupSupervisor.Do()
//		require.NoError(t, err)
//
//		require.Equal(t, 0, mockStore.UpdateBackupMetadataCalls)
//	})
//
//	t.Run("mock backup trigger", func(t *testing.T) {
//		logger := testlib.MakeLogger(t)
//
//		cluster := &model.Cluster{ID: model.NewID()}
//		installation := &model.Installation{
//			ID:        model.NewID(),
//			State:     model.InstallationStateHibernating,
//			Database:  model.InstallationDatabaseMultiTenantRDSPostgres,
//			Filestore: model.InstallationFilestoreBifrost,
//		}
//		mockStore := &mockBackupStore{
//			Cluster:      cluster,
//			Installation: installation,
//			BackupMetadataPending: []*model.InstallationBackup{
//				{ID: model.NewID(), InstallationID: installation.ID, State: model.InstallationBackupStateBackupRequested},
//			},
//			ClusterInstallations: []*model.ClusterInstallation{{
//				ID:             model.NewID(),
//				ClusterID:      cluster.ID,
//				InstallationID: installation.ID,
//				State:          model.ClusterInstallationStateStable,
//			}},
//			UnlockChan: make(chan interface{}),
//		}
//
//		backupSupervisor := supervisor.NewBackupSupervisor(mockStore, &mockBackupProvisioner{}, &mockAWS{}, "instanceID", logger)
//		err := backupSupervisor.Do()
//		require.NoError(t, err)
//
//		<-mockStore.UnlockChan
//		require.Equal(t, 2, mockStore.UpdateBackupMetadataCalls)
//	})
//}

func TestInstallationDBRestorationSupervisor_Supervise(t *testing.T) {

	t.Run("transition to restoration", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		mockRestoreOp := &mockRestoreProvisioner{}

		installation, clusterInstallation, backup := setupRestoreRequiredResources(t, sqlStore)

		restorationOp := &model.InstallationDBRestorationOperation{
			InstallationID:          installation.ID,
			BackupID:                backup.ID,
			State:                   model.InstallationDBRestorationStateRequested,
		}
		err := sqlStore.CreateInstallationDBRestoration(restorationOp)
		require.NoError(t, err)

		backupSupervisor := supervisor.NewInstallationDBRestorationSupervisor(sqlStore, &mockAWS{}, mockRestoreOp, "instanceID", logger)
		backupSupervisor.Supervise(restorationOp)

		// Assert
		restorationOp, err = sqlStore.GetInstallationDBRestoration(restorationOp.ID)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationDBRestorationStateBeginning, restorationOp.State)
		assert.Equal(t, clusterInstallation.ID, restorationOp.ClusterInstallationID)
		assert.Equal(t, model.InstallationStateHibernating, restorationOp.TargetInstallationState)

		installation, err = sqlStore.GetInstallation(installation.ID, false,false)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationStateDBRestorationInProgress, installation.State)

		backup, err = sqlStore.GetInstallationBackup(backup.ID)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationBackupStateRestorationInProgress, backup.State)
	})

	t.Run("trigger restoration", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		mockRestoreOp := &mockRestoreProvisioner{}

		installation, clusterInstallation, backup := setupRestoreRequiredResources(t, sqlStore)

		restorationOp := &model.InstallationDBRestorationOperation{
			InstallationID:          installation.ID,
			BackupID:                backup.ID,
			ClusterInstallationID:   clusterInstallation.ID,
			State:                   model.InstallationDBRestorationStateBeginning,
		}
		err := sqlStore.CreateInstallationDBRestoration(restorationOp)
		require.NoError(t, err)

		backupSupervisor := supervisor.NewInstallationDBRestorationSupervisor(sqlStore, &mockAWS{}, mockRestoreOp, "instanceID", logger)
		backupSupervisor.Supervise(restorationOp)

		// Assert
		restorationOp, err = sqlStore.GetInstallationDBRestoration(restorationOp.ID)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationDBRestorationStateInProgress, restorationOp.State)
	})

	t.Run("check restoration status", func(t *testing.T) {
		for _, testCase := range []struct{
		    description string
		    mockRestoreOp *mockRestoreProvisioner
		    expectedState model.InstallationDBRestorationState
		}{
			{
				description:   "when restore finished",
				mockRestoreOp:  &mockRestoreProvisioner{RestoreCompleteTime: 100},
				expectedState: model.InstallationDBRestorationStateFinalizing,
			},
			{
				description:   "when still in progress",
				mockRestoreOp:  &mockRestoreProvisioner{RestoreCompleteTime: -1},
				expectedState: model.InstallationDBRestorationStateInProgress,
			},
			{
				description:   "when non terminal error",
				mockRestoreOp:  &mockRestoreProvisioner{RestoreCompleteTime: -1, err: errors.New("some error")},
				expectedState: model.InstallationDBRestorationStateInProgress,
			},
			{
				description:   "when terminal error",
				mockRestoreOp:  &mockRestoreProvisioner{RestoreCompleteTime: -1, err: provisioner.ErrJobBackoffLimitReached},
				expectedState: model.InstallationDBRestorationStateFailed,
			},
		} {
		    t.Run(testCase.description, func(t *testing.T) {
				logger := testlib.MakeLogger(t)
				sqlStore := store.MakeTestSQLStore(t, logger)
				defer store.CloseConnection(t, sqlStore)

				installation, clusterInstallation, backup := setupRestoreRequiredResources(t, sqlStore)

				restorationOp := &model.InstallationDBRestorationOperation{
					InstallationID:          installation.ID,
					BackupID:                backup.ID,
					State:                   model.InstallationDBRestorationStateInProgress,
					ClusterInstallationID: clusterInstallation.ID,
				}
				err := sqlStore.CreateInstallationDBRestoration(restorationOp)
				require.NoError(t, err)

				backupSupervisor := supervisor.NewInstallationDBRestorationSupervisor(sqlStore, &mockAWS{}, testCase.mockRestoreOp, "instanceID", logger)
				backupSupervisor.Supervise(restorationOp)

				// Assert
				restorationOp, err = sqlStore.GetInstallationDBRestoration(restorationOp.ID)
				require.NoError(t, err)
				assert.Equal(t, testCase.expectedState, restorationOp.State)
				assert.Equal(t, clusterInstallation.ID, restorationOp.ClusterInstallationID)
		    })
		}
	})


	t.Run("finalizing restoration", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		defer store.CloseConnection(t, sqlStore)

		mockRestoreOp := &mockRestoreProvisioner{}

		installation, clusterInstallation, backup := setupRestoreRequiredResources(t, sqlStore)

		restorationOp := &model.InstallationDBRestorationOperation{
			InstallationID:          installation.ID,
			BackupID:                backup.ID,
			State:                   model.InstallationDBRestorationStateFinalizing,
			ClusterInstallationID: clusterInstallation.ID,
			TargetInstallationState: model.InstallationStateHibernating,
		}
		err := sqlStore.CreateInstallationDBRestoration(restorationOp)
		require.NoError(t, err)

		backupSupervisor := supervisor.NewInstallationDBRestorationSupervisor(sqlStore, &mockAWS{}, mockRestoreOp, "instanceID", logger)
		backupSupervisor.Supervise(restorationOp)

		// Assert
		restorationOp, err = sqlStore.GetInstallationDBRestoration(restorationOp.ID)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationDBRestorationStateSucceeded, restorationOp.State)

		installation, err = sqlStore.GetInstallation(installation.ID, false,false)
		require.NoError(t, err)
		assert.Equal(t, model.InstallationStateHibernating, installation.State)

		// TODO: check backup state if you decide to change it - set back to succeeded
	})


	//t.Run("do not trigger backup if installation not hibernated", func(t *testing.T) {
	//	logger := testlib.MakeLogger(t)
	//	sqlStore := store.MakeTestSQLStore(t, logger)
	//	mockBackupOp := &mockBackupProvisioner{}
	//
	//	installation, _ := setupBackupRequiredResources(t, sqlStore)
	//	installation.State = model.InstallationStateStable
	//	err := sqlStore.UpdateInstallationState(installation)
	//	require.NoError(t, err)
	//
	//	backupMeta := &model.InstallationBackup{
	//		InstallationID: installation.ID,
	//		State:          model.InstallationBackupStateBackupRequested,
	//	}
	//	err = sqlStore.CreateInstallationBackup(backupMeta)
	//	require.NoError(t, err)
	//
	//	backupSupervisor := supervisor.NewBackupSupervisor(sqlStore, mockBackupOp, &mockAWS{}, "instanceID", logger)
	//	backupSupervisor.Supervise(backupMeta)
	//
	//	// Assert
	//	backupMeta, err = sqlStore.GetInstallationBackup(backupMeta.ID)
	//	require.NoError(t, err)
	//	assert.Equal(t, model.InstallationBackupStateBackupRequested, backupMeta.State)
	//})
	//
	//t.Run("set backup as failed if installation deleted", func(t *testing.T) {
	//	logger := testlib.MakeLogger(t)
	//	sqlStore := store.MakeTestSQLStore(t, logger)
	//	mockBackupOp := &mockBackupProvisioner{}
	//
	//	backupMeta := &model.InstallationBackup{
	//		InstallationID: "deleted-installation-id",
	//		State:          model.InstallationBackupStateBackupRequested,
	//	}
	//	err := sqlStore.CreateInstallationBackup(backupMeta)
	//	require.NoError(t, err)
	//
	//	backupSupervisor := supervisor.NewBackupSupervisor(sqlStore, mockBackupOp, &mockAWS{}, "instanceID", logger)
	//	backupSupervisor.Supervise(backupMeta)
	//
	//	// Assert
	//	backupMeta, err = sqlStore.GetInstallationBackup(backupMeta.ID)
	//	require.NoError(t, err)
	//	assert.Equal(t, model.InstallationBackupStateBackupFailed, backupMeta.State)
	//})

	//
	//t.Run("cleanup backup", func(t *testing.T) {
	//	logger := testlib.MakeLogger(t)
	//	sqlStore := store.MakeTestSQLStore(t, logger)
	//	mockBackupOp := &mockBackupProvisioner{}
	//
	//	installation, clusterInstallation := setupBackupRequiredResources(t, sqlStore)
	//
	//	backup := &model.InstallationBackup{
	//		InstallationID:        installation.ID,
	//		ClusterInstallationID: clusterInstallation.ID,
	//		State:                 model.InstallationBackupStateDeletionRequested,
	//	}
	//	err := sqlStore.CreateInstallationBackup(backup)
	//	require.NoError(t, err)
	//
	//	backup.DataResidence = &model.S3DataResidence{
	//		Region:     "us-east",
	//		URL:        aws.S3URL,
	//		Bucket:     "my-bucket",
	//		PathPrefix: installation.ID,
	//		ObjectKey:  "backup-123",
	//	}
	//	err = sqlStore.UpdateInstallationBackupSchedulingData(backup)
	//	require.NoError(t, err)
	//
	//	backupSupervisor := supervisor.NewBackupSupervisor(sqlStore, mockBackupOp, &mockAWS{}, "instanceID", logger)
	//	backupSupervisor.Supervise(backup)
	//
	//	// Assert
	//	backup, err = sqlStore.GetInstallationBackup(backup.ID)
	//	require.NoError(t, err)
	//	assert.Equal(t, model.InstallationBackupStateDeleted, backup.State)
	//	assert.NotEqualValues(t, 0, backup.DeleteAt)
	//})
	//
	//t.Run("full backup lifecycle", func(t *testing.T) {
	//	logger := testlib.MakeLogger(t)
	//	sqlStore := store.MakeTestSQLStore(t, logger)
	//	mockBackupOp := &mockBackupProvisioner{}
	//
	//	installation, clusterInstallation := setupBackupRequiredResources(t, sqlStore)
	//
	//	backup := &model.InstallationBackup{
	//		InstallationID:        installation.ID,
	//		ClusterInstallationID: clusterInstallation.ID,
	//		State:                 model.InstallationBackupStateBackupRequested,
	//	}
	//	err := sqlStore.CreateInstallationBackup(backup)
	//	require.NoError(t, err)
	//
	//	// Requested -> InProgress
	//	backupSupervisor := supervisor.NewBackupSupervisor(sqlStore, mockBackupOp, &mockAWS{}, "instanceID", logger)
	//	backupSupervisor.Supervise(backup)
	//
	//	backup, err = sqlStore.GetInstallationBackup(backup.ID)
	//	require.NoError(t, err)
	//	assert.Equal(t, model.InstallationBackupStateBackupInProgress, backup.State)
	//	assert.Equal(t, clusterInstallation.ID, backup.ClusterInstallationID)
	//
	//	// In progress -> Succeeded
	//	mockBackupOp.BackupStartTime = 100
	//	backupSupervisor.Supervise(backup)
	//
	//	backup, err = sqlStore.GetInstallationBackup(backup.ID)
	//	require.NoError(t, err)
	//	assert.Equal(t, model.InstallationBackupStateBackupSucceeded, backup.State)
	//
	//	// Deletion requested -> Deleted
	//	backup.State = model.InstallationBackupStateDeletionRequested
	//	err = sqlStore.UpdateInstallationBackupState(backup)
	//	require.NoError(t, err)
	//
	//	backupSupervisor.Supervise(backup)
	//
	//	backup, err = sqlStore.GetInstallationBackup(backup.ID)
	//	require.NoError(t, err)
	//	assert.Equal(t, model.InstallationBackupStateDeleted, backup.State)
	//	assert.NotEqualValues(t, 0, backup.DeleteAt)
	//})
}

func setupRestoreRequiredResources(t *testing.T, sqlStore *store.SQLStore) (*model.Installation, *model.ClusterInstallation, *model.InstallationBackup) {
	installation, clusterInstallation := setupBackupRequiredResources(t, sqlStore)

	backup := &model.InstallationBackup{
		InstallationID: installation.ID,
		State:          model.InstallationBackupStateBackupSucceeded,
	}
	err := sqlStore.CreateInstallationBackup(backup)
	require.NoError(t, err)

	return installation, clusterInstallation, backup
}
