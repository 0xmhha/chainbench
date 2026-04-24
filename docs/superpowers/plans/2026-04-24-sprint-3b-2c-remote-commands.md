# Sprint 3b.2c — Remote Commands Expansion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Ship three new read-only remote commands (`node.chain_id`, `node.balance`, `node.gas_price`) via a shared `dialNode` helper. Tighten `network.attach` with auth validation at input boundary (INVALID_ARGS for structurally-invalid auth). Absorb M4 fully by delegating `resolveNodeIDFromString` through the network-aware `resolveNode`.

**Architecture:** New methods on `remote.Client` (ChainID/BalanceAt/GasPrice); `remote.ValidateAuth` for attach-time checks; handlers-layer `dialNode` helper extracts dial+auth boilerplate shared across 4 commands; local-lifecycle handlers unchanged (internal refactor only).

**Tech Stack:** Go 1.25, go-ethereum ethclient (existing dep), net/http/httptest for mocks.

Spec: `docs/superpowers/specs/2026-04-24-sprint-3b-2c-remote-commands.md`

---

## Commit Discipline

- English, no Co-Authored-By, no "Generated with Claude Code", no emoji.

## File Structure

**Modify:**
- `network/internal/drivers/remote/client.go` — add ChainID, BalanceAt, GasPrice
- `network/internal/drivers/remote/client_test.go` — tests for new methods
- `network/internal/drivers/remote/auth.go` — add ValidateAuth
- `network/internal/drivers/remote/auth_test.go` — tests for ValidateAuth
- `network/schema/command.json` — add 3 new enum entries
- `network/internal/types/command_gen.go` — regenerated
- `network/cmd/chainbench-net/handlers.go` — new handlers + dialNode helper + resolveNodeIDFromString delegation
- `network/cmd/chainbench-net/handlers_test.go` — new tests
- `network/cmd/chainbench-net/e2e_test.go` — Go E2E coverage
- `docs/VISION_AND_ROADMAP.md` — mark 3b.2c complete

**Create:**
- `tests/unit/tests/node-remote-reads.sh` — bash E2E for all 3 new commands

---

## Task 1 — remote.Client new methods + ValidateAuth

**Files:**
- Modify: `network/internal/drivers/remote/client.go`
- Modify: `network/internal/drivers/remote/client_test.go`
- Modify: `network/internal/drivers/remote/auth.go`
- Modify: `network/internal/drivers/remote/auth_test.go`

- [ ] **Step 1: Write failing tests for new Client methods**

Append to `client_test.go`:

```go
import "math/big"
// (plus common, if not imported)
import "github.com/ethereum/go-ethereum/common"

func TestClient_ChainID(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req struct{ Method string; ID json.RawMessage }
        _ = json.NewDecoder(r.Body).Decode(&req)
        w.Header().Set("Content-Type", "application/json")
        if req.Method == "eth_chainId" {
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x2a"}`, req.ID)
            return
        }
        fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
    }))
    defer srv.Close()
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    c, err := Dial(ctx, srv.URL)
    if err != nil { t.Fatalf("Dial: %v", err) }
    defer c.Close()
    cid, err := c.ChainID(ctx)
    if err != nil { t.Fatalf("ChainID: %v", err) }
    if cid.Cmp(big.NewInt(42)) != 0 {
        t.Errorf("ChainID = %v, want 42", cid)
    }
}

func TestClient_BalanceAt(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req struct{ Method string; ID json.RawMessage }
        _ = json.NewDecoder(r.Body).Decode(&req)
        w.Header().Set("Content-Type", "application/json")
        if req.Method == "eth_getBalance" {
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x100"}`, req.ID)
            return
        }
        fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
    }))
    defer srv.Close()
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    c, err := Dial(ctx, srv.URL)
    if err != nil { t.Fatalf("Dial: %v", err) }
    defer c.Close()
    addr := common.HexToAddress("0x0000000000000000000000000000000000000001")
    bal, err := c.BalanceAt(ctx, addr, nil) // latest
    if err != nil { t.Fatalf("BalanceAt: %v", err) }
    if bal.Cmp(big.NewInt(0x100)) != 0 {
        t.Errorf("balance = %v, want 0x100", bal)
    }
}

