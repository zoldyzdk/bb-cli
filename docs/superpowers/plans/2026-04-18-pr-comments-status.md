# PR Comments Status Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add resolution status display and filtering to the `bb pr comments` command.

**Architecture:** Extend the Comment model to parse the `resolution` field from Bitbucket API, update the output format to show `[RESOLVED]`/`[OPEN]` labels, and add `--resolved`/`--unresolved` flags for filtering.

**Tech Stack:** Go, Cobra CLI

---

## File Structure

| File | Responsibility |
|------|----------------|
| `internal/models/pullrequest.go` | Add `CommentResolution` struct and `Resolution` field to `Comment` |
| `cmd/pr_comments.go` | Display status labels, add filtering flags and logic |

---

### Task 1: Add CommentResolution Model

**Files:**
- Modify: `internal/models/pullrequest.go:123-132`

- [ ] **Step 1: Add CommentResolution struct and Resolution field**

Add the new struct before the `Comment` struct, and add the field to `Comment`:

```go
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
```

- [ ] **Step 2: Verify the code compiles**

Run: `go build .`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
git add internal/models/pullrequest.go
git commit -m "feat(models): add Resolution field to Comment struct"
```

---

### Task 2: Display Status Labels in Output

**Files:**
- Modify: `cmd/pr_comments.go:58-84`

- [ ] **Step 1: Update the comment header to include status label**

In `runPRComments`, update the loop that prints comments. Replace the header formatting section:

```go
for i, c := range comments {
	if c.Deleted {
		continue
	}

	status := "OPEN"
	if c.Resolution != nil {
		status = "RESOLVED"
	}

	location := ""
	if c.Inline != nil {
		location = fmt.Sprintf(" [%s", c.Inline.Path)
		if c.Inline.To != nil {
			location += fmt.Sprintf(":%d", *c.Inline.To)
		}
		location += "]"
	}

	fmt.Printf("--- Comment #%d [%s]%s ---\n", c.ID, status, location)
	fmt.Printf("Author: %s\n", c.User.DisplayName)
	fmt.Printf("Date:   %s\n", c.CreatedOn.Format("2006-01-02 15:04:05"))

	content := strings.TrimSpace(c.Content.Raw)
	if content != "" {
		fmt.Printf("\n%s\n", content)
	}

	if i < len(comments)-1 {
		fmt.Println()
	}
}
```

- [ ] **Step 2: Verify the code compiles**

Run: `go build .`
Expected: No errors

- [ ] **Step 3: Manual test (optional)**

Run: `./bb pr comments <pr-id>` on a PR with comments
Expected: Each comment shows `[RESOLVED]` or `[OPEN]` in the header

- [ ] **Step 4: Commit**

```bash
git add cmd/pr_comments.go
git commit -m "feat(pr-comments): display resolution status in comment output"
```

---

### Task 3: Add Filtering Flags

**Files:**
- Modify: `cmd/pr_comments.go:13-26` (flags)
- Modify: `cmd/pr_comments.go:50-56` (filtering logic)

- [ ] **Step 1: Add flag variables and register flags**

Add the flag variables near the top of the file (after `prCommentsLimit`):

```go
var prCommentsLimit int
var prCommentsResolved bool
var prCommentsUnresolved bool
```

Update the `init()` function to register the new flags:

```go
func init() {
	prCmd.AddCommand(prCommentsCmd)
	prCommentsCmd.Flags().IntVar(&prCommentsLimit, "limit", 50, "Maximum number of comments to display")
	prCommentsCmd.Flags().BoolVar(&prCommentsResolved, "resolved", false, "Show only resolved comments")
	prCommentsCmd.Flags().BoolVar(&prCommentsUnresolved, "unresolved", false, "Show only unresolved comments")
}
```

- [ ] **Step 2: Add filtering logic before the display loop**

In `runPRComments`, after fetching comments and before the "No comments" check, add filtering:

```go
comments, err := client.ListPullRequestComments(workspace, repo, prID, prCommentsLimit)
if err != nil {
	return fmt.Errorf("failed to list comments: %w", err)
}

// Filter by resolution status if flags are set (but not both)
if prCommentsResolved && !prCommentsUnresolved {
	filtered := make([]models.Comment, 0)
	for _, c := range comments {
		if c.Resolution != nil {
			filtered = append(filtered, c)
		}
	}
	comments = filtered
} else if prCommentsUnresolved && !prCommentsResolved {
	filtered := make([]models.Comment, 0)
	for _, c := range comments {
		if c.Resolution == nil {
			filtered = append(filtered, c)
		}
	}
	comments = filtered
}
// If both flags are set or neither, show all (default behavior)

if len(comments) == 0 {
	fmt.Printf("No comments on PR #%d\n", prID)
	return nil
}
```

- [ ] **Step 3: Add models import if not present**

Ensure the import section includes the models package:

```go
import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/zoldyzdk/bb-cli/internal/api"
	"github.com/zoldyzdk/bb-cli/internal/config"
	"github.com/zoldyzdk/bb-cli/internal/models"
)
```

- [ ] **Step 4: Verify the code compiles**

Run: `go build .`
Expected: No errors

- [ ] **Step 5: Manual test**

Test all three scenarios:
```bash
./bb pr comments <pr-id>              # shows all with status labels
./bb pr comments <pr-id> --resolved   # shows only resolved
./bb pr comments <pr-id> --unresolved # shows only open
```

- [ ] **Step 6: Commit**

```bash
git add cmd/pr_comments.go
git commit -m "feat(pr-comments): add --resolved and --unresolved filter flags"
```

---

## Summary

After completing all tasks:
- Comments display `[RESOLVED]` or `[OPEN]` status in the header
- `--resolved` flag filters to only resolved comments
- `--unresolved` flag filters to only open comments
- Default behavior shows all comments with status labels
