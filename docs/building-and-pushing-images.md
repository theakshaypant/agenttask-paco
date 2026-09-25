# Building and pushing task images to GHCR

This repository owns two container images — one per Go binary under
`cmd/`. The other two pipeline steps (`paco-diff`, `paco-post`) always use
the real, unmodified upstream `ghcr.io/chmouel/agents-image:latest`; there
is nothing to build or push for those.

| Image | `ko` build ID | Published as |
|---|---|---|
| `cmd/build-request` | `build-request` | `ghcr.io/theakshaypant/agenttask-paco/build-request` |
| `cmd/translate-result` | `translate-result` | `ghcr.io/theakshaypant/agenttask-paco/translate-result` |

Both are built with [`ko`](https://ko.build) (no `Dockerfile` — `ko`
compiles the Go binary and assembles a minimal, distroless,
multi-arch OCI image directly from source, per [`.ko.yaml`](../.ko.yaml)).

## Prerequisites

- [`ko`](https://ko.build) installed (`go install github.com/google/ko@latest`
  or the `ko-build/setup-ko` GitHub Action).
- Authenticated to `ghcr.io` with push access to
  `github.com/theakshaypant/agenttask-paco`. Any bearer token with
  `write:packages` scope works — there's no need to manage a separate PAT
  by hand, the `gh` CLI can supply one directly:

  ```sh
  # One-time: make sure the gh CLI's token has packages scope.
  gh auth refresh -h github.com -s write:packages

  gh auth token | ko login ghcr.io --username "$(gh api user -q .login)" --password-stdin
  # or simply:
  gh auth token | docker login ghcr.io -u "$(gh api user -q .login)" --password-stdin
  ```

  In CI, no manual token is needed at all — the workflow below uses the
  automatic, ephemeral `${{ github.token }}` (`GITHUB_TOKEN`), which is
  granted `packages: write` purely by the workflow's own `permissions:`
  block. A classic PAT is only ever needed if you specifically want to
  push from your own machine using long-lived credentials instead of
  `gh auth token`.

## Building locally (no push)

```sh
ko build --bare -L ./cmd/build-request
ko build --bare -L ./cmd/translate-result
```

`-L` loads the image into the local Docker/Podman daemon instead of
pushing, useful for smoke-testing a Task's image before publishing it.

## Building and pushing to GHCR manually

```sh
KO_DOCKER_REPO=ghcr.io/theakshaypant/agenttask-paco/build-request \
  ko build --bare -t latest --platform=linux/amd64,linux/arm64 ./cmd/build-request

KO_DOCKER_REPO=ghcr.io/theakshaypant/agenttask-paco/translate-result \
  ko build --bare -t latest --platform=linux/amd64,linux/arm64 ./cmd/translate-result
```

Use a specific tag (git SHA, semver) instead of `latest` for anything other
than local iteration, and update the `image:` field in
[`.tekton/tasks/build-request.yaml`](../.tekton/tasks/build-request.yaml) /
[`.tekton/tasks/translate-result.yaml`](../.tekton/tasks/translate-result.yaml)
to match.

## Automated builds (GitHub Actions)

[`​.github/workflows/container.yaml`](../.github/workflows/container.yaml)
builds and pushes both images on every push to `cmd/**`, `pkg/**`,
`go.mod`/`go.sum`, or `.ko.yaml`/the workflow file itself — mirroring the
root `tekton-pac` repo's own `ko`-based publish workflow. It tags images by
git ref (`vX.Y.Z` for tags, `pr-<number>` for pull requests, the sanitized
branch/ref name otherwise) so nothing is force-pushed under an ambiguous
`latest` tag from CI.
