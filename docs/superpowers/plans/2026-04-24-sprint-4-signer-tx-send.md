# Sprint 4 — Signer Boundary + tx.send Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Ship `network/internal/signer` (env-only), wire `node.tx_send` handler through it with existing `remote.Client` extended for nonce + gas + broadcast. Lock the key-redaction contract with unit + bash boundary tests. Ship `docs/SECURITY_KEY_HANDLING.md`.

**Architecture:** New `signer` package exposes `Alias`, `Signer` interface, `Load(alias)` factory; sealed struct implements `slog.LogValuer` returning `"***"`. Handler calls `signer.Load` per-request → `dialNode` (reuses 3b.2c) → auto-fills missing nonce/gas/gas_price via `remote.Client` → `SignTx` → `SendTransaction`. Security boundary test spawns the binary end-to-end and greps stdout/stderr/log for key leakage.

**Tech Stack:** Go 1.25, existing go-ethereum deps (crypto, core/types, ethclient), stdlib `log/slog`, new env plumbing.

Spec: `docs/superpowers/specs/2026-04-24-sprint-4-signer-tx-send.md`

---

## Commit Discipline

- English, no Co-Authored-By, no "Generated with Claude Code", no emoji.
- Key material must never appear in test assertion strings, error messages, or log lines — this is enforced at review time.

## File Structure

**Create:**
- `network/internal/signer/signer.go`
- `network/internal/signer/signer_test.go`
- `tests/unit/tests/security-key-boundary.sh`
- `docs/SECURITY_KEY_HANDLING.md`

**Modify:**
- `network/internal/drivers/remote/client.go` — add PendingNonceAt, EstimateGas, SendTransaction
- `network/internal/drivers/remote/client_test.go` — tests for new methods
- `network/schema/command.json` — add `"node.tx_send"`
- `network/internal/types/command_gen.go` — regenerated
- `network/cmd/chainbench-net/handlers.go` — newHandleNodeTxSend + registration
- `network/cmd/chainbench-net/handlers_test.go` — unit tests
- `network/cmd/chainbench-net/e2e_test.go` — Go E2E
- `docs/VISION_AND_ROADMAP.md` — mark Sprint 4 complete

---

## Task 1 — signer package (env-only)

**Files:**
- Create: `network/internal/signer/signer.go`
- Create: `network/internal/signer/signer_test.go`

- [ ] **Step 1: Write failing tests (key redaction + load + sign)**

```go
// network/internal/signer/signer_test.go
package signer_test

import (
    "bytes"
    "context"
    "errors"
    "fmt"
    "log/slog"
    "math/big"
    "strings"
    "testing"

    "github.com/ethereum/go-ethereum/core/types"

    "github.com/0xmhha/chainbench/network/internal/signer"
)

// keyHex64 is a deterministic test key. It's synthetic — not associated with
// any real funds — but tests treat it as secret and assert it never appears
// in any observable output.
const keyHex64 = "b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291"
// addrFor keyHex64, precomputed: 0x0ABA0CEC0f3e11e4A10b1F7E5f1cAb5F66DfB4e5

// withSignerEnv sets the env var for alias and restores on cleanup.
func withSignerEnv(t *testing.T, alias, value string) {
    t.Helper()
    t.Setenv("CHAINBENCH_SIGNER_"+strings.ToUpper(alias)+"_KEY", value)
}

func TestLoad_HappyPath(t *testing.T) {
    withSignerEnv(t, "alice", "0x"+keyHex64)
    s, err := signer.Load("alice")
    if err != nil {
        t.Fatalf("Load: %v", err)
    }
    if s.Address().Hex() == "0x0000000000000000000000000000000000000000" {
        t.Error("address is zero; key probably not ingested")
    }
}

func TestLoad_NoPrefix(t *testing.T) {
    withSignerEnv(t, "bob", keyHex64) // no 0x prefix
    if _, err := signer.Load("bob"); err != nil {
        t.Errorf("Load without 0x prefix should succeed: %v", err)
    }
}

func TestLoad_MissingEnv(t *testing.T) {
    _, err := signer.Load("ghost")
    if !errors.Is(err, signer.ErrUnknownAlias) {
        t.Errorf("err = %v, want ErrUnknownAlias", err)
    }
}

func TestLoad_InvalidKey(t *testing.T) {
    withSignerEnv(t, "bad", "0xnot-hex")
    _, err := signer.Load("bad")
    if !errors.Is(err, signer.ErrInvalidKey) {
        t.Errorf("err = %v, want ErrInvalidKey", err)
    }
    // Error must not echo the invalid value.
    if strings.Contains(err.Error(), "not-hex") {
        t.Errorf("err leaks env value: %v", err)
    }
}

func TestLoad_BadAlias(t *testing.T) {
    cases := []string{"", "has space", "has/slash", "../traverse"}
    for _, name := range cases {
        t.Run(name, func(t *testing.T) {
            _, err := signer.Load(signer.Alias(name))
            if !errors.Is(err, signer.ErrInvalidAlias) {
                t.Errorf("err = %v, want ErrInvalidAlias", err)
            }
        })
    }
}

func TestSigner_LogValueRedacts(t *testing.T) {
    withSignerEnv(t, "carol", "0x"+keyHex64)
    s, err := signer.Load("carol")
    if err != nil {
        t.Fatal(err)
    }

    var buf bytes.Buffer
    logger := slog.New(slog.NewTextHandler(&buf, nil))
    logger.Info("signer load", "signer", s)

    out := buf.String()
    if !strings.Contains(out, "***") {
        t.Errorf("log should contain redaction marker ***: %q", out)
    }
    if strings.Contains(out, keyHex64) {
        t.Errorf("log leaks key material: %q", out)
    }
}

func TestSigner_SprintfRedacts(t *testing.T) {
    withSignerEnv(t, "dave", "0x"+keyHex64)
    s, err := signer.Load("dave")
    if err != nil {
        t.Fatal(err)
    }
    for _, format := range []string{"%v", "%+v", "%#v", "%s"} {
        t.Run(format, func(t *testing.T) {
            out := fmt.Sprintf(format, s)
            if strings.Contains(out, keyHex64) {
                t.Errorf("fmt %s leaks key: %q", format, out)
            }
        })
    }
}

func TestSigner_SignTx_Roundtrip(t *testing.T) {
    withSignerEnv(t, "eve", "0x"+keyHex64)
    s, err := signer.Load("eve")
    if err != nil {
        t.Fatal(err)
    }

    chainID := big.NewInt(1)
    tx := types.NewTx(&types.LegacyTx{
        Nonce:    0,
        GasPrice: big.NewInt(1),
        Gas:      21000,
        To:       nil,
        Value:    big.NewInt(0),
        Data:     nil,
    })
    signed, err := s.SignTx(context.Background(), tx, chainID)
    if err != nil {
        t.Fatalf("SignTx: %v", err)
    }
    sender, err := types.Sender(types.LatestSignerForChainID(chainID), signed)
    if err != nil {
        t.Fatalf("recover sender: %v", err)
    }
    if sender != s.Address() {
        t.Errorf("recovered sender %s != signer.Address() %s", sender.Hex(), s.Address().Hex())
    }
}
```

