// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	"github.com/mattermost/mattermost-cloud/internal/tools/utils"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/components"
	"github.com/mattermost/mattermost-cloud/internal/provisioner"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/webhook"
	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
)

// installationDBMigrationStore abstracts the database operations required by the supervisor.
type installationDBMigrationStore interface {
	GetUnlockedInstallationDBMigrationsPendingWork() ([]*model.DBMigrationOperation, error)
	GetInstallationDBMigration(id string) (*model.DBMigrationOperation, error)
	UpdateInstallationDBMigrationState(dbMigration *model.DBMigrationOperation) error
	UpdateInstallationDBMigration(dbMigration *model.DBMigrationOperation) error
	dBMigrationOperationLockStore

	TriggerInstallationRestoration(installation *model.Installation, backup *model.InstallationBackup) (*model.InstallationDBRestorationOperation, error)
	GetInstallationDBRestorationOperation(id string) (*model.InstallationDBRestorationOperation, error)
	UpdateInstallationDBRestorationOperationState(dbRestoration *model.InstallationDBRestorationOperation) error
	UpdateInstallationDBRestorationOperation(dbRestoration *model.InstallationDBRestorationOperation) error

	IsInstallationBackupRunning(installationID string) (bool, error)
	CreateInstallationBackup(backup *model.InstallationBackup) error
	GetInstallationBackup(id string) (*model.InstallationBackup, error)
	UpdateInstallationBackupState(backupMeta *model.InstallationBackup) error
	installationBackupLockStore

	GetInstallation(installationID string, includeGroupConfig, includeGroupConfigOverrides bool) (*model.Installation, error)
	UpdateInstallation(installation *model.Installation) error
	installationLockStore

	GetClusterInstallations(*model.ClusterInstallationFilter) ([]*model.ClusterInstallation, error)
	GetClusterInstallation(clusterInstallationID string) (*model.ClusterInstallation, error)
	clusterInstallationLockStore

	GetCluster(id string) (*model.Cluster, error)

	GetWebhooks(filter *model.WebhookFilter) ([]*model.Webhook, error)

	model.InstallationDatabaseStoreInterface
}

// TODO: clusterResourceProvisioner / SecretsProvisioner?
type dbMigrationProvisioner interface {
	ClusterInstallationProvisioner(version string) provisioner.ClusterInstallationProvisioner
}

type databaseProvider interface {
	GetDatabase(installationID, dbType string) model.Database
}

// DBMigrationSupervisor finds pending work and effects the required changes.
//
// The degree of parallelism is controlled by a weighted semaphore, intended to be shared with
// other clients needing to coordinate background jobs.
type DBMigrationSupervisor struct {
	store      installationDBMigrationStore
	aws        aws.AWS
	dbProvider databaseProvider
	instanceID string
	logger     log.FieldLogger

	// TODO: idealy remove and use cluster CI supervisor?
	dbMigrationProvisioner dbMigrationProvisioner
}

// NewBackupSupervisor creates a new BackupSupervisor.
func NewInstallationDBMigrationSupervisor(
	store installationDBMigrationStore,
	aws aws.AWS,
	dbProvider databaseProvider,
	instanceID string,
	provisioner dbMigrationProvisioner,
	logger log.FieldLogger) *DBMigrationSupervisor {
	return &DBMigrationSupervisor{
		store:                  store,
		aws:                    aws,
		dbProvider:             dbProvider,
		instanceID:             instanceID,
		logger:                 logger,
		dbMigrationProvisioner: provisioner,
	}
}

// Shutdown performs graceful shutdown tasks for the supervisor.
func (s *DBMigrationSupervisor) Shutdown() {
	s.logger.Debug("Shutting down installation db restoration supervisor")
}

// Do looks for work to be done on any pending backups and attempts to schedule the required work.
func (s *DBMigrationSupervisor) Do() error {
	installationDBMigrations, err := s.store.GetUnlockedInstallationDBMigrationsPendingWork()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to query for pending work")
		return nil
	}

	for _, migration := range installationDBMigrations {
		s.Supervise(migration)
	}

	return nil
}

