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
	installationDBRestorationTable = "InstallationDBRestorationOperation"
)

var installationDBRestorationSelect sq.SelectBuilder

func init() {
	installationDBRestorationSelect = sq.
		Select("ID",
			"InstallationID",
			"BackupID",
			"RequestAt",
			"State",
			"TargetInstallationState",
			"ClusterInstallationID",
			"CompleteAt",
			"DeleteAt",
			"LockAcquiredBy",
			"LockAcquiredAt",
		).
		From(installationDBRestorationTable)
}


// CreateInstallationDBRestoration records installation db restoration to the database, assigning it a unique ID.
func (sqlStore *SQLStore) CreateInstallationDBRestoration(dbRestoration *model.InstallationDBRestorationOperation) error {
	dbRestoration.ID = model.NewID()
	dbRestoration.RequestAt = GetMillis()

	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Insert(installationDBRestorationTable).
		SetMap(map[string]interface{}{
			"ID": dbRestoration.ID,
			"InstallationID": dbRestoration.InstallationID,
			"BackupID": dbRestoration.BackupID,
			"State": dbRestoration.State,
			"RequestAt": dbRestoration.RequestAt,
			"TargetInstallationState": dbRestoration.TargetInstallationState,
			"ClusterInstallationID": dbRestoration.ClusterInstallationID,
			"CompleteAt": dbRestoration.CompleteAt,
			"DeleteAt": 0,
			"LockAcquiredBy": dbRestoration.LockAcquiredBy,
			"LockAcquiredAt": dbRestoration.LockAcquiredAt,
		}),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create installation db restoration operation")
	}

	return nil
}

// GetInstallationDBRestoration fetches the given installation db restoration.
func (sqlStore *SQLStore) GetInstallationDBRestoration(id string) (*model.InstallationDBRestorationOperation, error) {
	builder := installationDBRestorationSelect.
		Where("ID = ?", id)

	var restorationOp model.InstallationDBRestorationOperation
	err := sqlStore.getBuilder(sqlStore.db, &restorationOp, builder)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to query for installation db restoration")
	}

	return &restorationOp, nil
}

// GetInstallationDBRestorations fetches the given page of created installation db restoration. The first page is 0.
func (sqlStore *SQLStore) GetInstallationDBRestorations(filter *model.InstallationDBRestorationFilter) ([]*model.InstallationDBRestorationOperation, error) {
	builder := installationDBRestorationSelect.
		OrderBy("RequestAt DESC")
	builder = sqlStore.applyInstallationDBRestorationFilter(builder, filter)

	var restorationOps []*model.InstallationDBRestorationOperation
	err := sqlStore.selectBuilder(sqlStore.db, &restorationOps, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for installation db restorations")
	}

	return restorationOps, nil
}

// GetUnlockedInstallationDBRestorationsPendingWork returns unlocked installation db restorations in a pending state.
func (sqlStore *SQLStore) GetUnlockedInstallationDBRestorationsPendingWork() ([]*model.InstallationDBRestorationOperation, error) {
	builder := installationDBRestorationSelect.
		Where(sq.Eq{
			"State": model.AllInstallationDBRestorationStatesPendingWork,
		}).
		Where("LockAcquiredAt = 0").
		OrderBy("RequestAt ASC")

	var restorationOps []*model.InstallationDBRestorationOperation
	err := sqlStore.selectBuilder(sqlStore.db, &restorationOps, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to query for installation db restorations")
	}

	return restorationOps, nil
}


// UpdateInstallationDBRestorationState updates the given installation db restoration state.
func (sqlStore *SQLStore) UpdateInstallationDBRestorationState(dbRestoration *model.InstallationDBRestorationOperation) error {
	return sqlStore.updateInstallationDBRestorationFields(
		sqlStore.db,
		dbRestoration.ID, map[string]interface{}{
			"State": dbRestoration.State,
		})
}

// UpdateInstallationDBRestoration updates the given installation db restoration.
func (sqlStore *SQLStore) UpdateInstallationDBRestoration(dbRestoration *model.InstallationDBRestorationOperation) error {
	return sqlStore.updateInstallationDBRestoration(sqlStore.db, dbRestoration)
}

func (sqlStore *SQLStore) updateInstallationDBRestoration(db execer, dbRestoration *model.InstallationDBRestorationOperation) error {
	return sqlStore.updateInstallationDBRestorationFields(
		db,
		dbRestoration.ID, map[string]interface{}{
			"State": dbRestoration.State,
			"TargetInstallationState": dbRestoration.TargetInstallationState,
			"ClusterInstallationID": dbRestoration.ClusterInstallationID,
			"CompleteAt": dbRestoration.CompleteAt,
		})
}

func (sqlStore *SQLStore) updateInstallationDBRestorationFields(db execer, id string, fields map[string]interface{}) error {
	_, err := sqlStore.execBuilder(db, sq.
		Update(installationDBRestorationTable).
		SetMap(fields).
		Where("ID = ?", id))
	if err != nil {
		return errors.Wrapf(err, "failed to update installation db restoration fields: %s", getMapKeys(fields))
	}

	return nil
}

// TODO: tests
// UpdateInstallationRestorationResources updates installation, installation backup and installation db restoration in a single transaction.
func (sqlStore *SQLStore) UpdateInstallationRestorationResources(installation *model.Installation, backup *model.InstallationBackup, dbRestoration *model.InstallationDBRestorationOperation) error {
	tx, err := sqlStore.beginTransaction(sqlStore.db)
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer tx.RollbackUnlessCommitted()

	err = sqlStore.updateInstallationDBRestoration(tx, dbRestoration)
	if err != nil {
		return errors.Wrap(err, "failed to update installation db restoration")
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

// LockInstallationDBRestoration marks the InstallationDBRestoration as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockInstallationDBRestoration(id, lockerID string) (bool, error) {
	return sqlStore.lockRows(installationDBRestorationTable, []string{id}, lockerID)
}

// LockInstallationDBRestorations marks InstallationDBRestorations as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockInstallationDBRestorations(ids []string, lockerID string) (bool, error) {
	return sqlStore.lockRows(installationDBRestorationTable, ids, lockerID)
}

// UnlockInstallationDBRestoration releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockInstallationDBRestoration(id, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows(installationDBRestorationTable, []string{id}, lockerID, force)
}

// UnlockInstallationDBRestorations releases a locks previously acquired against a caller.
func (sqlStore *SQLStore) UnlockInstallationDBRestorations(ids []string, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows(installationDBRestorationTable, ids, lockerID, force)
}

func (sqlStore *SQLStore) applyInstallationDBRestorationFilter(builder sq.SelectBuilder, filter *model.InstallationDBRestorationFilter) sq.SelectBuilder {
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
