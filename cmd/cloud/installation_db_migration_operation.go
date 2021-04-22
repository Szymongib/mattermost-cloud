// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

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

	installationDBMigrationRequestCmd.Flags().String("installation", "", "The id of the installation to be migrated.")
	installationDBMigrationRequestCmd.Flags().String("destination-db", model.InstallationDatabaseMultiTenantRDSPostgres, "The destination database type.")
	installationDBMigrationRequestCmd.Flags().String("multi-tenant-db", "", "The id of the destination multi tenant db.")
	installationDBMigrationRequestCmd.MarkFlagRequired("installation")
	installationDBMigrationRequestCmd.MarkFlagRequired("destination-db")
	installationDBMigrationRequestCmd.MarkFlagRequired("multi-tenant-db")

	installationDBMigrationsListCmd.Flags().String("installation", "", "The id of the installation to query operations.")
	installationDBMigrationsListCmd.Flags().String("state", "", "The state to filter operations by.")
	registerPagingFlags(installationDBMigrationsListCmd)
	installationDBMigrationsListCmd.Flags().Bool("table", false, "Whether to display output in a table or not.")

	installationDBMigrationOperationCmd.AddCommand(installationRestorationRequestCmd)
	installationDBMigrationOperationCmd.AddCommand(installationDBMigrationsListCmd)
}

var installationDBMigrationOperationCmd = &cobra.Command{
	Use:   "db-migration",
	Short: "Manipulate installation db migration operations managed by the provisioning server.",
}

var installationDBMigrationRequestCmd = &cobra.Command{
	Use:   "request",
	Short: "Request database migration to different DB",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		// For now only multi-tenant postgres DB is supported.
		installationID, _ := command.Flags().GetString("installation")
		destinationDB, _ := command.Flags().GetString("destination-db")
		multiTenantDBID, _ := command.Flags().GetString("multi-tenant-db")

		request := &model.DBMigrationRequest{
			InstallationID:         installationID,
			DestinationDatabase:    destinationDB,
			DestinationMultiTenant: &model.MultiTenantDBMigrationData{DatabaseID: multiTenantDBID},
		}

		dryRun, _ := command.Flags().GetBool("dry-run")
		if dryRun {
			err := printJSON(request)
			if err != nil {
				return errors.Wrap(err, "failed to print API request")
			}

			return nil
		}

		migrationOperation, err := client.MigrateInstallationDatabase(request)
		if err != nil {
			return errors.Wrap(err, "failed to request installation database migration")
		}

		err = printJSON(migrationOperation)
		if err != nil {
			return err
		}

		return nil
	},
}

var installationDBMigrationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installation database migration operations",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationID, _ := command.Flags().GetString("installation")
		state, _ := command.Flags().GetString("state")
		paging := parsePagingFlags(command)

		request := &model.GetDBMigrationOperationsRequest{
			Paging:         paging,
			InstallationID: installationID,
			State:          state,
		}

		dbMigrationOperations, err := client.GetInstallationDBMigrationOperations(request)
		if err != nil {
			return errors.Wrap(err, "failed to list installation database migration operations")
		}

		outputToTable, _ := command.Flags().GetBool("table")
		if outputToTable {
			table := tablewriter.NewWriter(os.Stdout)
			table.SetAlignment(tablewriter.ALIGN_LEFT)
			table.SetHeader([]string{"ID", "INSTALLATION ID", "STATE", "REQUEST AT"})

			for _, migration := range dbMigrationOperations {
				table.Append([]string{
					migration.ID,
					migration.InstallationID,
					string(migration.State),
					utils.TimeFromMillis(migration.RequestAt).Format("2006-01-02 15:04:05 -0700 MST"),
				})
			}
			table.Render()

			return nil
		}

		err = printJSON(dbMigrationOperations)
		if err != nil {
			return err
		}

		return nil
	},
}
