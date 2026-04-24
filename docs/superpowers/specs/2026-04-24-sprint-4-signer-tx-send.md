# Sprint 4 — Signer Boundary + tx.send Design Spec

> 2026-04-24 · VISION §6 Sprint 4 — 서명 경계 + 보안 검증 (S4/S5)
> Scope: env-based signer package + `node.tx_send` command. Keystore
> provider + EIP-1559 fields deferred to Sprint 4b.

## 1. Goal

Give chainbench-net the ability to sign and broadcast transactions against
any attached network, using private keys sourced **only** from env vars at
process-spawn time. Key material must not appear in stdout, stderr, logs, or
state files under any code path.

## 2. Non-Goals (Deferred)

- **Keystore-based signer** — `CHAINBENCH_SIGNER_<ALIAS>_KEYSTORE` + password
  env. Sprint 4b. Env-only first is cleaner to validate end-to-end.
- **EIP-1559 dynamic fees** (`maxFeePerGas`, `maxPriorityFeePerGas`) — Sprint 4b.
  Sprint 4 uses legacy `eth_gasPrice`.
- **Tx confirmation waiting** — Sprint 4 broadcasts and returns tx hash; caller
  polls receipts. Sprint 4b may add `tx.wait`.
- **Multi-send / batching** — single tx per command.
- **Contract deployment typed commands** — deployments work via `to: ""` +
  `data: "0x..."` but no specialized command.
- **Byte-level `Sign([]byte) ([]byte, error)` API** from VISION §5.17.5 — the
  pragmatic `SignTx(ctx, *types.Transaction, *big.Int)` is what handlers
  actually need. Bytes-in/bytes-out can arrive later if a non-EVM chain is
  onboarded.

## 3. Security Contract (non-negotiable)

1. Private key material lives **only** in the signer struct's private field.
2. No getter exposes the key. `Address()` is the only public observation.
3. All code paths that could touch a signer instance redact via
   `slog.LogValuer` → `"***"`.
4. Error messages reference the **alias**, never the key value.
5. State files store only aliases; key material never persisted.
6. Injected via env var at process spawn; process termination = key destruction.

## 4. User-Facing Surface

### 4.1 `node.tx_send`

```json
{
  "command": "node.tx_send",
  "args": {
    "network":   "sepolia",
    "node_id":   "node1",
    "signer":    "alice",
    "to":        "0x...",
    "value":     "0x0",
    "data":      "0x",
    "gas":       21000,
    "gas_price": "0x3b9aca00",
    "nonce":     42
  }
}
```

**args:**
- `network`, `node_id` — resolve to the RPC endpoint (same as other node.*)
- `signer` (string, required) — alias looked up via `signer.Load`
- `to` (string, required; `""` for contract creation) — 0x hex address
- `value` (string, optional, default `"0x0"`) — hex wei
- `data` (string, optional, default `"0x"`) — hex bytes
- `gas` (integer, optional) — if omitted, `EstimateGas(msg)` used
- `gas_price` (string, optional) — if omitted, `SuggestGasPrice()` used
- `nonce` (integer, optional) — if omitted, `PendingNonceAt(from)` used

**Result:** `{tx_hash: "0x..."}`

**Errors:**
- INVALID_ARGS — missing signer/to, malformed hex, bad address
- UPSTREAM_ERROR — node unreachable, nonce/gas fetch failure, broadcast rejected
- INTERNAL — signer tx construction failure (unreachable under correct input)

### 4.2 `signer.Load` env format

```
CHAINBENCH_SIGNER_<ALIAS>_KEY=0x<64-hex-chars>
```

`<ALIAS>` is uppercased when scanning env; the handler's `signer` arg is
passed as-is and uppercased before env lookup. `signer: "alice"` →
`CHAINBENCH_SIGNER_ALICE_KEY`.

Signer is loaded on demand (per-tx-send call), not at process start. This
avoids persisting the Signer struct in a long-lived map and keeps the key
material's lifetime tight to the request.

## 5. Package Layout

```
network/internal/signer/
├── signer.go      # Alias, Signer interface, sealed struct, Load, LogValuer
└── signer_test.go # table-driven key-redaction tests + Sign happy path
```

**API:**

```go
package signer

type Alias string

type Signer interface {
    Address() common.Address
    SignTx(ctx context.Context, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error)
}

// Load resolves an alias to a Signer. Env-only in Sprint 4.
// Returns ErrUnknownAlias when no provider has the key.
func Load(alias Alias) (Signer, error)

// Error sentinels.
var (
    ErrUnknownAlias = errors.New("signer: unknown alias")
    ErrInvalidAlias = errors.New("signer: alias must be non-empty and [A-Za-z0-9_-]+")
    ErrInvalidKey   = errors.New("signer: key material is not a valid hex private key")
)
```

The sealed struct:

