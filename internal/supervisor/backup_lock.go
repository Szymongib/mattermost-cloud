// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	log "github.com/sirupsen/logrus"
)

type backupMetadataLockStore interface {
	LockBackupMetadata(installationID, lockerID string) (bool, error)
	UnlockBackupMetadata(installationID, lockerID string, force bool) (bool, error)
}

type backupMetadataLock struct {
	backupMetadataID string
	lockerID       string
	store          backupMetadataLockStore
	logger         log.FieldLogger
}

func newBackupLock(backupMetadataID, lockerID string, store backupMetadataLockStore, logger log.FieldLogger) *backupMetadataLock {
	return &backupMetadataLock{
		backupMetadataID: backupMetadataID,
		lockerID:       lockerID,
		store:          store,
		logger:         logger,
	}
}

func (l *backupMetadataLock) TryLock() bool {
	locked, err := l.store.LockBackupMetadata(l.backupMetadataID, l.lockerID)
	if err != nil {
		l.logger.WithError(err).Error("failed to lock backup metadata")
		return false
	}

	return locked
}

func (l *backupMetadataLock) Unlock() {
	unlocked, err := l.store.UnlockBackupMetadata(l.backupMetadataID, l.lockerID, false)
	if err != nil {
		l.logger.WithError(err).Error("failed to unlock backup metadata")
	} else if unlocked != true {
		l.logger.Error("failed to release lock for backup metadata")
	}
}
