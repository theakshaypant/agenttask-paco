# Task: `paco-diff`

**File:** [`.tekton/tasks/paco-diff.yaml`](../../.tekton/tasks/paco-diff.yaml)
**Image:** `ghcr.io/chmouel/agents-image:latest` (real, unmodified, upstream
`paco-cli` — this repo owns no image for this task)

## What it does

Runs `paco-cli`'s own `diff` subcommand unchanged. It fetches into the
shared `source` workspace:

- `.pr.diff` — the pull request's unified diff.
- `.existing-feedback.txt` — any prior review comments already posted (so
  the model doesn't repeat itself across `/paco review` re-runs).
- `.tekton/ai/REVIEW.md` — this repository's own project-specific review
  rules (see the file at the repo root), if present.
- `.toolchain-versions` — detected language/toolchain versions.
- `.paco-failed` / an empty `.pr.diff` — when there's nothing reviewable
  (for example, a diff limited to generated files).

## Params

| Name | Description |
|---|---|
| `repo_owner` | GitHub org/user owning the repo. |
| `repo_name` | Repository name. |
| `pull_request_number` | PR number to diff. |
| `comment_id` | GitHub comment ID from the triggering `issue_comment` webhook (empty for `pull_request` events). |
| `git_auth_secret` | Name of the Kubernetes Secret PaC injects with a `git-provider-token` key. |

## Build/push

Not applicable — this task always uses the real upstream
`ghcr.io/chmouel/agents-image:latest`. Do not build or push a
repo-specific replacement for it.
