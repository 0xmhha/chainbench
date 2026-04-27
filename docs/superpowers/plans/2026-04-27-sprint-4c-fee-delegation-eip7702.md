# Sprint 4c — Fee Delegation + EIP-7702 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Add `Signer.SignHash`, EIP-7702 SetCodeTx via `node.tx_send`
authorization_list extension, and Fee Delegation 0x16 via the new
`node.tx_fee_delegation_send` command. Both depend on hash-level signing
that Sprint 4 deferred. Lock the redaction boundary across the new path.

**Architecture:** `Signer` interface gains `SignHash(ctx, hash) ([]byte, error)`
returning the standard 65-byte ECDSA signature. EIP-7702 signs each authorization
list entry via the authorizer's `SignHash`, then the sender signs the SetCodeTx
via `SignTx`. Fee delegation builds the inner DynamicFeeTx (sender-signed via
`SignTx`), constructs the keccak256 outer hash, then fee_payer signs via
`SignHash`. Final RLP is broadcast via a new `remote.Client.SendRawTransaction`
wrapper that bypasses the typed-tx layer for chain-specific bytes.

**Tech Stack:** Go 1.25, `go-ethereum/core/types` (SetCodeTx, SetCodeAuthorization,
SignSetCode), `go-ethereum/crypto` (Sign), `go-ethereum/rlp`, stdlib.

Spec: `docs/superpowers/specs/2026-04-27-sprint-4c-fee-delegation-eip7702.md`

---

## Commit Discipline

- English commit messages.
- NO `Co-Authored-By` trailer; NO "Generated with Claude Code"; NO emoji.
- Conventional Commits prefix: `feat(scope):` / `fix(scope):` / `test(scope):` / `refactor(scope):` / `docs:` / `style(scope):` / `chore(scope):`.
- Key material / passwords / signatures NEVER appear in test assertion strings, error messages, or log lines. Reviewers enforce this.

## File Structure

**Modify:**
- `network/internal/signer/signer.go` — add `SignHash` to interface + sealed impl
- `network/internal/signer/signer_test.go` — `SignHash` happy + recover + redaction tests
- `network/internal/drivers/remote/client.go` — add `SendRawTransaction(ctx, raw) error`
- `network/internal/drivers/remote/client_test.go` — `SendRawTransaction` happy + reject
- `network/cmd/chainbench-net/handlers_node_tx.go` — extend `newHandleNodeTxSend` for SetCode; add `newHandleNodeTxFeeDelegationSend`
- `network/cmd/chainbench-net/handlers.go` — register `node.tx_fee_delegation_send`; comment update
- `network/cmd/chainbench-net/handlers_test.go` — SetCode + FeeDelegation unit tests
- `network/cmd/chainbench-net/e2e_test.go` — Go E2E for both
- `network/schema/command.json` — add `"node.tx_fee_delegation_send"`
- `network/internal/types/command_gen.go` — regenerated
- `tests/unit/tests/security-key-boundary.sh` — Scenario 4 (fee delegation)
- `docs/SECURITY_KEY_HANDLING.md` — note `SignHash` contract addition
- `docs/EVALUATION_CAPABILITY.md` — flip 0x16 + 0x4 cells
- `docs/VISION_AND_ROADMAP.md` — tick Sprint 4c
- `docs/NEXT_WORK.md` — drop resolved P2.5 row

**Create:**
- `tests/unit/tests/node-tx-set-code.sh`
- `tests/unit/tests/node-tx-fee-delegation.sh`

---

## Task 1 — Signer.SignHash

**Files:**
- Modify: `network/internal/signer/signer.go`
- Modify: `network/internal/signer/signer_test.go`

- [ ] **Step 1: Write failing tests**

Append to `signer_test.go`:

```go
func TestSigner_SignHash_Happy(t *testing.T) {
    withSignerEnv(t, "alice", "0x"+keyHex64)
    s, err := signer.Load("alice")
    if err != nil { t.Fatal(err) }
    hash := common.HexToHash("0x" + strings.Repeat("a", 64))
    sig, err := s.SignHash(context.Background(), hash)
    if err != nil { t.Fatalf("SignHash: %v", err) }
    if len(sig) != 65 { t.Errorf("sig length = %d, want 65", len(sig)) }
    pub, err := crypto.SigToPub(hash[:], sig)
    if err != nil { t.Fatalf("SigToPub: %v", err) }
    if got := crypto.PubkeyToAddress(*pub); got != s.Address() {
        t.Errorf("recovered addr = %s, signer addr = %s", got.Hex(), s.Address().Hex())
    }
}

func TestSigner_SignHash_KeystoreLoaded(t *testing.T) {
    dir := t.TempDir()
    path, want := keystoreFixture(t, dir, "secret")
    t.Setenv("CHAINBENCH_SIGNER_BOB_KEYSTORE", path)
    t.Setenv("CHAINBENCH_SIGNER_BOB_KEYSTORE_PASSWORD", "secret")
    s, err := signer.Load("bob")
    if err != nil { t.Fatal(err) }
    hash := common.HexToHash("0x" + strings.Repeat("b", 64))
    sig, err := s.SignHash(context.Background(), hash)
    if err != nil { t.Fatal(err) }
    pub, _ := crypto.SigToPub(hash[:], sig)
    if got := crypto.PubkeyToAddress(*pub); got != want {
        t.Errorf("recovered = %s, want %s", got.Hex(), want.Hex())
    }
}

func TestSigner_SignHash_RedactionBoundary(t *testing.T) {
    withSignerEnv(t, "carol", "0x"+keyHex64)
    s, err := signer.Load("carol")
    if err != nil { t.Fatal(err) }
    // Force an error path via zero-length hash. crypto.Sign requires
    // exactly 32 bytes; passing a wrong-shape input should fail in a way
    // that DOES NOT echo key bytes.
    _, err = s.SignHash(context.Background(), common.Hash{})
    // Allowed: the implementation may treat zero-hash as valid and return
    // a signature. If it does, we instead probe via a different angle:
    // assert the signer's String/GoString still redact after a SignHash
    // call (state should not leak across).
    out := fmt.Sprintf("%v %+v %#v %s", s, s, s, s)
    if strings.Contains(out, keyHex64) {
        t.Errorf("formatter leaks key after SignHash: %q", out)
    }
    _ = err
}
```

