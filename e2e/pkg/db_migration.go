// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package pkg

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"time"

	"github.com/mattermost/mattermost-cloud/model"
)

func WaitForDBMigrationToFinish(client *model.Client, opID string, log logrus.FieldLogger) error {
	errCount := 0
	err := WaitForFunc(16*time.Minute, 10*time.Second, func() (bool, error) {
		operation, err := client.GetInstallationDBMigrationOperation(opID)
		if err != nil {
			log.WithError(err).Error("Error while waiting for db migration")
			errCount++
			if errCount > 3 {
				return false, err
			}
			return false, nil
		}

		if operation.State == model.InstallationDBMigrationStateSucceeded {
			return true, nil
		}
		if operation.State == model.InstallationDBMigrationStateFailed {
			return false, fmt.Errorf("db migration operation %q failed", operation.ID)
		}
		return false, nil
	})
	return err
}

func WaitForDBMigrationRollbackToFinish(client *model.Client, opID string, log logrus.FieldLogger) error {
	errCount := 0
	err := WaitForFunc(16*time.Minute, 10*time.Second, func() (bool, error) {
		operation, err := client.GetInstallationDBMigrationOperation(opID)
		if err != nil {
			log.WithError(err).Error("Error while waiting for db migration rollback")
			errCount++
			if errCount > 3 {
				return false, err
			}
			return false, nil
		}

		if operation.State == model.InstallationDBMigrationStateRollbackFinished {
			return true, nil
		}
		return false, nil
	})
	return err
}
