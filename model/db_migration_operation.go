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

// TODO: test
// NewDBMigrationOperationFromReader will create a DBMigrationOperation from an
// io.Reader with JSON data.
func NewDBMigrationOperationFromReader(reader io.Reader) (*DBMigrationOperation, error) {
	var dBMigrationOperation DBMigrationOperation
	err := json.NewDecoder(reader).Decode(&dBMigrationOperation)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode DBMigrationOperation")
	}

	return &dBMigrationOperation, nil
}

// NewDBMigrationOperationsFromReader will create a slice of DBMigrationOperations from an
// io.Reader with JSON data.
func NewDBMigrationOperationsFromReader(reader io.Reader) ([]*DBMigrationOperation, error) {
	dBMigrationOperations := []*DBMigrationOperation{}
	err := json.NewDecoder(reader).Decode(&dBMigrationOperations)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode DBMigrationOperations")
	}

	return dBMigrationOperations, nil
}

// RestoreStrategy
// - DatabaseType
// - ? BackupType?


// Validation
/*

	- Installation cannot be in DestinationDB
	- Both DB types need to be the same
	- Installation not in MigratedInstallationIDs - could not be restore?

	???
	- Both Databases in the same VPC? -
*/

// Flow
/*

	- Create Backup for the Installation
	- Wait for backup done | Create new logical DB in other cluster (with user etc)
	- Switch Installation to new DB cluster
		- Add Installation to MigratedInstallationIDs of old MultitenantDB
	- Recreate Secrets - Update Cluster Installation?
		- Do I need to delete old? or is it the same name? - I think the same
	- Restore Installation from specific backup
	- (optional - based on param) Cleanup old database


*/

// Other
/*

	- Add MigratedInstallationIDs to MultitenantDatabase? - to not to delete data right away?
	- Rollback migration option? - Switch Installation to previous DB
		- New data is lost.
		- Need to be in MigratedInstallationIDs on the old one.
*/
