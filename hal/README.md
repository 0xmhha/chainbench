# chainbench-hal

Network abstraction layer for chainbench. Provides a uniform command/event
interface over local, remote, and (future) ssh-remote chain nodes. Invoked as a
subprocess by the chainbench CLI and MCP server.

See `docs/VISION_AND_ROADMAP.md` §5.15–5.17 for the design.

## Build

    go build -o bin/chainbench-hal ./cmd/chainbench-hal

## Develop

    go generate ./...
    go test ./...
