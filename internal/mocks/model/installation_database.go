// Copyright (c) Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//
// Code generated by MockGen. DO NOT EDIT.
// Source: ../../../model/installation_database.go

// Package mocks is a generated GoMock package.
package mocks

import (
	gomock "github.com/golang/mock/gomock"
	model "github.com/mattermost/mattermost-cloud/model"
	logrus "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	reflect "reflect"
)

// MockDatabase is a mock of Database interface
type MockDatabase struct {
	ctrl     *gomock.Controller
	recorder *MockDatabaseMockRecorder
}

// MockDatabaseMockRecorder is the mock recorder for MockDatabase
type MockDatabaseMockRecorder struct {
	mock *MockDatabase
}

// NewMockDatabase creates a new mock instance
func NewMockDatabase(ctrl *gomock.Controller) *MockDatabase {
	mock := &MockDatabase{ctrl: ctrl}
	mock.recorder = &MockDatabaseMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockDatabase) EXPECT() *MockDatabaseMockRecorder {
	return m.recorder
}

// Provision mocks base method
func (m *MockDatabase) Provision(store model.InstallationDatabaseStoreInterface, logger logrus.FieldLogger) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Provision", store, logger)
	ret0, _ := ret[0].(error)
	return ret0
}

// Provision indicates an expected call of Provision
func (mr *MockDatabaseMockRecorder) Provision(store, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Provision", reflect.TypeOf((*MockDatabase)(nil).Provision), store, logger)
}

// Teardown mocks base method
func (m *MockDatabase) Teardown(store model.InstallationDatabaseStoreInterface, keepData bool, logger logrus.FieldLogger) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Teardown", store, keepData, logger)
	ret0, _ := ret[0].(error)
	return ret0
}

// Teardown indicates an expected call of Teardown
func (mr *MockDatabaseMockRecorder) Teardown(store, keepData, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Teardown", reflect.TypeOf((*MockDatabase)(nil).Teardown), store, keepData, logger)
}

// Snapshot mocks base method
func (m *MockDatabase) Snapshot(store model.InstallationDatabaseStoreInterface, logger logrus.FieldLogger) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Snapshot", store, logger)
	ret0, _ := ret[0].(error)
	return ret0
}

// Snapshot indicates an expected call of Snapshot
func (mr *MockDatabaseMockRecorder) Snapshot(store, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Snapshot", reflect.TypeOf((*MockDatabase)(nil).Snapshot), store, logger)
}

// GenerateDatabaseSecret mocks base method
func (m *MockDatabase) GenerateDatabaseSecret(store model.InstallationDatabaseStoreInterface, logger logrus.FieldLogger) (*v1.Secret, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GenerateDatabaseSecret", store, logger)
	ret0, _ := ret[0].(*v1.Secret)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GenerateDatabaseSecret indicates an expected call of GenerateDatabaseSecret
func (mr *MockDatabaseMockRecorder) GenerateDatabaseSecret(store, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GenerateDatabaseSecret", reflect.TypeOf((*MockDatabase)(nil).GenerateDatabaseSecret), store, logger)
}

// RefreshResourceMetadata mocks base method
func (m *MockDatabase) RefreshResourceMetadata(store model.InstallationDatabaseStoreInterface, logger logrus.FieldLogger) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RefreshResourceMetadata", store, logger)
	ret0, _ := ret[0].(error)
	return ret0
}

// RefreshResourceMetadata indicates an expected call of RefreshResourceMetadata
func (mr *MockDatabaseMockRecorder) RefreshResourceMetadata(store, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RefreshResourceMetadata", reflect.TypeOf((*MockDatabase)(nil).RefreshResourceMetadata), store, logger)
}

// MigrateOut mocks base method
func (m *MockDatabase) MigrateOut(store model.InstallationDatabaseStoreInterface, dbMigration *model.DBMigrationOperation, logger logrus.FieldLogger) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MigrateOut", store, dbMigration, logger)
	ret0, _ := ret[0].(error)
	return ret0
}

// MigrateOut indicates an expected call of MigrateOut
func (mr *MockDatabaseMockRecorder) MigrateOut(store, dbMigration, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MigrateOut", reflect.TypeOf((*MockDatabase)(nil).MigrateOut), store, dbMigration, logger)
}

// MigrateTo mocks base method
func (m *MockDatabase) MigrateTo(store model.InstallationDatabaseStoreInterface, dbMigration *model.DBMigrationOperation, logger logrus.FieldLogger) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "MigrateTo", store, dbMigration, logger)
	ret0, _ := ret[0].(error)
	return ret0
}

// MigrateTo indicates an expected call of MigrateTo
func (mr *MockDatabaseMockRecorder) MigrateTo(store, dbMigration, logger interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "MigrateTo", reflect.TypeOf((*MockDatabase)(nil).MigrateTo), store, dbMigration, logger)
}

