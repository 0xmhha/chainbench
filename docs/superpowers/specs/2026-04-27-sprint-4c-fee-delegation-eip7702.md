# Sprint 4c — Fee Delegation (0x16) + EIP-7702 SetCode (0x4) Design Spec

> 2026-04-27 · VISION §6 Sprint 4c (Sprint 4 series continuation)
> Scope: extend the Sprint 4b tx surface with (1) a generic `SignHash` on the
> Signer interface so chain-specific tx types can compose multiple signatures
> without breaking the sealed boundary, (2) **EIP-7702 SetCodeTx (0x4)** as a
> new branch in `node.tx_send`, (3) **Fee Delegation (0x16)** — a go-stablenet-
> specific tx type — as a new command `node.tx_fee_delegation_send` with
> chain-type allowlist. Builds the chain-specific tx cells of the evaluation
> matrix (`docs/EVALUATION_CAPABILITY.md` §2 rows 3-4).

## 1. Goal

After Sprint 4b the Go `network/` covered legacy + EIP-1559 + receipt polling.
Sprint 4c closes the two remaining tx-type cells in the evaluation matrix:

- **EIP-7702 SetCodeTx** — standard EIP supported by go-ethereum natively
  (`types.SetCodeTx`, `types.SignSetCode`). Authorization-list semantics let
  one tx temporarily install code at multiple signer addresses. Non-go-stablenet
  chains support this too.
- **Fee Delegation (0x16)** — go-stablenet's own tx type that lets a fee payer
  cover gas for a sender. RLP envelope = sender's standard DynamicFeeTx fields
  with sender V/R/S, plus fee_payer address + fee_payer V/R/S. Two distinct
  signers required.

Both sit on top of dynamic-fee semantics. Both need hash-level signing — for
SetCodeTx authorizations and for the fee-payer outer hash. That is the gap
left open at the end of Sprint 4: `Signer.SignHash([]byte)` was deliberately
deferred but is the cleanest way to support both tx types without leaking
private key material across the package boundary.

## 2. Non-Goals (Deferred)

- **Adapter `SupportedTxTypes()` interface method** — Sprint 4c uses a
  hardcoded chain-type allowlist for 0x16 in the handler (`stablenet`, `wbft`).
  Promoting to an adapter contract is a Sprint 5 / 4d concern when more
  chain-specific divergences accumulate.
- **Fee Delegation receipts as a distinct status** — `node.tx_wait` already
  handles 0x16 broadcasts via standard `eth_getTransactionReceipt` (the
  receipt format is identical to DynamicFeeTx). No new wait command.
- **Auto-fill of EIP-7702 authorization nonces** — caller supplies the
  authorization list with explicit `nonce` per entry. Querying
  `eth_getTransactionCount` per authorizer would multiply RPC round-trips
  and obscure the test author's intent.
- **Multi-signer authorization-list signing in one call** — each authorization
  has its own `signer` alias; the handler resolves and signs each. This is
  in scope. What's NOT in scope: deferred / async authorization signing.
- **Tx replay / tampering helpers** — bash `fee_delegate.py` has a
  `--tamper sender|feepayer` mode for negative tests. The Go path does NOT
  expose this; negative paths are tested via INVALID_ARGS guards (bad alias,
  malformed addresses, mixed legacy/1559).
- **Contract deploy / call / event decode** — Sprint 4d.
- **MCP exposure** — Sprint 5.

## 3. Security Contract (carry-over + additions)

1. All Sprint 4 / 4b invariants hold (see `docs/SECURITY_KEY_HANDLING.md`).
2. **`SignHash` returns the 65-byte ECDSA signature only** — it does NOT
   expose the private key, and the underlying `*ecdsa.PrivateKey` never
   leaves the `signer` package. The signature shape (R || S || V) is the
   standard Ethereum form returned by `crypto.Sign`.
