# chainbench

A Go-first, multi-chain local test bench for geth-family chains. It brings up
multi-node local networks, verifies consensus, and runs test cases against them —
driven from a CLI, from an MCP server for AI agents, or from a live dashboard,
all on one shared Go core.

Supported chains: **stablenet** and **wbft** (WBFT/BFT family) and **wemix**
(PoA/etcd family). Adding another chain on an existing consensus family is
data-only (a manifest + a thin plugin); see [Adding a chain](#adding-a-chain).

> ⚠️ **TEST FIXTURE ONLY**: this repository commits a set of preset validator
> keys under `keys/preset/` for reproducible local testing. The password is `1`,
> the keystore is trivially decryptable, and all nodes bind to `127.0.0.1`.
> **Never import these keys into any mainnet, testnet, or shared environment.**
> Treat them as public. See `keys/preset/README.md`.

## Features

- **One Go core, three surfaces** — a CLI (`chainbench`), an MCP server
  (`chainbench-mcp`) for AI agents, and a dashboard daemon (`chainbenchd`), all
  calling the same core packages.
- **Consensus-family plugins** — the primary extension axis is the consensus
  algorithm (`wbft`, `poa`); a chain is a thin plugin that selects a family and
  supplies a declarative manifest.
- **Three-phase pipeline** — `setup` (plan → provision → launch) → `verify`
  (block production, node info) → `test` (gated test cases), each phase a
  standalone step over a shared `NodeSet`.
- **Local or remote nodes** — a `Driver` abstraction launches nodes as local
  subprocesses or over SSH.
- **No Docker required** — pure process-based, runs on macOS and Linux.

## Prerequisites

| Dependency | Version | Required for |
|------------|---------|--------------|
| Go | 1.25+ | building and running chainbench |
| a chain binary | latest | the node binary for the chain under test (`gstable` / `gwbft` / `gwemix`), built from its own repo |

The legacy bash CLI, Python profile tooling, and Node.js MCP server have been
removed; chainbench is a single Go module.

## Build

```bash
go build ./...                                  # build + type-check everything
go build -o bin/chainbench     ./cmd/chainbench
go build -o bin/chainbench-mcp ./cmd/chainbench-mcp
go build -o bin/chainbenchd    ./cmd/chainbenchd
```

`setup.sh` builds the three binaries and registers them on `$PATH` for local use.

The full test suite is deterministic (httptest / fake binaries), so it runs
without any real chain binary:

```bash
go test ./...
```

## Quick start (CLI)

Bring up a local stablenet network, verify it, and run the applicable tests. A
real chain binary is required for `--launch`; point `--binary` at it.

```bash
# plan + provision + launch a 2-validator network
chainbench setup --chain stablenet --validators 2 --endpoints 1 \
  --keys-dir keys/preset --data-dir /tmp/cb --binary /path/to/gstable --launch

# verify block production against the running nodes
chainbench verify --data-dir /tmp/cb

# run the test cases that apply to this chain (reports per-chain coverage)
chainbench test --data-dir /tmp/cb

# stop the network
chainbench stop --data-dir /tmp/cb
```

Without a real binary, you can still exercise the full lifecycle with a fake
node script (see `docs/dev/HandOff.md §3`).

## CLI reference

| Command | Purpose |
|---------|---------|
| `chains` | list the registered chains |
| `setup --chain --validators --endpoints [--provision] [--launch] [--binary] [--keys-dir] [--data-dir]` | plan / provision / launch a network |
| `verify --rpc <url>… \| --data-dir <dir>` | confirm block production and report node info |
| `test --rpc <url>… \| --data-dir <dir> [--name] [--category]` | run gated test cases; prints `coverage = ran / applicable` |
| `stop --data-dir <dir>` | stop the network's nodes by PID |
| `status` / `clean` | show / remove a network's state |
| `node rpc --rpc <url> --method <m> [--params …]` | arbitrary JSON-RPC passthrough |
| `consensus --chain --rpc <url>` | query the validator/producer set (manifest-driven method) |
| `faucet --chain --rpc --from-key --to --amount` | fund an address from a genesis-allocated key |
| `contract deploy \| call`, `tx send \| wait`, `account`, `genesis`, `state`, `log` | tx / contract / account / genesis / state / log helpers |
| `hardfork --data-dir --to-chain --block [--to-binary]` | binary-swap upgrade (homogeneous fork) |
| `upgrade run …` | concurrent consensus-family handoff (e.g. wemix → wbft) |
| `report --data-dir <dir>` | show stored run results (`ok` / `failed` / `skipped`) |

A persistent `--dashboard <url>` flag forwards setup/verify/test events to a
running `chainbenchd` over SSE. Run `chainbench <command> --help` for the full
flag set.

## MCP integration

`chainbench-mcp` is a self-contained Go MCP server (JSON-RPC over stdio) exposing
30 tools that call the same core as the CLI — lifecycle
(`chainbench_start`/`stop`/`status`), tests (`chainbench_test_run`/`test_list`/
`report`), RPC/tx/contract/consensus queries, logs, and remote-node tools.
Register it with your agent by pointing at the built `chainbench-mcp` binary
(`{"mcpServers": {"chainbench": {"command": "chainbench-mcp"}}}`).

## Tests

Test cases are Go `testkit.Case` values registered at `init()` under
`tests/<family>/<category>/`, blank-imported by `tests/all`. Each case declares
`ChainCompat` (which chains it applies to) and `RequiresCaps`; the runner skips
the rest and reports **coverage** (`ran / applicable`) so a chain that gates most
cases out reads as under-tested rather than green. See `tests/README.md` for the
conventions and the add-a-chain-to-a-test path.

## Adding a chain

There are two ways, depending on whether the chain is first-party or your own
project's.

### Project-supplied (no chainbench change)

If your chain reuses a built-in consensus family (`wbft` or `poa`), point
chainbench at your own manifest — no code change, no rebuild:

```bash
chainbench setup \
  --manifest        ../my-chain/chainbench.json \
  --genesis-template ../my-chain/genesis.json \
  --keys-dir ../my-chain/keys --binary ../my-chain/bin/gmychain --launch
```

The manifest is the same schema as `manifests/chains/*.json`; set `"protocol"`
to a built-in accounts profile (`stablenet` / `wbft` / `wemix`) to borrow its tx
types and account model. Only a genuinely new consensus algorithm needs code.

The same `--manifest` / `--genesis-template` flags work on `consensus` and
`faucet` (and `verify` / `test` read the chain from the launched network's
`nodeset.json`, so they need no extra flag).

### First-party (embedded in the tool)

To ship a chain with chainbench itself:

1. `manifests/chains/<id>.json` — the declarative manifest.
2. `manifests/genesis/<id>.json` — the genesis template (for template-based
   families).
3. `pkg/chains/<id>/<id>.go` — a thin plugin: load the manifest, select the
   family, supply the accounts protocol, and `registry.Register`.
4. Add a blank import to `pkg/chains/all/all.go`.
5. Register the chain's accounts protocol from the plugin's `init()`
   (`protocol.Register`) if the SDK does not already know it.

Either way, chain-specific bindings (e.g. stablenet governance) live under
`pkg/chains/<id>/…`, not in the generic core.

## Project structure

```
chainbench/
├── cmd/
│   ├── chainbench/         # Go CLI (cobra)
│   ├── chainbench-mcp/     # MCP server (Go, single binary)
│   └── chainbenchd/        # dashboard daemon (HTTP/SSE)
├── pkg/
│   ├── core/               # chain-agnostic core: registry, config, pipeline
│   │                       #   (setup/verify/attach/testrun), driver, genesis,
│   │                       #   nodeconfig, node, rpc, obs, probe, remote, …
│   ├── consensus/          # consensus families (wbft, poa) + upgrade handoff
│   ├── chains/             # chain plugins (stablenet, wbft, wemix) + bindings
│   ├── accounts/           # chain-agnostic account/tx/ABI boundary over the SDK
│   ├── mcp/                # MCP tool handlers (same core as the CLI)
│   ├── dashboard/          # SSE dashboard server
│   └── testkit/            # test-case framework (Case / T / Report)
├── manifests/              # declarative chain manifests + genesis templates
├── profiles/               # local network profiles (YAML)
├── keys/preset/            # preset validator keys (TEST FIXTURE ONLY)
└── tests/                  # Go test cases (tests/all) + repro scripts (tests/repro)
```

## Preset keys

`keys/preset/` holds fixed keys for 5 nodes so validator addresses are identical
across runs (reproducible logs/debugging). TEST FIXTURE ONLY — see the warning
above.

| Node | Address | Role |
|------|---------|------|
| node1 | `0xc17d493883eaa3b4cceb0f214b273392d562f9d8` | Validator |
| node2 | `0x2493a84a8f83cb87fdcbe0bb3b2d313f69a58d3c` | Validator |
| node3 | `0x8c4a10b9108d49b9d23f764464090831d9c17764` | Validator |
| node4 | `0x8eb79036bc0f3aba136ef18b3a2fb8c1188939a6` | Validator |
| node5 | `0x5400d8b543eaf6738c7b44799623bea88fd0f5ee` | Endpoint |

## Contributing

1. Create a feature branch (`git checkout -b feat/my-feature`).
2. Keep `pkg/core` free of chain/consensus imports (the boundary is
   compiler-enforced); chain-specific code lives in `pkg/chains/*` and
   `pkg/consensus/*`.
3. Commit with [Conventional Commits](https://www.conventionalcommits.org/).
4. Ensure `gofmt`, `go vet`, and `go test ./...` pass.
5. Open a pull request.

## License

Licensed under the [Apache License 2.0](LICENSE).
