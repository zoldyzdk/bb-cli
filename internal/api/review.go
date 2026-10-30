package api

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/zoldyzdk/bb-cli/internal/models"
)

func (c *Client) ListAllOpenPullRequests(workspace, repo string) ([]models.PullRequest, error) {
	path := repoPath(workspace, repo) + "/pullrequests?state=OPEN&pagelen=50&sort=created_on"
	var prs []models.PullRequest
	seen := make(map[string]bool)
	for path != "" {
		if seen[path] {
			return nil, fmt.Errorf("API pagination loop")
		}
		seen[path] = true
		var page models.PaginatedResponse[models.PullRequest]
		if err := c.Get(path, &page); err != nil {
			return nil, err
		}
		prs = append(prs, page.Values...)
		path = page.Next
	}
	return prs, nil
}

func (c *Client) GetSourceFile(workspace, repo, commit, file string) (string, error) {
	return c.GetRaw(repoPath(workspace, repo) + "/src/" + url.PathEscape(commit) + "/" + file)
}

var reviewCommitHash = regexp.MustCompile(`^[a-fA-F0-9]{7,40}$`)
var reviewRepositorySlug = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)

// GetPinnedPullRequest expands the abbreviated hashes returned in PR metadata.
// Git fetch needs the full object ID, not a branch tip that may change mid-review.
func (c *Client) GetPinnedPullRequest(workspace, repo string, id int) (*models.PullRequest, error) {
	pr, err := c.GetPullRequest(workspace, repo, id)
	if err != nil {
		return nil, err
	}
	sourceWorkspace, sourceRepo := workspace, repo
	if pr.Source.Repository != nil && pr.Source.Repository.FullName != "" {
		parts := strings.Split(pr.Source.Repository.FullName, "/")
		if len(parts) != 2 || !reviewRepositorySlug.MatchString(parts[0]) || !reviewRepositorySlug.MatchString(parts[1]) {
			return nil, fmt.Errorf("invalid PR source repository")
		}
		sourceWorkspace, sourceRepo = parts[0], parts[1]
	}
	pr.Source.Commit.Hash, err = c.resolveReviewCommit(sourceWorkspace, sourceRepo, pr.Source.Commit.Hash)
	if err != nil {
		return nil, err
	}
	pr.Destination.Commit.Hash, err = c.resolveReviewCommit(workspace, repo, pr.Destination.Commit.Hash)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

func (c *Client) resolveReviewCommit(workspace, repo, hash string) (string, error) {
	if !reviewCommitHash.MatchString(hash) {
		return "", fmt.Errorf("invalid PR commit hash")
	}
	if len(hash) == 40 {
		return hash, nil
	}
	var commit models.MergeCommit
	if err := c.Get(repoPath(workspace, repo)+"/commit/"+hash, &commit); err != nil {
		return "", err
	}
	if len(commit.Hash) != 40 || !reviewCommitHash.MatchString(commit.Hash) || !strings.HasPrefix(strings.ToLower(commit.Hash), strings.ToLower(hash)) {
		return "", fmt.Errorf("resolved PR commit does not match pinned hash")
	}
	return commit.Hash, nil
}