func TestClient_GasPrice(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req struct{ Method string; ID json.RawMessage }
        _ = json.NewDecoder(r.Body).Decode(&req)
        w.Header().Set("Content-Type", "application/json")
        if req.Method == "eth_gasPrice" {
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x3b9aca00"}`, req.ID) // 1 gwei
            return
        }
        fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
    }))
    defer srv.Close()
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    c, err := Dial(ctx, srv.URL)
    if err != nil { t.Fatalf("Dial: %v", err) }
    defer c.Close()
    gp, err := c.GasPrice(ctx)
    if err != nil { t.Fatalf("GasPrice: %v", err) }
    if gp.Cmp(big.NewInt(1_000_000_000)) != 0 {
        t.Errorf("gas_price = %v, want 1 gwei", gp)
    }
}
```

- [ ] **Step 2: Implement new Client methods**

Append to `client.go`:

```go
// Add imports if not present:
//   "math/big"
//   "github.com/ethereum/go-ethereum/common"

// ChainID returns the chain id reported by the endpoint.
func (c *Client) ChainID(ctx context.Context) (*big.Int, error) {
    id, err := c.rpc.ChainID(ctx)
    if err != nil {
        return nil, fmt.Errorf("remote.ChainID: %w", err)
    }
    return id, nil
}

// BalanceAt returns the balance of address at the given block. Pass nil for latest.
func (c *Client) BalanceAt(ctx context.Context, address common.Address, blockNumber *big.Int) (*big.Int, error) {
    bal, err := c.rpc.BalanceAt(ctx, address, blockNumber)
    if err != nil {
        return nil, fmt.Errorf("remote.BalanceAt: %w", err)
    }
    return bal, nil
}

// GasPrice returns the current suggested gas price (eth_gasPrice).
func (c *Client) GasPrice(ctx context.Context) (*big.Int, error) {
    gp, err := c.rpc.SuggestGasPrice(ctx)
    if err != nil {
        return nil, fmt.Errorf("remote.GasPrice: %w", err)
    }
    return gp, nil
}
```

- [ ] **Step 3: Add ValidateAuth tests to auth_test.go**

```go
func TestValidateAuth_NilIsOK(t *testing.T) {
    if err := ValidateAuth(nil); err != nil {
        t.Errorf("nil auth should be valid: %v", err)
    }
    if err := ValidateAuth(types.Auth{}); err != nil {
        t.Errorf("empty auth should be valid: %v", err)
    }
}

func TestValidateAuth_ValidAPIKey(t *testing.T) {
    if err := ValidateAuth(types.Auth{"type": "api-key", "env": "KEY"}); err != nil {
        t.Errorf("valid api-key should pass: %v", err)
    }
}

func TestValidateAuth_ValidJWT(t *testing.T) {
    if err := ValidateAuth(types.Auth{"type": "jwt", "env": "TOK"}); err != nil {
        t.Errorf("valid jwt should pass: %v", err)
    }
}

func TestValidateAuth_SSHPasswordPasses(t *testing.T) {
    // ssh-password is persisted but ignored by RPC client; attach accepts it.
    if err := ValidateAuth(types.Auth{"type": "ssh-password", "user": "root", "host": "h"}); err != nil {
        t.Errorf("ssh-password should pass attach validation: %v", err)
    }
}

func TestValidateAuth_UnknownType(t *testing.T) {
    if err := ValidateAuth(types.Auth{"type": "totally-made-up"}); err == nil {
        t.Error("unknown type should fail")
    }
}

func TestValidateAuth_MissingType(t *testing.T) {
    if err := ValidateAuth(types.Auth{"env": "KEY"}); err == nil {
        t.Error("missing type should fail")
    }
}

func TestValidateAuth_APIKey_MissingEnv(t *testing.T) {
    if err := ValidateAuth(types.Auth{"type": "api-key"}); err == nil {
        t.Error("api-key without env should fail")
    }
}

