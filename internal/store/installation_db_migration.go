// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"database/sql"
	"encoding/json"
	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

const (
	installationDBMigrationTable = "DBMigrationOperation"
)

// TODO: implement it and test

var installationDBMigrationSelect sq.SelectBuilder

func init() {
	installationDBMigrationSelect = sq.
		Select("ID",
			"InstallationID",
			"RequestAt",
			"State",
			"SourceDatabase",
			"DestinationDatabase",
			"SourceMultiTenantRaw",
			"DestinationMultiTenantRaw",
			"BackupID",
			"InstallationDBRestorationOperationID",
			"CompleteAt",
			"DeleteAt",
			"LockAcquiredBy",
			"LockAcquiredAt",
		).
		From(installationDBMigrationTable)
}

type rawDBMigrationOperation struct {
	*model.DBMigrationOperation
	SourceMultiTenantRaw []byte
	DestinationMultiTenantRaw []byte
}

type rawDBMigrationOperations []*rawDBMigrationOperation

func (r *rawDBMigrationOperation) toDBMigrationOperation() (*model.DBMigrationOperation, error) {
	// We only need to set values that are converted from a raw database format.
	var err error
	if len(r.SourceMultiTenantRaw) > 0 {
		data := model.MultiTenantDBMigrationData{}
		err = json.Unmarshal(r.SourceMultiTenantRaw, &data)
		if err != nil {
			return nil, err
		}
		r.DBMigrationOperation.SourceMultiTenant = &data
	}
	if len(r.DestinationMultiTenantRaw) > 0 {
		data := model.MultiTenantDBMigrationData{}
		err = json.Unmarshal(r.DestinationMultiTenantRaw, &data)
		if err != nil {
			return nil, err
		}
		r.DBMigrationOperation.DestinationMultiTenant = &data
	}

	return r.DBMigrationOperation, nil
}

func (r *rawDBMigrationOperations) toDBMigrationOperations() ([]*model.DBMigrationOperation, error) {
	if r == nil {
		return []*model.DBMigrationOperation{}, nil
	}
	migrationOperations := make([]*model.DBMigrationOperation, 0, len(*r))

	for _, raw := range *r {
		operation, err := raw.toDBMigrationOperation()
		if err != nil {
			return nil, errors.Wrap(err, "failed to create migration operation from raw")
		}
		migrationOperations = append(migrationOperations, operation)
	}
	return migrationOperations, nil
}


// CreateInstallationDBMigration records installation db migration to the database, assigning it a unique ID.
func (sqlStore *SQLStore) CreateInstallationDBMigration(dbMigration *model.DBMigrationOperation) error {
	dbMigration.ID = model.NewID()
	dbMigration.RequestAt = GetMillis()

	multiTenantSourceRaw, err := json.Marshal(dbMigration.SourceMultiTenant)
	if err != nil {
		return errors.Wrap(err, "failed to marshal source multi tenant db")
	}
	multiTenantDestinationRaw, err := json.Marshal(dbMigration.DestinationMultiTenant)
	if err != nil {
		return errors.Wrap(err, "failed to marshal destination multi tenant db")
	}

	_, err = sqlStore.execBuilder(sqlStore.db, sq.
		Insert(installationDBMigrationTable).
		SetMap(map[string]interface{}{
			"ID":                                   dbMigration.ID,
			"InstallationID":                       dbMigration.InstallationID,
			"RequestAt":                            dbMigration.RequestAt,
			"State":                                dbMigration.State,
			"SourceDatabase":                       dbMigration.SourceDatabase,
			"DestinationDatabase":                  dbMigration.DestinationDatabase,
			"SourceMultiTenantRaw":                 multiTenantSourceRaw,
			"DestinationMultiTenantRaw":            multiTenantDestinationRaw,
			"BackupID":                             dbMigration.BackupID,
			"InstallationDBRestorationOperationID": dbMigration.InstallationDBRestorationOperationID,
			"CompleteAt":                           dbMigration.CompleteAt,
			"DeleteAt":                             0,
			"LockAcquiredBy":                       dbMigration.LockAcquiredBy,
			"LockAcquiredAt":                       dbMigration.LockAcquiredAt,
		}),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create installation db migration operation")
	}

	return nil
}

// GetInstallationDBMigration fetches the given installation db migration.
func (sqlStore *SQLStore) GetInstallationDBMigration(id string) (*model.DBMigrationOperation, error) {
	builder := installationDBMigrationSelect.
		Where("ID = ?", id)

	var migrationOpRaw rawDBMigrationOperation
	err := sqlStore.getBuilder(sqlStore.db, &migrationOpRaw, builder)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to query for installation db migration")
	}

	migrationOp, err := migrationOpRaw.toDBMigrationOperation()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create migration operation from raw")
	}

	return migrationOp, nil
}

// GetInstallationDBMigrations fetches the given page of created installation db migration. The first page is 0.
func (sqlStore *SQLStore) GetInstallationDBMigrations(filter *model.InstallationDBMigrationFilter) ([]*model.DBMigrationOperation, error) {
	builder := installationDBMigrationSelect.
		OrderBy("RequestAt DESC")
	builder = sqlStore.applyInstallationDBMigrationFilter(builder, filter)

	return sqlStore.getDBMigrationOperations(builder)
}

