# Sprint 4b — Keystore + EIP-1559 + tx.wait Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Extend the Sprint 4 baseline with keystore-backed signer loading,
EIP-1559 dynamic-fee opt-in for `node.tx_send`, and a new `node.tx_wait`
command that polls receipts with bounded backoff. Lock the keystore key
boundary with extended unit + bash boundary tests.

**Architecture:** Keystore branch lives inside the existing `signer` package
(no new package); `Load` checks raw-key env first, then `_KEYSTORE` +
`_KEYSTORE_PASSWORD`. Handler-side, `node.tx_send` adds `max_fee_per_gas` /
`max_priority_fee_per_gas` and a tx-type selector that rejects mixed legacy +
1559. `node.tx_wait` is a new handler in `handlers_node_tx.go` that polls
`remote.Client.TransactionReceipt` (newly added) until the receipt arrives or
the deadline elapses. The Sprint 4 redaction contract is preserved everywhere.

**Tech Stack:** Go 1.25, `go-ethereum/accounts/keystore`, existing
`ethclient` + `core/types`, stdlib only otherwise.

Spec: `docs/superpowers/specs/2026-04-27-sprint-4b-keystore-1559-tx-wait.md`

---

## Commit Discipline

- English commit messages.
- NO `Co-Authored-By` trailer; NO "Generated with Claude Code"; NO emoji.
- Conventional Commits prefix: `feat(scope):` / `fix(scope):` /
  `test(scope):` / `refactor(scope):` / `docs:` / `style(scope):` /
  `chore(scope):`.
- Key material / passwords MUST NEVER appear in test assertion strings,
  error messages, or log lines. Reviewers enforce this.

## File Structure

**Modify:**
- `network/internal/signer/signer.go` — keystore branch + helpers
- `network/internal/signer/signer_test.go` — keystore tests
- `network/internal/drivers/remote/client.go` — `TransactionReceipt`
- `network/internal/drivers/remote/client_test.go` — receipt tests
- `network/cmd/chainbench-net/handlers_node_tx.go` — 1559 selector + `node.tx_wait`
- `network/cmd/chainbench-net/handlers.go` — register `node.tx_wait`
- `network/cmd/chainbench-net/handlers_test.go` — 1559 + tx_wait unit tests
- `network/cmd/chainbench-net/e2e_test.go` — E2E for both flows
- `network/schema/command.json` — add `"node.tx_wait"`
- `network/internal/types/command_gen.go` — regenerated
- `tests/unit/tests/security-key-boundary.sh` — keystore variant
- `docs/SECURITY_KEY_HANDLING.md` — promote keystore to current
- `docs/VISION_AND_ROADMAP.md` — tick Sprint 4b
- `docs/EVALUATION_CAPABILITY.md` — bump cells
- `docs/NEXT_WORK.md` — drop resolved P2 row

**Create:**
- `tests/unit/tests/node-tx-wait.sh`

---

## Task 1 — signer keystore provider

**Files:**
- Modify: `network/internal/signer/signer.go`
- Modify: `network/internal/signer/signer_test.go`

- [ ] **Step 1: Write failing tests**

Append to `signer_test.go`:

```go
func keystoreFixture(t *testing.T, dir, password string) (path string, addr common.Address) {
    t.Helper()
    key, err := crypto.GenerateKey()
    if err != nil { t.Fatal(err) }
    id, err := uuid.NewRandom()
    if err != nil { t.Fatal(err) }
    k := &keystore.Key{
        Id:         id,
        Address:    crypto.PubkeyToAddress(key.PublicKey),
        PrivateKey: key,
    }
    enc, err := keystore.EncryptKey(k, password, keystore.LightScryptN, keystore.LightScryptP)
    if err != nil { t.Fatal(err) }
    path = filepath.Join(dir, "keystore.json")
    if err := os.WriteFile(path, enc, 0o600); err != nil { t.Fatal(err) }
    return path, k.Address
}

func TestLoad_Keystore_Happy(t *testing.T) {
    dir := t.TempDir()
    path, want := keystoreFixture(t, dir, "secret")
    t.Setenv("CHAINBENCH_SIGNER_ALICE_KEYSTORE", path)
    t.Setenv("CHAINBENCH_SIGNER_ALICE_KEYSTORE_PASSWORD", "secret")
    s, err := signer.Load("alice")
    if err != nil { t.Fatalf("Load: %v", err) }
    if s.Address() != want {
        t.Errorf("addr = %s, want %s", s.Address().Hex(), want.Hex())
    }
}

func TestLoad_Keystore_WrongPassword(t *testing.T) {
    dir := t.TempDir()
    path, _ := keystoreFixture(t, dir, "right")
    t.Setenv("CHAINBENCH_SIGNER_BOB_KEYSTORE", path)
    t.Setenv("CHAINBENCH_SIGNER_BOB_KEYSTORE_PASSWORD", "wrong")
    _, err := signer.Load("bob")
    if !errors.Is(err, signer.ErrInvalidKey) {
        t.Errorf("err = %v, want ErrInvalidKey", err)
    }
    if strings.Contains(err.Error(), "wrong") {
        t.Errorf("error leaks password: %v", err)
    }
}

func TestLoad_Keystore_MissingPasswordEnv(t *testing.T) {
    dir := t.TempDir()
    path, _ := keystoreFixture(t, dir, "secret")
    t.Setenv("CHAINBENCH_SIGNER_CAROL_KEYSTORE", path)
    // Deliberately do not set password.
    _, err := signer.Load("carol")
    if !errors.Is(err, signer.ErrInvalidKey) {
        t.Errorf("err = %v, want ErrInvalidKey", err)
    }
}

func TestLoad_Keystore_FileNotFound(t *testing.T) {
    t.Setenv("CHAINBENCH_SIGNER_DAVE_KEYSTORE", "/nonexistent/keystore.json")
    t.Setenv("CHAINBENCH_SIGNER_DAVE_KEYSTORE_PASSWORD", "x")
    _, err := signer.Load("dave")
    if !errors.Is(err, signer.ErrInvalidKey) {
        t.Errorf("err = %v, want ErrInvalidKey", err)
    }
}

func TestLoad_Keystore_RawKeyWins(t *testing.T) {
    // Both env paths set: raw KEY must take precedence.
    dir := t.TempDir()
    path, ksAddr := keystoreFixture(t, dir, "p")
    t.Setenv("CHAINBENCH_SIGNER_EVE_KEYSTORE", path)
    t.Setenv("CHAINBENCH_SIGNER_EVE_KEYSTORE_PASSWORD", "p")
    rawKey := "0x" + keyHex64
    t.Setenv("CHAINBENCH_SIGNER_EVE_KEY", rawKey)
    s, err := signer.Load("eve")
    if err != nil { t.Fatal(err) }
    if s.Address() == ksAddr {
        t.Errorf("address came from keystore; raw KEY should win")
    }
}

func TestLoad_Keystore_RedactionBoundary(t *testing.T) {
    dir := t.TempDir()
    path, _ := keystoreFixture(t, dir, "p")
    t.Setenv("CHAINBENCH_SIGNER_FRANK_KEYSTORE", path)
    t.Setenv("CHAINBENCH_SIGNER_FRANK_KEYSTORE_PASSWORD", "p")
    s, err := signer.Load("frank")
    if err != nil { t.Fatal(err) }
    // %v / %+v / %#v / %s and slog.TextHandler must all redact.
    var buf bytes.Buffer
    logger := slog.New(slog.NewTextHandler(&buf, nil))
    logger.Info("loaded", "signer", s)
    if !strings.Contains(buf.String(), "***") {
        t.Errorf("slog must redact: %q", buf.String())
    }
    for _, format := range []string{"%v", "%+v", "%#v", "%s"} {
        out := fmt.Sprintf(format, s)
        if strings.Contains(out, "PrivateKey") || strings.Contains(out, "ecdsa") {
            t.Errorf("fmt %s leaks structure: %q", format, out)
        }
    }
}
```

