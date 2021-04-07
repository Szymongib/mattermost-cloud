package model

// TODO: questions
// - Can you migrate between different types?


type DBMigrationRequest struct {
	InstallationID string

	BackupID string

	DestinationDatabase string // DB type

	DestinationMultitenant DestinationMultiTenantDB

	//DestinationMultiTenantMultiSchema

	DestinationSingleTenant DestinationSingleTenantDB


	DestinationMultitenantDatabaseID string
}

type DestinationMultiTenantDB struct {
	DatabaseID string
}

type DestinationSingleTenantDB struct {
	DatabaseID string
}

// RestoreStrategy
// - DatabaseType
// - ? BackupType?

type DBMigration struct {
	ID string

	InstallationID string
	SourceDatabaseID string
	DestinationDatabaseID string

	// VPC ID ?

	State string

	// InstallationBackupID is an id of InstallationBackup created as a first step of migration.
	InstallationBackupID string


}

type DBMigrationState string

const (
	DMMigrationStateRequested DBMigrationState = "migration-requested"
	DMMigrationStateInProgress DBMigrationState = "migration-in-progress"
	DMMigrationStateSucceeded DBMigrationState = "succeeded"
	DMMigrationStateFailed DBMigrationState = "failed"
)

// Validation
/*

	- Installation cannot be in DestinationDB
	- Both DB types need to be the same
	- Installation not in MigratedInstallationIDs - could not be restore?


	???
	- Both Databases in the same VPC? - Chyba nie


*/

// Constraints
/*

	- Installation needs to be hibernated the whole time??

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
