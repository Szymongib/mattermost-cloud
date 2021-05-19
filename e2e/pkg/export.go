// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package pkg

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"os"
	"os/exec"

	//mmv5 "github.com/mattermost/mattermost-server/v5/model"
	"github.com/pkg/errors"
)

// TODO: remove all but data source?

// SqlSettings is struct copied from Mattermost Server
type SqlSettings struct {
	DriverName                  *string  `restricted:"true"`
	DataSource                  *string  `restricted:"true"`
	DataSourceReplicas          []string `restricted:"true"`
	DataSourceSearchReplicas    []string `restricted:"true"`
	MaxIdleConns                *int     `restricted:"true"`
	ConnMaxLifetimeMilliseconds *int     `restricted:"true"`
	MaxOpenConns                *int     `restricted:"true"`
	Trace                       *bool    `restricted:"true"`
	AtRestEncryptKey            *string  `restricted:"true"`
	QueryTimeout                *int     `restricted:"true"`
}

func GetConnectionString(client *model.Client, clusterInstallationID string) (string, error) {
	// TODO: replce it with get config?
	out, err := client.RunMattermostCLICommandOnClusterInstallation(clusterInstallationID, []string{"config", "show", "--json"})
	if err != nil {
		return "", errors.Wrap(err, "while execing config show")
	}

	settings := struct {
		SqlSettings SqlSettings
	}{}

	err = json.Unmarshal(out, &settings)
	if err != nil {
		return "", errors.Wrap(err, "while unmarshalling sql setting")
	}

	return *settings.SqlSettings.DataSource, nil
}

type BulkLine struct {
	Type string `json:"type"`
}

type ExportStats struct {
	Teams          int
	Channels       int
	Users          int
	Posts          int
	DirectChannels int
	DirectPosts    int
}

func GetBulkExportStats(client *model.Client, kubeClient kubernetes.Interface, clusterInstallationID, installationID string, logger logrus.FieldLogger) (ExportStats, error) {
	fileName := fmt.Sprintf("export-ci-%s.json", clusterInstallationID)

	_, err := client.RunMattermostCLICommandOnClusterInstallation(clusterInstallationID, []string{"export", "bulk", fileName})
	if err != nil {
		return ExportStats{}, errors.Wrap(err, "while execing export csv")
	}

	podClient := kubeClient.CoreV1().Pods(installationID)

	pods, err := podClient.List(context.Background(), metav1.ListOptions{
		LabelSelector: "app=mattermost",
	})
	if err != nil {
		return ExportStats{}, errors.Wrap(err, "while getting pods")
	}

	destination := fileName
	defer func() {
		err := os.Remove(destination)
		if err != nil {
			logger.WithError(err).Warnf("failed to cleanup file %s", destination)
		}
	}()

	// File will be on only one pod
	// if file does not exist kubectl cp exits with 0 code
	// but does not change local file.
	for _, pod := range pods.Items {
		copyFrom := fmt.Sprintf("%s/%s:/mattermost/%s", pod.Namespace, pod.Name, fileName)
		cmd := exec.Command("kubectl", "cp", copyFrom, destination)
		err := cmd.Run()
		if err != nil {
			return ExportStats{}, errors.Wrap(err, "while copying import file from pod")
		}
	}

	file, err := os.Open(destination)
	if err != nil {
		return ExportStats{}, errors.Wrap(err, "failed to open export file")
	}
	defer file.Close()

	exportStats := ExportStats{}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := BulkLine{}
		err := json.Unmarshal(scanner.Bytes(), &line)
		if err != nil {
			return ExportStats{}, errors.Wrap(err, "while unmarshalling export line")
		}

		switch line.Type {
		case "team":
			exportStats.Teams++
		case "channel":
			exportStats.Channels++
		case "post":
			exportStats.Posts++
		case "user":
			exportStats.Users++
		case "direct_channel":
			exportStats.DirectChannels++
		case "direct_post":
			exportStats.DirectPosts++
		}
	}
	if err := scanner.Err(); err != nil {
		return ExportStats{}, errors.Wrap(err, "error scaning export file")
	}

	return exportStats, nil
}