3. **Fee delegation signing uses two distinct signers**. The handler resolves
   `signer` (sender) and `fee_payer` (also a signer alias) via separate
   `signer.Load` calls. Either Load failure short-circuits with INVALID_ARGS
   (bad alias) or UPSTREAM_ERROR (config failure). No combined / shared
   signer surface — each retains its own redaction boundary.
4. **EIP-7702 authorization-list signing**: each authorization entry has a
   `signer` alias; the handler loads each and uses `SignHash` on the
   authorization's `SigHash()`. Authorizers are independent of the tx sender.
5. **No raw key material in error paths**. Both 0x16 and 0x4 RLP construction
   happens inside the handler, never serialized to logs / state / errors.
   Existing redaction tests (Sprint 4 + 4b) must keep passing untouched.

## 4. User-Facing Surface

### 4.1 `Signer.SignHash`

Interface extension in `network/internal/signer/signer.go`:

```go
type Signer interface {
    Address() common.Address
    SignTx(ctx context.Context, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error)
    SignHash(ctx context.Context, hash common.Hash) ([]byte, error)  // NEW
}
```

Implementation contract:
- Returns the 65-byte signature `crypto.Sign(hash[:], s.key)` produces.
- Errors wrapped via `fmt.Errorf("signer.SignHash(%s): %w", s.alias, err)` —
  alias only, never key bytes.
- ctx is reserved for future HSM / remote-signer paths (same convention as
  `SignTx` per Sprint 4 ctx-doc).

### 4.2 `node.tx_send` extended for EIP-7702

