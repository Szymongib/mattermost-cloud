package model

type InstallationMigration struct {
	ID string
	InstallationID string

	DestinationDatabase string // DB type
	DestinationMultiTenant DestinationMultiTenantDB

	BackupID string
	OldDatabaseID string
	// Restoration ID?

}

type DatabaseMigration struct {
	BackupID string

	DestinationDatabase string // DB type
	DestinationMultiTenant DestinationMultiTenantDB
	DestinationSingleTenant DestinationSingleTenantDB
}



