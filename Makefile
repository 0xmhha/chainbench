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

lint-tool: ## install the pinned golangci-lint into bin/ (skipped when it matches)
	@if [ -z "$(GOLANGCI_VERSION)" ]; then echo ".golangci-version is missing"; exit 1; fi; \
	have=$$($(GOLANGCI) version 2>/dev/null | sed -n 's/.*version \([0-9][^ ]*\).*/v\1/p'); \
	if [ "$$have" = "$(GOLANGCI_VERSION)" ]; then \
	  echo "golangci-lint $(GOLANGCI_VERSION) ($(GOLANGCI))"; \
	else \
	  echo "golangci-lint $${have:-not installed} -> installing $(GOLANGCI_VERSION)"; \
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
