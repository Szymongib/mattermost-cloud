// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package db_migration

import (
	"github.com/vrischmann/envconfig"
	"os"

	"github.com/mattermost/mattermost-cloud/e2e/workflow"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type TestConfig struct {
	CloudURL string `envconfig:"default=http://localhost:8075"`
	DestinationDB string `envconfig:"default=rds-cluster-multitenant-050365fcbb1170e4b-migration"`
	InstallationDBType string `envconfig:"default=aws-multitenant-rds-postgres"`
	InstallationFileStoreType string `envconfig:"default=bifrost"`
}

type DBMigrationTest struct {
	Flow     *workflow.DBMigrationFlow
	Workflow *workflow.Workflow
}

func SetupDBMigrationCommitTest() (*DBMigrationTest, error) {
	flow, err := setupDBMigrationTestFlow()
	if err != nil {
		return nil, err
	}
	work := commitDBMigrationWorkflow(flow)

	return &DBMigrationTest{
		Flow:     flow,
		Workflow: work,
	}, nil
}

func SetupDBMigrationRollbackTest() (*DBMigrationTest, error) {
	flow, err := setupDBMigrationTestFlow()
	if err != nil {
		return nil, err
	}
	work := rollbackDBMigrationWorkflow(flow)

	return &DBMigrationTest{
		Flow:     flow,
		Workflow: work,
	}, nil
}

func setupDBMigrationTestFlow() (*workflow.DBMigrationFlow,error) {
	var config TestConfig
	err := envconfig.Init(&config)
	if err != nil {
		return nil, errors.Wrap(err, "Unable to read environment configuration")
	}

	client := model.NewClient(config.CloudURL)

	params := workflow.DBMigrationFlowParams{
		InstallationFlowParams: workflow.InstallationFlowParams{
			DBType:        config.InstallationDBType,
			FileStoreType: config.InstallationFileStoreType,
		},
		DestinationDBID: config.DestinationDB,
	}

	return workflow.NewDBMigrationFlow(params, client, logrus.New()), nil
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
