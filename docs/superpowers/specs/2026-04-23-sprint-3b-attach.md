# Sprint 3b — Remote Network Attach Design Spec

> 2026-04-23 · Sub-sprint of VISION §6 Sprint 3
> Scope: `network.attach <url>` + state persistence. Builds on Sprint 3a probe.

## 1. Goal

Enable attaching an arbitrary EVM RPC endpoint as a named, persisted network
that subsequent `network.load` calls can resolve by name. Attach reuses
Sprint 3a's `probe.Detect` to auto-classify `chain_type` / `chain_id`.

## 2. Non-Goals (Explicitly Deferred)

- **RemoteDriver (go-ethereum/ethclient)** — deferred to Sprint 3b.2 when a
  first remote-node operation (e.g., `node.block_number`, `node.rpc`) lands.
  Attach does not need a persistent RPC client; one probe call suffices.
- **Authentication (API key / JWT)** — deferred to 3b.2.
- **Multi-node remote discovery** — attach records exactly one node with the
  supplied URL; 3b.2 may grow validator-set enumeration.
- **`resolveNodeID` M4 refactor** — still deferred. M4 absorbs naturally when
  remote *node* commands land in 3b.2 (not in 3b).
- **Write operations against remotes** — needs signer (Sprint 4).

## 3. User-Facing Surface

### 3.1 `network.attach`

```json
{"command":"network.attach","args":{"rpc_url":"https://...","name":"sepolia","override":null}}
```

**args:**
- `rpc_url` (string, required) — same vocabulary as `network.probe`.
- `name` (string, required) — network identifier. Must match `^[a-z0-9][a-z0-9_-]*$`
  (same pattern as `network.json` schema). Must NOT equal `"local"` (reserved).
- `override` (string, optional) — chain_type override passed to probe.

**Result data on success:**
```json
{
  "name": "sepolia",
  "chain_type": "ethereum",
  "chain_id": 11155111,
  "rpc_url": "https://...",
  "nodes": [...],
  "created": true
}
```

- `created: true` on new save, `false` on overwrite.

**Errors:**
- `INVALID_ARGS` — missing rpc_url/name, name == "local", name pattern mismatch,
  probe-level input error (via `probe.IsInputError`).
- `UPSTREAM_ERROR` — probe endpoint failure, state file write failure.

### 3.2 `network.load` — behavior change

Existing: only accepts `name: "local"`.
Post-3b: any name resolves:
- `"local"` → `pids.json` + `current-profile.yaml` (unchanged).
- other → `state/networks/<name>.json` (written by attach).

Error for unknown name: `UPSTREAM_ERROR` wrapping `ErrStateNotFound` (matches
existing LoadActive semantics).

## 4. State Model

New directory at `<state-dir>/networks/` contains one JSON file per attached
remote network, named `<network.name>.json`. File content is the serialized
`types.Network` (matches `network.json` JSON schema).

Example:
```
state/
├── pids.json                       # local network state (unchanged)
├── current-profile.yaml            # local profile (unchanged)
└── networks/
    ├── sepolia.json                # remote attached "sepolia"
    └── mainnet-infura.json         # remote attached "mainnet-infura"
```

Writes are atomic (write to `<name>.json.tmp`, fsync, rename).

## 5. Detection & Construction

Attach flow:

1. Validate args at handler boundary (name pattern, name != "local", rpc_url present).
2. Call `probe.Detect({RPCURL, Override})` with default timeout.
3. Construct `types.Network` with:
   - `Name` = args.name
   - `ChainType` = probe result ChainType
   - `ChainId` = probe result ChainID (cast int64 → int per existing precedent)
   - `Nodes` = `[{ Id: "node1", Provider: "remote", Http: args.rpc_url }]`
4. Persist via `state.SaveRemote(stateDir, network)`.
5. Return `created` boolean based on whether file existed before.

## 6. Package Layout

- `network/internal/state/remote.go` — `SaveRemote`, `LoadRemote` (package),
  path helpers.
- `network/internal/state/network.go` — `LoadActive` extended to route by name.
- `network/cmd/chainbench-net/handlers.go` — `newHandleNetworkAttach`.
- `network/schema/command.json` — already includes `network.attach`? Must check
  and add if absent.

## 7. Schema & Wire

Check `command.json` for `"network.attach"` enum membership. If absent, add it.
`event.json` Result is untyped per-command; no schema change needed.

## 8. Error & Logging

- Log attach at INFO with `rpc_url`, resolved `chain_type`, `chain_id`, `name`.
  No body/headers.
- Refuse names containing `/`, `..`, or absolute-path chars (defense against
  path traversal in state dir). The `^[a-z0-9][a-z0-9_-]*$` pattern already
  blocks these but enforce at handler too for defense-in-depth.

## 9. Testing Strategy

**Unit (state/remote_test.go):**
- SaveRemote → file exists under networks/<name>.json, content roundtrips.
- LoadRemote → reads back.
- LoadActive with non-local name → routes to LoadRemote.
- LoadActive with unknown name → ErrStateNotFound.
- Atomic write: partial failure leaves no orphan file.

**Unit (handlers_test.go):**
- Attach happy path (stablenet mock) → state file created + result matches.
- Attach rejects name == "local" → INVALID_ARGS.
- Attach rejects name pattern mismatch → INVALID_ARGS.
- Attach with probe input error → INVALID_ARGS.
- Attach with upstream error → UPSTREAM_ERROR.
- Attach twice same name → `created: false` on second.

**Go E2E (e2e_test.go):**
- attach via root cmd, then network.load the same name, assert equality.

**Bash (tests/unit/tests/network-attach.sh):**
- attach via `cb_net_call`, jq-assert persisted fields.

## 10. Non-Functional

- Attach is idempotent wrt network name (overwrite allowed).
- State file writes bounded: one call = one file, no temp leaks on error.
- No global lock — single-process assumption matches existing pids.json handling.

## 11. Out-of-Scope Reminders (session-local, not memory)

Still deferred (unchanged from 3a spec §13):
- 2b.3 M3 — APIError.Details structured fields.
- 2c M3 — jq 3→1 call consolidation in `_cb_net_parse_result`.
- 2c M4 — jq version gate.

New deferrals (introduced by 3b scope boundaries):
- Sprint 3b.2 — RemoteDriver (ethclient), auth, first remote-node command
  (`node.rpc` or `node.block_number`). M4 `resolveNodeID` parameterization
  absorbed here.
- Sprint 3a minor — `isKnownOverride` duplicates signatures chain-type enum.
  Left untouched; revisit when 5th chain type added.
