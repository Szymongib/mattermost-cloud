// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package pkg

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"time"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

func CreateHAInstallation(client *model.Client, db, filestore string, group string) (*model.InstallationDTO, error) {
	installationDNS := GetDNS()

	// TODO: allow customization
	installationReq := model.CreateInstallationRequest{
		OwnerID:   "e2e-test",
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

func WaitForInstallation(dns string, log logrus.FieldLogger) error {
	err := WaitForFunc(20*time.Minute, 10*time.Second, func() (bool, error) {
		resp, err := http.Get(PingURL(dns))
		if err != nil {
			log.WithError(err).Error("Error while pining installation")
			return false, nil
		}
		return resp.StatusCode == http.StatusOK, nil
	})
	return err
}

func WaitForHibernation(client *model.Client, installationID string, log logrus.FieldLogger) error {
	err := WaitForFunc(5*time.Minute, 10*time.Second, func() (bool, error) {
		installation, err := client.GetInstallation(installationID, &model.GetInstallationRequest{})
		if err != nil {
			return false, errors.Wrap(err, "while waiting for hibernation")
		}

		if installation.State == model.InstallationStateHibernating {
			return true, nil
		}
		log.Infof("Installation %s not hibernated: %s", installationID, installation.State)
		return false, nil
	})
	return err
}

func WaitForStable(client *model.Client, installationID string, log logrus.FieldLogger) error {
	err := WaitForFunc(5*time.Minute, 10*time.Second, func() (bool, error) {
		installation, err := client.GetInstallation(installationID, &model.GetInstallationRequest{})
		if err != nil {
			return false, errors.Wrap(err, "while waiting for stable")
		}

		if installation.State == model.InstallationStateStable {
			return true, nil
		}
		log.Infof("Installation %s not stable: %s", installationID, installation.State)
		return false, nil
	})
	return err
}

func PingURL(dns string) string {
	return fmt.Sprintf("https://%s/api/v4/system/ping", dns)
}