- [ ] **Step 2: Verify RED**

```bash
cd network && go test ./internal/signer/... -v
```

Expected: compile failure — `undefined: signer.Load`, `ErrUnknownAlias`, etc.

- [ ] **Step 3: Implement signer.go**

```go
// Package signer is the chainbench-net signing boundary. Private key material
// lives ONLY inside sealed structs of this package; there is no accessor or
// reflection-visible export.
//
// Key material enters the process exclusively via env vars at spawn time:
//
//     CHAINBENCH_SIGNER_<ALIAS>_KEY=0x<64-hex-chars>
//
// Load() resolves an alias by uppercasing it and reading the matching env var.
// The returned Signer exposes Address() and SignTx() — no Export(), no
// GetKey(), no serialization path. slog.LogValuer on the sealed struct
// renders "***" to prevent accidental disclosure through structured logging.
package signer

import (
    "context"
    "crypto/ecdsa"
    "errors"
    "fmt"
    "log/slog"
    "math/big"
    "os"
    "regexp"
    "strings"

    "github.com/ethereum/go-ethereum/common"
    "github.com/ethereum/go-ethereum/core/types"
    "github.com/ethereum/go-ethereum/crypto"
)

type Alias string

type Signer interface {
    Address() common.Address
    SignTx(ctx context.Context, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error)
}

var (
    ErrUnknownAlias = errors.New("signer: unknown alias")
    ErrInvalidAlias = errors.New("signer: alias must be [A-Za-z0-9_-]+")
    ErrInvalidKey   = errors.New("signer: key material is not a valid hex private key")
)

var aliasRE = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// sealed holds the private key material. Field is unexported; no method
// returns the key bytes. LogValue redacts for any slog attr consumer.
type sealed struct {
    alias Alias
    addr  common.Address
    key   *ecdsa.PrivateKey
}

func (s *sealed) Address() common.Address { return s.addr }

func (s *sealed) SignTx(_ context.Context, tx *types.Transaction, chainID *big.Int) (*types.Transaction, error) {
    if tx == nil {
        return nil, fmt.Errorf("signer.SignTx: tx is nil")
    }
    if chainID == nil {
        return nil, fmt.Errorf("signer.SignTx: chainID is nil")
    }
    signed, err := types.SignTx(tx, types.LatestSignerForChainID(chainID), s.key)
    if err != nil {
        // err from go-ethereum is not expected to carry the key; defensive
        // wrap references the alias only.
        return nil, fmt.Errorf("signer.SignTx(%s): %w", s.alias, err)
    }
    return signed, nil
}

// LogValue implements slog.LogValuer so any structured-log attr containing a
// Signer renders as "***" instead of the underlying struct.
func (*sealed) LogValue() slog.Value { return slog.StringValue("***") }

// String / GoString implement the fmt.Stringer and fmt.GoStringer contracts
// so %v, %+v, %#v, %s all render as the redacted placeholder.
func (*sealed) String() string   { return "<signer:***>" }
func (*sealed) GoString() string { return "<signer:***>" }

// Load resolves alias → Signer via env var. Returns ErrInvalidAlias for
// structurally bad names, ErrUnknownAlias when no env var is set,
// ErrInvalidKey when the env value is not a valid hex private key.
func Load(alias Alias) (Signer, error) {
    a := string(alias)
    if a == "" || !aliasRE.MatchString(a) {
        return nil, fmt.Errorf("%w: %q", ErrInvalidAlias, a)
    }
    envName := "CHAINBENCH_SIGNER_" + strings.ToUpper(a) + "_KEY"
    raw := os.Getenv(envName)
    if raw == "" {
        return nil, fmt.Errorf("%w: %s (env %s not set)", ErrUnknownAlias, a, envName)
    }
    hex := strings.TrimPrefix(raw, "0x")
    // Validate without ever embedding the raw value in an error message.
    key, err := crypto.HexToECDSA(hex)
    if err != nil {
        return nil, fmt.Errorf("%w: alias=%s (env %s)", ErrInvalidKey, a, envName)
    }
    addr := crypto.PubkeyToAddress(key.PublicKey)
    return &sealed{alias: alias, addr: addr, key: key}, nil
}
```

