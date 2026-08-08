# PR Status Command Design

## Goal

Add `bb pr status <pr-id>` to summarize whether a Bitbucket pull request appears ready to merge and what is missing when it is not ready.

The command should use Bitbucket as the source of truth where possible. It should infer merge requirements from repository branch restrictions when the authenticated user has permission, and it should support CLI overrides for requirements that cannot be inferred or that the user wants to make stricter.

## User Experience

Default usage:

```bash
bb pr status 123
```

Default output is human-readable and exits `0` when the command itself succeeds. It does not fail the shell simply because the PR is not ready.

Example:

```text
PR #123 Add search feature

Merge readiness: NOT READY

[OK]      State: OPEN
[OK]      Draft: no
[FAIL]    Approvals: 1/2
[FAIL]    Comments: 3 unresolved
[OK]      Tasks: 0 open
[UNKNOWN] Builds: branch rules unavailable
[OK]      Conflicts: none

Missing:
- 1 more approval required
- Resolve 3 open comments
```

Supported flags:

```text
--required-approvals int
--required-default-reviewer-approvals int
--required-successful-builds int
--ignore-comments
--ignore-tasks
--ignore-builds
--fail-on-not-ready
--json
```

Manual requirement flags override inferred branch restriction requirements. Ignore flags remove that category from readiness evaluation, while still allowing the command to print available observed facts if useful.

`--fail-on-not-ready` exits `1` only when the readiness decision is `NOT READY`. Without it, `bb pr status` is an inspection command.

## Readiness States

The command has three top-level readiness states:

- `READY`: all required checks pass.
- `NOT READY`: at least one required check fails.
- `UNKNOWN`: no required check failed, but at least one required rule could not be evaluated.

The command must not report `READY` when Bitbucket requirements are unavailable and no explicit override was supplied for that category.

## Checks

Initial v1 checks:

- PR state is `OPEN`.
- PR is not draft.
- PR has no merge conflicts.
- Required approval count is satisfied.
- Required default reviewer approval count is satisfied when inferable.
- No unresolved comments.
- No open tasks.
- Required successful build/status count is satisfied when inferable.

The readiness output should separate observed facts from inferred requirements. For example, if branch restrictions cannot be read, the command can report the number of observed approvals but should mark the required threshold as unknown unless the user provided `--required-approvals`.

## Bitbucket API Feasibility

The existing project already has these pieces:

- `GetPullRequest` via `GET /repositories/{workspace}/{repo_slug}/pullrequests/{pull_request_id}`.
- `ListPullRequestComments` via `GET /repositories/{workspace}/{repo_slug}/pullrequests/{pull_request_id}/comments`.
- `PullRequest` fields for `state`, `draft`, `task_count`, `reviewers`, and `participants`.
- `Participant.Approved` for approval counting.
- `Comment.Resolution` for unresolved comment counting.

The implementation should add API coverage for:

- `GET /repositories/{workspace}/{repo_slug}/pullrequests/{pull_request_id}/tasks` to count open tasks directly.
- `GET /repositories/{workspace}/{repo_slug}/pullrequests/{pull_request_id}/statuses` to summarize build/status results.
- `GET /repositories/{workspace}/{repo_slug}/pullrequests/{pull_request_id}/conflicts` to detect merge conflicts.
- `GET /repositories/{workspace}/{repo_slug}/branch-restrictions` to infer destination branch merge checks when accessible.

Branch restriction access can require stronger repository permissions than normal PR reads. If that API returns an authorization error, `bb pr status` should degrade gracefully: print observable status, mark inferred requirements as unavailable, and produce `UNKNOWN` unless manual requirement flags cover the unavailable checks.

If branch restrictions are accessible and no matching restriction applies to the PR destination branch, that category has no inferred threshold. That is different from an unavailable category and should not force `UNKNOWN`.

References:

- Bitbucket Pull Requests REST API: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-pullrequests/
- Bitbucket Branch Restrictions REST API: https://developer.atlassian.com/cloud/bitbucket/rest/api-group-branch-restrictions/

## Decisions

Validated during design brainstorming (2026-08-08):

- **Scope:** full v1 — all checks, branch-restriction inference, soft `UNKNOWN`, all flags, and `--json`.
- **Structure:** Approach A — thin focused API methods; `cmd/pr_status.go` orchestrates fetch and builds the snapshot; `internal/status` owns evaluate + human/JSON formatting from day one.
- **Not chosen:** API facade that returns a full status context; parallel fetch (`errgroup`) in v1.