```go
type sealed struct {
    alias Alias
    addr  common.Address
    key   *ecdsa.PrivateKey // unexported, no accessor
}

func (*sealed) LogValue() slog.Value { return slog.StringValue("***") }
func (s *sealed) Address() common.Address { return s.addr }
func (s *sealed) SignTx(ctx, tx, chainID) (*types.Transaction, error) {
    return types.SignTx(tx, types.LatestSignerForChainID(chainID), s.key)
}
```

## 6. Handler Layout

`newHandleNodeTxSend(stateDir string) Handler`:

1. Parse args, validate structural inputs (alias + address + hex fields).
2. `resolveNode(stateDir, network, nodeID)` to get the node endpoint.
3. `signer.Load(Alias(args.signer))` — fail fast on unknown alias.
4. `dialNode(ctx, &node)` — reuses the 3b.2c helper for auth + dial.
5. Fetch `chainID` if not cached; fetch missing nonce/gas/gas_price via client.
6. Construct `*types.Transaction` with legacy fields (LegacyTx).
7. `s.SignTx(ctx, tx, chainID)` → signed tx.
8. `client.SendTransaction(ctx, signedTx)` (new remote.Client method).
9. Return `{tx_hash: signedTx.Hash().Hex()}`.

## 7. remote.Client Additions

```go
func (c *Client) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error)
func (c *Client) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error)
func (c *Client) SendTransaction(ctx context.Context, tx *types.Transaction) error
```

Thin wrappers over ethclient counterparts, same error-wrap convention.

## 8. Security Boundary Tests

### 8.1 Go unit (`signer_test.go`)

- `TestSigner_LogValueRedacts` — `slog.Info("signer", "s", s)` produces
  output containing `"***"` and nothing matching the raw key.
- `TestSigner_AddressIsStable` — same alias/key → same address.
- `TestSigner_SignTx_Roundtrip` — sign a known tx, recover sender matches address.
- `TestLoad_MissingEnvIsUnknownAlias` — no env var → `ErrUnknownAlias`.
- `TestLoad_BadKeyIsInvalidKey` — env has non-hex value → `ErrInvalidKey`.
- `TestLoad_BadAliasIsInvalidAlias` — alias with spaces/special chars → `ErrInvalidAlias`.
- **Critical redaction tests**:
  - error messages from `Load` / `SignTx` never contain the key hex.
  - `fmt.Sprintf("%+v", signer)` never contains the key hex.
  - `fmt.Sprintf("%#v", signer)` never contains the key hex (via the sealed
    struct hiding the field — go reflection on unexported fields is allowed,
    so the test asserts on the string output not on reflect).

### 8.2 Bash security boundary (`tests/unit/tests/security-key-boundary.sh`)

- Build chainbench-net binary
- Set `CHAINBENCH_SIGNER_ALICE_KEY=0x<known-hex>` as env
- Spawn binary with a tx.send command against a mock RPC (fake chain_id +
  accept-any signed tx)
- Capture stdout + stderr + log file
- Grep all three for:
  - The raw hex key
  - The hex key without 0x prefix
  - Any substring longer than 32 chars resembling hex
- All greps must return zero matches.

## 9. Schema

- `command.json` enum already lists `"tx.send"` — but this sprint introduces
  `"node.tx_send"` (node-scoped; matches the other `node.*` pattern).
  **Decision**: keep VISION's `tx.send` as a later cross-network command; for
  Sprint 4 we add `"node.tx_send"` to the enum. Regenerate `command_gen.go`.

## 10. Documentation

Create `docs/SECURITY_KEY_HANDLING.md`:

- Threat model (who can see keys, where they live)
- Env var convention + examples
- Sprint 4 boundaries (env only) + what's coming (keystore)
- Operator checklist: never commit env files, rotate on suspected exposure
- Developer contract: signer package is the ONLY place keys live

## 11. Error Classification

| Path | Code |
|---|---|
| Malformed JSON args | INVALID_ARGS |
| Missing signer / to | INVALID_ARGS |
| Invalid hex address / value / data | INVALID_ARGS |
| Unknown signer alias | INVALID_ARGS |
| Invalid key in env | UPSTREAM_ERROR (config failure; not caller's fault per request) |
| nonce / gas / gas_price fetch failure | UPSTREAM_ERROR |
| SendTransaction rejection | UPSTREAM_ERROR |
| SignTx internal error | INTERNAL |

## 12. Out-of-Scope Reminders (session-local)

**Deferred from this sprint to 4b+:**
- Keystore provider (`CHAINBENCH_SIGNER_<ALIAS>_KEYSTORE` + password)
- EIP-1559 dynamic fee fields
- Transaction receipt polling (`tx.wait`)
- Byte-level `Sign([]byte)` API

**Still deferred from earlier sprints:**
- 2b.3 M3, 2c M3/M4, 3a isKnownOverride SSoT, 3b created TOCTOU, 3b.2a I-2, 3b.2b Minor
- 3b.2c minors (tail_log M4, BadAddress variants, types.Auth typed refactor)
- 3c minors (cloneMap shallow, %v formatting, getInt string-numeric, fixture gaps)
