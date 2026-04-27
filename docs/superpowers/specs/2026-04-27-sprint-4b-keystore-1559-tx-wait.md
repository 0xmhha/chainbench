# Sprint 4b — Keystore + EIP-1559 + tx.wait Design Spec

> 2026-04-27 · VISION §6 Sprint 4b (Sprint 4 follow-on)
> Scope: extend the Sprint 4 signer + tx.send surface with (1) a keystore-backed
> provider, (2) EIP-1559 dynamic-fee support in `node.tx_send`, and (3) a new
> `node.tx_wait` command for receipt polling. Builds the next layer of the
> evaluation harness tx-axis (per `docs/EVALUATION_CAPABILITY.md` §2).

## 1. Goal

Round out the legacy/env-only Sprint 4 baseline with the three pieces that an
evaluation-harness caller realistically needs to drive an end-to-end value
transfer + verify:

- **Keystore signer** — operators with existing keystore JSON files don't have
  to extract raw hex into env. Loaded transparently via the same `Signer`
  interface; handlers don't change.
- **EIP-1559 fees** — chains that have moved past legacy gas pricing (Sepolia,
  modern eth-mainnet forks). Opt-in: legacy stays default for backward
  compatibility.
- **`node.tx_wait`** — close the loop. Today callers `node.tx_send` and have to
  poll RPC themselves to confirm. evaluation harness needs a single command
  that returns receipt status.

## 2. Non-Goals (Deferred)

- **HSM / hardware wallet** — Sprint 5+ when remote-signer scenarios surface.
- **Fee Delegation (0x16) / EIP-7702 (0x4)** — Sprint 4c (chain-specific tx
  types per `docs/EVALUATION_CAPABILITY.md` §2).
- **Contract deploy / call / event decode** — Sprint 4d.
- **Subscription-based confirmation** — `node.tx_wait` polls; WS subscription
  is `subscription.open` territory (Sprint 5+).
- **Multi-tx batch / atomicity** — single tx per command.
- **Receipt detail beyond status + key fields** — see §4.3 surface; full tx
  trace / debug_traceTransaction is out of scope.

## 3. Security Contract (carry-over + additions)

1. All Sprint 4 invariants hold (see `docs/SECURITY_KEY_HANDLING.md`).
2. **Keystore decryption happens inside `network/internal/signer` only.**
   Decrypted key bytes never leave the package; the resulting `*sealed` is
   indistinguishable from an env-loaded one (same redaction surface).
3. **Keystore password env var is read via `os.Getenv` exactly once per Load
   call.** Not cached, not echoed.
4. Error messages for keystore failure reference the env var names and alias
   only — never the keystore path content, never the password, never the
   decrypted key.
5. The bash security-key-boundary test gains a keystore variant: spawn the
   binary with `CHAINBENCH_SIGNER_ALICE_KEYSTORE` + `_KEYSTORE_PASSWORD`,
   `node.tx_send`, then grep stdout/stderr/log for the underlying raw hex.

## 4. User-Facing Surface

### 4.1 Keystore provider

**Env vars (per alias, mutually exclusive with the raw-key form):**

```
CHAINBENCH_SIGNER_<ALIAS>_KEYSTORE=/path/to/keystore.json
CHAINBENCH_SIGNER_<ALIAS>_KEYSTORE_PASSWORD=<password>
```

Resolution order in `signer.Load`:

1. If `CHAINBENCH_SIGNER_<ALIAS>_KEY` is set → use raw hex (Sprint 4 path).
2. Else if `CHAINBENCH_SIGNER_<ALIAS>_KEYSTORE` is set:
   - Read the file (errors → `ErrInvalidKey` with env var name only).
   - Read `_KEYSTORE_PASSWORD` (empty / unset → `ErrInvalidKey`).
   - `keystore.DecryptKey(json, password)` → `*ecdsa.PrivateKey`.
   - Wrong password / corrupt file → `ErrInvalidKey` (no leak).
3. Else → `ErrUnknownAlias` (same as Sprint 4).

The `Signer` interface stays unchanged. Handlers using `signer.Load(alias)`
work for either provider transparently.

### 4.2 EIP-1559 in `node.tx_send`

Two new optional args:

