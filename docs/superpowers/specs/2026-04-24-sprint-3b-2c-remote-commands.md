# Sprint 3b.2c — Additional Remote Commands + M4 Full Absorption Design Spec

> 2026-04-24 · Sub-sprint of VISION §6 Sprint 3
> Scope: detailed expansion of remote read-only surface. Three new commands,
> `dialNode` helper extraction, attach-time auth validation, and M4 (resolveNodeID
> hardcoded "local") full absorption.

## 1. Goal

Make an attached remote network genuinely usable for common read-only queries:
chain_id, balance, gas_price. Share the dial+auth boilerplate across all
remote handlers via a `dialNode` helper. Tighten `network.attach` so a
structurally-invalid `auth` payload is rejected up front rather than failing
later at dial time.

## 2. Non-Goals (Deferred)

- **`tx.send` / signer** — Sprint 4.
- **Generic `node.rpc` passthrough** — structurally loose; YAGNI until a real
  use case appears.
- **WebSocket subscriptions** — `node.subscribe_new_heads` etc. future.
- **401/403 distinct APIError code** — ethclient error surface is generic; will
  fold UPSTREAM_ERROR for now. Can promote to separate code if operator
  feedback demands it.
- **Per-block historical queries beyond balance** — `node.balance` takes an
  optional `block_number`; other commands are latest-only in 3b.2c.

## 3. User-Facing Surface

### 3.1 `node.chain_id`

Args: `{network?: string (default "local"), node_id: string}`
Result: `{network, node_id, chain_id: number}` (chain_id from `eth_chainId`).

### 3.2 `node.balance`

Args: `{network?, node_id, address: string (0x-prefixed hex), block_number?: number|string}`
- `block_number` defaults to `"latest"`. Accept integer (treated as hex block number), `"latest"`, `"earliest"`, `"pending"`.
Result: `{network, node_id, address, block, balance: string (0x-prefixed hex wei)}`
- balance returned as hex string to preserve precision beyond `float64` range.

### 3.3 `node.gas_price`

Args: `{network?, node_id}`
Result: `{network, node_id, gas_price: string (0x-prefixed hex wei)}`
- Uses `eth_gasPrice` (legacy pre-EIP-1559 tip). For tip+baseFee we'd need a
  separate command (`node.fee_history`) — 3b.2c scope is minimal.

### 3.4 `network.attach` — auth validation at attach time

Current (3b.2b): attach persists any auth payload; errors surface at dial time.
New: `ValidateAuth(auth)` runs post-unmarshal, before SaveRemote. Invalid
payload → `INVALID_ARGS` with specific message.

**Validation rules (match `remote.AuthFromNode`):**
- `auth.type` must be one of `"api-key"`, `"jwt"`, `"ssh-password"` (ssh is persisted but unused by RPC; validation still passes it).
- `auth.env` required non-empty for `api-key` and `jwt`.
- Unknown types → INVALID_ARGS.

### 3.5 Existing handlers — no user-facing changes

M4 absorption is internal. `resolveNodeID` continues to return `(nodeID, num, error)` — the local lifecycle handlers still call it unchanged. Its implementation now delegates to `resolveNode(stateDir, "local", nodeID)`.

## 4. Architecture Changes

### 4.1 `remote.Client` — new read-only methods

```go
func (c *Client) ChainID(ctx context.Context) (*big.Int, error)
func (c *Client) BalanceAt(ctx context.Context, address common.Address, blockNumber *big.Int) (*big.Int, error)
func (c *Client) GasPrice(ctx context.Context) (*big.Int, error)
```

Thin wrappers over `ethclient.Client` counterparts. Errors wrap via
`fmt.Errorf("remote.<Method>: %w", err)` per package convention.

### 4.2 `dialNode` helper (in handlers package)

```go
// dialNode opens a remote.Client for the given node, wiring auth if Node.Auth
// is populated. Caller owns the Close() lifecycle. Returns the same
// APIError-wrapped errors as the existing block_number handler.
func dialNode(ctx context.Context, node *types.Node) (*remote.Client, error)
```

Body:
```go
var rt http.RoundTripper
if len(node.Auth) > 0 {
    got, aerr := remote.AuthFromNode(node, os.Getenv)
    if aerr != nil {
        return nil, NewUpstream("auth setup", aerr)
    }
    rt = got
}
client, err := remote.DialWithOptions(ctx, node.Http, remote.DialOptions{Transport: rt})
if err != nil {
    return nil, NewUpstream(fmt.Sprintf("dial %s", node.Http), err)
}
return client, nil
```

