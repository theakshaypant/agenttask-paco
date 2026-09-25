# SPDX-License-Identifier: Apache-2.0
# Makefile for agenttask-paco — an independent Go module, isolated from the
# root tekton-pac repo's ./pkg/... ./cmd/... ./test/... build scope.

GO ?= go
GOLANGCI_LINT ?= golangci-lint

.PHONY: build
build:
	$(GO) build ./...

.PHONY: fmt
fmt:
	$(GO) fmt ./...

.PHONY: vet
vet:
	$(GO) vet ./...

.PHONY: test
test: vet
	$(GO) test ./... -count=1

.PHONY: lint
lint:
	$(GOLANGCI_LINT) run ./...

.PHONY: check
check: lint test

.PHONY: clean
clean:
	rm -rf bin coverage.out
