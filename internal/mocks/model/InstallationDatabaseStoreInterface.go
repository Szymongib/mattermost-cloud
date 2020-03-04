// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import (
	model "github.com/mattermost/mattermost-cloud/model"
	mock "github.com/stretchr/testify/mock"
)

// InstallationDatabaseStoreInterface is an autogenerated mock type for the InstallationDatabaseStoreInterface type
type InstallationDatabaseStoreInterface struct {
	mock.Mock
}

// GetClusterInstallations provides a mock function with given fields: filter
func (_m *InstallationDatabaseStoreInterface) GetClusterInstallations(filter *model.ClusterInstallationFilter) ([]*model.ClusterInstallation, error) {
	ret := _m.Called(filter)

	var r0 []*model.ClusterInstallation
	if rf, ok := ret.Get(0).(func(*model.ClusterInstallationFilter) []*model.ClusterInstallation); ok {
		r0 = rf(filter)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]*model.ClusterInstallation)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(*model.ClusterInstallationFilter) error); ok {
		r1 = rf(filter)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}