Exactly what `newHandleNodeBlockNumber` does today, factored out. All four
remote handlers (block_number, chain_id, balance, gas_price) use it.

### 4.3 `remote.ValidateAuth(auth types.Auth) error`

```go
// ValidateAuth reports whether auth is a structurally valid Auth payload.
// Returns nil for nil/empty auth (unauthenticated). Returns typed errors
// for unknown type or missing required fields. Call this at input boundaries
// (e.g., network.attach) to fail fast on malformed auth configuration.
func ValidateAuth(auth types.Auth) error
```

Used by `newHandleNetworkAttach` post-unmarshal to reject invalid auth with
INVALID_ARGS before persistence. `AuthFromNode` retains its own runtime
checks as defense-in-depth.

### 4.4 M4 absorption — `resolveNodeID` via `resolveNode`

Current `resolveNodeIDFromString` duplicates `state.LoadActive` + node lookup
from `resolveNode`. New implementation delegates:

```go
func resolveNodeIDFromString(stateDir, nodeID string) (string, string, error) {
    // Validate node<N> prefix + numeric suffix (local-only convention).
    if nodeID == "" { return "", "", NewInvalidArgs("args.node_id is required") }
    if !strings.HasPrefix(nodeID, "node") { return "", "", NewInvalidArgs(...) }
    num := strings.TrimPrefix(nodeID, "node")
    if num == "" { return "", "", NewInvalidArgs("node_id missing numeric suffix") }
    // Delegate existence check to resolveNode.
    if _, _, err := resolveNode(stateDir, "local", nodeID); err != nil {
        return "", "", err
    }
    return nodeID, num, nil
}
```

`resolveNodeID(stateDir, args)` unchanged (still a thin arg-unmarshal wrapper).

Local lifecycle handlers continue to use `resolveNodeID` — no signature
changes, no event regression risk.

## 5. Schema Changes

`command.json` enum gains three entries: `"node.chain_id"`, `"node.balance"`,
`"node.gas_price"`. Regenerate `command_gen.go`.

`network.json` — no change.

## 6. Error Classification

| Path | Code |
|---|---|
| Malformed JSON args | INVALID_ARGS |
| Missing required field (node_id, address) | INVALID_ARGS |
| Invalid address format | INVALID_ARGS |
| Invalid block_number format | INVALID_ARGS |
| Unknown network | UPSTREAM_ERROR |
| Unknown node | INVALID_ARGS |
| Auth setup failure (env var empty etc.) | UPSTREAM_ERROR |
| Dial failure | UPSTREAM_ERROR |
| RPC error (ethclient return) | UPSTREAM_ERROR |
| ValidateAuth failure (attach time) | INVALID_ARGS |

## 7. Testing Strategy

**Unit (remote package):**
- `TestClient_ChainID` / `BalanceAt` / `GasPrice` — httptest mock with JSON-RPC responses
- `TestValidateAuth` — table-driven cases: nil, unknown type, missing env, valid api-key, valid jwt, valid ssh-password (passes through)

**Unit (handlers package):**
- Each of `newHandleNodeChainID`, `_Balance`, `_GasPrice`: happy path + common error paths (missing args, unknown node, RPC failure)
- `newHandleNetworkAttach_InvalidAuthRejected` — pass auth with unknown type, assert INVALID_ARGS, assert state file not written

**Go E2E:**
- Attach a protected mock RPC (api-key auth), then call each new command; assert auth header reached the server.

**Bash E2E:**
- One new file covering all three commands in a single flow (attach → chain_id → balance → gas_price). Keeps bash test count manageable.

## 8. Out-of-Scope Reminders (session-local)

**Still deferred:**
- 2b.3 M3 — APIError.Details
- 2c M3 — jq 3→1
- 2c M4 — jq version gate
- 3a Minor — isKnownOverride SSoT
- 3b Minor — created flag TOCTOU, valid-but-unattached bash wire coverage
- 3b.2a I-2 — Node.Http empty guard (IPC variant)
- 3b.2a M-2 — `resolveNodeID` rename (absorbed functionally in 3b.2c; rename deferred)
- 3b.2b Minor — malformed URL fall-through in DialWithOptions

**New deferrals introduced by 3b.2c scope:**
- `node.fee_history` / EIP-1559 max-priority-fee — standalone follow-up
- `node.subscribe_new_heads` (WebSocket) — Sprint 4+
- Historical queries for `chain_id`/`gas_price` — not useful at block scope
- 401/403 distinct APIError code — promotes from 3b.2b deferred list into 3b.2c deferred list; re-evaluate if operators report confusion
