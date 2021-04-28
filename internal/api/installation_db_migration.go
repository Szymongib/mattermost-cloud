// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/internal/components"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/webhook"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

// initInstallationMigration registers installation migration operation endpoints on the given router.
func initInstallationMigration(apiRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc) *contextHandler {
		return newContextHandler(context, handler)
	}

	restorationsRouter := apiRouter.PathPrefix("/operations/database/migrations").Subrouter()

	restorationsRouter.Handle("", addContext(handleTriggerInstallationDatabaseMigration)).Methods("POST")
	restorationsRouter.Handle("", addContext(handleGetInstallationDBMigrationOperations)).Methods("GET")

	//restorationRouter := apiRouter.PathPrefix("/operations/database/restoration/{restoration:[A-Za-z0-9]{26}}").Subrouter()
	//restorationRouter.Handle("", addContext(handleGetInstallationDBRestorationOperation)).Methods("GET")
}

// TODO: comments + tests
func handleTriggerInstallationDatabaseMigration(c *Context, w http.ResponseWriter, r *http.Request) {
	c.Logger = c.Logger.WithField("action", "migrate-installation-database")

	migrationRequest, err := model.NewInstallationDBMigrationRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	c.Logger = c.Logger.WithField("installation", migrationRequest.InstallationID)

	newState := model.InstallationStateDBMigrationInProgress

	installationDTO, status, unlockOnce := getInstallationForTransition(c, migrationRequest.InstallationID, newState)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	currentDB, err := c.Store.GetMultitenantDatabaseForInstallationID(installationDTO.ID)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to get current multi-tenant database for installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = validateDBMigration(c, installationDTO.Installation, migrationRequest, currentDB)
	if err != nil {
		c.Logger.WithError(err).Errorf("Cannot migrate installation database")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	dbMigrationOperation := &model.InstallationDBMigrationOperation{
		InstallationID:         migrationRequest.InstallationID,
		SourceDatabase:         installationDTO.Database,
		DestinationDatabase:    migrationRequest.DestinationDatabase,
		SourceMultiTenant:      &model.MultiTenantDBMigrationData{DatabaseID: currentDB.ID},
		DestinationMultiTenant: migrationRequest.DestinationMultiTenant,
	}

	oldInstallationState := installationDTO.State

	dbMigrationOperation, err = c.Store.TriggerInstallationDBMigration(dbMigrationOperation, installationDTO.Installation)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to trigger DB migration operation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	webhookPayload := &model.WebhookPayload{
		Type:      model.TypeInstallationDBMigration,
		ID:        dbMigrationOperation.ID,
		NewState:  string(dbMigrationOperation.State),
		OldState:  "n/a",
		Timestamp: time.Now().UnixNano(),
		ExtraData: map[string]string{"Installation": dbMigrationOperation.InstallationID, "Environment": c.Environment},
	}
	err = webhook.SendToAllWebhooks(c.Store, webhookPayload, c.Logger.WithField("webhookEvent", webhookPayload.NewState))
	if err != nil {
		c.Logger.WithError(err).Error("Unable to process and send webhooks")
	}

	installationWebhookPayload := &model.WebhookPayload{
		Type:      model.TypeInstallation,
		ID:        installationDTO.ID,
		NewState:  installationDTO.State,
		OldState:  oldInstallationState,
		Timestamp: time.Now().UnixNano(),
		ExtraData: map[string]string{"DNS": installationDTO.DNS, "Environment": c.Environment},
	}
	err = webhook.SendToAllWebhooks(c.Store, installationWebhookPayload, c.Logger.WithField("webhookEvent", installationWebhookPayload.NewState))
	if err != nil {
		c.Logger.WithError(err).Error("Unable to process and send webhooks")
	}

	unlockOnce()
	c.Supervisor.Do()

	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, dbMigrationOperation)
}

func handleGetInstallationDBMigrationOperations(c *Context, w http.ResponseWriter, r *http.Request) {
	c.Logger = c.Logger.
		WithField("action", "list-installation-db-migrations")

	paging, err := parsePaging(r.URL)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse paging parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	installationID := r.URL.Query().Get("installation")
	state := r.URL.Query().Get("state")
	var states []model.InstallationDBMigrationOperationState
	if state != "" {
		states = append(states, model.InstallationDBMigrationOperationState(state))
	}

	dbMigrations, err := c.Store.GetInstallationDBMigrationOperations(&model.InstallationDBMigrationFilter{
		Paging:         paging,
		InstallationID: installationID,
		States:         states,
	})
	if err != nil {
		c.Logger.WithError(err).Error("Failed to list installation migrations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, dbMigrations)
}

func validateDBMigration(c *Context, installation *model.Installation, migrationRequest *model.InstallationDBMigrationRequest, currentDB *model.MultitenantDatabase) error {
	if migrationRequest.DestinationDatabase != model.InstallationDatabaseMultiTenantRDSPostgres ||
		installation.Database != model.InstallationDatabaseMultiTenantRDSPostgres {
		return errors.Errorf("db migration is supported when both source and destination are %q database", model.InstallationDatabaseMultiTenantRDSPostgres)
	}

	if migrationRequest.DestinationMultiTenant == nil {
		return errors.New("destination database data not provided")
	}

	destinationDB, err := c.Store.GetMultitenantDatabase(migrationRequest.DestinationMultiTenant.DatabaseID)
	if err != nil {
		return errors.Wrap(err, "failed to get destination multi-tenant database")
	}
	if destinationDB == nil {
		return errors.Errorf("destination database with id %q not found", migrationRequest.DestinationMultiTenant.DatabaseID)
	}

	if currentDB.ID == destinationDB.ID {
		return errors.New("destination database is the same as current")
	}

	if currentDB.VpcID != destinationDB.VpcID {
		return errors.New("databases VPCs do not match, only migration inside the same VPC is supported")
	}

	err = components.ValidateDBMigrationDestination(c.Store, destinationDB, installation.ID, aws.DefaultRDSMultitenantDatabasePostgresCountLimit)
	if err != nil {
		return errors.Wrap(err, "destination database validation failed")
	}

	return nil
}
