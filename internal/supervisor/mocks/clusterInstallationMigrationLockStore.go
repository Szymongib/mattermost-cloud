// Code generated by mockery v1.0.0. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// clusterInstallationMigrationLockStore is an autogenerated mock type for the clusterInstallationMigrationLockStore type
type clusterInstallationMigrationLockStore struct {
	mock.Mock
}

// LockClusterInstallationMigration provides a mock function with given fields: migrationID, lockerID
func (_m *clusterInstallationMigrationLockStore) LockClusterInstallationMigration(migrationID string, lockerID string) (bool, error) {
	ret := _m.Called(migrationID, lockerID)

	var r0 bool
	if rf, ok := ret.Get(0).(func(string, string) bool); ok {
		r0 = rf(migrationID, lockerID)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(migrationID, lockerID)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// UnlockClusterInstallationMigration provides a mock function with given fields: migrationID, lockerID, force
func (_m *clusterInstallationMigrationLockStore) UnlockClusterInstallationMigration(migrationID string, lockerID string, force bool) (bool, error) {
	ret := _m.Called(migrationID, lockerID, force)

	var r0 bool
	if rf, ok := ret.Get(0).(func(string, string, bool) bool); ok {
		r0 = rf(migrationID, lockerID, force)
	} else {
		r0 = ret.Get(0).(bool)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string, bool) error); ok {
		r1 = rf(migrationID, lockerID, force)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}