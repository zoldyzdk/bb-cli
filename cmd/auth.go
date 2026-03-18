package cmd

import (
	"github.com/spf13/cobra"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with Bitbucket",
	Long:  `Manage authentication credentials for the Bitbucket Cloud API.`,
}

func init() {
	rootCmd.AddCommand(authCmd)
}
