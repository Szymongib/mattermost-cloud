// Copyright (c) YEAR-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package db_migration

import (
	"github.com/mattermost/mattermost-cloud/e2e/workflow"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"os"
)

type DBMigrationTest struct {
	Flow *workflow.DBMigrationFlow
	Workflow *workflow.Workflow
}

func SetupDBMigrationCommitTest() (*DBMigrationTest, error) {
	flow := setupDBMigrationTestFlow()
	work := commitDBMigrationWorkflow(flow)

	return &DBMigrationTest{
		Flow:     flow,
		Workflow: work,
	}, nil
}

func SetupDBMigrationRollbackTest() (*DBMigrationTest, error) {
	flow := setupDBMigrationTestFlow()
	work := rollbackDBMigrationWorkflow(flow)

	return &DBMigrationTest{
		Flow:     flow,
		Workflow: work,
	}, nil
}

func setupDBMigrationTestFlow() (*workflow.DBMigrationFlow) {
	// TODO: read envs etc

	provisionerURL := StrEnvOrDefault("PROVISIONER_URL", "http://localhost:8075")
	dbType :=          model.InstallationDatabaseMultiTenantRDSPostgres
	fileStoreType :=   model.InstallationFilestoreBifrost
	//SourceDBID:      "rds-cluster-multitenant-050365fcbb1170e4b-07061b50",
	destinationDBID := "rds-cluster-multitenant-050365fcbb1170e4b-migration"


	client := model.NewClient(provisionerURL)

	params := workflow.DBMigrationFlowParams{
		InstallationFlowParams: workflow.InstallationFlowParams{
			DBType:        dbType,
			FileStoreType: fileStoreType,
		},
		DestinationDBID:        destinationDBID,
	}

	return workflow.NewDBMigrationFlow(params, client, logrus.New())
}


func (w *DBMigrationTest) Run() error {
	err := workflow.RunWorkflow(w.Workflow, logrus.New())
	if err != nil {
		return errors.Wrap(err, "error running workflow")
	}
	return nil
}


func StrEnvOrDefault(env, def string) string {
	val := os.Getenv(env)
	if val == "" {
		return def
	}
	return val
}