Imports to add: `"context"`, `"strings"`, `common` (already there from fixtures), `crypto`.

- [ ] **Step 2: Verify RED**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/signer/... -count=1 -v -run SignHash
```

Expected: compile failure — `s.SignHash undefined`.

- [ ] **Step 3: Implement**

Update interface in `signer.go`:

```go
type Signer interface {
    Address() common.Address
    SignTx(ctx context.Context, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error)
    SignHash(ctx context.Context, hash common.Hash) ([]byte, error)
}
```

Add method on `*sealed`:

```go
// SignHash returns the standard 65-byte ECDSA signature (R || S || V) over
// the supplied 32-byte hash. Used by chain-specific tx types whose envelopes
// require multiple signatures composed from raw hashes — EIP-7702 authorization
// tuples, go-stablenet fee-delegation outer hash. Errors reference the alias
// only; the underlying private key is never embedded.
//
// ctx is unused today (CPU-bound secp256k1) — preserved for future HSM /
// remote-signer paths, mirroring SignTx.
func (s *sealed) SignHash(_ context.Context, hash common.Hash) ([]byte, error) {
    sig, err := crypto.Sign(hash[:], s.key)
    if err != nil {
        return nil, fmt.Errorf("signer.SignHash(%s): %w", s.alias, err)
    }
    return sig, nil
}
```

- [ ] **Step 4: GREEN + lint**

```bash
cd network && go test ./internal/signer/... -count=1 -v
cd network && go test ./... -count=1 -timeout=60s
go -C network vet ./...
gofmt -l network/
```

- [ ] **Step 5: Commit**

```bash
git -C /Users/wm-it-22-00661/Work/github/tools/chainbench add network/internal/signer/
git -C /Users/wm-it-22-00661/Work/github/tools/chainbench commit -m "feat(signer): add SignHash for chain-specific tx envelopes

The Signer interface gains SignHash(ctx, common.Hash) returning the
standard 65-byte ECDSA signature. Sprint 4 deliberately deferred this
in favor of the typed SignTx; Sprint 4c needs it for two cases:
EIP-7702 SetCodeTx authorization tuples and go-stablenet
FeeDelegateDynamicFeeTx (0x16) fee-payer outer hash. Both compose
multiple signatures into a single tx envelope and so cannot be
expressed via SignTx alone.

The sealed struct's redaction surface is unchanged — the new method
does not introduce any code path that could surface key bytes in
errors, logs, or returned values. Errors from crypto.Sign are wrapped
with the alias only, mirroring the existing SignTx convention.

Tests cover happy path with sender recovery via SigToPub for both raw
and keystore signers, plus a redaction probe across the new method."
```

---

## Task 2 — remote.Client.SendRawTransaction

**Files:**
- Modify: `network/internal/drivers/remote/client.go`
- Modify: `network/internal/drivers/remote/client_test.go`

- [ ] **Step 1: Write failing tests**

Append to `client_test.go`:

```go
func TestClient_SendRawTransaction_Happy(t *testing.T) {
    var sentParam string
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req struct {
            Method string
            ID     json.RawMessage
            Params []json.RawMessage
        }
        _ = json.NewDecoder(r.Body).Decode(&req)
        w.Header().Set("Content-Type", "application/json")
        if req.Method == "eth_sendRawTransaction" {
            if len(req.Params) > 0 {
                _ = json.Unmarshal(req.Params[0], &sentParam)
            }
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x%s"}`, req.ID, strings.Repeat("a", 64))
            return
        }
        fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
    }))
    defer srv.Close()
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    c, err := Dial(ctx, srv.URL)
    if err != nil { t.Fatal(err) }
    defer c.Close()
    raw := []byte{0x16, 0xc0}
    if err := c.SendRawTransaction(ctx, raw); err != nil {
        t.Fatalf("SendRawTransaction: %v", err)
    }
    if sentParam != "0x16c0" {
        t.Errorf("sent param = %q, want 0x16c0", sentParam)
    }
}