func TestValidateAuth_JWT_MissingEnv(t *testing.T) {
    if err := ValidateAuth(types.Auth{"type": "jwt"}); err == nil {
        t.Error("jwt without env should fail")
    }
}
```

- [ ] **Step 4: Implement ValidateAuth in auth.go**

```go
// ValidateAuth reports whether auth is structurally valid.
// Returns nil for empty auth (unauthenticated). Callers at input boundaries
// (e.g., network.attach) use this to fail fast on malformed configuration
// before persistence. AuthFromNode re-validates the same rules at dial time.
func ValidateAuth(auth types.Auth) error {
    if len(auth) == 0 {
        return nil
    }
    rawType, ok := auth["type"].(string)
    if !ok || rawType == "" {
        return fmt.Errorf("auth: missing or non-string 'type' field")
    }
    switch rawType {
    case "api-key", "jwt":
        envName, _ := auth["env"].(string)
        if envName == "" {
            return fmt.Errorf("auth(%s): 'env' field is required", rawType)
        }
    case "ssh-password":
        // SSH fields validated by the SSH driver when it lands; attach accepts.
    default:
        return fmt.Errorf("auth: unknown type %q", rawType)
    }
    return nil
}
```

- [ ] **Step 5: Run tests**

```bash
cd network && go test ./internal/drivers/remote/... -v -count=1 -timeout=30s
```
Expected: all PASS, coverage stays ≥95%.

- [ ] **Step 6: Full suite + lint**

```bash
cd network && go test ./... -count=1 -timeout=60s
go -C network vet ./...
gofmt -l network/
```

- [ ] **Step 7: Commit**

```bash
git add network/internal/drivers/remote/
git commit -m "feat(drivers/remote): add ChainID/BalanceAt/GasPrice + ValidateAuth

New Client methods are thin wrappers over ethclient counterparts, error-
wrapped with the remote.<Method> prefix per package convention.

ValidateAuth checks a types.Auth payload without mutating it; returns nil
for empty auth. Handlers use this at input boundaries (attach) to reject
structurally-invalid configurations up front with INVALID_ARGS rather
than deferring to dial time and returning UPSTREAM_ERROR."
```

---

## Task 2 — schema + dialNode helper + 3 new handlers

**Files:**
- Modify: `network/schema/command.json` — add 3 enum entries
- Modify: `network/internal/types/command_gen.go` — regenerated
- Modify: `network/cmd/chainbench-net/handlers.go`
- Modify: `network/cmd/chainbench-net/handlers_test.go`

- [ ] **Step 1: Schema**

Edit `network/schema/command.json`. Add `"node.chain_id"`, `"node.balance"`, `"node.gas_price"` to the command enum. Keep alphabetical grouping within the `node.*` section.

- [ ] **Step 2: Regenerate types**

```bash
cd network && go generate ./...
```

Verify `command_gen.go` gained three new consts.

- [ ] **Step 3: Write failing handler tests**

Append to `handlers_test.go` (use existing `newTestBus`, and reuse the stablenet-shaped mock pattern from prior tests):

```go
// Shared helper for the three new handlers' happy paths — mock server that
// returns canned results for eth_chainId / eth_getBalance / eth_gasPrice.
func newReadMockRPC(t *testing.T) *httptest.Server {
    t.Helper()
    return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req struct{ Method string; ID json.RawMessage }
        _ = json.NewDecoder(r.Body).Decode(&req)
        w.Header().Set("Content-Type", "application/json")
        switch req.Method {
        case "eth_chainId":
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x2a"}`, req.ID)
        case "eth_getBalance":
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x100"}`, req.ID)
        case "eth_gasPrice":
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x3b9aca00"}`, req.ID)
        default:
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
        }
    }))
}

// Helper to pre-save a remote network pointing at srv.
func saveRemoteFixture(t *testing.T, stateDir, name, url string) {
    t.Helper()
    net := &types.Network{
        Name: name, ChainType: "ethereum", ChainId: 1,
        Nodes: []types.Node{{Id: "node1", Provider: "remote", Http: url}},
    }
    if err := state.SaveRemote(stateDir, net); err != nil {
        t.Fatal(err)
    }
}

