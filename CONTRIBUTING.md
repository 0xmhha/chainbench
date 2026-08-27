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
   - Relevant log output (`chainbench log --data-dir <dir> --pattern <text>`, or
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
   make check        # gofmt check + go vet + golangci-lint (pinned) + go test
   ```
   Individually: `make fmt-check`, `make vet`, `make lint`, `make test`,
   `make test-race`.

   CI (`.github/workflows/ci.yml`) runs more than that, so a green `make check`
   is not a green CI. What it adds:
   ```bash
   bash scripts/check-secrets.sh --all                 # secret scan
   find scripts tests -name '*.sh' -exec bash -n {} \; # shell syntax
   go build ./...
   go test -race ./...                                 # make test is not -race
   ```
   To exercise a real network end to end, compose one with
   `chainbench net up --data-dir /tmp/cb --chain stablenet --binary <node>` and
   tear it down with `chainbench net stop --data-dir /tmp/cb`.
6. **Commit** using [Conventional Commits](https://www.conventionalcommits.org/):
   ```
   feat: add new stress test for large transactions
   fix: resolve port conflict detection on macOS
   docs: update profile schema reference
   refactor: simplify genesis template substitution
   ```
7. **Push** and open a Pull Request against `main`

## Development Guide

### Project Layout

```
cmd/chainbench/          Go CLI (cobra commands; groups in keyringcmd/, netcmd/, netmapcmd/)
cmd/chainbench-mcp/      Go MCP server for AI integration (single binary)
cmd/chainbench-dashboard/ Dashboard daemon
internal/core/           chain-agnostic core (registry, machine, filestore, driver,
                         process, keyring, netmap, genesis, nodeconfig, rpc, session, ...)
internal/consensus/      consensus families (wbft, poa) + upgrade orchestration
internal/chains/         chain plugins (stablenet, wbft, wemix, external) + manifests,
                         genesis templates, and capability catalogs
internal/chainsetup/     composes a chain up to producing blocks
internal/testengine/     runs tests on an already-composed chain
internal/netmap/         server set, placement, and the one dial-wiring point
internal/app/            workflow layer MCP reaches (DSL -> setup -> test -> report)
internal/mcp/            MCP tool handlers (through internal/app)
internal/testkit/        test-case framework
profiles/                local network profiles
tests/                   Go test cases (tests/all) + reproduction scripts (tests/repro)
```

### Adding a CLI Command

1. Add `cmd/chainbench/<name>.go` with a `new<Name>Cmd() *cobra.Command`.
2. Register it in `cmd/chainbench/root.go` (`root.AddCommand(...)`).
3. Keep the logic in the core module that owns it, never in the command file.
   The CLI calls those modules directly; MCP reaches the same features through
   `internal/app`, so a feature added in a module is reachable from both
   surfaces. See
   [`docs/dev/architecture/architecture-v2.md`](docs/dev/architecture/architecture-v2.md).

### Adding a Test

1. Add a `testkit.Case` under `tests/<category>/` and register it in `tests/all`.
2. Run with `go test ./...` or `chainbench test --rpc <url> --category <cat>`.

### Adding a Profile

1. Create YAML in `profiles/` or `profiles/custom/`
2. Use `inherits: default` to extend the base profile
3. Only override fields that differ

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
- **No breaking changes** to profile schema without migration path

## License

By contributing, you agree that your contributions will be licensed under the [Apache License 2.0](LICENSE).