Imports to add:
```go
import (
    "os"
    "path/filepath"

    "github.com/ethereum/go-ethereum/accounts/keystore"
    "github.com/ethereum/go-ethereum/crypto"
    "github.com/google/uuid"
)
```

- [ ] **Step 2: Verify RED**

```bash
cd network && go test ./internal/signer/... -count=1 -v
```

Expected: keystore tests fail (Load returns ErrUnknownAlias since the
keystore branch doesn't exist yet).

- [ ] **Step 3: Implement keystore branch in signer.go**

Add:

```go
const (
    envKeySuffix             = "_KEY"
    envKeystoreSuffix        = "_KEYSTORE"
    envKeystorePasswordSuffix = "_KEYSTORE_PASSWORD"
)

func envName(alias, suffix string) string {
    return "CHAINBENCH_SIGNER_" + strings.ToUpper(alias) + suffix
}

// loadFromKeystore reads an encrypted keystore file at `path` and decrypts
// it with `password`. Any failure returns an ErrInvalidKey wrapper that
// references the alias and the relevant env var name only — never the file
// content, the path content, or the password value.
func loadFromKeystore(alias Alias, path, password string) (*sealed, error) {
    a := string(alias)
    if password == "" {
        return nil, fmt.Errorf("%w: alias=%s (env %s not set)",
            ErrInvalidKey, a, envName(a, envKeystorePasswordSuffix))
    }
    raw, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("%w: alias=%s (env %s unreadable)",
            ErrInvalidKey, a, envName(a, envKeystoreSuffix))
    }
    k, err := keystore.DecryptKey(raw, password)
    if err != nil {
        return nil, fmt.Errorf("%w: alias=%s (env %s decrypt failed)",
            ErrInvalidKey, a, envName(a, envKeystoreSuffix))
    }
    return &sealed{
        alias: alias,
        addr:  crypto.PubkeyToAddress(k.PrivateKey.PublicKey),
        key:   k.PrivateKey,
    }, nil
}
```

Modify `Load` to add the keystore branch after the raw-key branch:

```go
func Load(alias Alias) (Signer, error) {
    a := string(alias)
    if a == "" || !aliasRE.MatchString(a) {
        return nil, fmt.Errorf("%w: %q", ErrInvalidAlias, a)
    }
    keyEnv := envName(a, envKeySuffix)
    if raw := os.Getenv(keyEnv); raw != "" {
        return loadFromRawKey(alias, raw, keyEnv)
    }
    ksEnv := envName(a, envKeystoreSuffix)
    if path := os.Getenv(ksEnv); path != "" {
        password := os.Getenv(envName(a, envKeystorePasswordSuffix))
        return loadFromKeystore(alias, path, password)
    }
    return nil, fmt.Errorf("%w: %s (env %s and %s not set)",
        ErrUnknownAlias, a, keyEnv, ksEnv)
}

// loadFromRawKey factors the existing Sprint 4 path so Load reads cleanly.
func loadFromRawKey(alias Alias, raw, envName string) (*sealed, error) {
    hexStr := strings.TrimPrefix(raw, "0x")
    key, err := crypto.HexToECDSA(hexStr)
    if err != nil {
        return nil, fmt.Errorf("%w: alias=%s (env %s)",
            ErrInvalidKey, string(alias), envName)
    }
    return &sealed{
        alias: alias,
        addr:  crypto.PubkeyToAddress(key.PublicKey),
        key:   key,
    }, nil
}
```

Imports to add:
```go
import "github.com/ethereum/go-ethereum/accounts/keystore"
```

- [ ] **Step 4: Run tests, iterate**

```bash
cd network && go test ./internal/signer/... -v -count=1
```

Expected: all PASS, including new keystore cases AND the Sprint 4 cases
(regression check).

- [ ] **Step 5: Full suite + lint**

```bash
cd network && go test ./... -count=1 -timeout=60s
cd network && go vet ./...
gofmt -l network/
```

- [ ] **Step 6: Commit**

```bash
git add network/internal/signer/
git commit -m "feat(signer): keystore-backed key loader

Extend signer.Load to fall back to CHAINBENCH_SIGNER_<ALIAS>_KEYSTORE
plus _KEYSTORE_PASSWORD when the raw _KEY env var is unset. Decrypt
via go-ethereum/accounts/keystore.DecryptKey; the resulting *sealed is
the same shape as the raw-key path so handlers see one Signer
interface.

Resolution order: raw _KEY wins over _KEYSTORE so operators can pin
a deterministic test key without removing the keystore env. Both
unset still surfaces ErrUnknownAlias.

All keystore failure modes (missing password, unreadable file, decrypt
mismatch) wrap ErrInvalidKey with the alias and env var name only.
File content, path bytes, and password are never embedded in error
messages or log lines. Boundary tests cover %v/%+v/%#v/%s and slog
on a keystore-loaded signer."
```

---

## Task 2 — EIP-1559 selection in node.tx_send

**Files:**
- Modify: `network/cmd/chainbench-net/handlers_node_tx.go`
- Modify: `network/cmd/chainbench-net/handlers_test.go`

- [ ] **Step 1: Write failing tests**

Append to `handlers_test.go`:

```go
// decodeBroadcastTxType pulls the first byte of the rlp-prefixed raw hex from
// eth_sendRawTransaction params. Type-0 (legacy) does not have a leading
// type byte; types 1+ do. We surface an int (0 for legacy) for assertions.
func decodeBroadcastTxType(t *testing.T, rawHex string) int {
    t.Helper()
    raw := strings.TrimPrefix(rawHex, "0x")
    if len(raw) < 2 { t.Fatalf("raw too short: %q", rawHex) }
    var b [1]byte
    if _, err := hex.Decode(b[:], []byte(raw[:2])); err != nil {
        t.Fatalf("decode: %v", err)
    }
    // Legacy txs are RLP-encoded lists starting with 0xc0..0xff.
    if b[0] >= 0xc0 { return 0 }
    return int(b[0])
}

func TestHandleNodeTxSend_DynamicFee_Happy(t *testing.T) {
    var sentRaw string
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req struct {
            Method string
            ID     json.RawMessage
            Params []json.RawMessage
        }
        _ = json.NewDecoder(r.Body).Decode(&req)
        w.Header().Set("Content-Type", "application/json")
        switch req.Method {
        case "eth_chainId":
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1"}`, req.ID)
        case "eth_sendRawTransaction":
            if len(req.Params) > 0 {
                _ = json.Unmarshal(req.Params[0], &sentRaw)
            }
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1111111111111111111111111111111111111111111111111111111111111111"}`, req.ID)
        default:
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
        }
    }))
    defer srv.Close()

    stateDir := t.TempDir()
    saveRemoteFixture(t, stateDir, "tn", srv.URL)
    t.Setenv("CHAINBENCH_SIGNER_ALICE_KEY", "0xb71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

    h := newHandleNodeTxSend(stateDir)
    args, _ := json.Marshal(map[string]any{
        "network":                  "tn",
        "node_id":                  "node1",
        "signer":                   "alice",
        "to":                       "0x0000000000000000000000000000000000000002",
        "value":                    "0x0",
        "gas":                      21000,
        "max_fee_per_gas":          "0x59682f00",
        "max_priority_fee_per_gas": "0x3b9aca00",
        "nonce":                    0,
    })
    bus, _ := newTestBus(t)
    defer bus.Close()
    if _, err := h(args, bus); err != nil {
        t.Fatalf("handler: %v", err)
    }
    if sentRaw == "" {
        t.Fatal("mock did not see eth_sendRawTransaction")
    }
    if got := decodeBroadcastTxType(t, sentRaw); got != 2 {
        t.Errorf("tx type = %d, want 2 (DynamicFee)", got)
    }
}