New optional arg `authorization_list` — array of authorization tuples:

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
    "gas":   100000,
    "nonce": 42,
    "authorization_list": [
      {
        "chain_id": "0x1",
        "address":  "0xabcd...",
        "nonce":    "0x0",
        "signer":   "bob"
      },
      ...
    ]
  }
}
```

**Selection rules (3-way, extends Sprint 4b's 2-way)**:

| Provided fields | Tx type | Notes |
|---|---|---|
| `gas_price` only | Legacy (0x0) | Sprint 4 unchanged |
| `max_fee_per_gas` + `max_priority_fee_per_gas`, no `authorization_list` | DynamicFee (0x2) | Sprint 4b unchanged |
| `authorization_list` non-empty (requires both 1559 fields) | SetCode (0x4) | NEW Sprint 4c |
| `authorization_list` + `gas_price` | INVALID_ARGS | mixed: 1559 fields required |
| `authorization_list` + partial 1559 | INVALID_ARGS | both 1559 fields required |
| `authorization_list` empty array `[]` | DynamicFee (0x2) | empty list is normal-1559, not SetCode |

**Authorization-list entry validation** (each entry, before any signer.Load):
- `chain_id` (string, required) — hex; 0 means valid on any chain
- `address` (string, required) — 0x-prefixed 20-byte hex (delegate target)
- `nonce` (string, required) — hex
- `signer` (string, required) — alias

Bad shape on any field → INVALID_ARGS.

**Signing flow**:
1. Resolve sender via `signer.Load(req.Signer)` (existing).
2. For each authorization entry:
   - Resolve authorizer via `signer.Load(auth.Signer)`.
   - Build `types.SetCodeAuthorization{ChainID, Address, Nonce}` (V/R/S zero).
   - Call authorizer's `SignHash(ctx, auth.SigHash())`.
   - Fill in V/R/S from the 65-byte sig.
3. Build `types.SetCodeTx` with all fields.
4. Sender's `SignTx(ctx, ethtypes.NewTx(setCodeTx), chainID)`.
5. Broadcast as usual.

**Result**: `{tx_hash: "0x..."}` (same as 1559).

### 4.3 `node.tx_fee_delegation_send`

NEW command. go-stablenet specific.

```json
{
  "command": "node.tx_fee_delegation_send",
  "args": {
    "network":    "stablenet-mainnet",
    "node_id":    "node1",
    "signer":     "alice",
    "fee_payer":  "fpayer-1",
    "to":         "0x...",
    "value":      "0x0",
    "data":       "0x",
    "max_fee_per_gas":          "0x59682f00",
    "max_priority_fee_per_gas": "0x3b9aca00",
    "gas":   21000,
    "nonce": 7
  }
}
```

**Required**: `signer`, `fee_payer`, `to`, both 1559 fields, `gas`, `nonce`,
`network`, `node_id`. The handler does NOT auto-fill nonce / gas / fees for
fee-delegation tx (caller's responsibility — chain-specific testing intent
demands explicit values).

**Chain-type allowlist**: handler checks the resolved network's `chain_type`
and returns NOT_SUPPORTED unless it's in `{"stablenet", "wbft"}`.

**Construction**:
1. Resolve sender + fee_payer via two separate `signer.Load`.
2. Sender signs the inner DynamicFeeTx (`signer.SignTx(ctx, dynTx, chainID)`)
   to get sender V/R/S.
3. Build the fee-payer outer hash:
   `keccak256(0x16 || rlp([sender_payload_with_sig, fee_payer_addr]))`.
4. Fee payer's `SignHash(ctx, outer_hash)` → 65-byte sig → fpV / fpR / fpS.
5. Final RLP: `0x16 || rlp([sender_payload_with_sig, fee_payer_addr, fpV, fpR, fpS])`.
6. Broadcast via `client.SendTransaction` — but `ethtypes.Transaction`
   doesn't natively understand 0x16 (it's go-stablenet specific). We bypass
   the typed Tx layer and use `client.SendRawTransaction` (new wrapper
   method on `remote.Client`) that takes raw bytes.

**Result**: `{tx_hash: "0x..."}` (computed locally as `keccak256(rawTxBytes)`).

**Error mapping**:
- INVALID_ARGS — malformed args, bad addresses, bad hex, missing required
  fields, signer alias errors (`ErrInvalidAlias`/`ErrUnknownAlias` for either signer or fee_payer)
- UPSTREAM_ERROR — dial failure, ChainID fetch failure, key load config error
  (`ErrInvalidKey`), broadcast failure
- NOT_SUPPORTED — `chain_type` not in `{"stablenet", "wbft"}`
- INTERNAL — RLP encoding failure (invariant breach)

## 5. Package Layout

**Modify:**
- `network/internal/signer/signer.go` — add `SignHash` to the interface and `*sealed` impl; redaction tests for SignHash error path.
- `network/internal/signer/signer_test.go` — `SignHash` happy + recoverable + redaction tests.
- `network/internal/drivers/remote/client.go` — add `SendRawTransaction(ctx, rawBytes []byte) error`.
- `network/internal/drivers/remote/client_test.go` — happy + reject tests.
- `network/cmd/chainbench-net/handlers_node_tx.go` — extend `newHandleNodeTxSend` with authorization_list path; add `newHandleNodeTxFeeDelegationSend`.
- `network/cmd/chainbench-net/handlers.go` — register `node.tx_fee_delegation_send`; comment update.
- `network/cmd/chainbench-net/handlers_test.go` — unit tests for both paths.
- `network/cmd/chainbench-net/e2e_test.go` — E2E for SetCode + Fee Delegation.
- `network/schema/command.json` — add `"node.tx_fee_delegation_send"`.
- `network/internal/types/command_gen.go` — regenerated.
- `tests/unit/tests/node-tx-set-code.sh` — new bash test for EIP-7702 happy path.
- `tests/unit/tests/node-tx-fee-delegation.sh` — new bash test for 0x16 happy path + chain-type NOT_SUPPORTED.
- `tests/unit/tests/security-key-boundary.sh` — extend with Scenario 4 (fee delegation: both sender + fee_payer keys must not leak).
- `docs/SECURITY_KEY_HANDLING.md` — note `SignHash` contract; redaction surface unchanged.
- `docs/EVALUATION_CAPABILITY.md` — flip Fee Delegation + EIP-7702 cells from `❌ Sprint 4c (가칭)` to `✅ Sprint 4c`.
- `docs/VISION_AND_ROADMAP.md` — tick Sprint 4c.
- `docs/NEXT_WORK.md` — drop the resolved 4c row from §3.

## 6. signer SignHash shape

```go
// internal/signer/signer.go

