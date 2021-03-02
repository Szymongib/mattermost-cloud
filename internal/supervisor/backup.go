package supervisor

import (
	"github.com/mattermost/mattermost-cloud/internal/metrics"
	"github.com/mattermost/mattermost-cloud/internal/provisioner"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/webhook"
	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
	"time"
)

// TODO: this Supervisior could run few different backups at a time

// backupMetadataStore abstracts the database operations required to query installations.
type backupMetadataStore interface {
	GetUnlockedBackupMetadataPendingWork() ([]*model.BackupMetadata, error)
	GetBackupMetadata(id string) (*model.BackupMetadata, error)
	UpdateBackupMetadataState(backupMeta *model.BackupMetadata) error
	UpdateBackupSchedulingData(backupMeta *model.BackupMetadata) error
	UpdateBackupStartTime(backupMeta *model.BackupMetadata) error

	LockBackupMetadata(installationID, lockerID string) (bool, error)
	UnlockBackupMetadata(installationID, lockerID string, force bool) (bool, error)

	GetInstallation(installationID string, includeGroupConfig, includeGroupConfigOverrides bool) (*model.Installation, error)
	LockInstallation(installationID, lockerID string) (bool, error)
	UnlockInstallation(installationID, lockerID string, force bool) (bool, error)

	GetClusterInstallations(*model.ClusterInstallationFilter) ([]*model.ClusterInstallation, error)
	GetClusterInstallation(clusterInstallationID string) (*model.ClusterInstallation, error)
	LockClusterInstallations(clusterInstallationID []string, lockerID string) (bool, error)
	UnlockClusterInstallations(clusterInstallationID []string, lockerID string, force bool) (bool, error)

	GetClusters(clusterFilter *model.ClusterFilter) ([]*model.Cluster, error)
	GetCluster(id string) (*model.Cluster, error)
	LockCluster(clusterID, lockerID string) (bool, error)
	UnlockCluster(clusterID string, lockerID string, force bool) (bool, error)

	GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error)
}

type BackupOperator interface {
	TriggerBackup(backupMeta *model.BackupMetadata, cluster *model.Cluster, installation *model.Installation) (*model.S3DataResidence, error)
	CheckBackupStatus(backupMeta *model.BackupMetadata, cluster *model.Cluster) (int64, error)
}

// InstallationSupervisor finds installations pending work and effects the required changes.
//
// The degree of parallelism is controlled by a weighted semaphore, intended to be shared with
// other clients needing to coordinate background jobs.
type BackupSupervisor struct {
	store             backupMetadataStore
	//provisioner       installationProvisioner
	aws               aws.AWS
	instanceID        string
	//keepDatabaseData  bool // TODO: for deleting?
	//keepFilestoreData bool // TODO: for deleting?
	logger            log.FieldLogger
	metrics           *metrics.CloudMetrics

	backupOperator   BackupOperator
}

// NewInstallationSupervisor creates a new InstallationSupervisor.
func NewBackupSupervisor(
	store backupMetadataStore,
	//installationProvisioner installationProvisioner,
	backupOperator BackupOperator,
	aws aws.AWS,
	instanceID string,
	logger log.FieldLogger,
	metrics *metrics.CloudMetrics) *BackupSupervisor {
	return &BackupSupervisor{
		store:             store,
		//provisioner:       installationProvisioner,
		backupOperator: backupOperator,
		aws:               aws,
		instanceID:        instanceID,
		//keepDatabaseData:  keepDatabaseData,
		//keepFilestoreData: keepFilestoreData,
		logger:            logger,
		metrics:           metrics,
	}
}

// Shutdown performs graceful shutdown tasks for the installation supervisor.
func (s *BackupSupervisor) Shutdown() {
	s.logger.Debug("Shutting down backup supervisor")
}

// Do looks for work to be done on any pending installations and attempts to schedule the required work.
func (s *BackupSupervisor) Do() error {
	installations, err := s.store.GetUnlockedBackupMetadataPendingWork()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to query for backup pending work")
		return nil
	}

	for _, installation := range installations {
		s.Supervise(installation)
	}

	return nil
}

