package model

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewInstallationDBRestorationOperationFromReader(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		installationDBRestorationOperation, err := NewInstallationDBRestorationOperationFromReader(bytes.NewReader([]byte(
			"",
		)))
		require.NoError(t, err)
		require.Equal(t, &InstallationDBRestorationOperation{}, installationDBRestorationOperation)
	})

	t.Run("invalid", func(t *testing.T) {
		installationDBRestorationOperation, err := NewInstallationDBRestorationOperationFromReader(bytes.NewReader([]byte(
			"{test",
		)))
		require.Error(t, err)
		require.Nil(t, installationDBRestorationOperation)
	})

	t.Run("valid", func(t *testing.T) {
		installationDBRestorationOperation, err := NewInstallationDBRestorationOperationFromReader(bytes.NewReader([]byte(
			`{"ID":"id", "InstallationID":"Installation", "BackupID": "backup", "RequestAt": 10}`,
	)))
		require.NoError(t, err)
		require.Equal(t, &InstallationDBRestorationOperation{
			ID:                      "id",
			InstallationID:          "Installation",
			BackupID:                "backup",
			RequestAt:               10,
		}, installationDBRestorationOperation)
	})
}

func TestNewInstallationDBRestorationOperationsFromReader(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		installationDBRestorationOperations, err := NewInstallationDBRestorationOperationsFromReader(bytes.NewReader([]byte(
			"",
		)))
		require.NoError(t, err)
		require.Equal(t, []*InstallationDBRestorationOperation{}, installationDBRestorationOperations)
	})

	t.Run("invalid", func(t *testing.T) {
		installationDBRestorationOperations, err := NewInstallationDBRestorationOperationsFromReader(bytes.NewReader([]byte(
			"{test",
		)))
		require.Error(t, err)
		require.Nil(t, installationDBRestorationOperations)
	})

	t.Run("valid", func(t *testing.T) {
		installationDBRestorationOperations, err := NewInstallationDBRestorationOperationsFromReader(bytes.NewReader([]byte(
			`[
	{"ID":"id", "InstallationID":"Installation", "BackupID": "backup", "RequestAt": 10},
	{"ID":"id2", "InstallationID":"Installation2", "BackupID": "backup2", "RequestAt": 20}
]`,
	)))
		require.NoError(t, err)
		require.Equal(t, []*InstallationDBRestorationOperation{
			{
				ID:                      "id",
				InstallationID:          "Installation",
				BackupID:                "backup",
				RequestAt:               10,
			},
			{
				ID:                      "id2",
				InstallationID:          "Installation2",
				BackupID:                "backup2",
				RequestAt:               20,
			},
		}, installationDBRestorationOperations)
	})
}
