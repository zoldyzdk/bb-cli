# bb-cli Quick Start

Get up and running with the Bitbucket CLI in a few minutes.

## Install with Go

Requires **Go 1.25** or newer (see `go.mod`).

```bash
go install github.com/zoldyzdk/bb-cli@latest
```

`@latest` resolves to the **highest semver tag** (`v*`) on this repository. To pin a release:

```bash
go install github.com/zoldyzdk/bb-cli@v0.1.0
```

Check the installed build:

```bash
bb version
```

## 1. Build the CLI

```bash
cd bb-cli
go build -o bb .
```

Add the binary to your PATH, or run it directly:

```bash
./bb --help
```

## 2. Create a Bitbucket API Token

1. Go to [Bitbucket Settings > Personal access tokens](https://id.atlassian.com/manage-profile/security/api-tokens)
2. Click **Create API token with scopes**
3. Give it a name (e.g. `bb-cli`)
4. Grant at least: **Pull requests: Read and write**
5. Copy the token (you won't see it again)

## 3. Log In

```bash
./bb auth login
```

You'll be prompted for:

- **Bitbucket username (email)** — your Atlassian account email
- **Bitbucket API token** — the token you just created (input is hidden)

Credentials are saved to `~/.config/bb-cli/config.json`.

## 4. Set Your Workspace and Repo

**Option A: Use flags**

```bash
./bb pr list --workspace my-workspace --repo my-repo
```

**Option B: Edit the config file**

Edit `~/.config/bb-cli/config.json`:

```json
{
  "username": "you@example.com",
  "token": "ATBB...",
  "workspace": "my-workspace",
  "repo": "my-repo"
}
```

**Option C: Run from a Bitbucket git repo**

If you're inside a repo with a Bitbucket remote (e.g. `git@bitbucket.org:workspace/repo.git`), workspace and repo are detected automatically.

## 5. Common Commands

### List pull requests

```bash
./bb pr list
./bb pr list --state MERGED
./bb pr list --state OPEN --limit 10
```

### Create a pull request

```bash
./bb pr create --title "Add search feature" --source feat/search
./bb pr create -t "Fix login bug" -s fix/login --description "Fixes redirect issue"
./bb pr create -t "WIP: New API" -s feat/api --draft
```

### View a pull request

```bash
./bb pr view 123
```

### List comments on a PR

```bash
./bb pr comments 123
./bb pr comments 123 --limit 20
```

## 6. Check Auth Status

```bash
./bb auth status
```

Shows your logged-in user and stored workspace/repo.

## 7. Version

```bash
bb version
```

Prints the embedded module version, Go toolchain version, and VCS metadata when the binary was built with module information (for example via `go install`).

## Troubleshooting

| Problem | Solution |
|--------|----------|
| `not logged in` | Run `bb auth login` |
| `workspace is required` | Use `--workspace` and `--repo`, or set them in config, or run from a Bitbucket git repo |
| `API error (401)` | Token may be expired or revoked. Create a new token and run `bb auth login` again |
| `API error (404)` | Check that workspace and repo names are correct (case-sensitive) |
| `go install` errors about Go version | Upgrade Go to at least the version in this repo’s `go.mod` |
