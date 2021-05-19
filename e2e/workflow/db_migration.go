// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package workflow

import (
	"context"
	"k8s.io/client-go/kubernetes"
	"strings"

	"github.com/mattermost/mattermost-cloud/e2e/pkg"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func NewDBMigrationFlow(params DBMigrationFlowParams, client *model.Client, kubeClient kubernetes.Interface, logger logrus.FieldLogger) *DBMigrationFlow {
	installationFlow := NewInstallationFlow(params.InstallationFlowParams, client, kubeClient, logger)

	return &DBMigrationFlow{
		InstallationFlow: installationFlow,
		client:           client,
		kubeClient:       kubeClient,
		logger:           logger.WithField("flow", "db-migration"),
		Params:           params,
		Meta:             DBMigrationFlowMeta{},
	}
}

type DBMigrationFlow struct {
	*InstallationFlow

	client     *model.Client
	kubeClient kubernetes.Interface
	logger     logrus.FieldLogger

	Params DBMigrationFlowParams
	Meta   DBMigrationFlowMeta
}

type DBMigrationFlowParams struct {
	InstallationFlowParams
	DestinationDBID string
}

type DBMigrationFlowMeta struct {
	SourceDBID           string
	MigrationOperationID string

	MigratedDBConnStr string
}

func (w *DBMigrationFlow) GetMultiTenantDBID(ctx context.Context) error {
	dbs, err := w.client.GetMultitenantDatabases(&model.GetDatabasesRequest{
		Paging: model.AllPagesNotDeleted(),
	})
	if err != nil {
		return errors.Wrap(err, "while getting multi tenant dbs")
	}

	installationDB, found := findInstallationDB(dbs, w.InstallationFlow.Meta.InstallationID)
	if !found {
		return errors.New("failed to find multi tenant database for installation")
	}
	w.logger.Infof("Found installation multi tenant db with ID: %s", installationDB.ID)

	w.Meta.SourceDBID = installationDB.ID

	return nil
}

func (w *DBMigrationFlow) RunDBMigration(ctx context.Context) error {
	if w.Meta.MigrationOperationID == "" {
		migrationOP, err := w.client.MigrateInstallationDatabase(&model.InstallationDBMigrationRequest{
			InstallationID:         w.InstallationFlow.Meta.InstallationID,
			DestinationDatabase:    model.InstallationDatabaseMultiTenantRDSPostgres,
			DestinationMultiTenant: &model.MultiTenantDBMigrationData{DatabaseID: w.Params.DestinationDBID},
		})
		if err != nil {
			return errors.Wrap(err, "while triggering migration")
		}
		w.Meta.MigrationOperationID = migrationOP.ID
	}

	err := pkg.WaitForDBMigrationToFinish(w.client, w.Meta.MigrationOperationID, w.logger)
	if err != nil {
		return errors.Wrap(err, "while waiting for migration")
	}

	return nil
}

func (w *DBMigrationFlow) AssertMigrationSuccessful(ctx context.Context) error {
	connStr, err := pkg.GetConnectionString(w.client, w.InstallationFlow.Meta.ClusterInstallationID)
	if err != nil {
		return errors.Wrap(err, "while getting connection str")
	}
	w.Meta.MigratedDBConnStr = connStr

	if w.InstallationFlow.Meta.ConnectionString == w.Meta.MigratedDBConnStr {
		return errors.New("error: connection strings are equal")
	}

	if !strings.Contains(w.Meta.MigratedDBConnStr, w.Params.DestinationDBID) {
		return errors.New("error: migrated connection string does not contain destination db id")
	}

	export, err := pkg.GetBulkExportStats(w.client, w.kubeClient, w.InstallationFlow.Meta.ClusterInstallationID, w.InstallationFlow.Meta.InstallationID, w.logger)
	if err != nil {
		return errors.Wrap(err, "while getting CSV export")
	}
	w.logger.Infof("Bulk export stats after migration: %v", export)
	if export != w.InstallationFlow.Meta.BulkExportStats {
		return errors.Errorf("error: export after migration differs from original export, original: %v, new: %v", w.InstallationFlow.Meta.BulkExportStats, export)
		//w.logger.Errorf( "error: export after migration differs from original export, original: %v, new: %v", w.InstallationFlow.Meta.BulkExportStats, export)
	}

	return nil
}

func (w *DBMigrationFlow) CommitMigration(ctx context.Context) error {
	migrationOP, err := w.client.CommitInstallationDBMigration(w.Meta.MigrationOperationID)
	if err != nil {
		return errors.Wrap(err, "while committing migration")
	}
	if migrationOP.State != model.InstallationDBMigrationStateCommitted {
		return errors.Errorf("installation db migration state in not commited, state: %s", migrationOP.State)
	}

	return nil
}

func (w *DBMigrationFlow) RollbackMigration(ctx context.Context) error {
	migrationOP, err := w.client.GetInstallationDBMigrationOperation(w.Meta.MigrationOperationID)
	if err != nil {
		return errors.Wrap(err, "while getting migration operation to roll back")
	}
	if migrationOP.State == model.InstallationDBMigrationStateRollbackFinished {
		w.logger.Info("db migration already rolled back")
		return nil
	}

	if migrationOP.State == model.InstallationDBMigrationStateSucceeded {
		migrationOP, err = w.client.RollbackInstallationDBMigration(w.Meta.MigrationOperationID)
		if err != nil {
			return errors.Wrap(err, "while rolling back migration")
		}
	}

	if migrationOP.State != model.InstallationDBMigrationStateRollbackRequested {
		return errors.Errorf("db migration operation is in unexpected state: %s", migrationOP.State)
	}

	err = pkg.WaitForDBMigrationRollbackToFinish(w.client, migrationOP.ID, w.logger)
	if err != nil {
		return errors.Wrap(err, "while waiting for rollback to finish")
	}

	return nil
}

func (w *DBMigrationFlow) AssertRollbackSuccessful(ctx context.Context) error {
	connStr, err := pkg.GetConnectionString(w.client, w.InstallationFlow.Meta.ClusterInstallationID)
	if err != nil {
		return errors.Wrap(err, "while getting connection str")
	}

	if w.InstallationFlow.Meta.ConnectionString != connStr {
		return errors.New("error: connection string does not match original connection string")
	}

	if !strings.Contains(connStr, w.Meta.SourceDBID) {
		return errors.New("error: connection string does not contain source db id")
	}

	export, err := pkg.GetBulkExportStats(w.client, w.kubeClient, w.InstallationFlow.Meta.ClusterInstallationID, w.InstallationFlow.Meta.InstallationID, w.logger)
	if err != nil {
		return errors.Wrap(err, "while getting CSV export")
	}
	w.logger.Infof("Bulk export stats after rollback: %v", export)
	if export != w.InstallationFlow.Meta.BulkExportStats {
		return errors.Errorf("error: export after rollback differs from original export, original: %v, new: %v", w.InstallationFlow.Meta.BulkExportStats, export)
		//w.logger.Errorf( "error: export after rollback differs from original export, original: %v, new: %v", w.InstallationFlow.Meta.BulkExportStats, export)
	}

	return nil
}

func findInstallationDB(dbs []*model.MultitenantDatabase, installationID string) (model.MultitenantDatabase, bool) {
	for _, db := range dbs {
		if db.Installations.Contains(installationID) {
			return *db, true
		}
	}
	return model.MultitenantDatabase{}, false
}