package models

import "time"

type Link struct {
	Href string `json:"href"`
	Name string `json:"name,omitempty"`
}

type PRLinks struct {
	Self     Link `json:"self"`
	HTML     Link `json:"html"`
	Commits  Link `json:"commits"`
	Approve  Link `json:"approve"`
	Diff     Link `json:"diff"`
	DiffStat Link `json:"diffstat"`
	Comments Link `json:"comments"`
	Activity Link `json:"activity"`
	Merge    Link `json:"merge"`
	Decline  Link `json:"decline"`
}

type User struct {
	DisplayName string `json:"display_name"`
	UUID        string `json:"uuid"`
	Nickname    string `json:"nickname"`
	AccountID   string `json:"account_id"`
	Type        string `json:"type"`
}

type Branch struct {
	Name string `json:"name"`
}

type Repository struct {
	Type     string `json:"type"`
	FullName string `json:"full_name"`
	Name     string `json:"name"`
	UUID     string `json:"uuid"`
}

type Endpoint struct {
	Branch     Branch      `json:"branch"`
	Repository *Repository `json:"repository,omitempty"`
}

type RenderedContent struct {
	Raw    string `json:"raw"`
	Markup string `json:"markup"`
	HTML   string `json:"html"`
}

type Rendered struct {
	Title       RenderedContent `json:"title"`
	Description RenderedContent `json:"description"`
	Reason      RenderedContent `json:"reason"`
}

type Summary struct {
	Raw    string `json:"raw"`
	Markup string `json:"markup"`
	HTML   string `json:"html"`
}

type MergeCommit struct {
	Hash string `json:"hash"`
}

type Participant struct {
	User     User   `json:"user"`
	Role     string `json:"role"`
	Approved bool   `json:"approved"`
	State    string `json:"state"`
}

type PullRequest struct {
	Type              string       `json:"type"`
	ID                int          `json:"id"`
	Title             string       `json:"title"`
	Description       string       `json:"description"`
	State             string       `json:"state"`
	Author            User         `json:"author"`
	Source            Endpoint     `json:"source"`
	Destination       Endpoint     `json:"destination"`
	MergeCommit       *MergeCommit `json:"merge_commit,omitempty"`
	CommentCount      int          `json:"comment_count"`
	TaskCount         int          `json:"task_count"`
	CloseSourceBranch bool         `json:"close_source_branch"`
	ClosedBy          *User        `json:"closed_by,omitempty"`
	Reason            string       `json:"reason"`
	CreatedOn         time.Time    `json:"created_on"`
	UpdatedOn         time.Time    `json:"updated_on"`
	Reviewers         []User       `json:"reviewers"`
	Participants      []Participant `json:"participants"`
	Links             PRLinks      `json:"links"`
	Rendered          Rendered     `json:"rendered"`
	Summary           Summary      `json:"summary"`
	Draft             bool         `json:"draft"`
}

type CreatePullRequestBody struct {
	Title             string    `json:"title"`
	Description       string    `json:"description,omitempty"`
	Source            Endpoint  `json:"source"`
	Destination       *Endpoint `json:"destination,omitempty"`
	CloseSourceBranch bool      `json:"close_source_branch,omitempty"`
	Reviewers         []User    `json:"reviewers,omitempty"`
	Draft             bool      `json:"draft,omitempty"`
}

type CommentContent struct {
	Raw    string `json:"raw"`
	Markup string `json:"markup"`
	HTML   string `json:"html"`
}

type CommentInline struct {
	From *int   `json:"from,omitempty"`
	To   *int   `json:"to,omitempty"`
	Path string `json:"path"`
}

type CommentResolution struct {
	Type      string    `json:"type"`
	User      User      `json:"user"`
	CreatedOn time.Time `json:"created_on"`
}

type Comment struct {
	ID         int                `json:"id"`
	Content    CommentContent     `json:"content"`
	CreatedOn  time.Time          `json:"created_on"`
	UpdatedOn  time.Time          `json:"updated_on"`
	User       User               `json:"user"`
	Inline     *CommentInline     `json:"inline,omitempty"`
	Parent     *Comment           `json:"parent,omitempty"`
	Deleted    bool               `json:"deleted"`
	Resolution *CommentResolution `json:"resolution,omitempty"`
}

type PaginatedResponse[T any] struct {
	Size     int    `json:"size"`
	Page     int    `json:"page"`
	PageLen  int    `json:"pagelen"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Values   []T    `json:"values"`
}

type APIError struct {
	Error struct {
		Message string `json:"message"`
		Detail  string `json:"detail,omitempty"`
	} `json:"error"`
}
