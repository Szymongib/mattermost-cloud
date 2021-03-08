package model

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/url"
	"testing"
)

func TestGetBackupsMetadataRequest_ApplyToURL(t *testing.T) {
	req := &GetBackupsMetadataRequest{
		ClusterInstallationID: "my-ci",
		State:                 "failed",
		Page:                  1,
		PerPage:               5,
		IncludeDeleted:        true,
	}

	u, err := url.Parse("https://provisioner/backups")
	require.NoError(t, err)

	req.ApplyToURL(u)

	assert.Equal(t, req.ClusterInstallationID, u.Query().Get("cluster_installation"))
	assert.Equal(t, req.State, u.Query().Get("state"))
	assert.Equal(t, "1", u.Query().Get("page"))
	assert.Equal(t,  "5", u.Query().Get("per_page"))
	assert.Equal(t, "true", u.Query().Get("include_deleted"))
}
