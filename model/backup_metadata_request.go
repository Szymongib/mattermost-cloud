package model

import (
	"net/url"
	"strconv"
)

// GetBackupsMetadataRequest describes the parameters to request a list of installation backups.
type GetBackupsMetadataRequest struct {
	ClusterInstallationID      string
	State string
	Page           int
	PerPage        int
	IncludeDeleted bool
}

// ApplyToURL modifies the given url to include query string parameters for the request.
func (request *GetBackupsMetadataRequest) ApplyToURL(u *url.URL) {
	q := u.Query()
	q.Add("cluster_installation", request.ClusterInstallationID)
	q.Add("state", request.State)
	q.Add("page", strconv.Itoa(request.Page))
	q.Add("per_page", strconv.Itoa(request.PerPage))
	if request.IncludeDeleted {
		q.Add("include_deleted", "true")
	}
	u.RawQuery = q.Encode()
}
