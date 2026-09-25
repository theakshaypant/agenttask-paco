# agenttask-paco

Proof-of-concept [AgentTask](https://github.com/openshift-pipelines/agenttask) + [Pipelines-as-Code](https://pipelinesascode.com) integration. Replaces the `review` step of [`.tekton/paco.yaml`](https://github.com/tektoncd/pipelines-as-code/blob/main/.tekton/paco.yaml) with an `AgentTask` CustomRun that routes analysis through OpenShift Lightspeed instead of `opencode`/Vertex.

## What it does

Reuses the real `paco-cli` binary for GitHub integration (`diff`/`post` steps) and replaces only the LLM analysis engine:
- `paco diff` fetches PR diff + existing feedback
- `build-request` (new) renders the prompt from workspace artifacts
- `review` runs as an `AgentTask` CustomRun ([agenttask](https://github.com/openshift-pipelines/agenttask) + [agenttask-adapter-lightspeed](https://github.com/openshift-pipelines/agenttask-adapter-lightspeed))
- `translate-result` (new) maps the `AnalysisResult` to paco-cli's schema
- `paco post` publishes the review comment

From PaC's perspective: same triggers (`/paco review`), same output (sticky summary + inline comments), same `.tekton/ai/REVIEW.md` rules file. The analysis engine is a Tekton `CustomRun` instead of an in-Task LLM call.

## Architecture

```text
paco diff (paco-cli)
  → build-request
  → review (AgentTask CustomRun)
  → translate-result
  → paco post (paco-cli)
```

| Step | What it does |
|---|---|
| `paco diff` | Unmodified `paco-cli`. Fetches PR diff, existing feedback, `.tekton/ai/REVIEW.md`, toolchain versions. |
| `build-request` | Renders the prompt from workspace artifacts. Detects skip conditions. Enforces 32KB limit. |
| `review` | `AgentTask` CustomRun. Reconciled by `agenttask` + `agenttask-adapter-lightspeed`. Creates an `AnalysisResult`. |
| `translate-result` | Fetches the `AnalysisResult` (dynamic client) and writes `.paco-review.json` (paco-cli schema). |
| `paco post` | Unmodified `paco-cli`. Posts the sticky summary + inline comments to GitHub. |

See [`docs/`](docs/README.md) for per-task breakdown and image build instructions.

## Prerequisites

Cluster-level (one-time install):
- Tekton Pipelines v0.44+ (`CustomRun` support)
- [agenttask](https://github.com/openshift-pipelines/agenttask) controller + CRD
- [agenttask-adapter-lightspeed](https://github.com/openshift-pipelines/agenttask-adapter-lightspeed) controller
- OpenShift Lightspeed Agentic Operator with a cluster-scoped `Agent/tekton-analysis` backed by an `LLMProvider`
- This repo's `config/` manifests: `kubectl apply -k config/` (installs the `pr-review` `AgentTask` CR + RBAC)

Per-repository:
- Copy [`.tekton/`](.tekton/) to your repo
- PaC auto-resolves tasks on webhook trigger
- Reuses existing `{{git_auth_secret}}` — no new secrets needed

## Developer experience

Same as `.tekton/paco.yaml` today:
- Trigger: `/paco review` or `/paco summary` PR comments
- Output: sticky summary + inline code comments
- Config: `.tekton/ai/REVIEW.md` rules file
- New: reviews appear as `CustomRun` objects (`kubectl get runs`)

## Development

```console
make build   # Build cmd/build-request and cmd/translate-result
make test    # Unit tests (offline, fake clients)
make lint    # golangci-lint
make check   # lint + test
```

## Demo (real PaC integration)

Tested via actual PaC webhook flow, not synthetic e2e:

1. Install PaC on local Kind using the upstream dev environment (see `pipelines-as-code` repo's `docs/content/docs/dev/testing.md`)
2. Forward webhooks: `gosmee client "$TEST_GITEA_SMEEURL" http://controller.paac-127-0-0-1.nip.io`
3. Trigger via PR comment (`/paco review`) or PR event
4. Observe `CustomRun` reconciliation via `kubectl get runs`

Unit tests remain fully offline.

## Scope & Limitations

**Per-line comments are prompt-driven, not schema-enforced**  
Lightspeed's `AnalysisResult` has no native per-line field. `build-request` asks the model to append a ` ```paco-review ` JSON block with `comments[]`. `translate-result` extracts it via `promptbuild.ExtractEmbedded`. Falls back to summary-only if the model omits it.

**Model selection is cluster-wide**  
The `analysis-v1` profile pins a single `Agent/tekton-analysis`. Model choice happens at agent config time (cluster admin sets the `LLMProvider`), not per-pipeline like paco-cli's `model` param.

**Single review flow**  
Detects "review" vs "summary" mode from trigger comment. No multi-role registry or custom profiles.

**Experimental dependencies**  
- `agenttask` v0.0.0-poc.1
- `agenttask-adapter-lightspeed` (untagged)
- Lightspeed Agentic Operator (pre-release API)

Uses dynamic clients to avoid pinning unstable APIs.

**Artifact schema coupling**  
`pkg/artifact` re-declares paco-cli's internal schema (unexported). If paco-cli changes workspace filenames, this must update to match.

## What this validates

AgentTask as a Tekton primitive:
- AI agent runs appear as `CustomRun` objects with Tekton-native reconciliation
- RBAC + audit through standard Kubernetes mechanisms
- Workspace sharing via Tekton's existing propagation model
- Agentic steps compose with regular `Task`s in a single pipeline

The adapter pattern (Tekton workspace ↔ agent runtime) generalizes beyond Lightspeed — same approach works for Langchain, custom agentic frameworks, etc.