// Supervise schedules the required work on the given backup.
func (s *DBMigrationSupervisor) Supervise(migration *model.DBMigrationOperation) {
	logger := s.logger.WithFields(log.Fields{
		"dbMigrationOperation": migration.ID,
	})

	lock := newDBMigrationOperationLock(migration.ID, s.instanceID, s.store, logger)
	if !lock.TryLock() {
		return
	}
	defer lock.Unlock()

	// Before working on the migration operation, it is crucial that we ensure that it
	// was not updated to a new state by another provisioning server.
	originalState := migration.State
	migration, err := s.store.GetInstallationDBMigration(migration.ID)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get refreshed migration")
		return
	}
	if migration.State != originalState {
		logger.WithField("oldRestorationState", originalState).
			WithField("newRestorationState", migration.State).
			Warn("Another provisioner has worked on this migration; skipping...")
		return
	}

	logger.Debugf("Supervising migration in state %s", migration.State)

	newState := s.transitionMigration(migration, s.instanceID, logger)

	migration, err = s.store.GetInstallationDBMigration(migration.ID)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get migration and thus persist state %s", newState)
		return
	}

	if migration.State == newState {
		return
	}

	oldState := migration.State
	migration.State = newState

	err = s.store.UpdateInstallationDBMigrationState(migration)
	if err != nil {
		logger.WithError(err).Errorf("Failed to set migration state to %s", newState)
		return
	}

	webhookPayload := &model.WebhookPayload{
		Type:      model.TypeDBMigration,
		ID:        migration.ID,
		NewState:  string(migration.State),
		OldState:  string(oldState),
		Timestamp: time.Now().UnixNano(),
		ExtraData: map[string]string{"Environment": s.aws.GetCloudEnvironmentName()},
	}
	err = webhook.SendToAllWebhooks(s.store, webhookPayload, logger.WithField("webhookEvent", webhookPayload.NewState))
	if err != nil {
		logger.WithError(err).Error("Unable to process and send webhooks")
	}

	logger.Debugf("Transitioned db migration from %s to %s", oldState, migration.State)
}

// transitionMigration works with the given db migration to transition it to a final state.
func (s *DBMigrationSupervisor) transitionMigration(dbMigration *model.DBMigrationOperation, instanceID string, logger log.FieldLogger) model.DBMigrationOperationState {
	switch dbMigration.State {
	case model.DBMigrationStateRequested:
		return s.triggerInstallationBackup(dbMigration, instanceID, logger)
	case model.DBMigrationStateInstallationBackupInProgress:
		return s.waitForInstallationBackup(dbMigration, instanceID, logger)
	case model.DBMigrationStateDatabaseSwitch:
		return s.switchDatabase(dbMigration, instanceID, logger)
	case model.DBMigrationStateRefreshSecrets:
		return s.refreshCredentials(dbMigration, instanceID, logger)
	case model.DBMigrationStateTriggerRestoration:
		return s.triggerInstallationRestoration(dbMigration, instanceID, logger)
	case model.DBMigrationStateRestorationInProgress:
		return s.waitForInstallationRestoration(dbMigration, instanceID, logger)
	case model.DbMigrationStateFinalizing:
		return s.finalizeMigration(dbMigration, instanceID, logger)
	case model.DBMigrationStateFailing:
		return s.failMigration(dbMigration, instanceID, logger)
	default:
		logger.Warnf("Found migration pending work in unexpected state %s", dbMigration.State)
		return dbMigration.State
	}
}

// TODO: This goes to API
//func (s *DBMigrationSupervisor) beginMigration(dbMigration *model.DBMigrationOperation, instanceID string, logger log.FieldLogger) model.DBMigrationOperationState {
//
//	installation, lock, err := getAndLockInstallation(s.store, dbMigration.InstallationID, instanceID, logger)
//	if err != nil {
//		logger.WithError(err).Error("failed to get and lock installation")
//		return dbMigration.State
//	}
//	defer lock.Unlock()
//
//	if installation.State != model.InstallationStateHibernating {
//		logger.Errorf("Cannot begin database migration, expected installation to be hibernated is in state %q", installation.State)
//		return dbMigration.State
//	}
//
//	installation.State = model.InstallationStateDBMigrationInProgress
//	err = s.store.UpdateInstallation(installation)
//	if err != nil {
//		logger.WithError(err).Error("Failed to update installation state")
//		return dbMigration.State
//	}
//
//	return model.DBMigrationStateTriggerInstallationBackup
//}