func TestHandleNodeTxSend_MixedFeeFields(t *testing.T) {
    h := newHandleNodeTxSend(t.TempDir())
    args, _ := json.Marshal(map[string]any{
        "network":         "tn",
        "node_id":         "node1",
        "signer":          "alice",
        "to":              "0x0000000000000000000000000000000000000002",
        "gas_price":       "0x1",
        "max_fee_per_gas": "0x2",
    })
    bus, _ := newTestBus(t)
    defer bus.Close()
    _, err := h(args, bus)
    var api *APIError
    if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
        t.Errorf("want INVALID_ARGS, got %v", err)
    }
}

func TestHandleNodeTxSend_PartialDynamicFee(t *testing.T) {
    h := newHandleNodeTxSend(t.TempDir())
    args, _ := json.Marshal(map[string]any{
        "network":         "tn",
        "node_id":         "node1",
        "signer":          "alice",
        "to":              "0x0000000000000000000000000000000000000002",
        "max_fee_per_gas": "0x2",
    })
    bus, _ := newTestBus(t)
    defer bus.Close()
    _, err := h(args, bus)
    var api *APIError
    if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
        t.Errorf("want INVALID_ARGS, got %v", err)
    }
}
```

- [ ] **Step 2: Verify RED**

```bash
cd network && go test ./cmd/chainbench-net/... -run 'TestHandleNodeTxSend_DynamicFee|TestHandleNodeTxSend_MixedFeeFields|TestHandleNodeTxSend_PartialDynamicFee' -v -count=1
```

Expected: compile passes; new tests fail (handler still always emits Legacy).

- [ ] **Step 3: Implement 1559 branch in handlers_node_tx.go**

Extend the request struct with the two new fields:

```go
var req struct {
    Network                string  `json:"network"`
    NodeID                 string  `json:"node_id"`
    Signer                 string  `json:"signer"`
    To                     string  `json:"to"`
    Value                  string  `json:"value"`
    Data                   string  `json:"data"`
    Gas                    *uint64 `json:"gas"`
    GasPrice               string  `json:"gas_price"`
    MaxFeePerGas           string  `json:"max_fee_per_gas"`
    MaxPriorityFeePerGas   string  `json:"max_priority_fee_per_gas"`
    Nonce                  *uint64 `json:"nonce"`
}
```

After the existing `req.Value` / `req.Data` parsing, before signer.Load, add
the fee-mode selection block:

```go
hasLegacy := req.GasPrice != ""
hasMaxFee := req.MaxFeePerGas != ""
hasTip    := req.MaxPriorityFeePerGas != ""

if hasLegacy && (hasMaxFee || hasTip) {
    return nil, NewInvalidArgs("args.gas_price cannot be combined with args.max_fee_per_gas or args.max_priority_fee_per_gas")
}
if hasMaxFee != hasTip {
    return nil, NewInvalidArgs("args.max_fee_per_gas and args.max_priority_fee_per_gas must both be provided when using EIP-1559")
}
useDynamicFee := hasMaxFee && hasTip

var maxFee, maxPriorityFee *big.Int
if useDynamicFee {
    var ok bool
    maxFee, ok = new(big.Int).SetString(strings.TrimPrefix(req.MaxFeePerGas, "0x"), 16)
    if !ok {
        return nil, NewInvalidArgs(fmt.Sprintf("args.max_fee_per_gas is not valid hex: %q", req.MaxFeePerGas))
    }
    maxPriorityFee, ok = new(big.Int).SetString(strings.TrimPrefix(req.MaxPriorityFeePerGas, "0x"), 16)
    if !ok {
        return nil, NewInvalidArgs(fmt.Sprintf("args.max_priority_fee_per_gas is not valid hex: %q", req.MaxPriorityFeePerGas))
    }
}
```

After `chainID` / `nonce` are resolved and `gas` is computed, swap the
LegacyTx construction for a branch:

```go
var unsigned *ethtypes.Transaction
if useDynamicFee {
    unsigned = ethtypes.NewTx(&ethtypes.DynamicFeeTx{
        ChainID:   chainID,
        Nonce:     nonce,
        GasTipCap: maxPriorityFee,
        GasFeeCap: maxFee,
        Gas:       gas,
        To:        &to,
        Value:     value,
        Data:      data,
    })
} else {
    gasPrice, err := resolveGasPrice(ctx, client, req.GasPrice)
    if err != nil {
        return nil, err
    }
    unsigned = ethtypes.NewTx(&ethtypes.LegacyTx{
        Nonce:    nonce,
        GasPrice: gasPrice,
        Gas:      gas,
        To:       &to,
        Value:    value,
        Data:     data,
    })
}
```

(The legacy `resolveGasPrice` call moves under the `else` branch so the
1559 path doesn't waste an `eth_gasPrice` round-trip.)

- [ ] **Step 4: Run tests, iterate**

```bash
cd network && go test ./cmd/chainbench-net/... -count=1 -timeout=60s
```

Expected: all green including the existing `TestHandleNodeTxSend_Happy`
(legacy regression).

- [ ] **Step 5: Lint + commit**

```bash
cd network && go vet ./... && gofmt -l network/
git add network/cmd/chainbench-net/handlers_node_tx.go network/cmd/chainbench-net/handlers_test.go
git commit -m "feat(network-net): EIP-1559 dynamic fee in node.tx_send

