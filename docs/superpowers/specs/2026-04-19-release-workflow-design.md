# Release workflow: `go install` and semver tags

## Goal

Let users install the CLI with the Go toolchain from GitHub using **`go install`**, with **`@latest` resolving to the highest semver tag** (not arbitrary `main` commits). Add **`bb version`** so installs can be verified. Add **CI on `v*` tags** so every release tag is automatically tested.

## Assumptions

- The module **`github.com/zoldyzdk/bb-cli`** stays **public** on GitHub so the module proxy and `go install` work without extra auth for end users.
- The **`main` package remains at the repository root** so the install path stays `github.com/zoldyzdk/bb-cli` with no `/cmd/...` suffix.
- **Releases are defined only by pushed Git tags** matching SemVer as expected by Go: `vMAJOR.MINOR.PATCH` (for example `v0.2.0`, `v1.0.0`).

## Publisher workflow

1. Merge changes to `main` until the tree is releasable.
2. Choose the next tag by semver rules relative to existing tags.
3. Create an **annotated** tag on the intended commit (recommended for clarity and tooling): `git tag -a v0.2.0 -m "Release v0.2.0"`.
4. Push the tag: `git push origin v0.2.0`.
5. **CI** runs on that tag push (see below). If it fails, fix on `main` and issue a **new patch tag**; do not move existing tags.

## Consumer documentation (README)

Document:

- **Latest semver release:** `go install github.com/zoldyzdk/bb-cli@latest`
- **Pinned version:** `go install github.com/zoldyzdk/bb-cli@v0.2.0` (example)
- **Minimum Go version** required to build (match the `go` directive in `go.mod`).

Optional note: `@latest` chooses the **highest compatible semver tag** for the module path; untagged `main` is not what `@latest` follows when proper `v*` tags exist.

## CI on version tags

Add a **GitHub Actions** workflow that runs when a **`v*`** tag is pushed.

- **Trigger:** `push` with `tags: ['v*']`.
- **Steps:** checkout at the tag, **setup Go** using **`go-version-file: go.mod`** so the workflow tracks the module’s declared toolchain line.
- **Commands:** `go test ./...` and **`go vet ./...`** for a bit of static signal without extra dependencies.

No artifact upload, Goreleaser, or Homebrew in scope unless added later.

## `bb version` command

### Behavior

- Top-level **`bb version`** subcommand, plus **`bb --version`** and **`bb -v`** on the root command with identical output.
- **No Bitbucket workspace/repo** required; it must not call `resolveWorkspaceAndRepo` or need credentials.
- Version information comes from **`runtime/debug.ReadBuildInfo()`** so binaries built with **`go install`** embed module version, Go version, and (when available) VCS metadata from the build.

### Output format

Human-readable lines to **stdout**, suitable for copy-paste in bug reports:

1. **`bb <module version>`** — use `Main.Version` from build info; if empty or missing, print **`unknown`** for the version token.
2. **`go <go version>`** — use `BuildInfo.GoVersion`.
3. If the setting **`vcs.revision`** exists: line **`commit <revision>`** (short display is acceptable if the toolchain provides a short hash; otherwise full hash).
4. If the setting **`vcs.time`** exists: line **`time <vcs.time>`**.

Exit code **0** on success. If **`ReadBuildInfo()`** returns `ok == false`, print a single line explaining that build info is unavailable, still exit **0** (binary is usable; information is just missing).

### Flags

The root command also supports **`bb --version`** and **`bb -v`** (Cobra’s default version flags when `Version` is set). They print the **same lines** as `bb version`, using build info captured at process start for the flag path and the same formatter for the subcommand.

## Testing and verification

- **CI:** tag workflow runs `go test ./...` and `go vet ./...`.
- **Local:** after implementation, `go build -o bb . && ./bb version` should print coherent lines; `go install` from a tag should show the tagged module version when installed via the proxy.

## Out of scope

- Changing major version / `v2` module path rules.
- Binary artifacts, checksum files, or third-party installers.
- Running full CI on every `main` push (only tag workflow is specified here).
