package cmd

import (
	"fmt"
	"os"
	"os/exec"
)

func gitFetch(remote, remoteBranch, localBranch string, force bool) error {
	refspec := fmt.Sprintf("%s:%s", remoteBranch, localBranch)
	args := []string{"fetch", remote, refspec}
	if force {
		args = append(args, "--force")
	}

	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git fetch failed: %w", err)
	}
	return nil
}

func gitCheckout(branch string) error {
	cmd := exec.Command("git", "checkout", branch)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git checkout failed: %w", err)
	}
	return nil
}