func TestClient_SendRawTransaction_Reject(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req struct{ ID json.RawMessage }
        _ = json.NewDecoder(r.Body).Decode(&req)
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32000,"message":"invalid tx"}}`, req.ID)
    }))
    defer srv.Close()
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    c, err := Dial(ctx, srv.URL)
    if err != nil { t.Fatal(err) }
    defer c.Close()
    err = c.SendRawTransaction(ctx, []byte{0x16, 0xc0})
    if err == nil { t.Fatal("expected error") }
    if !strings.Contains(err.Error(), "remote.SendRawTransaction") {
        t.Errorf("err missing wrap prefix: %v", err)
    }
}
```

- [ ] **Step 2: Implement**

Add to `client.go`:

```go
// SendRawTransaction broadcasts pre-encoded RLP bytes via eth_sendRawTransaction.
// Used by chain-specific tx types whose envelopes ethclient.SendTransaction does
// not understand (e.g. go-stablenet FeeDelegateDynamicFeeTx 0x16). The caller
// is responsible for constructing + signing the bytes; this wrapper only
// formats the hex payload and forwards.
func (c *Client) SendRawTransaction(ctx context.Context, raw []byte) error {
    var result string
    if err := c.rpc.Client().CallContext(ctx, &result, "eth_sendRawTransaction", hexutil.Encode(raw)); err != nil {
        return fmt.Errorf("remote.SendRawTransaction: %w", err)
    }
    _ = result
    return nil
}
```

Add `hexutil` import: `"github.com/ethereum/go-ethereum/common/hexutil"`.

Note: `c.rpc.Client()` exposes the underlying `rpc.Client`. If `c.rpc` is the
ethclient handle, double-check the access pattern in the existing client.go —
adjust if the field is named differently or the helper is exposed via a
different accessor.

- [ ] **Step 3: GREEN + commit**

```bash
cd network && go test ./internal/drivers/remote/... -count=1 -v -run SendRawTransaction
cd network && go test ./... -count=1 -timeout=60s
git add network/internal/drivers/remote/
git commit -m "feat(drivers/remote): add SendRawTransaction for chain-specific tx envelopes

ethclient.SendTransaction only understands the standard typed
transactions (legacy / 1559 / blob / setcode). go-stablenet's
FeeDelegateDynamicFeeTx (type 0x16) lives outside that set; broadcasting
it requires submitting raw RLP bytes via eth_sendRawTransaction.

The new wrapper takes []byte and forwards via the underlying rpc.Client,
hex-encoding once and surfacing endpoint rejections as
remote.SendRawTransaction-prefixed errors. Caller owns construction +
signature."
```

---

## Task 3 — node.tx_send authorization_list (EIP-7702)

**Files:**
- Modify: `network/cmd/chainbench-net/handlers_node_tx.go`
- Modify: `network/cmd/chainbench-net/handlers_test.go`

- [ ] **Step 1: Write failing tests**

Append to `handlers_test.go`:

```go
// decodeBroadcastTxType is reused from Sprint 4b (handlers_test.go).
// SetCode txs have first byte 0x04.

func TestHandleNodeTxSend_SetCode_Happy_SingleAuth(t *testing.T) {
    var sentRaw string
    srv := newSetCodeMock(t, &sentRaw)  // helper below
    defer srv.Close()
    stateDir := t.TempDir()
    saveRemoteFixture(t, stateDir, "tn", srv.URL)
    t.Setenv("CHAINBENCH_SIGNER_ALICE_KEY", "0x"+keyHexA)  // sender
    t.Setenv("CHAINBENCH_SIGNER_BOB_KEY",   "0x"+keyHexB)  // authorizer

    h := newHandleNodeTxSend(stateDir)
    args, _ := json.Marshal(map[string]any{
        "network": "tn", "node_id": "node1", "signer": "alice",
        "to":     "0x0000000000000000000000000000000000000002",
        "max_fee_per_gas":          "0x59682f00",
        "max_priority_fee_per_gas": "0x3b9aca00",
        "gas":   100000,
        "nonce": 0,
        "authorization_list": []map[string]any{
            {"chain_id": "0x1", "address": "0x000000000000000000000000000000000000beef", "nonce": "0x0", "signer": "bob"},
        },
    })
    bus, _ := newTestBus(t); defer bus.Close()
    if _, err := h(args, bus); err != nil { t.Fatalf("handler: %v", err) }
    if got := decodeBroadcastTxType(t, sentRaw); got != 4 {
        t.Errorf("tx type = %d, want 4 (SetCode)", got)
    }
}

func TestHandleNodeTxSend_SetCode_Happy_MultiAuth(t *testing.T) {
    /* same as above with two auth entries; assert tx type == 4 */
}

func TestHandleNodeTxSend_AuthorizationListWithLegacy(t *testing.T) {
    h := newHandleNodeTxSend(t.TempDir())
    args, _ := json.Marshal(map[string]any{
        "network": "tn", "node_id": "node1", "signer": "alice",
        "to":        "0x0000000000000000000000000000000000000002",
        "gas_price": "0x1",
        "authorization_list": []map[string]any{{"chain_id":"0x1","address":"0x..","nonce":"0x0","signer":"bob"}},
    })
    bus, _ := newTestBus(t); defer bus.Close()
    _, err := h(args, bus)
    var api *APIError
    if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
        t.Errorf("want INVALID_ARGS, got %v", err)
    }
}

func TestHandleNodeTxSend_AuthorizationListWithoutTip(t *testing.T) { /* INVALID_ARGS */ }
func TestHandleNodeTxSend_AuthorizationListEmptyIsDynamicFee(t *testing.T) { /* assert tx type == 2 */ }
func TestHandleNodeTxSend_SetCode_AuthSignerUnknown(t *testing.T) { /* INVALID_ARGS */ }
func TestHandleNodeTxSend_SetCode_BadAuthAddress(t *testing.T) { /* INVALID_ARGS */ }
func TestHandleNodeTxSend_SetCode_BadAuthChainID(t *testing.T) { /* INVALID_ARGS */ }
```

