# Paco review: direct LLM invocation vs. the AgentTask paradigm

There are two ways to run the **`/paco review`** AI code reviewer. From a
repository's point of view they're interchangeable: same triggers, same
`.tekton/ai/REVIEW.md` rules, same PR output. What changes is **who executes the
agent, and through what contract.**

- **A — Direct invocation** ([`tekton-pac/.tekton/paco.yaml`][paco-yaml]): the
  pipeline runs the agent itself, in-process, as an ordinary build step.
- **B — AgentTask paradigm** ([this repo](.tekton/pr-review-pipeline.yaml)): the
  pipeline *declares* an agentic step as a Tekton `AgentTask` CustomRun, and an
  adapter controller reconciles it out-of-band.

Note that this compares paradigms, not backends. The engine behind an AgentTask
is swappable: this PoC uses OpenShift Lightspeed, but an `opencode` adapter could
run Approach A's exact engine under the same model.

---

## The essential difference

Approach A is imperative. The pipeline contains and runs the agent, so the
engine, model, and credentials all live inside the Task.

Approach B is declarative. The pipeline says "I need an agent to handle this
request, fulfilled by this adapter," and a controller does the rest. The engine
lives behind the adapter and is named only indirectly.

Everything below follows from that one distinction.

---

## Trade-offs

| Dimension | A — Direct invocation | B — AgentTask paradigm |
|---|---|---|
| **Who runs the agent** | The pipeline itself | A controller, out-of-band, on the pipeline's behalf |
| **Contract** | Implicit — whatever the step's script does | Explicit & typed — declared `params` in, scalar `results` out |
| **Coupling to a runtime** | Engine baked into the Task | Engine behind an adapter; pipeline is engine-agnostic |
| **Reuse** | Copy/adapt the Task per pipeline | One `AgentTask` CR, referenced by any pipeline |
| **Swapping the engine** | Rewrite the Task | Point `adapterRef` at a different adapter |
| **The AI run as an object** | No — transient step, logs only | Yes — a `CustomRun`, inspectable like any K8s resource |
| **Audit / RBAC** | Pipeline SA + secret access | Standard K8s RBAC on the run and its results |
| **Who owns the runtime** | Each pipeline author | A platform team, centrally, behind the adapter |
| **Operational weight** | Just the engine image | An agent controller + CRD + an adapter to install and run |
| **Maturity** | Established, production path | PoC — controller `v0.0.0-poc.1`, reference adapter untagged |

The core trade-off: A gives you simplicity and a self-contained pipeline; B gives
you a reusable, governed, runtime-decoupled primitive, and you pay for it with
extra infrastructure to stand up and operate.

### The trade-offs that matter most

Here's what's actually at stake behind that table.

**Delegation and a typed contract.** With classic paco, the pipeline *is* the
integration: it holds the engine and runs it. The AgentTask model puts a
declarative boundary between "I need an agent" and "here is the runtime that
provides one" — a CustomRun with typed `params`/`results` and an `adapterRef`.
The pipeline no longer cares *how* the agent runs.

**Decoupling, reuse, and portability.** Since the engine lives behind the
adapter, the same `AgentTask` CR works across pipelines, and you can change the
backend without touching any of them. This is the point of the *opencode-adapter*
idea: you could keep classic paco's exact engine and still get the AgentTask
contract, just by running it behind an adapter.

**AI runs as governed objects.** Direct invocation leaves the agent as an opaque
step whose only trace is container logs. As a CustomRun, each run becomes a
Kubernetes object: reconciled under standard RBAC, inspectable with
`kubectl get runs`, and carrying whatever status the adapter records.

**Who owns the runtime.** Direct invocation puts the runtime in each pipeline
author's hands. Delegation lets a platform team own it centrally — the adapter
and its backend — while repos just "request a review" against a stable contract.

**Operational weight and maturity.** The cost of delegation is that there's more
to run. An agent controller, its CRD, and an adapter all have to exist on the
cluster before any pipeline can reference an `AgentTask`. Right now that stack is
pre-release, which is a real adoption risk regardless of how good the paradigm is.