- [ ] **Step 4: Run tests, iterate**

```bash
cd network && go test ./internal/signer/... -v -count=1
```

Expected: all PASS.

- [ ] **Step 5: Full suite + lint**

```bash
cd network && go test ./... -count=1 -timeout=60s
cd network && go vet ./...
gofmt -l network/
```

- [ ] **Step 6: Commit**

```bash
git add network/internal/signer/
git commit -m "feat(signer): env-based signing boundary with key redaction

New network/internal/signer package. Exposes Alias, Signer interface
(Address + SignTx only), Load(alias) factory. Key material is sourced
from CHAINBENCH_SIGNER_<ALIAS>_KEY env vars at request time; the
sealed struct has no accessor and implements slog.LogValuer + Stringer
+ GoStringer so %v / %+v / %#v / %s / structured logging all render as
<signer:***> or \"***\".

Error messages reference the alias and env var name only — never the
key value. Bad input surfaces as ErrInvalidAlias, ErrUnknownAlias,
or ErrInvalidKey sentinels. Keystore provider + EIP-1559 fields are
deferred to Sprint 4b per spec."
```

---

## Task 2 — remote.Client additions (PendingNonceAt, EstimateGas, SendTransaction)

**Files:**
- Modify: `network/internal/drivers/remote/client.go`
- Modify: `network/internal/drivers/remote/client_test.go`

- [ ] **Step 1: Write failing tests**

Append to `client_test.go`:

```go
func TestClient_PendingNonceAt(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req struct{ Method string; ID json.RawMessage }
        _ = json.NewDecoder(r.Body).Decode(&req)
        w.Header().Set("Content-Type", "application/json")
        if req.Method == "eth_getTransactionCount" {
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x7"}`, req.ID)
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

    n, err := c.PendingNonceAt(ctx, common.HexToAddress("0x01"))
    if err != nil { t.Fatalf("PendingNonceAt: %v", err) }
    if n != 7 { t.Errorf("nonce = %d, want 7", n) }
}

func TestClient_EstimateGas(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req struct{ Method string; ID json.RawMessage }
        _ = json.NewDecoder(r.Body).Decode(&req)
        w.Header().Set("Content-Type", "application/json")
        if req.Method == "eth_estimateGas" {
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x5208"}`, req.ID) // 21000
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
    from := common.HexToAddress("0x01")
    to := common.HexToAddress("0x02")
    gas, err := c.EstimateGas(ctx, ethereum.CallMsg{From: from, To: &to})
    if err != nil { t.Fatalf("EstimateGas: %v", err) }
    if gas != 21000 { t.Errorf("gas = %d, want 21000", gas) }
}

func TestClient_SendTransaction(t *testing.T) {
    var receivedHex string
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req struct {
            Method string
            ID     json.RawMessage
            Params []json.RawMessage
        }
        _ = json.NewDecoder(r.Body).Decode(&req)
        w.Header().Set("Content-Type", "application/json")
        if req.Method == "eth_sendRawTransaction" {
            // Param[0] is the signed raw tx hex
            if len(req.Params) > 0 {
                _ = json.Unmarshal(req.Params[0], &receivedHex)
            }
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0xabc123"}`, req.ID)
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

    // Build + send a minimal unsigned tx (signature zeroes — server just echoes back).
    tx := types.NewTx(&types.LegacyTx{Nonce: 0, GasPrice: big.NewInt(1), Gas: 21000})
    if err := c.SendTransaction(ctx, tx); err != nil {
        t.Fatalf("SendTransaction: %v", err)
    }
    if receivedHex == "" {
        t.Error("server did not receive a signed tx param")
    }
}
```

Add imports: `"math/big"`, `"github.com/ethereum/go-ethereum"`, `"github.com/ethereum/go-ethereum/core/types"` (common already imported).

- [ ] **Step 2: Implement in client.go**

```go
// Add import: "github.com/ethereum/go-ethereum", "github.com/ethereum/go-ethereum/core/types"

// PendingNonceAt returns the next available nonce for account (pending + mined).
func (c *Client) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
    n, err := c.rpc.PendingNonceAt(ctx, account)
    if err != nil {
        return 0, fmt.Errorf("remote.PendingNonceAt: %w", err)
    }
    return n, nil
}

// EstimateGas asks the endpoint for the gas required to execute msg.
func (c *Client) EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error) {
    g, err := c.rpc.EstimateGas(ctx, msg)
    if err != nil {
        return 0, fmt.Errorf("remote.EstimateGas: %w", err)
    }
    return g, nil
}

