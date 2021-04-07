package supervisor

import (
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/webhook"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"time"
)

// installationDBRestorationStore abstracts the database operations required by the supervisor.
type installationDBRestorationStore interface {
	// TODO: adjust
	GetUnlockedInstallationDBRestorationsPendingWork() ([]*model.InstallationDBRestorationOperation, error)
	GetInstallationDBRestoration(id string) (*model.InstallationDBRestorationOperation, error)

	UpdateInstallationDBRestorationState(dbRestoration *model.InstallationDBRestorationOperation) error
	UpdateInstallationAndRestoration(installation *model.Installation, dbRestoration *model.InstallationDBRestorationOperation) error
	UpdateInstallationDBRestoration(dbRestoration *model.InstallationDBRestorationOperation) error

	LockInstallationDBRestorations(ids []string, lockerID string) (bool, error)
	UnlockInstallationDBRestorations(ids []string, lockerID string, force bool) (bool, error)

	UpdateInstallation(installation *model.Installation) error

	GetUnlockedInstallationBackupPendingWork() ([]*model.InstallationBackup, error)
	GetInstallationBackup(id string) (*model.InstallationBackup, error)
	UpdateInstallationBackupState(backupMeta *model.InstallationBackup) error
	UpdateInstallationBackupSchedulingData(backupMeta *model.InstallationBackup) error
	UpdateInstallationBackupStartTime(backupMeta *model.InstallationBackup) error
	DeleteInstallationBackup(id string) error

	LockInstallationBackups(backupIDs []string, lockerID string) (bool, error)
	UnlockInstallationBackups(backupIDs []string, lockerID string, force bool) (bool, error)

	GetInstallation(installationID string, includeGroupConfig, includeGroupConfigOverrides bool) (*model.Installation, error)
	LockInstallation(installationID, lockerID string) (bool, error)
	UnlockInstallation(installationID, lockerID string, force bool) (bool, error)

	GetClusterInstallations(*model.ClusterInstallationFilter) ([]*model.ClusterInstallation, error)
	GetClusterInstallation(clusterInstallationID string) (*model.ClusterInstallation, error)
	LockClusterInstallations(clusterInstallationID []string, lockerID string) (bool, error)
	UnlockClusterInstallations(clusterInstallationID []string, lockerID string, force bool) (bool, error)

	GetCluster(id string) (*model.Cluster, error)

	GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error)
}

// restoreOperator abstracts different restoration operations required by the installation db restoration supervisor.
type restoreOperator interface {
	TriggerRestore(installation *model.Installation, backup *model.InstallationBackup, cluster *model.Cluster) error
	CheckRestoreStatus(backupMeta *model.InstallationBackup, cluster *model.Cluster) (int64, error)
	CleanupRestoreJob(backup *model.InstallationBackup, cluster *model.Cluster) error
}

// InstallationDBRestorationSupervisor finds pending work and effects the required changes.
//
// The degree of parallelism is controlled by a weighted semaphore, intended to be shared with
// other clients needing to coordinate background jobs.
type InstallationDBRestorationSupervisor struct {
	store      installationDBRestorationStore
	aws               aws.AWS
	instanceID string
	logger     log.FieldLogger

	restoreOperator restoreOperator
}

// NewBackupSupervisor creates a new BackupSupervisor.
func NewInstallationDBRestorationSupervisor(
	store installationDBRestorationStore,
	aws aws.AWS,
	restoreOperator restoreOperator,
	instanceID string,
	logger log.FieldLogger) *InstallationDBRestorationSupervisor {
	return &InstallationDBRestorationSupervisor{
		store:          store,
		aws: aws,
		restoreOperator: restoreOperator,
		instanceID:     instanceID,
		logger:         logger,
	}
}

// Shutdown performs graceful shutdown tasks for the supervisor.
func (s *InstallationDBRestorationSupervisor) Shutdown() {
	s.logger.Debug("Shutting down installation db restoration supervisor")
}

// Do looks for work to be done on any pending backups and attempts to schedule the required work.
func (s *InstallationDBRestorationSupervisor) Do() error {
	installationDBRestorations, err := s.store.GetUnlockedInstallationDBRestorationsPendingWork()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to query for pending work")
		return nil
	}

	for _, restoration := range installationDBRestorations {
		s.Supervise(restoration)
	}

	return nil
}