func TestHandleNodeChainID_Happy(t *testing.T) {
    srv := newReadMockRPC(t); defer srv.Close()
    stateDir := t.TempDir()
    saveRemoteFixture(t, stateDir, "tn", srv.URL)
    h := newHandleNodeChainID(stateDir)
    args, _ := json.Marshal(map[string]any{"network": "tn", "node_id": "node1"})
    bus, _ := newTestBus(t)
    data, err := h(args, bus)
    if err != nil { t.Fatalf("handler: %v", err) }
    if data["network"] != "tn" || data["node_id"] != "node1" {
        t.Errorf("echo fields: %+v", data)
    }
    if cid, ok := data["chain_id"].(uint64); !ok || cid != 42 {
        t.Errorf("chain_id = %v (%T), want 42", data["chain_id"], data["chain_id"])
    }
}

func TestHandleNodeBalance_Happy(t *testing.T) {
    srv := newReadMockRPC(t); defer srv.Close()
    stateDir := t.TempDir()
    saveRemoteFixture(t, stateDir, "tn", srv.URL)
    h := newHandleNodeBalance(stateDir)
    args, _ := json.Marshal(map[string]any{
        "network": "tn", "node_id": "node1",
        "address": "0x0000000000000000000000000000000000000001",
    })
    bus, _ := newTestBus(t)
    data, err := h(args, bus)
    if err != nil { t.Fatalf("handler: %v", err) }
    if data["balance"] != "0x100" {
        t.Errorf("balance = %v, want 0x100", data["balance"])
    }
}

func TestHandleNodeBalance_BadAddress(t *testing.T) {
    srv := newReadMockRPC(t); defer srv.Close()
    stateDir := t.TempDir()
    saveRemoteFixture(t, stateDir, "tn", srv.URL)
    h := newHandleNodeBalance(stateDir)
    args, _ := json.Marshal(map[string]any{
        "network": "tn", "node_id": "node1",
        "address": "not-a-hex-address",
    })
    bus, _ := newTestBus(t)
    _, err := h(args, bus)
    var api *APIError
    if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
        t.Errorf("want INVALID_ARGS, got %v", err)
    }
}

func TestHandleNodeBalance_MissingAddress(t *testing.T) {
    stateDir := t.TempDir()
    h := newHandleNodeBalance(stateDir)
    args, _ := json.Marshal(map[string]any{"node_id": "node1"})
    bus, _ := newTestBus(t)
    _, err := h(args, bus)
    var api *APIError
    if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
        t.Errorf("want INVALID_ARGS, got %v", err)
    }
}

func TestHandleNodeGasPrice_Happy(t *testing.T) {
    srv := newReadMockRPC(t); defer srv.Close()
    stateDir := t.TempDir()
    saveRemoteFixture(t, stateDir, "tn", srv.URL)
    h := newHandleNodeGasPrice(stateDir)
    args, _ := json.Marshal(map[string]any{"network": "tn", "node_id": "node1"})
    bus, _ := newTestBus(t)
    data, err := h(args, bus)
    if err != nil { t.Fatalf("handler: %v", err) }
    if data["gas_price"] != "0x3b9aca00" {
        t.Errorf("gas_price = %v, want 0x3b9aca00", data["gas_price"])
    }
}

func TestHandleNetworkAttach_InvalidAuthRejectedAtAttach(t *testing.T) {
    rpcSrv := newStablenetMockRPC(t); defer rpcSrv.Close()
    stateDir := t.TempDir()
    h := newHandleNetworkAttach(stateDir)
    args, _ := json.Marshal(map[string]any{
        "rpc_url": rpcSrv.URL, "name": "bad",
        "auth": map[string]any{"type": "totally-unknown-type"},
    })
    bus, _ := newTestBus(t)
    _, err := h(args, bus)
    var api *APIError
    if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
        t.Errorf("want INVALID_ARGS, got %v", err)
    }
    // Ensure the state file was NOT written — attach must reject before persistence.
    if _, serr := os.Stat(filepath.Join(stateDir, "networks", "bad.json")); !os.IsNotExist(serr) {
        t.Errorf("state file should not exist after invalid-auth attach: %v", serr)
    }
}

