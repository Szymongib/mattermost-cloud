package api

import (
	"github.com/mattermost/mattermost-cloud/internal/components"
	"github.com/mattermost/mattermost-cloud/internal/webhook"
	"github.com/mattermost/mattermost-cloud/model"
	"net/http"
	"time"
)

// TODO: comments + tests
// TODO: move to backups?
func handleInstallationDatabaseRestore(c *Context,w http.ResponseWriter, r *http.Request) {
	c.Logger = c.Logger.
		WithField("action", "restore-installation-database")

	restoreRequest, err := model.NewInstallationDBRestorationRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	c.Logger = c.Logger.
		WithField("installation", restoreRequest.InstallationID).
		WithField("backup", restoreRequest.BackupID)

	installationDTO, status, unlockOnce := lockInstallation(c, restoreRequest.InstallationID)
	if status != 0 {
		if status == http.StatusNotFound {
			status = http.StatusInternalServerError
		}
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	// TODO: handle this better here and in backup case
	if installationDTO.State != model.InstallationStateHibernating {
		c.Logger.Errorf("installation needs to be hibernated to start restoration")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if installationDTO.APISecurityLock {
		logSecurityLockConflict("installation", c.Logger)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	backup, err := c.Store.GetInstallationBackup(restoreRequest.BackupID)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to get backup")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if backup == nil {
		c.Logger.Error("Backup not found")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	dbRestoration, err := components.TriggerInstallationDBRestoration(c.Store, installationDTO.Installation, backup)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to trigger installation db restoration")
		w.WriteHeader(components.ErrToStatus(err))
		return
	}

	webhookPayload := &model.WebhookPayload{
		Type:      model.TypeInstallationDBRestoration,
		ID:        dbRestoration.ID,
		NewState:  string(model.InstallationDBRestorationStateRequested),
		OldState:  "n/a",
		Timestamp: time.Now().UnixNano(),
		ExtraData: map[string]string{"Installation": dbRestoration.InstallationID, "Backup": dbRestoration.BackupID, "Environment": c.Environment},
	}

	err = webhook.SendToAllWebhooks(c.Store, webhookPayload, c.Logger.WithField("webhookEvent", webhookPayload.NewState))
	if err != nil {
		c.Logger.WithError(err).Error("Unable to process and send webhooks")
	}

	unlockOnce()
	c.Supervisor.Do()

	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, dbRestoration)
}

func handleGetInstallationDatabaseRestorationOperations(c *Context,w http.ResponseWriter, r *http.Request) {
	c.Logger = c.Logger.
		WithField("action", "list-installation-db-restorations")

	// TODO: filters and stuff

	dbRestorations, err := c.Store.GetInstallationDBRestorations(&model.InstallationDBRestorationFilter{
		Paging:                model.AllPagesWithDeleted(),
		IDs:                   nil,
		InstallationID:        "",
		ClusterInstallationID: "",
		States:                nil,
	})
	if err != nil {
		c.Logger.WithError(err).Error("Failed to list installation restorations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, dbRestorations)
}
