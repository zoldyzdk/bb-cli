package cmd

import (
	"github.com/spf13/cobra"
)

var prCmd = &cobra.Command{
	Use:   "pr",
	Short: "Manage pull requests",
	Long:  `Create, list, view, and interact with Bitbucket pull requests.`,
}

func init() {
	rootCmd.AddCommand(prCmd)
}
