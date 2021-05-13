// Copyright (c) YEAR-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package pkg

import (
	"fmt"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"net/http"
	"time"
)

func CreateHAInstallation(client *model.Client, db, filestore string, group string) (*model.InstallationDTO, error) {
	installationDNS := GetDNS()

	installationReq := model.CreateInstallationRequest{
		OwnerID:   "e2e-test", // TODO: allow customization
		GroupID:   group,
		Version:   "stable",
		Image:     "mattermost/mattermost-enterprise-edition",
		DNS:       installationDNS,
		Size:      "1000users",
		Affinity:  "multitenant",
		Database:  db,
		Filestore: filestore,
	}

	installation, err := client.CreateInstallation(&installationReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create installation")
	}

	return installation, nil
}

func WaitForInstallation(client *model.Client, dns string) error {
	err := WaitForFunc(20*time.Minute, 10*time.Second, func() (bool, error) {
		resp, err := http.Get(PingURL(dns))
		if err != nil {
			fmt.Println("Error while pining installation: ", err.Error())
			return false, nil
		}
		return resp.StatusCode == http.StatusOK, nil
	})
	return err
}

func WaitForHibernation(client *model.Client, installationID string) error {
	err := WaitForFunc(5*time.Minute, 10*time.Second, func() (bool, error) {
		installation, err := client.GetInstallation(installationID, &model.GetInstallationRequest{})
		if err != nil {
			// TODO: allow 3 fails
			return false, errors.Wrap(err, "while waiting for hibernation")
		}

		if installation.State == model.InstallationStateHibernating {
			return true, nil
		}
		fmt.Println("Installation not hibernated: ", installation.State, "  ", installation.ID)
		return false, nil
	})
	return err
}

func WaitForStable(client *model.Client, installationID string) error {
	err := WaitForFunc(5*time.Minute, 10*time.Second, func() (bool, error) {
		installation, err := client.GetInstallation(installationID, &model.GetInstallationRequest{})
		if err != nil {
			// TODO: allow 3 fails
			return false, errors.Wrap(err, "while waiting for stable")
		}

		if installation.State == model.InstallationStateStable {
			return true, nil
		}
		fmt.Println("Installation not stable: ", installation.State, "  ", installation.ID)
		return false, nil
	})
	return err
}

func PingURL(dns string) string {
	return fmt.Sprintf("https://%s/api/v4/system/ping", dns)
}