---

## Maintenance

Both approaches share the GitHub-facing plumbing — `paco-cli`
(`paco diff`/`paco post`) and the PaC-triggered PipelineRun — and maintain it
identically, so paco-cli releases, GitHub API changes, and PaC annotation changes
land on both equally. But that shared layer is only the PR-review boundary; it
falls away the moment the scope widens past PR review (see below). What actually
differs between the approaches is the agentic layer behind it.

**A — the Task and its engine.** Direct invocation keeps the engine inside the
Task, so maintaining the agent *is* maintaining the pipeline: bumping the image,
switching the model, or rotating credentials means editing the Task, repeated
wherever it's been copied. Little to track (one project plus the engine image, on
a production-tested path), but nothing to upgrade centrally — the work scales with
the number of pipelines.

**B — the controller, CRD, and adapter.** Delegation adds a stack to keep alive
but concentrates its upkeep: a platform team upgrades the controller or adapter
once and every referencing pipeline picks it up untouched. The cost is that
you're tracking several independently versioned upstreams, today pre-release
(controller `v0.0.0-poc.1`, adapter untagged) — so expect churn and breaking
changes at the CRD boundary — plus cluster lifecycle work (install the CRD, run
the controller and adapter) that A doesn't have.

**Scope changes the math.** For PR review alone, B's extra stack is hard to
justify against A's self-contained Task. But direct invocation doesn't
generalize: a new use case (CI triage, security triage, release notes) isn't a
tweak to the Task, it's a fresh integration with its own engine wiring,
credentials, and output path — do it five times and you maintain five of them. B's
stack is the same whether it's reviewing a PR or triaging a build, so its heavy
fixed cost is paid once and amortized, and each new use case is cheap (declare an
`AgentTask`, handle its typed results). As scope widens the comparison inverts:
A's integrations pile up while B's one stack serves them all. This is the
"Beyond PaC" reuse point, seen from the maintenance side.

