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
`attach` — see below.

## execution.chain: how a run treats an existing composition

`fresh` (the default) composes and launches the network as always; it does not
disturb another composition's processes or data.

`reuse-if-matching` reconciles a running network node by node before it deploys
or launches anything. For each node it compares the config and binary against
what the run would produce now, and probes whether the node is still answering.
A node whose inputs are unchanged and that is up is left running; only nodes
that drifted or stopped are torn down and brought back. Of fifteen nodes, if one
changed, one is redone and fourteen keep running. When this workspace has no
record of a node — a first run against a target something is already up on — the
baseline is read back from the running process itself: its `--config` is
recovered from the command line and hashed on the machine, so a node already up
with the config this run would give it is reused in place, and one up with a
different config is refused as foreign rather than composed over. The one
exception is the
genesis: it is shared by every node, so a changed genesis is a different chain
and cannot be reconciled onto a running network — the whole reuse is refused,
and nothing is touched. Use it to continue an expensive environment across runs.

`attach` tests an already-running chain. It does not create, deploy, or init, so
`chain up` refuses it — bring the chain up separately and use the attach/run
path.

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

`--workspace-config` is accepted by the one-shot `run`, by the step-form
`chain new` and `chain up`, and by the MCP tools `chainbench_chain_new` and
`chainbench_chain_up` — every surface resolves the target's `dataRoot` from the
file the same way.

## What is live today

- The workspace-config file format, its validation, and its `dataRoot` are
  live: `dataRoot` comes from this file, and the server-set no longer carries
  one.
- `--workspace-config` is wired into the step-form CLI (`chain new`/`chain up`)
  and the MCP step tools, not only the one-shot `run`.
- `execution.chain` is live for `chain up`: `fresh` (default),
  `reuse-if-matching` (per-node reconciliation, above), and `attach` (refused by
  up).
- `paths`, `binaryAliases`, `inputs`, and `presets` parse and validate, and
  their consumption (purpose-directory resolution, server file references,
  prepared/generated wiring) lands incrementally — see the handoff docs
  `docs/research/chainbench/analyses/09-workspace-config-refactoring-handoff.md`
  and `10-prepared-inputs-server-ref-handoff.md`. Until a field is wired, it is
  parsed but not yet acted on.