func (s *DBMigrationSupervisor) triggerInstallationBackup(dbMigration *model.DBMigrationOperation, instanceID string, logger log.FieldLogger) model.DBMigrationOperationState {

	// TODO: Allow passing backupID to migrate from?
	// Maybe do it as MVP?

	// TODO: not sure if I need this lock here
	installation, lock, err := getAndLockInstallation(s.store, dbMigration.InstallationID, instanceID, logger)
	if err != nil {
		logger.WithError(err).Error("failed to get and lock installation")
		return dbMigration.State
	}
	defer lock.Unlock()

	// TODO: This is not ideal, because the backup will start and in DB migration updated fails
	// backup will not be able to start until the previous one finishes.

	backup, err := components.TriggerInstallationBackup(s.store, installation)
	if err != nil {
		logger.WithError(err).Error("Failed to trigger installation backup")
		return dbMigration.State
	}

	dbMigration.BackupID = backup.ID
	err = s.store.UpdateInstallationDBMigration(dbMigration)
	if err != nil {
		logger.WithError(err).Error("Failed to set backup ID for DB migration")
		return dbMigration.State
	}

	return model.DBMigrationStateInstallationBackupInProgress
}

func (s *DBMigrationSupervisor) waitForInstallationBackup(dbMigration *model.DBMigrationOperation, instanceID string, logger log.FieldLogger) model.DBMigrationOperationState {
	backup, err := s.store.GetInstallationBackup(dbMigration.BackupID)
	if err != nil {
		logger.WithError(err).Error("Failed to get installation backup")
		return dbMigration.State
	}

	switch backup.State {
	case model.InstallationBackupStateBackupSucceeded:
		logger.Info("Backup for migration finished successfully")
		return model.DBMigrationStateDatabaseSwitch
	case model.InstallationBackupStateBackupFailed:
		logger.Error("Backup for migration failed")
		return model.DBMigrationStateFailing
	case model.InstallationBackupStateBackupInProgress, model.InstallationBackupStateBackupRequested:
		logger.Debug("Backup for migration in progress")
		return dbMigration.State
	default:
		logger.Errorf("Unexpected state of installation backup for migration: %q", backup.State)
		return dbMigration.State
	}
}

func (s *DBMigrationSupervisor) switchDatabase(dbMigration *model.DBMigrationOperation, instanceID string, logger log.FieldLogger) model.DBMigrationOperationState {

	installation, lock, err := getAndLockInstallation(s.store, dbMigration.InstallationID, instanceID, logger)
	if err != nil {
		logger.WithError(err).Error("failed to get and lock installation")
		return dbMigration.State
	}
	defer lock.Unlock()

	// Validate migration is ok?

	sourceDB := s.dbProvider.GetDatabase(installation.ID, dbMigration.SourceDatabase)

	err = sourceDB.MigrateOut(s.store, dbMigration, logger)
	if err != nil {
		logger.WithError(err).Errorf("Failed to migrate installation out of database")
		return dbMigration.State
	}

	destinationDB := s.dbProvider.GetDatabase(installation.ID, dbMigration.DestinationDatabase)
	err = destinationDB.MigrateTo(s.store, dbMigration, logger)
	if err != nil {
		logger.WithError(err).Errorf("Failed to migrate installation to database")
		return dbMigration.State
	}

	installation.Database = dbMigration.DestinationDatabase
	err = s.store.UpdateInstallation(installation)
	if err != nil {
		logger.WithError(err).Errorf("Failed to switch database for installation")
		return dbMigration.State
	}

	return model.DBMigrationStateRefreshSecrets
}

func (s *DBMigrationSupervisor) refreshCredentials(dbMigration *model.DBMigrationOperation, instanceID string, logger log.FieldLogger) model.DBMigrationOperationState {
	installation, lock, err := getAndLockInstallation(s.store, dbMigration.InstallationID, instanceID, logger)
	if err != nil {
		logger.WithError(err).Error("failed to get and lock installation")
		return dbMigration.State
	}
	defer lock.Unlock()

	cis, err := s.store.GetClusterInstallations(&model.ClusterInstallationFilter{InstallationID: installation.ID, Paging: model.AllPagesNotDeleted()})
	if err != nil {
		logger.WithError(err).Errorf("Failed to get cluster installations")
		return dbMigration.State
	}

	for _, ci := range cis {
		cluster, err := s.store.GetCluster(ci.ClusterID)
		if err != nil {
			logger.WithError(err).Errorf("Failed to get cluster")
			return dbMigration.State
		}

		err = s.dbMigrationProvisioner.ClusterInstallationProvisioner(installation.CRVersion).
			RefreshSecrets(cluster, installation, cis[0])
		if err != nil {
			logger.WithError(err).Errorf("Failed to refresh credentials of cluster installation")
			return dbMigration.State
		}
	}

	// TODO: here it will be immediate cause installation is scaled down? What will happen with update job etc?
	// But secrets will be fine anyway

	// TODO: anything else????

	return model.DBMigrationStateTriggerRestoration
}

