package main

import (
	"github.com/mattermost/mattermost-cloud/internal/tools/utils"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"os"
)

func init() {
	//installationGetCmd.Flags().String("installation", "", "The id of the installation to be fetched.")
	//installationGetCmd.Flags().Bool("include-group-config", true, "Whether to include group configuration in the installation or not.")
	//installationGetCmd.Flags().Bool("include-group-config-overrides", true, "Whether to include a group configuration override summary in the installation or not.")
	//installationGetCmd.Flags().Bool("hide-license", true, "Whether to hide the license value in the output or not.")
	//installationGetCmd.MarkFlagRequired("installation")
	//
	//installationListCmd.Flags().String("owner", "", "The owner ID to filter installations by.")
	//installationListCmd.Flags().String("group", "", "The group ID to filter installations.")
	//installationListCmd.Flags().String("state", "", "The state to filter installations by.")
	//installationListCmd.Flags().String("dns", "", "The dns name to filter installations by.")
	//installationListCmd.Flags().Bool("include-group-config", true, "Whether to include group configuration in the installations or not.")
	//installationListCmd.Flags().Bool("include-group-config-overrides", true, "Whether to include a group configuration override summary in the installations or not.")
	//installationListCmd.Flags().Bool("hide-license", true, "Whether to hide the license value in the output or not.")
	//installationListCmd.Flags().Bool("table", false, "Whether to display the returned installation list in a table or not.")
	//registerPagingFlags(installationListCmd)

	installationRestorationRequestCmd.Flags().String("installation", "", "The id of the installation to be restored.")
	installationRestorationRequestCmd.Flags().String("backup", "", "The id of the backup to restore.")
	installationRestorationRequestCmd.MarkFlagRequired("installation")
	installationRestorationRequestCmd.MarkFlagRequired("backup")

	installationRestorationsListCmd.Flags().String("installation", "", "The id of the installation to query operations.")
	installationRestorationsListCmd.Flags().String("state", "", "The state to filter operations by.")
	installationRestorationsListCmd.Flags().String("cluster-installation", "", "The cluster installation to filter operations by.")
	registerPagingFlags(installationRestorationsListCmd)
	installationRestorationsListCmd.Flags().Bool("table", false, "Whether to display output in a table or not.")

	//installationDatabaseMigrationCmd.Flags().String("installation", "", "The id of the installation to be migrated.")
	//installationDatabaseMigrationCmd.Flags().String("multi-tenant-db", "", "The id of the destination multi tenant db.")
	//installationDatabaseMigrationCmd.MarkFlagRequired("installation")
	//installationDatabaseMigrationCmd.MarkFlagRequired("multi-tenant-db")

	installationRestorationOperationCmd.AddCommand(installationRestorationRequestCmd)
	installationRestorationOperationCmd.AddCommand(installationRestorationsListCmd)
}

var installationRestorationOperationCmd = &cobra.Command{
	Use:   "restoration",
	Short: "Manipulate installation restoration operations managed by the provisioning server.",
}

var installationRestorationRequestCmd = &cobra.Command{
	Use:   "request",
	Short: "Request database restoration",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationID, _ := command.Flags().GetString("installation")
		backupID, _ := command.Flags().GetString("backup")

		installationDTO, err := client.RestoreInstallationDatabase(installationID, backupID)
		if err != nil {
			return errors.Wrap(err, "failed to request installation database restoration")
		}

		err = printJSON(installationDTO)
		if err != nil {
			return err
		}

		return nil
	},
}

var installationRestorationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installation database restoration operations",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationID, _ := command.Flags().GetString("installation")
		clusterInstallationID, _ := command.Flags().GetString("cluster-installation")
		state, _ := command.Flags().GetString("state")
		paging := parsePagingFlags(command)

		request := &model.GetInstallationDBRestorationOperationsRequest{
			Paging:                paging,
			InstallationID:        installationID,
			ClusterInstallationID: clusterInstallationID,
			State:                 state,
		}

		dbRestorationOperations, err := client.GetInstallationDBRestorationOperations(request)
		if err != nil {
			return errors.Wrap(err, "failed to list installation database restoration operations")
		}

		outputToTable, _ := command.Flags().GetBool("table")
		if outputToTable {
			table := tablewriter.NewWriter(os.Stdout)
			table.SetAlignment(tablewriter.ALIGN_LEFT)
			table.SetHeader([]string{"ID", "INSTALLATION ID", "BACKUP ID", "STATE", "CLUSTER INSTALLATION ID", "TARGET INSTALLATION STATE", "REQUEST AT"})

			for _, restoration := range dbRestorationOperations {
				table.Append([]string{
					restoration.ID,
					restoration.InstallationID,
					restoration.BackupID,
					string(restoration.State),
					restoration.ClusterInstallationID,
					restoration.TargetInstallationState,
					utils.TimeFromMillis(restoration.RequestAt).Format("2006-01-02 15:04:05 -0700 MST"),
				})
			}
			table.Render()

			return nil
		}

		err = printJSON(dbRestorationOperations)
		if err != nil {
			return err
		}

		return nil
	},
}
