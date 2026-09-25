# Task: `paco-post`

**File:** [`.tekton/tasks/paco-post.yaml`](../../.tekton/tasks/paco-post.yaml)
**Image:** `ghcr.io/chmouel/agents-image:latest` (real, unmodified, upstream
`paco-cli` — this repo owns no image for this task)

## What it does

Runs `paco-cli`'s own `post` subcommand unchanged. It reads
`.paco-review.json` (written by `translate-result` instead of `paco-cli`'s
own `review` step) from the shared workspace and posts/updates the sticky
PR review — including genuine inline per-line comments when present,
validated against `.valid-lines.json` — exactly as it would if `paco-cli`
had produced the review itself.

## Params

| Name | Description |
|---|---|
| `repo_owner` | GitHub org/user owning the repo. |
| `repo_name` | Repository name. |
| `pull_request_number` | PR number to post to. |
| `git_auth_secret` | Name of the Kubernetes Secret PaC injects with a `git-provider-token` key. |

## Build/push

Not applicable — this task always uses the real upstream
`ghcr.io/chmouel/agents-image:latest`. Do not build or push a
repo-specific replacement for it.
