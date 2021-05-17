// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package workflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-cloud/e2e/pkg"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func NewDBMigrationFlow(params DBMigrationFlowParams, client *model.Client, logger logrus.FieldLogger) *DBMigrationFlow {
	installationFlow := NewInstallationFlow(params.InstallationFlowParams, client, logger)

	return &DBMigrationFlow{
		InstallationFlow: installationFlow,
		client:           client,
		logger:           logger.WithField("flow", "db-migration"),
		Params:           params,
		Meta:             DBMigrationFlowMeta{SourceDBID: "rds-cluster-multitenant-050365fcbb1170e4b-07061b50"}, // TODO: temporary hack
	}
}

type DBMigrationFlow struct {
	*InstallationFlow

	client *model.Client
	logger logrus.FieldLogger

	Params DBMigrationFlowParams
	Meta   DBMigrationFlowMeta
}

type DBMigrationFlowParams struct {
	InstallationFlowParams
	DestinationDBID string
}

type DBMigrationFlowMeta struct {
	SourceDBID           string // TODO: for now hardcode
	MigrationOperationID string

	MigratedDBConnStr string
}

func (w *DBMigrationFlow) GetMultiTenantDBID(ctx context.Context) error {
	// TODO: query all DBs and find instsallation ID?
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

	err := pkg.WaitForDBMigrationToFinish(w.client, w.Meta.MigrationOperationID)
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
	fmt.Println("Migrated connection string: ", connStr)

	if w.InstallationFlow.Meta.ConnectionString == w.Meta.MigratedDBConnStr {
		return errors.New("error: connection strings are equal")
	}

	if !strings.Contains(w.Meta.MigratedDBConnStr, w.Params.DestinationDBID) {
		return errors.New("error: migrated connection string does not contain destination db id")
	}

	//dataExport, err := pkg.ExportCSV(w.client, w.Meta.CI)
	//if err != nil {
	//	return errors.Wrap(err, "while getting CSV export")
	//}
	//w.Meta.MigratedCSVExport = dataExport
	//fmt.Println("Migrated CSV export: ", dataExport)

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
		fmt.Println("db migration already rolled back")
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

	err = pkg.WaitForDBMigrationRollbackToFinish(w.client, migrationOP.ID)
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

	//dataExport, err := pkg.ExportCSV(w.client, w.Meta.CI)
	//if err != nil {
	//	return errors.Wrap(err, "while getting CSV export")
	//}
	//w.Meta.OriginalCSVExport = dataExport
	//fmt.Println("Original CSV export: ", dataExport)

	return nil
}