// GetUnlockedInstallationDBMigrationsPendingWork returns unlocked installation db migrations in a pending state.
func (sqlStore *SQLStore) GetUnlockedInstallationDBMigrationsPendingWork() ([]*model.DBMigrationOperation, error) {
	builder := installationDBMigrationSelect.
		Where(sq.Eq{
			"State": model.AllInstallationDBMigrationOperationsStatesPendingWork,
		}).
		Where("LockAcquiredAt = 0").
		OrderBy("RequestAt ASC")

	return sqlStore.getDBMigrationOperations(builder)
}

func (sqlStore *SQLStore) getDBMigrationOperations(builder builder) ([]*model.DBMigrationOperation, error) {
	var rawMigrationOps rawDBMigrationOperations
	err := sqlStore.selectBuilder(sqlStore.db, &rawMigrationOps, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for installation db migrations")
	}

	migrationOps, err := rawMigrationOps.toDBMigrationOperations()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create migration operations from raw")
	}

	return migrationOps, nil
}


// UpdateInstallationDBMigrationState updates the given installation db migration state.
func (sqlStore *SQLStore) UpdateInstallationDBMigrationState(dbMigration *model.DBMigrationOperation) error {
	return sqlStore.updateInstallationDBMigrationFields(
		sqlStore.db,
		dbMigration.ID, map[string]interface{}{
			"State": dbMigration.State,
		})
}

// UpdateInstallationDBMigration updates the given installation db migration.
func (sqlStore *SQLStore) UpdateInstallationDBMigration(dbMigration *model.DBMigrationOperation) error {
	return sqlStore.updateInstallationDBMigration(sqlStore.db, dbMigration)
}

func (sqlStore *SQLStore) updateInstallationDBMigration(db execer, dbMigration *model.DBMigrationOperation) error {
	return sqlStore.updateInstallationDBMigrationFields(
		db,
		dbMigration.ID, map[string]interface{}{
			"State":                                dbMigration.State,
			"BackupID":                             dbMigration.BackupID,
			"InstallationDBRestorationOperationID": dbMigration.InstallationDBRestorationOperationID,
			"CompleteAt":                           dbMigration.CompleteAt,
		})
}

func (sqlStore *SQLStore) updateInstallationDBMigrationFields(db execer, id string, fields map[string]interface{}) error {
	_, err := sqlStore.execBuilder(db, sq.
		Update(installationDBMigrationTable).
		SetMap(fields).
		Where("ID = ?", id))
	if err != nil {
		return errors.Wrapf(err, "failed to update installation db migration fields: %s", getMapKeys(fields))
	}

	return nil
}

//// TODO: tests
//// UpdateInstallationMigrationResources updates installation, installation backup and installation db migration in a single transaction.
//func (sqlStore *SQLStore) UpdateInstallationMigrationResources(installation *model.Installation, backup *model.InstallationBackup, dbMigration *model.DBMigrationOperation) error {
//	tx, err := sqlStore.beginTransaction(sqlStore.db)
//	if err != nil {
//		return errors.Wrap(err, "failed to start transaction")
//	}
//	defer tx.RollbackUnlessCommitted()
//
//	err = sqlStore.updateInstallationDBMigration(tx, dbMigration)
//	if err != nil {
//		return errors.Wrap(err, "failed to update installation db migration")
//	}
//
//	err = sqlStore.updateInstallation(tx, installation)
//	if err != nil {
//		return errors.Wrap(err, "failed to update installation")
//	}
//
//	err = sqlStore.updateInstallationBackupState(tx, backup)
//	if err != nil {
//		return errors.Wrap(err, "failed to update installation backup")
//	}
//
//	err = tx.Commit()
//	if err != nil {
//		return errors.Wrap(err, "failed to commit transaction")
//	}
//	return nil
//}

// LockDBMigrationOperation marks the DBMigrationOperation as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockDBMigrationOperation(id, lockerID string) (bool, error) {
	return sqlStore.lockRows(installationDBMigrationTable, []string{id}, lockerID)
}

// LockDBMigrationOperations marks DBMigrationOperations as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockDBMigrationOperations(ids []string, lockerID string) (bool, error) {
	return sqlStore.lockRows(installationDBMigrationTable, ids, lockerID)
}

// UnlockDBMigrationOperation releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockDBMigrationOperation(id, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows(installationDBMigrationTable, []string{id}, lockerID, force)
}

// UnlockDBMigrationOperations releases a locks previously acquired against a caller.
func (sqlStore *SQLStore) UnlockDBMigrationOperations(ids []string, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows(installationDBMigrationTable, ids, lockerID, force)
}

func (sqlStore *SQLStore) applyInstallationDBMigrationFilter(builder sq.SelectBuilder, filter *model.InstallationDBMigrationFilter) sq.SelectBuilder {
	builder = applyPagingFilter(builder, filter.Paging)

	if len(filter.IDs) > 0 {
		builder = builder.Where(sq.Eq{"ID": filter.IDs})
	}
	if filter.InstallationID != "" {
		builder = builder.Where("InstallationID = ?", filter.InstallationID)
	}
	if filter.ClusterInstallationID != "" {
		builder = builder.Where("ClusterInstallationID = ?", filter.ClusterInstallationID)
	}
	if len(filter.States) > 0 {
		builder = builder.Where(sq.Eq{
			"State": filter.States,
		})
	}

	return builder
}
