package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zoldyzdk/bb-cli/internal/api"
	"github.com/zoldyzdk/bb-cli/internal/config"
)

var prDiffNameOnly bool

var prDiffCmd = &cobra.Command{
	Use:   "diff <pr-id>",
	Short: "View the diff of a pull request",
	Long:  `Display the diff of a pull request. Shows the raw unified diff by default.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runPRDiff,
}

func init() {
	prCmd.AddCommand(prDiffCmd)
	prDiffCmd.Flags().BoolVar(&prDiffNameOnly, "name-only", false, "Display only names of changed files")
}

func runPRDiff(cmd *cobra.Command, args []string) error {
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
	diff, err := client.GetPullRequestDiff(workspace, repo, prID)
	if err != nil {
		return fmt.Errorf("failed to get pull request diff: %w", err)
	}

	if prDiffNameOnly {
		for _, name := range extractFileNames(diff) {
			fmt.Println(name)
		}
		return nil
	}

	fmt.Print(diff)
	return nil
}

func extractFileNames(diff string) []string {
	seen := make(map[string]bool)
	var files []string

	lines := strings.Split(diff, "\n")
	for i, line := range lines {
		if !strings.HasPrefix(line, "+++ ") {
			continue
		}
		if strings.HasPrefix(line, "+++ /dev/null") {
			if i > 0 && strings.HasPrefix(lines[i-1], "--- a/") {
				name := strings.TrimPrefix(lines[i-1], "--- a/")
				if !seen[name] {
					seen[name] = true
					files = append(files, name)
				}
			}
			continue
		}
		if strings.HasPrefix(line, "+++ b/") {
			name := strings.TrimPrefix(line, "+++ b/")
			if !seen[name] {
				seen[name] = true
				files = append(files, name)
			}
		}
	}

	return files
}
