package model

// TODO: remove installation prefix from everything
type DBMigrationOperation struct {
	ID string
	InstallationID string
	RequestAt int64
	State DBMigrationOperationState

	SourceDatabase string // TODO: set based on installation
	DestinationDatabase string // DB type

	SourceMultiTenant MultiTenantDBMigrationData

	DestinationMultiTenant MultiTenantDBMigrationData

	BackupID string
	OldDatabaseID string
	InstallationDBRestorationOperationID string

	CompleteAt int64

	DeleteAt int64
	LockAcquiredBy             *string
	LockAcquiredAt             int64
}

type MultiTenantDBMigrationData struct {
	DatabaseID string
}

type DBMigrationOperationState string

// TODO: comments
const (
	DBMigrationStateRequested  DBMigrationOperationState = "db-migration-requested"
	DBMigrationStateTriggerInstallationBackup  DBMigrationOperationState = "db-migration-trigger-installation-backup"
	DBMigrationStateInstallationBackupInProgress  DBMigrationOperationState = "db-migration-installation-backup-in-progress"

	DBMigrationStateDatabaseSwitch  DBMigrationOperationState = "db-migration-database switch"

	DBMigrationStateRefreshSecrets  DBMigrationOperationState = "db-migration-refresh-secrets"

	DBMigrationStateTriggerRestoration    DBMigrationOperationState = "db-migration-trigger-restoration"
	DBMigrationStateRestorationInProgress DBMigrationOperationState = "db-migration-restoration-in-progress"
	DbMigrationStateFinalizing DBMigrationOperationState = "db-migration-finalizing"

	DBMigrationStateFailing  DBMigrationOperationState = "db-migration-failing"

	DBMigrationStateSucceeded  DBMigrationOperationState = "db-migration-succeeded"
	DBMigrationStateFailed     DBMigrationOperationState = "db-migration-failed"
)

// AllInstallationBackupStatesPendingWork is a list of all backup states that
// the supervisor will attempt to transition towards stable on the next "tick".
var AllInstallationDBMigrationOperationsStatesPendingWork = []DBMigrationOperationState{
	DBMigrationStateRequested,
	DBMigrationStateTriggerInstallationBackup,
	DBMigrationStateInstallationBackupInProgress,
	DBMigrationStateDatabaseSwitch,
	DBMigrationStateRefreshSecrets,
	DBMigrationStateTriggerRestoration,
	DBMigrationStateRestorationInProgress,
	DBMigrationStateFailing,
}


//type DatabaseMigration struct {
//	BackupID string
//
//	DestinationDatabase string // DB type
//	DestinationMultiTenant DestinationMultiTenantDB
//	DestinationSingleTenant DestinationSingleTenantDB
//}
//
//