// SendTransaction broadcasts a signed transaction. Caller constructs + signs
// tx outside this package; remote.Client only forwards the RLP.
func (c *Client) SendTransaction(ctx context.Context, tx *types.Transaction) error {
    if err := c.rpc.SendTransaction(ctx, tx); err != nil {
        return fmt.Errorf("remote.SendTransaction: %w", err)
    }
    return nil
}
```

- [ ] **Step 3: Run tests**

```bash
cd network && go test ./internal/drivers/remote/... -v -count=1 -timeout=30s
```

- [ ] **Step 4: Full suite + commit**

```bash
cd network && go test ./... -count=1 -timeout=60s
git add network/internal/drivers/remote/
git commit -m "feat(drivers/remote): add PendingNonceAt + EstimateGas + SendTransaction

Three new read/write wrappers over ethclient counterparts, same
remote.<Method> error-wrapping convention. PendingNonceAt and
EstimateGas support tx.send's auto-fill path when the caller omits
nonce or gas; SendTransaction broadcasts the signed RLP returned by
the signer."
```

---

## Task 3 — node.tx_send handler + schema

**Files:**
- Modify: `network/schema/command.json` — add `"node.tx_send"`
- Modify: `network/internal/types/command_gen.go` — regenerated
- Modify: `network/cmd/chainbench-net/handlers.go`
- Modify: `network/cmd/chainbench-net/handlers_test.go`

- [ ] **Step 1: Schema**

Add `"node.tx_send"` to `command.json` enum.

- [ ] **Step 2: Regenerate**

```bash
cd network && go generate ./...
```

- [ ] **Step 3: Write failing handler tests**

Append to `handlers_test.go`:

```go
// TestHandleNodeTxSend_Happy exercises the full flow with explicit nonce/gas
// to keep the test hermetic (no EstimateGas / PendingNonceAt round-trips).
func TestHandleNodeTxSend_Happy(t *testing.T) {
    // Mock records the raw signed tx hex so we can verify a broadcast happened.
    var sawSendRaw bool
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
            sawSendRaw = true
            // Arbitrary 32-byte hash echo.
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1111111111111111111111111111111111111111111111111111111111111111"}`, req.ID)
        default:
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
        }
    }))
    defer srv.Close()

    stateDir := t.TempDir()
    saveRemoteFixture(t, stateDir, "tn", srv.URL)
    // Well-known synthetic key (NOT real funds) — test-only.
    t.Setenv("CHAINBENCH_SIGNER_ALICE_KEY", "0xb71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")

    h := newHandleNodeTxSend(stateDir)
    args, _ := json.Marshal(map[string]any{
        "network":   "tn",
        "node_id":   "node1",
        "signer":    "alice",
        "to":        "0x0000000000000000000000000000000000000002",
        "value":     "0x0",
        "gas":       21000,
        "gas_price": "0x1",
        "nonce":     0,
    })
    bus, _ := newTestBus(t)
    defer bus.Close()

    data, err := h(args, bus)
    if err != nil {
        t.Fatalf("handler: %v", err)
    }
    if !sawSendRaw {
        t.Error("mock did not see eth_sendRawTransaction")
    }
    if tx, _ := data["tx_hash"].(string); !strings.HasPrefix(tx, "0x") || len(tx) != 66 {
        t.Errorf("tx_hash shape wrong: %v", data["tx_hash"])
    }
}

func TestHandleNodeTxSend_MissingSigner(t *testing.T) {
    h := newHandleNodeTxSend(t.TempDir())
    args, _ := json.Marshal(map[string]any{"node_id": "node1", "to": "0x02"})
    bus, _ := newTestBus(t)
    defer bus.Close()
    _, err := h(args, bus)
    var api *APIError
    if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
        t.Errorf("want INVALID_ARGS, got %v", err)
    }
}

func TestHandleNodeTxSend_MissingTo(t *testing.T) {
    h := newHandleNodeTxSend(t.TempDir())
    args, _ := json.Marshal(map[string]any{"node_id": "node1", "signer": "alice"})
    bus, _ := newTestBus(t)
    defer bus.Close()
    _, err := h(args, bus)
    var api *APIError
    if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
        t.Errorf("want INVALID_ARGS, got %v", err)
    }
}

