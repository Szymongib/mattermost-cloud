package supervisor

//// TODO: decide where to put this file
//
//type dbMigration struct {
// 	store *store.SQLStore
//}
//
//func (d *dbMigration) migrateDB(installation *model.Installation, dbMigration model.DBMigrationRequest) error {
//
//	// Backup Installation
//
//	// Backup done
//
//	err := d.switchDB(installation, dbMigration)
//
//
//	// Here run restore
//
//	// Optionaly cleanup old database
//}
//
//func (d *dbMigration) requestBackup(installation *model.Installation, dbMigration model.DBMigrationRequest) (string, error) {
//	if err := model.EnsureBackupCompatible(installation); err != nil {
//		return "", errors.Wrap(err, "installation cannot be backed up") // TODO: current state
//	}
//
//	backup := &model.InstallationBackup{
//		InstallationID: installation.ID,
//		State:          model.InstallationBackupStateBackupRequested,
//	}
//
//	// TODO: this should be single transaction probably?
//	err := d.store.CreateInstallationBackup(backup)
//	if err != nil {
//		return "", errors.Wrap(err, "failed to create installation backup") // TODO: current state
//	}
//
//	dbMigration.BackupID = backup.ID
//	// TODO: call update
//
//	return model.InstallationStateDBMigrationDatabaseBackup
//}
//
//func (d *dbMigration) switchDB(installation *model.Installation, dbMigration model.DBMigrationRequest) error {
//	if installation.Database == model.InstallationDatabaseMultiTenantRDSPostgres &&
//		dbMigration.DestinationDatabase == model.InstallationDatabaseMultiTenantRDSPostgres {
//		return d.migrateMultitenantDB(installation, dbMigration)
//	}
//
//	return errors.Errorf("database migration not supported from %q to %q database type", installation.Database, dbMigration.DestinationDatabase)
//}
//
//func (d *dbMigration) migrateMultitenantDB(installation *model.Installation, dbMigration model.DBMigrationRequest) error {
//
//	err := d.store.SwitchInstallationDatabase(installation.ID, dbMigration.DestinationMultitenant.DatabaseID)
//	if err != nil {
//		return errors.Wrap(err, "failed to switch multitenant database for installation")
//	}
//	//installation.Database = model.InstallationDatabaseMultiTenantRDSPostgres
//	//err = d.store.UpdateInstallation(installation)
//
//	// TODO: update installation database type in db
//
//	// Now provision new database
//	// Generate secrets
//	// Update cluster installations
//}
//
//
