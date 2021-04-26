package api

import (
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/webhook"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"net/http"
	"time"
)

// TODO: comments + tests
// TODO: move to backups?
func handleInstallationDatabaseMigration(c *Context, w http.ResponseWriter, r *http.Request) {
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

	dbMigrationOperation := &model.DBMigrationOperation{
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
	clusterInstallationID := r.URL.Query().Get("cluster_installation")
	state := r.URL.Query().Get("state")
	var states []model.DBMigrationOperationState
	if state != "" {
		states = append(states, model.DBMigrationOperationState(state))
	}

	dbMigrations, err := c.Store.GetInstallationDBMigrationOperations(&model.InstallationDBMigrationFilter{
		Paging:                paging,
		InstallationID:        installationID,
		ClusterInstallationID: clusterInstallationID,
		States:                states,
	})
	if err != nil {
		c.Logger.WithError(err).Error("Failed to list installation migrations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, dbMigrations)
}

func validateDBMigration(c *Context, installation *model.Installation, migrationRequest *model.DBMigrationRequest, currentDB *model.MultitenantDatabase) error {
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

	if currentDB.VpcID != destinationDB.VpcID {
		return errors.New("databases VPCs do not match, only migration inside the same VPC is supported")
	}

	weight, err := c.Store.GetInstallationsTotalDatabaseWeight(destinationDB.Installations)
	if err != nil {
		return errors.Wrap(err, "failed to check total weight of installations in destination database")
	}
	if weight >= aws.DefaultRDSMultitenantDatabasePostgresCountLimit {
		return errors.Errorf("cannot migrate to database, installations weight reached the limit: %f", weight)
	}

	return nil
}
