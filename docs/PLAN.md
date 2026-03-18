# bb-cli -- A Bitbucket CLI (like `gh` for GitHub)

## Overview

A Go CLI tool built with [Cobra](https://cobra.dev/docs/) that interacts with the [Bitbucket Cloud REST API v2.0](https://developer.atlassian.com/cloud/bitbucket/rest). The MVP focuses on three pillars: **authentication**, **pull request management**, and **PR comment retrieval**.

## Architecture

```
                          +---------------------+
                          |     bb CLI (Cobra)   |
                          +----------+----------+
                                     |
                    +----------------+----------------+
                    |                                  |
             +------+------+                   +------+------+
             |  auth       |                   |  pr         |
             +------+------+                   +------+------+
                    |                                  |
             +------+------+              +------------+------------+
             | login|status|              | list | create| view | comments|
             +------+------+              +-----+-------+------+--------+
                    |                                  |
             +------+------+                   +------+------+
             | Config Mgr  |                   | HTTP Client |
             +------+------+                   +------+------+
                    |                                  |
      +-------------+-------------+            +------+------+
      | ~/.config/bb-cli/config.json|          | Bitbucket   |
      +---------------------------+            | REST API    |
                                               +-------------+
```

## Project Structure

```
bb-cli/
  main.go                    # Entry point
  go.mod / go.sum            # Go module files
  PLAN.md                    # This file
  cmd/
    root.go                  # Root command, global flags (--workspace, --repo)
    auth.go                  # `bb auth` parent command
    auth_login.go            # `bb auth login` -- store credentials
    auth_status.go           # `bb auth status` -- verify token
    pr.go                    # `bb pr` parent command
    pr_list.go               # `bb pr list` -- list open/merged/declined PRs
    pr_create.go             # `bb pr create` -- create a new PR
    pr_view.go               # `bb pr view <id>` -- show PR details
    pr_comments.go           # `bb pr comments <id>` -- list PR comments
    git_remote.go            # Infer workspace/repo from git remote URL
  internal/
    api/
      client.go              # HTTP client wrapper (base URL, auth header, error handling)
      pullrequests.go        # PR-specific API calls (list, create, get, comments)
    config/
      config.go              # Read/write credentials + defaults from config file
    models/
      pullrequest.go         # Go structs for PR, Comment, Author, Branch, etc.
```

## Authentication Strategy

Bitbucket supports **API Tokens** (successor to app passwords). The user authenticates via HTTP Basic Auth where:

- **Username** = Atlassian account email
- **Password** = API token (created at Bitbucket settings)

The `bb auth login` command prompts for these interactively and stores them in `~/.config/bb-cli/config.json`. The token is never logged or echoed to stdout (input is masked using `golang.org/x/term`).

Reference: [Bitbucket Auth Docs](https://developer.atlassian.com/cloud/bitbucket/rest/intro/#authentication) -- "To authenticate with an API token, use Basic HTTP Authentication as per RFC-2617, where the username is your Atlassian email and password is the API token."

## Bitbucket API Endpoints (MVP)

Base URL: `https://api.bitbucket.org/2.0`

| Operation          | Method | Endpoint                                                              |
|--------------------|--------|-----------------------------------------------------------------------|
| Verify Auth        | GET    | `/user`                                                               |
| List PRs           | GET    | `/repositories/{workspace}/{repo_slug}/pullrequests?state=OPEN`       |
| Create PR          | POST   | `/repositories/{workspace}/{repo_slug}/pullrequests`                  |
| Get PR             | GET    | `/repositories/{workspace}/{repo_slug}/pullrequests/{id}`             |
| List PR Comments   | GET    | `/repositories/{workspace}/{repo_slug}/pullrequests/{id}/comments`    |

### Create PR -- Minimum Request Body

```json
{
  "title": "My Title",
  "source": {
    "branch": {
      "name": "feature-branch"
    }
  }
}
```

Optional fields: `destination`, `description`, `reviewers`, `draft`, `close_source_branch`.

## MVP Commands

### `bb auth login`

Prompts for Bitbucket username (email) and API token. Stores them in config file. Validates credentials by calling `GET /user`.

```
$ bb auth login
Bitbucket username (email): user@example.com
Bitbucket API token: ****
Validating credentials... authenticated as John Doe
Credentials saved to ~/.config/bb-cli/config.json
```

### `bb auth status`

Reads stored credentials and calls `GET /user` to confirm they are still valid. Prints the authenticated user's display name.

```
$ bb auth status
Logged in as: John Doe
Username:     user@example.com
Workspace:    my-workspace
Repository:   my-repo
```

### `bb pr list`

Lists pull requests for the configured workspace/repo.

Flags:
- `--state` -- OPEN (default), MERGED, DECLINED, SUPERSEDED
- `--limit` -- Maximum results (default 25)

```
$ bb pr list --state OPEN
ID    TITLE                          AUTHOR      SOURCE              STATE
123   Fix login redirect             Jane Doe    fix/login-redirect  OPEN
124   Add user profile page          John Doe    feat/user-profile   OPEN
```

### `bb pr create`

Creates a new pull request.

Flags:
- `--title` / `-t` -- PR title (required)
- `--source` / `-s` -- Source branch name (required)
- `--destination` / `-d` -- Destination branch (optional, defaults to main)
- `--description` -- PR description text
- `--reviewer` -- Reviewer UUID (repeatable)
- `--draft` -- Create as draft
- `--close-source-branch` -- Close source branch after merge

```
$ bb pr create --title "Add search feature" --source feat/search --description "Implements full-text search"
Pull request #125 created successfully
Title:  Add search feature
State:  OPEN
URL:    https://bitbucket.org/my-workspace/my-repo/pull-requests/125
```

### `bb pr view <id>`

Displays detailed information about a specific pull request.

```
$ bb pr view 123
#123 Fix login redirect
State:       OPEN
Author:      Jane Doe
Source:      fix/login-redirect
Destination: main
Created:     2026-03-15 10:30:00
Updated:     2026-03-16 14:22:00
Comments:    5
Tasks:       2
Reviewers:   John Doe, Alice Smith

--- Description ---
Fixes the login redirect bug where users were sent to a 404 page.

URL: https://bitbucket.org/my-workspace/my-repo/pull-requests/123
```

### `bb pr comments <id>`

Lists all comments on a pull request.

Flags:
- `--limit` -- Maximum comments to display (default 50)

```
$ bb pr comments 123
--- Comment #1 ---
Author: John Doe
Date:   2026-03-15 11:00:00

Looks good, just one small suggestion on line 42.

--- Comment #2 [src/auth/login.go:42] ---
Author: Alice Smith
Date:   2026-03-15 12:30:00

Can we add a unit test for this redirect logic?
```

## Global Flags

- `--workspace` / `-w` -- Bitbucket workspace slug
- `--repo` / `-r` -- Repository slug

These can also be set in the config file or inferred automatically from the current directory's git remote URL.

## Workspace/Repo Resolution Order

1. `--workspace` / `--repo` CLI flags (highest priority)
2. Values stored in `~/.config/bb-cli/config.json`
3. Parsed from the current directory's git remote origin URL (pattern: `bitbucket.org/{workspace}/{repo}`)

## Config File Format

Location: `~/.config/bb-cli/config.json`

```json
{
  "username": "user@example.com",
  "token": "ATBB...",
  "workspace": "my-workspace",
  "repo": "my-repo"
}
```

## Key Dependencies

| Package                        | Purpose                              |
|--------------------------------|--------------------------------------|
| `github.com/spf13/cobra`      | CLI framework                        |
| `github.com/rodaine/table`    | Pretty table output for PR lists     |
| `golang.org/x/term`           | Secure password/token input          |
| `net/http` + `encoding/json`  | HTTP client and JSON serialization   |

## Error Handling

- All API errors return structured JSON with a `error.message` field -- these are parsed and displayed to the user
- HTTP 401 -- prompts user to run `bb auth login`
- HTTP 404 -- "repository or PR not found"
- Network errors -- clear message with connection details

## Building

```bash
go build -o bb .
```

## Future Enhancements (Post-MVP)

- `bb pr merge <id>` -- Merge a pull request
- `bb pr approve <id>` -- Approve a pull request
- `bb pr decline <id>` -- Decline a pull request
- `bb pr diff <id>` -- View pull request diff
- `bb repo list` -- List repositories in a workspace
- `bb pr comment <id> --body "..."` -- Add a comment to a PR
- Shell completion scripts (bash, zsh, fish)
- JSON output mode (`--json` flag) for scripting
- Colored terminal output
- Pagination support (automatic `next` page fetching)
