# chainbench-net

Network abstraction layer for chainbench. Provides a uniform command/event
interface over local, remote, and (future) ssh-remote chain nodes. Invoked as a
subprocess by the chainbench CLI and MCP server.

See `docs/VISION_AND_ROADMAP.md` §5.15–5.17 for the design.

## Prerequisites

- Go 1.25+ (required by the `go-jsonschema` code generator dependency)

## Build

    go build -o bin/chainbench-net ./cmd/chainbench-net

## Develop

    go generate ./...
    go test ./...

## Runtime

The `run` subcommand reads a wire command envelope from stdin and emits an
NDJSON result terminator on stdout. Structured logs go to stderr.

    echo '{"command":"network.load","args":{"name":"local"}}' | chainbench-net run

Environment:

- `CHAINBENCH_STATE_DIR` — directory containing `pids.json` and
  `current-profile.yaml`. Defaults to `state` relative to the current
  working directory.
- `CHAINBENCH_NET_LOG_LEVEL` — `debug` | `info` | `warn` | `error`
  (default `info`).
- `CHAINBENCH_NET_LOG` — optional path to write logs instead of stderr.

Exit codes follow the wire error-code table (VISION §5): 0 success,
1 generic/INVALID_ARGS/UPSTREAM_ERROR/INTERNAL, 2 NOT_SUPPORTED,
3 PROTOCOL_ERROR.

## Tools

Development-only tool dependencies are pinned via `tools.go` under the `tools`
build tag. Normal builds exclude that file. To validate the tool pin and its
transitive graph use:

    go build -tags tools ./...
    go list  -tags tools ./...
