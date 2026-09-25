# Step: `review` (AgentTask CustomTask, not a Tekton `Task`)

**File:** [`config/agenttask-pr-review.yaml`](../../config/agenttask-pr-review.yaml)
(the `AgentTask` custom resource) and its `taskRef` in
[`.tekton/pr-review-pipeline.yaml`](../../.tekton/pr-review-pipeline.yaml).

There is no Tekton `Task` or container image for this step, and nothing
here to build or push. It's a `taskRef` to a namespace-scoped
`agent.tekton.dev/v1alpha1 AgentTask` custom resource (pre-installed via
`kubectl apply -k config/`), executed by the `agenttask` controller +
`agenttask-adapter-lightspeed` adapter (installed once, cluster-wide — see
the repo root `README.md`'s "Install" section), which in turn drives the
real OpenShift Lightspeed Agentic Operator against a real LLM provider.

## What it does

Takes the single free-text `request` param (the prompt rendered by
`build-request`) and routes it through the Lightspeed `analysis-v1`
profile — a read-only analysis mode that never takes remediation actions
against the cluster. The prompt asks the model to review the PR diff per
this repo's own `.tekton/ai/REVIEW.md` rules and (best-effort) append a
fenced `` ```paco-review `` JSON block with per-line comments.

## Params

| Name | Description |
|---|---|
| `request` | The rendered prompt string (from `build-request`'s `request` result). |

## Results

| Name | Description |
|---|---|
| `outcome` | `action-required` or `no-action-required`. |
| `analysis-result-name` | Name of the native `agentic.openshift.io/v1alpha1 AnalysisResult` object `translate-result` fetches next. |

## Prerequisites

A cluster-scoped `Agent` named `tekton-analysis` (the fixed name the
`analysis-v1` profile requires), backed by a real `LLMProvider`, plus the
`agenttask` and `agenttask-adapter-lightspeed` controllers installed and
running. See the repo root `README.md` for the full cluster-admin install
steps.