func TestHandleNodeTxSend_UnknownSigner(t *testing.T) {
    srv := newReadMockRPC(t); defer srv.Close()
    stateDir := t.TempDir()
    saveRemoteFixture(t, stateDir, "tn", srv.URL)
    os.Unsetenv("CHAINBENCH_SIGNER_GHOST_KEY")

    h := newHandleNodeTxSend(stateDir)
    args, _ := json.Marshal(map[string]any{
        "network": "tn",
        "node_id": "node1",
        "signer":  "ghost",
        "to":      "0x0000000000000000000000000000000000000002",
    })
    bus, _ := newTestBus(t)
    defer bus.Close()
    _, err := h(args, bus)
    var api *APIError
    if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
        t.Errorf("want INVALID_ARGS, got %v", err)
    }
}

func TestHandleNodeTxSend_BadToAddress(t *testing.T) {
    h := newHandleNodeTxSend(t.TempDir())
    args, _ := json.Marshal(map[string]any{
        "network": "tn",
        "node_id": "node1",
        "signer":  "alice",
        "to":      "not-a-hex-address",
    })
    bus, _ := newTestBus(t)
    defer bus.Close()
    _, err := h(args, bus)
    var api *APIError
    if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
        t.Errorf("want INVALID_ARGS, got %v", err)
    }
}

func TestAllHandlers_IncludesTxSend(t *testing.T) {
    h := allHandlers("x", "y")
    if _, ok := h["node.tx_send"]; !ok {
        t.Error("allHandlers missing node.tx_send")
    }
}
```

- [ ] **Step 4: Implement newHandleNodeTxSend**

Add imports to handlers.go: `"github.com/ethereum/go-ethereum/core/types"`, `"github.com/0xmhha/chainbench/network/internal/signer"`. (big, common already there.)

```go
// newHandleNodeTxSend signs + broadcasts a legacy-format transaction against
// the resolved node. Missing nonce/gas/gas_price are auto-filled via the
// remote.Client. Signer alias is resolved per-request via signer.Load.
//
// Args: {network?, node_id, signer, to, value?, data?, gas?, gas_price?, nonce?}
// Result: {tx_hash: "0x..."}
func newHandleNodeTxSend(stateDir string) Handler {
    return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
        var req struct {
            Network  string `json:"network"`
            NodeID   string `json:"node_id"`
            Signer   string `json:"signer"`
            To       string `json:"to"`
            Value    string `json:"value"`
            Data     string `json:"data"`
            Gas      *uint64 `json:"gas"`
            GasPrice string `json:"gas_price"`
            Nonce    *uint64 `json:"nonce"`
        }
        if len(args) > 0 {
            if err := json.Unmarshal(args, &req); err != nil {
                return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
            }
        }
        if req.Signer == "" {
            return nil, NewInvalidArgs("args.signer is required")
        }
        if req.To == "" {
            return nil, NewInvalidArgs("args.to is required")
        }
        if !common.IsHexAddress(req.To) {
            return nil, NewInvalidArgs(fmt.Sprintf("args.to is not a valid hex address: %q", req.To))
        }
        to := common.HexToAddress(req.To)

        // Parse optional hex fields.
        value := big.NewInt(0)
        if req.Value != "" {
            parsed, ok := new(big.Int).SetString(strings.TrimPrefix(req.Value, "0x"), 16)
            if !ok {
                return nil, NewInvalidArgs(fmt.Sprintf("args.value is not valid hex: %q", req.Value))
            }
            value = parsed
        }
        var data []byte
        if req.Data != "" && req.Data != "0x" {
            trimmed := strings.TrimPrefix(req.Data, "0x")
            decoded, err := hex.DecodeString(trimmed)
            if err != nil {
                return nil, NewInvalidArgs(fmt.Sprintf("args.data is not valid hex: %q", req.Data))
            }
            data = decoded
        }

        s, serr := signer.Load(signer.Alias(req.Signer))
        if serr != nil {
            if errors.Is(serr, signer.ErrInvalidAlias) ||
                errors.Is(serr, signer.ErrUnknownAlias) {
                return nil, NewInvalidArgs(serr.Error())
            }
            return nil, NewUpstream("signer load", serr)
        }

        _, node, err := resolveNode(stateDir, req.Network, req.NodeID)
        if err != nil {
            return nil, err
        }

        ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
        defer cancel()
        client, err := dialNode(ctx, &node)
        if err != nil {
            return nil, err
        }
        defer client.Close()

        chainID, err := client.ChainID(ctx)
        if err != nil {
            return nil, NewUpstream("eth_chainId", err)
        }

        nonce, err := resolveNonce(ctx, client, s.Address(), req.Nonce)
        if err != nil {
            return nil, err
        }
        gasPrice, err := resolveGasPrice(ctx, client, req.GasPrice)
        if err != nil {
            return nil, err
        }
        gas, err := resolveGas(ctx, client, req.Gas, s.Address(), to, value, data)
        if err != nil {
            return nil, err
        }

        unsigned := types.NewTx(&types.LegacyTx{
            Nonce:    nonce,
            GasPrice: gasPrice,
            Gas:      gas,
            To:       &to,
            Value:    value,
            Data:     data,
        })
        signed, err := s.SignTx(ctx, unsigned, chainID)
        if err != nil {
            return nil, NewInternal("sign tx", err)
        }
        if err := client.SendTransaction(ctx, signed); err != nil {
            return nil, NewUpstream("eth_sendRawTransaction", err)
        }
        return map[string]any{"tx_hash": signed.Hash().Hex()}, nil
    }
}

