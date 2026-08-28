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
takes a manifest and a thin plugin that registers it, not a fork of the tool.

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
> unrecoverable and controllable by anyone.
>
> This includes **plaintext private keys written inline in test source** (the
> genesis-funded faucet key is the upstream go-ethereum test key, already public
> in every geth fork). A secret scanner run over this repository **will report
> findings, and those findings are expected** — every one of them is a fixture,
> and the fixtures are public by design so local runs stay reproducible. Treat a
> finding as real only if it is outside `keys/preset/` and outside test code.
>
> See [`keys/preset/README.md`](keys/preset/README.md) and
> [`docs/SECURITY_KEY_HANDLING.md`](docs/SECURITY_KEY_HANDLING.md).

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
  (`chainbench-mcp`) for AI agents, and a dashboard daemon (`chainbench-dashboard`) all
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
go build -o bin/chainbench-dashboard    ./cmd/chainbench-dashboard

# ...or build all + register on $PATH for local use
./setup.sh

# verify the build (no chain binary needed)
go test ./...
```

### Quick start

Compose a local stablenet network into a workspace, verify it, run the
applicable tests, and tear it down. Starting needs a real chain binary — point
`--binary` at it.

```bash
chainbench net up --workspace-dir /tmp/cb --chain stablenet --binary /path/to/gstable \
  --validators 2 --endpoints 1 --keys keys/preset