Extend node.tx_send args with optional max_fee_per_gas /
max_priority_fee_per_gas. When both are provided the handler builds a
DynamicFeeTx (type 0x2); when only gas_price is provided the legacy
path stays unchanged. Mixed legacy + 1559 or partial 1559 are rejected
as INVALID_ARGS at the boundary so structural mistakes never reach the
signer.

Auto-fill semantics: in this sprint, opting into 1559 requires both
fields. eth_feeHistory-based inference is a follow-up if needed; the
narrow surface keeps the failure modes simple.

types.LatestSignerForChainID already supports DynamicFeeTx so the
signer requires no change. Tests cover happy 1559, mixed-field reject,
and partial-1559 reject; the legacy happy path remains a regression
guard."
```

---

## Task 3 — remote.Client.TransactionReceipt + node.tx_wait

**Files:**
- Modify: `network/internal/drivers/remote/client.go`
- Modify: `network/internal/drivers/remote/client_test.go`
- Modify: `network/cmd/chainbench-net/handlers_node_tx.go`
- Modify: `network/cmd/chainbench-net/handlers.go`
- Modify: `network/cmd/chainbench-net/handlers_test.go`
- Modify: `network/schema/command.json`
- Modify: `network/internal/types/command_gen.go` (regenerated)

- [ ] **Step 1: client TransactionReceipt — failing test + implementation**

Append to `client_test.go`:

```go
func TestClient_TransactionReceipt_Found(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req struct{ Method string; ID json.RawMessage }
        _ = json.NewDecoder(r.Body).Decode(&req)
        w.Header().Set("Content-Type", "application/json")
        if req.Method == "eth_getTransactionReceipt" {
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{
                "transactionHash":"0x%s",
                "blockHash":"0x2222222222222222222222222222222222222222222222222222222222222222",
                "blockNumber":"0x1",
                "gasUsed":"0x5208",
                "status":"0x1",
                "contractAddress":null,
                "logs":[]}}`, req.ID, strings.Repeat("a", 64))
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
    h := common.HexToHash("0x" + strings.Repeat("a", 64))
    rcpt, err := c.TransactionReceipt(ctx, h)
    if err != nil { t.Fatalf("TransactionReceipt: %v", err) }
    if rcpt.Status != 1 { t.Errorf("status = %d, want 1", rcpt.Status) }
}

func TestClient_TransactionReceipt_NotFound(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req struct{ Method string; ID json.RawMessage }
        _ = json.NewDecoder(r.Body).Decode(&req)
        w.Header().Set("Content-Type", "application/json")
        if req.Method == "eth_getTransactionReceipt" {
            // ethclient.TransactionReceipt returns ethereum.NotFound when result is null.
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":null}`, req.ID)
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
    _, err = c.TransactionReceipt(ctx, common.HexToHash("0x"+strings.Repeat("b", 64)))
    if !errors.Is(err, ethereum.NotFound) {
        t.Errorf("err = %v, want ethereum.NotFound", err)
    }
}
```

Add to `client.go`:

```go
// TransactionReceipt fetches the receipt for a tx hash. Returns
// ethereum.NotFound (verbatim) when the endpoint reports a null result,
// so callers can distinguish "still pending" from a real RPC failure
// without string-matching error messages.
func (c *Client) TransactionReceipt(ctx context.Context, hash common.Hash) (*types.Receipt, error) {
    rcpt, err := c.rpc.TransactionReceipt(ctx, hash)
    if err != nil {
        if errors.Is(err, ethereum.NotFound) {
            return nil, ethereum.NotFound
        }
        return nil, fmt.Errorf("remote.TransactionReceipt: %w", err)
    }
    return rcpt, nil
}
```

Add `errors` import if not already present.

- [ ] **Step 2: schema + regen**

Add `"node.tx_wait"` to `command.json` enum (alphabetical position).
Then:

```bash
cd network && go generate ./...
```

- [ ] **Step 3: Write failing handler tests**

Append to `handlers_test.go`:

```go
func TestHandleNodeTxWait_SuccessImmediate(t *testing.T) {
    txHash := "0x" + strings.Repeat("a", 64)
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req struct{ Method string; ID json.RawMessage }
        _ = json.NewDecoder(r.Body).Decode(&req)
        w.Header().Set("Content-Type", "application/json")
        if req.Method == "eth_getTransactionReceipt" {
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{
                "transactionHash":%q,
                "blockHash":"0x%s",
                "blockNumber":"0x10",
                "gasUsed":"0x5208",
                "effectiveGasPrice":"0x3b9aca00",
                "status":"0x1",
                "contractAddress":null,
                "logs":[]}}`, req.ID, txHash, strings.Repeat("b", 64))
            return
        }
        fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
    }))
    defer srv.Close()
    stateDir := t.TempDir()
    saveRemoteFixture(t, stateDir, "tn", srv.URL)
    h := newHandleNodeTxWait(stateDir)
    args, _ := json.Marshal(map[string]any{
        "network":  "tn",
        "node_id":  "node1",
        "tx_hash":  txHash,
    })
    bus, _ := newTestBus(t); defer bus.Close()
    data, err := h(args, bus)
    if err != nil { t.Fatalf("handler: %v", err) }
    if got, _ := data["status"].(string); got != "success" {
        t.Errorf("status = %v, want success", data["status"])
    }
    if got, _ := data["block_number"].(uint64); got != 16 {
        t.Errorf("block_number = %v, want 16", data["block_number"])
    }
}

func TestHandleNodeTxWait_FailedReceipt(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req struct{ Method string; ID json.RawMessage }
        _ = json.NewDecoder(r.Body).Decode(&req)
        w.Header().Set("Content-Type", "application/json")
        if req.Method == "eth_getTransactionReceipt" {
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{
                "transactionHash":"0x%s",
                "blockHash":"0x%s",
                "blockNumber":"0x10","gasUsed":"0x5208","effectiveGasPrice":"0x1",
                "status":"0x0","contractAddress":null,"logs":[]}}`, req.ID, strings.Repeat("a",64), strings.Repeat("b",64))
            return
        }
        fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
    }))
    defer srv.Close()
    stateDir := t.TempDir()
    saveRemoteFixture(t, stateDir, "tn", srv.URL)
    h := newHandleNodeTxWait(stateDir)
    args, _ := json.Marshal(map[string]any{
        "network": "tn", "node_id": "node1", "tx_hash": "0x"+strings.Repeat("a",64),
    })
    bus, _ := newTestBus(t); defer bus.Close()
    data, err := h(args, bus)
    if err != nil { t.Fatalf("handler: %v", err) }
    if got, _ := data["status"].(string); got != "failed" {
        t.Errorf("status = %v, want failed", data["status"])
    }
}