The helper `newSetCodeMock` should respond with `eth_chainId: 0x1` and capture
the `eth_sendRawTransaction` param into `*sentRaw`. Follow the existing
`TestHandleNodeTxSend_DynamicFee_Happy` mock shape.

For `keyHexA` and `keyHexB`, define new test-only synthetic keys in the test
file. Use distinct hex (e.g. `keyHex64` for A, a new `keyHex64B` for B) so
sender and authorizer have different addresses.

- [ ] **Step 2: Implement**

In `handlers_node_tx.go`, extend the request struct:

```go
type authEntry struct {
    ChainID string `json:"chain_id"`
    Address string `json:"address"`
    Nonce   string `json:"nonce"`
    Signer  string `json:"signer"`
}

var req struct {
    /* existing fields */
    AuthorizationList []authEntry `json:"authorization_list"`
}
```

After the existing fee-mode validation, add:

```go
useSetCode := false
if len(req.AuthorizationList) > 0 {
    if !useDynamicFee {
        return nil, NewInvalidArgs("authorization_list requires both max_fee_per_gas and max_priority_fee_per_gas")
    }
    useSetCode = true
}
```

Where the tx is constructed, branch:

```go
if useSetCode {
    auths, err := buildAuthorizations(ctx, req.AuthorizationList, chainID)
    if err != nil { return nil, err }
    unsigned = ethtypes.NewTx(&ethtypes.SetCodeTx{
        ChainID:   uint256.MustFromBig(chainID),
        Nonce:     nonce,
        GasTipCap: uint256.MustFromBig(maxPriorityFee),
        GasFeeCap: uint256.MustFromBig(maxFee),
        Gas:       gas,
        To:        to,
        Value:     uint256.MustFromBig(value),
        Data:      data,
        AuthList:  auths,
    })
} else if useDynamicFee {
    /* existing 1559 path */
} else {
    /* existing legacy path */
}
```

Add `buildAuthorizations` helper:

```go
func buildAuthorizations(ctx context.Context, entries []authEntry, txChainID *big.Int) ([]ethtypes.SetCodeAuthorization, error) {
    out := make([]ethtypes.SetCodeAuthorization, 0, len(entries))
    for i, e := range entries {
        // Validation
        if e.Signer == "" || e.Address == "" || e.ChainID == "" || e.Nonce == "" {
            return nil, NewInvalidArgs(fmt.Sprintf("args.authorization_list[%d]: chain_id/address/nonce/signer are all required", i))
        }
        if !common.IsHexAddress(e.Address) {
            return nil, NewInvalidArgs(fmt.Sprintf("args.authorization_list[%d].address invalid: %q", i, e.Address))
        }
        cid, ok := new(big.Int).SetString(strings.TrimPrefix(e.ChainID, "0x"), 16)
        if !ok {
            return nil, NewInvalidArgs(fmt.Sprintf("args.authorization_list[%d].chain_id invalid: %q", i, e.ChainID))
        }
        nonce, ok := new(big.Int).SetString(strings.TrimPrefix(e.Nonce, "0x"), 16)
        if !ok {
            return nil, NewInvalidArgs(fmt.Sprintf("args.authorization_list[%d].nonce invalid: %q", i, e.Nonce))
        }
        // Signer load
        s, serr := signer.Load(signer.Alias(e.Signer))
        if serr != nil {
            if errors.Is(serr, signer.ErrInvalidAlias) || errors.Is(serr, signer.ErrUnknownAlias) {
                return nil, NewInvalidArgs(fmt.Sprintf("args.authorization_list[%d].signer: %v", i, serr))
            }
            return nil, NewUpstream(fmt.Sprintf("authorization_list[%d] signer load", i), serr)
        }
        // Build + sign
        auth := ethtypes.SetCodeAuthorization{
            ChainID: *uint256.MustFromBig(cid),
            Address: common.HexToAddress(e.Address),
            Nonce:   nonce.Uint64(),
        }
        sig, sigErr := s.SignHash(ctx, auth.SigHash())
        if sigErr != nil {
            return nil, NewInternal(fmt.Sprintf("sign authorization[%d]", i), sigErr)
        }
        // Decompose 65-byte sig into V/R/S.
        if len(sig) != 65 { return nil, NewInternal("authorization signature wrong length", nil) }
        auth.R = *uint256.NewInt(0).SetBytes(sig[0:32])
        auth.S = *uint256.NewInt(0).SetBytes(sig[32:64])
        auth.V = sig[64]
        out = append(out, auth)
    }
    return out, nil
}
```

