package api

import (
	"fmt"

	"github.com/zoldyzdk/bb-cli/internal/models"
)

func (c *Client) ListBranchRestrictions(workspace, repo string, limit int) ([]models.BranchRestriction, error) {
	path := fmt.Sprintf("%s/branch-restrictions?pagelen=%d", repoPath(workspace, repo), limit)
	var result models.PaginatedResponse[models.BranchRestriction]
	if err := c.Get(path, &result); err != nil {
		return nil, err
	}
	return result.Values, nil
}
