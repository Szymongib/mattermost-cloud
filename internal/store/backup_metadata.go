package store

import (
	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

const (
	backupMetadataTable = "BackupMetadata"
)

var backupMetadataSelect sq.SelectBuilder

func init() {
	backupMetadataSelect = sq.
		Select(
			"ID", "InstallationID", "DataResidenceRaw", "State", "RequestAt", "StartAt", "DeleteAt", "LockAcquiredBy", "LockAcquiredAt",
		).
		From(backupMetadataTable)
}

func (sqlStore *SQLStore) IsBackupRunning(installationID string) (bool, error) {
	var totalResult countResult
	builder := sq.
		Select("Count (*)").
		From(backupMetadataTable).
		Where("InstallationId = ?", installationID).
		Where(sq.Or{
			sq.Expr("State = ?", model.BackupStateBackupRequested),
			sq.Expr("State = ?", model.BackupStateBackupInProgress),
		}).
		Where("DeleteAt = 0")
	err := sqlStore.selectBuilder(sqlStore.db, &totalResult, builder)
	if err != nil {
		return false, errors.Wrap(err, "failed to count ongoing backups")
	}

	ongoingBackups, err := totalResult.value()
	if err != nil {
		return false, errors.Wrap(err, "failed to value of ongoing backups")
	}

	return ongoingBackups > 0, nil
}


// CreateBackupMetadata record backup metadata to the database, assigning it a unique ID.
func (sqlStore *SQLStore) CreateBackupMetadata(backupMeta *model.BackupMetadata) error {

	backupMeta.ID = model.NewID()
	backupMeta.RequestAt = GetMillis()

	// TODO: data residence - empty? Or leave it null?

	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Insert(backupMetadataTable).
		SetMap(map[string]interface{}{
			"ID":                     backupMeta.ID,
			"InstallationID":                     backupMeta.InstallationId,
			"DataResidenceRaw":                     nil,
			"State":                  backupMeta.State,
			"RequestAt":               backupMeta.RequestAt,
			"StartAt":               0,
			"DeleteAt":               0,
			"LockAcquiredBy":         nil,
			"LockAcquiredAt":         0,
		}),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create backup metadata")
	}

	return nil
}

// TODO: get, update etc
