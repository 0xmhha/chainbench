# chainbench

> A Go-first, multi-chain local test bench for geth-family blockchains — bring up
> multi-node networks, verify consensus, and run tests, from a CLI, an MCP server
> for AI agents, or a live dashboard, all on one shared core.

[![CI](https://github.com/0xmhha/chainbench/actions/workflows/ci.yml/badge.svg)](https://github.com/0xmhha/chainbench/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white)](go.mod)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

chainbench spins up local, multi-node blockchain networks without Docker, drives
them through a three-phase pipeline (**setup → verify → test**), and reports
per-chain test coverage. Adding a chain that reuses an existing consensus family
is data-only — a manifest and a thin plugin, no fork of the tool.

**Supported chains:** `stablenet` and `wbft` (WBFT/BFT family), and `wemix`
(PoA/etcd family). It also drives a **concurrent consensus-family upgrade** —
a live PoA→BFT hardfork handoff (`wemix` → `wbft`).

> [!CAUTION]
> **The preset keys and addresses are TEST FIXTURES — never use them in production.**
> The validator keys, keystores, and addresses under `keys/preset/` exist only for
> reproducible local testing. They are committed to a public repository with a
> trivially decryptable keystore, and every node binds to `127.0.0.1`, so their
> private keys are effectively public knowledge.
>
> **NEVER use any key, keystore, or address that appears in this repository — in
> `keys/preset/`, in profiles, in manifests, or in tests — on any production,
> mainnet, testnet, staging, or shared network.** They are for disposable, local
> throwaway networks ONLY. Any funds or authority assigned to them are
> unrecoverable and controllable by anyone. See
> [`keys/preset/README.md`](keys/preset/README.md).

## Table of contents

- [Features](#features)
- [Getting started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Install](#install)
  - [Quick start](#quick-start)
- [Usage](#usage)
  - [CLI](#cli)
  - [Consensus-family upgrade](#consensus-family-upgrade)
  - [MCP server](#mcp-server)
  - [Dashboard](#dashboard)
- [Adding a chain](#adding-a-chain)
- [Testing](#testing)
- [Architecture](#architecture)
- [Contributing](#contributing)
- [License](#license)

## Features

- **One core, three surfaces** — a CLI (`chainbench`), an MCP server
  (`chainbench-mcp`) for AI agents, and a dashboard daemon (`chainbenchd`) all
  call the same Go core, so every surface stays behaviorally identical.
- **Consensus-family plugins** — the primary extension axis is the consensus
  algorithm (`wbft`, `poa`); a chain is a thin plugin that selects a family and
  supplies a declarative manifest.
- **Three-phase pipeline** — `setup` (plan → provision → launch) → `verify`
  (block production, node info) → `test` (gated cases), each a standalone step
  over a shared `NodeSet`.
- **Coverage-aware testing** — cases declare which chains they apply to; the
  runner skips the rest and reports `coverage = ran / applicable`, so a chain
  that gates most cases out reads as under-tested rather than green.
- **Concurrent hardfork handoff** — reproduce a live PoA→BFT upgrade from a
  golden profile, with the from-chain producing up to the fork and the successor
  validators taking over after it.
- **Local or remote nodes** — a `Driver` abstraction launches nodes as local
  subprocesses or over SSH. No Docker; runs on macOS and Linux.

## Getting started

### Prerequisites

| Dependency | Version | Required for |
|---|---|---|
| [Go](https://go.dev/dl/) | 1.25+ | building and running chainbench |
| a chain binary | — | launching real nodes (`gstable` / `gwbft` / `gwemix`), built from its own repo |

chainbench is a single Go module; the legacy bash CLI, Python tooling, and
Node.js MCP server have been removed. The **test suite is deterministic**
(httptest + fake binaries), so it runs green without any real chain binary — a
real binary is only needed to `--launch` a live network.

### Install

```bash
git clone https://github.com/0xmhha/chainbench.git
cd chainbench

# build the three binaries
go build -o bin/chainbench     ./cmd/chainbench
go build -o bin/chainbench-mcp ./cmd/chainbench-mcp
go build -o bin/chainbenchd    ./cmd/chainbenchd

# ...or build all + register on $PATH for local use
./setup.sh

# verify the build (no chain binary needed)
go test ./...
```

### Quick start

Bring up a local stablenet network, verify it, run the applicable tests, and
tear it down. `--launch` needs a real chain binary — point `--binary` at it.

```bash
chainbench setup --chain stablenet --validators 2 --endpoints 1 \
  --keys-dir keys/preset --data-dir /tmp/cb --binary /path/to/gstable --launch

chainbench verify --data-dir /tmp/cb   # confirm block production + node info
chainbench test   --data-dir /tmp/cb   # run applicable cases; prints coverage
chainbench stop   --data-dir /tmp/cb   # stop the nodes
```

No chain binary handy? You can still exercise the whole lifecycle with a fake
node script — see [`docs/dev/HandOff.md`](docs/dev/HandOff.md) §3.

> [!WARNING]
> `--keys-dir keys/preset` and every address it produces are **test-only
> fixtures**. Use them exclusively for local, throwaway networks — **never in any
> production or shared environment** (see the caution above).

## Usage

### CLI

Run `chainbench <command> --help` for the full flag set of any command.

| Command | Purpose |
|---|---|
| `chains` | list the registered chains |
| `setup` | plan / provision / launch a network (`--chain --validators --endpoints [--provision] [--launch] [--binary] [--keys-dir] [--data-dir]`) |
| `verify` | confirm block production and report node info (`--rpc <url>…` or `--data-dir`) |
| `test` | run gated test cases (`--rpc` or `--data-dir`, `[--name] [--category]`); prints `coverage = ran / applicable` |
| `status` / `stop` / `clean` | show / stop (by PID) / stop-and-remove a launched network |
| `node rpc \| start \| stop` | arbitrary JSON-RPC passthrough, or relaunch/stop a single node |
| `consensus` | query the validator/producer set (manifest-driven RPC method) |
| `tx send \| wait` | sign and send a transaction, or wait for a receipt |
| `contract deploy \| call` | deploy from bytecode, or make a read-only `eth_call` |
| `account` | inspect account state over RPC |
| `faucet` | fund an address from a genesis-allocated key |
| `log` | search a setup's per-node logs |
| `report` | show stored run/test results (`ok` / `failed` / `skipped`) |
| `hardfork` | binary-swap upgrade at a fork block (homogeneous fork) |
| `upgrade genesis \| run` | concurrent consensus-family handoff (see below) |

A persistent `--dashboard <url>` flag forwards `setup` / `verify` / `test` events
to a running `chainbenchd` over SSE.

### Consensus-family upgrade

`upgrade run` reproduces a live **PoA → BFT hardfork handoff**: the from-chain
(`wemix` + etcd) produces blocks up to a fork height, then the successor
validators (`wbft`) — which synced the pre-fork chain — take over producing
after it. The exact, verified-live environment is captured in a **golden
profile** as the single source of truth.

```bash
chainbench upgrade run \
  --profile profiles/wemix-upgrade.yaml \
  --from-binary /path/to/gwemix \
  --to-binary   /path/to/gwbft \
  --wait 60
```

See [`profiles/wemix-upgrade.yaml`](profiles/wemix-upgrade.yaml) for the encoded
conditions (uniform network id, disjoint producers/validators, BFT quorum,
paired fork sections). `upgrade genesis` builds just the merged handoff genesis.

### MCP server

`chainbench-mcp` is a self-contained Go MCP server (JSON-RPC over stdio) exposing
**30 tools** that call the same core as the CLI — lifecycle, tests, RPC / tx /
contract / consensus queries, logs, and remote-node tools. Register it with your
agent by pointing at the built binary:

```json
{ "mcpServers": { "chainbench": { "command": "chainbench-mcp" } } }
```

Beyond the built-in tools, chainbench exposes a **layered capability catalog**:
common features shared by every chain plus chain-specific ones (e.g. stablenet
governance, wemix bootstrap), addressed as `<version>.<chain>.<name>`. Call the
`chainbench.capabilities` tool (or `chainbench capabilities [--chain]` on the
CLI) to discover what a chain supports. Adding a chain's features is data-only —
a `.jsonl` catalog plus handlers under `pkg/mcp/features/<project>/`; see
[`pkg/mcp/features/README.md`](pkg/mcp/features/README.md).

### Dashboard

`chainbenchd` serves a live dashboard: a Server-Sent Events stream at `/events`
and run state at `/api/runs`, with a built Svelte SPA under `/app/`. Point CLI
runs at it with `--dashboard`:

```bash
chainbenchd --addr 127.0.0.1:8787 &
chainbench verify --data-dir /tmp/cb --dashboard http://127.0.0.1:8787
```

The SPA source lives in [`web/`](web/) (Svelte 5 + Vite); see `web/README.md` to
rebuild it.

## Adding a chain

**Project-supplied (no chainbench change).** If your chain reuses a built-in
consensus family (`wbft` or `poa`), point chainbench at your own manifest — no
code, no rebuild:

```bash
chainbench setup \
  --manifest         ../my-chain/chainbench.json \
  --genesis-template ../my-chain/genesis.json \
  --keys-dir ../my-chain/keys --binary ../my-chain/bin/gmychain --launch
```

The manifest uses the same schema as [the built-in chains' `manifest.json`](pkg/chains);
set `"protocol"` to a built-in accounts profile (`stablenet` / `wbft` / `wemix`)
to borrow its tx types and account model. The `--manifest` / `--genesis-template`
flags also work on `consensus` and `faucet`.

**First-party (embedded in the tool).** To ship a chain with chainbench: add
`pkg/chains/<id>/manifest.json` and `pkg/chains/<id>/genesis.json`, a thin plugin at
`pkg/chains/<id>/<id>.go` that registers via `registry.Register`, and a blank
import in `pkg/chains/all/all.go`. Chain-specific bindings live under
`pkg/chains/<id>/…`, never in the generic core.

Only a genuinely new consensus algorithm needs a new `pkg/consensus/<family>`.

## Testing

```bash
go test ./...        # deterministic; no chain binary required
gofmt -l . && go vet ./...
```

Test cases are Go `testkit.Case` values registered at `init()` under
`tests/<family>/<category>/`, blank-imported by `tests/all`. Each declares
`ChainCompat` and `RequiresCaps`; the runner gates and reports **coverage**. See
[`tests/README.md`](tests/README.md) for the conventions, plus `tests/repro/`
for end-to-end shell reproductions.

## Architecture

```
chainbench/
├── cmd/
│   ├── chainbench/       # CLI (cobra)
│   ├── chainbench-mcp/   # MCP server (single binary)
│   └── chainbenchd/      # dashboard daemon (HTTP/SSE)
├── pkg/
│   ├── core/             # chain-agnostic core: registry, config, pipeline
│   │                     #   (setup/verify/attach/testrun), driver, genesis,
│   │                     #   nodeconfig, rpc, obs, probe, remote, portplan, …
│   ├── consensus/        # consensus families (wbft, poa) + upgrade handoff
│   ├── chains/           # one folder per chain: plugin + manifest + genesis +
│   │                     #   bindings + capabilities (stablenet, wbft, wemix, external)
│   ├── accounts/         # account/tx/ABI boundary over the accounts SDK
│   ├── mcp/              # MCP tool handlers (same core as the CLI)
│   ├── dashboard/        # SSE server + embedded Svelte SPA
│   └── testkit/          # test-case framework (Case / T / Report)
├── profiles/             # network + golden upgrade profiles (YAML)
├── keys/preset/          # preset validator keys (TEST FIXTURE ONLY)
├── tests/                # Go test cases (tests/all) + repro scripts (tests/repro)
└── web/                  # dashboard SPA source (Svelte + Vite)
```

The design contract: `pkg/core` never imports a chain or consensus package (the
boundary is compiler-enforced). Chain knowledge lives in `pkg/chains/*`, and
consensus knowledge in `pkg/consensus/*`.

## Contributing

1. Create a feature branch (`git checkout -b feat/my-feature`).
2. Keep `pkg/core` free of chain/consensus imports; chain-specific code lives in
   `pkg/chains/*` and `pkg/consensus/*`.
3. Use [Conventional Commits](https://www.conventionalcommits.org/).
4. Ensure `gofmt`, `go vet`, and `go test ./...` pass.
5. Open a pull request.

## License

Licensed under the [Apache License 2.0](LICENSE).
