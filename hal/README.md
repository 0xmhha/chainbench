# chainbench-hal

Network abstraction layer for chainbench. Provides a uniform command/event
interface over local, remote, and (future) ssh-remote chain nodes. Invoked as a
subprocess by the chainbench CLI and MCP server.

See `docs/VISION_AND_ROADMAP.md` §5.15–5.17 for the design.

## Prerequisites

- Go 1.25+ (required by the `go-jsonschema` code generator dependency)

## Build

    go build -o bin/chainbench-hal ./cmd/chainbench-hal

## Develop

    go generate ./...
    go test ./...

## Tools

Development-only tool dependencies are pinned via `tools.go` under the `tools`
build tag. Normal builds exclude that file. To validate the tool pin and its
transitive graph use:

    go build -tags tools ./...
    go list  -tags tools ./...
