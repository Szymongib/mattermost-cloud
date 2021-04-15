package model

import (
	"encoding/json"
	"github.com/pkg/errors"
	"io"
)

// TODO: remove installation prefix from everything
type DBMigrationOperation struct {
	ID string
	InstallationID string
	RequestAt int64
	State DBMigrationOperationState

	SourceDatabase string // TODO: set based on installation
	DestinationDatabase string // DB type

	SourceMultiTenant *MultiTenantDBMigrationData

	DestinationMultiTenant *MultiTenantDBMigrationData

	BackupID string
	InstallationDBRestorationOperationID string

	// TODO: in te future add target Installation state?

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
	DBMigrationStateInstallationBackupInProgress,
	DBMigrationStateDatabaseSwitch,
	DBMigrationStateRefreshSecrets,
	DBMigrationStateTriggerRestoration,
	DbMigrationStateFinalizing,
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


// TODO: test
// NewInstallationDBRestorationOperationsFromReader will create a []*InstallationDBRestorationOperation from an
// io.Reader with JSON data.
func NewDBMigrationOperationsFromReader(reader io.Reader) ([]*DBMigrationOperation, error) {
	var restorations []*DBMigrationOperation
	err := json.NewDecoder(reader).Decode(&restorations)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode db migration operations")
	}

	return restorations, nil
}

// NewDBMigrationOperationFromReader will create a DBMigrationOperation from an
// io.Reader with JSON data.
func NewDBMigrationOperationFromReader(reader io.Reader) (*DBMigrationOperation, error) {
	var migrationOperation DBMigrationOperation
	err := json.NewDecoder(reader).Decode(&migrationOperation)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode migration operation")
	}

	return &migrationOperation, nil
}
