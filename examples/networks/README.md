# Network examples

Ready-to-copy `state/networks/<name>.json` files. A network state file describes
a single network as a list of **nodes**, each bound to a **provider** that
declares what the node can do.

## Per-node provider model

Every node carries a `provider`:

| provider | capabilities | meaning |
|----------|--------------|---------|
| `local` | `admin, fs, network-topology, process, rpc, ws` | a node this host runs (full lifecycle + filesystem + process control) |
| `remote` | `rpc, ws` | a node reached only over JSON-RPC / WebSocket (no process or filesystem access) |
| `ssh-remote` | `fs, process, rpc, ws` | a node on another host reached over SSH. RPC is tunneled through the SSH connection (5b.1); `tail_log` (fs) and node stop/start/restart (process) run as SSH shell commands (5b.2). |

A network may mix providers freely — that is a **hybrid** network.

## Capability lower bound

A network's capabilities are the **set intersection** of its nodes' provider
capabilities — the conservative lower bound of what *every* node can satisfy.
For `hybrid-example.json` (3 `local` + 1 `remote`):

```
local  ∩ remote = {admin,fs,network-topology,process,rpc,ws} ∩ {rpc,ws} = {rpc, ws}
```

So a hybrid network exposes only `rpc` and `ws`. Operations that require
`process` (node stop/start) or `fs` (log tailing) are automatically gated:
tests carrying `requires_capabilities: [process]` are **skipped** against this
network, because not every node can be controlled.

## Using `hybrid-example.json`

```bash
# 1. Materialize the network state (v1 flow is a manual copy — see note below)
mkdir -p state/networks
cp examples/networks/hybrid-example.json state/networks/hybrid-example.json

# 2. Query its capabilities (pure state-file read — no RPC dial)
chainbench-net <<'EOF'
{"command":"network.capabilities","args":{"network":"hybrid-example"}}
EOF
# => { "network": "hybrid-example", "capabilities": ["rpc", "ws"] }
```

Through MCP the same lower bound is returned by
`chainbench_network_capabilities({ network: "hybrid-example" })`.

## Remote auth

A `remote` node may reference credentials by **environment variable name only**
— never inline the header value (it would land in a committed file). The
`auth` block names the env var; the driver reads it at dial time:

```json
"auth": { "type": "api-key", "header": "Authorization", "env": "CHAINBENCH_REMOTE_API_KEY" }
```

`type` may be `api-key` or `jwt`. Set the value in your shell, e.g.
`export CHAINBENCH_REMOTE_API_KEY=...`.

## SSH-remote nodes (`ssh-remote-example.json`)

An `ssh-remote` node reaches a chain node on another host by **tunneling RPC
through an SSH connection** — useful when the node's RPC port is bound to the
remote host's loopback and not exposed publicly. Read-only RPC works in Sprint
5b.1; the node's `http` is the RPC endpoint as seen *from the remote host*
(e.g. `http://127.0.0.1:8545`).

Auth is `ssh-password`. As with `remote` auth, the password is supplied by an
**environment variable named in `env`** — never inline the password:

```json
"auth": { "type": "ssh-password", "user": "deploy", "host": "10.0.0.42", "port": 22, "env": "CHAINBENCH_SSH_PASSWORD" }
```

```bash
export CHAINBENCH_SSH_PASSWORD=...        # the SSH password (never stored/logged)
```

**Host key verification** (MITM protection) defaults to `known_hosts`
(`CHAINBENCH_SSH_KNOWN_HOSTS`, else `~/.ssh/known_hosts`). An unknown or
mismatched host key is rejected. To bypass for an ephemeral test host, set
`CHAINBENCH_SSH_INSECURE_HOST_KEY=1` (a loud, explicit opt-in — do not use
against hosts you care about).

### process / fs over SSH (`provider_meta`)

`tail_log` and node lifecycle (stop/start/restart) on an ssh-remote node run as
**SSH shell commands you configure in `provider_meta`**:

```json
"provider_meta": {
  "log_file":    "/var/lib/gstable/node.log",
  "start_cmd":   "systemctl start gstable",
  "stop_cmd":    "systemctl stop gstable",
  "restart_cmd": "systemctl restart gstable"
}
```

- `tail_log` runs `tail -n <lines> -- <log_file>` over SSH.
- `node.stop`/`start`/`restart` run the matching `*_cmd`. If `restart_cmd` is
  absent, restart composes `stop_cmd` → `start_cmd`.
- A node that omits a given command (or `log_file`) reports the capability at
  the network level but returns `NOT_SUPPORTED` for that specific operation.
- These commands are **operator-supplied** (you wrote them in this file) and run
  on the remote shell. They are not caller-controlled — an LLM/test only selects
  the node, never the command string.

## Constructing these networks

- **`ssh-remote`**: `network.attach` builds an ssh-remote node directly — pass
  `provider: "ssh-remote"`, the `ssh-password` `auth`, and `provider_meta`; it
  probes chain_id/chain_type over the SSH tunnel and persists the node. No
  hand-written file needed.
- **`remote`**: `network.attach` builds a pure-remote single node (HTTP probe).
- **hybrid** (mixed local+remote in one network): no dedicated compose command
  yet — copy `hybrid-example.json` into `state/networks/<name>.json` (or
  hand-write one against
  [`network/schema/network.json`](../../network/schema/network.json)). A compose
  command and a `chainbench network attach` CLI/MCP surface are planned follow-ups.

The structure of `hybrid-example.json` is exercised end-to-end by
`tests/unit/tests/network-hybrid-capabilities.sh`.