// --- helpers (paste near dialNode) ---

func resolveNonce(ctx context.Context, client *remote.Client, from common.Address, explicit *uint64) (uint64, error) {
    if explicit != nil {
        return *explicit, nil
    }
    n, err := client.PendingNonceAt(ctx, from)
    if err != nil {
        return 0, NewUpstream("eth_getTransactionCount", err)
    }
    return n, nil
}

func resolveGasPrice(ctx context.Context, client *remote.Client, explicit string) (*big.Int, error) {
    if explicit != "" {
        v, ok := new(big.Int).SetString(strings.TrimPrefix(explicit, "0x"), 16)
        if !ok {
            return nil, NewInvalidArgs(fmt.Sprintf("args.gas_price is not valid hex: %q", explicit))
        }
        return v, nil
    }
    gp, err := client.GasPrice(ctx)
    if err != nil {
        return nil, NewUpstream("eth_gasPrice", err)
    }
    return gp, nil
}

func resolveGas(ctx context.Context, client *remote.Client, explicit *uint64, from, to common.Address, value *big.Int, data []byte) (uint64, error) {
    if explicit != nil {
        return *explicit, nil
    }
    msg := ethereum.CallMsg{From: from, To: &to, Value: value, Data: data}
    g, err := client.EstimateGas(ctx, msg)
    if err != nil {
        return 0, NewUpstream("eth_estimateGas", err)
    }
    return g, nil
}
```

Add `"encoding/hex"` import + `ethereum "github.com/ethereum/go-ethereum"` if not already present.

- [ ] **Step 5: Register handler**

In `allHandlers`:
```go
"node.tx_send": newHandleNodeTxSend(stateDir),
```

- [ ] **Step 6: Run tests**

```bash
cd network && go test ./cmd/chainbench-net/... -v -run 'TestHandleNodeTxSend|TestAllHandlers_IncludesTxSend' -count=1 -timeout=30s
cd network && go test ./... -count=1 -timeout=60s
go -C network vet ./...
gofmt -l network/
```

- [ ] **Step 7: Commit**

```bash
git add network/schema/ network/internal/types/command_gen.go network/cmd/chainbench-net/handlers.go network/cmd/chainbench-net/handlers_test.go
git commit -m "feat(network-net): node.tx_send signing + broadcast

Composes signer.Load + dialNode + remote.Client.{PendingNonceAt,
EstimateGas,SendTransaction} into a single command that signs a
legacy-format tx with the aliased private key from env and broadcasts
through the attached network.

