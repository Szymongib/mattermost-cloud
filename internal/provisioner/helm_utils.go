// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/tools/helm"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// helmDeployment deploys Helm charts.
type helmDeployment struct {
	chartDeploymentName string
	chartName           string
	namespace           string
	setArgument         string
	valuesPath          string
	desiredVersion      string

	cluster         *model.Cluster
	kopsProvisioner *KopsProvisioner
	kops            *kops.Cmd
	logger          log.FieldLogger
}

func (d *helmDeployment) Update() error {
	logger := d.logger.WithField("helm-update", d.chartName)

	logger.Infof("Refreshing helm chart %s -- may trigger service upgrade", d.chartName)
	err := upgradeHelmChart(*d, d.kops.GetKubeConfigPath(), logger)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("got an error trying to upgrade the helm chart %s", d.chartName))
	}
	return nil
}

func (d *helmDeployment) Delete() error {
	logger := d.logger.WithField("helm-delete", d.chartDeploymentName)

	logger.Infof("Deleting helm chart %s", d.chartDeploymentName)
	err := deleteHelmChart(*d, d.kops.GetKubeConfigPath(), logger)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("got an error trying to delete the helm chart %s", d.chartDeploymentName))
	}
	return nil
}

// waitForHelmRunning is used to check when Helm is ready to install charts.
func waitForHelmRunning(ctx context.Context, configPath string) error {
	for {
		cmd := exec.Command("helm", "ls", "--kubeconfig", configPath)
		var out bytes.Buffer
		cmd.Stderr = &out
		cmd.Run()
		if out.String() == "" {
			return nil
		}
		select {
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "timed out waiting for helm to become ready")
		case <-time.After(5 * time.Second):
		}
	}
}

// helmRepoAdd adds new helm repos
func helmRepoAdd(repoName, repoURL string, logger log.FieldLogger) error {
	logger.Infof("Adding helm repo %s", repoName)
	arguments := []string{
		"repo",
		"add",
		repoName,
		repoURL,
	}

	helmClient, err := helm.NewV3(logger)
	if err != nil {
		return errors.Wrap(err, "unable to create helm wrapper")
	}
	defer helmClient.Close()

	err = helmClient.RunGenericCommand(arguments...)
	if err != nil {
		return errors.Wrapf(err, "unable to add repo %s", repoName)
	}

	return helmRepoUpdate(logger)
}

// helmRepoUpdate updates the helm repos to get latest available charts
func helmRepoUpdate(logger log.FieldLogger) error {
	arguments := []string{
		"repo",
		"update",
	}

	helmClient, err := helm.NewV3(logger)
	if err != nil {
		return errors.Wrap(err, "unable to create helm wrapper")
	}
	defer helmClient.Close()

	err = helmClient.RunGenericCommand(arguments...)
	if err != nil {
		return errors.Wrap(err, "unable to update helm repos")
	}

	return nil
}

// upgradeHelmChart is used to upgrade Helm deployments.
func upgradeHelmChart(chart helmDeployment, configPath string, logger log.FieldLogger) error {
	arguments := []string{
		"--debug",
		"upgrade",
		chart.chartDeploymentName,
		chart.chartName,
		"--kubeconfig", configPath,
		"-f", chart.valuesPath,
		"--namespace", chart.namespace,
		"--install",
		"--create-namespace",
		"--wait",
	}
	if chart.setArgument != "" {
		arguments = append(arguments, "--set", chart.setArgument)
	}
	if chart.desiredVersion != "" {
		arguments = append(arguments, "--version", chart.desiredVersion)
	}

	helmClient, err := helm.NewV3(logger)
	if err != nil {
		return errors.Wrap(err, "unable to create helm wrapper")
	}
	defer helmClient.Close()

	err = helmClient.RunGenericCommand(arguments...)
	if err != nil {
		return errors.Wrapf(err, "unable to upgrade helm chart %s", chart.chartName)
	}

	return nil
}

// deleteHelmChart is used to delete Helm charts.
func deleteHelmChart(chart helmDeployment, configPath string, logger log.FieldLogger) error {
	arguments := []string{
		"--debug",
		"delete",
		"--kubeconfig", configPath,
		"--purge", chart.chartDeploymentName,
	}

	helmClient, err := helm.New(logger)
	if err != nil {
		return errors.Wrap(err, "unable to create helm wrapper")
	}
	defer helmClient.Close()

	err = helmClient.RunGenericCommand(arguments...)
	if err != nil {
		return errors.Wrapf(err, "unable to delete helm chart %s", chart.chartDeploymentName)
	}

	return nil
}

