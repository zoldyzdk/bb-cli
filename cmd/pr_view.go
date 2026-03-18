package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zoldyzdk/bb-cli/internal/api"
	"github.com/zoldyzdk/bb-cli/internal/config"
)

var prViewCmd = &cobra.Command{
	Use:   "view <pr-id>",
	Short: "View a pull request",
	Long:  `Display detailed information about a specific pull request.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runPRView,
}

func init() {
	prCmd.AddCommand(prViewCmd)
}

func runPRView(cmd *cobra.Command, args []string) error {
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

	draftLabel := ""
	if pr.Draft {
		draftLabel = " (DRAFT)"
	}

	fmt.Printf("#%d %s%s\n", pr.ID, pr.Title, draftLabel)
	fmt.Printf("State:       %s\n", pr.State)
	fmt.Printf("Author:      %s\n", pr.Author.DisplayName)
	fmt.Printf("Source:      %s\n", pr.Source.Branch.Name)
	fmt.Printf("Destination: %s\n", pr.Destination.Branch.Name)
	fmt.Printf("Created:     %s\n", pr.CreatedOn.Format("2006-01-02 15:04:05"))
	fmt.Printf("Updated:     %s\n", pr.UpdatedOn.Format("2006-01-02 15:04:05"))
	fmt.Printf("Comments:    %d\n", pr.CommentCount)
	fmt.Printf("Tasks:       %d\n", pr.TaskCount)

	if len(pr.Reviewers) > 0 {
		names := make([]string, len(pr.Reviewers))
		for i, r := range pr.Reviewers {
			names[i] = r.DisplayName
		}
		fmt.Printf("Reviewers:   %s\n", strings.Join(names, ", "))
	}

	if pr.Description != "" {
		fmt.Printf("\n--- Description ---\n%s\n", pr.Description)
	}

	fmt.Printf("\nURL: %s\n", pr.Links.HTML.Href)

	return nil
}
