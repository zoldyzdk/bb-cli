package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/zoldyzdk/bb-cli/internal/config"
)

var (
	cfgWorkspace string
	cfgRepo      string
)

var rootCmd = &cobra.Command{
	Use:   "bb",
	Short: "Bitbucket CLI - like gh for GitHub",
	Long:  `bb is a command-line tool for interacting with Bitbucket Cloud repositories, pull requests, and more.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgWorkspace, "workspace", "w", "", "Bitbucket workspace slug")
	rootCmd.PersistentFlags().StringVarP(&cfgRepo, "repo", "r", "", "Bitbucket repository slug")
}

func resolveWorkspaceAndRepo() (string, string, error) {
	workspace := cfgWorkspace
	repo := cfgRepo

	if workspace == "" || repo == "" {
		cfg, err := config.Load()
		if err == nil {
			if workspace == "" {
				workspace = cfg.Workspace
			}
			if repo == "" {
				repo = cfg.Repo
			}
		}
	}

	if workspace == "" || repo == "" {
		w, r := inferFromGitRemote()
		if workspace == "" {
			workspace = w
		}
		if repo == "" {
			repo = r
		}
	}

	if workspace == "" {
		return "", "", fmt.Errorf("workspace is required: use --workspace flag, set in config, or run from a Bitbucket git repo")
	}
	if repo == "" {
		return "", "", fmt.Errorf("repo is required: use --repo flag, set in config, or run from a Bitbucket git repo")
	}

	return workspace, repo, nil
}
