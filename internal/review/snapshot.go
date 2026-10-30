package review

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/zoldyzdk/bb-cli/internal/models"
)

type Snapshot struct {
	Root, Checkout, Control, Diff string
	Truncated                     bool
}
type Snapshotter interface {
	Prepare(context.Context, Enrollment, *models.PullRequest, string) (*Snapshot, error)
}
type GitSnapshotter struct{ LocalURL string }

func (g GitSnapshotter) Prepare(ctx context.Context, enrollment Enrollment, pr *models.PullRequest, token string) (_ *Snapshot, resultErr error) {
	if !commitHash.MatchString(pr.Source.Commit.Hash) || !commitHash.MatchString(pr.Destination.Commit.Hash) {
		return nil, fmt.Errorf("PR has no valid pinned commits")
	}
	root, err := os.MkdirTemp("", "bb-review-")
	if err != nil {
		return nil, err
	}
	defer func() {
		if resultErr != nil {
			os.RemoveAll(root)
		}
	}()
	s := &Snapshot{Root: root, Checkout: filepath.Join(root, "source"), Control: filepath.Join(root, "control")}
	for _, dir := range []string{s.Checkout, s.Control} {
		if err := os.Mkdir(dir, 0700); err != nil {
			return nil, err
		}
	}
	askpass := filepath.Join(s.Control, "git-askpass.sh")
	if err := os.WriteFile(askpass, []byte("#!/bin/sh\ncase \"$1\" in *Username*) printf '%s\\n' 'x-token-auth' ;; *) printf '%s\\n' \"$BB_REVIEW_GIT_TOKEN\" ;; esac\n"), 0700); err != nil {
		return nil, err
	}
	env := append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0", "GIT_ASKPASS="+askpass, "GIT_LFS_SKIP_SMUDGE=1", "BB_REVIEW_GIT_TOKEN="+token)
	git := func(args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, "git", append([]string{"-c", "core.hooksPath=/dev/null", "-c", "core.attributesFile=/dev/null"}, args...)...)
		cmd.Dir = s.Checkout
		cmd.Env = env
		configureProcess(cmd)
		output := &limitedOutput{limit: 1024 * 1024}
		cmd.Stdout = output
		err := cmd.Run()
		if err != nil {
			return "", fmt.Errorf("unable to prepare pinned Git snapshot (%s); verify repository access", args[0])
		}
		if args[0] == "diff" {
			s.Truncated = output.truncated
		}
		return output.String(), nil
	}
	if _, err = git("init", "-q"); err != nil {
		return nil, err
	}
	url := "https://bitbucket.org/" + enrollment.Workspace + "/" + enrollment.Repository + ".git"
	if g.LocalURL != "" {
		url = g.LocalURL
	}
	if _, err = git("remote", "add", "origin", url); err != nil {
		return nil, err
	}
	if _, err = git("fetch", "--no-tags", "--no-recurse-submodules", "origin", pr.Destination.Commit.Hash); err != nil {
		return nil, err
	}
	remote := "origin"
	if pr.Source.Repository != nil && pr.Source.Repository.FullName != "" && pr.Source.Repository.FullName != enrollment.Workspace+"/"+enrollment.Repository && g.LocalURL == "" {
		parts := strings.Split(pr.Source.Repository.FullName, "/")
		if len(parts) != 2 || !ValidRepository(parts[0], parts[1]) {
			return nil, fmt.Errorf("invalid PR source repository")
		}
		remote = "https://bitbucket.org/" + pr.Source.Repository.FullName + ".git"
	}
	if _, err = git("fetch", "--no-tags", "--no-recurse-submodules", remote, pr.Source.Commit.Hash); err != nil {
		return nil, err
	}
	if _, err = git("checkout", "--detach", pr.Source.Commit.Hash); err != nil {
		return nil, err
	}
	diff, err := git("diff", "--no-ext-diff", "--no-textconv", pr.Destination.Commit.Hash+"..."+pr.Source.Commit.Hash, "--")
	if err != nil {
		return nil, err
	}
	s.Diff = diff
	if err := os.WriteFile(filepath.Join(s.Control, "diff.patch"), []byte(diff), 0600); err != nil {
		return nil, err
	}
	return s, nil
}

// Drain stdout while retaining only the configured budget.
type limitedOutput struct {
	bytes.Buffer
	limit     int
	truncated bool
}

func (b *limitedOutput) Write(data []byte) (int, error) {
	n := len(data)
	remaining := b.limit - b.Len()
	if n > remaining {
		b.truncated = true
		data = data[:remaining]
	}
	_, _ = b.Buffer.Write(data)
	return n, nil
}

// Kill the subprocess group on timeout so Git/MCP children cannot outlive a run.
func configureProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}
		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
}