// Registration sanity — three new commands present.
func TestAllHandlers_IncludesNewRemoteReadCommands(t *testing.T) {
    h := allHandlers("x", "y")
    for _, name := range []string{"node.chain_id", "node.balance", "node.gas_price"} {
        if _, ok := h[name]; !ok {
            t.Errorf("allHandlers missing %s", name)
        }
    }
}
```

- [ ] **Step 4: Implement dialNode helper in handlers.go**

Add near `resolveNode`:

```go
// dialNode opens a remote.Client for the resolved node, wiring auth if
// node.Auth is populated. Caller owns Close. Errors are mapped to APIError
// sentinels: UPSTREAM_ERROR for auth setup or dial failures.
func dialNode(ctx context.Context, node *types.Node) (*remote.Client, error) {
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
}
```

- [ ] **Step 5: Refactor `newHandleNodeBlockNumber` to use dialNode**

The existing in-body dial+auth code (~7 lines) becomes `client, err := dialNode(ctx, &node); if err != nil { return nil, err }; defer client.Close()`.

- [ ] **Step 6: Implement the three new handlers**

Add after `newHandleNodeBlockNumber`:

```go
// newHandleNodeChainID returns a handler for node.chain_id.
// Args: {network?, node_id}
// Result: {network, node_id, chain_id: <uint64>}
func newHandleNodeChainID(stateDir string) Handler {
    return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
        var req struct {
            Network string `json:"network"`
            NodeID  string `json:"node_id"`
        }
        if len(args) > 0 {
            if err := json.Unmarshal(args, &req); err != nil {
                return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
            }
        }
        networkName, node, err := resolveNode(stateDir, req.Network, req.NodeID)
        if err != nil {
            return nil, err
        }
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        client, err := dialNode(ctx, &node)
        if err != nil {
            return nil, err
        }
        defer client.Close()
        cid, err := client.ChainID(ctx)
        if err != nil {
            return nil, NewUpstream("eth_chainId", err)
        }
        return map[string]any{
            "network":  networkName,
            "node_id":  node.Id,
            "chain_id": cid.Uint64(),
        }, nil
    }
}

// newHandleNodeBalance returns a handler for node.balance.
// Args: {network?, node_id, address: "0x...", block_number?: <int>|"latest"|"earliest"|"pending"}
// Result: {network, node_id, address, block: <string>, balance: "0x..."}
func newHandleNodeBalance(stateDir string) Handler {
    return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
        var req struct {
            Network     string          `json:"network"`
            NodeID      string          `json:"node_id"`
            Address     string          `json:"address"`
            BlockNumber json.RawMessage `json:"block_number"`
        }
        if len(args) > 0 {
            if err := json.Unmarshal(args, &req); err != nil {
                return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
            }
        }
        if req.Address == "" {
            return nil, NewInvalidArgs("args.address is required")
        }
        if !common.IsHexAddress(req.Address) {
            return nil, NewInvalidArgs(fmt.Sprintf("args.address is not a valid hex address: %q", req.Address))
        }
        addr := common.HexToAddress(req.Address)

        var blockNum *big.Int
        blockLabel := "latest"
        if len(req.BlockNumber) > 0 {
            // Try integer first, then string label.
            var asInt int64
            var asStr string
            if err := json.Unmarshal(req.BlockNumber, &asInt); err == nil {
                if asInt < 0 {
                    return nil, NewInvalidArgs("args.block_number must be non-negative")
                }
                blockNum = big.NewInt(asInt)
                blockLabel = fmt.Sprintf("%d", asInt)
            } else if err := json.Unmarshal(req.BlockNumber, &asStr); err == nil {
                switch asStr {
                case "latest", "earliest", "pending":
                    blockLabel = asStr
                    // ethclient interprets nil as "latest"; earliest=0, pending unsupported.
                    // For 3b.2c we route "earliest" to block 0 and leave "pending" as nil
                    // (ethclient treats nil as "latest" — document this limitation).
                    if asStr == "earliest" {
                        blockNum = big.NewInt(0)
                    }
                default:
                    return nil, NewInvalidArgs(fmt.Sprintf("args.block_number label invalid: %q", asStr))
                }
            } else {
                return nil, NewInvalidArgs("args.block_number must be an integer or a block label string")
            }
        }

        networkName, node, err := resolveNode(stateDir, req.Network, req.NodeID)
        if err != nil {
            return nil, err
        }
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        client, err := dialNode(ctx, &node)
        if err != nil {
            return nil, err
        }
        defer client.Close()
        bal, err := client.BalanceAt(ctx, addr, blockNum)
        if err != nil {
            return nil, NewUpstream("eth_getBalance", err)
        }
        return map[string]any{
            "network": networkName,
            "node_id": node.Id,
            "address": req.Address,
            "block":   blockLabel,
            "balance": "0x" + bal.Text(16),
        }, nil
    }
}

