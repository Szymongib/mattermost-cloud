package supervisor

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type clusterInstallationClaimStore interface {
	GetClusterInstallations(*model.ClusterInstallationFilter) ([]*model.ClusterInstallation, error)
	clusterInstallationLockStore
}

func claimClusterInstallation(store clusterInstallationClaimStore, installation *model.Installation, instanceID string, logger log.FieldLogger) (*model.ClusterInstallation, *clusterInstallationLock, error) {
	clusterInstallationFilter := &model.ClusterInstallationFilter{
		InstallationID: installation.ID,
		Paging:         model.AllPagesNotDeleted(),
	}
	clusterInstallations, err := store.GetClusterInstallations(clusterInstallationFilter)
	if err != nil {
		return nil, nil, errors.Wrap(err, "Failed to get cluster installations")

	}

	if len(clusterInstallations) == 0 {
		return nil, nil, errors.Wrap(err, "Expected at least one cluster installation for the installation but found none")
	}

	claimedCI := clusterInstallations[0]
	ciLock := newClusterInstallationLock(claimedCI.ID, instanceID, store, logger)
	if !ciLock.TryLock() {
		return nil, nil, errors.Errorf("Failed to lock cluster installation %s", claimedCI.ID)
	}

	return claimedCI, ciLock, nil
}

type getAndLockInstallationStore interface {
	GetInstallation(installationID string, includeGroupConfig, includeGroupConfigOverrides bool) (*model.Installation, error)
	installationLockStore
}

func getAndLockInstallation(store getAndLockInstallationStore, installationID, instanceID string, logger log.FieldLogger) (*model.Installation, *installationLock, error) {
	installation, err := store.GetInstallation(installationID, false, false)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get installation")
	}
	if installation == nil {
		return nil, nil, errors.New("could not found the installation")
	}

	lock := newInstallationLock(installation.ID, instanceID, store, logger)
	if !lock.TryLock() {
		logger.Debugf("Failed to lock installation %s", installation.ID)
		return nil, nil, errors.New("failed to lock installation")
	}
	return installation, lock, nil
}

//func claimClusterInstallationID(store clusterInstallationClaimStore, installationID string) (*model.ClusterInstallation, error) {
//	clusterInstallationFilter := &model.ClusterInstallationFilter{
//		InstallationID: installationID,
//		Paging:         model.AllPagesNotDeleted(),
//	}
//	clusterInstallations, err := store.GetClusterInstallations(clusterInstallationFilter)
//	if err != nil {
//		return nil, errors.Wrap(err, "Failed to get cluster installations")
//	}
//
//	if len(clusterInstallations) == 0 {
//		return  nil, errors.Wrap(err, "Expected at least one cluster installation for the installation but found none")
//	}
//
//	return clusterInstallations[0], nil
//}
