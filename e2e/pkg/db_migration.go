// Copyright (c) YEAR-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package pkg

import (
	"fmt"
	"time"

	"github.com/mattermost/mattermost-cloud/model"
)

func WaitForDBMigrationToFinish(client *model.Client, opID string) error {
	errCount := 0
	err := WaitForFunc(16*time.Minute, 10*time.Second, func() (bool, error) {
		operation, err := client.GetInstallationDBMigrationOperation(opID)
		if err != nil {
			fmt.Println("Error while waiting for db migration: ", err.Error())
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

func WaitForDBMigrationRollbackToFinish(client *model.Client, opID string) error {
	errCount := 0
	err := WaitForFunc(16*time.Minute, 10*time.Second, func() (bool, error) {
		operation, err := client.GetInstallationDBMigrationOperation(opID)
		if err != nil {
			fmt.Println("Error while waiting for db migration: ", err.Error())
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
