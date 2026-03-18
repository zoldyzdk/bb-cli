package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zoldyzdk/bb-cli/internal/api"
	"github.com/zoldyzdk/bb-cli/internal/config"
)

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Long:  `Verify that stored credentials are valid by calling the Bitbucket API.`,
	RunE:  runAuthStatus,
}

func init() {
	authCmd.AddCommand(authStatusCmd)
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if !cfg.HasCredentials() {
		return fmt.Errorf("not logged in. Run 'bb auth login' to authenticate")
	}

	client := api.NewClient(cfg.Username, cfg.Token)
	user, err := client.GetCurrentUser()
	if err != nil {
		return fmt.Errorf("authentication check failed: %w", err)
	}

	fmt.Printf("Logged in as: %s\n", user.DisplayName)
	fmt.Printf("Username:     %s\n", cfg.Username)

	if cfg.Workspace != "" {
		fmt.Printf("Workspace:    %s\n", cfg.Workspace)
	}
	if cfg.Repo != "" {
		fmt.Printf("Repository:   %s\n", cfg.Repo)
	}

	return nil
}
