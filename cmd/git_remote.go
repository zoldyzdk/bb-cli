package cmd

import (
	"net/url"
	"os/exec"
	"regexp"
	"strings"
)

var sshRemotePattern = regexp.MustCompile(`git@bitbucket\.org:([^/]+)/(.+?)(?:\.git)?$`)

func inferFromGitRemote() (workspace string, repo string) {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return "", ""
	}

	remote := strings.TrimSpace(string(out))
	if remote == "" {
		return "", ""
	}

	if strings.HasPrefix(remote, "git@") {
		matches := sshRemotePattern.FindStringSubmatch(remote)
		if len(matches) == 3 {
			return matches[1], matches[2]
		}
		return "", ""
	}

	u, err := url.Parse(remote)
	if err != nil {
		return "", ""
	}

	if !strings.Contains(u.Host, "bitbucket.org") {
		return "", ""
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return "", ""
	}

	repoSlug := parts[1]
	repoSlug = strings.TrimSuffix(repoSlug, ".git")

	return parts[0], repoSlug
}
