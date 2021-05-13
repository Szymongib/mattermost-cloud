// Copyright (c) YEAR-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package pkg

import (
	"encoding/json"

	"github.com/mattermost/mattermost-cloud/model"

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

func ExportCSV(client *model.Client, clusterInstallationID string) (string, error) {
	out, err := client.RunMattermostCLICommandOnClusterInstallation(clusterInstallationID, []string{"export", "csv", "--exportFrom", "0"})
	if err != nil {
		return "", errors.Wrap(err, "while execing export csv")
	}

	return string(out), nil
}
