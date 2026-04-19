# PR Comments Status Display

Add resolution status display and filtering to the `bb pr comments` command.

## Goal

Show whether each comment is resolved or open, and allow filtering by status. This helps users quickly identify which comments still need attention.

## Design

### Model Changes

Add a new struct and field to `internal/models/pullrequest.go`:

```go
type CommentResolution struct {
    Type      string    `json:"type"`
    User      User      `json:"user"`
    CreatedOn time.Time `json:"created_on"`
}

type Comment struct {
    // ... existing fields ...
    Resolution *CommentResolution `json:"resolution,omitempty"`
}
```

When `Resolution` is `nil`, the comment is open. When populated, the comment is resolved.

### Output Format

The comment header line includes the status label after the comment ID:

**Resolved inline comment:**
```
--- Comment #42 [RESOLVED] [src/main.go:15] ---
Author: John Doe
Date:   2026-04-15 10:30:00

This needs to be refactored.
```

**Open inline comment:**
```
--- Comment #43 [OPEN] [src/main.go:22] ---
Author: Jane Smith
Date:   2026-04-16 14:20:00

Can we add error handling here?
```

**Open general comment (no file location):**
```
--- Comment #44 [OPEN] ---
Author: Jane Smith
Date:   2026-04-16 15:00:00

Overall looks good!
```

Order: `Comment #ID` → `[STATUS]` → `[location]` (if inline)

### Filtering Flags

Two new boolean flags:

| Flag | Behavior |
|------|----------|
| `--resolved` | Show only resolved comments |
| `--unresolved` | Show only unresolved (open) comments |
| (neither) | Show all comments (default) |

**Usage:**
```bash
bb pr comments 123                  # all comments with status
bb pr comments 123 --unresolved     # only open comments
bb pr comments 123 --resolved       # only resolved comments
```

**Edge case:** If both flags are provided, treat as default (show all).

## Files to Modify

| File | Change |
|------|--------|
| `internal/models/pullrequest.go` | Add `CommentResolution` struct and `Resolution` field to `Comment` |
| `cmd/pr_comments.go` | Add `--resolved`/`--unresolved` flags, update output format, add filtering logic |

No changes needed to `internal/api/pullrequests.go` — the API already fetches comments and Bitbucket returns the `resolution` field; we just need to parse it.
