package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/zoldyzdk/bb-cli/internal/api"
	"github.com/zoldyzdk/bb-cli/internal/models"
	"github.com/zoldyzdk/bb-cli/internal/review"
)

func init() {
	var once bool
	watch := &cobra.Command{Use: "watch", Short: "Poll enrolled repositories every five minutes; generate private reviews", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		settings, project, err := reviewConfiguration()
		if err != nil {
			return err
		}
		if len(project.Repositories) == 0 {
			return fmt.Errorf("no repositories enrolled; add the pilot to config/repositories.yaml")
		}
		for _, r := range project.Repositories {
			if settings.Tokens[r.Workspace+"/"+r.Repository] == "" {
				return fmt.Errorf("repository token missing for %s/%s", r.Workspace, r.Repository)
			}
		}
		release, err := reviewLock(project, "monitor")
		if err != nil {
			return err
		}
		defer release()
		ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		monitors := map[string]*review.Monitor{}
		for _, r := range project.Repositories {
			key := r.Workspace + "/" + r.Repository
			m, err := review.OpenMonitor(filepath.Join(project.ReportsDir, ".state"), key)
			if err != nil {
				return err
			}
			monitors[key] = m
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Local review monitor started. Poll interval: five minutes. Ctrl+C stops it.")
		for {
			interval := 5 * time.Minute
			for _, r := range project.Repositories {
				key := r.Workspace + "/" + r.Repository
				client := api.NewRepositoryClient(settings.Tokens[key]).WithContext(ctx)
				prs, err := client.ListAllOpenPullRequests(r.Workspace, r.Repository)
				if err != nil {
					if ctx.Err() != nil {
						return nil
					}
					if api.IsForbidden(err) || api.IsUnauthorized(err) {
						return err
					}
					fmt.Fprintf(cmd.ErrOrStderr(), "Unable to poll %s; next cycle will retry\n", key)
					var httpErr *api.HTTPError
					if errors.As(err, &httpErr) && httpErr.RetryAfter > interval {
						interval = httpErr.RetryAfter
					}
					if once {
						return err
					}
					continue
				}
				m := monitors[key]
				baseline := !m.Initialized
				err = m.Poll(ctx, prs, func(runCtx context.Context, pr models.PullRequest) error {
					err := executePrivateReview(runCtx, cmd, settings, project, r, pr.ID)
					if err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "Review failed for %s PR #%d: %v\n", key, pr.ID, err)
					}
					return err
				})
				if err != nil {
					if ctx.Err() != nil {
						return nil
					}
					return err
				}
				if baseline {
					fmt.Fprintf(cmd.OutOrStdout(), "Baseline saved for %s: %d existing PRs skipped\n", key, len(prs))
				}
			}
			if once {
				return nil
			}
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(interval):
			}
		}
	}}
	watch.Flags().BoolVar(&once, "once", false, "Run one polling cycle and exit")
	prReviewCmd.AddCommand(watch)
}
