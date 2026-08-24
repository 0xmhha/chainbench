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
   - Relevant log output (`chainbench log search` or raw log files)

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
5. **Test** your changes:
   ```bash
   chainbench init && chainbench start
   chainbench test run all
   chainbench stop
   ```
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
cmd/chainbench/     Go CLI (cobra commands)
cmd/chainbench-mcp/ Go MCP server for AI integration (single binary)
cmd/chainbench-dashboard/    Dashboard daemon
pkg/core/           shared core (config, genesis, nodeconfig, driver, rpc, logs, state, ...)
pkg/consensus/      consensus families (wbft, poa) + upgrade orchestration
pkg/chains/         chain plugins (stablenet, wbft, wemix)
pkg/mcp/            MCP tool handlers (call the same core packages as the CLI)
pkg/testkit/        test-case framework
manifests/          declarative chain manifests + genesis templates
profiles/           local network profiles
tests/              Go test cases (tests/all) + reproduction scripts (tests/repro)
```

### Adding a CLI Command

1. Add `cmd/chainbench/<name>.go` with a `new<Name>Cmd() *cobra.Command`.
2. Register it in `cmd/chainbench/root.go` (`root.AddCommand(...)`).
3. Keep the logic in `pkg/core` so the MCP tool for the same capability can share
   it (the two surfaces must behave identically).

### Adding a Test

1. Add a `testkit.Case` under `tests/<category>/` and register it in `tests/all`.
2. Run with `go test ./...` or `chainbench test --rpc <url> --category <cat>`.

### Adding a Profile

1. Create YAML in `profiles/` or `profiles/custom/`
2. Use `inherits: default` to extend the base profile
3. Only override fields that differ

### Modifying the MCP Server

1. Edit the Go tool handlers in `pkg/mcp/` (each tool calls the same `pkg/core`
   packages the CLI uses, so both surfaces stay identical).
2. Build: `go build -o bin/chainbench-mcp ./cmd/chainbench-mcp`
3. Test: `go test ./pkg/mcp/...` (tools are covered with httptest/fakes).

## Conventions

- **Shell scripts**: Use `bash` with `set -euo pipefail`. Follow existing patterns in `lib/`.
- **Commit messages**: [Conventional Commits](https://www.conventionalcommits.org/) format
- **Branch names**: `feat/`, `fix/`, `docs/`, `refactor/` prefixes
- **No breaking changes** to profile schema without migration path

## License

By contributing, you agree that your contributions will be licensed under the [Apache License 2.0](LICENSE).
