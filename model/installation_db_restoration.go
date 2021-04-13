package model

import (
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"strings"
)

// TODO: decide on naming - should we stick to operation?
type InstallationDBRestorationOperation struct {
	ID string
	InstallationID string
	BackupID string
	RequestAt int64
	State InstallationDBRestorationState

	TargetInstallationState string // Decided based on Installation State when the restoration starts
	ClusterInstallationID string

	CompleteAt int64

	DeleteAt int64
	LockAcquiredBy             *string
	LockAcquiredAt             int64
	//Lock
}

// InstallationDBRestorationState represents the state of backup.
type InstallationDBRestorationState string

const (
	// InstallationDBRestorationStateRequested is a requested installation db restoration that was not yet started.
	InstallationDBRestorationStateRequested InstallationDBRestorationState = "installation-db-restoration-requested"
	// InstallationDBRestorationStateBeginning is an installation db restoration that is ready to be started.
	InstallationDBRestorationStateBeginning InstallationDBRestorationState = "installation-db-restoration-beginning"
	// InstallationDBRestorationStateInProgress is an installation db restoration that is currently running.
	InstallationDBRestorationStateInProgress InstallationDBRestorationState = "installation-db-restoration-in-progress"
	// InstallationDBRestorationStateFinalizing is an installation db restoration that is finalizing restoration.
	InstallationDBRestorationStateFinalizing InstallationDBRestorationState = "installation-db-restoration-finishing"
	// InstallationDBRestorationStateSucceeded is an installation db restoration that have finished with success.
	InstallationDBRestorationStateSucceeded InstallationDBRestorationState = "installation-db-restoration-succeeded"
	// InstallationDBRestorationStateFailed is an installation db restoration that have failed.
	InstallationDBRestorationStateFailed InstallationDBRestorationState = "installation-db-restoration-failed"
	// InstallationDBRestorationStateInvalid is an installation db restoration that is invalid.
	InstallationDBRestorationStateInvalid InstallationDBRestorationState = "installation-db-restoration-invalid"
)

// AllInstallationBackupStatesPendingWork is a list of all backup states that
// the supervisor will attempt to transition towards stable on the next "tick".
var AllInstallationDBRestorationStatesPendingWork = []InstallationDBRestorationState{
	InstallationDBRestorationStateRequested,
	InstallationDBRestorationStateBeginning,
	InstallationDBRestorationStateInProgress,
	InstallationDBRestorationStateFinalizing,
}

// TODO: add include finished or something
// InstallationDBRestorationFilter describes the parameters used to constrain a set of installation-db-restoration.
type InstallationDBRestorationFilter struct {
	Paging
	IDs                   []string
	InstallationID        string
	ClusterInstallationID string
	States                []InstallationDBRestorationState
}

func EnsureReadyForDBRestoration(installation *Installation, backup *InstallationBackup) error {
	if installation.ID != backup.InstallationID {
		return errors.New("Backup belongs to different installation")
	}
	if backup.State != InstallationBackupStateBackupSucceeded {
		return errors.Errorf("Only backups in succeeded state can be restored, the state is %q", backup.State)
	}
	if backup.DeleteAt > 0 {
		return errors.New("Backup files are deleted")
	}

	return EnsureInstallationReadyForDBRestoration(installation)
}

func EnsureInstallationReadyForDBRestoration(installation *Installation) error {
	var errs []string

	if installation.State != InstallationStateHibernating && installation.State != InstallationStateDBMigrationInProgress {
		errs = append(errs, fmt.Sprintf("invalid installation state, only hibernated installations can be restored, state is %q", installation.State))
	}

	if installation.Database != InstallationDatabaseMultiTenantRDSPostgres &&
		installation.Database != InstallationDatabaseSingleTenantRDSPostgres {
		errs = append(errs, fmt.Sprintf("invalid installation database, db restoration is supported only for Postgres database, the database type is %q", installation.Database))
	}

	if installation.Filestore == InstallationFilestoreMinioOperator {
		errs = append(errs, "invalid installation file store, cannot restore database for installation using local Minio file store")
	}

	if len(errs) > 0 {
		return errors.Errorf("some settings are incompatible with db restpration: %s", strings.Join(errs, "; "))
	}

	return nil
}

func DetermineRestorationTargetState(installation *Installation) (string, error) {
	switch installation.State {
	case InstallationStateHibernating:
		return InstallationStateHibernating, nil
	}
	return "", errors.Errorf("restoration is not supported for installation in state %s", installation.State)
}


// TODO: test
// NewInstallationDBRestorationOperationsFromReader will create a []*InstallationDBRestorationOperation from an
// io.Reader with JSON data.
func NewInstallationDBRestorationOperationsFromReader(reader io.Reader) ([]*InstallationDBRestorationOperation, error) {
	var restorations []*InstallationDBRestorationOperation
	err := json.NewDecoder(reader).Decode(&restorations)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode installation db restore operation")
	}

	return restorations, nil
}
