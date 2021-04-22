package supervisor

//
//func (s *InstallationSupervisor) migrationRequested(installation *model.Installation, logger log.FieldLogger) string {
//
//	// get migration from DB
//	migration := &model.InstallationMigration{}
//
//	// Based on props decide what are we migrating and what steps are necessary
//
//
//
//}
//
//func (s *InstallationSupervisor) triggerInstallationBackup(installation *model.Installation, logger log.FieldLogger) string {
//
//	// get migration from DB
//	migration := &model.InstallationMigration{}
//
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
//	migration.ID = backup.ID
//	// TODO update
//
//}
//
//func (s *InstallationSupervisor) waitForInstallationBackup(installation *model.Installation, logger log.FieldLogger) string {
//
//	// get migration from DB
//	migration := &model.InstallationMigration{}
//
//	backup, err := s.store.GetInstallationBackup(migration.BackupID)
//	if err != nil {
//		// TODO
//	}
//
//	if backup.State == Success {
//		Ok
//	}
//	if Fail {
//		NotOk
//	}
//	return Wait
//}
//
//func (s *InstallationSupervisor) backupCompleted(installation *model.Installation, logger log.FieldLogger) string {
//
//	// If needed switch clusters
//
//}
//
//func (s *InstallationSupervisor) switchDatabases(installation *model.Installation, logger log.FieldLogger) string {
//	// get migration from DB
//	migration := &model.InstallationMigration{}
//
//	if installation.Database == model.InstallationDatabaseMultiTenantRDSPostgres &&
//		migration.DestinationDatabase == model.InstallationDatabaseMultiTenantRDSPostgres {
//		err := s.migrateMultitenantPostgresDB(installation, dbMigration, logger)
//		if err != nil {
//			// TODO
//		}
//	} else {
//		return errors.Errorf("database migration not supported from %q to %q database type", installation.Database, dbMigration.DestinationDatabase)
//	}
//
//	// Provision new databse
//	err := s.resourceUtil.GetDatabaseForInstallation(installation).Provision(s.store, logger)
//	if err != nil {
//		return
//	}
//}
//
//func (s *InstallationSupervisor) migrateMultitenantPostgresDB(installation *model.Installation, migration model.InstallationMigration, logger log.FieldLogger) error {
//
//	err := s.store.SwitchInstallationDatabase(installation.ID, migration.DestinationMultiTenant.DatabaseID)
//	if err != nil {
//		return errors.Wrap(err, "failed to switch multitenant database for installation")
//	}
//}
//
//func (s *InstallationSupervisor) regenerateSecrets(installation *model.Installation, migration model.InstallationMigration, logger log.FieldLogger) error {
//
//	// For each cluster installation run the update?
//
//}
//
//func (s *InstallationSupervisor) restoreDatabase(installation *model.Installation, migration model.InstallationMigration, logger log.FieldLogger) error {
//
//	// basically run the restore
//
//}
//
//func (s *InstallationSupervisor) restoreDatabaseInProgress(installation *model.Installation, migration model.InstallationMigration, logger log.FieldLogger) error {
//
//	// wait for restore to finish
//
//}
//
//func (s *InstallationSupervisor) migrationCompleted(installation *model.Installation, migration model.InstallationMigration, logger log.FieldLogger) error {
//
//	// transition back to hibernated state
//	// Optionally cleanup the old database
//
//}
//
//