// newHandleNodeGasPrice returns a handler for node.gas_price.
// Args: {network?, node_id}
// Result: {network, node_id, gas_price: "0x..."}
func newHandleNodeGasPrice(stateDir string) Handler {
    return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
        var req struct {
            Network string `json:"network"`
            NodeID  string `json:"node_id"`
        }
        if len(args) > 0 {
            if err := json.Unmarshal(args, &req); err != nil {
                return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
            }
        }
        networkName, node, err := resolveNode(stateDir, req.Network, req.NodeID)
        if err != nil {
            return nil, err
        }
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        client, err := dialNode(ctx, &node)
        if err != nil {
            return nil, err
        }
        defer client.Close()
        gp, err := client.GasPrice(ctx)
        if err != nil {
            return nil, NewUpstream("eth_gasPrice", err)
        }
        return map[string]any{
            "network":   networkName,
            "node_id":   node.Id,
            "gas_price": "0x" + gp.Text(16),
        }, nil
    }
}
```

Add imports to handlers.go: `"math/big"`, `"github.com/ethereum/go-ethereum/common"`.

- [ ] **Step 7: Register handlers in `allHandlers`**

```go
"node.chain_id":     newHandleNodeChainID(stateDir),
"node.balance":      newHandleNodeBalance(stateDir),
"node.gas_price":    newHandleNodeGasPrice(stateDir),
```

- [ ] **Step 8: Wire attach-time auth validation**

Inside `newHandleNetworkAttach`, after the `req.Auth` unmarshal into `types.Auth`, call `remote.ValidateAuth(auth)` and return INVALID_ARGS on error:

```go
if len(req.Auth) > 0 {
    var auth types.Auth
    if err := json.Unmarshal(req.Auth, &auth); err != nil {
        return nil, NewInvalidArgs(fmt.Sprintf("args.auth: %v", err))
    }
    if err := remote.ValidateAuth(auth); err != nil {
        return nil, NewInvalidArgs(err.Error())
    }
    net.Nodes[0].Auth = auth
}
```

Ensure this runs BEFORE probe.Detect — validation is pure structural, should not require a round trip. Actually: **leave probe first** (probe checks endpoint reachable + classifies chain_type). Auth validation after probe is fine because probe is unauthenticated anyway. The critical thing is validation must run BEFORE SaveRemote. Currently the handler does probe → stat (for created flag) → build net → save. Insert ValidateAuth between probe success and build-net.

- [ ] **Step 9: Run handler tests**

```bash
cd network && go test ./cmd/chainbench-net/... -v -count=1 -timeout=60s
```
Expected: all PASS including new tests and pre-existing.

- [ ] **Step 10: Full suite + lint**

```bash
cd network && go test ./... -count=1 -timeout=60s
go -C network vet ./...
gofmt -l network/
```

- [ ] **Step 11: Commit**

```bash
git add network/schema/command.json network/internal/types/command_gen.go network/cmd/chainbench-net/
git commit -m "feat(network-net): node.chain_id / node.balance / node.gas_price + attach auth validation

Three new read-only remote commands, all dispatched through a shared
dialNode helper that folds in the auth wiring once. node.block_number
migrates to the same helper (no behavior change, -6 lines).

network.attach now calls remote.ValidateAuth on the supplied auth
payload before persisting — structurally invalid configurations
(unknown type, missing required env field) are rejected with
INVALID_ARGS at attach time rather than surfacing later at dial as
UPSTREAM_ERROR.