// Supervise schedules the required work on the given backup metadata.
func (s *BackupSupervisor) Supervise(backupMetadata *model.BackupMetadata) {
	logger := s.logger.WithFields(log.Fields{
		"backupMetadata": backupMetadata.ID,
	})

	lock := newBackupLock(backupMetadata.ID, s.instanceID, s.store, logger)
	if !lock.TryLock() {
		return
	}
	defer lock.Unlock()

	// Before working on the backupMetadata, it is crucial that we ensure that it
	// was not updated to a new state by another provisioning server.
	originalState := backupMetadata.State
	backupMetadata, err := s.store.GetBackupMetadata(backupMetadata.ID)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get refreshed backupMetadata")
		return
	}
	if backupMetadata.State != originalState {
		logger.WithField("oldBackupState", originalState).
			WithField("newBackupState", backupMetadata.State).
			Warn("Another provisioner has worked on this backupMetadata; skipping...")
		return
	}

	logger.Debugf("Supervising backupMetadata in state %s", backupMetadata.State)

	newState := s.transitionBackup(backupMetadata, s.instanceID, logger)

	backupMetadata, err = s.store.GetBackupMetadata(backupMetadata.ID)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get backup metadata and thus persist state %s", newState)
		return
	}

	if backupMetadata.State == newState {
		return
	}

	oldState := backupMetadata.State
	backupMetadata.State = newState

	err = s.store.UpdateBackupMetadataState(backupMetadata)
	if err != nil {
		logger.WithError(err).Errorf("Failed to set backup metadata state to %s", newState)
		return
	}

	environment, err := s.aws.GetCloudEnvironmentName()
	if err != nil {
		logger.WithError(err).Error("getting the AWS Cloud environment")
		return
	}

	webhookPayload := &model.WebhookPayload{
		Type:      model.TypeInstallation,
		ID:        backupMetadata.ID,
		NewState:  string(backupMetadata.State),
		OldState:  string(oldState),
		Timestamp: time.Now().UnixNano(),
		ExtraData: map[string]string{"Environment": environment},
	}
	err = webhook.SendToAllWebhooks(s.store, webhookPayload, logger.WithField("webhookEvent", webhookPayload.NewState))
	if err != nil {
		logger.WithError(err).Error("Unable to process and send webhooks")
	}

	logger.Debugf("Transitioned backup metadata from %s to %s", oldState, backupMetadata.State)
}


// transitionBackup works with the given installation to transition it to a final state.
func (s *BackupSupervisor) transitionBackup(backupMetadata *model.BackupMetadata, instanceID string, logger log.FieldLogger) model.BackupState {
	switch backupMetadata.State {
	case model.BackupStateBackupRequested:
		return s.triggerBackup(backupMetadata, instanceID, logger)

	case model.BackupStateBackupInProgress:
		return s.monitorBackup(backupMetadata, instanceID, logger)


		// TODO: will need to do deletion
	//case model.InstallationStateDeletionRequested,
	//	model.InstallationStateDeletionInProgress:
	//	return s.deleteInstallation(backupMetadata, instanceID, logger)
	//
	//case model.InstallationStateDeletionFinalCleanup:
	//	return s.finalDeletionCleanup(backupMetadata, logger)

	default:
		logger.Warnf("Found backup metadata pending work in unexpected state %s", backupMetadata.State)
		return backupMetadata.State
	}
}

