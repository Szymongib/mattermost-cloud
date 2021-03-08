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

type mockBackupMetadataStore struct {
	BackupMetadata *model.BackupMetadata
	Cluster        *model.Cluster
	//Installation                            *model.Installation
	//ClusterInstallation                     *model.ClusterInstallation
	//UnlockedClusterInstallationsPendingWork []*model.ClusterInstallation
	//ClusterInstallations                    []*model.ClusterInstallation
	//
	UnlockChan chan interface{}

	UpdateBackupMetadataCalls int
}

func (s mockBackupMetadataStore) GetUnlockedBackupMetadataPendingWork() ([]*model.BackupMetadata, error) {
	return []*model.BackupMetadata{}, nil
}

func (s mockBackupMetadataStore) GetBackupMetadata(id, installationID string) (*model.BackupMetadata, error) {
	return s.BackupMetadata, nil
}

func (s *mockBackupMetadataStore) UpdateBackupMetadataState(backupMeta *model.BackupMetadata) error {
	return nil
}

func (s mockBackupMetadataStore) UpdateBackupSchedulingData(backupMeta *model.BackupMetadata) error {
	panic("implement me")
}

func (s mockBackupMetadataStore) UpdateBackupStartTime(backupMeta *model.BackupMetadata) error {
	panic("implement me")
}

func (s mockBackupMetadataStore) LockBackupMetadata(installationID, lockerID string) (bool, error) {
	return true, nil
}

func (s *mockBackupMetadataStore) UnlockBackupMetadata(installationID, lockerID string, force bool) (bool, error) {
	s.UpdateBackupMetadataCalls++
	panic("implement me")
}

func (s mockBackupMetadataStore) GetInstallation(installationID string, includeGroupConfig, includeGroupConfigOverrides bool) (*model.Installation, error) {
	panic("implement me")
}

func (s mockBackupMetadataStore) LockInstallation(installationID, lockerID string) (bool, error) {
	panic("implement me")
}

func (s mockBackupMetadataStore) UnlockInstallation(installationID, lockerID string, force bool) (bool, error) {
	panic("implement me")
}

func (s mockBackupMetadataStore) GetClusterInstallations(filter *model.ClusterInstallationFilter) ([]*model.ClusterInstallation, error) {
	panic("implement me")
}

func (s mockBackupMetadataStore) GetClusterInstallation(clusterInstallationID string) (*model.ClusterInstallation, error) {
	panic("implement me")
}

func (s mockBackupMetadataStore) LockClusterInstallations(clusterInstallationID []string, lockerID string) (bool, error) {
	return true, nil
}

func (s mockBackupMetadataStore) UnlockClusterInstallations(clusterInstallationID []string, lockerID string, force bool) (bool, error) {
	if s.UnlockChan != nil {
		close(s.UnlockChan)
	}
	return true, nil
}

func (s mockBackupMetadataStore) GetCluster(id string) (*model.Cluster, error) {
	return s.Cluster, nil
}

func (s mockBackupMetadataStore) GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error) {
	return nil, nil
}

type mockBackupOperator struct {
	BackupStartTime int64
	err             error
}

func (b *mockBackupOperator) TriggerBackup(backupMeta *model.BackupMetadata, cluster *model.Cluster, installation *model.Installation) (*model.S3DataResidence, error) {
	return &model.S3DataResidence{URL: "file-store.com"}, b.err
}

func (b *mockBackupOperator) CheckBackupStatus(backupMeta *model.BackupMetadata, cluster *model.Cluster) (int64, error) {
	return b.BackupStartTime, b.err
}

func TestBackupSupervisorDo(t *testing.T) {
	t.Run("no backup pending work", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		mockStore := &mockBackupMetadataStore{}
		mockBackupOp := &mockBackupOperator{}

		backupSupervisor := supervisor.NewBackupSupervisor(mockStore, mockBackupOp, &mockAWS{}, "instanceID", logger, nil)
		err := backupSupervisor.Do()
		require.NoError(t, err)

		require.Equal(t, 0, mockStore.UpdateBackupMetadataCalls)
	})

	//t.Run("mock cluster creation", func(t *testing.T) {
	//	logger := testlib.MakeLogger(t)
	//
	//	cluster := &model.Cluster{ID: model.NewID()}
	//	installation := &model.Installation{ID: model.NewID()}
	//	mockStore := &mockClusterInstallationStore{
	//		Cluster:      cluster,
	//		Installation: installation,
	//		UnlockedClusterInstallationsPendingWork: []*model.ClusterInstallation{{
	//			ID:             model.NewID(),
	//			ClusterID:      cluster.ID,
	//			InstallationID: installation.ID,
	//			State:          model.ClusterInstallationStateCreationRequested,
	//		}},
	//		UnlockChan: make(chan interface{}),
	//	}
	//	mockStore.ClusterInstallation = mockStore.UnlockedClusterInstallationsPendingWork[0]
	//
	//	supervisor := supervisor.NewClusterInstallationSupervisor(mockStore, &mockClusterInstallationProvisioner{}, &mockAWS{}, "instanceID", logger)
	//	err := supervisor.Do()
	//	require.NoError(t, err)
	//
	//	<-mockStore.UnlockChan
	//	require.Equal(t, 2, mockStore.UpdateClusterInstallationCalls)
	//})
}