**Who writes the adapter?** One honest caveat on "pluggable": `adapterRef`
doesn't remove the pipeline-to-engine integration, it relocates it into the
adapter — and today both the controller and the only adapter that exists
(Lightspeed) are maintained by us. "Swap the backend" assumes an adapter for that
backend already exists, and for anything but Lightspeed it doesn't yet; someone
has to write and maintain one. That's still a better shape than A, since the
integration is written once and reused rather than redone per pipeline (this is
the opencode-adapter idea: you *could* run classic paco's engine behind an
adapter, once it's written), but the ownership doesn't vanish — it concentrates on
whoever maintains the adapters. Until backends' own maintainers supply them, the
pluggability is real in principle and narrow in practice.

---

## Decision factors

Direct invocation (A) is the right choice when:

- A single, known agent runtime is fine and unlikely to change.
- You want the lightest footprint and the fewest moving parts to operate.
- You value the production-tested path.

The AgentTask paradigm (B) is the right choice when:

- You want agentic steps to be a reusable, typed primitive shared across many
  pipelines, rather than a bespoke integration each time.
- You want pipelines decoupled from the agent runtime, so the backend can evolve
  or be swapped without rewriting pipelines.
- You want AI runs to be first-class, auditable Kubernetes objects under standard
  RBAC.
- A platform team wants to own the agent runtime centrally while repos just
  request work against a stable contract.

What argues against B today is maturity and operational weight: a pre-release
controller/adapter stack you have to install and run. That's an adoption risk
independent of whether the paradigm itself is sound.

---

## Beyond PaC: AgentTask in the wider Tekton ecosystem

Paco review is just the first use of the paradigm. Because `AgentTask` is a
general Tekton Custom Task (a `CustomRun`), the real payoff is ecosystem-wide
rather than PaC-specific:

- **Any pipeline, any trigger.** PaC is only one way to start a PipelineRun. The
  same `AgentTask` CR can go into pipelines started by Tekton Triggers /
  EventListeners, cron, or a manual run. An agentic step becomes available
  anywhere Tekton runs, not just on PR comments.
- **Agent judgment drives control flow.** Because an `AgentTask` emits typed
  results (e.g. `outcome`), downstream `when` conditions, `finally` tasks, and
  later tasks can branch on what the agent concluded — gating a release, opening
  a ticket, or failing a pipeline on the agent's verdict.
- **Pluggable runtimes.** `adapterRef` decouples the pipeline from the engine, so
  one primitive can serve many backends (cluster-aware diagnosis via Lightspeed,
  or a custom adapter for anything else) without each pipeline re-integrating an
  LLM.

Some concrete directions this opens up across Tekton (the first is shipped here;
the rest illustrate the same primitive):

- **PR / code review** — this project.
- **CI/CD failure triage** — an agentic step that diagnoses a failed build or
  test run, summarizes the root cause, and posts it back to the PR or chat.
- **Security & vulnerability triage** — analyze scanner output (SAST, image
  scans) and prioritize findings before they gate a release.
- **Release engineering** — changelog / release-note generation, or PR and issue
  summarization as a pipeline step.
- **Supply-chain & policy evaluation** — agentic assessment of attestations or
  policy compliance that gates promotion.
- **Incident response & remediation** — the Lightspeed adapter's native strength:
  multi-phase analysis → execution → verification, with approval gates and
  RBAC-scoped actions, so Ops pipelines can go beyond read-only review into
  governed remediation.

The thread running through all of these is reuse at scale. Once the agentic step
is a typed, governed Tekton primitive, it composes into existing pipelines
instead of each one building its own one-off LLM integration.

---

## Reference

### What a repository sees (identical either way)

- **Triggers** — the same PaC annotations: `/paco review`, `/paco summary`, and
  PR `opened`/`reopened` on `main`.
- **Inputs** — the same params (`repo_owner`, `repo_name`,
  `pull_request_number`, `trigger_comment`, `comment_id`, `git_auth_secret`).
- **Rules** — both read `.tekton/ai/REVIEW.md` from the base branch.
- **GitHub boundary** — both use the real `paco-cli` for `paco diff`/`paco post`.
- **Output** — both end in a `.paco-review.json` (paco-cli's schema) that
  `paco post` publishes as a sticky summary plus inline comments.

### Components

| | A — Direct invocation | B — AgentTask paradigm |
|---|---|---|
| The agentic step | In-process call within a build step | `AgentTask` CustomRun (`agent.tekton.dev/v1alpha1`) reconciled by an adapter |
| Paradigm CRD | — | `AgentTask` (typed `params` → `results`, `adapterRef`) |
| Engine (this instance) | opencode → Vertex AI (Anthropic) | OpenShift Lightspeed — *swappable; an `opencode` adapter could run A's engine here* |
| Who owns the runtime | Pipeline author | Platform team (controller + adapter) |
| Dependency maturity | Established | agent controller `v0.0.0-poc.1`, reference adapter untagged |

### Sources

- Classic paco pipeline — [`tekton-pac/.tekton/paco.yaml`][paco-yaml]
- AgentTask pipeline — [`.tekton/pr-review-pipeline.yaml`](.tekton/pr-review-pipeline.yaml)
- This repo's [`README.md`](README.md), [`cmd/`](cmd), [`pkg/`](pkg), [`config/`](config)
- paco-cli — [`github.com/pipelines-as-code/paco-cli`][paco-cli]
- AgentTask controller — [`github.com/openshift-pipelines/agenttask`][agenttask]
- Reference adapter — [`github.com/openshift-pipelines/agenttask-adapter-lightspeed`][adapter]

[paco-yaml]: https://github.com/tektoncd/pipelines-as-code/blob/main/.tekton/paco.yaml
[paco-cli]: https://github.com/pipelines-as-code/paco-cli
[agenttask]: https://github.com/openshift-pipelines/agenttask
[adapter]: https://github.com/openshift-pipelines/agenttask-adapter-lightspeed
