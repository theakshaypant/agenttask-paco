# agenttask-paco

A demo Custom Task–based PR reviewer for [Pipelines-as-Code](https://pipelinesascode.com),
built as a drop-in replacement for the "review" step of
[`.tekton/paco.yaml`](../.tekton/paco.yaml) / [`paco-cli`](https://github.com/pipelines-as-code/paco-cli).

> Status: experimental demo / proof of concept. Not a supported PaC feature.

This is an independent Go module (its own `go.mod`), intentionally isolated from the root
`tekton-pac` repository's build/test/lint scope (which is scoped to
`./pkg/... ./cmd/... ./test/...`), the same way `docs/` is its own module in that repo.

## What this is (and isn't)

- **It is not** a remake of Pipelines-as-Code's built-in
  [`AI analysis`](https://pipelinesascode.com/docs/api/settings/#ai-analysis-settings)
  feature (`pkg/llm`) — that's an unrelated, in-reconciler Go feature.
- **It is not** `paco-cli` itself, and does **not** depend on it as a library, binary, or
  image at build time.
- **It is** a pipeline that reuses the real, unmodified `paco-cli` binary
  (github.com/pipelines-as-code/paco-cli) for its `diff` and `post` steps (fetching the PR
  diff/existing feedback, and posting the sticky summary comment back to GitHub) and
  replaces only the middle `review` step's *engine*: instead of `paco-cli`'s own `opencode`
  + Vertex AI call, the LLM analysis runs as a real Tekton `AgentTask` `CustomRun`, handled
  by [`openshift-pipelines/agenttask`](https://github.com/openshift-pipelines/agenttask) +
  [`agenttask-adapter-lightspeed`](https://github.com/openshift-pipelines/agenttask-adapter-lightspeed)
  + a real OpenShift Lightspeed Agentic Operator backed by a real OpenAI `LLMProvider`.

This also ties back to the CustomRun status/annotation support added to `tekton-pac` in
this same session (`pkg/kubeinteraction/status/task_status.go`): when this pipeline runs
through real Pipelines-as-Code, a failing `review` AgentTask `CustomRun` now correctly
surfaces in PaC's GitHub Check annotations and PR failure comments.

## Architecture

```text
paco diff (real paco-cli)
  -> build-request (ours)
  -> review (AgentTask CustomRun, agenttask + agenttask-adapter-lightspeed)
  -> translate-result (ours)
  -> paco post (real paco-cli)
```

| Step | Binary/CR | What it does |
|---|---|---|
| `paco diff` | real `paco-cli` | Fetches the PR diff, existing review feedback, `.tekton/ai/REVIEW.md` rules, and detected toolchain versions into the shared workspace. |
| `build-request` (`cmd/build-request`) | ours | Reads those artifacts, renders the single free-text prompt sent to the review engine, bounded to the adapter's 32KB limit. Also detects the paco-cli-diff-step "skip" case (no reviewable diff) and short-circuits. |
| `review` | `AgentTask` (`agent.tekton.dev/v1alpha1`) | A Custom Task step handled by the real `agenttask` + `agenttask-adapter-lightspeed` controllers, which create a native OpenShift Lightspeed `AgenticRun` behind the scenes and return `outcome` (`action-required`/`no-action-required`) + `analysis-result-name`. |
| `translate-result` (`cmd/translate-result`) | ours | Fetches the resulting `agentic.openshift.io/v1alpha1 AnalysisResult` object (via a dynamic client) and maps its `diagnosis`/`options` fields into paco-cli's own `.paco-review.json` schema (`summary`, `review_score`, `comments[]`). |
| `paco post` | real `paco-cli` | Reads `.paco-review.json` (now written by `translate-result` instead of `paco-cli`'s own `review` step) and posts/updates the sticky PR summary comment, unmodified. |

See [`docs/`](docs/README.md) for a per-task breakdown and instructions on
building/pushing this repo's own task images to `ghcr.io`.

## Prerequisites

- Tekton Pipelines with `CustomRun` support.
- The `agenttask` CRD + controller and the `agenttask-adapter-lightspeed` controller
  installed (see their own repos for install instructions).
- A real OpenShift Lightspeed Agentic Operator deployment, with a cluster-scoped
  `Agent/tekton-analysis` (the fixed name the `analysis-v1` profile requires) configured
  with analysis-only, read-only tools, and a real `LLMProvider` backed by OpenAI.
- This repo's own `config/` applied to the cluster once, by an operator
  (`kubectl apply -k config/`): only the `pr-review` `AgentTask` CR and RBAC for the
  pipeline `ServiceAccount`. Nothing else needs a separate cluster install:
  `build-request`, `translate-result`, `paco-diff`, and `paco-post` are all ordinary
  `Task`s that live as plain files under [`.tekton/tasks/`](.tekton/tasks/) alongside the
  `PipelineRun` in [`.tekton/pr-review-pipeline.yaml`](.tekton/pr-review-pipeline.yaml) —
  Pipelines-as-Code recursively resolves every `Task`/`Pipeline`/`PipelineRun` it finds
  under a repo's whole `.tekton/` directory automatically at webhook-trigger time, the
  same way this monorepo's own [`.tekton/tasks/`](../.tekton/tasks/) (`goreleaser.yaml`,
  `cache-fetch.yaml`, ...) is referenced by name from `.tekton/release-pipeline.yaml`. Only
  the `review` `AgentTask`/`CustomRun` step must stay a `taskRef` to a real cluster-scoped
  `AgentTask` CR — that's a Custom Task instance, not a resolvable file.

## Install (end-user experience)

Once a cluster admin has installed the prerequisites above, an end user's repository only
needs a copy of this repo's whole [`.tekton/`](.tekton/) directory (`pr-review-pipeline.yaml`
plus `tasks/paco-diff.yaml`, `tasks/build-request.yaml`, `tasks/translate-result.yaml`,
`tasks/paco-post.yaml`) — same as `.tekton/paco.yaml` today, just split across a few files
instead of one. It reuses PaC's standard `{{git_auth_secret}}` for GitHub auth; there's no
separate per-repo secret to create.

`paco-diff`/`paco-post` use the same `ghcr.io/chmouel/agents-image:latest` image
`.tekton/paco.yaml` itself uses (it bundles the real, unmodified `paco-cli` binary); only
the `review` step avoids that image, since `agents-image` also bundles `opencode`/Vertex,
which this repo replaces with the AgentTask/Lightspeed Custom Task instead.

## Development

```console
make build   # go build ./...
make test    # unit tests (offline, fake/dynamic clients — no cluster or API key needed)
make lint    # golangci-lint
make check   # lint + test
```

## Demo: real Pipelines-as-Code integration

Rather than a scripted Kind e2e that renders and applies a `PipelineRun` by hand, this
repo is meant to be exercised through an actual Pipelines-as-Code install driven by a real
GitHub/Gitea webhook — the same dev workflow this monorepo's own contributors use (see
[`docs/content/docs/dev/testing.md`](../docs/content/docs/dev/testing.md)):

1. Install a real PaC on a local Kind cluster from this checkout of `tekton-pac`:

   ```console
   TEST_GITEA_SMEEURL=https://hook.pipelinesascode.com/<your-id> \
     PAC_DIR=$PWD ../tmp/dev/kind/install.sh -p
   ```

2. Forward the webhook relay to the in-cluster controller:

   ```console
   gosmee client --saveDir /tmp/replays "$TEST_GITEA_SMEEURL" http://controller.paac-127-0-0-1.nip.io
   ```

3. Create a `Repository` CR pointing PaC at the git repo hosting this `.tekton/`
   directory (a real GitHub repo, or the install script's own Forgejo instance), then open
   a pull request or comment `/paco review`/`/paco summary` on one — exactly as
   `.tekton/paco.yaml` is exercised today.

Unit tests remain fully offline and never require any of the above.

## Known limitations

- **Per-line inline comments are best-effort, not schema-guaranteed** — Lightspeed's
  `AnalysisResult` itself has no native per-line/file field (only free-text
  `diagnosis.summary`/`options[].diagnosis.summary`), so `build-request`'s prompt (see
  `pkg/promptbuild`) asks the model to append a fenced ` ```paco-review ` JSON block
  (`review_score`, `security_sensitive`, `comments[]`) to its normal prose response.
  `translate-result` (`pkg/analysisresult.ToReview`) extracts that block via
  `promptbuild.ExtractEmbedded` and hands the resulting `comments[]` straight to the
  unmodified `paco post` step, which posts genuine inline PR review comments exactly as it
  does for its own opencode-based reviews. `review_score.rating` likewise comes from the
  embedded block when present, falling back to a default (1 for no-action-required, 4 for
  action-required) otherwise. This depends on the underlying model reliably following the
  fenced-block instructions embedded in free text (not a hard schema the adapter itself
  enforces) — if the model omits or malforms the block, `translate-result` falls back
  gracefully to a comments-free, summary-only review rather than failing the pipeline.
- **Model/provider selection is not per-call** — the `agenttask-adapter-lightspeed`
  `analysis-v1` profile is pinned to a single cluster `Agent/tekton-analysis`; the actual
  model/provider is whatever `LLMProvider` a cluster admin configured for that `Agent`, not
  something this pipeline or its params can select (unlike `.tekton/paco.yaml`'s own
  `model`/`reasoning_effort` params).
- **Single review flow only** — no multi-role registry; "review" vs "summary" mode is
  still detected from the trigger comment (mirroring paco-cli itself), but there's no
  concept of additional custom roles.
- **Experimental upstream dependencies** — `agenttask` (tag `v0.0.0-poc.1`),
  `agenttask-adapter-lightspeed` (untagged), and the Lightspeed Agentic Operator's API are
  all PoC-stage with no compatibility guarantees; `agenttask-adapter-lightspeed` also pins
  an unreleased pseudo-version of the Lightspeed API, which is why `pkg/analysisresult`
  uses a dynamic/unstructured client instead of importing that module directly.
- **`pkg/artifact`'s filenames are duplicated, not imported, from paco-cli** — paco-cli's
  `internal/artifact`/`internal/review` packages are unexported, so this repo re-declares
  the same filenames/schema by hand; if paco-cli changes them, this repo must be updated
  to match.