Missing nonce / gas / gas_price are auto-filled via RPC; callers can
pass explicit values to skip the fetches. chainId comes from the
endpoint (no profile overrides at this layer). Errors surface per the
spec matrix (INVALID_ARGS for structural issues including unknown
signer / bad address, UPSTREAM_ERROR for node interactions, INTERNAL
for sign failures)."
```

---

## Task 4 — Go E2E + bash security boundary test

**Files:**
- Modify: `network/cmd/chainbench-net/e2e_test.go`
- Create: `tests/unit/tests/security-key-boundary.sh`

- [ ] **Step 1: Go E2E**

Append:

```go
func TestE2E_NodeTxSend_AgainstAttachedRemote(t *testing.T) {
    var sawSendRaw bool
    rpcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req struct{ Method string `json:"method"`; ID json.RawMessage `json:"id"` }
        _ = json.NewDecoder(r.Body).Decode(&req)
        w.Header().Set("Content-Type", "application/json")
        switch req.Method {
        case "eth_chainId":
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1"}`, req.ID)
        case "eth_sendRawTransaction":
            sawSendRaw = true
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x2222222222222222222222222222222222222222222222222222222222222222"}`, req.ID)
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
    attachCmd := fmt.Sprintf(`{"command":"network.attach","args":{"rpc_url":%q,"name":"txs-e2e"}}`, rpcSrv.URL)
    root.SetIn(strings.NewReader(attachCmd))
    root.SetOut(&stdout); root.SetErr(&stderr); root.SetArgs([]string{"run"})
    if err := root.Execute(); err != nil {
        t.Fatalf("attach: %v stderr=%s", err, stderr.String())
    }

    // tx_send
    stdout.Reset(); stderr.Reset()
    root2 := newRootCmd()
    sendCmd := `{"command":"node.tx_send","args":{"network":"txs-e2e","node_id":"node1","signer":"alice","to":"0x0000000000000000000000000000000000000002","value":"0x0","gas":21000,"gas_price":"0x1","nonce":0}}`
    root2.SetIn(strings.NewReader(sendCmd))
    root2.SetOut(&stdout); root2.SetErr(&stderr); root2.SetArgs([]string{"run"})
    if err := root2.Execute(); err != nil {
        t.Fatalf("tx_send: %v stderr=%s", err, stderr.String())
    }

    // Verify broadcast happened + result is ok + stderr did not contain key.
    if !sawSendRaw {
        t.Error("mock did not see eth_sendRawTransaction")
    }
    // Key-material leak guard at the E2E layer.
    for _, label := range []string{"stdout", "stderr"} {
        var buf *bytes.Buffer
        if label == "stdout" { buf = &stdout } else { buf = &stderr }
        if strings.Contains(buf.String(), "b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291") {
            t.Errorf("%s leaks key material", label)
        }
    }
}
```

- [ ] **Step 2: Bash security-key-boundary test**

Use `tests/unit/tests/node-block-number.sh` as hardening template (trap EXIT INT TERM HUP, readiness poll, ephemeral port, set -euo pipefail, bash 3.2 safe). Mock must handle eth_chainId + eth_sendRawTransaction:

```bash
#!/usr/bin/env bash
# tests/unit/tests/security-key-boundary.sh
# Spawns chainbench-net, feeds a tx.send command with a known private key
# via env, and asserts that the key never appears in stdout / stderr /
# log output — this is the S4 security boundary from VISION §5.17.5.
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

# Known test-only key (synthetic — not a real funded account).
KEY_HEX="b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291"
export CHAINBENCH_SIGNER_ALICE_KEY="0x${KEY_HEX}"

PORT="$(python3 -c 'import socket
s=socket.socket()
s.bind(("127.0.0.1",0))
print(s.getsockname()[1])
s.close()')"

MOCK_LOG="${TMPDIR_ROOT}/mock.log"
python3 - "$PORT" >"${MOCK_LOG}" 2>&1 <<'PYEOF' &
import sys, json, http.server, socketserver
port = int(sys.argv[1])
class H(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get('Content-Length', 0))
        req = {}
        try: req = json.loads(self.rfile.read(n))
        except: pass
        m = req.get('method'); rid = req.get('id')
        if m == 'eth_chainId':
            body = {'jsonrpc':'2.0','id':rid,'result':'0x1'}
        elif m == 'eth_sendRawTransaction':
            body = {'jsonrpc':'2.0','id':rid,'result':'0x' + 'a'*64}
        elif m in ('istanbul_getValidators', 'wemix_getReward'):
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

# Readiness poll
mock_ready=0
for _ in 1 2 3 4 5 6 7 8 9 10; do
  if (echo > "/dev/tcp/127.0.0.1/${PORT}") 2>/dev/null; then mock_ready=1; break; fi
  sleep 0.1
done
[[ "${mock_ready}" -eq 1 ]] || { echo "FATAL: mock not listening" >&2; cat "${MOCK_LOG}" >&2; exit 1; }

# shellcheck disable=SC1091
source "${CHAINBENCH_DIR}/lib/network_client.sh"

# attach the mock
describe "security: attach sets up test network"
attach_data="$(cb_net_call "network.attach" "{\"rpc_url\":\"http://127.0.0.1:${PORT}\",\"name\":\"sec\"}")"
assert_eq "$(jq -r .name <<<"$attach_data")" "sec" "attach name"

# Capture stdout + stderr of tx.send separately
describe "security: tx.send does not leak key material"
TX_LOG="${TMPDIR_ROOT}/tx.log"
TX_ERR="${TMPDIR_ROOT}/tx.err"
export CHAINBENCH_NET_LOG="${TX_LOG}"
tx_data="$(cb_net_call "node.tx_send" "{\"network\":\"sec\",\"node_id\":\"node1\",\"signer\":\"alice\",\"to\":\"0x0000000000000000000000000000000000000002\",\"value\":\"0x0\",\"gas\":21000,\"gas_price\":\"0x1\",\"nonce\":0}" 2>"${TX_ERR}")"
assert_eq "$(jq -r .tx_hash <<<"$tx_data" | cut -c1-2)" "0x" "tx_hash starts with 0x"

# The boundary assertion: scan stdout, stderr, and log file for the raw key.
for label in "stdout" "stderr" "log"; do
  case "$label" in
    stdout) content="$tx_data" ;;
    stderr) content="$(cat "${TX_ERR}" 2>/dev/null || true)" ;;
    log)    content="$(cat "${TX_LOG}" 2>/dev/null || true)" ;;
  esac
  if echo "$content" | grep -q "${KEY_HEX}"; then
    echo "FAIL: ${label} leaks raw key material" >&2
    exit 1
  fi
  # Also check without 0x prefix (paranoia).
  if echo "$content" | grep -qi "${KEY_HEX}"; then
    echo "FAIL: ${label} leaks key (case-insensitive match)" >&2
    exit 1
  fi
done
assert_eq "leak-checked" "leak-checked" "no key material in stdout/stderr/log"

unit_summary
```

- [ ] **Step 3: Run tests**

```bash
cd network && go test ./cmd/chainbench-net/... -run TestE2E_NodeTxSend -v -timeout=60s
cd network && go test ./... -count=1 -timeout=60s
bash tests/unit/tests/security-key-boundary.sh
bash tests/unit/run.sh  # must remain 100%
```

- [ ] **Step 4: Commit**

```bash
git add network/cmd/chainbench-net/e2e_test.go tests/unit/tests/security-key-boundary.sh
git commit -m "test: E2E tx.send + bash security-key-boundary assertion

