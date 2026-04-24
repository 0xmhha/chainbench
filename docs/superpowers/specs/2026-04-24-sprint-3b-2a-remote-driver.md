# Sprint 3b.2a — RemoteDriver + first remote-node command Design Spec

> 2026-04-24 · Sub-sprint of VISION §6 Sprint 3
> Scope: introduce `go-ethereum/ethclient` dep, add minimal `drivers/remote`
> package, ship `node.block_number` as the first command that works on both
> local and remote networks. Absorb M4 via a new `resolveNode` helper.

## 1. Goal

Make it possible to query `block_number` against any attached network (local
or remote). Introduce the ethclient dep infrastructure that Sprint 4 (signer +
tx.send) and 3b.2b (auth) will build on.

## 2. Non-Goals (Deferred)

- **Authentication (API key / JWT)** — Sprint 3b.2b. Attach and block_number
  work against unauthenticated public RPCs (e.g., local Anvil, public Infura
  without key) in 3b.2a.
- **Generic `node.rpc` passthrough** — YAGNI until needed. `node.block_number`
  is a concrete, typed command.
- **Other remote-node commands** (`node.balance`, `node.tx_send`, etc.) —
  future sprints. Pattern established here; additions are routine.
- **Local lifecycle refactor** — `node.stop/start/restart/tail_log` still
  hardcode `"local"`. They are shell-exec operations with no remote analogue.
  If called with non-local `network`, they return NOT_SUPPORTED (new guard).
- **Stablenet-specific typed wrappers** — use raw `rpc.Client.Call()` when
  needed (same pattern as Sprint 3a probe). ethclient covers standard eth_*.

## 3. ethclient Compatibility Note

- Module path: `github.com/ethereum/go-ethereum`. Go-stablenet is a soft fork
  with the same module path but no new typed ethclient methods.
- Wire protocol: go-stablenet / wbft / wemix all speak standard JSON-RPC for
  `eth_*`. Upstream ethclient works against them unchanged.
- Chain-specific methods (`istanbul_*`, `wemix_*`) go through raw
  `rpc.Client.Call()` — same pattern Sprint 3a probe already uses.

## 4. User-Facing Surface

### 4.1 `node.block_number`

```json
{"command":"node.block_number","args":{"network":"sepolia","node_id":"node1"}}
```

**args:**
- `network` (string, optional) — default `"local"`. Must resolve via
  `state.LoadActive` (either pids.json for local or networks/<name>.json).
- `node_id` (string, required) — must exist in the resolved network.

**Result data on success:**
```json
{"network":"sepolia","node_id":"node1","block_number":12345678}
```

**Errors:**
- `INVALID_ARGS` — missing node_id, bad network pattern, node not in network.
- `UPSTREAM_ERROR` — network state not found, ethclient dial/call failure.

### 4.2 Existing local-only handlers (`node.stop/start/restart/tail_log`)

These accept an optional `network` arg. If present and not `"local"`, return
`NOT_SUPPORTED` with a message that remote lifecycle management is not
supported. If absent or `"local"`, existing behavior unchanged.

## 5. Package Layout

```
network/internal/drivers/remote/
├── client.go       # Client struct + Dial + BlockNumber + Close
└── client_test.go  # TDD — httptest.Server mocking JSON-RPC
```

**Package API:**
```go
package remote

type Client struct {
    rpc *ethclient.Client
}

func Dial(ctx context.Context, url string) (*Client, error)
func (c *Client) BlockNumber(ctx context.Context) (uint64, error)
func (c *Client) Close()
```

Thin wrapper. Seam for future auth Transport injection (3b.2b), method grouping,
and mock injection. No direct ethclient exposure to handlers — callers talk to
`remote.Client`.

## 6. New Handler Infrastructure

### 6.1 `resolveNode` helper

Added alongside existing `resolveNodeID` / `resolveNodeIDFromString`:

```go
// resolveNode resolves (network, node_id) → (*types.Network, *types.Node).
// networkName=="" defaults to "local". Validates name shape for non-local.
// Errors: INVALID_ARGS for malformed inputs or unknown node;
//         UPSTREAM_ERROR for missing network state file.
func resolveNode(stateDir, networkName, nodeID string) (*types.Network, *types.Node, error)
```

### 6.2 Existing `resolveNodeID` — unchanged, still hardcodes "local"

Local-only handlers continue to use it. M4 absorbed only where needed.

### 6.3 Local-handler remote-network guard

`node.stop/start/restart/tail_log` get a cheap pre-check:
```go
if req.Network != "" && req.Network != "local" {
    return nil, NewNotSupported("operation only supported on the local network")
}
```

`APIError` already has `NewNotSupported` constructor.

## 7. Dependency Impact

New imports bring in go-ethereum's transitive deps:
- `github.com/ethereum/go-ethereum` (root)
- Likely: holiman/uint256, go-stack, btcsuite/btcd, and many more.

`go.sum` will grow significantly (expected; unavoidable for ethclient).
Document in commit message.

## 8. Testing Strategy

**Unit (`drivers/remote/client_test.go`):**
- Mock server returning `eth_blockNumber` → `"0x10"` (16).
- Dial → BlockNumber → Close.
- Dial failure (bad URL scheme, unreachable).
- BlockNumber RPC error (server returns JSON-RPC error).

**Unit (`handlers_test.go` new tests):**
- `node.block_number` happy path (local + remote mocks).
- Missing node_id → INVALID_ARGS.
- Unknown network → UPSTREAM_ERROR.
- Unknown node_id in network → INVALID_ARGS.
- RPC failure → UPSTREAM_ERROR.
- Local-handler guard: `node.stop` with `network:"remote"` → NOT_SUPPORTED.

**Go E2E (`e2e_test.go`):**
- Attach → node.block_number on attached network → assert 0x10 result.

**Bash E2E (`tests/unit/tests/node-block-number.sh`):**
- Attach + block_number roundtrip via `cb_net_call`.

## 9. Wire Schema

- `command.json` enum: add `"node.block_number"`.
- `event.json` unchanged (Result is untyped per-command).

## 10. Error & Logging

- Dial failures logged at WARN with endpoint (not body).
- No key material involved in 3b.2a (auth is 3b.2b).
- Standard slog through handler dispatcher.

## 11. Out-of-Scope Reminders (session-local)

Still deferred:
- 2b.3 M3 — APIError.Details structured fields
- 2c M3 — jq 3→1 call consolidation
- 2c M4 — jq version gate
- 3a Minor — `isKnownOverride` duplicates signatures enum
- 3b Minor — `created` flag TOCTOU (single-process OK)

New deferrals (introduced by 3b.2a scope):
- **3b.2b** — API key / JWT auth via Transport wrapping; env var scheme
  (`CHAINBENCH_REMOTE_<NAME>_KEY` etc.); handler args for auth.
- **3b.2c (optional)** — generic `node.rpc` passthrough.
- **Sprint 4** — tx.send + signer package + key boundary tests.
