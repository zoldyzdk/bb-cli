package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/zoldyzdk/bb-cli/internal/api"
	"github.com/zoldyzdk/bb-cli/internal/config"
)

var (
	prCheckoutBranch string
	prCheckoutForce  bool
)

var prCheckoutCmd = &cobra.Command{
	Use:   "checkout <pr-id>",
	Short: "Check out a pull request branch",
	Long:  `Fetch the source branch of a pull request and check it out locally.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runPRCheckout,
}

func init() {
	prCmd.AddCommand(prCheckoutCmd)
	prCheckoutCmd.Flags().StringVarP(&prCheckoutBranch, "branch", "b", "", "Local branch name to use (defaults to PR source branch)")
	prCheckoutCmd.Flags().BoolVarP(&prCheckoutForce, "force", "f", false, "Reset existing local branch to latest PR state")
}

func runPRCheckout(cmd *cobra.Command, args []string) error {
	prID, err := strconv.Atoi(args[0])
	if err != nil {
		return fmt.Errorf("invalid PR ID: %s", args[0])
	}

	workspace, repo, err := resolveWorkspaceAndRepo()
	if err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if !cfg.HasCredentials() {
		return fmt.Errorf("not logged in. Run 'bb auth login' first")
	}

	client := api.NewClient(cfg.Username, cfg.Token)
	pr, err := client.GetPullRequest(workspace, repo, prID)
	if err != nil {
		return fmt.Errorf("failed to get pull request: %w", err)
	}

	localBranch := pr.Source.Branch.Name
	if prCheckoutBranch != "" {
		localBranch = prCheckoutBranch
	}

	if err := gitFetch("origin", pr.Source.Branch.Name, localBranch, prCheckoutForce); err != nil {
		return err
	}

	if err := gitCheckout(localBranch); err != nil {
		return err
	}

	fmt.Printf("Switched to branch '%s' (PR #%d: %s)\n", localBranch, pr.ID, pr.Title)
	return nil
}