type helmReleaseJSON struct {
	Name       string `json:"name"`
	Revision   string `json:"revision"`
	Updated    string `json:"updated"`
	Status     string `json:"status"`
	Chart      string `json:"chart"`
	AppVersion string `json:"appVersion"`
	Namespace  string `json:"namespace"`
}

// HelmListOutput is a struct for holding the unmarshaled
// representation of the output from helm list --output json
type HelmListOutput []helmReleaseJSON

func (l HelmListOutput) containsRelease(name string) bool {
	for _, rel := range l {
		if rel.Name == name {
			return true
		}
	}
	return false
}

func (l HelmListOutput) asSlice() []helmReleaseJSON {
	return l
}

func (l HelmListOutput) asListOutput() *HelmListOutput {
	return &l
}

func (d *helmDeployment) List() (*HelmListOutput, error) {
	arguments := []string{
		"list",
		"--kubeconfig", d.kops.GetKubeConfigPath(),
		"--output", "json",
		"--all-namespaces",
	}

	logger := d.logger.WithFields(log.Fields{
		"cmd": "helm3",
	})

	helmClient, err := helm.NewV3(logger)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create helm wrapper")
	}
	defer helmClient.Close()

	rawOutput, err := helmClient.RunCommandRaw(arguments...)
	if err != nil {
		if len(rawOutput) > 0 {
			logger.Debugf("Helm output was:\n%s\n", string(rawOutput))
		}
		return nil, errors.Wrap(err, "while listing Helm Releases")
	}

	var helmList HelmListOutput
	err = json.Unmarshal(rawOutput, &helmList)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal JSON output from helm list")
	}

	return helmList.asListOutput(), nil

}

func (d *helmDeployment) Version() (string, error) {
	output, err := d.List()
	if err != nil {
		return "", errors.Wrap(err, "while getting Helm Deployment version")
	}

	for _, release := range output.asSlice() {
		if release.Name == d.chartDeploymentName {
			return release.Chart, nil
		}
	}

	return "", errors.Errorf("unable to get version for chart %s", d.chartDeploymentName)
}

// TryMigrate migrates Helm release from version 2 to 3 if it exists.
func (d *helmDeployment) TryMigrate() error {
	logger := d.logger.WithField("operation", "migrate")

	hasTiller, err := tillerExists(logger, d.kops.GetKubeConfigPath())
	if err != nil {
		return errors.Wrap(err, "failed to check if Tiller exists on cluster")
	}
	if !hasTiller {
		logger.Debugf("Tiller does not exist on cluster, skipping migration")
		return nil
	}

	listV2, err := d.ListV2()
	if err != nil {
		return errors.Wrap(err, "failed to list Helm 2 releases")
	}

	listV3, err := d.List()
	if err != nil {
		return errors.Wrap(err, "failed to list Helm 3 releases")
	}

	release := d.chartDeploymentName
	if listV2.containsRelease(release) && !listV3.containsRelease(release) {
		logger.Debugf("Release '%s' found with Helm 2 and not found with Helm 3. Starting the migration...", release)
		return MigrateRelease(logger, d.kops.GetKubeConfigPath(), release)
	}

	logger.Debugf("Skipping migration of release '%s'", release)
	return nil
}

func tillerExists(logger log.FieldLogger, kubeConfigPath string) (bool, error) {
	helm2, err := helm.New(logger)
	if err != nil {
		return false, errors.Wrap(err, "failed to initialize Helm 2 client")
	}

	args := []string{
		"version",
		"--kubeconfig", kubeConfigPath,
	}

	_, err = helm2.RunCommandRaw(args...)
	if err == nil {
		return true, nil
	}

	if ee, ok := err.(*exec.ExitError); ok {
		if strings.Contains(string(ee.Stderr), "Error: could not find tiller") {
			return false, nil
		}
	}

	return false, err
}

func helm2Cleanup(logger log.FieldLogger, kubeConfigPath string) error {
	log := logger.WithField("operation", "cleanup")

	hasTiller, err := tillerExists(log, kubeConfigPath)
	if err != nil {
		return errors.Wrap(err, "failed to check if Tiller exists on cluster")
	}
	if !hasTiller {
		logger.Debugf("Tiller does not exist on cluster, skipping cleanup")
		return nil
	}

	return CleanupAll(log, kubeConfigPath)
}
