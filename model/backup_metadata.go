package model

import (
	"encoding/json"
	"github.com/pkg/errors"
	"io"
)

type BackupMetadata struct {
	ID             string
	InstallationId string
	DataResidence  *S3DataResidence
	State          BackupState
	RequestAt      int64
	StartAt        int64 // TODO: Job creation timestamp?
	DeleteAt       int64
	LockAcquiredBy *string
	LockAcquiredAt int64

}

type S3DataResidence struct {
	Region string
	Bucket string
}

type BackupState string

const (
	BackupStateBackupRequested BackupState = "backup-requested"
	BackupStateBackupInProgress BackupState = "backup-in-progress"
	BackupStateBackupSucceeded BackupState = "backup-succeeded"
	BackupStateBackupFailed BackupState = "backup-failed"
)

// NewBackupMetadataFromReader will create a BackupMetadata from an
// io.Reader with JSON data.
func NewBackupMetadataFromReader(reader io.Reader) (*BackupMetadata, error) {
	var backupMetadata BackupMetadata
	err := json.NewDecoder(reader).Decode(&backupMetadata)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode backup metadata")
	}

	return &backupMetadata, nil
}
