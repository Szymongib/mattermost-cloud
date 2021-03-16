// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/model"
)

// initInstallationBackup registers installation backups endpoints on the given router.
func initInstallationBackup(apiRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc) *contextHandler {
		return newContextHandler(context, handler)
	}

	backupsRouter := apiRouter.PathPrefix("/backups").Subrouter()

	backupsRouter.Handle("", addContext(handleRequestInstallationBackup)).Methods("POST")
	backupsRouter.Handle("", addContext(handleGetInstallationBackups)).Methods("GET")

	backupRouter := apiRouter.PathPrefix("/backup/{backup:[A-Za-z0-9]{26}}").Subrouter()
	backupRouter.Handle("", addContext(handleGetInstallationBackup)).Methods("GET")
	backupRouter.Handle("", addContext(handleDeleteInstallationBackup)).Methods("DELETE")
}

// handleRequestInstallationBackup responds to POST /api/installations/backups,
// requests backup of Installation's data.
func handleRequestInstallationBackup(c *Context, w http.ResponseWriter, r *http.Request) {
	backupRequest, err := model.NewInstallationBackupRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	c.Logger = c.Logger.
		WithField("installation", backupRequest.InstallationID).
		WithField("action", "request-backup")

	installationDTO, status, unlockOnce := lockInstallation(c, backupRequest.InstallationID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if err := model.EnsureBackupCompatible(installationDTO.Installation); err != nil {
		c.Logger.WithError(err).Error("installation cannot be backed up")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	backupRunning, err := c.Store.IsInstallationBackupRunning(installationDTO.ID)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to check if backup is running for Installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if backupRunning {
		c.Logger.Error("Backup for the installation is already requested or in progress")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	backup := &model.InstallationBackup{
		InstallationID: installationDTO.ID,
		State:          model.InstallationBackupStateBackupRequested,
	}

	err = c.Store.CreateInstallationBackup(backup)
	if err != nil {
		c.Logger.Error("Failed to create backup metadata")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.Supervisor.Do()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, backup)
}

// handleGetInstallationBackups responds to GET /api/installations/backups,
// returns backups metadata.
func handleGetInstallationBackups(c *Context, w http.ResponseWriter, r *http.Request) {
	c.Logger = c.Logger.
		WithField("action", "list-installation-backups")

	page, perPage, includeDeleted, err := parsePaging(r.URL)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse paging parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	installationID := r.URL.Query().Get("installation")
	clusterInstallationID := r.URL.Query().Get("cluster_installation")
	state := r.URL.Query().Get("state")

	filter := &model.InstallationBackupFilter{
		InstallationID:        installationID,
		ClusterInstallationID: clusterInstallationID,
		State:                 model.InstallationBackupState(state),
		Page:                  page,
		PerPage:               perPage,
		IncludeDeleted:        includeDeleted,
	}

	backupsMeta, err := c.Store.GetInstallationBackups(filter)
	if err != nil {
		c.Logger.WithError(err).Error("failed to list installation backups")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, backupsMeta)
}

// handleGetInstallationBackup responds to GET /api/installations/backup/{backup},
// returns metadata of specified backup.
func handleGetInstallationBackup(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	backupID := vars["backup"]
	c.Logger = c.Logger.
		WithField("installation", installationID).
		WithField("backup", backupID).
		WithField("action", "get-installation-backup")

	backupMetadata, err := c.Store.GetInstallationBackup(backupID)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to get backup")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if backupMetadata == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, backupMetadata)
}

// handleDeleteInstallationBackup responds to DELETE /api/installations/backup/{backup},
// returns metadata of specified backup.
func handleDeleteInstallationBackup(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	backupID := vars["backup"]
	c.Logger = c.Logger.
		WithField("backup", backupID).
		WithField("action", "delete-installation-backup")

	backup, status, unlockOnce := lockInstallationBackup(c, backupID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if backup.APISecurityLock {
		logSecurityLockConflict("backup", c.Logger)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	backup.State = model.InstallationBackupStateDeletionRequested

	err := c.Store.UpdateInstallationBackupState(backup)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to delete backup metadata")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	unlockOnce()
	c.Supervisor.Do()

	w.WriteHeader(http.StatusAccepted)
}
