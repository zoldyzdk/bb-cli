package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/zoldyzdk/bb-cli/internal/api"
	"github.com/zoldyzdk/bb-cli/internal/config"
	"golang.org/x/term"
)

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Log in to Bitbucket",
	Long:  `Authenticate with Bitbucket Cloud using your email and an API token. Credentials are stored locally in ~/.config/bb-cli/config.json.`,
	RunE:  runAuthLogin,
}

func init() {
	authCmd.AddCommand(authLoginCmd)
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Bitbucket username (email): ")
	username, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read username: %w", err)
	}
	username = strings.TrimSpace(username)

	fmt.Print("Bitbucket API token: ")
	tokenBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return fmt.Errorf("failed to read token: %w", err)
	}
	fmt.Println()
	token := strings.TrimSpace(string(tokenBytes))

	if username == "" || token == "" {
		return fmt.Errorf("username and token are required")
	}

	fmt.Print("Validating credentials... ")
	client := api.NewClient(username, token)
	user, err := client.GetCurrentUser()
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}
	fmt.Printf("authenticated as %s\n", user.DisplayName)

	cfg, err := config.Load()
	if err != nil {
		cfg = &config.Config{}
	}

	cfg.Username = username
	cfg.Token = token

	if err := config.Save(cfg); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	fmt.Println("Credentials saved to ~/.config/bb-cli/config.json")
	return nil
}
