# agenttask-paco AI Review Rules

Paco uses these project-specific rules in addition to checking for concrete
bugs, security issues, and missed edge cases.

## Review Priorities

- Report only issues introduced or exposed by the diff. Do not comment on
  unchanged code unless the change makes an existing defect reachable.
- Prefer correctness, security, data loss, and compatibility findings over
  style. Explain the concrete failure mode, not just the rule.
- Do not report formatting or other issues an existing formatter/linter
  (`gofmt`, `golangci-lint`) will catch.
- Avoid speculative findings. If a claim depends on repository context
  absent from the diff, do not present it as a defect.
- Report one comment per root cause. Do not repeat the same issue at every
  affected call site.

## Architecture Boundaries

- This module has **no dependency on `paco-cli`** (as a library, binary, or
  artifact contract) and **no dependency on `ghcr.io/chmouel/agents-image`
  for the review step**. Flag any change that reintroduces either.
- The real `ghcr.io/chmouel/agents-image:latest` image is fine to reuse for
  the `paco-diff`/`paco-post` steps only (it bundles the real `paco-cli`
  binary); the `review` step must always go through the `agent.tekton.dev
  AgentTask` Custom Task backed by `agenttask-adapter-lightspeed`, never
  `opencode`/Vertex.
- `AgentTask`/`CustomRun` only carries a single opaque `request` string
  param and two results (`outcome`, `analysis-result-name`). Do not add
  extra params to `config/agenttask-pr-review.yaml` expecting them to reach
  the adapter — assemble everything into the `request` string beforehand.
- Talk to the native `agentic.openshift.io/v1alpha1 AnalysisResult` object
  via a dynamic/unstructured client (`pkg/analysisresult`), not a typed
  client — the upstream API module is pinned to an unreleased pseudo-version.

## Testing

- Use table-driven tests with an anonymous struct slice
  (`tests := []struct{...}{...}`) iterated with
  `for _, tt := range tests { t.Run(tt.name, ...) }`.
- No underscores in test function names; use PascalCase.
- Use `gotest.tools/v3/assert` for assertions — never `testify`.
- Unit tests must stay fully offline (fake/fixture clients, no real
  cluster, no real `OPENAI_API_KEY`); reserve real-LLM/real-cluster
  behavior for the (separately documented) Kind e2e path.

## Error Handling

- Wrap errors with `fmt.Errorf("...: %w", err)` when crossing an
  abstraction boundary or when the added context helps identify the failed
  operation.
- Treat `pkg/artifact.Workspace.ReadOptional` results (`.existing-feedback.txt`,
  `.tekton/ai/REVIEW.md`, `.toolchain-versions`) as genuinely optional —
  never fail the pipeline because one of them is missing.
- Do not silently convert required-operation failures (reading `.pr.diff`,
  fetching the `AnalysisResult`) into success.

## Style / Go Idioms

- Keep `context.Context` as the first parameter for functions that need
  one.
- Prefer explicit returns over naked returns.
- Keep the `request` string bounded (`promptbuild.Bound`, 32KB) — never
  remove the truncation guard, since the adapter enforces this limit.

## Security

- Secrets (`git_auth_secret`, any LLM provider credentials) must be sourced
  from Kubernetes Secrets, never hardcoded or logged in plaintext.
- The `request` string embeds untrusted PR diff/comment content; scrub or
  bound it before logging, and never execute it as a shell command.
- `.tekton/ai/REVIEW.md` content is trusted repository-owner input (it
  ships with the repo, not the PR); do not conflate it with untrusted PR
  content when reasoning about prompt-injection risk.

## Dependencies

- Changes to `go.mod`/`go.sum` in this module must not reintroduce a
  dependency on `paco-cli` or reduce this module's independence from the
  root `tekton-pac` Go module.
