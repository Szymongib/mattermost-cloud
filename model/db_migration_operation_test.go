package model

import (
	"bytes"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewDBMigrationOperationFromReader(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		dBMigrationOperation, err := NewDBMigrationOperationFromReader(bytes.NewReader([]byte(
			"",
		)))
		require.NoError(t, err)
		require.Equal(t, &DBMigrationOperation{}, dBMigrationOperation)
	})

	t.Run("invalid", func(t *testing.T) {
		dBMigrationOperation, err := NewDBMigrationOperationFromReader(bytes.NewReader([]byte(
			"{test",
		)))
		require.Error(t, err)
		require.Nil(t, dBMigrationOperation)
	})

	t.Run("valid", func(t *testing.T) {
		dBMigrationOperation, err := NewDBMigrationOperationFromReader(bytes.NewReader([]byte(
			`{"ID":"id", "InstallationID": "installation", "RequestAt": 10, "State": "db-migration-requested"}`,
	)))
		require.NoError(t, err)
		require.Equal(t, &DBMigrationOperation{
			ID:                                   "id",
			InstallationID:                       "installation",
			RequestAt:                            10,
			State:                                DBMigrationStateRequested,
		}, dBMigrationOperation)
	})
}

func TestNewDBMigrationOperationsFromReader(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		dBMigrationOperations, err := NewDBMigrationOperationsFromReader(bytes.NewReader([]byte(
			"",
		)))
		require.NoError(t, err)
		require.Equal(t, []*DBMigrationOperation{}, dBMigrationOperations)
	})

	t.Run("invalid", func(t *testing.T) {
		dBMigrationOperations, err := NewDBMigrationOperationsFromReader(bytes.NewReader([]byte(
			"{test",
		)))
		require.Error(t, err)
		require.Nil(t, dBMigrationOperations)
	})

	t.Run("valid", func(t *testing.T) {
		dBMigrationOperations, err := NewDBMigrationOperationsFromReader(bytes.NewReader([]byte(
			`[
	{"ID":"id", "InstallationID": "installation", "RequestAt": 10, "State": "db-migration-requested"},
	{"ID":"id2", "InstallationID": "installation2", "RequestAt": 20, "State": "db-migration-requested"}
]`,
	)))
		require.NoError(t, err)
		require.Equal(t, []*DBMigrationOperation{
			{
				ID:                                   "id",
				InstallationID:                       "installation",
				RequestAt:                            10,
				State:                                DBMigrationStateRequested,
			},
			{
				ID:                                   "id2",
				InstallationID:                       "installation2",
				RequestAt:                            20,
				State:                                DBMigrationStateRequested,
			},
		}, dBMigrationOperations)
	})
}