func TestBackupMetadataSupervisorSupervise(t *testing.T) {

	t.Run("trigger backup", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		mockBackupOp := &mockBackupOperator{}

		installation, clusterInstallation := setupBackupRequiredResources(t, sqlStore)

		backupMeta := &model.BackupMetadata{
			InstallationID: installation.ID,
			State:          model.BackupStateBackupRequested,
		}
		err := sqlStore.CreateBackupMetadata(backupMeta)
		require.NoError(t, err)

		backupSupervisor := supervisor.NewBackupSupervisor(sqlStore, mockBackupOp, &mockAWS{}, "instanceID", logger, nil)
		backupSupervisor.Supervise(backupMeta)

		// Assert
		backupMeta, err = sqlStore.GetBackupMetadata(backupMeta.ID, "")
		require.NoError(t, err)
		assert.Equal(t, model.BackupStateBackupInProgress, backupMeta.State)
		assert.Equal(t, clusterInstallation.ID, backupMeta.ClusterInstallationID)
		assert.Equal(t, "file-store.com", backupMeta.DataResidence.URL)
	})

	t.Run("do not trigger backup if installation not hibernated", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		mockBackupOp := &mockBackupOperator{}

		installation, _ := setupBackupRequiredResources(t, sqlStore)
		installation.State = model.InstallationStateStable
		err := sqlStore.UpdateInstallationState(installation)
		require.NoError(t, err)

		backupMeta := &model.BackupMetadata{
			InstallationID: installation.ID,
			State:          model.BackupStateBackupRequested,
		}
		err = sqlStore.CreateBackupMetadata(backupMeta)
		require.NoError(t, err)

		backupSupervisor := supervisor.NewBackupSupervisor(sqlStore, mockBackupOp, &mockAWS{}, "instanceID", logger, nil)
		backupSupervisor.Supervise(backupMeta)

		// Assert
		backupMeta, err = sqlStore.GetBackupMetadata(backupMeta.ID, "")
		require.NoError(t, err)
		assert.Equal(t, model.BackupStateBackupRequested, backupMeta.State)
	})

	t.Run("set backup as failed if installation deleted", func(t *testing.T) {
		logger := testlib.MakeLogger(t)
		sqlStore := store.MakeTestSQLStore(t, logger)
		mockBackupOp := &mockBackupOperator{}

		backupMeta := &model.BackupMetadata{
			InstallationID: "deleted-installation-id",
			State:          model.BackupStateBackupRequested,
		}
		err := sqlStore.CreateBackupMetadata(backupMeta)
		require.NoError(t, err)

		backupSupervisor := supervisor.NewBackupSupervisor(sqlStore, mockBackupOp, &mockAWS{}, "instanceID", logger, nil)
		backupSupervisor.Supervise(backupMeta)

		// Assert
		backupMeta, err = sqlStore.GetBackupMetadata(backupMeta.ID, "")
		require.NoError(t, err)
		assert.Equal(t, model.BackupStateBackupFailed, backupMeta.State)
	})

	t.Run("check backup status", func(t *testing.T) {
		for _, testCase := range []struct {
			description   string
			mockBackupOp  *mockBackupOperator
			expectedState model.BackupState
		}{
			{
				description:   "when backup finished",
				mockBackupOp:  &mockBackupOperator{BackupStartTime: 100},
				expectedState: model.BackupStateBackupSucceeded,
			},
			{
				description:   "when still in progress",
				mockBackupOp:  &mockBackupOperator{BackupStartTime: -1},
				expectedState: model.BackupStateBackupInProgress,
			},
			{
				description:   "when non terminal error",
				mockBackupOp:  &mockBackupOperator{BackupStartTime: -1, err: errors.New("some error")},
				expectedState: model.BackupStateBackupInProgress,
			},
			{
				description:   "when terminal error",
				mockBackupOp:  &mockBackupOperator{BackupStartTime: -1, err: provisioner.ErrJobBackoffLimitReached},
				expectedState: model.BackupStateBackupFailed,
			},
		} {
			t.Run(testCase.description, func(t *testing.T) {
				logger := testlib.MakeLogger(t)
				sqlStore := store.MakeTestSQLStore(t, logger)

				installation, clusterInstallation := setupBackupRequiredResources(t, sqlStore)

				backupMeta := &model.BackupMetadata{
					InstallationID:        installation.ID,
					ClusterInstallationID: clusterInstallation.ID,
					State:                 model.BackupStateBackupInProgress,
				}
				err := sqlStore.CreateBackupMetadata(backupMeta)
				require.NoError(t, err)

				backupSupervisor := supervisor.NewBackupSupervisor(sqlStore, testCase.mockBackupOp, &mockAWS{}, "instanceID", logger, nil)
				backupSupervisor.Supervise(backupMeta)

				// Assert
				backupMeta, err = sqlStore.GetBackupMetadata(backupMeta.ID, "")
				require.NoError(t, err)
				assert.Equal(t, testCase.expectedState, backupMeta.State)

				if testCase.mockBackupOp.BackupStartTime > 0 {
					assert.Equal(t, testCase.mockBackupOp.BackupStartTime, backupMeta.StartAt)
				}
			})
		}
	})

	//expectClusterInstallationState := func(t *testing.T, sqlStore *store.SQLStore, clusterInstallation *model.ClusterInstallation, expectedState string) {
	//	t.Helper()
	//
	//	clusterInstallation, err := sqlStore.GetClusterInstallation(clusterInstallation.ID)
	//	require.NoError(t, err)
	//	require.Equal(t, expectedState, clusterInstallation.State)
	//}
	//
	//t.Run("missing cluster", func(t *testing.T) {
	//	testCases := []struct {
	//		Description   string
	//		InitialState  string
	//		ExpectedState string
	//	}{
	//		{"on create", model.ClusterInstallationStateCreationRequested, model.ClusterInstallationStateCreationFailed},
	//		{"on delete", model.ClusterInstallationStateDeletionRequested, model.ClusterInstallationStateDeletionFailed},
	//	}
	//
	//	for _, tc := range testCases {
	//		t.Run(tc.Description, func(t *testing.T) {
	//			logger := testlib.MakeLogger(t)
	//			sqlStore := store.MakeTestSQLStore(t, logger)
	//			supervisor := supervisor.NewClusterInstallationSupervisor(sqlStore, &mockClusterInstallationProvisioner{}, &mockAWS{}, "instanceID", logger)
	//
	//			installation := &model.Installation{}
	//			err := sqlStore.CreateInstallation(installation, nil)
	//			require.NoError(t, err)
	//
	//			clusterInstallation := &model.ClusterInstallation{
	//				ClusterID:      model.NewID(),
	//				InstallationID: installation.ID,
	//				Namespace:      "namespace",
	//				State:          tc.InitialState,
	//			}
	//			err = sqlStore.CreateClusterInstallation(clusterInstallation)
	//			require.NoError(t, err)
	//
	//			supervisor.Supervise(clusterInstallation)
	//			expectClusterInstallationState(t, sqlStore, clusterInstallation, tc.ExpectedState)
	//		})
	//	}
	//})
	//
	//t.Run("missing installation", func(t *testing.T) {
	//	testCases := []struct {
	//		Description   string
	//		InitialState  string
	//		ExpectedState string
	//	}{
	//		{"on create", model.ClusterInstallationStateCreationRequested, model.ClusterInstallationStateCreationFailed},
	//		{"on delete", model.ClusterInstallationStateDeletionRequested, model.ClusterInstallationStateDeletionFailed},
	//	}
	//
	//	for _, tc := range testCases {
	//		t.Run(tc.Description, func(t *testing.T) {
	//			logger := testlib.MakeLogger(t)
	//			sqlStore := store.MakeTestSQLStore(t, logger)
	//			supervisor := supervisor.NewClusterInstallationSupervisor(sqlStore, &mockClusterInstallationProvisioner{}, &mockAWS{}, "instanceID", logger)
	//
	//			cluster := &model.Cluster{}
	//			err := sqlStore.CreateCluster(cluster, nil)
	//			require.NoError(t, err)
	//
	//			clusterInstallation := &model.ClusterInstallation{
	//				ClusterID:      cluster.ID,
	//				InstallationID: model.NewID(),
	//				Namespace:      "namespace",
	//				State:          tc.InitialState,
	//			}
	//			err = sqlStore.CreateClusterInstallation(clusterInstallation)
	//			require.NoError(t, err)
	//
	//			supervisor.Supervise(clusterInstallation)
	//			expectClusterInstallationState(t, sqlStore, clusterInstallation, tc.ExpectedState)
	//		})
	//	}
	//})
	//
	//t.Run("transition", func(t *testing.T) {
	//	testCases := []struct {
	//		Description   string
	//		InitialState  string
	//		ExpectedState string
	//	}{
	//		{"unexpected state", model.ClusterInstallationStateStable, model.ClusterInstallationStateStable},
	//		{"creation requested", model.ClusterInstallationStateCreationRequested, model.ClusterInstallationStateReconciling},
	//		{"creation reconciling", model.ClusterInstallationStateReconciling, model.ClusterInstallationStateStable},
	//		{"deletion requested", model.ClusterInstallationStateDeletionRequested, model.ClusterInstallationStateDeleted},
	//	}
	//
	//	for _, tc := range testCases {
	//		t.Run(tc.Description, func(t *testing.T) {
	//			logger := testlib.MakeLogger(t)
	//			sqlStore := store.MakeTestSQLStore(t, logger)
	//			supervisor := supervisor.NewClusterInstallationSupervisor(sqlStore, &mockClusterInstallationProvisioner{}, &mockAWS{}, "instanceID", logger)
	//
	//			cluster := &model.Cluster{}
	//			err := sqlStore.CreateCluster(cluster, nil)
	//			require.NoError(t, err)
	//
	//			installation := &model.Installation{}
	//			err = sqlStore.CreateInstallation(installation, nil)
	//			require.NoError(t, err)
	//
	//			clusterInstallation := &model.ClusterInstallation{
	//				ClusterID:      cluster.ID,
	//				InstallationID: installation.ID,
	//				Namespace:      "namespace",
	//				State:          tc.InitialState,
	//			}
	//			err = sqlStore.CreateClusterInstallation(clusterInstallation)
	//			require.NoError(t, err)
	//
	//			supervisor.Supervise(clusterInstallation)
	//			expectClusterInstallationState(t, sqlStore, clusterInstallation, tc.ExpectedState)
	//		})
	//	}
	//})
	//
	//t.Run("state has changed since cluster installation was selected to be worked on", func(t *testing.T) {
	//	logger := testlib.MakeLogger(t)
	//	sqlStore := store.MakeTestSQLStore(t, logger)
	//	supervisor := supervisor.NewClusterInstallationSupervisor(sqlStore, &mockClusterInstallationProvisioner{}, &mockAWS{}, "instanceID", logger)
	//
	//	cluster := &model.Cluster{}
	//	err := sqlStore.CreateCluster(cluster, nil)
	//	require.NoError(t, err)
	//
	//	installation := &model.Installation{}
	//	err = sqlStore.CreateInstallation(installation, nil)
	//	require.NoError(t, err)
	//
	//	clusterInstallation := &model.ClusterInstallation{
	//		ClusterID:      cluster.ID,
	//		InstallationID: installation.ID,
	//		Namespace:      "namespace",
	//		State:          model.ClusterInstallationStateReconciling,
	//	}
	//	err = sqlStore.CreateClusterInstallation(clusterInstallation)
	//	require.NoError(t, err)
	//
	//	// The stored cluster installation is ClusterInstallationStateReconciling,
	//	// so we will pass in a cluster installation with state of
	//	// ClusterInstallationStateCreationRequested to simulate stale state.
	//	clusterInstallation.State = model.ClusterInstallationStateCreationRequested
	//
	//	supervisor.Supervise(clusterInstallation)
	//	expectClusterInstallationState(t, sqlStore, clusterInstallation, model.ClusterInstallationStateReconciling)
	//})
}

func setupBackupRequiredResources(t *testing.T, sqlStore *store.SQLStore) (*model.Installation, *model.ClusterInstallation) {
	installation := testlib.CreateBackupCompatibleInstallation(t, sqlStore)

	cluster := &model.Cluster{}
	err := sqlStore.CreateCluster(cluster, nil)
	require.NoError(t, err)

	clusterInstallation := &model.ClusterInstallation{InstallationID: installation.ID, ClusterID: cluster.ID}
	err = sqlStore.CreateClusterInstallation(clusterInstallation)
	require.NoError(t, err)

	return installation, clusterInstallation
}
