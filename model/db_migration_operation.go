package model

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
)

// DBMigrationOperation contains information about installation's database migration operation.
type DBMigrationOperation struct {
	ID             string
	InstallationID string
	RequestAt      int64
	State          DBMigrationOperationState
	// SourceDatabase is current Installation database.
	SourceDatabase string
	// DestinationDatabase is database type to which migration will be performed.
	DestinationDatabase string

	// For now only supported migration is from multi-tenant DB to multi-tenant DB.
	SourceMultiTenant                    *MultiTenantDBMigrationData
	DestinationMultiTenant               *MultiTenantDBMigrationData
	BackupID                             string
	InstallationDBRestorationOperationID string
	CompleteAt                           int64
	DeleteAt                             int64
	LockAcquiredBy                       *string
	LockAcquiredAt                       int64
}

// MultiTenantDBMigrationData represents migration data for Multi-tenant database.
type MultiTenantDBMigrationData struct {
	DatabaseID string
}

type DBMigrationOperationState string

// TODO: comments
const (
	DBMigrationStateRequested                    DBMigrationOperationState = "db-migration-requested"
	DBMigrationStateInstallationBackupInProgress DBMigrationOperationState = "db-migration-installation-backup-in-progress"

	DBMigrationStateDatabaseSwitch DBMigrationOperationState = "db-migration-database switch"

	DBMigrationStateRefreshSecrets DBMigrationOperationState = "db-migration-refresh-secrets"

	DBMigrationStateTriggerRestoration         DBMigrationOperationState = "db-migration-trigger-restoration"
	DBMigrationStateRestorationInProgress      DBMigrationOperationState = "db-migration-restoration-in-progress"
	DBMigrationStateUpdatingInstallationConfig DBMigrationOperationState = "db-migration-updating-installation-config"
	DBMigrationStateFinalizing                 DBMigrationOperationState = "db-migration-finalizing"

	DBMigrationStateFailing DBMigrationOperationState = "db-migration-failing"

	DBMigrationStateSucceeded DBMigrationOperationState = "db-migration-succeeded"
	DBMigrationStateFailed    DBMigrationOperationState = "db-migration-failed"
)

// AllInstallationBackupStatesPendingWork is a list of all backup states that
// the supervisor will attempt to transition towards stable on the next "tick".
var AllInstallationDBMigrationOperationsStatesPendingWork = []DBMigrationOperationState{
	DBMigrationStateRequested,
	DBMigrationStateInstallationBackupInProgress,
	DBMigrationStateDatabaseSwitch,
	DBMigrationStateRefreshSecrets,
	DBMigrationStateTriggerRestoration,
	DBMigrationStateFinalizing,
	DBMigrationStateRestorationInProgress,
	DBMigrationStateUpdatingInstallationConfig,
	DBMigrationStateFailing,
}

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
