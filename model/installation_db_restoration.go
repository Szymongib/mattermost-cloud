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
	ID string // TODO: remove if you leave it inline
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
	Paging
}

// InstallationDBRestorationState represents the state of backup.
type InstallationDBRestorationState string

const (
	// InstallationDBRestorationStateStateRequested is a requested installation db restoration that was not yet started.
	InstallationDBRestorationStateStateRequested InstallationDBRestorationState = "installation-db-restoration-requested"

	// InstallationDBRestorationStateStateTriggeringRestoration is an installation db restoration that is currently being started.
	InstallationDBRestorationStateStateTriggeringRestoration InstallationDBRestorationState = "installation-db-restoration-triggering"

	// InstallationDBRestorationStateStateInProgress is an installation db restoration that is currently running.
	InstallationDBRestorationStateStateInProgress InstallationDBRestorationState = "installation-db-restoration-in-progress"

	// InstallationDBRestorationStateStateFinishing is an installation db restoration that is finalizing restoration.
	InstallationDBRestorationStateStateFinishing InstallationDBRestorationState = "installation-db-restoration-finishing"

	// TODO: will need more states probably

	// InstallationDBRestorationStateStateSucceeded is an installation db restoration that have finished with success.
	InstallationDBRestorationStateStateSucceeded InstallationDBRestorationState = "installation-db-restoration-succeeded"
	// InstallationDBRestorationStateStateFailed if an installation db restoration that have failed.
	InstallationDBRestorationStateStateFailed InstallationDBRestorationState = "installation-db-restoration-failed"
)

// AllInstallationBackupStatesPendingWork is a list of all backup states that
// the supervisor will attempt to transition towards stable on the next "tick".
var AllInstallationDBRestorationStatesPendingWork = []InstallationDBRestorationState{
	InstallationDBRestorationStateStateRequested,
	InstallationDBRestorationStateStateTriggeringRestoration,
	InstallationDBRestorationStateStateInProgress,
	InstallationDBRestorationStateStateFinishing,
}

// InstallationDBRestorationFilter describes the parameters used to constrain a set of installation-db-restoration.
type InstallationDBRestorationFilter struct {
	Paging
	IDs                   []string
	InstallationID        string
	ClusterInstallationID string
	States                []InstallationDBRestorationState
}

func EnsureReadyForDBRestoration(installation *Installation) error {
	var errs []string

	if installation.State != InstallationStateHibernating &&
		installation.State != InstallationStateDBRestorationInProgress {
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


type InstallationDBRestoration struct {
	ID string // TODO: remove if you leave it inline
	InstallationID string
	BackupID string
	RequestAt int64

	ClusterInstallationID string

	CompleteAt int64

	/// States? Original? Finished?
}

type InstallationDBRestorationRequest struct {
	BackupID string
}


// TODO: test
// NewInstallationDBRestorationRequestFromReader will create a InstallationDBRestorationRequest from an
// io.Reader with JSON data.
func NewInstallationDBRestorationRequestFromReader(reader io.Reader) (*InstallationDBRestorationRequest, error) {
	var restoreRequest InstallationDBRestorationRequest
	err := json.NewDecoder(reader).Decode(&restoreRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode installation db restore request")
	}

	return &restoreRequest, nil
}