func TestHandleNodeTxWait_NotFoundThenSuccess(t *testing.T) {
    var calls int
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req struct{ Method string; ID json.RawMessage }
        _ = json.NewDecoder(r.Body).Decode(&req)
        w.Header().Set("Content-Type", "application/json")
        if req.Method == "eth_getTransactionReceipt" {
            calls++
            if calls == 1 {
                fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":null}`, req.ID)
                return
            }
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{
                "transactionHash":"0x%s","blockHash":"0x%s","blockNumber":"0x1",
                "gasUsed":"0x5208","status":"0x1","contractAddress":null,"logs":[]}}`,
                req.ID, strings.Repeat("a",64), strings.Repeat("b",64))
            return
        }
        fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
    }))
    defer srv.Close()
    stateDir := t.TempDir()
    saveRemoteFixture(t, stateDir, "tn", srv.URL)
    h := newHandleNodeTxWait(stateDir)
    args, _ := json.Marshal(map[string]any{
        "network": "tn", "node_id": "node1", "tx_hash": "0x"+strings.Repeat("a",64),
        "timeout_ms": 5000,
    })
    bus, _ := newTestBus(t); defer bus.Close()
    data, err := h(args, bus)
    if err != nil { t.Fatalf("handler: %v", err) }
    if got, _ := data["status"].(string); got != "success" {
        t.Errorf("status = %v, want success", data["status"])
    }
    if calls < 2 {
        t.Errorf("expected at least 2 polls, got %d", calls)
    }
}

func TestHandleNodeTxWait_Timeout(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req struct{ Method string; ID json.RawMessage }
        _ = json.NewDecoder(r.Body).Decode(&req)
        w.Header().Set("Content-Type", "application/json")
        if req.Method == "eth_getTransactionReceipt" {
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":null}`, req.ID)
            return
        }
        fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
    }))
    defer srv.Close()
    stateDir := t.TempDir()
    saveRemoteFixture(t, stateDir, "tn", srv.URL)
    h := newHandleNodeTxWait(stateDir)
    args, _ := json.Marshal(map[string]any{
        "network": "tn", "node_id": "node1", "tx_hash": "0x"+strings.Repeat("a",64),
        "timeout_ms": 1500,
    })
    bus, _ := newTestBus(t); defer bus.Close()
    data, err := h(args, bus)
    if err != nil { t.Fatalf("handler: %v", err) }
    if got, _ := data["status"].(string); got != "pending" {
        t.Errorf("status = %v, want pending", data["status"])
    }
}

func TestHandleNodeTxWait_BadHash(t *testing.T) {
    h := newHandleNodeTxWait(t.TempDir())
    args, _ := json.Marshal(map[string]any{
        "network": "tn", "node_id": "node1", "tx_hash": "not-a-hash",
    })
    bus, _ := newTestBus(t); defer bus.Close()
    _, err := h(args, bus)
    var api *APIError
    if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
        t.Errorf("want INVALID_ARGS, got %v", err)
    }
}

func TestHandleNodeTxWait_TimeoutOutOfRange(t *testing.T) {
    h := newHandleNodeTxWait(t.TempDir())
    args, _ := json.Marshal(map[string]any{
        "network": "tn", "node_id": "node1", "tx_hash": "0x"+strings.Repeat("a",64),
        "timeout_ms": 50,
    })
    bus, _ := newTestBus(t); defer bus.Close()
    _, err := h(args, bus)
    var api *APIError
    if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
        t.Errorf("want INVALID_ARGS, got %v", err)
    }
}

func TestAllHandlers_IncludesTxWait(t *testing.T) {
    h := allHandlers("x", "y")
    if _, ok := h["node.tx_wait"]; !ok {
        t.Error("allHandlers missing node.tx_wait")
    }
}
```

- [ ] **Step 4: Implement newHandleNodeTxWait + register**

Append to `handlers_node_tx.go`:

```go
const (
    minTxWaitMs     = 1000
    defaultTxWaitMs = 60000
    maxTxWaitMs     = 600000
    txWaitInitial   = 200 * time.Millisecond
    txWaitCap       = 2 * time.Second
)

// newHandleNodeTxWait polls eth_getTransactionReceipt with bounded backoff
// until the receipt is available, the context deadline elapses, or an
// upstream error surfaces. ethereum.NotFound is the polling-tick signal
// (still pending); other RPC errors are upstream failures.
//
// Args:    {network?, node_id, tx_hash, timeout_ms? (1000..600000, default 60000)}
// Result:  {status: "success"|"failed"|"pending", tx_hash, block_number?,
//           block_hash?, gas_used?, effective_gas_price?, contract_address?,
//           logs_count?}
//
// On terminal "pending" (timeout), only {status, tx_hash} are returned —
// the caller decides whether to retry.
func newHandleNodeTxWait(stateDir string) Handler {
    return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
        var req struct {
            Network   string `json:"network"`
            NodeID    string `json:"node_id"`
            TxHash    string `json:"tx_hash"`
            TimeoutMs *int   `json:"timeout_ms"`
        }
        if len(args) > 0 {
            if err := json.Unmarshal(args, &req); err != nil {
                return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
            }
        }
        if req.TxHash == "" {
            return nil, NewInvalidArgs("args.tx_hash is required")
        }
        // 0x + 64 lowercase/uppercase hex chars
        if len(req.TxHash) != 66 || !strings.HasPrefix(req.TxHash, "0x") {
            return nil, NewInvalidArgs(fmt.Sprintf("args.tx_hash must be 0x + 32-byte hex: %q", req.TxHash))
        }
        if _, err := hex.DecodeString(strings.TrimPrefix(req.TxHash, "0x")); err != nil {
            return nil, NewInvalidArgs(fmt.Sprintf("args.tx_hash is not valid hex: %q", req.TxHash))
        }
        timeoutMs := defaultTxWaitMs
        if req.TimeoutMs != nil {
            timeoutMs = *req.TimeoutMs
        }
        if timeoutMs < minTxWaitMs || timeoutMs > maxTxWaitMs {
            return nil, NewInvalidArgs(fmt.Sprintf(
                "args.timeout_ms must be %d..%d, got %d",
                minTxWaitMs, maxTxWaitMs, timeoutMs,
            ))
        }

        _, node, err := resolveNode(stateDir, req.Network, req.NodeID)
        if err != nil {
            return nil, err
        }
        ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
        defer cancel()
        client, err := dialNode(ctx, &node)
        if err != nil {
            return nil, err
        }
        defer client.Close()

        hash := common.HexToHash(req.TxHash)
        delay := txWaitInitial
        for {
            rcpt, rerr := client.TransactionReceipt(ctx, hash)
            if rerr == nil {
                return receiptToResult(req.TxHash, rcpt), nil
            }
            if !errors.Is(rerr, ethereum.NotFound) {
                return nil, NewUpstream("eth_getTransactionReceipt", rerr)
            }
            // NotFound: backoff or fall through to "pending" on deadline.
            select {
            case <-ctx.Done():
                return map[string]any{"status": "pending", "tx_hash": req.TxHash}, nil
            case <-time.After(delay):
            }
            delay *= 2
            if delay > txWaitCap {
                delay = txWaitCap
            }
        }
    }
}

