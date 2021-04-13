// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"database/sql"
	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

const (
	installationDBMigrationTable = "DBMigrationOperation"
)

// TODO: implement it

var installationDBMigrationSelect sq.SelectBuilder

func init() {
	installationDBMigrationSelect = sq.
		Select("ID",




			"DeleteAt",
			"LockAcquiredBy",
			"LockAcquiredAt",
		).
		From(installationDBMigrationTable)
}


// CreateInstallationDBMigration records installation db migration to the database, assigning it a unique ID.
func (sqlStore *SQLStore) CreateInstallationDBMigration(dbMigration *model.DBMigrationOperation) error {
	dbMigration.ID = model.NewID()
	dbMigration.RequestAt = GetMillis()

	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Insert(installationDBMigrationTable).
		SetMap(map[string]interface{}{
			"ID": dbMigration.ID,
			"InstallationID": dbMigration.InstallationID,




			"DeleteAt": 0,
			"LockAcquiredBy": dbMigration.LockAcquiredBy,
			"LockAcquiredAt": dbMigration.LockAcquiredAt,
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

	var migrationOp model.DBMigrationOperation
	err := sqlStore.getBuilder(sqlStore.db, &migrationOp, builder)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to query for installation db migration")
	}

	return &migrationOp, nil
}

// GetInstallationDBMigrations fetches the given page of created installation db migration. The first page is 0.
func (sqlStore *SQLStore) GetInstallationDBMigrations(filter *model.InstallationDBMigrationFilter) ([]*model.DBMigrationOperation, error) {
	builder := installationDBMigrationSelect.
		OrderBy("RequestAt DESC")
	builder = sqlStore.applyInstallationDBMigrationFilter(builder, filter)

	var migrationOps []*model.DBMigrationOperation
	err := sqlStore.selectBuilder(sqlStore.db, &migrationOps, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for installation db migrations")
	}

	return migrationOps, nil
}

// GetUnlockedInstallationDBMigrationsPendingWork returns unlocked installation db migrations in a pending state.
func (sqlStore *SQLStore) GetUnlockedInstallationDBMigrationsPendingWork() ([]*model.DBMigrationOperation, error) {
	builder := installationDBMigrationSelect.
		Where(sq.Eq{
			"State": model.AllInstallationDBMigrationOperationsStatesPendingWork,
		}).
		Where("LockAcquiredAt = 0").
		OrderBy("RequestAt ASC")

	var migrationOps []*model.DBMigrationOperation
	err := sqlStore.selectBuilder(sqlStore.db, &migrationOps, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for installation db migrations")
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
			"State": dbMigration.State,
			"TargetInstallationState": dbMigration.TargetInstallationState,
			"ClusterInstallationID": dbMigration.ClusterInstallationID,
			"CompleteAt": dbMigration.CompleteAt,
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

// TODO: tests
// UpdateInstallationMigrationResources updates installation, installation backup and installation db migration in a single transaction.
func (sqlStore *SQLStore) UpdateInstallationMigrationResources(installation *model.Installation, backup *model.InstallationBackup, dbMigration *model.DBMigrationOperation) error {
	tx, err := sqlStore.beginTransaction(sqlStore.db)
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer tx.RollbackUnlessCommitted()

	err = sqlStore.updateInstallationDBMigration(tx, dbMigration)
	if err != nil {
		return errors.Wrap(err, "failed to update installation db migration")
	}

	err = sqlStore.updateInstallation(tx, installation)
	if err != nil {
		return errors.Wrap(err, "failed to update installation")
	}

	err = sqlStore.updateInstallationBackupState(tx, backup)
	if err != nil {
		return errors.Wrap(err, "failed to update installation backup")
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}
	return nil
}

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
