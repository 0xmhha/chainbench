# chainbench — build / test / lint / fmt
# Usage: `make help`

GO       ?= go
BIN_DIR  ?= bin
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -ldflags "-X main.version=$(VERSION)"
PKGS     ?= ./...
CMDS     := chainbench chainbench-dashboard chainbench-mcp
# The linter is pinned so a local run and CI reach the same verdict. The version
# lives in one file that both this and .github/workflows/ci.yml read: an older
# golangci-lint reported three findings here that the pinned one does not, and
# chasing a difference like that is time spent on the tool rather than the code.
GOLANGCI_VERSION := $(shell cat .golangci-version 2>/dev/null)
GOLANGCI ?= $(BIN_DIR)/golangci-lint

.DEFAULT_GOAL := help

.PHONY: help build $(CMDS) test test-race test-e2e cover lint lint-tool fmt fmt-check vet tidy secrets check clean

help: ## list the targets
	@grep -hE '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) | sort | \
	  awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

## --- build ---------------------------------------------------------------
build: $(CMDS) ## build every binary into bin/

$(CMDS): ## build one binary (e.g. make chainbench)
	$(GO) build $(LDFLAGS) -o $(BIN_DIR)/$@ ./cmd/$@

## --- test ----------------------------------------------------------------
test: ## unit and integration tests (e2e excluded)
	$(GO) test $(PKGS)

test-race: ## the same under -race
	$(GO) test -race $(PKGS)

test-e2e: ## e2e tests (they skip themselves without chain binaries)
	$(GO) test -tags e2e $(PKGS)

cover: ## coverage into coverage.html
	$(GO) test -coverprofile=coverage.out $(PKGS)
	$(GO) tool cover -html=coverage.out -o coverage.html

## --- lint / fmt / vet ----------------------------------------------------
lint: lint-tool ## golangci-lint (.golangci.yml), at the version CI pins
	$(GOLANGCI) run

# The guard asks two questions, because the pinned version alone is not enough
# to know a local binary can be used. The linter PARSES our source, so the Go it
# was built with has to be at least the language version go.mod targets. A build
# that is too old refuses the whole run:
#
#   can't load config: the Go language version (go1.25) used to build
#   golangci-lint is lower than the targeted Go version (1.26.8)
#
# Measured 2026-10-06. bin/golangci-lint was v2.12.2 built with go1.25.13,
# installed while go.mod still said 1.25. Raising go.mod to 1.26.8 (#440) left
# the version string matching, so this target reported the linter present and
# skipped reinstalling, and `make lint` and `make check` failed on every tree
# while CI stayed green — CI downloads a release binary built with go1.26.2.
# A gate that only the author's machine fails is a gate nobody runs.
lint-tool: ## install the pinned golangci-lint into bin/ (skipped when it matches)
	@if [ -z "$(GOLANGCI_VERSION)" ]; then echo ".golangci-version is missing"; exit 1; fi; \
	out=$$($(GOLANGCI) version 2>/dev/null); \
	have=$$(printf '%s' "$$out" | sed -n 's/.*version \([0-9][^ ]*\).*/v\1/p'); \
	built=$$(printf '%s' "$$out" | sed -n 's/.*built with go\([0-9][0-9]*\.[0-9][0-9]*\).*/\1/p'); \
	want=$$(sed -n 's/^go \([0-9][0-9]*\.[0-9][0-9]*\).*/\1/p' go.mod); \
	lang=$$($(GO) env GOVERSION | sed -n 's/^go\([0-9][0-9]*\.[0-9][0-9]*\).*/\1/p'); \
	num() { printf '%s' "$$1" | awk -F. '{print $$1*1000+$$2}'; }; \
	if [ "$$have" = "$(GOLANGCI_VERSION)" ] && [ -n "$$built" ] && \
	   [ "$$(num $$built)" -ge "$$(num $$want)" ]; then \
	  echo "golangci-lint $(GOLANGCI_VERSION), built with go$$built ($(GOLANGCI))"; \
	else \
	  if [ -n "$$lang" ] && [ "$$(num $$lang)" -lt "$$(num $$want)" ]; then \
	    echo "this toolchain is go$$lang and go.mod targets $$want — the linter built here could not read this module."; \
	    echo "install Go $$want or newer, or lower go.mod's go directive."; \
	    exit 1; \
	  fi; \
	  if [ "$$have" = "$(GOLANGCI_VERSION)" ]; then \
	    echo "golangci-lint $(GOLANGCI_VERSION) is present but built with go$${built:-?}, older than go.mod's $$want -> rebuilding with go$$lang"; \
	  else \
	    echo "golangci-lint $${have:-not installed} -> installing $(GOLANGCI_VERSION)"; \
	  fi; \
	  GOBIN=$(abspath $(BIN_DIR)) $(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION); \
	fi

fmt: ## gofmt -w over the tree
	gofmt -w .

fmt-check: ## name the files gofmt would change (CI gate)
	@out=$$(gofmt -l .); [ -z "$$out" ] || { echo "these files need gofmt:"; echo "$$out"; exit 1; }

vet: ## go vet
	$(GO) vet $(PKGS)

## --- housekeeping --------------------------------------------------------
tidy: ## go mod tidy
	$(GO) mod tidy

secrets: ## scan the tree for secrets (pre-commit gate)
	bash scripts/check-secrets.sh --all

check: fmt-check vet lint test ## everything a commit should pass

clean: ## remove build output
	rm -rf $(BIN_DIR) coverage.out coverage.html
