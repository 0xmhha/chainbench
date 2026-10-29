# Contributing to chainbench

Thank you for your interest in contributing to chainbench! This document provides guidelines and instructions for contributing.

## Code of Conduct

Be respectful and constructive in all interactions. We are committed to providing a welcoming and inclusive environment for everyone.

## How to Contribute

### Reporting Bugs

1. Check [existing issues](https://github.com/0xmhha/chainbench/issues) to avoid duplicates
2. Open a new issue with:
   - Clear, descriptive title
   - Steps to reproduce
   - Expected vs actual behavior
   - OS and environment details
   - Relevant log output (`chainbench log --workspace-dir <dir> --pattern <text>`, or
     raw log files)

### Suggesting Features

Open an issue with the `enhancement` label. Describe:
- The problem you're trying to solve
- Your proposed solution
- Any alternatives you considered

### Submitting Pull Requests

1. **Fork** the repository
2. **Clone** your fork and set up the development environment:
   ```bash
   git clone https://github.com/<your-username>/chainbench.git
   cd chainbench
   ./setup.sh
   ```
3. **Branch** from `main`:
   ```bash
   git checkout -b feat/my-feature
   ```
4. **Make changes** following the conventions below
5. **Test** your changes. `make check` is the main local gate:
   ```bash
   make check        # fmt-check vet lint test tc-review sweep-test case-docs fixture-test
   ```
   Individually: `make fmt-check`, `make vet`, `make lint`, `make test`,
   `make test-race`, `make tc-review`, `make sweep-test`, `make case-docs`,
   `make fixture-test`.

   CI (`.github/workflows/ci.yml`) runs more than that, so a green `make check`
   is not a green CI. What it adds:
   ```bash
   bash scripts/check-secrets.sh --all                 # secret scan
   find scripts tests -name '*.sh' -exec bash -n {} \; # shell syntax
   go build ./...
   go test -race ./...                                 # make test is not -race
   go vet -tags e2e ./...                              # the e2e build tag
   npm ci --prefix web && npm run check --prefix web   # svelte-check
   node --test tests/webui/*.test.mjs                  # Web UI node tests
   ```
   After changing anything under `web/`, run `npm --prefix web run build`
   (it writes the embedded `internal/dashboard/spa/`) before the Go build and
   commit the built files.
   To exercise a real network end to end, compose one with
   `chainbench chain up --workspace-dir /tmp/cb --chain stablenet --binary <node>` and
   tear it down with `chainbench chain stop --workspace-dir /tmp/cb`.
6. **Commit** using [Conventional Commits](https://www.conventionalcommits.org/):
   ```
   feat: add new stress test for large transactions
   fix: resolve port conflict detection on macOS
   docs: update preset schema reference
   refactor: simplify genesis template substitution
   ```
7. **Push** and open a Pull Request against `main`

## Development Guide

### Project Layout

```
cmd/chainbench/          Go CLI (cobra; command groups in chaincmd/, suitecmd/,
                         keyringcmd/, lifecyclecmd/, networkcmd/, txcmd/, ...)
cmd/chainbench-mcp/      Go MCP server for AI integration (single binary)
cmd/chainbench-dashboard/ Web dashboard daemon (serves the embedded SPA and /api/v1)
internal/core/           chain-agnostic core (registry, filestore, process, keyring,
                         genesis, nodeconfig, inspector, rpc, session, ...)
internal/consensus/      consensus families (wbft, poa) + upgrade orchestration
internal/chains/         chain plugins (stablenet, wbft, wemix, external) + manifests,
                         genesis templates, and capability catalogs
internal/chainsetup/     composes a chain up to producing blocks
internal/testengine/     runs tests on an already-composed chain
internal/dsl/            DSL spec grammar, validation and interpretation
internal/resource/       servers, port slots and placement a network is composed from
internal/app/            workflow layer MCP and the dashboard reach (DSL -> setup -> test -> report)
internal/mcp/            MCP tool handlers (through internal/app)
internal/dashboard/      dashboard HTTP server, /api/v1 handlers and the embedded spa/
internal/webcli/         runs the chainbench CLI for dashboard jobs
web/                     Svelte 5 + Vite source of the dashboard SPA
presets/chain/           chain-presets a case names (chain, nodes, binaries, upgrade)
tests/                   DSL cases (tests/tc), e2e tests (tests/e2e), repros (tests/repro),
                         Web UI tests and live acceptance runners (tests/webui)
```

### Adding a CLI Command

1. Add the command to the group package it belongs to
   (`cmd/chainbench/<group>cmd/`), exposed as a `New<Name>() *cobra.Command`.
2. Register a new top-level command in `cmd/chainbench/root.go` (`root.AddCommand(...)`).
3. Keep the logic in the core module that owns it, never in the command file.
   The CLI calls those modules directly; MCP reaches the same features through
   `internal/app`, so a feature added in a module is reachable from both
   surfaces. See
   [`docs/dev/architecture/architecture-v2.md`](docs/dev/architecture/architecture-v2.md).

### Adding a Test

1. Write a DSL case under `tests/tc/` — see
   [`docs/guide/dsl-authoring.md`](docs/guide/dsl-authoring.md), and
   [`tests/README.md`](tests/README.md) for when a case belongs in `tests/e2e/`
   instead.
2. Check it offline with `chainbench validate <case.json>`, then run it with
   `chainbench run --workspace-dir <ws> <case.json>` (compose the network the
   case declares), `chainbench run --attach --workspace-dir <ws> <case.json>`
   (the network that workspace already composed), or
   `chainbench run --rpc <url> --chain <id> <case.json>` (a live network).
   `--attach` does not combine with `--rpc`.

### Modifying the MCP Server

1. Edit the Go tool handlers in `internal/mcp/` (`*_tools.go`, registered in
   `tools.go`). A tool calls `internal/app`, which wraps the core modules the
   CLI calls directly, so both surfaces reach the same feature.
2. Build: `go build -o bin/chainbench-mcp ./cmd/chainbench-mcp`
3. Test: `go test ./internal/mcp/...` (tools are covered with httptest/fakes).
   `internal/arch` also checks that MCP imports stay on the app layer.

## Conventions

- **Shell scripts**: Use `bash` with `set -euo pipefail`. Follow existing patterns in `scripts/` and `env/docker/`.
- **Commit messages**: [Conventional Commits](https://www.conventionalcommits.org/) format
- **Branch names**: `feat/`, `fix/`, `docs/`, `refactor/` prefixes
- **No breaking changes** to the DSL or preset schema without a migration path

## License

By contributing, you agree that your contributions will be licensed under the [Apache License 2.0](LICENSE).
