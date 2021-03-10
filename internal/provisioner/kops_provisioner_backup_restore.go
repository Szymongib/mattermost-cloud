package provisioner

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func (provisioner *KopsProvisioner) TriggerBackup(backupMetadata *model.InstallationBackup, cluster *model.Cluster, installation *model.Installation) (*model.S3DataResidence, error) {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      cluster.ID,
		"installation": installation.ID,
		"backup":       backupMetadata.ID,
	})
	logger.Info("Triggering backup for installation")

	k8sClient, invalidateCache, err := provisioner.k8sClient(cluster.ProvisionerMetadataKops.Name, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create k8s client")
	}
	defer invalidateCache(err)

	filestoreCfg, filestoreSecret, err := provisioner.resourceUtil.GetFilestore(installation).
		GenerateFilestoreSpecAndSecret(provisioner.store, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get files store configuration for installation")
	}
	// InstallationBackup is not supported for local MinIO storage, therefore this should not happen
	if filestoreCfg == nil || filestoreSecret == nil {
		return nil, errors.New("file store secret and config cannot be empty for backup")
	}
	dbSecret, err := provisioner.resourceUtil.GetDatabase(installation).GenerateDatabaseSecret(provisioner.store, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get database configuration")
	}
	// InstallationBackup is not supported for local MySQL, therefore this should not happen
	if dbSecret == nil {
		return nil, errors.New("database secret cannot be empty for backup")
	}

	jobsClient := k8sClient.Clientset.BatchV1().Jobs(installation.ID)

	return provisioner.BackupOperator.TriggerBackup(jobsClient, backupMetadata, installation, filestoreCfg, dbSecret.Name, logger)
}

// CheckBackupStatus checks status of running backup job,
// returns job start time, when the job finished or -1 if it is still running.
func (provisioner *KopsProvisioner) CheckBackupStatus(backupMetadata *model.InstallationBackup, cluster *model.Cluster) (int64, error) {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":      cluster.ID,
		"installation": backupMetadata.InstallationID,
		"backup":       backupMetadata.ID,
	})
	logger.Info("Checking backup status for installation")

	k8sClient, invalidateCache, err := provisioner.k8sClient(cluster.ProvisionerMetadataKops.Name, logger)
	if err != nil {
		return -1, errors.Wrap(err, "failed to create k8s client")
	}
	defer invalidateCache(err)

	jobsClient := k8sClient.Clientset.BatchV1().Jobs(backupMetadata.InstallationID)

	return provisioner.BackupOperator.CheckBackupStatus(jobsClient, backupMetadata, logger)
}
