// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package db_migration

import (
	"encoding/json"
	"github.com/mattermost/mattermost-cloud/e2e/pkg"
	"github.com/mattermost/mattermost-cloud/e2e/workflow"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vrischmann/envconfig"
	"k8s.io/client-go/kubernetes"
)

type TestConfig struct {
	CloudURL                  string `envconfig:"default=http://localhost:8075"`
	DestinationDB             string `envconfig:"default=rds-cluster-multitenant-050365fcbb1170e4b-migration"`
	InstallationDBType        string `envconfig:"default=aws-multitenant-rds-postgres"`
	InstallationFileStoreType string `envconfig:"default=bifrost"`
	Cleanup                   bool   `envconfig:"default=false"`
}

type DBMigrationTest struct {
	Logger   logrus.FieldLogger
	Flow     *workflow.DBMigrationFlow
	Workflow *workflow.Workflow
	Cleanup  bool
}

func SetupDBMigrationCommitTest() (*DBMigrationTest, error) {
	logger := logrus.WithField("test", "db-migration-commit")

	config, err := readConfig(logger)
	if err != nil {
		return nil, err
	}

	flow, err := setupDBMigrationTestFlow(config, logger)
	if err != nil {
		return nil, err
	}
	work := commitDBMigrationWorkflow(flow)

	return &DBMigrationTest{
		Logger:   logger,
		Flow:     flow,
		Workflow: work,
		Cleanup:  config.Cleanup,
	}, nil
}

func SetupDBMigrationRollbackTest() (*DBMigrationTest, error) {
	logger := logrus.WithField("test", "db-migration-rollback")

	config, err := readConfig(logger)
	if err != nil {
		return nil, err
	}

	flow, err := setupDBMigrationTestFlow(config, logger)
	if err != nil {
		return nil, err
	}
	work := rollbackDBMigrationWorkflow(flow)

	return &DBMigrationTest{
		Logger:   logger,
		Flow:     flow,
		Workflow: work,
		Cleanup:  config.Cleanup,
	}, nil
}

func readConfig(logger logrus.FieldLogger) (TestConfig, error) {
	var config TestConfig
	err := envconfig.Init(&config)
	if err != nil {
		return TestConfig{}, errors.Wrap(err, "unable to read environment configuration")
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return TestConfig{}, errors.Wrap(err, "failed to marshal config to json")
	}

	logger.Infof("Test Config: %s", configJSON)

	return config, nil
}

func setupDBMigrationTestFlow(config TestConfig, logger logrus.FieldLogger) (*workflow.DBMigrationFlow, error) {
	client := model.NewClient(config.CloudURL)

	params := workflow.DBMigrationFlowParams{
		InstallationFlowParams: workflow.InstallationFlowParams{
			DBType:        config.InstallationDBType,
			FileStoreType: config.InstallationFileStoreType,
		},
		DestinationDBID: config.DestinationDB,
	}

	kubeClient, err := getKubeClient()
	if err != nil {
		return nil, err
	}

	return workflow.NewDBMigrationFlow(params, client, kubeClient, logger), nil
}

func getKubeClient() (kubernetes.Interface, error) {
	k8sConfig, err := pkg.GetK8sConfig()
	if err != nil {
		return nil, errors.Wrap(err, "while getting kubeconfig")
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, errors.Wrap(err, "while creating kube client")
	}

	return clientset, nil
}

func (w *DBMigrationTest) Run() error {
	err := workflow.RunWorkflow(w.Workflow, w.Logger)
	if err != nil {
		return errors.Wrap(err, "error running workflow")
	}
	return nil
}
