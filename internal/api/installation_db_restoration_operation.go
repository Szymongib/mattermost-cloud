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
// handleTriggerInstallationDatabaseRestoration
func handleTriggerInstallationDatabaseRestoration(c *Context, w http.ResponseWriter, r *http.Request) {
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

	newState := model.InstallationStateDBRestorationInProgress

	installationDTO, status, unlockOnce := getInstallationForTransition(c, restoreRequest.InstallationID, newState)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

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

func handleGetInstallationDatabaseRestorationOperations(c *Context, w http.ResponseWriter, r *http.Request) {
	c.Logger = c.Logger.
		WithField("action", "list-installation-db-restorations")

	paging, err := parsePaging(r.URL)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse paging parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	installationID := r.URL.Query().Get("installation")
	clusterInstallationID := r.URL.Query().Get("cluster_installation")
	state := r.URL.Query().Get("state")
	var states []model.InstallationDBRestorationState
	if state != "" {
		states = append(states, model.InstallationDBRestorationState(state))
	}

	dbRestorations, err := c.Store.GetInstallationDBRestorationOperations(&model.InstallationDBRestorationFilter{
		Paging:                paging,
		InstallationID:        installationID,
		ClusterInstallationID: clusterInstallationID,
		States:                states,
	})
	if err != nil {
		c.Logger.WithError(err).Error("Failed to list installation restorations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, dbRestorations)
}