func (s *sealed) SignHash(_ context.Context, hash common.Hash) ([]byte, error) {
    sig, err := crypto.Sign(hash[:], s.key)
    if err != nil {
        return nil, fmt.Errorf("signer.SignHash(%s): %w", s.alias, err)
    }
    return sig, nil
}
```

`crypto.Sign` returns 65 bytes. Caller (handler) decomposes into V (last byte)
+ R (bytes 0..32) + S (bytes 32..64) for embedding into auth tuples or fee-
delegation outer payload.

## 7. node.tx_send authorization_list flow

Extends the existing 1559 selector. After fee-mode validation runs, check
`authorization_list` length:

```go
if len(req.AuthorizationList) > 0 {
    if !useDynamicFee {
        return nil, NewInvalidArgs("authorization_list requires both max_fee_per_gas and max_priority_fee_per_gas")
    }
    useSetCode = true
}
```

The 3-way branch then constructs `*ethtypes.Transaction`:
- legacy: existing
- dynamic-fee, no auth list: existing 4b path
- set-code: build authorizations (sign each via authorizer's `SignHash`), then
  `ethtypes.NewTx(&ethtypes.SetCodeTx{...})`

Sender then signs the wrapping tx via `signer.SignTx(ctx, unsigned, chainID)`.
Note: `types.LatestSignerForChainID(chainID)` already supports SetCodeTx.

## 8. node.tx_fee_delegation_send shape

```go
const (
    feeDelegateTxType = 0x16
)