func receiptToResult(txHash string, rcpt *ethtypes.Receipt) map[string]any {
    status := "failed"
    if rcpt.Status == 1 {
        status = "success"
    }
    out := map[string]any{
        "status":              status,
        "tx_hash":             txHash,
        "block_number":        rcpt.BlockNumber.Uint64(),
        "block_hash":          rcpt.BlockHash.Hex(),
        "gas_used":            rcpt.GasUsed,
        "logs_count":          len(rcpt.Logs),
        "contract_address":    "",
        "effective_gas_price": "0x0",
    }
    if rcpt.ContractAddress != (common.Address{}) {
        out["contract_address"] = rcpt.ContractAddress.Hex()
    }
    if rcpt.EffectiveGasPrice != nil {
        out["effective_gas_price"] = "0x" + rcpt.EffectiveGasPrice.Text(16)
    }
    return out
}
```

Imports to add (if missing): `"time"`, `ethereum "github.com/ethereum/go-ethereum"`.

Register in `handlers.go` allHandlers:

```go
"node.tx_wait":     newHandleNodeTxWait(stateDir),
```

Update the file-layout comment block in `handlers.go` to mention the new handler.

- [ ] **Step 5: Run tests**

```bash
cd network && go test ./internal/drivers/remote/... -count=1 -v -run TransactionReceipt
cd network && go test ./cmd/chainbench-net/... -count=1 -v -run 'TestHandleNodeTxWait|TestAllHandlers_IncludesTxWait'
cd network && go test ./... -count=1 -timeout=60s
go -C network vet ./...
gofmt -l network/
```

- [ ] **Step 6: Commit (split for clarity — receipt vs handler)**

```bash
git add network/internal/drivers/remote/
git commit -m "feat(drivers/remote): add TransactionReceipt with ethereum.NotFound passthrough

Wraps ethclient.TransactionReceipt with the same remote.<Method> error
convention. ethereum.NotFound is propagated verbatim so callers can
distinguish 'still pending' from real RPC failures without
string-matching."

git add network/schema/command.json network/internal/types/command_gen.go \
        network/cmd/chainbench-net/handlers.go \
        network/cmd/chainbench-net/handlers_node_tx.go \
        network/cmd/chainbench-net/handlers_test.go
git commit -m "feat(network-net): node.tx_wait receipt polling

New command waits for a tx receipt with bounded exponential backoff
(200ms -> 2s cap, default 60s deadline, configurable 1s..10min).
ethereum.NotFound is the polling tick signal; other RPC failures
abort with UPSTREAM_ERROR. Result shape matches the spec — full
status / block / gas / logs_count surface on terminal receipts; just
{status: pending, tx_hash} on timeout so callers can retry with a
larger budget.

Tests cover immediate success, failed receipt, NotFound-then-success
polling loop, timeout fall-through, malformed tx_hash, out-of-range
timeout, and dispatcher registration."
```

---

## Task 4 — bash boundary update + new tx_wait test + Go E2E

**Files:**
- Modify: `tests/unit/tests/security-key-boundary.sh`
- Create: `tests/unit/tests/node-tx-wait.sh`
- Modify: `network/cmd/chainbench-net/e2e_test.go`

- [ ] **Step 1: Extend security-key-boundary.sh — keystore variant**

After the existing env-key block, add a second describe-block that:
1. Generates a synthetic keystore via a tiny Go helper or a python keystore
   library. Easiest: build a small one-shot Go program in the test that
   writes the keystore JSON via `keystore.EncryptKey`. Or invoke the
   chainbench-net binary to generate a keystore (out of scope for 4b).
2. Sets `CHAINBENCH_SIGNER_BOB_KEYSTORE` + `_KEYSTORE_PASSWORD`.
3. Runs `node.tx_send` (signer: "bob"), captures stdout/stderr/log.
4. Greps for the underlying raw-hex of the generated key AND the password
   string. Both must be absent.

A pragmatic shape for the helper: emit a tiny standalone program inside the
test under `${TMPDIR_ROOT}/gen-keystore/main.go`, `go run` it, then read
back. Keep the password short and the hex private to the helper run.

```bash
describe "security: keystore variant of tx.send"

GEN_DIR="${TMPDIR_ROOT}/gen-keystore"
mkdir -p "${GEN_DIR}"
cat > "${GEN_DIR}/main.go" <<'GOEOF'
package main

import (
    "crypto/ecdsa"
    "encoding/hex"
    "fmt"
    "os"

    "github.com/ethereum/go-ethereum/accounts/keystore"
    "github.com/ethereum/go-ethereum/crypto"
    "github.com/google/uuid"
)

func main() {
    if len(os.Args) < 3 {
        fmt.Fprintln(os.Stderr, "usage: gen-keystore <out-path> <password>")
        os.Exit(2)
    }
    outPath := os.Args[1]
    password := os.Args[2]
    pk, err := crypto.GenerateKey()
    if err != nil { panic(err) }
    id, _ := uuid.NewRandom()
    k := &keystore.Key{Id: id, Address: crypto.PubkeyToAddress(pk.PublicKey), PrivateKey: pk}
    enc, err := keystore.EncryptKey(k, password, keystore.LightScryptN, keystore.LightScryptP)
    if err != nil { panic(err) }
    if err := os.WriteFile(outPath, enc, 0o600); err != nil { panic(err) }
    // Print the raw hex to stderr for the test harness to capture and grep
    // against. The hex is rotated per-test via crypto.GenerateKey so it
    // never matches a real funded account.
    var pkVal *ecdsa.PrivateKey = pk
    fmt.Fprintln(os.Stderr, hex.EncodeToString(crypto.FromECDSA(pkVal)))
}
GOEOF

KS_PATH="${TMPDIR_ROOT}/keystore.json"
KS_PASSWORD="ks-test-pass"

GEN_LOG="${TMPDIR_ROOT}/gen.log"
( cd "${CHAINBENCH_DIR}/network" && go run "${GEN_DIR}/main.go" "${KS_PATH}" "${KS_PASSWORD}" ) 2>"${GEN_LOG}" >/dev/null
GEN_KEY_HEX="$(cat "${GEN_LOG}")"
[[ -n "${GEN_KEY_HEX}" ]] || { echo "FATAL: keystore gen produced no hex" >&2; exit 1; }

export CHAINBENCH_SIGNER_BOB_KEYSTORE="${KS_PATH}"
export CHAINBENCH_SIGNER_BOB_KEYSTORE_PASSWORD="${KS_PASSWORD}"

KS_TX_LOG="${TMPDIR_ROOT}/ks-tx.log"
KS_TX_ERR="${TMPDIR_ROOT}/ks-tx.err"
export CHAINBENCH_NET_LOG="${KS_TX_LOG}"
ks_tx_data="$(cb_net_call "node.tx_send" "{\"network\":\"sec\",\"node_id\":\"node1\",\"signer\":\"bob\",\"to\":\"0x0000000000000000000000000000000000000002\",\"value\":\"0x0\",\"gas\":21000,\"gas_price\":\"0x1\",\"nonce\":0}" 2>"${KS_TX_ERR}")"
assert_eq "$(jq -r .tx_hash <<<"$ks_tx_data" | cut -c1-2)" "0x" "ks tx_hash starts with 0x"