```json
{
  "command": "node.tx_send",
  "args": {
    "network": "sepolia",
    "node_id": "node1",
    "signer": "alice",
    "to": "0x...",
    "value": "0x0",
    "max_fee_per_gas":          "0x59682f00",
    "max_priority_fee_per_gas": "0x3b9aca00",
    "gas":   21000,
    "nonce": 42
  }
}
```

**Selection rules:**

| Provided fields | Tx type | Notes |
|---|---|---|
| `gas_price` only | Legacy (0x0) | Sprint 4 behavior unchanged |
| `max_fee_per_gas` AND `max_priority_fee_per_gas` | DynamicFee (0x2) | both required if either is set |
| `gas_price` + any 1559 field | INVALID_ARGS | mixing rejected at boundary |
| Only one of the two 1559 fields | INVALID_ARGS | partial 1559 rejected |
| None of the three | Legacy auto-fill | unchanged from Sprint 4 |

**Auto-fill semantics for 1559**: in this sprint, if the user opts into 1559
they must provide BOTH fields. Inferring the partner from `eth_feeHistory` or
`eth_maxPriorityFeePerGas` is a Sprint 4b-followup so the failure surface
stays narrow.

**Signer side**: `types.LatestSignerForChainID` already supports DynamicFeeTx;
no signer change needed.

### 4.3 `node.tx_wait`

```json
{
  "command": "node.tx_wait",
  "args": {
    "network":    "sepolia",
    "node_id":    "node1",
    "tx_hash":    "0x...",
    "timeout_ms": 60000
  }
}
```

**args:**
- `network`, `node_id` — same resolution as other node.* commands
- `tx_hash` (required, 0x-prefixed 32-byte hex) — what to wait for
- `timeout_ms` (optional, 1000..600000, default 60000) — overall deadline

**Result on success / known failure (receipt observed):**

```json
{
  "status":              "success" | "failed",
  "tx_hash":             "0x...",
  "block_number":        12345,
  "block_hash":          "0x...",
  "gas_used":            21000,
  "effective_gas_price": "0x3b9aca00",
  "contract_address":    "0x..."  | "",
  "logs_count":          0
}
```

`status: "success"` ↔ receipt status `0x1`; `"failed"` ↔ `0x0`.
`contract_address` is empty string when the tx is not a contract-creation.
`logs_count` is `len(receipt.Logs)` to keep the result small; full log list
belongs in a future `node.events_get`.

**Result on timeout (no receipt observed):**

```json
{
  "status":  "pending",
  "tx_hash": "0x..."
}
```

`pending` is a normal terminal — the caller decides whether to retry with a
larger `timeout_ms` or treat the tx as stuck.

**Error mapping:**
- INVALID_ARGS — missing tx_hash, malformed tx_hash, timeout out of range
- UPSTREAM_ERROR — dial failure, RPC failure other than `not found`
  (`ethereum.NotFound` is treated as a normal "still pending" tick during
  polling, NOT an error)
- NOT_SUPPORTED — n/a (works on every EVM endpoint)

**Polling strategy:** exponential backoff starting at 200ms, doubling up to a
2s cap, until either the receipt is found, the deadline elapses, or the
context is cancelled. This is hidden inside the handler; the caller sees a
single response.

## 5. Package Layout

**Create:** none (no new packages — keystore lives inside `signer`).

**Modify:**
- `network/internal/signer/signer.go` — extend `Load` with keystore branch.
- `network/internal/signer/signer_test.go` — keystore happy / wrong-password /
  bad-file / boundary-leak tests.
- `network/internal/drivers/remote/client.go` — add `TransactionReceipt`.
- `network/internal/drivers/remote/client_test.go` — happy + not-found tests.
- `network/cmd/chainbench-net/handlers_node_tx.go` — 1559 selection + handler
  for `node.tx_wait`.
- `network/cmd/chainbench-net/handlers.go` — register `node.tx_wait`.
- `network/cmd/chainbench-net/handlers_test.go` — unit tests for both.
- `network/cmd/chainbench-net/e2e_test.go` — E2E tx_send (1559) + tx_wait paths.
- `network/schema/command.json` — add `"node.tx_wait"`.
- `network/internal/types/command_gen.go` — regenerated.
- `tests/unit/tests/security-key-boundary.sh` — extend to keystore variant.
- `tests/unit/tests/node-tx-wait.sh` — new bash test for the wait flow.
- `docs/SECURITY_KEY_HANDLING.md` — keystore section moves from "planned" to
  "current"; threat-model section adds keystore specifics.
