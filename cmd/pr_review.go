package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/zoldyzdk/bb-cli/internal/api"
	"github.com/zoldyzdk/bb-cli/internal/review"
	"golang.org/x/term"
)

var reviewProjectDir string

var prReviewCmd = &cobra.Command{
	Use: "review <pr-id>", Short: "Generate a private Hermes review; never publish to Bitbucket", Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.Atoi(args[0])
		if err != nil || id < 1 {
			return fmt.Errorf("invalid PR ID")
		}
		settings, project, err := reviewConfiguration()
		if err != nil {
			return err
		}
		workspace, repo, err := resolveWorkspaceAndRepo()
		if err != nil {
			return err
		}
		var enrollment *review.Enrollment
		for _, r := range project.Repositories {
			if r.Workspace == workspace && r.Repository == repo {
				copy := r
				enrollment = &copy
				break
			}
		}
		if enrollment == nil {
			return fmt.Errorf("repository is not enrolled in the review project")
		}
		ctx, cancel := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		return executePrivateReview(ctx, cmd, settings, project, *enrollment, id)
	},
}

func init() {
	prCmd.AddCommand(prReviewCmd)
	prReviewCmd.PersistentFlags().StringVar(&reviewProjectDir, "project-dir", "", "Review project directory (otherwise use saved configuration)")
	configure := &cobra.Command{Use: "configure", Short: "Save the local review project path", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		if reviewProjectDir == "" {
			return fmt.Errorf("use --project-dir to select the review project")
		}
		project, err := review.LoadProject(reviewProjectDir)
		if err != nil {
			return err
		}
		settings, err := review.LoadSettings()
		if err != nil {
			return err
		}
		settings.ProjectDir = project.Dir
		if err := review.SaveSettings(settings); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Review project configured: %s\nReports: %s\nEnrolled repositories: %d\n", project.Dir, project.ReportsDir, len(project.Repositories))
		return nil
	}}
	prReviewCmd.AddCommand(configure)
	var tokenStdin bool
	token := &cobra.Command{Use: "token", Short: "Store a repository access token privately", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		workspace, repo, err := resolveWorkspaceAndRepo()
		if err != nil {
			return err
		}
		if !review.ValidRepository(workspace, repo) {
			return fmt.Errorf("invalid repository")
		}
		var data []byte
		if tokenStdin {
			data, err = io.ReadAll(io.LimitReader(cmd.InOrStdin(), 8193))
		} else {
			if !term.IsTerminal(int(os.Stdin.Fd())) {
				return fmt.Errorf("use --token-stdin for non-interactive input")
			}
			fmt.Fprint(cmd.ErrOrStderr(), "Repository access token (hidden): ")
			data, err = term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Fprintln(cmd.ErrOrStderr())
		}
		if err != nil {
			return fmt.Errorf("unable to read token")
		}
		value := strings.TrimSpace(string(data))
		if value == "" || len(value) > 8192 || strings.ContainsAny(value, "\r\n\t ") {
			return fmt.Errorf("invalid repository token")
		}
		settings, err := review.LoadSettings()
		if err != nil {
			return err
		}
		settings.Tokens[workspace+"/"+repo] = value
		if err := review.SaveSettings(settings); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Repository token saved for %s/%s; no personal-token fallback\n", workspace, repo)
		return nil
	}}
	token.Flags().BoolVar(&tokenStdin, "token-stdin", false, "Read token from standard input instead of the hidden terminal prompt")
	prReviewCmd.AddCommand(token)
}

func reviewConfiguration() (*review.Settings, *review.Project, error) {
	settings, err := review.LoadSettings()
	if err != nil {
		return nil, nil, err
	}
	dir := reviewProjectDir
	if dir == "" {
		dir = settings.ProjectDir
	}
	if dir == "" {
		return nil, nil, fmt.Errorf("configure the review project with bb pr review configure --project-dir <path>")
	}
	project, err := review.LoadProject(dir)
	return settings, project, err
}

func executePrivateReview(ctx context.Context, cmd *cobra.Command, settings *review.Settings, project *review.Project, enrollment review.Enrollment, id int) error {
	token := settings.Tokens[enrollment.Workspace+"/"+enrollment.Repository]
	if token == "" {
		return fmt.Errorf("repository token missing for %s/%s; use bb pr review token", enrollment.Workspace, enrollment.Repository)
	}
	var release func()
	for {
		var err error
		release, err = reviewLock(project, "execution")
		if err == nil {
			break
		}
		if !errors.Is(err, syscall.EWOULDBLOCK) && !errors.Is(err, syscall.EAGAIN) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	defer release()
	engine := review.Engine{Project: project, API: api.NewRepositoryClient(token).WithContext(ctx), Snapshot: review.GitSnapshotter{}, Runner: review.HermesRunner{}}
	fmt.Fprintf(cmd.OutOrStdout(), "Reviewing %s/%s PR #%d with %s\n", enrollment.Workspace, enrollment.Repository, id, enrollment.Profile)
	path, err := engine.Review(ctx, enrollment, id, token)
	if path != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Private draft: %s.md\nStructured report: %s.json\n", path, path)
	}
	return err
}

func reviewLock(project *review.Project, name string) (func(), error) {
	dir := filepath.Join(project.ReportsDir, ".state")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(filepath.Join(dir, name+".lock"), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, fmt.Errorf("another review %s is active: %w", name, err)
	}
	return func() { syscall.Flock(int(f.Fd()), syscall.LOCK_UN); f.Close() }, nil
}