for label in "stdout" "stderr" "log"; do
  case "$label" in
    stdout) content="$ks_tx_data" ;;
    stderr) content="$(cat "${KS_TX_ERR}" 2>/dev/null || true)" ;;
    log)    content="$(cat "${KS_TX_LOG}" 2>/dev/null || true)" ;;
  esac
  if echo "$content" | grep -qi "${GEN_KEY_HEX}"; then
    echo "FAIL: ${label} leaks keystore raw key" >&2; exit 1
  fi
  if echo "$content" | grep -q "${KS_PASSWORD}"; then
    echo "FAIL: ${label} leaks keystore password" >&2; exit 1
  fi
done
assert_eq "leak-checked" "leak-checked" "no keystore key/password in stdout/stderr/log"
```

- [ ] **Step 2: New tests/unit/tests/node-tx-wait.sh**

Use the same template scaffolding (mock RPC, ephemeral port, trap) as
`security-key-boundary.sh`. Mock returns a successful receipt on first
poll; assert `status == "success"` and that block_number / gas_used echo.

```bash
#!/usr/bin/env bash
# tests/unit/tests/node-tx-wait.sh
# Exercises node.tx_wait against a mock RPC returning a successful receipt.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck disable=SC1091
source "${SCRIPT_DIR}/lib/assert.sh"
CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
export CHAINBENCH_DIR

TMPDIR_ROOT="$(mktemp -d)"
STATE_DIR="${TMPDIR_ROOT}/state"
mkdir -p "${STATE_DIR}"
export CHAINBENCH_STATE_DIR="${STATE_DIR}"

BINARY="${TMPDIR_ROOT}/chainbench-net-test"
( cd "${CHAINBENCH_DIR}/network" && go build -o "${BINARY}" ./cmd/chainbench-net ) >/dev/null 2>&1 || {
  echo "FATAL: build failed" >&2; rm -rf "${TMPDIR_ROOT}"; exit 1
}
export CHAINBENCH_NET_BIN="${BINARY}"

PORT="$(python3 -c 'import socket
s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')"

MOCK_LOG="${TMPDIR_ROOT}/mock.log"
python3 - "$PORT" >"${MOCK_LOG}" 2>&1 <<'PYEOF' &
import sys, json, http.server, socketserver
port = int(sys.argv[1])
class H(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get('Content-Length', 0))
        try: req = json.loads(self.rfile.read(n))
        except: req = {}
        m = req.get('method'); rid = req.get('id')
        if m == 'eth_chainId':
            body = {'jsonrpc':'2.0','id':rid,'result':'0x1'}
        elif m == 'eth_getTransactionReceipt':
            body = {'jsonrpc':'2.0','id':rid,'result':{
                'transactionHash':'0x'+'a'*64,
                'blockHash':'0x'+'b'*64,
                'blockNumber':'0x10',
                'gasUsed':'0x5208',
                'effectiveGasPrice':'0x1',
                'status':'0x1',
                'contractAddress':None,
                'logs':[]}}
        elif m in ('istanbul_getValidators','wemix_getReward'):
            body = {'jsonrpc':'2.0','id':rid,'error':{'code':-32601,'message':'nf'}}
        else:
            body = {'jsonrpc':'2.0','id':rid,'error':{'code':-32601,'message':'nf'}}
        raw = json.dumps(body).encode()
        self.send_response(200)
        self.send_header('Content-Type','application/json')
        self.send_header('Content-Length', str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)
    def log_message(self, *a, **k): pass
with socketserver.TCPServer(('127.0.0.1', port), H) as srv:
    srv.serve_forever()
PYEOF
MOCK_PID=$!

cleanup() {
  kill "${MOCK_PID}" 2>/dev/null || true
  wait "${MOCK_PID}" 2>/dev/null || true
  rm -rf "${TMPDIR_ROOT}"
}
trap cleanup EXIT INT TERM HUP

mock_ready=0
for _ in 1 2 3 4 5 6 7 8 9 10; do
  if (echo > "/dev/tcp/127.0.0.1/${PORT}") 2>/dev/null; then mock_ready=1; break; fi
  sleep 0.1
done
[[ "${mock_ready}" -eq 1 ]] || { echo "FATAL: mock not listening" >&2; cat "${MOCK_LOG}" >&2; exit 1; }

# shellcheck disable=SC1091
source "${CHAINBENCH_DIR}/lib/network_client.sh"

describe "tx_wait: attach mock then poll receipt"
attach_data="$(cb_net_call "network.attach" "{\"rpc_url\":\"http://127.0.0.1:${PORT}\",\"name\":\"wt\"}")"
assert_eq "$(jq -r .name <<<"$attach_data")" "wt" "attach name"

wait_data="$(cb_net_call "node.tx_wait" "{\"network\":\"wt\",\"node_id\":\"node1\",\"tx_hash\":\"0x$(printf 'a%.0s' {1..64})\",\"timeout_ms\":3000}")"
assert_eq "$(jq -r .status <<<"$wait_data")" "success" "status is success"
assert_eq "$(jq -r .block_number <<<"$wait_data")" "16" "block_number = 0x10 (16)"

unit_summary
```

- [ ] **Step 3: Go E2E**

Append to `e2e_test.go` a 1559 happy-path test plus a tx_wait happy-path
test, modeled on the existing `TestE2E_NodeTxSend_AgainstAttachedRemote`:

```go
func TestE2E_NodeTxSend_DynamicFee_AgainstAttachedRemote(t *testing.T) {
    var sentRaw string
    rpcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req struct{ Method string `json:"method"`; ID json.RawMessage `json:"id"`; Params []json.RawMessage `json:"params"` }
        _ = json.NewDecoder(r.Body).Decode(&req)
        w.Header().Set("Content-Type", "application/json")
        switch req.Method {
        case "eth_chainId":
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1"}`, req.ID)
        case "eth_sendRawTransaction":
            if len(req.Params) > 0 {
                _ = json.Unmarshal(req.Params[0], &sentRaw)
            }
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x%s"}`, req.ID, strings.Repeat("c", 64))
        case "istanbul_getValidators", "wemix_getReward":
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
        default:
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
        }
    }))
    defer rpcSrv.Close()

    stateDir := t.TempDir()
    t.Setenv("CHAINBENCH_STATE_DIR", stateDir)
    t.Setenv("CHAINBENCH_SIGNER_ALICE_KEY", "0xb71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

    // attach
    var stdout, stderr bytes.Buffer
    root := newRootCmd()
    attachCmd := fmt.Sprintf(`{"command":"network.attach","args":{"rpc_url":%q,"name":"dyn"}}`, rpcSrv.URL)
    root.SetIn(strings.NewReader(attachCmd))
    root.SetOut(&stdout); root.SetErr(&stderr); root.SetArgs([]string{"run"})
    if err := root.Execute(); err != nil { t.Fatalf("attach: %v stderr=%s", err, stderr.String()) }

    // tx_send (1559)
    stdout.Reset(); stderr.Reset()
    root2 := newRootCmd()
    sendCmd := `{"command":"node.tx_send","args":{"network":"dyn","node_id":"node1","signer":"alice","to":"0x0000000000000000000000000000000000000002","value":"0x0","gas":21000,"max_fee_per_gas":"0x59682f00","max_priority_fee_per_gas":"0x3b9aca00","nonce":0}}`
    root2.SetIn(strings.NewReader(sendCmd))
    root2.SetOut(&stdout); root2.SetErr(&stderr); root2.SetArgs([]string{"run"})
    if err := root2.Execute(); err != nil { t.Fatalf("tx_send: %v stderr=%s", err, stderr.String()) }

    if sentRaw == "" { t.Fatal("mock did not see eth_sendRawTransaction") }
    if got := decodeBroadcastTxType(t, sentRaw); got != 2 {
        t.Errorf("tx type = %d, want 2 (DynamicFee)", got)
    }
}