- `docs/VISION_AND_ROADMAP.md` §6 — mark Sprint 4b complete.
- `docs/EVALUATION_CAPABILITY.md` — bump `🚧 Sprint 4b` cells to `✅`.
- `docs/NEXT_WORK.md` §3 P2 / §3.4 — drop the resolved follow-ups.

## 6. signer extension shape

```go
// internal/signer/signer.go

// envKeystorePath / envKeystorePassword names follow the existing
// CHAINBENCH_SIGNER_<ALIAS>_<SUFFIX> convention so deployment tooling can
// continue to use a single naming scheme.
func envKeystorePathName(alias string) string     { return "CHAINBENCH_SIGNER_" + strings.ToUpper(alias) + "_KEYSTORE" }
func envKeystorePasswordName(alias string) string { return envKeystorePathName(alias) + "_PASSWORD" }

// Load (extended):
//   1. raw KEY env (Sprint 4 path) — wins if set
//   2. KEYSTORE + KEYSTORE_PASSWORD env (Sprint 4b)
//   3. ErrUnknownAlias

// loadFromKeystore is unexported; invoked from Load when raw KEY is unset
// and KEYSTORE is set. Path / password validation errors all map to
// ErrInvalidKey with no leakage of file content or password.
func loadFromKeystore(alias Alias, path, password string) (*sealed, error)
```

`go-ethereum/accounts/keystore` provides `DecryptKey(json []byte, password string) (*Key, error)` whose `*Key.PrivateKey` is the `*ecdsa.PrivateKey` we wrap into `sealed`.

## 7. node.tx_wait handler shape

```go
// internal/cmd/chainbench-net/handlers_node_tx.go

const (
    minTxWaitMs     = 1000
    defaultTxWaitMs = 60000
    maxTxWaitMs     = 600000
)

func newHandleNodeTxWait(stateDir string) Handler {
    return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
        // parse + validate {network, node_id, tx_hash, timeout_ms?}
        // dial node, poll TransactionReceipt with exponential backoff
        // return {status, tx_hash, block_number?, block_hash?, gas_used?,
        //         effective_gas_price?, contract_address?, logs_count?}
    }
}
```

The handler MUST treat `ethereum.NotFound` as "keep polling", not as an
upstream error. Other RPC errors abort with UPSTREAM_ERROR.

## 8. Tests

### 8.1 signer keystore tests

`network/internal/signer/signer_test.go` — append:

- `TestLoad_Keystore_Happy` — create a keystore file via `keystore.EncryptKey`,
  set env, load, assert address matches.
- `TestLoad_Keystore_WrongPassword` — `_KEYSTORE_PASSWORD` mismatched →
  `ErrInvalidKey`. Error message must NOT contain the password.
- `TestLoad_Keystore_MissingPasswordEnv` — path set but no password env →
  `ErrInvalidKey`.
- `TestLoad_Keystore_FileNotFound` — invalid path → `ErrInvalidKey`. Error
  must NOT echo the path's content (trivially true since file doesn't exist;
  assertion guards against future regressions).
- `TestLoad_Keystore_RawKeyWins` — both `_KEY` and `_KEYSTORE` set → raw KEY
  path is taken. Address derived from KEY, not from the keystore file.
- `TestLoad_Keystore_RedactionBoundary` — load via keystore, format with
  `%v / %+v / %#v / %s` and slog → no key hex appears.

### 8.2 EIP-1559 handler tests

`handlers_test.go` — append:

- `TestHandleNodeTxSend_DynamicFee_Happy` — both 1559 fields → mock receives
  type-2 raw tx (assert via decoding the broadcast bytes back).
- `TestHandleNodeTxSend_MixedFeeFields` — `gas_price` + `max_fee_per_gas` →
  INVALID_ARGS.
- `TestHandleNodeTxSend_PartialDynamicFee` — only `max_fee_per_gas` → INVALID_ARGS.
- Existing `TestHandleNodeTxSend_Happy` keeps working as the legacy path.

### 8.3 node.tx_wait handler + E2E tests

