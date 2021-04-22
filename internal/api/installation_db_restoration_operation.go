package api

import (
	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/internal/components"
	"github.com/mattermost/mattermost-cloud/internal/webhook"
	"github.com/mattermost/mattermost-cloud/model"
	"net/http"
	"time"
)

// initInstallationRestoration registers installation restoration operation endpoints on the given router.
func initInstallationRestoration(apiRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc) *contextHandler {
		return newContextHandler(context, handler)
	}

	restorationsRouter := apiRouter.PathPrefix("/operations/database/restorations").Subrouter()

	restorationsRouter.Handle("", addContext(handleTriggerInstallationDBRestoration)).Methods("POST")
	restorationsRouter.Handle("", addContext(handleGetInstallationDBRestorationOperations)).Methods("GET")

	restorationRouter := apiRouter.PathPrefix("/operations/database/restoration/{restoration:[A-Za-z0-9]{26}}").Subrouter()
	restorationRouter.Handle("", addContext(handleGetInstallationDBRestorationOperation)).Methods("GET")
}

// TODO:  tests

// handleTriggerInstallationDBRestoration responds to POST /api/installations/operations/database/restorations,
// requests restoration of Installation's data.
func handleTriggerInstallationDBRestoration(c *Context, w http.ResponseWriter, r *http.Request) {
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, dbRestoration)
}

// handleGetInstallationDBRestorationOperations responds to GET /api/installations/operations/database/restorations,
// returns list of installation restoration operation.
func handleGetInstallationDBRestorationOperations(c *Context, w http.ResponseWriter, r *http.Request) {
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, dbRestorations)
}

// handleGetInstallationDBRestorationOperation responds to GET /api/installations/operations/database/restoration/{restoration},
// returns specified installation restoration operation.
func handleGetInstallationDBRestorationOperation(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	restorationID := vars["restoration"]

	c.Logger = c.Logger.
		WithField("action", "get-installation-db-restoration").
		WithField("restoration-operation", restorationID)

	dbRestorationOp, err := c.Store.GetInstallationDBRestorationOperation(restorationID)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to get installation restoration")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if dbRestorationOp == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, dbRestorationOp)
}