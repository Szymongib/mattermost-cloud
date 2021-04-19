package api

import (
	"fmt"
	"github.com/mattermost/mattermost-cloud/model"
	"net/http"
)

// TODO: comments + tests
// TODO: move to backups?
func handleInstallationDatabaseMigration(c *Context,w http.ResponseWriter, r *http.Request) {
	c.Logger = c.Logger.WithField("action", "migrate-installation-database")

	////TODO: remove the backdoor lol
	//db, err := c.Store.GetMultitenantDatabase("rds-cluster-multitenant-050365fcbb1170e4b-07061b50")
	//if err != nil {
	//	panic(err)
	//}
	////db2, err := c.Store.GetMultitenantDatabase("rds-cluster-multitenant-0dead5c7d41f280d2-fcf071c1")
	////if err != nil {
	////	panic(err)
	////}
	//
	//db.Installations = db.Installations[:len(db.Installations)-1]
	////db.Installations.Add("u85ky7xjgfgstyskh1ruxrzhka")
	////db2.Installations.Remove("u85ky7xjgfgstyskh1ruxrzhka")
	//
	//err = c.Store.UpdateMultitenantDatabase(db)
	//if err != nil {
	//	panic(err)
	//}
	////err = c.Store.UpdateMultitenantDatabase(db2)
	////if err != nil {
	////	panic(err)
	////}
	//
	//return

	migrationRequest, err := model.NewDBMigrationRequestFromReader(r.Body)
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

	// TODO: validate

	// If not multitenant postgres (both source and destination) - fail
	// Get current multitenant DB
	// Check if VPC is the same
	// Check if new DB has enough space?

	fmt.Println(installationDTO.ID)
	dbs, err := c.Store.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{InstallationID: installationDTO.ID, Paging: model.AllPagesNotDeleted(), MaxInstallationsLimit: 500})
	if err != nil {
		c.Logger.WithError(err).Error("Failed to get multitenant db for installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if len(dbs) == 0 {
		c.Logger.Error("Multitenant db not found for installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	dbMigrationOperation := &model.DBMigrationOperation{
		InstallationID:                       migrationRequest.InstallationID,
		State:                                model.DBMigrationStateRequested,
		SourceDatabase:                       installationDTO.Database,
		DestinationDatabase:                  model.InstallationDatabaseMultiTenantRDSPostgres, // TODO
		SourceMultiTenant:                    &model.MultiTenantDBMigrationData{DatabaseID: dbs[0].ID},
		DestinationMultiTenant:               migrationRequest.DestinationMultiTenant,
	}

	// TODO: one transaction
	err = c.Store.CreateInstallationDBMigration(dbMigrationOperation)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to create DB migration operation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	installationDTO.State = model.InstallationStateDBMigrationInProgress
	err = c.Store.UpdateInstallation(installationDTO.Installation)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to update installation state")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	unlockOnce()
	c.Supervisor.Do()

	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, dbMigrationOperation)


	//dbRestoration, err := components.TriggerInstallationDBRestoration(c.Store, installationDTO.Installation, backup)
	//if err != nil {
	//	c.Logger.WithError(err).Error("Failed to trigger installation db restoration")
	//	w.WriteHeader(components.ErrToStatus(err))
	//	return
	//}
	//
	//webhookPayload := &model.WebhookPayload{
	//	Type:      model.TypeInstallationDBRestoration,
	//	ID:        dbRestoration.ID,
	//	NewState:  string(model.InstallationDBRestorationStateRequested),
	//	OldState:  "n/a",
	//	Timestamp: time.Now().UnixNano(),
	//	ExtraData: map[string]string{"Installation": dbRestoration.InstallationID, "Backup": dbRestoration.BackupID, "Environment": c.Environment},
	//}
	//
	//err = webhook.SendToAllWebhooks(c.Store, webhookPayload, c.Logger.WithField("webhookEvent", webhookPayload.NewState))
	//if err != nil {
	//	c.Logger.WithError(err).Error("Unable to process and send webhooks")
	//}
	//
	//unlockOnce()
	//c.Supervisor.Do()
	//
	//w.WriteHeader(http.StatusAccepted)
	//outputJSON(c, w, dbRestoration)
}

//func handleGetInstallationDatabaseRestorationOperations(c *Context,w http.ResponseWriter, r *http.Request) {
//	c.Logger = c.Logger.
//		WithField("action", "list-installation-db-restorations")
//
//	// TODO: filters and stuff
//
//	dbRestorations, err := c.Store.GetInstallationDBRestorationOperations(&model.InstallationDBRestorationFilter{
//		Paging:                model.AllPagesWithDeleted(),
//		IDs:                   nil,
//		InstallationID:        "",
//		ClusterInstallationID: "",
//		States:                nil,
//	})
//	if err != nil {
//		c.Logger.WithError(err).Error("Failed to list installation restorations")
//		w.WriteHeader(http.StatusInternalServerError)
//		return
//	}
//
//	w.WriteHeader(http.StatusOK)
//	outputJSON(c, w, dbRestorations)
//}