Imports to add: `"github.com/holiman/uint256"`. (go-ethereum's tx fields use uint256.)

- [ ] **Step 3: GREEN + commit**

```bash
cd network && go test ./cmd/chainbench-net/... -count=1 -v -run 'TestHandleNodeTxSend_SetCode|TestHandleNodeTxSend_AuthorizationList'
cd network && go test ./... -count=1 -timeout=60s
git add network/cmd/chainbench-net/handlers_node_tx.go network/cmd/chainbench-net/handlers_test.go
git commit -m "feat(network-net): EIP-7702 SetCodeTx via authorization_list

node.tx_send gains an optional authorization_list arg; presence
upgrades the fee-mode selector from 2-way (legacy/1559) to 3-way
(legacy/1559/SetCode). SetCode requires both 1559 fields — mixing
with gas_price or omitting the tip is rejected as INVALID_ARGS at
the boundary before any signer.Load.

Each authorization tuple resolves its own signer alias, signs the
EIP-7702 SigHash via the new Signer.SignHash, and the resulting
65-byte signature decomposes into (V, R, S). The sender then signs
the wrapping SetCodeTx via the existing SignTx path —
types.LatestSignerForChainID already handles SetCodeTx.

Empty authorization_list ([]) stays on the DynamicFee path so the
caller cannot accidentally upgrade to SetCode by sending an empty
list."
```

---

## Task 4 — Fee Delegation 0x16 (`node.tx_fee_delegation_send`)

**Files:**
- Modify: `network/schema/command.json` — add `"node.tx_fee_delegation_send"`
- Modify: `network/internal/types/command_gen.go` — regenerated
- Modify: `network/cmd/chainbench-net/handlers.go` — register
- Modify: `network/cmd/chainbench-net/handlers_node_tx.go` — add handler
- Modify: `network/cmd/chainbench-net/handlers_test.go` — unit tests

- [ ] **Step 1: schema + regen**

Add `"node.tx_fee_delegation_send"` to `command.json` enum (after
`node.tx_send`). Run `cd network && go generate ./...`.

- [ ] **Step 2: Write failing tests**

Append to `handlers_test.go`:

```go
const feeDelegateTypeByte = 0x16

func TestHandleNodeTxFeeDelegationSend_Happy_Stablenet(t *testing.T) {
    var sentRaw string
    srv := httptest.NewServer(http.HandlerFunc(func(w, r) {
        /* mock: eth_chainId 0x1, eth_sendRawTransaction captures sentRaw */
    }))
    defer srv.Close()
    stateDir := t.TempDir()
    // Save remote network with chain_type explicitly stablenet:
    saveRemoteFixtureWithChainType(t, stateDir, "stab", srv.URL, "stablenet")
    t.Setenv("CHAINBENCH_SIGNER_ALICE_KEY", "0x"+keyHexA)
    t.Setenv("CHAINBENCH_SIGNER_FPAYER_KEY", "0x"+keyHexB)

    h := newHandleNodeTxFeeDelegationSend(stateDir)
    args, _ := json.Marshal(map[string]any{
        "network":   "stab",
        "node_id":   "node1",
        "signer":    "alice",
        "fee_payer": "fpayer",
        "to":        "0x0000000000000000000000000000000000000002",
        "value":     "0x0",
        "max_fee_per_gas":          "0x59682f00",
        "max_priority_fee_per_gas": "0x3b9aca00",
        "gas":   21000,
        "nonce": 7,
    })
    bus, _ := newTestBus(t); defer bus.Close()
    data, err := h(args, bus)
    if err != nil { t.Fatalf("handler: %v", err) }
    if !strings.HasPrefix(sentRaw, "0x16") {
        t.Errorf("raw tx not 0x16-prefixed: %q", sentRaw)
    }
    if tx, _ := data["tx_hash"].(string); !strings.HasPrefix(tx, "0x") || len(tx) != 66 {
        t.Errorf("bad tx_hash: %v", data["tx_hash"])
    }
}

func TestHandleNodeTxFeeDelegationSend_ChainTypeNotSupported_Ethereum(t *testing.T) {
    /* chain_type "ethereum" → NOT_SUPPORTED */
}

func TestHandleNodeTxFeeDelegationSend_ChainTypeNotSupported_Wemix(t *testing.T) {
    /* chain_type "wemix" → NOT_SUPPORTED */
}

func TestHandleNodeTxFeeDelegationSend_MissingFeePayer(t *testing.T) { /* INVALID_ARGS */ }
func TestHandleNodeTxFeeDelegationSend_FeePayerAliasUnknown(t *testing.T) { /* INVALID_ARGS */ }
func TestHandleNodeTxFeeDelegationSend_BadToAddress(t *testing.T) { /* INVALID_ARGS */ }
func TestHandleNodeTxFeeDelegationSend_MissingMaxFeeFields(t *testing.T) { /* INVALID_ARGS */ }
func TestHandleNodeTxFeeDelegationSend_MissingNonce(t *testing.T) { /* INVALID_ARGS */ }
func TestHandleNodeTxFeeDelegationSend_MissingGas(t *testing.T) { /* INVALID_ARGS */ }
func TestAllHandlers_IncludesTxFeeDelegationSend(t *testing.T) {
    h := allHandlers("x", "y")
    if _, ok := h["node.tx_fee_delegation_send"]; !ok {
        t.Error("allHandlers missing node.tx_fee_delegation_send")
    }
}
```

You may need a new `saveRemoteFixtureWithChainType` helper in handlers_test.go
that wraps `saveRemoteFixture` and injects `chain_type` into the persisted JSON.
Look at how `state.SaveRemote` constructs the file and replicate the
chain_type assignment.

- [ ] **Step 3: Implement**

Append to `handlers_node_tx.go`:

```go
const feeDelegateTxType = 0x16

var feeDelegationAllowedChains = map[string]bool{
    "stablenet": true,
    "wbft":      true,
}

// newHandleNodeTxFeeDelegationSend implements the go-stablenet
// FeeDelegateDynamicFeeTx (type 0x16). Two signers are required: the sender
// signs a standard DynamicFeeTx-shaped payload via SignTx, then the
// fee_payer signs keccak256(0x16 || rlp([sender_payload_with_sig, fp_addr]))
// via SignHash. Final RLP is broadcast via SendRawTransaction since the
// type is not part of go-ethereum's typed tx set.
//
// Args: {network, node_id, signer, fee_payer, to, value?, data?,
//        max_fee_per_gas, max_priority_fee_per_gas, gas, nonce}
// Result: {tx_hash}
//
// Error mapping:
//   INVALID_ARGS   — missing args, malformed hex, unknown signer alias for
//                    sender or fee_payer
//   NOT_SUPPORTED  — chain_type not in {stablenet, wbft}
//   UPSTREAM_ERROR — config (signer.ErrInvalidKey), dial / chainID / broadcast
//   INTERNAL       — RLP encode / signature / SignTx failure
func newHandleNodeTxFeeDelegationSend(stateDir string) Handler {
    return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
        var req struct {
            Network              string  `json:"network"`
            NodeID               string  `json:"node_id"`
            Signer               string  `json:"signer"`
            FeePayer             string  `json:"fee_payer"`
            To                   string  `json:"to"`
            Value                string  `json:"value"`
            Data                 string  `json:"data"`
            Gas                  *uint64 `json:"gas"`
            MaxFeePerGas         string  `json:"max_fee_per_gas"`
            MaxPriorityFeePerGas string  `json:"max_priority_fee_per_gas"`
            Nonce                *uint64 `json:"nonce"`
        }
        // ... parse, validate (see spec §11 error matrix)
        // ... resolve network, check chain_type allowlist
        // ... dial, fetch chainID
        // ... build sender's DynamicFeeTx, call s.SignTx
        // ... build fee-payer outer hash:
        //     keccak256(0x16 || rlp([sender_inner_with_sig, fp_addr]))
        // ... fp.SignHash, decompose into V/R/S
        // ... build final RLP: 0x16 || rlp([sender_inner_with_sig, fp_addr, fpV, fpR, fpS])
        // ... client.SendRawTransaction(ctx, rawTx)
        // ... return {"tx_hash": "0x" + hex.EncodeToString(crypto.Keccak256(rawTx))}
    }
}
```

The RLP encoding mirrors the Python reference at `tests/regression/lib/fee_delegate.py`. Key reference snippet to follow:

```
inner_with_sig = [chainID, nonce, tipCap, feeCap, gas, to, value, data,
                  accessList=[], senderV, senderR, senderS]
fp_payload     = [inner_with_sig, fp_addr]
fp_sig_hash    = keccak256(0x16 || rlp.encode(fp_payload))
final_payload  = [inner_with_sig, fp_addr, fpV, fpR, fpS]
raw_tx         = 0x16 || rlp.encode(final_payload)
```

Use `github.com/ethereum/go-ethereum/rlp` for encoding, `crypto.Keccak256` for
hashing. Sender V is normalized to 0/1 (chainID-aware via SignTx), but for the
inner sig embedded in 0x16 we use the raw bytes of the signed sender tx.
**Easiest**: build a `*types.DynamicFeeTx`, use sender's SignTx → signed tx →
extract V/R/S via `signed.RawSignatureValues()`. Then assemble RLP manually.

Register in `allHandlers`:

```go
"node.tx_fee_delegation_send": newHandleNodeTxFeeDelegationSend(stateDir),
```

Update the file-layout comment in `handlers.go`.

- [ ] **Step 4: GREEN + commit**

```bash
cd network && go test ./cmd/chainbench-net/... -count=1 -v -run 'TxFeeDelegation|TestAllHandlers_IncludesTxFeeDelegationSend'
cd network && go test ./... -count=1 -timeout=60s
git add network/schema/command.json network/internal/types/command_gen.go \
        network/cmd/chainbench-net/handlers.go network/cmd/chainbench-net/handlers_node_tx.go \
        network/cmd/chainbench-net/handlers_test.go
git commit -m "feat(network-net): node.tx_fee_delegation_send (go-stablenet 0x16)

New command for go-stablenet's FeeDelegateDynamicFeeTx. The handler
loads two signers — sender + fee_payer — via the existing
signer.Load. Sender signs a standard DynamicFeeTx; fee_payer signs
keccak256(0x16 || rlp([sender_inner_with_sig, fp_addr])) via
SignHash. The final 0x16 envelope is broadcast through
remote.Client.SendRawTransaction since ethclient.SendTransaction does
not understand chain-specific typed envelopes.

Chain-type guard: handler returns NOT_SUPPORTED unless the resolved
network's chain_type is in {stablenet, wbft}. The allowlist is
hardcoded for now; promoting to an Adapter.SupportedTxTypes() method
is a Sprint 5 concern.

Per spec §4.3 / §11, all argument validation happens before any
signer load or RPC round-trip. The two-signer arrangement does not
weaken the redaction boundary — both keys live in their own *sealed
struct and never share state."
```

---

## Task 5 — bash + Go E2E + boundary tests + docs

**Files:**
- Modify: `tests/unit/tests/security-key-boundary.sh` — Scenario 4
- Create: `tests/unit/tests/node-tx-set-code.sh`
- Create: `tests/unit/tests/node-tx-fee-delegation.sh`
- Modify: `network/cmd/chainbench-net/e2e_test.go`
- Modify: `docs/SECURITY_KEY_HANDLING.md`
- Modify: `docs/EVALUATION_CAPABILITY.md`
- Modify: `docs/VISION_AND_ROADMAP.md`
- Modify: `docs/NEXT_WORK.md`

- [ ] **Step 1: Bash boundary Scenario 4**

Append to `security-key-boundary.sh` after Scenario 3:

```bash
describe "security: fee-delegation tx does not leak sender or fee_payer keys"

# Reuse the keystore generated by Scenario 3 for fee_payer (signer "bob"
# already loaded via _KEYSTORE).
# Use a third key for the sender (signer "carol", raw _KEY).
CAROL_KEY_HEX="$(python3 -c 'import os, secrets; print(secrets.token_hex(32))')"
export CHAINBENCH_SIGNER_CAROL_KEY="0x${CAROL_KEY_HEX}"

# attach a network with chain_type=stablenet (the mock advertises 0x1 chain_id).
attach_data="$(cb_net_call "network.attach" \
  "{\"rpc_url\":\"http://127.0.0.1:${PORT}\",\"name\":\"sec-fd\",\"override\":\"stablenet\"}" 2>/dev/null)"
assert_eq "$(jq -r .name <<<"$attach_data")" "sec-fd" "fd attach name"

FD_LOG="${TMPDIR_ROOT}/fd-tx.log"
FD_ERR="${TMPDIR_ROOT}/fd-tx.err"
export CHAINBENCH_NET_LOG="${FD_LOG}"

FD_ENV='{"command":"node.tx_fee_delegation_send","args":{"network":"sec-fd","node_id":"node1","signer":"carol","fee_payer":"bob","to":"0x0000000000000000000000000000000000000003","value":"0x0","max_fee_per_gas":"0x1","max_priority_fee_per_gas":"0x1","gas":21000,"nonce":0}}'
fd_rc=0
printf '%s\n' "${FD_ENV}" | "${BINARY}" run >"${TMPDIR_ROOT}/fd.out" 2>"${FD_ERR}" || fd_rc=$?
assert_eq "$fd_rc" "0" "fee_delegation binary exit code"

for label in stdout stderr log; do
  case "$label" in
    stdout) cf="${TMPDIR_ROOT}/fd.out" ;;
    stderr) cf="${FD_ERR}" ;;
    log)    cf="${FD_LOG}" ;;
  esac
  if grep -qi -- "${CAROL_KEY_HEX}" "${cf}" 2>/dev/null; then
    echo "FAIL: ${label} leaks sender key" >&2; exit 1
  fi
  if grep -qi -- "${GEN_KEY_HEX}" "${cf}" 2>/dev/null; then
    echo "FAIL: ${label} leaks fee_payer keystore key" >&2; exit 1
  fi
  if grep -q -- "${KS_PASSWORD}" "${cf}" 2>/dev/null; then
    echo "FAIL: ${label} leaks fee_payer password" >&2; exit 1
  fi
done
assert_eq "leak-checked" "leak-checked" "no fd-key/password in stdout/stderr/log"
```

- [ ] **Step 2: New `tests/unit/tests/node-tx-set-code.sh`**

Standalone test like `node-tx-wait.sh`. Mock returns `eth_chainId: 0x1` +
captures `eth_sendRawTransaction` param. Call `node.tx_send` with
`authorization_list` of one entry; assert the captured raw tx starts with
`0x04` (SetCode type byte) and that result has `tx_hash`.

- [ ] **Step 3: New `tests/unit/tests/node-tx-fee-delegation.sh`**

Standalone. Mock advertises `eth_chainId: 0x1` + (during attach probe) a
positive `wbft_*` or stablenet-marker response so the network gets
`chain_type: stablenet` (or use `--override stablenet` in the attach call).
Call `node.tx_fee_delegation_send` with synthetic sender + fee_payer KEY
env vars; assert raw tx starts with `0x16`. Also test the `ethereum`
override → NOT_SUPPORTED.

- [ ] **Step 4: Go E2E**

Append to `e2e_test.go`:

- `TestE2E_NodeTxSend_SetCode_AgainstAttachedRemote` — cobra in-process,
  attach + send with one authorization; assert mock saw type-4 raw tx.
- `TestE2E_NodeTxFeeDelegationSend_AgainstAttachedRemote` — cobra
  in-process, attach with `--override stablenet`, send fee-delegation tx;
  assert mock saw type-0x16 raw tx.

- [ ] **Step 5: Docs**

`SECURITY_KEY_HANDLING.md`:
- "Boundary Enforcement" section — add a bullet for Scenario 4
- "Developer Contract" — add the new bullet "SignHash returns 65 raw
  bytes; callers (handlers) compose into V/R/S without ever touching the
  underlying private key."

`EVALUATION_CAPABILITY.md`:
- §2 Tx matrix: `Fee Delegation (0x16)` Go col `❌ Sprint 4c (가칭)` → `✅ Sprint 4c`
- §2 Tx matrix: `EIP-7702 SetCode (0x4)` Go col `❌ Sprint 4c (가칭)` → `✅ Sprint 4c`
- §5 Sprint roadmap: 4c row `🚧 계획` → `✅ 완료 (2026-04-27)`
- §6 Coverage: Go `network/` from ~35% → ~45% (estimate)
- Append a one-line completion note at end

`VISION_AND_ROADMAP.md`:
- §6 Sprint 4c block: tick + "완료 (2026-04-27)"
- Header `최종 업데이트` line bumped

`NEXT_WORK.md`:
- §3 P2.5 Sprint 4c row → "✅ 완료 (2026-04-27)" with the 5+ commits this sprint produced
- §3.4 — drop any rows that this sprint resolves (likely none — fee-delegation
  follow-ups stay open as 4d/5)
- Header `최종 업데이트` bumped

- [ ] **Step 6: Full matrix verification**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./... -count=1 -timeout=120s
bash /Users/wm-it-22-00661/Work/github/tools/chainbench/tests/unit/run.sh
go -C /Users/wm-it-22-00661/Work/github/tools/chainbench/network vet ./...
gofmt -l /Users/wm-it-22-00661/Work/github/tools/chainbench/network/
```

All green. Bash count: was 28 → expect 30 (added node-tx-set-code.sh +
node-tx-fee-delegation.sh).

- [ ] **Step 7: Single docs+tests commit**

```bash
git add tests/unit/tests/security-key-boundary.sh tests/unit/tests/node-tx-set-code.sh tests/unit/tests/node-tx-fee-delegation.sh network/cmd/chainbench-net/e2e_test.go docs/
git commit -m "test+docs(sprint-4c): SetCode + fee-delegation E2E + boundary + roadmap

Three new layers of test coverage plus all roadmap docs ticked:

- security-key-boundary.sh Scenario 4: fee-delegation tx with two
  distinct signer keys (sender raw + fee_payer keystore). Asserts
  none of {sender hex, fee_payer raw hex, fee_payer keystore password}
  surfaces in stdout / stderr / log.

- tests/unit/tests/node-tx-set-code.sh: spawn binary against a python
  mock; one-authorization SetCodeTx broadcast; assert wire byte == 0x04.

- tests/unit/tests/node-tx-fee-delegation.sh: spawn binary; attach
  with --override stablenet; assert wire byte == 0x16 on broadcast +
  NOT_SUPPORTED for ethereum override.

- Two cobra in-process E2E tests for the same flows.

- SECURITY_KEY_HANDLING extended for SignHash + Scenario 4.
- EVALUATION_CAPABILITY ticks 0x4 + 0x16 cells, bumps Go coverage.
- VISION_AND_ROADMAP + NEXT_WORK absorb the sprint commits."
```

- [ ] **Step 8: Report**

Commit range, package-level test counts, bash test counts, capability
matrix delta, deferrals confirmed (Sprint 4d for contract/event/state;
Sprint 5 for MCP; Adapter.SupportedTxTypes promotion).
