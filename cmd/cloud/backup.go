package main

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	backupCmd.PersistentFlags().String("server", defaultLocalServerAPI, "The provisioning server whose API will be queried.")
	backupCmd.PersistentFlags().Bool("dry-run", false, "When set to true, only print the API request without sending it.")

	backupRequestCmd.Flags().String("installation", "", "The installation id to be backed up.")
	backupRequestCmd.MarkFlagRequired("installation")

	backupListCmd.Flags().String("installation", "", "The installation id for which the backups should be listed.")
	backupListCmd.Flags().String("state", "", "The state to filter backups by.")
	backupListCmd.Flags().Int("page", 0, "The page of installations to fetch, starting at 0.")
	backupListCmd.Flags().Int("per-page", 100, "The number of installations to fetch per page.")
	backupListCmd.Flags().Bool("include-deleted", false, "Whether to include deleted installations.")
	backupListCmd.MarkFlagRequired("installation") // TODO: this is temporary before I change API

	backupGetCmd.Flags().String("installation", "", "The id of the installation to backup.")
	backupGetCmd.Flags().String("backup", "", "The id of the backup to get.")
	backupGetCmd.MarkFlagRequired("installation") // TODO: this is temporary before I change API
	backupGetCmd.MarkFlagRequired("backup")

	//installationBackupCmd.MarkFlagRequired("backup")

	backupCmd.AddCommand(backupRequestCmd)
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupGetCmd)

	//installationCmd.AddCommand(installationCreateCmd)
	//installationCmd.AddCommand(installationUpdateCmd)
	//installationCmd.AddCommand(installationDeleteCmd)
	//installationCmd.AddCommand(installationHibernateCmd)
	//installationCmd.AddCommand(installationWakeupCmd)
	//installationCmd.AddCommand(installationGetCmd)
	//installationCmd.AddCommand(installationListCmd)
	//installationCmd.AddCommand(installationShowStateReport)
	//installationCmd.AddCommand(installationBackupCmd)
	//installationCmd.AddCommand(installationBackupGetCmd)
	//installationCmd.AddCommand(installationAnnotationCmd)
}

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manipulate installation backups managed by the provisioning server.",
}

var backupRequestCmd = &cobra.Command{
	Use:   "request",
	Short: "Request an installation backup.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationID, _ := command.Flags().GetString("installation")

		backupMetadata, err := client.RequestInstallationBackup(installationID)
		if err != nil {
			return errors.Wrap(err, "failed to request installation backup")
		}

		return printJSON(backupMetadata)
	},
}

var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List backups metadata.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationID, _ := command.Flags().GetString("installation")
		clusterInstallationID, _ := command.Flags().GetString("cluster-installation")
		state, _ := command.Flags().GetString("state")
		page, _ := command.Flags().GetInt("page")
		perPage, _ := command.Flags().GetInt("per-page")
		includeDeleted, _ := command.Flags().GetBool("include-deleted")

		request := &model.GetBackupsMetadataRequest{
			ClusterInstallationID: clusterInstallationID,
			State:                 state,
			Page:                  page,
			PerPage:               perPage,
			IncludeDeleted:        includeDeleted,
		}

		// TODO: dry run

		backupMetadata, err := client.GetInstallationBackups(installationID, request)
		if err != nil {
			return errors.Wrap(err, "failed to get backup metadata")
		}

		err = printJSON(backupMetadata)
		if err != nil {
			return err
		}

		return nil
	},
}

var backupGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get backup metadata.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationID, _ := command.Flags().GetString("installation")
		backupID, _ := command.Flags().GetString("backup")

		backupMetadata, err := client.GetInstallationBackup(installationID, backupID)
		if err != nil {
			return errors.Wrap(err, "failed to get backup metadata")
		}

		err = printJSON(backupMetadata)
		if err != nil {
			return err
		}

		return nil
	},
}