Go E2E exercises attach -> tx.send against a mock RPC, verifies the
broadcast hit the endpoint and that the test-only private key does
not appear in stdout/stderr.

Bash test spawns the binary end-to-end with CHAINBENCH_SIGNER_ALICE_KEY
set, performs a tx.send, and greps stdout + stderr + log file for the
raw key. The test fails loudly (exit 1 with rationale) if any of the
three streams leak the key material — this locks the S4 boundary from
VISION §5.17.5."
```

---

## Task 5 — SECURITY doc + roadmap + final review

**Files:**
- Create: `docs/SECURITY_KEY_HANDLING.md`
- Modify: `docs/VISION_AND_ROADMAP.md`

- [ ] **Step 1: Write SECURITY_KEY_HANDLING.md**

```markdown
# Chainbench Key Handling Security Policy

> Last updated: 2026-04-24 (Sprint 4 env-signer)

## Threat Model

Chainbench-net signs transactions locally with private keys supplied by the
operator's deployment environment. The threat model assumes:

- The host machine is trusted — operators don't run chainbench-net on
  adversarial hardware.
- Other processes on the same host may observe stdout / stderr / log files.
- Remote RPCs MUST NEVER see plaintext key material — they receive only
  signed transactions.
- Any code path inside chainbench-net that could serialize a signer to
  stdout/stderr/log/disk is a contract violation.

## Key Injection (current — env only)

Private keys enter via env vars at chainbench-net spawn time:

```
CHAINBENCH_SIGNER_<ALIAS>_KEY=0x<64-hex-chars>
```

Where `<ALIAS>` matches `[A-Za-z0-9_-]+`. Commands reference the alias via
`signer: "<alias>"`; the handler does `signer.Load(alias)` → env read →
in-memory Signer. On process exit, the OS reclaims the memory.

### Never

- Do NOT commit the env value to git, shell rc files, or CI configs where
  history is not purged.
- Do NOT place the key in the network state file. State files hold only
  aliases.
- Do NOT send the env file over the network (LLM chat, Slack, etc.).

## Key Injection (planned — Sprint 4b)

Keystore provider: `CHAINBENCH_SIGNER_<ALIAS>_KEYSTORE` + `_KEYSTORE_PASSWORD`
env pair. Same alias model; on load, decrypt keystore file in memory.

## Operator Checklist

- [ ] Rotate keys on any suspected exposure — env history, error dumps,
  screenshots.
- [ ] Prefer short-lived test keys over long-lived production keys.
- [ ] Scrub `.env` files from CI artifact uploads and dev-machine backups.
- [ ] Verify on every release that `tests/unit/tests/security-key-boundary.sh`
  is part of CI's required green suite.

## Developer Contract

- Key material lives ONLY in the `network/internal/signer` package's sealed
  struct. No method exposes the key.
- `slog.LogValuer` redacts to `"***"` for any slog attr carrying a Signer.
- `Stringer` / `GoStringer` on the signer render `<signer:***>` to cover
  `%v` / `%+v` / `%#v` / `%s` formatting paths.
- Error messages reference the **alias** and **env var name** only — never
  the value.
- New code that handles a Signer must pass the grep test:
  `grep -rn 'signer\..*key\|privateKey\|PrivateKey' network/` surfaces no
  export, serialization, or log path.

## Boundary Enforcement

Two tests enforce the boundary:

1. `network/internal/signer/signer_test.go` — unit-level fmt + slog redaction
   tests across %v, %+v, %#v, %s.
2. `tests/unit/tests/security-key-boundary.sh` — end-to-end binary spawn +
   grep of stdout/stderr/log for the raw key hex.

Both must stay green for any PR touching signer code or any handler that
accepts a signer alias.

## Out of Scope (current sprint)

- HSM integration
- Multi-party signing / threshold keys
- Key derivation from seed phrases inside chainbench (assume operator
  derives externally and exports keys as env)
- Audit logging of sign operations (design open — would need redaction
  patterns for the alias/address pair)
```

- [ ] **Step 2: Update VISION roadmap**

Append after the 3c row:
```
- [x] `network/internal/signer` (env) + `node.tx_send` + `docs/SECURITY_KEY_HANDLING.md` + Go/bash 보안 경계 테스트 — Sprint 4 완료 (2026-04-24); keystore provider + EIP-1559 은 Sprint 4b
```

- [ ] **Step 3: Full matrix**

```bash
cd network && go test ./... -count=1 -timeout=60s
cd .. && bash tests/unit/run.sh
```

- [ ] **Step 4: Commit**

```bash
git add docs/SECURITY_KEY_HANDLING.md docs/VISION_AND_ROADMAP.md
git commit -m "docs: SECURITY_KEY_HANDLING.md + mark Sprint 4 complete

Documents the S4 security contract: env injection, alias naming,
developer contract, operator checklist, and the two tests that
enforce the boundary. Roadmap marks Sprint 4 complete with keystore +
EIP-1559 deferred to Sprint 4b."
```

- [ ] **Step 5: Report**

Commit range, test counts, coverage, deferrals (keystore, EIP-1559, tx.wait).