## Internal Structure

Follow the existing Cobra and API patterns:

- Add `cmd/pr_status.go` for command registration, flags, API orchestration, snapshot assembly, and exit-code handling. No readiness logic or formatting beyond calling `internal/status`.
- Extend `internal/api/pullrequests.go` with status-specific PR endpoints (tasks, statuses, conflicts). Keep methods focused — no status orchestration in the API layer.
- Add a branch restrictions API method, either in `internal/api/pullrequests.go` if kept small or in a focused `internal/api/branch_restrictions.go`.
- Extend `internal/models/pullrequest.go` or add focused model files for tasks, build statuses, conflicts, branch restrictions, and optional mergeability fields.
- Add `internal/status` with pure types and functions: `Snapshot`, `Requirements`, `Check`, `Result`, `Evaluate`, `FormatHuman`, `FormatJSON`. No network and no Cobra dependencies.

## Data Flow

1. Resolve workspace and repo using the existing resolution order (flags → config → git remote).
2. Load credentials using the existing config flow.
3. Hard-fetch the PR by ID (command fails if this fails).
4. Fetch comments, tasks, statuses, conflicts, and branch restrictions. Optional endpoint or branch-restriction auth failures are soft errors.
5. Convert Bitbucket responses into a small internal `Snapshot` (observed facts).
6. Resolve `Requirements` from branch restrictions plus CLI overrides / ignore flags.
7. `status.Evaluate(snapshot, requirements)` → per-check `OK|FAIL|UNKNOWN|INFO` and top-level `READY|NOT_READY|UNKNOWN`.
8. Render via `FormatHuman` or `FormatJSON`.
9. Apply exit behavior. Default is `0` on command success; `--fail-on-not-ready` returns `1` only for `NOT READY`.

Ignore flags remove that category from readiness evaluation while still allowing observed facts to be printed when useful. No matching branch restriction for the destination branch means no inferred threshold for that category — that is not the same as unavailable and must not force `UNKNOWN`.

## Error Handling

Hard errors:

- Invalid PR ID.
- Missing workspace/repo.
- Missing credentials.
- Failure to fetch the target PR.

Soft errors:

- Branch restrictions unavailable due to permissions.
- Optional check endpoints unavailable while the core PR was fetched.

Soft errors should appear as `[UNKNOWN]` lines and warnings in human output. They should appear as structured warnings in JSON output.

## JSON Output

`--json` should return a stable structure suitable for scripts:

```json
{
  "pull_request": {
    "id": 123,
    "title": "Add search feature",
    "state": "OPEN"
  },
  "readiness": "NOT_READY",
  "checks": [
    {
      "name": "approvals",
      "status": "FAIL",
      "observed": 1,
      "required": 2,
      "message": "1 more approval required"
    }
  ],
  "missing": [
    "1 more approval required"
  ],
  "warnings": []
}
```

JSON status values should be stable uppercase strings: `OK`, `FAIL`, `UNKNOWN`, and `INFO`. Use `INFO` for observed facts that are not part of readiness (for example an ignored category still printed for context).

JSON readiness values should be stable uppercase strings: `READY`, `NOT_READY`, and `UNKNOWN`. Human output may show spaced labels (`NOT READY`); JSON always uses the underscore form.

## Testing

Primary coverage lives in `internal/status` (no network):

- Readiness evaluation for `READY`, `NOT READY`, and `UNKNOWN`.
- Approval counting from `participants`.
- Default reviewer approval counting when branch restrictions identify that requirement.
- Branch restriction matching for the PR destination branch.
- Unresolved comment counting.
- Open task counting.
- Build/status threshold evaluation.
- Human output for the three readiness states.
- JSON output shape for scripts.

Optional: decode fixtures for new API model types. Prefer testing evaluate/format over full Cobra e2e. Live Bitbucket API verification stays manual because credentials, repository rules, and branch restriction permissions vary by workspace.

## Non-Goals For V1

- Actually merging the PR.
- Reproducing every Bitbucket merge rule with perfect fidelity when the API does not expose enough information.
- Editing branch restrictions or repository settings.
- Requiring branch restriction permissions for basic status output.
