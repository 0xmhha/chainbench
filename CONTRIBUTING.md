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
lib/cmd_*.sh     Shell command implementations
lib/common.sh    Shared utilities (logging, binary resolution)
lib/profile.sh   YAML profile parser
profiles/        Built-in and custom profiles
templates/       Genesis and TOML config templates
tests/           Built-in test suites
mcp-server/      TypeScript MCP server for AI integration
```

### Adding a CLI Command

1. Create `lib/cmd_<name>.sh`
2. Register in `chainbench.sh` dispatch case
3. Add to help text in `_cb_show_usage()`

### Adding a Test

1. Create `tests/<category>/your-test.sh`
2. Add header: `# Description: What this test verifies`
3. Source libraries:
   ```bash
   source "$(dirname "$0")/../lib/rpc.sh"
   source "$(dirname "$0")/../lib/assert.sh"
   ```
4. Use `test_start` / `test_result` framing
5. Tests are auto-discovered by `chainbench test list`

### Adding a Profile

1. Create YAML in `profiles/` or `profiles/custom/`
2. Use `inherits: default` to extend the base profile
3. Only override fields that differ

### Modifying the MCP Server

1. Edit TypeScript sources in `mcp-server/src/`
2. Build: `cd mcp-server && npm run build`
3. Test: restart Claude Code and verify tools work

## Conventions

- **Shell scripts**: Use `bash` with `set -euo pipefail`. Follow existing patterns in `lib/`.
- **Commit messages**: [Conventional Commits](https://www.conventionalcommits.org/) format
- **Branch names**: `feat/`, `fix/`, `docs/`, `refactor/` prefixes
- **No breaking changes** to profile schema without migration path

## License

By contributing, you agree that your contributions will be licensed under the [Apache License 2.0](LICENSE).
