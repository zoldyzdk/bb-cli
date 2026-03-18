package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zoldyzdk/bb-cli/internal/api"
	"github.com/zoldyzdk/bb-cli/internal/config"
)

var prCommentsLimit int

var prCommentsCmd = &cobra.Command{
	Use:   "comments <pr-id>",
	Short: "List comments on a pull request",
	Long:  `Retrieve and display all comments on a specific pull request.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runPRComments,
}

func init() {
	prCmd.AddCommand(prCommentsCmd)
	prCommentsCmd.Flags().IntVar(&prCommentsLimit, "limit", 50, "Maximum number of comments to display")
}

func runPRComments(cmd *cobra.Command, args []string) error {
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
	comments, err := client.ListPullRequestComments(workspace, repo, prID, prCommentsLimit)
	if err != nil {
		return fmt.Errorf("failed to list comments: %w", err)
	}

	if len(comments) == 0 {
		fmt.Printf("No comments on PR #%d\n", prID)
		return nil
	}

	for i, c := range comments {
		if c.Deleted {
			continue
		}

		location := ""
		if c.Inline != nil {
			location = fmt.Sprintf(" [%s", c.Inline.Path)
			if c.Inline.To != nil {
				location += fmt.Sprintf(":%d", *c.Inline.To)
			}
			location += "]"
		}

		fmt.Printf("--- Comment #%d%s ---\n", c.ID, location)
		fmt.Printf("Author: %s\n", c.User.DisplayName)
		fmt.Printf("Date:   %s\n", c.CreatedOn.Format("2006-01-02 15:04:05"))

		content := strings.TrimSpace(c.Content.Raw)
		if content != "" {
			fmt.Printf("\n%s\n", content)
		}

		if i < len(comments)-1 {
			fmt.Println()
		}
	}

	return nil
}