// Supervise schedules the required work on the given backup.
func (s *InstallationDBRestorationSupervisor) Supervise(restoration *model.InstallationDBRestorationOperation) {
	logger := s.logger.WithFields(log.Fields{
		"restorationOperation": restoration.ID,
	})

	lock := newInstallationDBRestorationLock(restoration.ID, s.instanceID, s.store, logger)
	if !lock.TryLock() {
		return
	}
	defer lock.Unlock()

	// Before working on the restoration, it is crucial that we ensure that it
	// was not updated to a new state by another provisioning server.
	originalState := restoration.State
	restoration, err := s.store.GetInstallationDBRestoration(restoration.ID)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get refreshed restoration")
		return
	}
	if restoration.State != originalState {
		logger.WithField("oldRestorationState", originalState).
			WithField("newRestorationState", restoration.State).
			Warn("Another provisioner has worked on this restoration; skipping...")
		return
	}

	logger.Debugf("Supervising restoration in state %s", restoration.State)

	newState := s.transitionRestoration(restoration, s.instanceID, logger)

	restoration, err = s.store.GetInstallationDBRestoration(restoration.ID)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get restoration and thus persist state %s", newState)
		return
	}

	if restoration.State == newState {
		return
	}

	oldState := restoration.State
	restoration.State = newState

	err = s.store.UpdateInstallationDBRestorationState(restoration)
	if err != nil {
		logger.WithError(err).Errorf("Failed to set backup state to %s", newState)
		return
	}

	webhookPayload := &model.WebhookPayload{
		Type:      model.TypeInstallationDBRestoration,
		ID:        restoration.ID,
		NewState:  string(restoration.State),
		OldState:  string(oldState),
		Timestamp: time.Now().UnixNano(),
		ExtraData: map[string]string{"Environment": s.aws.GetCloudEnvironmentName()},
	}
	err = webhook.SendToAllWebhooks(s.store, webhookPayload, logger.WithField("webhookEvent", webhookPayload.NewState))
	if err != nil {
		logger.WithError(err).Error("Unable to process and send webhooks")
	}

	logger.Debugf("Transitioned restoration from %s to %s", oldState, restoration.State)
}

// transitionRestoration works with the given restoration to transition it to a final state.
func (s *InstallationDBRestorationSupervisor) transitionRestoration(restoration *model.InstallationDBRestorationOperation, instanceID string, logger log.FieldLogger) model.InstallationDBRestorationState {
	switch restoration.State {
	case model.InstallationDBRestorationStateStateRequested:
		return s.transitionInstallation(restoration, instanceID, logger)

	case model.InstallationDBRestorationStateStateTriggeringRestoration:
		return s.triggerRestoration(restoration, instanceID, logger)

	case model.InstallationDBRestorationStateStateInProgress:
		return s.checkRestorationStatus(restoration, instanceID, logger)

	case model.InstallationDBRestorationStateStateFinishing:
		return s.finalizeRestoration(restoration, instanceID, logger)

	default:
		logger.Warnf("Found restoration pending work in unexpected state %s", restoration.State)
		return restoration.State
	}
}


func (s *InstallationDBRestorationSupervisor) transitionInstallation(restoration *model.InstallationDBRestorationOperation, instanceID string, logger log.FieldLogger) model.InstallationDBRestorationState {
	installation, lock, err := s.getAndLockInstallation(restoration.InstallationID, instanceID, logger)
	if err != nil {
		logger.WithError(err).Error("failed to get and lock installation")
		return restoration.State
	}
	defer lock.Unlock()

	err = model.EnsureReadyForDBRestoration(installation)
	if err != nil {
		logger.WithError(err).Error("Installation cannot be restored")
		return restoration.State // TODO: or some fail error?
	}

	// TODO: maybe check and lock backup for restoration here?

	// TODO: or perhaps store the previous state?
	targetState := restoration.TargetInstallationState
	if targetState == "" {
		targetState, err = model.DetermineRestorationTargetState(installation)
		if err != nil {
			logger.WithError(err).Errorf("failed to determine target state of installation")
			return restoration.State // TODO: or some fail error?
		}
	}

	restoration.TargetInstallationState = targetState
	err = s.store.UpdateInstallationDBRestoration(restoration)
	if err != nil {
		logger.WithError(err).Error("failed to set target installation state for restoration")
		return restoration.State
	}

	installation.State = model.InstallationStateDBRestorationInProgress
	err = s.store.UpdateInstallation(installation)
	if err != nil {
		logger.WithError(err).Error("failed to set target installation state for restoration")
		return restoration.State
	}

	return model.InstallationDBRestorationStateStateTriggeringRestoration
}