Schema: add 'node.chain_id', 'node.balance', 'node.gas_price' to the
command enum; command_gen.go regenerated."
```

---

## Task 3 — M4 full absorption

**Files:**
- Modify: `network/cmd/chainbench-net/handlers.go`

- [ ] **Step 1: Refactor resolveNodeIDFromString to delegate through resolveNode**

Replace the body with:

```go
func resolveNodeIDFromString(stateDir, nodeID string) (string, string, error) {
    // Local-only convention: node<N> where N is the numeric pids.json key.
    // chainbench.sh takes the raw numeric suffix as its arg.
    if nodeID == "" {
        return "", "", NewInvalidArgs("args.node_id is required")
    }
    if !strings.HasPrefix(nodeID, "node") {
        return "", "", NewInvalidArgs(fmt.Sprintf(`node_id must start with "node" prefix (got %q)`, nodeID))
    }
    num := strings.TrimPrefix(nodeID, "node")
    if num == "" {
        return "", "", NewInvalidArgs("node_id missing numeric suffix")
    }
    // Delegate existence check through resolveNode (M4 absorption — single
    // state.LoadActive call site). resolveNode does the node-lookup + error
    // mapping; we only need the numeric suffix the caller passes back to the
    // local chainbench.sh CLI.
    if _, _, err := resolveNode(stateDir, "local", nodeID); err != nil {
        return "", "", err
    }
    return nodeID, num, nil
}
```

- [ ] **Step 2: Run existing local-lifecycle tests**

```bash
cd network && go test ./cmd/chainbench-net/... -run 'TestHandleNodeStop|TestHandleNodeStart|TestHandleNodeRestart|TestHandleNodeTailLog' -v -count=1
```
Expected: all existing tests still pass — this is a pure internal refactor.

- [ ] **Step 3: Verify M4 absorption via final look**

```bash
grep -n 'Name: "local"' network/cmd/chainbench-net/handlers.go
```
Expected: the only remaining hardcoded `Name: "local"` is inside `resolveNodeIDFromString` — the one place where it's semantically correct (local-only numeric-suffix convention).

- [ ] **Step 4: Commit**

```bash
git add network/cmd/chainbench-net/handlers.go
git commit -m "refactor(network-net): absorb M4 — resolveNodeIDFromString delegates to resolveNode

Single state.LoadActive call site for node existence checks. Local-only
numeric-suffix validation is still local to this helper (chainbench.sh
argument shape), but the state-layer interaction is centralized in
resolveNode. No behavior change; internal cleanup."
```

---

## Task 4 — Go E2E + bash tests

**Files:**
- Modify: `network/cmd/chainbench-net/e2e_test.go`
- Create: `tests/unit/tests/node-remote-reads.sh`

- [ ] **Step 1: Go E2E covering the three new commands**

Append one E2E that attaches with auth, then hits all three commands:

```go
func TestE2E_NodeRemoteReads_WithAuth(t *testing.T) {
    var authSeen bool
    rpcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Header.Get("X-Api-Key") == "e2e-reads" {
            authSeen = true
        }
        var req struct{ Method string `json:"method"`; ID json.RawMessage `json:"id"` }
        _ = json.NewDecoder(r.Body).Decode(&req)
        w.Header().Set("Content-Type", "application/json")
        switch req.Method {
        case "eth_chainId":
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x2a"}`, req.ID)
        case "eth_getBalance":
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x500"}`, req.ID)
        case "eth_gasPrice":
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x64"}`, req.ID)
        case "istanbul_getValidators", "wemix_getReward":
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
        default:
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
        }
    }))
    defer rpcSrv.Close()

    stateDir := t.TempDir()
    t.Setenv("CHAINBENCH_STATE_DIR", stateDir)
    t.Setenv("READS_KEY", "e2e-reads")

    // attach
    runRoot := func(t *testing.T, stdinJSON string) map[string]any {
        t.Helper()
        var stdout, stderr bytes.Buffer
        root := newRootCmd()
        root.SetIn(strings.NewReader(stdinJSON))
        root.SetOut(&stdout)
        root.SetErr(&stderr)
        root.SetArgs([]string{"run"})
        if err := root.Execute(); err != nil {
            t.Fatalf("execute: %v stderr=%s", err, stderr.String())
        }
        var resultLine []byte
        scanner := bufio.NewScanner(&stdout)
        for scanner.Scan() {
            line := append([]byte(nil), scanner.Bytes()...)
            msg, _ := wire.DecodeMessage(line)
            if _, ok := msg.(wire.ResultMessage); ok {
                resultLine = line
            }
        }
        if resultLine == nil {
            t.Fatal("no result line")
        }
        var res struct {
            Ok   bool           `json:"ok"`
            Data map[string]any `json:"data"`
        }
        _ = json.Unmarshal(resultLine, &res)
        if !res.Ok {
            t.Fatalf("not ok: %s", resultLine)
        }
        return res.Data
    }

    attachCmd := fmt.Sprintf(
        `{"command":"network.attach","args":{"rpc_url":%q,"name":"reads-e2e","auth":{"type":"api-key","header":"X-Api-Key","env":"READS_KEY"}}}`,
        rpcSrv.URL,
    )
    runRoot(t, attachCmd)

    chainData := runRoot(t, `{"command":"node.chain_id","args":{"network":"reads-e2e","node_id":"node1"}}`)
    if cid, ok := chainData["chain_id"].(float64); !ok || cid != 42 {
        t.Errorf("chain_id = %v, want 42", chainData["chain_id"])
    }

    balData := runRoot(t, `{"command":"node.balance","args":{"network":"reads-e2e","node_id":"node1","address":"0x0000000000000000000000000000000000000001"}}`)
    if balData["balance"] != "0x500" {
        t.Errorf("balance = %v, want 0x500", balData["balance"])
    }

    gpData := runRoot(t, `{"command":"node.gas_price","args":{"network":"reads-e2e","node_id":"node1"}}`)
    if gpData["gas_price"] != "0x64" {
        t.Errorf("gas_price = %v, want 0x64", gpData["gas_price"])
    }

    if !authSeen {
        t.Errorf("mock never saw the auth header — dialNode may not be wiring auth")
    }
}
```

- [ ] **Step 2: Bash test**

Create `tests/unit/tests/node-remote-reads.sh` — follow the hardening pattern from `node-block-number.sh` (trap EXIT INT TERM HUP, readiness poll with fail-loud, ephemeral port, set -euo pipefail). Mock handles the same 3 eth_* methods. Scenarios:

```bash
# After attach
data="$(cb_net_call "node.chain_id" '{"network":"reads","node_id":"node1"}')"
assert_eq "$(jq -r .chain_id <<<"$data")" "42" "chain_id"

