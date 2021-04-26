// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-cloud/internal/store"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/internal/webhook"
	"github.com/mattermost/mattermost-cloud/model"
)

// initInstallation registers installation endpoints on the given router.
func initInstallation(apiRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc) *contextHandler {
		return newContextHandler(context, handler)
	}

	installationsRouter := apiRouter.PathPrefix("/installations").Subrouter()
	initInstallationBackup(installationsRouter, context)
	initInstallationRestoration(installationsRouter, context)

	// TODO: move migrations stuff
	installationsRouter.Handle("/database/migrate", addContext(handleInstallationDatabaseMigration)).Methods("POST")
	installationsRouter.Handle("/database/migrations", addContext(handleGetInstallationDBMigrationOperations)).Methods("GET")

	installationsRouter.Handle("", addContext(handleGetInstallations)).Methods("GET")
	installationsRouter.Handle("", addContext(handleCreateInstallation)).Methods("POST")
	installationsRouter.Handle("/count", addContext(handleGetNumberOfInstallations)).Methods("GET")
	installationsRouter.Handle("/status", addContext(handleGetInstallationsStatus)).Methods("GET")

	installationRouter := apiRouter.PathPrefix("/installation/{installation:[A-Za-z0-9]{26}}").Subrouter()
	installationRouter.Handle("", addContext(handleGetInstallation)).Methods("GET")
	installationRouter.Handle("", addContext(handleRetryCreateInstallation)).Methods("POST")
	installationRouter.Handle("/mattermost", addContext(handleUpdateInstallation)).Methods("PUT")
	installationRouter.Handle("/group/{group}", addContext(handleJoinGroup)).Methods("PUT")
	installationRouter.Handle("/group", addContext(handleLeaveGroup)).Methods("DELETE")
	installationRouter.Handle("/hibernate", addContext(handleHibernateInstallation)).Methods("POST")
	installationRouter.Handle("/wakeup", addContext(handleWakeupInstallation)).Methods("POST")
	installationRouter.Handle("", addContext(handleDeleteInstallation)).Methods("DELETE")
	installationRouter.Handle("/annotations", addContext(handleAddInstallationAnnotations)).Methods("POST")
	installationRouter.Handle("/annotation/{annotation-name}", addContext(handleDeleteInstallationAnnotation)).Methods("DELETE")
}

// handleGetInstallation responds to GET /api/installation/{installation}, returning the installation in question.
func handleGetInstallation(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	includeGroupConfig, includeGroupConfigOverrides, err := parseGroupConfig(r.URL)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse group config parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	installation, err := c.Store.GetInstallationDTO(installationID, includeGroupConfig, includeGroupConfigOverrides)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if installation == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, installation)
}

// handleGetInstallations responds to GET /api/installations, returning the specified page of installations.
func handleGetInstallations(c *Context, w http.ResponseWriter, r *http.Request) {
	var err error

	paging, err := parsePaging(r.URL)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse paging parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	includeGroupConfig, includeGroupConfigOverrides, err := parseGroupConfig(r.URL)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse group parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	owner := r.URL.Query().Get("owner")
	group := r.URL.Query().Get("group")
	state := r.URL.Query().Get("state")
	dns := r.URL.Query().Get("dns_name")

	filter := &model.InstallationFilter{
		OwnerID: owner,
		GroupID: group,
		State:   state,
		Paging:  paging,
		DNS:     dns,
	}

	installations, err := c.Store.GetInstallationDTOs(filter, includeGroupConfig, includeGroupConfigOverrides)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query installations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if installations == nil {
		installations = []*model.InstallationDTO{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, installations)
}

// handlerGetNumberOfInstallations responds to GET /api/installations/count, returning the
// number of non-deleted installations
func handleGetNumberOfInstallations(c *Context, w http.ResponseWriter, r *http.Request) {
	includeDeleted, err := parseBool(r.URL, "include_deleted", false)
	if err != nil {
		includeDeleted = false
	}
	installationsCount, err := c.Store.GetInstallationsCount(includeDeleted)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query the number of installations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	result := model.InstallationsCount{Count: installationsCount}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, result)
}

