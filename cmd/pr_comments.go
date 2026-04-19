package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zoldyzdk/bb-cli/internal/api"
	"github.com/zoldyzdk/bb-cli/internal/config"
	"github.com/zoldyzdk/bb-cli/internal/models"
)

var prCommentsLimit int
var prCommentsResolved bool
var prCommentsUnresolved bool

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
	prCommentsCmd.Flags().BoolVar(&prCommentsResolved, "resolved", false, "Show only resolved comments")
	prCommentsCmd.Flags().BoolVar(&prCommentsUnresolved, "unresolved", false, "Show only unresolved comments")
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

	// Filter by resolution status if flags are set (but not both)
	if prCommentsResolved && !prCommentsUnresolved {
		filtered := make([]models.Comment, 0)
		for _, c := range comments {
			if c.Resolution != nil {
				filtered = append(filtered, c)
			}
		}
		comments = filtered
	} else if prCommentsUnresolved && !prCommentsResolved {
		filtered := make([]models.Comment, 0)
		for _, c := range comments {
			if c.Resolution == nil {
				filtered = append(filtered, c)
			}
		}
		comments = filtered
	}

	if len(comments) == 0 {
		fmt.Printf("No comments on PR #%d\n", prID)
		return nil
	}

	for i, c := range comments {
		if c.Deleted {
			continue
		}

		status := "OPEN"
		if c.Resolution != nil {
			status = "RESOLVED"
		}

		location := ""
		if c.Inline != nil {
			location = fmt.Sprintf(" [%s", c.Inline.Path)
			if c.Inline.To != nil {
				location += fmt.Sprintf(":%d", *c.Inline.To)
			}
			location += "]"
		}

		fmt.Printf("--- Comment #%d [%s]%s ---\n", c.ID, status, location)
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
