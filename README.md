# chainbench

> Multi-chain local test bench for geth-family blockchains — compose multi-node
> networks, verify consensus, and run declarative tests from a CLI, an MCP
> server, or a live dashboard.

[![CI](https://github.com/0xmhha/chainbench/actions/workflows/ci.yml/badge.svg)](https://github.com/0xmhha/chainbench/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.25%2B-00ADD8?style=flat&logo=go&logoColor=white)](go.mod)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

**chainbench** brings up local, multi-node blockchain networks without Docker,
drives them step by step from keys to running nodes, and runs test cases written
as JSON against them. It supports `stablenet` and `wbft` (BFT family) and
`wemix` (PoA/etcd family), and reproduces a live PoA→BFT hardfork handoff.
Adding a chain that reuses an existing consensus family takes a manifest, not a
fork of the tool.

---

## Architecture

```
                 ┌──────────┐  ┌──────────────┐  ┌─────────────────────┐
   surfaces      │chainbench│  │chainbench-mcp│  │chainbench-dashboard │
                 │  (CLI)   │  │ (AI agents)  │  │    (HTTP/SSE)       │
                 └────┬─────┘  └──────┬───────┘  └──────────┬──────────┘
                      │               │                     │
                      └───────────────┴─────────────────────┘
                                      │
                 ┌────────────────────┴────────────────────┐
   workflow      │  chainsetup (compose)  │  testengine    │
                 └────────────────────┬────────────────────┘
                                      │
                 ┌────────────────────┴────────────────────┐
   core          │ registry · driver · keyring · genesis   │
                 │ nodeconfig · netmap · rpc · process     │
                 └────────────────────┬────────────────────┘
                      ┌───────────────┴───────────────┐
   plugins       │ consensus/ wbft · poa │ chains/ stablenet · wbft · wemix │
                 └───────────────────────────────────┘
```

`internal/core` never imports a chain or a consensus package. Tests in
`internal/arch` read the import graph and fail on any such edge, so the rule is
checked rather than written down. See
[`docs/dev/architecture/architecture-v2.md`](docs/dev/architecture/architecture-v2.md).

---

## Features

### Composition

- **Step-by-step bring-up** — `keys → place → genesis → config → build → deploy
  → init → start`, each runnable on its own or all at once with `chain up`.
- **Deterministic placement** — roles, hosts and non-colliding ports are
  computed from the key set, so the same declaration yields the same network.
- **Local or remote nodes** — a driver abstraction launches nodes as local
  subprocesses or over SSH. No Docker required; macOS and Linux.
- **Crash recovery** — `chain resume` reconciles recorded PIDs with the machine
  and continues from the first unfinished step.

### Testing

- **Declarative cases** — a test is a JSON document of steps and expectations,
  not Go code. `run` composes the network the case declares, runs it, and tears
  it down.
- **One corpus, three chains** — a case is written against a `stablenet-*`
  chain preset and swapped onto the others with `--chain-preset`, so 69 common
  cases cover all three without copies.
- **Capability gating** — a case declares what it requires; a target that does
  not provide it skips rather than fails, and a skip nobody declared is an
  error rather than a silent pass.
- **Offline validation** — `validate` parses and checks specs with no chain
  binary and no network.

### Surfaces

- **One core, three surfaces** — the CLI, the MCP server for AI agents, and the
  dashboard daemon all call the same Go core, so they stay behaviourally
  identical.
- **Consensus-family plugins** — the extension axis is the consensus algorithm
  (`wbft`, `poa`); a chain is a thin plugin that selects a family and supplies a
  declarative manifest.
- **Hardfork handoff** — reproduce a live PoA→BFT upgrade, the from-chain
  producing up to the fork and the successor validators taking over after it.

---

## Installation

```bash
git clone https://github.com/0xmhha/chainbench.git
cd chainbench

make build        # all three binaries into bin/
./setup.sh        # ...or build and put them on $PATH

make test         # verify the build; no chain binary needed
```

| Dependency | Version | Required for |
|---|---|---|
| [Go](https://go.dev/dl/) | 1.25+ | building and running chainbench |
| a chain binary | — | launching real nodes (`gstable` / `gwbft` / `gwemix`), built from its own repo |

The test suite is deterministic (httptest and fake binaries), so it passes
without any real chain binary. A real binary is needed only to launch a live
network.

---

## Quick Start

Compose a four-producer stablenet network into a workspace, verify it, and tear
it down.

```bash
chainbench chain up --workspace-dir /tmp/cb \
  --chain stablenet --binary /path/to/gstable --bp 4 --en 1

chainbench verify --workspace-dir /tmp/cb    # confirm block production
chainbench status --workspace-dir /tmp/cb    # show the node set

# run a case against the network already composed here
chainbench run --attach --workspace-dir /tmp/cb \
  tests/tc/common/rpc/CT-RPC-001-block-number-advances.json

chainbench chain stop --workspace-dir /tmp/cb
```

Or let a test case declare the network, compose it, run it and tear it down in
one command:

```bash
chainbench run tests/tc/common/node/CT-NODE-001-startup-block-production.json \
  --workspace-dir /tmp/cb --binary /path/to/gstable
```

Run the same case on another chain by swapping the preset:

```bash
chainbench run tests/tc/common/node/CT-NODE-001-startup-block-production.json \
  --workspace-dir /tmp/cb --chain-preset wemix-bp4 --binary /path/to/gwemix
```

`--plan` prints the network a case and its flags resolve to without composing
anything.

---

## Configuration

`chain up` composes a network; every step below it takes the same workspace.

| Flag | Default | Description |
|---|---|---|
| `--workspace-dir` | — | where the composition and its records live |
| `--chain` | — | `stablenet`, `wbft` or `wemix`; ignored with `--manifest` |
| `--binary` | — | node binary to launch |
| `--bp` | `4` | block-producing node count |
| `--en` | `0` | endpoint (non-producing) node count |
| `--pn` | `0` | proxy-tier node count; a family with no proxy tier refuses it |
| `--keys` | `presets/keys` | key set the network composes from |
| `--keys-source` | `preset` | `preset`, `generate`, or `declared` |
| `--set` | — | override a genesis config key, repeatable (`--set bohoBlock=10`) |
| `--overlay` | — | JSON overlay deep-merged into the genesis |
| `--server-set` | — | which servers exist and how to reach them |
| `--all-servers` | `false` | spread the network across every server, one node per host |
| `--docker` | `false` | servers are local docker containers; translate dials through the localmap |

`run` either composes or attaches.

| Flag | Description |
|---|---|
| `--workspace-dir` | compose the network the specs declare, then run against it |
| `--rpc` | attach to a live network by node RPC URL (repeatable) |
| `--attach` | run against the network this workspace already composed |
| `--chain-preset` | run every case on this preset instead of the one it names |
| `--keep-up` | leave the network running after the run |
| `--plan` | print the resolved network and stop |
| `--dashboard` | stream run events to a running dashboard |

---

## CLI

Run `chainbench <command> --help` for the full flag set of any command.

### Composition

| Command | Purpose |
|---|---|
| `chain up` | compose and launch in one command |
| `chain <step>` | one step at a time: `keys`, `place`, `genesis`, `config`, `build`, `deploy`, `init`, `start` |
| `chain status` / `show` / `health` | composition state, where a node is, per-node head block |
| `chain start` / `stop` / `restart` / `rm` | lifecycle of the composed data plane |
| `chain resume` | recover a workspace whose run died |
| `chain blueprint` | write the network a key set would compose, as an editable declaration |
| `resource` | the servers, port slots and capacity a network may be composed from |

### Tests

| Command | Purpose |
|---|---|
| `run` | run DSL specs: compose what they declare, or attach with `--rpc` |
| `validate` | parse and check specs without running them |
| `test list <dir>` | list the runnable DSL cases under a directory |
| `report` | show a run's report from its session directory |
| `baseline` | the approved fingerprint of an environment's prepared inputs |

### Inspection and operations

| Command | Purpose |
|---|---|
| `chains` / `capabilities` | registered chains; what each one exposes |
| `verify` | confirm a network is producing blocks |
| `consensus` | query the validator set through the chain's own RPC namespace |
| `node` / `network` | control one node; attach to and call running networks by name |
| `tx` / `contract` / `account` / `faucet` | send and wait, deploy and call, inspect, fund |
| `keyring` / `validator` | create, inspect and move key material; validator identities |
| `query` | the read-only commands, gathered |
| `log` / `file` | search per-node logs; move files to and from a server |
| `hardfork` | plan a binary swap at a fork block, keeping node data |
| `clean` | remove a launched network's data dir, or GC old sessions |

---

## Hardfork Handoff

A PoA→BFT handoff is declared, not scripted: the from-chain (`wemix` + etcd)
produces up to a fork height, then the successor validators (`wbft`), which
synced the pre-fork chain, take over after it.

```bash
chainbench run tests/tc/go-wemix/hardfork/01-croissant-successors-take-over.json \
  --workspace-dir /tmp/cb
```

The case names the chain preset
[`presets/chain/wemix-to-wbft.json`](presets/chain/wemix-to-wbft.json), whose
`upgrade` block points at the golden preset:

```json
"upgrade": { "fork": "croissant", "at": 20,
             "from": "default", "to": "next", "style": "concurrent" }
```

[`presets/chain/README.md`](presets/chain/README.md) records how a hardfork is
declared and what is checked: uniform network id, disjoint producers and
validators, BFT quorum, paired fork sections. To plan a binary swap against a
workspace that is already running, use `chainbench hardfork`.

---

## MCP Server

`chainbench-mcp` is a self-contained Go MCP server (JSON-RPC over stdio) that
exposes what the CLI drives: lifecycle, tests, RPC, transactions, contracts,
consensus queries, logs and remote-node tools.

```json
{ "mcpServers": { "chainbench": { "command": "chainbench-mcp" } } }
```

The tool set grows with the chains a binary imports, so ask the server rather
than a document — `tools/list` enumerates exactly what a build exposes.

Beyond the built-in tools, chainbench exposes a layered capability catalog:
features common to every chain plus chain-specific ones, addressed as
`<version>.<chain>.<name>`. Call `chainbench.capabilities`, or
`chainbench capabilities [--chain]` on the CLI, to discover what a chain
supports.

A capability is declared as data and bound in code, both inside the chain's own
package. The catalog is a `.jsonl` file of `capability.Descriptor` lines the
package embeds and loads in its `init`. Declaration alone is not enough: a
capability appears in discovery and in `tools/list` only once it is bound,
either by `capability.RegisterHandler` in the same `init` or by
`capability.RegisterFlat`, which maps an existing `chainbench_*` tool into the
catalog. Either way the chain package must be blank-imported from
`internal/chains/all/all.go`, or its `init` never runs. See
[`internal/core/registry/README.md`](internal/core/registry/README.md).

---

## Dashboard

`chainbench-dashboard` serves a Server-Sent Events stream at `/events`, run
state at `/api/runs`, and a built Svelte SPA under `/app/`.

```bash
chainbench-dashboard --addr 127.0.0.1:8787 &
chainbench verify --workspace-dir /tmp/cb --dashboard http://127.0.0.1:8787
```

The SPA source lives in [`web/`](web/) (Svelte 5 + Vite); see `web/README.md`
to rebuild it.

---

## Adding a Chain

**Project-supplied — no chainbench change.** If your chain reuses a built-in
consensus family (`wbft` or `poa`), point chainbench at your own manifest. No
code, no rebuild:

```bash
chainbench chain up --workspace-dir /tmp/my \
  --manifest         ../my-chain/chainbench.json \
  --genesis-template ../my-chain/genesis.json \
  --keys ../my-chain/keys --binary ../my-chain/bin/gmychain
```

The manifest uses the same schema as
[the built-in chains' `manifest.json`](internal/chains); set `"protocol"` to a
built-in accounts profile (`stablenet` / `wbft` / `wemix`) to borrow its
transaction types and account model.

**First-party — embedded in the tool.** Add
`internal/chains/<id>/manifest.json` and `internal/chains/<id>/genesis.json`, a
thin plugin at `internal/chains/<id>/<id>.go` that embeds them and registers via
`registry.Register` in its `init`, and a blank import in
`internal/chains/all/all.go`. Chain-specific bindings live under
`internal/chains/<id>/`, never in the generic core. Only a genuinely new
consensus algorithm needs a new `internal/consensus/<family>`.

---

## Testing

```bash
make test         # unit and integration; no chain binary required
make test-race    # with -race
make lint         # golangci-lint, the version CI pins
make fmt-check    # gofmt gate
make secrets      # secret scan, the pre-commit gate
```

The DSL corpus lives under `tests/tc/`:

| Directory | What is in it |
|---|---|
| `tests/tc/common/` | 69 cases that run on all three chains, one file per test case |
| `tests/tc/go-stablenet/`, `go-wbft/`, `go-wemix/` | cases for one chain only |
| `tests/e2e/` | end-to-end harness tests |
| `tests/repro/` | shell reproductions |

Run the whole common corpus, one network per case:

```bash
scripts/tcsweep.sh sweep.log tests/tc/common
```

See [`tests/tc/HOW-TO-USE.md`](tests/tc/HOW-TO-USE.md) for the
per-case commands and [`tests/README.md`](tests/README.md) for the conventions.

---

## Security

> [!CAUTION]
> **Every key in this repository is a test fixture. Never use one anywhere real.**

The validator keys, keystores and addresses under `presets/keys/` exist only for
reproducible local testing. They are committed to a public repository with a
trivially decryptable keystore, and every node binds to `127.0.0.1`, so their
private keys are effectively public knowledge.

Never use any key, keystore or address that appears in this repository — in
`presets/keys/`, in presets, in manifests, or in tests — on any production,
mainnet, testnet, staging or shared network. Not to hold value, not to seal a
block, not to sign anything. Anyone who can read this repository can sign as any
of them, so any funds or authority given to one is already gone, and rotating
afterwards does not undo what was signed in the meantime.

This covers every key in the tree, whatever produced it:

- the node identities under `presets/keys/node{1..5}/`, generated once and
  committed so a local network comes up with the same validators every time;
- the dev accounts under `presets/keys/dev1/`, which the harness mints on the
  first run that asks for one and reads back afterwards, so a label keeps naming
  one address;
- plaintext private keys written inline in test source — the genesis-funded
  faucet key is the upstream go-ethereum test key, already public in every geth
  fork.

A secret scanner reports nothing on this repository, and that is a decision, not
an accident. [`.betterleaks.toml`](.betterleaks.toml) allowlists exactly these
paths and leaves every rule of the default set on. **Treat a finding as real if
it is outside `presets/keys/` and outside test code** — the allowlist is not to
be widened to quiet one.

See [`presets/keys/README.md`](presets/keys/README.md) and
[`docs/SECURITY_KEY_HANDLING.md`](docs/SECURITY_KEY_HANDLING.md).

---

## Documentation

- [`docs/dev/architecture/architecture-v2.md`](docs/dev/architecture/architecture-v2.md) — layers and the import rule
- [`docs/guide/dsl-authoring.md`](docs/guide/dsl-authoring.md) — how to write a test case
- [`tests/tc/HOW-TO-USE.md`](tests/tc/HOW-TO-USE.md) — running any case, with a command per case
- [`docs/dev/chain-setup/README.md`](docs/dev/chain-setup/README.md) — bringing each chain up by hand
- [`env/docker/README.md`](env/docker/README.md) — the local fleet that stands in for remote servers
- [`docs/SECURITY_KEY_HANDLING.md`](docs/SECURITY_KEY_HANDLING.md) — key handling

---

## Contributing

1. Create a feature branch (`git checkout -b feat/my-feature`).
2. Keep `internal/core` free of chain and consensus imports; chain-specific code
   lives in `internal/chains/*` and `internal/consensus/*`.
3. Use [Conventional Commits](https://www.conventionalcommits.org/).
4. Ensure `make fmt-check`, `make vet` and `make test` pass.
5. Open a pull request.

---

## License

Licensed under the [Apache License 2.0](LICENSE).
