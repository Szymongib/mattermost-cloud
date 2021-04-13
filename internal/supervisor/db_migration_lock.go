// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import log "github.com/sirupsen/logrus"

type dBMigrationOperationLockStore interface {
	LockDBMigrationOperations(id []string, lockerID string) (bool, error)
	UnlockDBMigrationOperations(id []string, lockerID string, force bool) (bool, error)
}

type dBMigrationOperationLock struct {
	ids []string
	lockerID               string
	store                  dBMigrationOperationLockStore
	logger                 log.FieldLogger
}

func newDBMigrationOperationLock(id, lockerID string, store dBMigrationOperationLockStore, logger log.FieldLogger) *dBMigrationOperationLock {
	return &dBMigrationOperationLock{
		ids: []string{id},
		lockerID:               lockerID,
		store:                  store,
		logger:                 logger,
	}
}

func newDBMigrationOperationLocks(ids []string, lockerID string, store dBMigrationOperationLockStore, logger log.FieldLogger) *dBMigrationOperationLock {
	return &dBMigrationOperationLock{
		ids: ids,
		lockerID:               lockerID,
		store:                  store,
		logger:                 logger,
	}
}

func (l *dBMigrationOperationLock) TryLock() bool {
	locked, err := l.store.LockDBMigrationOperations(l.ids, l.lockerID)
	if err != nil {
		l.logger.WithError(err).Error("failed to lock dBMigrationOperations")
		return false
	}

	return locked
}

func (l *dBMigrationOperationLock) Unlock() {
	unlocked, err := l.store.UnlockDBMigrationOperations(l.ids, l.lockerID, false)
	if err != nil {
		l.logger.WithError(err).Error("failed to unlock dBMigrationOperations")
	} else if unlocked != true {
		l.logger.Error("failed to release lock for dBMigrationOperations")
	}
}
