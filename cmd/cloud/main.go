// Package main is the entry point to the Mattermost Cloud provisioning server and CLI.
package main

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cloud",
	Short: "Cloud is a tool to provision, manage, and monitor Kubernetes clusters.",
	Run: func(cmd *cobra.Command, args []string) {
		serverCmd.Run(cmd, args)
	},
	// SilenceErrors allows us to explicitly log the error returned from rootCmd below.
	SilenceErrors: true,
}

func init() {
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(clusterCmd)
	rootCmd.AddCommand(installationCmd)
	rootCmd.AddCommand(groupCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.WithError(err).Error("command failed")
		os.Exit(1)
	}
}
