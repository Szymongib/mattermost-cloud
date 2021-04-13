package components

import (
	"github.com/mattermost/mattermost-cloud/model"
	"net/http"
)

type installationRestorationStore interface {
	CreateInstallationDBRestoration(restoration *model.InstallationDBRestorationOperation) error
}

func TriggerInstallationDBRestoration(store installationRestorationStore, installation *model.Installation, backup *model.InstallationBackup) (*model.InstallationDBRestorationOperation, error) {
	if err := model.EnsureReadyForDBRestoration(installation, backup); err != nil {
		return nil, ErrWrap(http.StatusBadRequest, err, "installation cannot be restored")
	}

	dbRestoration := model.InstallationDBRestorationOperation{
		InstallationID:          installation.ID,
		BackupID:                backup.ID,
		State:                   model.InstallationDBRestorationStateRequested,
	}

	err := store.CreateInstallationDBRestoration(&dbRestoration)
	if err != nil {
		return nil, ErrWrap(http.StatusInternalServerError, err, "failed to create Installation DB restoration operation")
	}

	return &dbRestoration, nil
}
