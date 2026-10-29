# chainbench

> Multi-chain local test bench for geth-family blockchains. It composes
> multi-node networks, verifies consensus, and runs declarative tests from a
> CLI, an MCP server, or a Web dashboard.

[![CI](https://github.com/0xmhha/chainbench/actions/workflows/ci.yml/badge.svg)](https://github.com/0xmhha/chainbench/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.26.8%2B-00ADD8?style=flat&logo=go&logoColor=white)](go.mod)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

**chainbench** brings up multi-node blockchain networks on the local machine,
on SSH servers, or on local docker containers that stand in for servers. It
drives a network step by step from keys to running nodes, controls single nodes
during a test, and runs test cases written as JSON against the network. Each
run leaves a session record and a report.

Supported chains (`chainbench chains`):

| Chain | Consensus family | Binary | Default chain id |
|---|---|---|---|
| `stablenet` | `wbft` | `gstable` | 8283 |
| `wbft` | `wbft` | `gwbft` | 8284 |
| `wemix` | `poa` (etcd) | `gwemix` | 8285 |

A chain that reuses one of the two consensus families can also be supplied as
an external manifest without changing chainbench. chainbench also reproduces a
live PoA to BFT hardfork handoff (wemix to wbft).

---

## Architecture

```
   surfaces     chainbench (CLI)    chainbench-mcp (MCP, stdio)    chainbench-dashboard (Web)
                      |                     |                              |
                      |                     |                   runs the chainbench CLI
                      |                     |                   for jobs (internal/webcli)
                      v                     v                              |
   use cases    internal/app  <------------------------------------------+
                      |
   engines      internal/chainsetup (compose)   internal/testengine, internal/dsl (tests)
                      |
   core         internal/core/*  (registry, keyring, genesis, nodeconfig, process,
                                  remote, rpc, session, lifecycle, ...)
                internal/resource (servers, port slots, placement)
                      |
   plugins      internal/consensus/{wbft,poa}    internal/chains/{stablenet,wbft,wemix,common,external}
```

`internal/core` never imports a chain or a consensus package. Tests in
`internal/arch` read the import graph and fail on any such edge. See
[`docs/dev/architecture/architecture-v2.md`](docs/dev/architecture/architecture-v2.md).

---

## Installation

```bash
git clone https://github.com/0xmhha/chainbench.git
cd chainbench

make build        # chainbench, chainbench-dashboard, chainbench-mcp into bin/
make test         # unit and integration tests; no chain binary needed
```

| Dependency | Version | Required for |
|---|---|---|
| [Go](https://go.dev/dl/) | 1.26.8 (`go.mod`) | building chainbench |
| Node.js and npm | 24 in CI | rebuilding the Web SPA and running the Web tests |
| python3 | any recent | `make check` guards, `tests/webui` runners, repro scripts |
| a chain binary | | launching real nodes (`gstable`, `gwbft`, `gwemix`), built from its own repository |

Other ways to install:

- `./setup.sh` checks for go, python3 and curl, builds all three binaries into
  `bin/`, and symlinks `chainbench` and `chainbench-mcp` into `/usr/local/bin`
  (it may ask for sudo). It does not symlink `chainbench-dashboard`.
- `install.sh` is a curl installer that clones the repository into
  `~/.chainbench` and runs the setup there. `uninstall.sh` removes the
  `/usr/local/bin` symlink and `~/.chainbench`, including the session records
  stored under it.

`make help` lists every make target. A single binary builds with
`make chainbench`, `make chainbench-dashboard` or `make chainbench-mcp`.

### The Web SPA and the Go build

The dashboard embeds the built Svelte SPA from `internal/dashboard/spa/`
(`//go:embed all:spa`). The built files are committed, so `make build` works
without Node. After changing anything under `web/`, rebuild the SPA first and
then the Go binary:

```bash
npm --prefix web ci
npm --prefix web run build      # vite build into internal/dashboard/spa/
make chainbench-dashboard
```

---

## Quick Start

Compose a four-producer stablenet network into a workspace, verify it, run a
case against it, and stop it.

```bash
chainbench chain up --workspace-dir /tmp/cb \
  --chain stablenet --binary /path/to/gstable --bp 4 --en 1

chainbench verify --workspace-dir /tmp/cb    # confirm block production
chainbench status --workspace-dir /tmp/cb    # show the node set

# run a case against the network this workspace already composed
chainbench run --attach --workspace-dir /tmp/cb \
  tests/tc/common/rpc/CT-RPC-001-block-number-advances.json

chainbench chain stop --workspace-dir /tmp/cb
```

Or let a case declare the network, compose it, run it and tear it down in one
command:

```bash
chainbench run tests/tc/common/node/CT-NODE-001-startup-block-production.json \
  --workspace-dir /tmp/cb --binary /path/to/gstable
```

Run the same case on another chain by swapping the chain preset:

```bash
chainbench run tests/tc/common/node/CT-NODE-001-startup-block-production.json \
  --workspace-dir /tmp/cb --chain-preset wemix-bp4 --binary /path/to/gwemix
```

`--plan` prints the network a case and its flags resolve to without composing
anything. Keep workspace paths short: node IPC sockets have a 104-character
limit.

---

## CLI

Run `chainbench <command> --help` for the full flag set of any command. The
global flag `--dashboard <url>` streams events from any command to a running
dashboard.

### Composition (`chain`)

| Command | Purpose |
|---|---|
| `chain up` | compose and launch in one command (runs every step in order) |
| `chain new` | initialize a composition workspace for a chain and its target |
| `chain keys`, `place`, `genesis`, `config`, `build`, `deploy`, `init`, `start` | one step at a time |
| `chain enode` | show each node's enode, derived from keys and placement |
| `chain status` / `show` / `health` | which steps have run, where a node is, each node's head block |
| `chain stop` / `restart` / `rm` | lifecycle of the composed data plane |
| `chain logs` | the last lines of one node's log |
| `chain observe` | observe each recorded node's process and binary without changing anything |
| `chain resume` | recover a workspace whose run died and continue from the first unfinished step |
| `chain blueprint` | write the network a key set would compose as an editable declaration |

Main `chain up` flags:

| Flag | Default | Description |
|---|---|---|
| `--workspace-dir` | `~/.chainbench/<timestamp>/chainsetup` | where the composition and its records live |
| `--chain` | | `stablenet`, `wbft` or `wemix`; ignored with `--manifest` |
| `--binary` | | node binary path (required for `--stage start`) |
| `--bp` / `--en` / `--pn` | `4` / `0` / `0` | producer, endpoint and proxy-tier node counts |
| `--topology` | | per-node layout YAML; overrides the counts |
| `--blueprint` | | network declaration YAML with layout and node keys; `--keys-source` then defaults to `declared` |
| `--chain-preset` | | compose a chain-preset document as a test run would |
| `--keys` | `presets/keys` | key set the network composes from |
| `--keys-source` | `preset` | `preset`, `generate`, or `declared` |
| `--set` / `--overlay` | | override a genesis config key (repeatable) / deep-merge a JSON overlay |
| `--genesis-existing` | | use a finished genesis file instead of building one |
| `--manifest` / `--genesis-template` | | an external chain manifest and its genesis template |
| `--binaries name=path` | | resolve a topology binary name (repeatable) |
| `--launch-opt k=v` | | override a launch option (repeatable) |
| `--peering` | `mesh` | `mesh` or `proxied` |
| `--stage` | `start` | stop after `deploy`, `init` or `start` |
| `--target` | | data plane location: `/path`, `user@host:/path` or `ssh://user@host:port/path` |
| `--server-set` / `--workspace-config` | `server-set.yaml` when present | which servers exist / where the data root lives on them |
| `--all-servers` / `--server` / `--server-index` | | how nodes are spread over the server set |
| `--docker` | `false` | servers are local docker containers; translate dials through the localmap |
| `--plan` / `--json` | | print the resolved plan without composing / emit JSON |

### Tests

| Command | Purpose |
|---|---|
| `run` | run DSL specs: compose the declared network, or attach to a running one |
| `validate` | parse and check specs with no chain binary and no network |
| `test list <dir>` | list the runnable DSL cases under a directory |
| `migrate-spec <v1-spec.json>` | convert a v1 spec to the v2 grammar (`--out` writes a file) |
| `report` | show a run's report from its session directory |
| `baseline` | the approved fingerprint of an environment's prepared inputs |

`run` starts in one of three ways, and the combinations are checked
(`internal/app/start.go`):

| Form | Meaning |
|---|---|
| `run --workspace-dir <ws> <spec>` | compose the network the specs declare, run, tear down |
| `run --attach --workspace-dir <ws> <spec>` | run against the network that workspace already composed |
| `run --rpc <url> --chain <id> <spec>` | attach to a live network by node RPC URL (repeatable) |

`--attach` does not combine with `--rpc`, and `--rpc` does not combine with
`--workspace-dir`. A spec may also declare `env.attach` itself. Other `run`
flags include `--chain-preset`, `--chain-presets` (repeatable, targets run
serially), `--suite`, `--keep-up`, `--plan`, `--no-skips`, `--json`,
`--keys-source keyPreset|generate`, `--binary-for name=path`, `--wait-blocks`,
`--artifact-root` (default `~/.chainbench/sessions`), the server-set flags shown
for `chain up`, and `--docker`.

### Inspection and operations

| Command | Purpose |
|---|---|
| `chains` / `capabilities` | registered chains; what each one exposes |
| `status` / `stop` | a workspace's node set; stop its nodes by the PIDs it records |
| `verify` | confirm a network is producing blocks (`--baseline`, `--validators` add checks) |
| `consensus` | query the validator set through the chain's own RPC namespace |
| `node` / `network` | control one node; attach to and call running networks by name |
| `tx` / `contract` / `account` / `faucet` | send and wait, deploy and call, inspect, fund |
| `keyring` / `validator` | key sets (`new`, `add`, `list`, `show`, `import`, `export`); validator identities |
| `resource` | the servers, port slots and capacity a network may be composed from |
| `query` | the read-only commands, gathered |
| `log` / `file` | search per-node logs; copy files to and from a server |
| `hardfork` | plan a binary swap at a fork block, keeping node data |
| `clean` | remove a network's data dir, GC sessions (`--older-than`, `--keep-last`), or list stale compositions on servers (`--stale-compositions`, `--apply`) |
| `completion` | shell completion script |

---

## Hardfork Handoff

A PoA to BFT handoff is declared, not scripted. The from-chain (`wemix` with
etcd) produces up to a fork height, then the successor validators (`wbft`),
which synced the pre-fork chain, take over after it.

```bash
GWEMIX_BIN=/path/to/gwemix GWBFT_BIN=/path/to/gwbft \
chainbench run tests/tc/go-wemix/hardfork/01-croissant-successors-take-over.json \
  --workspace-dir /tmp/cb
```

The case names the chain preset
[`presets/chain/wemix-to-wbft.json`](presets/chain/wemix-to-wbft.json). Its
binaries default to `${GWEMIX_BIN:-gwemix}` and `${GWBFT_BIN:-gwbft}`, and its
`upgrade` block points at the successor binary:

```json
"upgrade": { "fork": "croissant", "at": 20,
             "from": "default", "to": "next", "style": "concurrent" }
```

[`presets/chain/README.md`](presets/chain/README.md) records how a hardfork is
declared and what is checked. To plan a binary swap against a running
workspace, use `chainbench hardfork`.

---

## MCP Server

`chainbench-mcp` is a Go MCP server that reads newline-delimited JSON-RPC on
stdin and writes to stdout. Register it like this:

```json
{ "mcpServers": { "chainbench": { "command": "chainbench-mcp" } } }
```

Its tools cover chain composition (`chainbench_chain_*`), networks, nodes,
key sets, resources, runs and validation, transactions, contracts, consensus
queries, logs and remote RPC. `tools/list` enumerates exactly what a build
exposes; the tool set grows with the chains the binary imports.

Beyond the `chainbench_*` tools, chainbench exposes a capability catalog of
features common to every chain plus chain-specific ones. Each capability is an
MCP tool named `chainbench.<version>.<chain>.<name>`, for example
`chainbench.v1.stablenet.governance.propose_mint`. Call the
`chainbench.capabilities` tool, or `chainbench capabilities [--chain]` on the
CLI, to discover what a chain supports.

A capability is declared as data and bound in code inside the chain's own
package. The catalog is a `.jsonl` file of `registry.Descriptor` lines
(`internal/chains/common/common.jsonl`, `internal/chains/{stablenet,wemix}/caps.jsonl`)
that the package embeds and loads with `registry.LoadCatalog` in its `init`. A
capability appears in discovery and in `tools/list` only once it is bound,
either by `registry.RegisterHandler` or by `registry.RegisterFlat`, which maps
an existing `chainbench_*` tool into the catalog. The chain package must be
blank-imported from `internal/chains/all/all.go`, or its `init` never runs. See
[`internal/core/registry/README.md`](internal/core/registry/README.md) and
[`docs/guide/mcp.md`](docs/guide/mcp.md).

---

## Web Dashboard

`chainbench-dashboard` serves the Web UI at `/` and its API under `/api/v1`.
The SPA screens are `/`, `/chains`, `/tests`, `/monitoring`, `/history` and
`/settings`, with detail pages under `/jobs/<id>` and `/nodes/<id>/<name>`.
`/legacy` is a no-JS fallback page and `/healthz` a health check. The one
Server-Sent Events stream at `/api/v1/events` carries job changes, what the
collector observes and the engine events CLI runs publish. The API contract is
[`docs/dev/web-ui.openapi.yaml`](docs/dev/web-ui.openapi.yaml).

The Web does not carry its own execution engine: jobs run the `chainbench` CLI,
found beside the dashboard binary, then on PATH, or named with `-chainbench`.
`make build` puts both in `bin/`. The current Web UI status and remaining work
are in [`docs/dev/web-ui-status.md`](docs/dev/web-ui-status.md).

| Flag | Default | Description |
|---|---|---|
| `-addr` | `127.0.0.1:8787` | listen address |
| `-accounts` | `false` | require registered accounts |
| `-web-mode` | `personal` | `personal` or `team` |
| `-deployment-root` | `chainbench-out/web-ui` | persistent data directory |
| `-chainbench` | beside the binary, then PATH | the CLI the Web runs |
| `-chain-presets` | `presets/chain` | chain preset directory |
| `-chain-analysis` | `docs/chain-analysis` | captured chain binary documentation |
| `-manifest-keys` | `presets/keys` | key preset for isolated manifest setup |
| `-manifest-assets` | | server-provisioned binary asset JSON enabling manifest management |
| `-artifact-root` | | session artifacts to serve under `/api/sessions` |
| `-tls-cert` / `-tls-key` | | TLS for the web service |
| `-trusted-proxy` | | TLS terminators whose `X-Forwarded-Proto` is trusted |
| `-legacy-observation` | off | loopback-only legacy event observer without control APIs |

By default the Web opens on loopback without registration or a password. It
uses a stable local operator for jobs and credentials while keeping browser
CSRF checks, redaction and audit records:

```bash
chainbench-dashboard -addr 127.0.0.1:8787
```

To require accounts, the first administrator signs up with the one-time
`setup.token` in the data directory:

```bash
chainbench-dashboard -accounts=true -addr 127.0.0.1:8787
```

Account-free access accepts `localhost` and loopback addresses only. It refuses
to start with `-web-mode team`, `-trusted-proxy` or TLS, and it rejects requests
that carry reverse-proxy forwarding headers. Switching modes preserves stored
accounts and artifacts but does not transfer personal credentials or job
ownership between a registered user and the local operator.

Team mode on an address other machines reach must terminate TLS itself or name
the TLS terminator in front of it:

```bash
chainbench-dashboard -accounts=true -web-mode team -addr 0.0.0.0:8443 -tls-cert cert.pem -tls-key key.pem
chainbench-dashboard -accounts=true -web-mode team -addr 10.0.0.2:8787 -trusted-proxy 10.0.0.5
```

CLI runs send their events in one of two ways. With a publisher token, the
service keeps every control API and the token authorizes event ingestion only:

```bash
export CHAINBENCH_PUBLISHER_TOKEN=$(openssl rand -hex 32)
chainbench-dashboard -addr 127.0.0.1:8787 &
chainbench verify --workspace-dir /tmp/cb --dashboard http://127.0.0.1:8787
```

A producer without a token is accepted only by the loopback legacy observer,
which serves `/events` and `/api/runs` and has no control APIs and no accounts:

```bash
chainbench-dashboard -legacy-observation -addr 127.0.0.1:8787 &
chainbench verify --workspace-dir /tmp/cb --dashboard http://127.0.0.1:8787
```

The SPA source lives in [`web/`](web/) (Svelte 5 and Vite 6); see
[`web/README.md`](web/README.md).

---

## Configuration Files

| File | Purpose |
|---|---|
| `server-set.yaml` | which servers exist and how to reach them; the default `--server-set` when present. Start from [`server-set.sample.yaml`](server-set.sample.yaml). |
| `workspace-config.yaml` | the target data root and its purpose directories. Start from [`workspace-config.sample.yaml`](workspace-config.sample.yaml). |
| `presets/chain/*.json` | chain presets: layout, genesis, config and launch options a case composes. See [`presets/chain/README.md`](presets/chain/README.md). |
| `presets/keys/` | the committed test key set. See [`presets/keys/README.md`](presets/keys/README.md). |

The real `server-set.yaml` and `workspace-config.yaml` are not committed. How to
write them is in [`docs/guide/config-files.md`](docs/guide/config-files.md).

Default locations: session records under `~/.chainbench/sessions`, workspaces
under `~/.chainbench/<timestamp>/chainsetup`, the default key set under
`~/.chainbench/keys/default` (override with `CHAINBENCH_KEYRING`), and Web data
under `chainbench-out/web-ui`.

Chain presets read some binaries and endpoints from environment variables, for
example `GWEMIX_BIN`, `GWBFT_BIN`, `GSTABLE_TESTNET_RPC` and
`GSTABLE_TESTNET_KEY_FILE`. The preset file names what it reads.

### Local docker servers

[`env/docker/`](env/docker/README.md) runs local Ubuntu containers with sshd
that stand in for remote servers. `env/docker/setup.sh` takes them from nothing
to ready (steps `check`, `generate`, `image`, `up`, `binaries`, `verify`; no
argument runs all of them; `--recreate` recreates the containers). It needs
`env/docker/accounts.env`, created from `accounts.env.sample`, and writes the
server sets, workspace config, localmap and compose file into
`env/docker/build/`. A run then targets the fleet with `--server-set`,
`--workspace-config`, `--docker` and `--all-servers`.

---

## Tests and Checks

```bash
make check        # fmt-check vet lint test tc-review sweep-test case-docs fixture-test
make test         # unit and integration; no chain binary required
make test-race    # with -race
make test-e2e     # go test -tags e2e; tests skip without chain binaries
make lint         # golangci-lint at the version in .golangci-version
make secrets      # secret scan, the pre-commit gate
make cover        # coverage.html
```

The unit tests are deterministic (httptest and fake binaries), so they pass
without any real chain binary. Web checks:

```bash
npm --prefix web run check                 # svelte-check
node --test tests/webui/*.test.mjs
WEBUI_RUNTIME_ROOT=/private/tmp/cbui PYTHONDONTWRITEBYTECODE=1 \
  python3 -m unittest discover -s tests/webui -p 'test_*.py'
```

CI (`.github/workflows/ci.yml`) runs the secret scan, `bash -n` over shell
scripts, the Web job above, gofmt, `go vet ./...`, `go build ./...`,
`go test -race ./...`, `make tc-review`, `make case-docs`, `make sweep-test`,
`go vet -tags e2e ./...` and golangci-lint.

The live Web UI acceptance runners (`tests/webui/verify.sh --criterion WEB-NN
--require-live`) need real chain binaries and owned fixtures; see
[`tests/webui/README.md`](tests/webui/README.md).

### Test corpus

| Directory | What is in it |
|---|---|
| `tests/tc/basic/` | 2 basic cases (wbft consensus, an attached chain) |
| `tests/tc/common/` | 71 cases that run on all three chains |
| `tests/tc/go-stablenet/` | 132 stablenet cases (regression, post-v1.0.0-change, vocabulary, mixed binaries, hardfork, external testnet) |
| `tests/tc/go-wbft/`, `go-wemix/` | 8 wbft and 7 wemix cases |
| `tests/e2e/` | Go end-to-end tests behind `-tags e2e` |
| `tests/repro/` | shell reproductions |
| `tests/webui/` | Web UI Node and Python tests and live acceptance runners |
| `examples/specs/` | v1 example specs; `chainbench migrate-spec` converts them to v2 |

Run a corpus directory, one network per case:

```bash
scripts/tcsweep.sh sweep.log tests/tc/common
```

See [`tests/tc/HOW-TO-USE.md`](tests/tc/HOW-TO-USE.md) for a command per case
and [`tests/README.md`](tests/README.md) for the conventions.

---

## Security

> [!CAUTION]
> **Every key in this repository is a test fixture. Never use one anywhere real.**

The validator keys, keystores and addresses under `presets/keys/` exist only for
reproducible local testing. They are committed to a public repository with a
trivially decryptable keystore, so their private keys are effectively public
knowledge.

Never use any key, keystore or address that appears in this repository (in
`presets/keys/`, in presets, in manifests, or in tests) on any production,
mainnet, testnet, staging or shared network. Anyone who can read this
repository can sign as any of them, so any funds or authority given to one is
already gone.

This covers every key in the tree, whatever produced it:

- the node identities under `presets/keys/node{1..5}/`, generated once and
  committed so a local network comes up with the same validators every time;
- the dev accounts under `presets/keys/dev1/`, which the harness mints on the
  first run that asks for one and reads back afterwards;
- plaintext private keys written inline in test source. The genesis-funded
  faucet key is the upstream go-ethereum test key, already public in every geth
  fork.

A secret scanner reports nothing on this repository by decision.
[`.betterleaks.toml`](.betterleaks.toml) allowlists exactly these paths and
leaves every rule of the default set on. **Treat a finding as real if it is
outside `presets/keys/` and outside test code.** Credentials for real servers
belong in private encrypted stores, not in the repository.

See [`presets/keys/README.md`](presets/keys/README.md) and
[`docs/SECURITY_KEY_HANDLING.md`](docs/SECURITY_KEY_HANDLING.md).

---

## Adding a Chain

**Project-supplied, no chainbench change.** If your chain reuses a built-in
consensus family (`wbft` or `poa`), point chainbench at your own manifest:

```bash
chainbench chain up --workspace-dir /tmp/my \
  --manifest         ../my-chain/chainbench.json \
  --genesis-template ../my-chain/genesis.json \
  --keys ../my-chain/keys --binary ../my-chain/bin/gmychain
```

The manifest uses the same schema as
[the built-in chains' `manifest.json`](internal/chains). Set `"protocol"` to a
built-in accounts profile (`stablenet`, `wbft` or `wemix`) to borrow its
transaction types and account model.

**First-party, embedded in the tool.** Add `internal/chains/<id>/manifest.json`
and `internal/chains/<id>/genesis.json`, a thin plugin at
`internal/chains/<id>/<id>.go` that embeds them and registers via
`registry.Register` in its `init`, and a blank import in
`internal/chains/all/all.go`. Only a new consensus algorithm needs a new
`internal/consensus/<family>`.

---

## Documentation

- [`docs/README.md`](docs/README.md): the documentation index
- [`docs/guide/dsl-authoring.md`](docs/guide/dsl-authoring.md): how to write a test case
- [`docs/guide/config-files.md`](docs/guide/config-files.md): `server-set.yaml` and `workspace-config.yaml`
- [`docs/guide/topology.md`](docs/guide/topology.md): the `--topology` file and layout sources
- [`docs/guide/keyring.md`](docs/guide/keyring.md): key sets and the `keyring` commands
- [`docs/guide/validator-set.md`](docs/guide/validator-set.md): validator sets larger than the committed preset
- [`docs/guide/mcp.md`](docs/guide/mcp.md): using the MCP server
- [`docs/dev/architecture/architecture-v2.md`](docs/dev/architecture/architecture-v2.md): layers and the import rule
- [`docs/dev/chain-setup/README.md`](docs/dev/chain-setup/README.md): bringing each chain up by hand
- [`docs/dev/web-ui-status.md`](docs/dev/web-ui-status.md): Web UI status and remaining work
- [`tests/tc/HOW-TO-USE.md`](tests/tc/HOW-TO-USE.md): running any case, with a command per case
- [`env/docker/README.md`](env/docker/README.md): the local fleet that stands in for remote servers
- [`docs/SECURITY_KEY_HANDLING.md`](docs/SECURITY_KEY_HANDLING.md): key handling

---

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md). In short: keep `internal/core` free
of chain and consensus imports, use [Conventional Commits](https://www.conventionalcommits.org/),
and make sure `make check` passes. After a `web/` change, rebuild the SPA with
`npm --prefix web run build` before the Go build and commit the result.

---

## License

Licensed under the [Apache License 2.0](LICENSE).
