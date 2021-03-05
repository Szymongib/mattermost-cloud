package model

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewBackupMetadataFromReader(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		backupMetadata, err := NewBackupMetadataFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, &BackupMetadata{}, backupMetadata)
	})

	t.Run("invalid", func(t *testing.T) {
		backupMetadata, err := NewBackupMetadataFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, backupMetadata)
	})

	t.Run("valid", func(t *testing.T) {
		backupMetadata, err := NewBackupMetadataFromReader(bytes.NewReader([]byte(
			`{"ID":"metadata-1", "StartAt": 100, "InstallationID":"installation-1"}`,
		)))
		require.NoError(t, err)
		require.Equal(t, &BackupMetadata{ID: "metadata-1", StartAt: 100, InstallationID: "installation-1"}, backupMetadata)
	})
}

func TestEnsureBackupCompatible(t *testing.T) {

	for _, testCase := range []struct {
		description   string
		installation  *Installation
		errorContains string
	}{
		{
			description: "valid installation",
			installation: &Installation{
				State:     InstallationStateHibernating,
				Database:  InstallationDatabaseMultiTenantRDSPostgres,
				Filestore: InstallationFilestoreBifrost,
			},
		},
		{
			description: "not hibernating",
			installation: &Installation{
				State:     InstallationStateStable,
				Database:  InstallationDatabaseMultiTenantRDSPostgres,
				Filestore: InstallationFilestoreBifrost,
			},
			errorContains: "invalid installation state",
		},
		{
			description: "invalid db",
			installation: &Installation{
				State:     InstallationStateHibernating,
				Database:  InstallationDatabaseMultiTenantRDSMySQL,
				Filestore: InstallationFilestoreBifrost,
			},
			errorContains: "invalid installation database",
		},
		{
			description: "invalid file store",
			installation: &Installation{
				State:     InstallationStateHibernating,
				Database:  InstallationDatabaseMultiTenantRDSPostgres,
				Filestore: InstallationFilestoreMinioOperator,
			},
			errorContains: "invalid installation file store",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			err := EnsureBackupCompatible(testCase.installation)
			if testCase.errorContains == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), testCase.errorContains)
			}
		})
	}
}
