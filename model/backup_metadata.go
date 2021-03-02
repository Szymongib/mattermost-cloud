package model

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"strings"
)

type BackupMetadata struct {
	ID             string
	InstallationID string
	// ClusterInstallationID is set when backup is scheduled.
	ClusterInstallationID string
	DataResidence  *S3DataResidence // TODO: DataResidence or Residency?
	State          BackupState
	RequestAt      int64
	// StartAt is a start time of job that successfully completed backup.
	StartAt        int64
	DeleteAt       int64
	LockAcquiredBy *string
	LockAcquiredAt int64
}

type S3DataResidence struct {
	Region    string
	URL       string
	Bucket    string
	ObjectKey string
}

type BackupState string

const (
	BackupStateBackupRequested BackupState = "backup-requested"
	BackupStateBackupInProgress BackupState = "backup-in-progress"
	BackupStateBackupSucceeded BackupState = "backup-succeeded"
	BackupStateBackupFailed BackupState = "backup-failed"
)

// AllBackupMetadataStatesPendingWork is a list of all backup metadata states that
// the supervisor will attempt to transition towards stable on the next "tick".
var AllBackupMetadataStatesPendingWork = []BackupState{
	BackupStateBackupRequested,
	BackupStateBackupInProgress,
}


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

func EnsureBackupCompatible(installation *Installation) error {
	var errs []string

	if installation.State != InstallationStateHibernating {
		errs = append(errs, "only hibernated installations can be backed up")
	}

	if installation.Database != InstallationDatabaseMultiTenantRDSPostgres &&
		installation.Database != InstallationDatabaseSingleTenantRDSPostgres {
		errs = append(errs, fmt.Sprintf("database backup supported only for Postgres database, the database type is %q", installation.Database))
	}

	if installation.Filestore == InstallationFilestoreMinioOperator {
		errs = append(errs, "cannot backup database for installation using local Minio file store")
	}

	if len(errs) > 0 {
		return errors.Errorf("some settings are incompatible with backup: [%s]", strings.Join(errs, "; "))
	}

	return nil
}
