# chainbench — build / test / lint / fmt
# Usage: `make help`

GO       ?= go
BIN_DIR  ?= bin
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -ldflags "-X main.version=$(VERSION)"
PKGS     ?= ./...
CMDS     := chainbench chainbenchd chainbench-mcp
# The linter is pinned so a local run and CI reach the same verdict. The version
# lives in one file that both this and .github/workflows/ci.yml read: an older
# golangci-lint reported three findings here that the pinned one does not, and
# chasing a difference like that is time spent on the tool rather than the code.
GOLANGCI_VERSION := $(shell cat .golangci-version 2>/dev/null)
GOLANGCI ?= $(BIN_DIR)/golangci-lint

.DEFAULT_GOAL := help

.PHONY: help build $(CMDS) test test-race test-e2e cover lint lint-tool fmt fmt-check vet tidy secrets check clean

help: ## 사용 가능한 타깃 목록
	@grep -hE '^[a-zA-Z0-9_-]+:.*?## ' $(MAKEFILE_LIST) | sort | \
	  awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

## --- build ---------------------------------------------------------------
build: $(CMDS) ## 모든 바이너리 빌드 → bin/

$(CMDS): ## 개별 바이너리 빌드 (예: make chainbench)
	$(GO) build $(LDFLAGS) -o $(BIN_DIR)/$@ ./cmd/$@

## --- test ----------------------------------------------------------------
test: ## 유닛/통합 테스트 (e2e 제외)
	$(GO) test $(PKGS)

test-race: ## -race 로 테스트 (동시성 검증)
	$(GO) test -race $(PKGS)

test-e2e: ## e2e 테스트 (바이너리 미제공 시 자동 skip)
	$(GO) test -tags e2e $(PKGS)

cover: ## 커버리지 → coverage.html
	$(GO) test -coverprofile=coverage.out $(PKGS)
	$(GO) tool cover -html=coverage.out -o coverage.html

## --- lint / fmt / vet ----------------------------------------------------
lint: lint-tool ## golangci-lint 실행 (.golangci.yml) — CI 와 같은 핀 버전으로
	$(GOLANGCI) run

lint-tool: ## .golangci-version 의 golangci-lint 를 bin/ 에 설치 (버전이 맞으면 건너뜀)
	@if [ -z "$(GOLANGCI_VERSION)" ]; then echo ".golangci-version 이 없다"; exit 1; fi; \
	have=$$($(GOLANGCI) version 2>/dev/null | sed -n 's/.*version \([0-9][^ ]*\).*/v\1/p'); \
	if [ "$$have" = "$(GOLANGCI_VERSION)" ]; then \
	  echo "golangci-lint $(GOLANGCI_VERSION) ($(GOLANGCI))"; \
	else \
	  echo "golangci-lint $${have:-미설치} → $(GOLANGCI_VERSION) 설치"; \
	  GOBIN=$(abspath $(BIN_DIR)) $(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION); \
	fi

fmt: ## gofmt -w (코드 포맷 적용)
	gofmt -w .

fmt-check: ## gofmt 미적용 파일 검출 (CI 게이트)
	@out=$$(gofmt -l .); [ -z "$$out" ] || { echo "gofmt 필요:"; echo "$$out"; exit 1; }

vet: ## go vet 정적 분석
	$(GO) vet $(PKGS)

## --- housekeeping --------------------------------------------------------
tidy: ## go mod tidy
	$(GO) mod tidy

secrets: ## 시크릿 스캔 (커밋 전 보안 게이트)
	bash scripts/check-secrets.sh --all

check: fmt-check vet lint test ## 커밋 전 종합 게이트 (fmt+vet+lint+test)

clean: ## 빌드 산출물 정리
	rm -rf $(BIN_DIR) coverage.out coverage.html