func newHandleNodeTxFeeDelegationSend(stateDir string) Handler {
    return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
        // parse → validate → resolveNode → chain_type allowlist guard →
        // dialNode → sender sigs → fee_payer hash → SignHash → RLP →
        // SendRawTransaction
    }
}
```

`remote.Client.SendRawTransaction(ctx, raw []byte)` is needed because
`ethclient.SendTransaction(*types.Transaction)` only knows the standard tx
types. The wrapper uses `c.rpc.Client().CallContext(ctx, &result, "eth_sendRawTransaction", hexutil.Encode(raw))`.

## 9. Tests

### 9.1 signer SignHash

- `TestSigner_SignHash_Happy` — known synthetic key + known hash → 65-byte sig; recover address via `crypto.SigToPub` and assert it matches `s.Address()`.
- `TestSigner_SignHash_RedactionBoundary` — error path (induced via mock or
  `(*sealed)(nil).SignHash` panic recovery — actual: just verify error wrap
  references alias, NOT key bytes).
- `TestSigner_SignHash_KeystoreLoaded` — keystore-backed signer also signs
  hashes; assert recoverable + redaction.

### 9.2 EIP-7702 (`TestHandleNodeTxSend_SetCode_*`)

- `_Happy_SingleAuth` — one authorization entry; mock receives type-4 raw tx
  (assert via decoding the broadcast bytes back).
- `_Happy_MultiAuth` — two entries with different signers; both V/R/S non-zero.
- `_MissingAuthFields` — entry without `chain_id` / `address` / `nonce` /
  `signer` → INVALID_ARGS each (table-driven).
- `_AuthSignerUnknown` — entry with unknown alias → INVALID_ARGS.
- `_AuthorizationListWithLegacy` — `authorization_list` + `gas_price` →
  INVALID_ARGS.
- `_AuthorizationListWithoutTip` — `authorization_list` + only `max_fee_per_gas`
  → INVALID_ARGS.
- `_EmptyAuthorizationListIsDynamicFee` — `[]` → DynamicFee (type 2), not SetCode.

### 9.3 Fee Delegation (`TestHandleNodeTxFeeDelegationSend_*`)

- `_Happy_Stablenet` — chain_type "stablenet" + valid sender + fee_payer; mock
  intercepts `eth_sendRawTransaction`, asserts first byte == 0x16.
- `_ChainTypeNotSupported_Ethereum` — chain_type "ethereum" → NOT_SUPPORTED.
- `_ChainTypeNotSupported_Wemix` — chain_type "wemix" → NOT_SUPPORTED.
- `_MissingFeePayer` — args without `fee_payer` → INVALID_ARGS.
- `_FeePayerAliasUnknown` → INVALID_ARGS.
- `_BadToAddress` / `_BadValueHex` / `_MissingNonce` / `_MissingGas` →
  INVALID_ARGS each.
- `_MissingMaxFeeFields` — args without 1559 fields → INVALID_ARGS.
- `_DispatcherRegistration` — `allHandlers` includes `node.tx_fee_delegation_send`.

### 9.4 E2E + bash boundary

- Go E2E (`TestE2E_NodeTxSend_SetCode_AgainstAttachedRemote`): cobra in-process
  attach + tx_send with one authorization; assert mock saw type-4 raw tx.
- Go E2E (`TestE2E_NodeTxFeeDelegationSend_AgainstAttachedRemote`): mock with
  chain_type-revealing endpoints (e.g., `wbft_*` available) + 0x16 broadcast;
  assert mock saw type-0x16 raw tx.
- `tests/unit/tests/node-tx-set-code.sh` — Python mock + happy path.
- `tests/unit/tests/node-tx-fee-delegation.sh` — Python mock pretending to be
  stablenet (probe path); happy 0x16 broadcast + chain_type NOT_SUPPORTED.
- `tests/unit/tests/security-key-boundary.sh` Scenario 4 — fee delegation
  variant: spawn binary with both `_KEY` and a separate `_KEYSTORE` for fee
  payer; run tx_fee_delegation_send; grep stdout/stderr/log for sender hex AND
  fee-payer raw hex AND keystore password. None must surface.

## 10. Schema

Add `"node.tx_fee_delegation_send"` to `command.json` enum (after
`node.tx_send`, alphabetical with peers). Regenerate `command_gen.go`.

`node.tx_send` enum unchanged — `authorization_list` is an args extension.

## 11. Error Classification

| Path | Code |
|---|---|
| Malformed JSON args | INVALID_ARGS |
| Missing signer / fee_payer / to | INVALID_ARGS |
| Bad hex (to / value / data / fee fields / authorization fields) | INVALID_ARGS |
| Mixed fee modes (legacy + 1559 / 1559 + auth_list with gas_price / partial 1559 + auth_list) | INVALID_ARGS |
| Unknown signer alias (sender / fee_payer / authorizer) | INVALID_ARGS |
| Invalid signer key in env | UPSTREAM_ERROR |
| chain_type not in {stablenet, wbft} for fee delegation | NOT_SUPPORTED |
| Dial / chainID / nonce / gas fetch failure | UPSTREAM_ERROR |
| Authorizer SignHash failure | INTERNAL (invariant — inputs validated earlier) |
| RLP encoding failure (fee delegation) | INTERNAL |
| `eth_sendRawTransaction` rejection | UPSTREAM_ERROR |
| SignTx / SignHash signature failure | INTERNAL |

## 12. Out-of-Scope Reminders

**Deferred to 4d / 5+:**
- Fee delegation negative tests with tampered signatures (caller composes via
  `node.tx_send` with a manually-constructed raw tx; out-of-scope here)
- Adapter `SupportedTxTypes()` interface — ad-hoc allowlist for 4c
- Receipt log decoding / contract deploy / event_get — Sprint 4d
- MCP exposure of any new command — Sprint 5
- Fee delegation on remote networks where the operator can't supply the
  fee_payer key (HSM scenario) — Sprint 5+

**Still deferred from earlier sprints (NEXT_WORK §3.4):**
- 2b.3 / 2c follow-ups, 3a SSoT redup, 3b TOCTOU, 3b.2c minors, 3c minors,
  Phase A/B/C holdovers