func (s *BackupSupervisor) triggerBackup(backupMetadata *model.BackupMetadata, instanceID string, logger log.FieldLogger) model.BackupState {
	installation, err := s.store.GetInstallation(backupMetadata.InstallationID, false, false)
	if err != nil {
		logger.WithError(err).Error("Failed to get installation")
		return backupMetadata.State
	}
	if installation == nil {
		logger.Errorf("Failed to get installation, with id %q", backupMetadata.InstallationID)
		return backupMetadata.State
	}

	installationLock := newInstallationLock(installation.ID, instanceID, s.store, logger)
	if !installationLock.TryLock() {
		logger.Errorf("Failed to lock installation %s", installation.ID)
		return backupMetadata.State
	}
	defer installationLock.Unlock()

	err = model.EnsureBackupCompatible(installation)
	if err != nil {
		logger.WithError(err).Errorf("Installation is not backup compatible %s", installation.ID)
		return backupMetadata.State
	}

	clusterInstallationFilter := &model.ClusterInstallationFilter{
		InstallationID: installation.ID,
		PerPage:        model.AllPerPage,
	}
	clusterInstallations, err := s.store.GetClusterInstallations(clusterInstallationFilter)
	if err != nil {
		logger.WithError(err).Error("Failed to get cluster installations")
		return backupMetadata.State
	}

	if len(clusterInstallations) == 0 {
		logger.WithError(err).Error("Expected at least one cluster installation to run backup but found none")
		return backupMetadata.State
	}
	// TODO: should I lock the Cluster Installation? - Maybe not if I lock Installation?
	backupCI := clusterInstallations[0]
	ciLock := newClusterInstallationLock(backupCI.ID, instanceID, s.store, logger)
	if !ciLock.TryLock() {
		logger.Errorf("Failed to lock cluster installation %s", backupCI.ID)
		return backupMetadata.State
	}
	defer ciLock.Unlock()

	// TODO: I should probably put either cluster id or cluster installation ID to the BackupMetadata

	cluster, err := s.store.GetCluster(backupCI.ClusterID)
	if err != nil {
		logger.WithError(err).Error("Failed to get cluster")
		return backupMetadata.State
	}

	//// TODO: do I want to lock cluster? It may become problematic - no scheduling etc
	//clusterLock := newClusterLock(cluster.ID, instanceID, s.store, logger)
	//if !clusterLock.TryLock() {
	//	logger.Errorf("Failed to lock cluster %s", cluster.ID)
	//	return backupMetadata.State
	//}
	//defer clusterLock.Unlock()

	// Start backup here?

	dataRes, err := s.backupOperator.TriggerBackup(backupMetadata, cluster, installation)
	if err != nil {
		logger.WithError(err).Error("Failed to trigger backup")
		return backupMetadata.State
	}

	backupMetadata.DataResidence = dataRes
	backupMetadata.ClusterInstallationID = backupCI.ID

	err = s.store.UpdateBackupSchedulingData(backupMetadata)
	if err != nil {
		logger.Error("Failed to update backup metadata data residency")
		return backupMetadata.State
	}

	return model.BackupStateBackupInProgress
}

func (s *BackupSupervisor) monitorBackup(backupMetadata *model.BackupMetadata, instanceID string, logger log.FieldLogger) model.BackupState {

	// TODO: Do I need the Installation here?

	//installation, err := s.store.GetInstallation(backupMetadata.InstallationID, false, false)
	//if err != nil {
	//	logger.WithError(err).Error("Failed to get installation")
	//	return backupMetadata.State
	//}
	//if installation == nil {
	//	logger.Errorf("Failed to get installation, with id %q", backupMetadata.InstallationID)
	//	return backupMetadata.State
	//}

	// TODO: sanity check that CI ID is not empty?

	backupCI, err := s.store.GetClusterInstallation(backupMetadata.ClusterInstallationID)
	if err != nil {
		logger.WithError(err).Error("Failed to get cluster installations")
		return backupMetadata.State
	}


	// TODO: should I lock the Cluster Installation? - Maybe not?
	//backupCI := clusterInstallations[0]
	//ciLock := newClusterInstallationLock(backupCI.ID, instanceID, s.store, logger)
	//if !ciLock.TryLock() {
	//	logger.Errorf("Failed to lock cluster installation %s", backupCI.ID)
	//	return backupMetadata.State
	//}
	//defer ciLock.Unlock()

	cluster, err := s.store.GetCluster(backupCI.ClusterID)
	if err != nil {
		logger.WithError(err).Error("Failed to get cluster")
		return backupMetadata.State
	}

	startTime, err := s.backupOperator.CheckBackupStatus(backupMetadata, cluster)
	if err != nil {
		if err == provisioner.ErrJobBackoffLimitReached {
			logger.WithError(err).Error("Backup job backoff limit reached, backup failed")
			return model.BackupStateBackupFailed
		}
		logger.WithError(err).Error("Failed to check backup state")
		return backupMetadata.State
	}

	if startTime <= 0 {
		logger.Debugf("Backup in progress")
		return backupMetadata.State
	}

	backupMetadata.StartAt = startTime

	err = s.store.UpdateBackupStartTime(backupMetadata)
	if err != nil {
		logger.Error("Failed to update backup metadata data start time")
		return backupMetadata.State
	}

	return model.BackupStateBackupSucceeded
}
