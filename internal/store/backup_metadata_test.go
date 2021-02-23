package store

import (
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestIsBackupRunning(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation := &model.Installation{
		State:     model.InstallationStateStable,
	}

	err := sqlStore.CreateInstallation(installation, nil)
	require.NoError(t, err)

	running, err := sqlStore.IsBackupRunning(installation.ID)
	require.NoError(t, err)
	require.False(t, running)

	metadata := &model.BackupMetadata{
		InstallationId: installation.ID,
		State:          model.BackupStateBackupRequested,
	}

	err = sqlStore.CreateBackupMetadata(metadata)
	require.NoError(t, err)

	running, err = sqlStore.IsBackupRunning(installation.ID)
	require.NoError(t, err)
	require.True(t, running)
}

func TestCreateBackupMetadata(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	installation := &model.Installation{
		State:     model.InstallationStateStable,
	}

	err := sqlStore.CreateInstallation(installation, nil)
	require.NoError(t, err)

	metadata := &model.BackupMetadata{
		InstallationId: installation.ID,
		State:          model.BackupStateBackupRequested,
	}

	err = sqlStore.CreateBackupMetadata(metadata)
	require.NoError(t, err)
	assert.NotEmpty(t, metadata.ID)

	t.Run("fail to create backup metadata for installation when other is requested", func(t *testing.T) {
		newMetadata := &model.BackupMetadata{
			InstallationId: installation.ID,
			State:          model.BackupStateBackupRequested,
		}

		err = sqlStore.CreateBackupMetadata(newMetadata)
		require.Error(t, err)
	})
}
