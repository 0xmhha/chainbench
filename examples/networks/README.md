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
| `ssh-remote` | `rpc, ws` | a node on another host reached over SSH. Read-only RPC is tunneled through the SSH connection (Sprint 5b.1). `fs`/`process` over SSH shell exec arrive in 5b.2. |

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

## Note: construction is manual in v1

There is no dedicated command yet to *compose* a hybrid network from a running
local network plus a remote alias — `network.attach` builds pure-remote
networks and `remote add` populates a separate registry. Copying an example
into `state/networks/` (or hand-writing one against
[`network/schema/network.json`](../../network/schema/network.json)) is the
supported v1 path; a construction command is a planned follow-up.

The structure of `hybrid-example.json` is exercised end-to-end by
`tests/unit/tests/network-hybrid-capabilities.sh`.
