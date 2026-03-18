package cmd

import (
	"fmt"
	"os"
	"strconv"

	"github.com/rodaine/table"
	"github.com/spf13/cobra"
	"github.com/zoldyzdk/bb-cli/internal/api"
	"github.com/zoldyzdk/bb-cli/internal/config"
)

var (
	prListState string
	prListLimit int
)

var prListCmd = &cobra.Command{
	Use:   "list",
	Short: "List pull requests",
	Long:  `List pull requests for a Bitbucket repository. By default shows open PRs.`,
	RunE:  runPRList,
}

func init() {
	prCmd.AddCommand(prListCmd)
	prListCmd.Flags().StringVar(&prListState, "state", "OPEN", "PR state filter: OPEN, MERGED, DECLINED, SUPERSEDED")
	prListCmd.Flags().IntVar(&prListLimit, "limit", 25, "Maximum number of PRs to display")
}

func runPRList(cmd *cobra.Command, args []string) error {
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
	prs, err := client.ListPullRequests(workspace, repo, prListState, prListLimit)
	if err != nil {
		return fmt.Errorf("failed to list pull requests: %w", err)
	}

	if len(prs) == 0 {
		fmt.Printf("No %s pull requests found in %s/%s\n", prListState, workspace, repo)
		return nil
	}

	tbl := table.New("ID", "TITLE", "AUTHOR", "SOURCE", "STATE").WithWriter(os.Stdout)

	for _, pr := range prs {
		title := pr.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}
		tbl.AddRow(
			strconv.Itoa(pr.ID),
			title,
			pr.Author.DisplayName,
			pr.Source.Branch.Name,
			pr.State,
		)
	}

	tbl.Print()
	return nil
}