// MockInstallationDatabaseStoreInterface is a mock of InstallationDatabaseStoreInterface interface
type MockInstallationDatabaseStoreInterface struct {
	ctrl     *gomock.Controller
	recorder *MockInstallationDatabaseStoreInterfaceMockRecorder
}

// MockInstallationDatabaseStoreInterfaceMockRecorder is the mock recorder for MockInstallationDatabaseStoreInterface
type MockInstallationDatabaseStoreInterfaceMockRecorder struct {
	mock *MockInstallationDatabaseStoreInterface
}

// NewMockInstallationDatabaseStoreInterface creates a new mock instance
func NewMockInstallationDatabaseStoreInterface(ctrl *gomock.Controller) *MockInstallationDatabaseStoreInterface {
	mock := &MockInstallationDatabaseStoreInterface{ctrl: ctrl}
	mock.recorder = &MockInstallationDatabaseStoreInterfaceMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockInstallationDatabaseStoreInterface) EXPECT() *MockInstallationDatabaseStoreInterfaceMockRecorder {
	return m.recorder
}

// GetClusterInstallations mocks base method
func (m *MockInstallationDatabaseStoreInterface) GetClusterInstallations(filter *model.ClusterInstallationFilter) ([]*model.ClusterInstallation, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetClusterInstallations", filter)
	ret0, _ := ret[0].([]*model.ClusterInstallation)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetClusterInstallations indicates an expected call of GetClusterInstallations
func (mr *MockInstallationDatabaseStoreInterfaceMockRecorder) GetClusterInstallations(filter interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetClusterInstallations", reflect.TypeOf((*MockInstallationDatabaseStoreInterface)(nil).GetClusterInstallations), filter)
}

// GetMultitenantDatabase mocks base method
func (m *MockInstallationDatabaseStoreInterface) GetMultitenantDatabase(multitenantdatabaseID string) (*model.MultitenantDatabase, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMultitenantDatabase", multitenantdatabaseID)
	ret0, _ := ret[0].(*model.MultitenantDatabase)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMultitenantDatabase indicates an expected call of GetMultitenantDatabase
func (mr *MockInstallationDatabaseStoreInterfaceMockRecorder) GetMultitenantDatabase(multitenantdatabaseID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMultitenantDatabase", reflect.TypeOf((*MockInstallationDatabaseStoreInterface)(nil).GetMultitenantDatabase), multitenantdatabaseID)
}

// GetMultitenantDatabases mocks base method
func (m *MockInstallationDatabaseStoreInterface) GetMultitenantDatabases(filter *model.MultitenantDatabaseFilter) ([]*model.MultitenantDatabase, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMultitenantDatabases", filter)
	ret0, _ := ret[0].([]*model.MultitenantDatabase)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMultitenantDatabases indicates an expected call of GetMultitenantDatabases
func (mr *MockInstallationDatabaseStoreInterfaceMockRecorder) GetMultitenantDatabases(filter interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMultitenantDatabases", reflect.TypeOf((*MockInstallationDatabaseStoreInterface)(nil).GetMultitenantDatabases), filter)
}

// GetMultitenantDatabaseForInstallationID mocks base method
func (m *MockInstallationDatabaseStoreInterface) GetMultitenantDatabaseForInstallationID(installationID string) (*model.MultitenantDatabase, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetMultitenantDatabaseForInstallationID", installationID)
	ret0, _ := ret[0].(*model.MultitenantDatabase)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetMultitenantDatabaseForInstallationID indicates an expected call of GetMultitenantDatabaseForInstallationID
func (mr *MockInstallationDatabaseStoreInterfaceMockRecorder) GetMultitenantDatabaseForInstallationID(installationID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetMultitenantDatabaseForInstallationID", reflect.TypeOf((*MockInstallationDatabaseStoreInterface)(nil).GetMultitenantDatabaseForInstallationID), installationID)
}

// GetInstallationsTotalDatabaseWeight mocks base method
func (m *MockInstallationDatabaseStoreInterface) GetInstallationsTotalDatabaseWeight(installationIDs []string) (float64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetInstallationsTotalDatabaseWeight", installationIDs)
	ret0, _ := ret[0].(float64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetInstallationsTotalDatabaseWeight indicates an expected call of GetInstallationsTotalDatabaseWeight
func (mr *MockInstallationDatabaseStoreInterfaceMockRecorder) GetInstallationsTotalDatabaseWeight(installationIDs interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetInstallationsTotalDatabaseWeight", reflect.TypeOf((*MockInstallationDatabaseStoreInterface)(nil).GetInstallationsTotalDatabaseWeight), installationIDs)
}

// CreateMultitenantDatabase mocks base method
func (m *MockInstallationDatabaseStoreInterface) CreateMultitenantDatabase(multitenantDatabase *model.MultitenantDatabase) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateMultitenantDatabase", multitenantDatabase)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateMultitenantDatabase indicates an expected call of CreateMultitenantDatabase
func (mr *MockInstallationDatabaseStoreInterfaceMockRecorder) CreateMultitenantDatabase(multitenantDatabase interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateMultitenantDatabase", reflect.TypeOf((*MockInstallationDatabaseStoreInterface)(nil).CreateMultitenantDatabase), multitenantDatabase)
}

// UpdateMultitenantDatabase mocks base method
func (m *MockInstallationDatabaseStoreInterface) UpdateMultitenantDatabase(multitenantDatabase *model.MultitenantDatabase) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateMultitenantDatabase", multitenantDatabase)
	ret0, _ := ret[0].(error)
	return ret0
}

// UpdateMultitenantDatabase indicates an expected call of UpdateMultitenantDatabase
func (mr *MockInstallationDatabaseStoreInterfaceMockRecorder) UpdateMultitenantDatabase(multitenantDatabase interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateMultitenantDatabase", reflect.TypeOf((*MockInstallationDatabaseStoreInterface)(nil).UpdateMultitenantDatabase), multitenantDatabase)
}

// LockMultitenantDatabase mocks base method
func (m *MockInstallationDatabaseStoreInterface) LockMultitenantDatabase(multitenantdatabaseID, lockerID string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LockMultitenantDatabase", multitenantdatabaseID, lockerID)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LockMultitenantDatabase indicates an expected call of LockMultitenantDatabase
func (mr *MockInstallationDatabaseStoreInterfaceMockRecorder) LockMultitenantDatabase(multitenantdatabaseID, lockerID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LockMultitenantDatabase", reflect.TypeOf((*MockInstallationDatabaseStoreInterface)(nil).LockMultitenantDatabase), multitenantdatabaseID, lockerID)
}

// UnlockMultitenantDatabase mocks base method
func (m *MockInstallationDatabaseStoreInterface) UnlockMultitenantDatabase(multitenantdatabaseID, lockerID string, force bool) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UnlockMultitenantDatabase", multitenantdatabaseID, lockerID, force)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UnlockMultitenantDatabase indicates an expected call of UnlockMultitenantDatabase
func (mr *MockInstallationDatabaseStoreInterfaceMockRecorder) UnlockMultitenantDatabase(multitenantdatabaseID, lockerID, force interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnlockMultitenantDatabase", reflect.TypeOf((*MockInstallationDatabaseStoreInterface)(nil).UnlockMultitenantDatabase), multitenantdatabaseID, lockerID, force)
}

// LockMultitenantDatabases mocks base method
func (m *MockInstallationDatabaseStoreInterface) LockMultitenantDatabases(ids []string, lockerID string) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "LockMultitenantDatabases", ids, lockerID)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// LockMultitenantDatabases indicates an expected call of LockMultitenantDatabases
func (mr *MockInstallationDatabaseStoreInterfaceMockRecorder) LockMultitenantDatabases(ids, lockerID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "LockMultitenantDatabases", reflect.TypeOf((*MockInstallationDatabaseStoreInterface)(nil).LockMultitenantDatabases), ids, lockerID)
}

// UnlockMultitenantDatabases mocks base method
func (m *MockInstallationDatabaseStoreInterface) UnlockMultitenantDatabases(ids []string, lockerID string, force bool) (bool, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UnlockMultitenantDatabases", ids, lockerID, force)
	ret0, _ := ret[0].(bool)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UnlockMultitenantDatabases indicates an expected call of UnlockMultitenantDatabases
func (mr *MockInstallationDatabaseStoreInterfaceMockRecorder) UnlockMultitenantDatabases(ids, lockerID, force interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnlockMultitenantDatabases", reflect.TypeOf((*MockInstallationDatabaseStoreInterface)(nil).UnlockMultitenantDatabases), ids, lockerID, force)
}

// GetSingleTenantDatabaseConfigForInstallation mocks base method
func (m *MockInstallationDatabaseStoreInterface) GetSingleTenantDatabaseConfigForInstallation(installationID string) (*model.SingleTenantDatabaseConfig, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetSingleTenantDatabaseConfigForInstallation", installationID)
	ret0, _ := ret[0].(*model.SingleTenantDatabaseConfig)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetSingleTenantDatabaseConfigForInstallation indicates an expected call of GetSingleTenantDatabaseConfigForInstallation
func (mr *MockInstallationDatabaseStoreInterfaceMockRecorder) GetSingleTenantDatabaseConfigForInstallation(installationID interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetSingleTenantDatabaseConfigForInstallation", reflect.TypeOf((*MockInstallationDatabaseStoreInterface)(nil).GetSingleTenantDatabaseConfigForInstallation), installationID)
}
