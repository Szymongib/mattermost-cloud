package model

import (
	"encoding/json"
	"github.com/pkg/errors"
	"io"
)

type InstallationDBRestorationOperation struct {
	ID string
	InstallationID string
	BackupID string
	RequestAt int64
	State InstallationDBRestorationState
	// TargetInstallationState is an installation State to which installation
	// will be transitioned when the restoration finishes successfully.
	TargetInstallationState string
	ClusterInstallationID string
	CompleteAt int64
	DeleteAt int64
	LockAcquiredBy             *string
	LockAcquiredAt             int64
}

// InstallationDBRestorationState represents the state of backup.
type InstallationDBRestorationState string

const (
	// InstallationDBRestorationStateRequested is a requested installation db restoration that was not yet started.
	InstallationDBRestorationStateRequested InstallationDBRestorationState = "installation-db-restoration-requested"
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
	if installation.State != InstallationStateHibernating && installation.State != InstallationStateDBMigrationInProgress {
		return errors.Errorf("invalid installation state, only hibernated installations can be restored, state is %q", installation.State)
	}

	return EnsureBackupRestoreCompatible(installation)
}

func DetermineAfterRestorationState(installation *Installation) (string, error) {
	switch installation.State {
	case InstallationStateHibernating:
		return InstallationStateHibernating, nil
	case InstallationStateDBMigrationInProgress:
		return InstallationStateDBMigrationInProgress, nil
	}
	return "", errors.Errorf("restoration is not supported for installation in state %s", installation.State)
}

// NewInstallationDBRestorationOperationFromReader will create a InstallationDBRestorationOperation from an
// io.Reader with JSON data.
func NewInstallationDBRestorationOperationFromReader(reader io.Reader) (*InstallationDBRestorationOperation, error) {
	var installationDBRestorationOperation InstallationDBRestorationOperation
	err := json.NewDecoder(reader).Decode(&installationDBRestorationOperation)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode InstallationDBRestorationOperation")
	}

	return &installationDBRestorationOperation, nil
}

// NewInstallationDBRestorationOperationsFromReader will create a slice of InstallationDBRestorationOperations from an
// io.Reader with JSON data.
func NewInstallationDBRestorationOperationsFromReader(reader io.Reader) ([]*InstallationDBRestorationOperation, error) {
	installationDBRestorationOperations := []*InstallationDBRestorationOperation{}
	err := json.NewDecoder(reader).Decode(&installationDBRestorationOperations)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode InstallationDBRestorationOperations")
	}

	return installationDBRestorationOperations, nil
}
