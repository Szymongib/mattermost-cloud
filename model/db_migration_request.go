package model

import (
	"encoding/json"
	"io"
	"net/url"

	"github.com/pkg/errors"
)

// TODO: whole validation logic
type DBMigrationRequest struct {
	InstallationID string

	DestinationDatabase string

	DestinationMultiTenant *MultiTenantDBMigrationData
}

type InstallationDBMigrationFilter struct {
	Paging
	IDs                   []string
	InstallationID        string
	ClusterInstallationID string
	States                []DBMigrationOperationState
}

// TODO: test - generate?
// NewInstallationDBRestorationRequestFromReader will create a InstallationDBRestorationRequest from an
// io.Reader with JSON data.
func NewDBMigrationRequestFromReader(reader io.Reader) (*DBMigrationRequest, error) {
	var migrationRequest DBMigrationRequest
	err := json.NewDecoder(reader).Decode(&migrationRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode db migration request")
	}

	return &migrationRequest, nil
}

// GetDBMigrationOperationsRequest describes the parameters to request
// a list of installation db migration operations.
type GetDBMigrationOperationsRequest struct {
	Paging
	InstallationID        string
	ClusterInstallationID string
	State                 string
}

// ApplyToURL modifies the given url to include query string parameters for the request.
func (request *GetDBMigrationOperationsRequest) ApplyToURL(u *url.URL) {
	q := u.Query()
	q.Add("installation", request.InstallationID)
	q.Add("cluster_installation", request.ClusterInstallationID)
	q.Add("state", request.State)
	request.Paging.AddToQuery(q)

	u.RawQuery = q.Encode()
}