- `TestHandleNodeTxWait_SuccessImmediate` — receipt available on first poll →
  `status: "success"` with all fields populated.
- `TestHandleNodeTxWait_FailedReceipt` — receipt with status `0x0` →
  `status: "failed"`, fields populated.
- `TestHandleNodeTxWait_NotFoundThenSuccess` — first poll returns NotFound,
  second returns receipt. Verifies the polling loop.
- `TestHandleNodeTxWait_Timeout` — mock always NotFound, timeout 1500ms →
  `status: "pending"`.
- `TestHandleNodeTxWait_BadHash` — malformed tx_hash → INVALID_ARGS.
- `TestHandleNodeTxWait_TimeoutOutOfRange` — `timeout_ms: 50` → INVALID_ARGS.
- `TestHandleNodeTxWait_UpstreamError` — RPC returns 500 → UPSTREAM_ERROR.

### 8.4 Bash boundary + flow tests

- `tests/unit/tests/security-key-boundary.sh` — keystore variant: build a
  keystore file in a temp dir, set `CHAINBENCH_SIGNER_ALICE_KEYSTORE` +
  `_KEYSTORE_PASSWORD`, run `node.tx_send`, grep for both the password and
  the underlying raw hex (must be neither in stdout/stderr/log).
- `tests/unit/tests/node-tx-wait.sh` — new test: attach a mock that returns a
  successful receipt on first try, run `node.tx_wait`, assert
  `status: "success"` and tx_hash echoes.

## 9. Schema

- Add `"node.tx_wait"` to `command.json` enum.
- Regenerate `command_gen.go` via `go generate ./...`.

## 10. Documentation

`docs/SECURITY_KEY_HANDLING.md`:
- Move "Key Injection (planned — Sprint 4b)" into "Key Injection (current —
  env-key OR keystore)".
- Document resolution order (raw KEY > keystore).
- Add operator note: keystore file permissions should be 0600; chainbench-net
  does NOT enforce this (best-effort warning would surface in logs only).

`docs/EVALUATION_CAPABILITY.md`:
- §2 Tx matrix: `EIP-1559 (0x2)` → ✅ in Go column / ❌ → 🚧 in MCP column.
  `Receipt polling (status+logs)` → ✅ in Go column.
- §5 Sprint roadmap: tick Sprint 4b row.

`docs/VISION_AND_ROADMAP.md` §6:
- Tick the Sprint 4b checkbox.

`docs/NEXT_WORK.md`:
- §3 P2 — Sprint 4b → ✅ (move ahead in timeline).
- §3.4 — drop any rows that are now closed by this sprint.

## 11. Error Classification

| Path | Code |
|---|---|
| Malformed JSON args | INVALID_ARGS |
| Missing / malformed tx_hash | INVALID_ARGS |
| Mixed legacy + 1559 fields | INVALID_ARGS |
| Partial 1559 (only one field) | INVALID_ARGS |
| timeout_ms out of [1000, 600000] | INVALID_ARGS |
| Keystore file unreadable | UPSTREAM_ERROR (config) — wraps ErrInvalidKey for handler classification |
| Keystore decrypt failure | UPSTREAM_ERROR (config) — wraps ErrInvalidKey |
| RPC failure during poll (not NotFound) | UPSTREAM_ERROR |
| Mock returns success → all fields populated | OK |

The signer-load classification keeps the Sprint 4 convention:
`ErrInvalidAlias` / `ErrUnknownAlias` → INVALID_ARGS, `ErrInvalidKey` →
UPSTREAM_ERROR.

## 12. Out-of-Scope Reminders

**Deferred from this sprint to 4c+ / 5+:**
- Fee Delegation (0x16) signing — Sprint 4c
- EIP-7702 SetCode signing — Sprint 4c
- Contract deploy / call / event decode — Sprint 4d
- WS subscription-based confirmation — Sprint 5+
- HSM / multi-sig — Sprint 5+
- `eth_feeHistory` based 1559 auto-fill — Sprint 4b follow-up if needed
- Receipt log decoding — Sprint 4d (`node.events_get`)

**Still deferred from earlier sprints (NEXT_WORK §3.4):**
- 2b.3 / 2c follow-ups, 3a SSoT redup, 3b TOCTOU, 3b.2c minors, 3c minors,
  Phase A/B/C holdovers
