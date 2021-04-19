package model

import (
	"encoding/json"
	"io"

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
	States                []InstallationDBRestorationState
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
