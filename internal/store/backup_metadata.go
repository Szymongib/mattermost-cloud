package store

import (
	"database/sql"
	"encoding/json"
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
		Where("InstallationID = ?", installationID).
		Where(sq.Eq{"State": model.AllBackupMetadataStatesPendingWork}).
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
			"ID":               backupMeta.ID,
			"InstallationID":   backupMeta.InstallationID,
			"DataResidenceRaw": nil,
			"State":            backupMeta.State,
			"RequestAt":        backupMeta.RequestAt,
			"StartAt":          0,
			"DeleteAt":         0,
			"LockAcquiredBy":   nil,
			"LockAcquiredAt":   0,
		}),
	)
	if err != nil {
		return errors.Wrap(err, "failed to create backup metadata")
	}

	return nil
}

type rawBackupMetadata struct {
	*model.BackupMetadata
	DataResidenceRaw []byte
}

type rawBackupsMetadata []*rawBackupMetadata

func (r *rawBackupMetadata) toBackupMetadata() (*model.BackupMetadata, error) {
	// We only need to set values that are converted from a raw database format.
	var err error
	dataResidence := model.S3DataResidence{}
	if len(r.DataResidenceRaw) > 0 {
		err = json.Unmarshal(r.DataResidenceRaw, &dataResidence)
		if err != nil {
			return nil, err
		}
	}
	r.BackupMetadata.DataResidence = &dataResidence

	return r.BackupMetadata, nil
}

func (r *rawBackupsMetadata) toBackupsMetadata() ([]*model.BackupMetadata, error) {
	if r == nil {
		return []*model.BackupMetadata{}, nil
	}
	backupsMeta := make([]*model.BackupMetadata, 0, len(*r))

	for _, raw := range *r {
		metadata, err := raw.toBackupMetadata()
		if err != nil {
			return nil, errors.Wrap(err, "failed to create backup metadata from raw")
		}
		backupsMeta = append(backupsMeta, metadata)
	}
	return backupsMeta, nil
}

// GetBackupMetadata fetches the given backup metadata by id.
func (sqlStore *SQLStore) GetBackupMetadata(id string) (*model.BackupMetadata, error) {
	var rawMetadata rawBackupMetadata
	err := sqlStore.getBuilder(sqlStore.db, &rawMetadata,
		backupMetadataSelect.Where("ID = ?", id),
	)
	if err == sql.ErrNoRows {
		return nil, nil
	} else if err != nil {
		return nil, errors.Wrap(err, "failed to get backup metadata by id")
	}

	backupMetadata, err := rawMetadata.toBackupMetadata()
	if err != nil {
		return backupMetadata, err
	}

	return backupMetadata, nil
}

// TODO: get, update etc

// TODO: test
// GetUnlockedInstallationsPendingWork returns an unlocked installation in a pending state.
func (sqlStore *SQLStore) GetUnlockedBackupMetadataPendingWork() ([]*model.BackupMetadata, error) {
	builder := backupMetadataSelect.
		Where(sq.Eq{
			"State": model.AllBackupMetadataStatesPendingWork,
		}).
		Where("LockAcquiredAt = 0").
		OrderBy("RequestAt ASC")

	var rawBackupsMeta rawBackupsMetadata
	err := sqlStore.selectBuilder(sqlStore.db, &rawBackupsMeta, builder)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get backup metadata pending work")
	}

	backupsMeta, err := rawBackupsMeta.toBackupsMetadata()
	if err != nil {
		return nil, err
	}

	return backupsMeta, nil
}

// UpdateBackupDataResidency updates the given backup metadata data residency.
func (sqlStore *SQLStore) UpdateBackupDataResidency(backupMeta *model.BackupMetadata) error {
	data, err := json.Marshal(backupMeta.DataResidence)
	if err != nil {
		return errors.Wrap(err, "failed to marshal data residency")
	}

	return sqlStore.updateBackupMetadataFields(
		backupMeta.ID, map[string]interface{}{
			"DataResidenceRaw": data,
		})
}


// UpdateBackupMetadataState updates the given backup metadata to a new state.
func (sqlStore *SQLStore) UpdateBackupMetadataState(backupMeta *model.BackupMetadata) error {
	return sqlStore.updateBackupMetadataFields(
		backupMeta.ID, map[string]interface{}{
		"State": backupMeta.State,
	})
}

func (sqlStore *SQLStore) updateBackupMetadataFields(id string, fields map[string]interface{}) error {
	_, err := sqlStore.execBuilder(sqlStore.db, sq.
		Update(backupMetadataTable).
		SetMap(fields).
		Where("ID = ?", id),
	)
	if err != nil {
		return errors.Wrapf(err, "failed to update backup metadata fields: %s", getMapKeys(fields))
	}

	return nil
}

// LockBackupMetadata marks the backup metadata as locked for exclusive use by the caller.
func (sqlStore *SQLStore) LockBackupMetadata(backupMetadataID, lockerID string) (bool, error) {
	return sqlStore.lockRows(backupMetadataTable, []string{backupMetadataID}, lockerID)
}

// UnlockBackupMetadata releases a lock previously acquired against a caller.
func (sqlStore *SQLStore) UnlockBackupMetadata(backupMetadataID, lockerID string, force bool) (bool, error) {
	return sqlStore.unlockRows(backupMetadataTable, []string{backupMetadataID}, lockerID, force)
}

func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
