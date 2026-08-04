# chainbench — build / test / lint / fmt
# Usage: `make help`

GO       ?= go
BIN_DIR  ?= bin
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS  := -ldflags "-X main.version=$(VERSION)"
PKGS     ?= ./...
CMDS     := chainbench chainbenchd chainbench-mcp
GOLANGCI ?= golangci-lint

.DEFAULT_GOAL := help

.PHONY: help build $(CMDS) test test-race test-e2e cover lint fmt fmt-check vet tidy secrets check clean

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
lint: ## golangci-lint 실행 (.golangci.yml)
	@command -v $(GOLANGCI) >/dev/null || { echo "golangci-lint 미설치 — https://golangci-lint.run/usage/install/"; exit 1; }
	$(GOLANGCI) run

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
