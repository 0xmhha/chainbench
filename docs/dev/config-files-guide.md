# server-set and workspace-config: creating the two files

chainbench takes two site-specific files so one DSL runs unchanged across
local, docker, and remote targets. You create them once per environment and
name them on the command line; you never edit the DSL for a new environment.

Neither real file is committed — only the two `*.sample.yaml` templates at the
repo root are tracked. Copy a sample, fill it in, and keep the real file
outside the repo or under a gitignored name (`*workspace-config.yaml`,
`*server-set.yaml`).

## What each file owns

| File | Owns | Does not own |
|---|---|---|
| **server-set** | the servers chainbench may use: their names, addresses, SSH login, port bands, and how many slots each host holds | the data root, test content |
| **workspace-config** | where files live on the target: `dataRoot` and the purpose directories under it, plus the local results root and how inputs are prepared | the server list, credentials, node count/roles |
| DSL (test spec) | test content, node composition, logical file references | absolute paths, credentials |

The split is the point: change servers by swapping the server-set, change the
target's directory layout by swapping the workspace-config, and the DSL stays
the same.

## Create a server-set

```sh
cp server-set.sample.yaml server-set.yaml   # gitignored
```

Fill in `pool.hosts` (a loopback address runs nodes on this machine; anything
else is reached over SSH), `slots`, `ports`, and the `ssh` block. A one-host
pool with several slots is this machine running a whole network; a many-host
pool with one slot each spreads one node per server.

`dataRoot` is **not** set here any more — it moved to workspace-config (see
below). A server-set that still sets `dataRoot` is rejected with a message
pointing here.

## Create a workspace-config

```sh
cp workspace-config.sample.yaml workspace-config.yaml   # gitignored
```

Set `dataRoot` to the absolute path on the target under which everything lives
(on a remote host, a path on that host; local, a path on this machine). Under
it, `paths` names one directory per purpose. `control.artifactRoot` is the one
local path — where run results and the final report land on the machine running
chainbench (`~` expands locally).

`inputs.mode` is `generated` (build test keys and genesis/config with the
in-process builders) or `prepared` (use files already on the target, named by a
`presets` bundle). `execution.chain` is `fresh`, `reuse-if-matching`, or
`attach`.

## Run with both files

```sh
chainbench run tests/tc/example.json \
  --server-set  ./server-set.yaml \
  --workspace-config ./workspace-config.yaml \
  --workspace-dir /tmp/control/chain-a
```

`--workspace-dir` is the local control directory (where the composition's state
is recorded); it is separate from the target's `dataRoot`. Add `--docker` when
the server-set's hosts are local docker containers.

## What is live today

- The workspace-config file format, its validation, and its `dataRoot` are
  live: `dataRoot` comes from this file, and the server-set no longer carries
  one.
- `paths`, `binaryAliases`, `inputs`, `execution`, and `presets` parse and
  validate, but their consumption (purpose-directory resolution, server file
  references, prepared/generated wiring, reuse and attach modes) lands
  incrementally — see the handoff docs
  `docs/research/chainbench/analyses/09-workspace-config-refactoring-handoff.md`
  and `10-prepared-inputs-server-ref-handoff.md`. Until a field is wired, it is
  parsed but not yet acted on.
