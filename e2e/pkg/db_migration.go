package pkg

import (
	"fmt"
	"github.com/mattermost/mattermost-cloud/model"
	"time"
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
