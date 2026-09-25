# Task: `build-request`

**File:** [`.tekton/tasks/build-request.yaml`](../../.tekton/tasks/build-request.yaml)
**Image:** `ghcr.io/theakshaypant/agenttask-paco/build-request:latest`
(this repo's own image — see [building-and-pushing-images.md](../building-and-pushing-images.md))
**Source:** [`cmd/build-request`](../../cmd/build-request)

## What it does

Reads the artifacts `paco-diff` wrote to the shared workspace
(`.pr.diff`, `.existing-feedback.txt`, `.tekton/ai/REVIEW.md`,
`.toolchain-versions`) and renders the single free-text prompt string
(`pkg/promptbuild.BuildPrompt`) that becomes the `request` param passed to
the `review` AgentTask step.

When `paco-diff` already decided to skip the run (no reviewable diff), this
command writes the final `.paco-review.json`/`.paco-failed` artifacts
itself — matching `paco-cli`'s own review-step behavior — and reports
`skip=true` so the pipeline bypasses both the `review` AgentTask and
`translate-result` steps.

## Params

| Name | Description |
|---|---|
| `trigger-comment` | The triggering comment text (`/paco review` vs `/paco summary`); determines prompt mode. |

## Results

| Name | Description |
|---|---|
| `request` | The rendered prompt sent to the `review` AgentTask. |
| `skip` | `"true"` when there's nothing to review; gates `review`/`translate-result`. |

## Build/push

```sh
# From the repo root
ko build --bare -t latest --push=false ./cmd/build-request   # local build, no push
KO_DOCKER_REPO=ghcr.io/theakshaypant/agenttask-paco/build-request \
  ko build --bare -t latest --platform=linux/amd64,linux/arm64 ./cmd/build-request
```

See [building-and-pushing-images.md](../building-and-pushing-images.md) for
authentication and CI details.