chainbench verify --workspace-dir /tmp/cb   # confirm block production + node info
chainbench test   --workspace-dir /tmp/cb   # run applicable cases; prints coverage
chainbench stop   --workspace-dir /tmp/cb   # stop the nodes
```

Or declare the network in a DSL env and let one command compose, run, and
tear down: `chainbench run --workspace-dir /tmp/cb tests/cases/stablenet/chain-up.json`.

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
| `net up` / `net <step>` | compose a network into a workspace, all at once or one step at a time (`--workspace-dir --chain --binary --validators --endpoints [--keys] [--set] [--overlay]`) |
| `net resume` | recover a workspace whose run died: reconcile recorded pids with the machine, continue from the first unfinished step, bring nodes back |
| `run` | run DSL specs: attach (`--rpc`) or compose what the specs declare (`--workspace-dir`) |
| `verify` | confirm block production and report node info (`--rpc <url>…` or `--workspace-dir`) |
| `test` | run the remaining Go-func cases (`--rpc` or `--workspace-dir`, `[--name] [--category]`); prints `coverage = ran / applicable` |
| `status` / `stop` / `clean` | show / stop (by PID) / stop-and-remove a composed network |
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
to a running `chainbench-dashboard` over SSE.

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

`chainbench-mcp` is a self-contained Go MCP server (JSON-RPC over stdio) that
exposes the same features the CLI drives — lifecycle, tests, RPC / tx / contract
/ consensus queries, logs, and remote-node tools. The tool set grows with the
chains a binary imports, so ask the server rather than a document: `tools/list`
enumerates exactly what this build exposes. Register it with your agent by
pointing at the built binary:

```json
{ "mcpServers": { "chainbench": { "command": "chainbench-mcp" } } }
```

Beyond the built-in tools, chainbench exposes a **layered capability catalog**:
common features shared by every chain plus chain-specific ones (e.g. stablenet
governance, wemix bootstrap), addressed as `<version>.<chain>.<name>`. Call the
`chainbench.capabilities` tool (or `chainbench capabilities [--chain]` on the
CLI) to discover what a chain supports. A capability is declared as data and bound in code, both inside the chain's own
package. The catalog is a `.jsonl` file of `capability.Descriptor` lines the
package embeds and loads in its `init` (e.g. `internal/chains/stablenet/caps.jsonl`
through `capability.LoadCatalog`). That declares the metadata and nothing more:
a capability appears in discovery and in `tools/list` only once it is **bound**.
Binding is either `capability.RegisterHandler` in the same `init` — what
`internal/chains/stablenet/caps.go` does for each governance verb, giving a
callable tool — or `capability.RegisterFlat`, which maps an already-existing
`chainbench_*` tool into the catalog for discovery. Either way the chain package
must be blank-imported from `internal/chains/all/all.go`, or its `init` never
runs. See
[`internal/core/capability/README.md`](internal/core/capability/README.md); the
flat tools live in `internal/mcp/*_tools.go`.

### Dashboard

`chainbench-dashboard` serves a live dashboard: a Server-Sent Events stream at `/events`
and run state at `/api/runs`, with a built Svelte SPA under `/app/`. Point CLI
runs at it with `--dashboard`:

```bash
chainbench-dashboard --addr 127.0.0.1:8787 &
chainbench verify --workspace-dir /tmp/cb --dashboard http://127.0.0.1:8787
```

The SPA source lives in [`web/`](web/) (Svelte 5 + Vite); see `web/README.md` to
rebuild it.

## Adding a chain

**Project-supplied (no chainbench change).** If your chain reuses a built-in
consensus family (`wbft` or `poa`), point chainbench at your own manifest — no
code, no rebuild:

```bash
chainbench net up --workspace-dir /tmp/my \
  --manifest         ../my-chain/chainbench.json \
  --genesis-template ../my-chain/genesis.json \
  --keys ../my-chain/keys --binary ../my-chain/bin/gmychain
```

The manifest uses the same schema as [the built-in chains' `manifest.json`](internal/chains);
set `"protocol"` to a built-in accounts profile (`stablenet` / `wbft` / `wemix`)
to borrow its tx types and account model. The `--manifest` / `--genesis-template`
flags also work on `consensus` and `faucet`.

**First-party (embedded in the tool).** To ship a chain with chainbench: add
`internal/chains/<id>/manifest.json` and `internal/chains/<id>/genesis.json`, a
thin plugin at `internal/chains/<id>/<id>.go` that embeds them and registers via
`registry.Register` in its `init`, and a blank import in
`internal/chains/all/all.go`. Chain-specific bindings live under
`internal/chains/<id>/…`, never in the generic core.

Only a genuinely new consensus algorithm needs a new `internal/consensus/<family>`.

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
│   └── chainbench-dashboard/      # dashboard daemon (HTTP/SSE)
├── internal/
│   ├── core/             # chain-agnostic core: registry, machine, filestore,
│   │                     #   driver, process, keyring (model/derive/store/
│   │                     #   operation), netmap, genesis, nodeconfig, rpc,
│   │                     #   obs, remote, portplan, session, …
│   ├── consensus/        # consensus families (wbft, poa) + upgrade handoff
│   ├── chains/           # one folder per chain: plugin + manifest + genesis +
│   │                     #   bindings + capabilities (stablenet, wbft, wemix, external)
│   ├── chainsetup/       # composes a chain up to producing blocks
│   ├── testengine/       # runs tests on a chain something else composed
│   ├── netmap/           # server set, placement, and the one dial-wiring point
│   ├── app/              # workflow layer MCP reaches (DSL → setup → test → report)
│   ├── accounts/         # account/tx/ABI boundary over the accounts SDK
│   ├── mcp/              # MCP tool handlers (through the app layer)
│   ├── dashboard/        # SSE server + embedded Svelte SPA
│   └── testkit/          # test-case framework (Case / T / Report)
├── profiles/             # network + golden upgrade profiles (YAML)
├── keys/preset/          # preset validator keys (TEST FIXTURE ONLY)
├── tests/                # Go test cases (tests/all) + repro scripts (tests/repro)
└── web/                  # dashboard SPA source (Svelte + Vite)
```

The design contract: `internal/core` never imports a chain or consensus package.
Tests in `internal/arch` read the import graph and fail on any such edge, so the
rule is checked, not just written down. Chain knowledge lives in
`internal/chains/*`, and consensus knowledge in `internal/consensus/*`.
The CLI calls core modules directly; MCP reaches the same features through
`internal/app` — see
[`docs/dev/architecture/architecture-v2.md`](docs/dev/architecture/architecture-v2.md).

## Contributing

1. Create a feature branch (`git checkout -b feat/my-feature`).
2. Keep `internal/core` free of chain/consensus imports; chain-specific code
   lives in `internal/chains/*` and `internal/consensus/*`.
3. Use [Conventional Commits](https://www.conventionalcommits.org/).
4. Ensure `gofmt`, `go vet`, and `go test ./...` pass.
5. Open a pull request.

## License

Licensed under the [Apache License 2.0](LICENSE).
