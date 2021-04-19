package components

import (
	"github.com/mattermost/mattermost-cloud/model"
	"net/http"
)

// TODO: what should be the name of this package

type installationBackupStore interface {
	IsInstallationBackupRunning(installationID string) (bool, error)
	CreateInstallationBackup(backup *model.InstallationBackup) error
}

func TriggerInstallationBackup(store installationBackupStore, installation *model.Installation) (*model.InstallationBackup, error) {
	if err := model.EnsureInstallationReadyForBackup(installation); err != nil {
		return nil, ErrWrap(http.StatusBadRequest, err, "installation cannot be backed up")
	}

	backupRunning, err := store.IsInstallationBackupRunning(installation.ID)
	if err != nil {
		return nil, ErrWrap(http.StatusInternalServerError, err, "failed to check if backup is running for Installation")
	}
	if backupRunning {
		return nil, ErrWrap(http.StatusBadRequest, err, "backup for the installation is already requested or in progress")
	}

	backup := &model.InstallationBackup{
		InstallationID: installation.ID,
		State:          model.InstallationBackupStateBackupRequested,
	}

	err = store.CreateInstallationBackup(backup)
	if err != nil {
		return nil, ErrWrap(http.StatusInternalServerError, err, "failed to create installation backup")
	}

	return backup, nil
}
