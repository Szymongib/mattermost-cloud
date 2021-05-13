// Copyright (c) YEAR-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//
package workflow

import (
	"context"
	"fmt"
	"github.com/mattermost/mattermost-cloud/e2e/pkg"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func NewInstallationFlow(params InstallationFlowParams, client *model.Client, logger logrus.FieldLogger) *InstallationFlow {
	return &InstallationFlow{
		client: client,
		logger: logger.WithField("flow", "installation"),
		Params: params,
		Meta:   InstallationFlowMeta{},
	}
}

type InstallationFlow struct {
	client *model.Client
	logger logrus.FieldLogger

	Params InstallationFlowParams
	Meta InstallationFlowMeta
}

type InstallationFlowParams struct {
	DBType          string
	FileStoreType   string
}

type InstallationFlowMeta struct {
	InstallationID        string
	InstallationDNS       string
	ClusterInstallationID string
	ConnectionString 	  string
}

func (w *InstallationFlow) CreateInstallation(ctx context.Context) error {
	if w.Meta.InstallationID == "" {
		installation, err := pkg.CreateHAInstallation(w.client, w.Params.DBType, w.Params.FileStoreType, "")
		if err != nil {
			return errors.Wrap(err, "while creating installation")
		}
		fmt.Println("Installation created: ", installation.ID)
		w.Meta.InstallationID = installation.ID
		w.Meta.InstallationDNS = installation.DNS
	}

	err := pkg.WaitForStable(w.client, w.Meta.InstallationID)
	if err != nil {
		return errors.Wrap(err, "while waiting for installation creation")
	}

	err = pkg.WaitForInstallation(w.client, w.Meta.InstallationDNS)
	if err != nil {
		return errors.Wrap(err, "while waiting for installation DNS")
	}

	return nil
}

func (w *InstallationFlow) GetCI(ctx context.Context) error {
	ci, err := w.client.GetClusterInstallations(&model.GetClusterInstallationsRequest{InstallationID: w.Meta.InstallationID, Paging: model.AllPagesNotDeleted()})
	if err != nil {
		return errors.Wrap(err, "while getting CI")
	}
	w.Meta.ClusterInstallationID = ci[0].ID

	return nil
}

func (w *InstallationFlow) GetConnectionStrAndExport(ctx context.Context) error {
	connectionString, err := pkg.GetConnectionString(w.client, w.Meta.ClusterInstallationID)
	if err != nil {
		return errors.Wrap(err, "while getting connection str")
	}
	w.Meta.ConnectionString = connectionString

	//dataExportOriginal1, err := pkg.ExportCSV(w.client, w.Meta.CI)
	//if err != nil {
	//	return errors.Wrap(err, "while getting CSV export")
	//}
	//w.Meta.OriginalCSVExport = dataExportOriginal1
	//fmt.Println("Original CSV export: ", dataExportOriginal1)

	return nil
}


func (w *InstallationFlow) PopulateSampleData(ctx context.Context) error {
	out, err := w.client.RunMattermostCLICommandOnClusterInstallation(w.Meta.ClusterInstallationID, []string{"sampledata", "--teams", "4", "--channels-per-team", "15"})
	if err != nil {
		return errors.Wrap(err, "while populating sample data for CI")
	}
	fmt.Println("Sample data generated: ", string(out))

	return nil
}

func (w *InstallationFlow) HibernateInstallation(ctx context.Context) error {
	installation, err := w.client.GetInstallation(w.Meta.InstallationID, &model.GetInstallationRequest{})
	if err != nil {
		return errors.Wrap(err, "while getting installation to hibernate")
	}
	if installation.State == model.InstallationStateHibernating {
		fmt.Println("installation already hibernating")
		return nil
	}

	installation, err = w.client.HibernateInstallation(w.Meta.InstallationID)
	if err != nil {
		return errors.Wrap(err, "while hibernating installation")
	}

	err = pkg.WaitForHibernation(w.client, w.Meta.InstallationID)
	if err != nil {
		return errors.Wrap(err, "while waiting for installation to hibernate")
	}

	return nil
}

func (w *InstallationFlow) WakeUpInstallation(ctx context.Context) error {
	installation, err := w.client.GetInstallation(w.Meta.InstallationID, &model.GetInstallationRequest{})
	if err != nil {
		return errors.Wrap(err, "while getting installation to wake up")
	}
	if installation.State == model.InstallationStateStable {
		fmt.Println("installation already woken up")
		return nil
	}

	if installation.State == model.InstallationStateHibernating {
		installation, err = w.client.WakeupInstallation(w.Meta.InstallationID)
		if err != nil {
			return errors.Wrap(err, "while waking up installation")
		}
	}

	if installation.State != model.InstallationStateWakeUpRequested &&
		installation.State != model.InstallationStateUpdateInProgress {
		return errors.Errorf("installation is in unexpected state: %s", installation.State)
	}

	err = pkg.WaitForStable(w.client, w.Meta.InstallationID)
	if err != nil {
		return errors.Wrap(err, "while waiting for installation to wake up")
	}

	return nil
}
