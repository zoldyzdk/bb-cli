package api

import (
	"fmt"

	"github.com/zoldyzdk/bb-cli/internal/models"
)

func (c *Client) ListDefaultReviewers(workspace, repo string, limit int) ([]models.User, error) {
	path := fmt.Sprintf("%s/default-reviewers?pagelen=%d", repoPath(workspace, repo), limit)
	var result models.PaginatedResponse[models.User]
	if err := c.Get(path, &result); err != nil {
		return nil, err
	}
	return result.Values, nil
}
