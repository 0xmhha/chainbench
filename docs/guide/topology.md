# Per-node topology (`--topology`)

> **[가이드]** 검증 기준 2026-09-11 · 모델 `internal/core/node/topology.go:22` ·
> 플래그 `cmd/chainbench/chaincmd/steps.go:84`, `chaincmd/up.go:91`.
> 명령·플래그는 `chainbench chain place --help` 가 이긴다.

`chainbench chain place` normally builds a network from positional counts —
`--validators N --endpoints M --proxies P` — which assigns roles by position and
gives every endpoint the same sync mode. A **topology file** replaces that with an
explicit, per-node layout, so the same file both drives the launch and documents
exactly how the chain is configured.

There are three layout sources, widest last: **counts** → **topology**
(`--topology`) → **blueprint** (`--blueprint`). A blueprint carries the layout *and*
the node keys in one declaration; giving both a blueprint and a topology is an
error (`chainsetup: allocate: a blueprint and a topology both describe the layout`).
`chainbench chain blueprint --from-topology <file>` rewrites a topology as the
wider declaration.

## Format

```yaml
chain: wbft            # selects the chain unless --chain is given
network: local
nodes:
  - { index: 1, role: bp, sync_mode: full }
  - { index: 2, role: en, sync_mode: full, bootnode: true }
  - { index: 3, role: en, sync_mode: archive }
  - { index: 4, role: bp }
```

- **index** — 1-based; the set must be contiguous `1..N` (index = launch order).
- **role** — `bp`/`validator` (block producer / staker), `en`/`endpoint`
  (non-producing RPC node), `pn`/`proxy` (proxy tier), or `boot`. At least one
  producer is required.
- **sync_mode** — `full` (default), `snap`, or `archive`. Set per node; an
  explicit value wins over the role-based default.
- **bootnode** — at most one node may set it.
- **binary** — a per-node binary *name*, resolved to a path by a repeatable
  `--binaries wbft=/path/gwbft`. This is what lets one network run mixed binaries.
- **config** / **key** — per-node config-file and key overrides.

See [`examples/topology.yaml`](../../examples/topology.yaml).

## Use

```sh
# Compose step by step: place the nodes from the file, then continue.
chainbench chain new --chain wbft --workspace-dir /tmp/ws
chainbench chain place --workspace-dir /tmp/ws --topology examples/topology.yaml

# Or in one command.
chainbench chain up --topology examples/topology.yaml --binary <path>

# Mixed binaries, resolved by name.
chainbench chain up --topology examples/topology.yaml \
  --binaries wbft=/path/gwbft --binaries wemix=/path/gwemix
```

`--topology` overrides `--validators`/`--endpoints`. The resulting node table —
roles, hosts, deterministic non-colliding ports — is recorded in the workspace, so
a running network's layout is inspectable with `chainbench chain show` and
`chainbench status`.

## How it plugs in

- `internal/core/node` — the model: `node.Topology`/`node.Entry`, `node.Load`,
  role/sync normalization (`bp`→validator, `en`→endpoint).
- `chainsetup.NetAllocate` (`internal/chainsetup/verbs_steps.go:147`) — resolves the
  three layout sources and refuses two at once, then allocates the node table
  under the server-set lock.
- The DSL declares the same thing inline (`NetAllocateIn.Topology`), so a spec
  needs no side file.