// handleGetInstallationsStatus responds to GET /api/installations/status,
// returning the status of all non-deleted installations
func handleGetInstallationsStatus(c *Context, w http.ResponseWriter, r *http.Request) {
	installationsStatus, err := c.Store.GetInstallationsStatus()
	if err != nil {
		c.Logger.WithError(err).Error("failed to query for installations status")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, installationsStatus)
}

// handleCreateInstallation responds to POST /api/installations, beginning the process of creating
// a new installation.
func handleCreateInstallation(c *Context, w http.ResponseWriter, r *http.Request) {
	createInstallationRequest, err := model.NewCreateInstallationRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var group *model.Group
	var status int
	groupUnlockOnce := func() {}
	if len(createInstallationRequest.GroupID) != 0 {
		group, status, groupUnlockOnce = lockGroup(c, createInstallationRequest.GroupID)
		if status != 0 {
			w.WriteHeader(status)
			return
		}
		defer groupUnlockOnce()
		if group.IsDeleted() {
			c.Logger.Errorf("cannot join installation to deleted group %s", createInstallationRequest.GroupID)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	installation := model.Installation{
		OwnerID:                    createInstallationRequest.OwnerID,
		GroupID:                    &createInstallationRequest.GroupID,
		Version:                    createInstallationRequest.Version,
		Image:                      createInstallationRequest.Image,
		DNS:                        createInstallationRequest.DNS,
		Database:                   createInstallationRequest.Database,
		Filestore:                  createInstallationRequest.Filestore,
		License:                    createInstallationRequest.License,
		Size:                       createInstallationRequest.Size,
		Affinity:                   createInstallationRequest.Affinity,
		APISecurityLock:            createInstallationRequest.APISecurityLock,
		MattermostEnv:              createInstallationRequest.MattermostEnv,
		SingleTenantDatabaseConfig: createInstallationRequest.SingleTenantDatabaseConfig.ToDBConfig(createInstallationRequest.Database),
		CRVersion:                  model.DefaultCRVersion,
		State:                      model.InstallationStateCreationRequested,
	}

	annotations, err := model.AnnotationsFromStringSlice(createInstallationRequest.Annotations)
	if err != nil {
		c.Logger.WithError(err).Error("failed to validate extra annotations")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = c.Store.CreateInstallation(&installation, annotations)
	if err != nil {
		c.Logger.WithError(err).Error("failed to create installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	webhookPayload := &model.WebhookPayload{
		Type:      model.TypeInstallation,
		ID:        installation.ID,
		NewState:  model.InstallationStateCreationRequested,
		OldState:  "n/a",
		Timestamp: time.Now().UnixNano(),
		ExtraData: map[string]string{"DNS": installation.DNS, "Environment": c.Environment},
	}
	err = webhook.SendToAllWebhooks(c.Store, webhookPayload, c.Logger.WithField("webhookEvent", webhookPayload.NewState))
	if err != nil {
		c.Logger.WithError(err).Error("Unable to process and send webhooks")
	}

	groupUnlockOnce()
	c.Supervisor.Do()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, installation.ToDTO(annotations))
}

// handleRetryCreateInstallation responds to POST /api/installation/{installation}, retrying a
// previously failed creation.
//
// Note that other operations on a installation may be retried by simply repeating the same request,
// but repeating handleCreateInstallation would create a second installation.
func handleRetryCreateInstallation(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	newState := model.InstallationStateCreationRequested

	installationDTO, status, unlockOnce := getInstallationForTransition(c, installationID, newState)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	err := updateInstallationState(c, installationDTO, newState)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to update installation state to %q", newState)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Notify even if we didn't make changes, to expedite even the no-op operations above.
	unlockOnce()
	c.Supervisor.Do()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, installationDTO)
}

// handleUpdateInstallation responds to PUT /api/installation/{installation}/mattermost,
// updating the installation to the Mattermost configuration embedded in the request.
func handleUpdateInstallation(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	patchInstallationRequest, err := model.NewPatchInstallationRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	newState := model.InstallationStateUpdateRequested

	installationDTO, status, unlockOnce := getInstallationForTransition(c, installationID, newState)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	oldState := installationDTO.State

	if patchInstallationRequest.Apply(installationDTO.Installation) {
		installationDTO.State = newState

		err = c.Store.UpdateInstallation(installationDTO.Installation)
		if err != nil {
			c.Logger.WithError(err).Error("failed to update installation")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		webhookPayload := &model.WebhookPayload{
			Type:      model.TypeInstallation,
			ID:        installationDTO.ID,
			NewState:  newState,
			OldState:  oldState,
			Timestamp: time.Now().UnixNano(),
			ExtraData: map[string]string{"DNS": installationDTO.DNS, "Environment": c.Environment},
		}
		err = webhook.SendToAllWebhooks(c.Store, webhookPayload, c.Logger.WithField("webhookEvent", webhookPayload.NewState))
		if err != nil {
			c.Logger.WithError(err).Error("Unable to process and send webhooks")
		}
	}

	unlockOnce()
	c.Supervisor.Do()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, installationDTO)
}

// handleJoinGroup responds to PUT /api/installation/{installation}/group/{group}, joining the group.
func handleJoinGroup(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	groupID := vars["group"]
	c.Logger = c.Logger.WithField("installation", installationID)

	installationDTO, status, installationUnlockOnce := lockInstallation(c, installationID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer installationUnlockOnce()

	if installationDTO.APISecurityLock {
		logSecurityLockConflict("installation", c.Logger)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	group, status, groupUnlockOnce := lockGroup(c, groupID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer groupUnlockOnce()
	if group.IsDeleted() {
		c.Logger.Errorf("cannot join installation to deleted group %s", groupID)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Update the installation, but don't directly modify the configuration.
	// The supervisor will manage this later.
	if installationDTO.GroupID == nil || *installationDTO.GroupID != groupID {
		installationDTO.GroupID = &groupID

		err := c.Store.UpdateInstallation(installationDTO.Installation)
		if err != nil {
			c.Logger.WithError(err).Error("failed to update installation")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	installationUnlockOnce()
	groupUnlockOnce()
	c.Supervisor.Do()

	w.WriteHeader(http.StatusOK)
}

// handleLeaveGroup responds to DELETE /api/installation/{installation}/group,
// leaving any existing group.
func handleLeaveGroup(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	retainConfig, err := parseBool(r.URL, "retain_config", true)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse retain_config setting")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	newState := model.InstallationStateUpdateRequested

	// TODO: does it make sense to enforce normal update-requested valid states?
	// Should there be more or less valid states? Review this when necessary.
	installationDTO, status, unlockOnce := getInstallationForTransition(c, installationID, newState)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if installationDTO.GroupID != nil {
		installationDTO.State = newState
		installationDTO.GroupID = nil
		installationDTO.GroupSequence = nil

		if retainConfig {
			// The installation is leaving the group, but the config is being set
			// to the group-merged version used while it was in the group. To do
			// so, we will get a merged copy of the installation out and will
			// manually update the necessary values.
			mergedInstallation, err := c.Store.GetInstallation(installationID, true, false)
			if err != nil {
				c.Logger.WithError(err).Error("failed to get group-merged installation")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			installationDTO.Version = mergedInstallation.Version
			installationDTO.Image = mergedInstallation.Image
			installationDTO.MattermostEnv = mergedInstallation.MattermostEnv
		}

		err := c.Store.UpdateInstallation(installationDTO.Installation)
		if err != nil {
			c.Logger.WithError(err).Error("failed to update installation")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	unlockOnce()
	c.Supervisor.Do()

	w.WriteHeader(http.StatusOK)
}

// handleHibernateInstallation responds to POST /api/installation/{installation}/hibernate,
// moving the installation into a hibernation state.
func handleHibernateInstallation(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	newState := model.InstallationStateHibernationRequested

	installationDTO, status, unlockOnce := getInstallationForTransition(c, installationID, newState)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	err := updateInstallationState(c, installationDTO, newState)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to update installation state to %q", newState)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	unlockOnce()
	c.Supervisor.Do()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, installationDTO)
}

// handleWakeupInstallation responds to POST /api/installation/{installation}/wakeup,
// moving the installation out of a hibernation state.
func handleWakeupInstallation(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	newState := model.InstallationStateWakeUpRequested

	installationDTO, status, unlockOnce := getInstallationForTransition(c, installationID, newState)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	err := updateInstallationState(c, installationDTO, newState)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to update installation state to %q", newState)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	unlockOnce()
	c.Supervisor.Do()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, installationDTO)
}

// handleDeleteInstallation responds to DELETE /api/installation/{installation}, beginning the process of
// deleting the installation.
func handleDeleteInstallation(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	newState := model.InstallationStateDeletionRequested

	installationDTO, status, unlockOnce := getInstallationForTransition(c, installationID, newState)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	runningBackups, err := c.Store.GetInstallationBackups(&model.InstallationBackupFilter{
		InstallationID: installationID,
		States:         model.AllInstallationBackupsStatesRunning,
		Paging:         model.AllPagesNotDeleted(),
	})
	if err != nil {
		c.Logger.WithError(err).Error("failed to get list of running backups")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(runningBackups) > 0 {
		c.Logger.Error("there are running backups for the installation, cannot delete")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = updateInstallationState(c, installationDTO, newState)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to update installation state to %q", newState)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	unlockOnce()
	c.Supervisor.Do()

	w.WriteHeader(http.StatusAccepted)
}

// handleAddInstallationAnnotations responds to POST /api/installation/{installation}/annotations,
// adds the set of annotations to the Installation.
func handleAddInstallationAnnotations(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID).WithField("action", "add-installation-annotations")

	installationDTO, status, unlockOnce := lockInstallation(c, installationID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if installationDTO.APISecurityLock {
		logSecurityLockConflict("installation", c.Logger)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	annotations, err := annotationsFromRequest(r)
	if err != nil {
		c.Logger.WithError(err).Error("failed to get annotations from request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	annotations, err = c.Store.CreateInstallationAnnotations(installationID, annotations)
	if err != nil {
		c.Logger.WithError(err).Error("failed to create installation annotations")
		if errors.Is(err, store.ErrInstallationAnnotationDoNotMatchClusters) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	installationDTO.Annotations = append(installationDTO.Annotations, annotations...)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, installationDTO)
}

// handleDeleteInstallationAnnotation responds to DELETE /api/installation/{installation}/annotation/{annotation-name},
// removes annotation from the Installation.
func handleDeleteInstallationAnnotation(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	annotationName := vars["annotation-name"]
	c.Logger = c.Logger.
		WithField("installation", installationID).
		WithField("action", "delete-installation-annotation").
		WithField("annotation-name", annotationName)

	installationDTO, status, unlockOnce := lockInstallation(c, installationID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if installationDTO.APISecurityLock {
		logSecurityLockConflict("installation", c.Logger)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	err := c.Store.DeleteInstallationAnnotation(installationID, annotationName)
	if err != nil {
		c.Logger.WithError(err).Error("failed delete cluster annotation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}

// updateInstallationState updates installation state in database and sends appropriate webhook.
func updateInstallationState(c *Context, installationDTO *model.InstallationDTO, newState string) error {
	if installationDTO.State == newState {
		return nil
	}

	webhookPayload := &model.WebhookPayload{
		Type:      model.TypeInstallation,
		ID:        installationDTO.ID,
		NewState:  newState,
		OldState:  installationDTO.State,
		Timestamp: time.Now().UnixNano(),
		ExtraData: map[string]string{"DNS": installationDTO.DNS, "Environment": c.Environment},
	}
	installationDTO.State = newState

	err := c.Store.UpdateInstallationState(installationDTO.Installation)
	if err != nil {
		return err
	}

	err = webhook.SendToAllWebhooks(c.Store, webhookPayload, c.Logger.WithField("webhookEvent", webhookPayload.NewState))
	if err != nil {
		c.Logger.WithError(err).Error("Unable to process and send webhooks")
	}

	return nil
}

// getInstallationForTransition locks the installation and validates if it can be transitioned to desired state.
func getInstallationForTransition(c *Context, installationID, newState string) (*model.InstallationDTO, int, func()) {
	installationDTO, status, unlockOnce := lockInstallation(c, installationID)
	if status != 0 {
		return nil, status, unlockOnce
	}

	if installationDTO.APISecurityLock {
		unlockOnce()
		logSecurityLockConflict("installation", c.Logger)
		return nil, http.StatusForbidden, unlockOnce
	}

	if !installationDTO.ValidTransitionState(newState) {
		unlockOnce()
		c.Logger.Warnf("unable to transition installation to %q while in state %q", newState, installationDTO.State)
		return nil, http.StatusBadRequest, unlockOnce
	}

	return installationDTO, 0, unlockOnce
}
