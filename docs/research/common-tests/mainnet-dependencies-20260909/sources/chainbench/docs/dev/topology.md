# Local network topology (`setup --topology`)

`chainbench setup` normally builds a local network from positional counts —
`--validators N --endpoints M` — which assigns roles by position and gives every
endpoint the same sync mode. A **topology file** replaces that with an explicit,
per-node layout, so the same file both drives the launch and documents exactly
how the chain is configured.

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
  (non-producing RPC node), or `boot`. At least one producer is required.
- **sync_mode** — `full` (default), `snap`, or `archive`. Set per node; an
  explicit value wins over the role-based default.
- **bootnode** — at most one node may set it; surfaced and persisted (the peering
  mesh itself is unchanged for now).

See [`examples/topology.yaml`](../../examples/topology.yaml).

## Use

```sh
chainbench setup --topology examples/topology.yaml --dry-run        # inspect the plan
chainbench setup --topology examples/topology.yaml --launch --binary <path>
```

`--topology` overrides `--validators`/`--endpoints`. On `--provision`/`--launch`
the file is copied into the data root as `topology.yaml`, so a running network's
layout is inspectable from its datadir.

## How it plugs in

- `internal/core/topology` — the config model: `Load`, `Validate`, role/sync
  normalization (`bp`→validator, `en`→endpoint), bootnode/counts helpers.
- `setup.BuildPlanWithTopology(cfg, plugin, dataRoot, *topology.Topology)` — when
  a topology is supplied it drives each node's role and sync mode; otherwise the
  positional counts are used (`BuildPlan` is the `topo == nil` wrapper).
- `driver.NodeSpec.SyncMode` — the per-node sync mode; the launch config
  generation prefers it over the role default (`effectiveSyncMode`).