func TestE2E_NodeTxWait_SuccessImmediate(t *testing.T) {
    rpcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req struct{ Method string `json:"method"`; ID json.RawMessage `json:"id"` }
        _ = json.NewDecoder(r.Body).Decode(&req)
        w.Header().Set("Content-Type", "application/json")
        switch req.Method {
        case "eth_chainId":
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1"}`, req.ID)
        case "eth_getTransactionReceipt":
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{
                "transactionHash":"0x%s","blockHash":"0x%s","blockNumber":"0x20",
                "gasUsed":"0x5208","effectiveGasPrice":"0x1",
                "status":"0x1","contractAddress":null,"logs":[]}}`, req.ID, strings.Repeat("a",64), strings.Repeat("b",64))
        case "istanbul_getValidators", "wemix_getReward":
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
        default:
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
        }
    }))
    defer rpcSrv.Close()

    stateDir := t.TempDir()
    t.Setenv("CHAINBENCH_STATE_DIR", stateDir)

    var stdout, stderr bytes.Buffer
    root := newRootCmd()
    root.SetIn(strings.NewReader(fmt.Sprintf(`{"command":"network.attach","args":{"rpc_url":%q,"name":"wt"}}`, rpcSrv.URL)))
    root.SetOut(&stdout); root.SetErr(&stderr); root.SetArgs([]string{"run"})
    if err := root.Execute(); err != nil { t.Fatalf("attach: %v", err) }

    stdout.Reset(); stderr.Reset()
    root2 := newRootCmd()
    waitCmd := fmt.Sprintf(`{"command":"node.tx_wait","args":{"network":"wt","node_id":"node1","tx_hash":"0x%s","timeout_ms":3000}}`, strings.Repeat("a",64))
    root2.SetIn(strings.NewReader(waitCmd))
    root2.SetOut(&stdout); root2.SetErr(&stderr); root2.SetArgs([]string{"run"})
    if err := root2.Execute(); err != nil { t.Fatalf("tx_wait: %v", err) }

    // Last NDJSON line is the result terminator. Just assert success status.
    lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
    last := lines[len(lines)-1]
    if !strings.Contains(last, `"status":"success"`) {
        t.Errorf("last line missing success: %q", last)
    }
}
```

- [ ] **Step 4: Run all tests**

```bash
cd network && go test ./... -count=1 -timeout=120s
cd /Users/wm-it-22-00661/Work/github/tools/chainbench && bash tests/unit/run.sh
```

- [ ] **Step 5: Commit**

```bash
git add tests/unit/tests/security-key-boundary.sh tests/unit/tests/node-tx-wait.sh \
        network/cmd/chainbench-net/e2e_test.go
git commit -m "test(sprint-4b): keystore boundary + tx_wait flows + 1559 E2E

Three new layers of test coverage for Sprint 4b's surface:

- security-key-boundary.sh gains a keystore variant: a per-run
  generated keystore + password is exercised through node.tx_send
  and stdout/stderr/log are scanned for the underlying raw hex AND
  the password. Both must be absent.

- tests/unit/tests/node-tx-wait.sh exercises the new node.tx_wait
  command end-to-end via the spawn binary with a python mock RPC
  serving an immediate successful receipt. Asserts status=success
  and block_number echoes.

- Two cobra in-process E2E cases: 1559 happy path verifies the
  broadcast tx type byte, and tx_wait happy path verifies the
  result terminator carries status=success on a single-poll receipt."
```

---

## Task 5 — Documentation + roadmap + final review

**Files:**
- Modify: `docs/SECURITY_KEY_HANDLING.md`
- Modify: `docs/EVALUATION_CAPABILITY.md`
- Modify: `docs/VISION_AND_ROADMAP.md`
- Modify: `docs/NEXT_WORK.md`

- [ ] **Step 1: SECURITY_KEY_HANDLING.md**

- Move "Key Injection (planned — Sprint 4b)" into the "Key Injection
  (current)" section, retitled as "Key Injection (current — env-key OR
  keystore)".
- Document resolution order: raw `_KEY` env wins over `_KEYSTORE`.
- Add an "Operator Notes" bullet on keystore file permissions (0600 expected;
  chainbench-net does NOT enforce — informational only).
- Update the "Boundary Enforcement" section to call out the keystore variant
  in `security-key-boundary.sh`.
- Bump the "Last updated" date to 2026-04-27 (Sprint 4b).

- [ ] **Step 2: EVALUATION_CAPABILITY.md**

- §2 Tx matrix:
  - `EIP-1559 (0x2)` → Go column from `🚧 Sprint 4b 계획` to `✅ Sprint 4b`.
  - `Receipt polling (status+logs)` → Go column from `🚧 Sprint 4b` to `✅ Sprint 4b`.
- §5 Sprint roadmap: tick 4b row.
- §6 coverage line: bump `Go network/` percentage upward (subjective; use the
  ratio of green Go cells to total).

- [ ] **Step 3: VISION_AND_ROADMAP.md**

- Tick the Sprint 4b checkbox under §6 with date 2026-04-27.

- [ ] **Step 4: NEXT_WORK.md**

- §3 Priority 2 (Sprint 4b) — mark resolved with the four commits from this
  sprint.
- §3.4 — drop any rows that are now closed.
- Bump `최종 업데이트` line where it appears.

- [ ] **Step 5: Full matrix**

```bash
cd network && go test ./... -count=1 -timeout=120s
cd /Users/wm-it-22-00661/Work/github/tools/chainbench && bash tests/unit/run.sh
```

Both must pass — green.

- [ ] **Step 6: Commit**

```bash
git add docs/
git commit -m "docs: mark Sprint 4b complete (keystore + EIP-1559 + tx.wait)

SECURITY_KEY_HANDLING promotes the keystore section from 'planned' to
'current', records the raw-key-wins-over-keystore resolution rule,
and points at the extended boundary test.

EVALUATION_CAPABILITY ticks the Sprint 4b cells in the tx and verify
matrices and bumps the Go-surface coverage line.

VISION_AND_ROADMAP and NEXT_WORK absorb the four commits from this
sprint and drop the resolved rows."
```

- [ ] **Step 7: Report**

Commit range, package-level test counts, bash test counts, capability
matrix delta, deferrals confirmed (4c fee delegation / 4d contract / 5
MCP).