data="$(cb_net_call "node.balance" '{"network":"reads","node_id":"node1","address":"0x0000000000000000000000000000000000000001"}')"
assert_eq "$(jq -r .balance <<<"$data")" "0x500" "balance"

data="$(cb_net_call "node.gas_price" '{"network":"reads","node_id":"node1"}')"
assert_eq "$(jq -r .gas_price <<<"$data")" "0x64" "gas_price"
```

- [ ] **Step 3: Run all tests**

```bash
cd network && go test ./cmd/chainbench-net/... -run TestE2E_NodeRemoteReads -v -timeout=60s
cd network && go test ./... -count=1 -timeout=60s
bash tests/unit/tests/node-remote-reads.sh
bash tests/unit/run.sh
```
Expected: all green.

- [ ] **Step 4: Commit**

```bash
git add network/cmd/chainbench-net/e2e_test.go tests/unit/tests/node-remote-reads.sh
git commit -m "test: E2E + bash for node.chain_id / node.balance / node.gas_price

Go E2E attaches a mock RPC with api-key auth, runs all three new
commands sequentially, and asserts the auth header reached the server
on at least one call (proves dialNode is threading auth through
uniformly). Bash test covers the subprocess path at parity."
```

---

## Task 5 — Final review + roadmap

- [ ] **Step 1: Full test matrix**

```bash
cd network && go test ./... -count=1 -timeout=60s
cd .. && bash tests/unit/run.sh
```

- [ ] **Step 2: Update roadmap**

Edit `docs/VISION_AND_ROADMAP.md` Sprint 3 entry. Add a line after the 3b.2b row:

```
- [x] Remote 읽기 커맨드 확장 (`node.chain_id`, `node.balance`, `node.gas_price`) + `dialNode` helper + attach auth 검증 + M4 완전 흡수 — Sprint 3b.2c 완료 (2026-04-24)
```

- [ ] **Step 3: Commit**

```bash
git add docs/VISION_AND_ROADMAP.md
git commit -m "docs: mark Sprint 3b.2c complete — remote read-only expansion"
```

- [ ] **Step 4: Report**

Commit range, test counts, coverage, deferrals (401/403, node.fee_history, WebSocket).
