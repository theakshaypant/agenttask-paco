# Task: `translate-result`

**File:** [`.tekton/tasks/translate-result.yaml`](../../.tekton/tasks/translate-result.yaml)
**Image:** `ghcr.io/theakshaypant/agenttask-paco/translate-result:latest`
(this repo's own image — see [building-and-pushing-images.md](../building-and-pushing-images.md))
**Source:** [`cmd/translate-result`](../../cmd/translate-result)

## What it does

Fetches the `agentic.openshift.io/v1alpha1 AnalysisResult` object the
`review` AgentTask step created (via a Kubernetes dynamic client — the
upstream typed client is pinned to an unreleased pseudo-version, see
`pkg/analysisresult`) and maps its `diagnosis`/`options` fields into
`paco-cli`'s own `.paco-review.json` schema (`pkg/review`), including any
embedded fenced `` ```paco-review `` per-line comment block
(`pkg/promptbuild.ExtractEmbedded`), so the unmodified `paco post` step can
publish it unchanged.

## Params

| Name | Description |
|---|---|
| `skip` | `"true"` short-circuits this task (matches `build-request`'s `skip` result). |
| `outcome` | `action-required` or `no-action-required`, from the `review` AgentTask's results. |
| `analysis-result-name` | Name of the `AnalysisResult` object to fetch. |

## RBAC

Requires the `ServiceAccount` running the `PipelineRun` to be bound to
[`config/rbac.yaml`](../../config/rbac.yaml)'s `Role` (`get` on
`agentic.openshift.io/v1alpha1 analysisresults`).

## Build/push

```sh
# From the repo root
ko build --bare -t latest --push=false ./cmd/translate-result   # local build, no push
KO_DOCKER_REPO=ghcr.io/theakshaypant/agenttask-paco/translate-result \
  ko build --bare -t latest --platform=linux/amd64,linux/arm64 ./cmd/translate-result
```

See [building-and-pushing-images.md](../building-and-pushing-images.md) for
authentication and CI details.
