package api

import (
	"fmt"
	"net/url"

	"github.com/zoldyzdk/bb-cli/internal/models"
)

func repoPath(workspace, repo string) string {
	return fmt.Sprintf("/repositories/%s/%s", workspace, repo)
}

func (c *Client) ListPullRequests(workspace, repo, state string, limit int, query string) ([]models.PullRequest, error) {
	path := fmt.Sprintf("%s/pullrequests?state=%s&pagelen=%d", repoPath(workspace, repo), state, limit)
	if query != "" {
		path += "&q=" + url.QueryEscape(query)
	}

	var result models.PaginatedResponse[models.PullRequest]
	if err := c.Get(path, &result); err != nil {
		return nil, err
	}

	return result.Values, nil
}

func (c *Client) CreatePullRequest(workspace, repo string, body *models.CreatePullRequestBody) (*models.PullRequest, error) {
	path := fmt.Sprintf("%s/pullrequests", repoPath(workspace, repo))

	var pr models.PullRequest
	if err := c.Post(path, body, &pr); err != nil {
		return nil, err
	}

	return &pr, nil
}

func (c *Client) GetPullRequest(workspace, repo string, prID int) (*models.PullRequest, error) {
	path := fmt.Sprintf("%s/pullrequests/%d", repoPath(workspace, repo), prID)

	var pr models.PullRequest
	if err := c.Get(path, &pr); err != nil {
		return nil, err
	}

	return &pr, nil
}

func (c *Client) ListPullRequestComments(workspace, repo string, prID, limit int) ([]models.Comment, error) {
	path := fmt.Sprintf("%s/pullrequests/%d/comments?pagelen=%d", repoPath(workspace, repo), prID, limit)

	var result models.PaginatedResponse[models.Comment]
	if err := c.Get(path, &result); err != nil {
		return nil, err
	}

	return result.Values, nil
}

func (c *Client) GetPullRequestDiff(workspace, repo string, prID int) (string, error) {
	path := fmt.Sprintf("%s/pullrequests/%d/diff", repoPath(workspace, repo), prID)
	return c.GetRaw(path)
}

func (c *Client) ListPullRequestTasks(workspace, repo string, prID, limit int) ([]models.PullRequestTask, error) {
	path := fmt.Sprintf("%s/pullrequests/%d/tasks?pagelen=%d", repoPath(workspace, repo), prID, limit)
	var result models.PaginatedResponse[models.PullRequestTask]
	if err := c.Get(path, &result); err != nil {
		return nil, err
	}
	return result.Values, nil
}

func (c *Client) ListPullRequestStatuses(workspace, repo string, prID, limit int) ([]models.CommitStatus, error) {
	path := fmt.Sprintf("%s/pullrequests/%d/statuses?pagelen=%d", repoPath(workspace, repo), prID, limit)
	var result models.PaginatedResponse[models.CommitStatus]
	if err := c.Get(path, &result); err != nil {
		return nil, err
	}
	return result.Values, nil
}

// ListPullRequestConflicts follows Bitbucket's redirect to the conflict list.
// Empty slice means no conflicts. Caller should soft-fail on error.
func (c *Client) ListPullRequestConflicts(workspace, repo string, prID int) ([]models.FileConflict, error) {
	path := fmt.Sprintf("%s/pullrequests/%d/conflicts", repoPath(workspace, repo), prID)
	var result models.PaginatedResponse[models.FileConflict]
	if err := c.Get(path, &result); err != nil {
		// Some responses may be a bare array — try that if paginated decode fails upstream.
		return nil, err
	}
	return result.Values, nil
}