func (s *DBMigrationSupervisor) triggerInstallationRestoration(dbMigration *model.DBMigrationOperation, instanceID string, logger log.FieldLogger) model.DBMigrationOperationState {

	// TODO: not sure if I need this lock here
	installation, lock, err := getAndLockInstallation(s.store, dbMigration.InstallationID, instanceID, logger)
	if err != nil {
		logger.WithError(err).Error("failed to get and lock installation")
		return dbMigration.State
	}
	defer lock.Unlock()

	// TODO: This is not ideal, because the backup will start and in DB migration updated fails
	// backup will not be able to start until the previous one finishes.

	backup, err := s.store.GetInstallationBackup(dbMigration.BackupID)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get backup")
		return dbMigration.State
	}
	if backup == nil {
		logger.Errorf("Backup not found on restoration phase")
		return model.DBMigrationStateFailing
	}

	dbRestoration, err := components.TriggerInstallationDBRestoration(s.store, installation, backup)
	if err != nil {
		s.logger.WithError(err).Error("Failed to trigger installation db restoration")
		return dbMigration.State
	}

	dbMigration.InstallationDBRestorationOperationID = dbRestoration.ID
	err = s.store.UpdateInstallationDBMigration(dbMigration)
	if err != nil {
		logger.WithError(err).Error("Failed to set restoration operation ID for DB migration")
		return dbMigration.State
	}

	return model.DBMigrationStateRestorationInProgress
}

func (s *DBMigrationSupervisor) waitForInstallationRestoration(dbMigration *model.DBMigrationOperation, instanceID string, logger log.FieldLogger) model.DBMigrationOperationState {
	restoration, err := s.store.GetInstallationDBRestorationOperation(dbMigration.InstallationDBRestorationOperationID)
	if err != nil {
		logger.WithError(err).Error("Failed to get installation restoration")
		return dbMigration.State
	}

	switch restoration.State {
	case model.InstallationDBRestorationStateSucceeded:
		logger.Info("Restoration for migration finished successfully")
		return model.DbMigrationStateFinalizing
	case model.InstallationDBRestorationStateFailed, model.InstallationDBRestorationStateInvalid:
		logger.Error("Restoration for migration failed or is invalid")
		return model.DBMigrationStateFailing
	default:
		logger.Debug("Restoration for migration in progress")
		return dbMigration.State
	}
}

func (s *DBMigrationSupervisor) finalizeMigration(dbMigration *model.DBMigrationOperation, instanceID string, logger log.FieldLogger) model.DBMigrationOperationState {

	installation, lock, err := getAndLockInstallation(s.store, dbMigration.InstallationID, instanceID, logger)
	if err != nil {
		logger.WithError(err).Error("failed to get and lock installation")
		return dbMigration.State
	}
	defer lock.Unlock()

	// TODO: maybe allow different states in future
	installation.State = model.InstallationStateHibernating
	err = s.store.UpdateInstallation(installation)
	if err != nil {
		logger.WithError(err).Errorf("Failed to set installation back to hibernating after migration")
		return dbMigration.State
	}

	dbMigration.CompleteAt = utils.GetMillis()
	err = s.store.UpdateInstallationDBMigration(dbMigration)
	if err != nil {
		logger.WithError(err).Errorf("Failed to set complete at for db migration")
		return dbMigration.State
	}

	return model.DBMigrationStateSucceeded
}

func (s *DBMigrationSupervisor) failMigration(dbMigration *model.DBMigrationOperation, instanceID string, logger log.FieldLogger) model.DBMigrationOperationState {

	installation, lock, err := getAndLockInstallation(s.store, dbMigration.InstallationID, instanceID, logger)
	if err != nil {
		logger.WithError(err).Error("failed to get and lock installation")
		return dbMigration.State
	}
	defer lock.Unlock()

	// TODO: maybe allow different states in future
	installation.State = model.InstallationStateDBMigrationFailed
	err = s.store.UpdateInstallation(installation)
	if err != nil {
		logger.WithError(err).Errorf("Failed to set installation back to hibernating after migration")
		return dbMigration.State
	}

	// TODO: anything eles?

	return model.DBMigrationStateFailed
}
