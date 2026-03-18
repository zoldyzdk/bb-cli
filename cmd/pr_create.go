package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/zoldyzdk/bb-cli/internal/api"
	"github.com/zoldyzdk/bb-cli/internal/config"
	"github.com/zoldyzdk/bb-cli/internal/models"
)

var (
	prCreateTitle             string
	prCreateSource            string
	prCreateDestination       string
	prCreateDescription       string
	prCreateReviewers         []string
	prCreateDraft             bool
	prCreateCloseSourceBranch bool
)

var prCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a pull request",
	Long:  `Create a new pull request on a Bitbucket repository.`,
	RunE:  runPRCreate,
}

func init() {
	prCmd.AddCommand(prCreateCmd)
	prCreateCmd.Flags().StringVarP(&prCreateTitle, "title", "t", "", "Pull request title (required)")
	prCreateCmd.Flags().StringVarP(&prCreateSource, "source", "s", "", "Source branch name (required)")
	prCreateCmd.Flags().StringVarP(&prCreateDestination, "destination", "d", "", "Destination branch (defaults to repo main branch)")
	prCreateCmd.Flags().StringVar(&prCreateDescription, "description", "", "Pull request description")
	prCreateCmd.Flags().StringSliceVar(&prCreateReviewers, "reviewer", nil, "Reviewer UUID (can be specified multiple times)")
	prCreateCmd.Flags().BoolVar(&prCreateDraft, "draft", false, "Create as draft pull request")
	prCreateCmd.Flags().BoolVar(&prCreateCloseSourceBranch, "close-source-branch", false, "Close source branch after merge")

	prCreateCmd.MarkFlagRequired("title")
	prCreateCmd.MarkFlagRequired("source")
}

func runPRCreate(cmd *cobra.Command, args []string) error {
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

	body := &models.CreatePullRequestBody{
		Title: prCreateTitle,
		Source: models.Endpoint{
			Branch: models.Branch{Name: prCreateSource},
		},
		Description:       prCreateDescription,
		CloseSourceBranch: prCreateCloseSourceBranch,
		Draft:             prCreateDraft,
	}

	if prCreateDestination != "" {
		body.Destination = &models.Endpoint{
			Branch: models.Branch{Name: prCreateDestination},
		}
	}

	for _, uuid := range prCreateReviewers {
		body.Reviewers = append(body.Reviewers, models.User{UUID: uuid})
	}

	client := api.NewClient(cfg.Username, cfg.Token)
	pr, err := client.CreatePullRequest(workspace, repo, body)
	if err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	fmt.Printf("Pull request #%d created successfully\n", pr.ID)
	fmt.Printf("Title:  %s\n", pr.Title)
	fmt.Printf("State:  %s\n", pr.State)
	fmt.Printf("URL:    %s\n", pr.Links.HTML.Href)

	return nil
}
