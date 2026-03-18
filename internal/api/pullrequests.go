package api

import (
	"fmt"

	"github.com/zoldyzdk/bb-cli/internal/models"
)

func repoPath(workspace, repo string) string {
	return fmt.Sprintf("/repositories/%s/%s", workspace, repo)
}

func (c *Client) ListPullRequests(workspace, repo, state string, limit int) ([]models.PullRequest, error) {
	path := fmt.Sprintf("%s/pullrequests?state=%s&pagelen=%d", repoPath(workspace, repo), state, limit)

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