func (s *InstallationDBRestorationSupervisor) triggerRestoration(restoration *model.InstallationDBRestorationOperation, instanceID string, logger log.FieldLogger) model.InstallationDBRestorationState  {
	installation, lock, err := s.getAndLockInstallation(restoration.InstallationID, instanceID, logger)
	if err != nil {
		logger.WithError(err).Error("failed to get and lock installation")
		return restoration.State
	}
	defer lock.Unlock()

	// TODO: should I check if installation is ok?

	backup, err := s.store.GetInstallationBackup(restoration.BackupID)
	if err != nil {
		logger.WithError(err).Error("failed to get backup")
		return restoration.State
	}
	// TODO: this checks as a backup function
	if backup == nil {
		logger.Error("Restoration cannot be performed, backup not found")
		return model.InstallationStateRestorationInvalid // TODO: co z tym
	}
	if backup.State != model.InstallationBackupStateBackupSucceeded {
		logger.Error("Restoration cannot be performed, backup is not in Succeeded state, state: %q", backup.State)
		return model.InstallationStateRestorationInvalid
	}

	// TODO: probably lock the backup here, so it cannot be deleted
	// TODO: do I need to mark the backup somehow, so no one deletes it while restoring?

	restoreCI, ciLock, err := claimClusterInstallation(s.store, installation, instanceID, logger)
	if err != nil {
		logger.WithError(err).Error("Failed to claim Cluster Installation for restoration")
		return restoration.State
	}
	defer ciLock.Unlock()

	cluster, err := s.store.GetCluster(restoreCI.ClusterID)
	if err != nil {
		logger.WithError(err).Error("Failed to get cluster for restoration")
		return restoration.State
	}

	restoration.ClusterInstallationID = restoreCI.ID
	err = s.store.UpdateInstallationDBRestoration(restoration)
	if err != nil {
		logger.WithError(err).Error("Failed to update installation db restoration")
		return restoration.State
	}

	err = s.restoreOperator.TriggerRestore(installation,backup, cluster)
	if err != nil {
		logger.WithError(err).Error("Failed to trigger restoration job")
		return restoration.State
	}

	return model.InstallationStateDBRestorationInProgress
}

func (s *InstallationDBRestorationSupervisor) checkRestorationStatus(restoration *model.InstallationDBRestorationOperation, instanceID string, logger log.FieldLogger) model.InstallationDBRestorationState {

	// TODO: check if restoration meta ok?

	backup, err := s.store.GetInstallationBackup(restoration.BackupID)
	if err != nil {
		logger.WithError(err).Error("failed to get backup")
		return restoration.State
	}
	// TODO: check if backup ok

	cluster, err := s.getClusterForRestoration(restoration)
	if err != nil {
		logger.WithError(err).Error("Failed to get cluster for restoration")
		return restoration.State
	}

	completeAt, err := s.restoreOperator.CheckRestoreStatus(backup, cluster)
	if err != nil {
		logger.WithError(err).Error("Failed to check restoration status")
		return restoration.State
	}
	if completeAt <= 0 {
		logger.Info("Database restoration still in progress")
		return restoration.State
	}

	restoration.CompleteAt = completeAt
	err = s.store.UpdateInstallationDBRestoration(restoration)
	if err != nil {
		logger.WithError(err).Error("Failed to update restoration")
		return restoration.State
	}

	return model.InstallationDBRestorationStateStateFinishing
}

func (s *InstallationDBRestorationSupervisor) finalizeRestoration(restoration *model.InstallationDBRestorationOperation, instanceID string, logger log.FieldLogger) model.InstallationDBRestorationState {
	installation, lock, err := s.getAndLockInstallation(restoration.InstallationID, instanceID, logger)
	if err != nil {
		logger.WithError(err).Error("failed to get and lock installation")
		return restoration.State
	}
	defer lock.Unlock()

	installation.State = restoration.TargetInstallationState
	err = s.store.UpdateInstallation(installation)
	if err != nil {
		logger.WithError(err).Error("failed to set installation to target state after restore")
		return restoration.State
	}

	return model.InstallationDBRestorationStateStateSucceeded
}

func (s *InstallationDBRestorationSupervisor) getAndLockInstallation(installationID, instanceID string, logger log.FieldLogger) (*model.Installation, *installationLock, error) {
	installation, err := s.store.GetInstallation(installationID, false, false)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get installation")
	}
	if installation == nil {
		return nil, nil, errors.New("could not found the installation")
	}

	lock := newInstallationLock(installation.ID, instanceID, s.store, logger)
	if !lock.TryLock() {
		logger.Debugf("Failed to lock installation %s", installation.ID)
		return nil, nil, errors.New("failed to lock installation")
	}
	return installation, lock, nil
}

// TODO: same as getClusterForBackup - align it somehow - maybe split big interfaces to smaller
func (s *InstallationDBRestorationSupervisor) getClusterForRestoration(restoration *model.InstallationDBRestorationOperation) (*model.Cluster, error) {
	backupCI, err := s.store.GetClusterInstallation(restoration.ClusterInstallationID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get cluster installations")
	}

	cluster, err := s.store.GetCluster(backupCI.ClusterID)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get cluster")
	}

	return cluster, nil
}
