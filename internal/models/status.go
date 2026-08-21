package models

import "time"

type PullRequestTask struct {
	ID        int            `json:"id"`
	State     string         `json:"state"` // UNRESOLVED | RESOLVED
	Content   CommentContent `json:"content"`
	CreatedOn time.Time      `json:"created_on"`
	UpdatedOn time.Time      `json:"updated_on"`
}

type CommitStatus struct {
	Type        string    `json:"type"`
	Key         string    `json:"key"`
	Name        string    `json:"name"`
	State       string    `json:"state"` // SUCCESSFUL | FAILED | INPROGRESS | STOPPED
	Description string    `json:"description"`
	URL         string    `json:"url"`
	CreatedOn   time.Time `json:"created_on"`
	UpdatedOn   time.Time `json:"updated_on"`
}

// FileConflict is one conflicted path from the conflicts endpoint (after redirects).
type FileConflict struct {
	Path string `json:"path"`
}

type BranchRestriction struct {
	ID              int     `json:"id"`
	Kind            string  `json:"kind"`
	Value           *int    `json:"value"`
	BranchMatchKind string  `json:"branch_match_kind"` // glob | branching_model
	Pattern         string  `json:"pattern"`
	BranchType      string  `json:"branch_type"`
	Type            string  `json:"type"`
}
