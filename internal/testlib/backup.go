package testlib

import (
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/require"
	"testing"
)

func CreateBackupCompatibleInstallation(t *testing.T, sqlStore *store.SQLStore) *model.Installation {
	installation := &model.Installation{
		Database:  model.InstallationDatabaseMultiTenantRDSPostgres,
		Filestore: model.InstallationFilestoreBifrost,
		State:     model.InstallationStateHibernating,
	}
	err := sqlStore.CreateInstallation(installation, nil)
	require.NoError(t, err)
	return installation
}

