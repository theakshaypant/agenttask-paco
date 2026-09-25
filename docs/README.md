# agenttask-paco docs

This directory documents the individual Tekton `Task`s that make up the
`pr-review` pipeline (`.tekton/pr-review-pipeline.yaml`), and how to build
and publish the two task images this repository owns.

| Task | File | Description |
|---|---|---|
| `paco-diff` | [tasks/paco-diff.md](tasks/paco-diff.md) | Fetches the PR diff/feedback/review-rules/toolchain artifacts (real `paco-cli`, unmodified). |
| `build-request` | [tasks/build-request.md](tasks/build-request.md) | Renders the review prompt sent to the `review` AgentTask (this repo's own image). |
| `review` (AgentTask) | [tasks/review-agenttask.md](tasks/review-agenttask.md) | Runs the review through `agenttask-adapter-lightspeed` (not a Tekton `Task`, no image to build). |
| `translate-result` | [tasks/translate-result.md](tasks/translate-result.md) | Maps the `AnalysisResult` into `paco-cli`'s review schema (this repo's own image). |
| `paco-post` | [tasks/paco-post.md](tasks/paco-post.md) | Posts the sticky review, including inline comments (real `paco-cli`, unmodified). |

See [building-and-pushing-images.md](building-and-pushing-images.md) for how
to build and push `build-request`/`translate-result` to `ghcr.io`.
