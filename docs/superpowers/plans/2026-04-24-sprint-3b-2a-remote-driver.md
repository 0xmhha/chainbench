# Sprint 3b.2a — RemoteDriver + node.block_number Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** First remote-node command (`node.block_number`) works against any attached network. Introduces `go-ethereum/ethclient` dep via `drivers/remote` package. Adds `resolveNode` helper (M4 absorption) and a NOT_SUPPORTED guard for local-only lifecycle handlers.

**Architecture:** Thin `remote.Client` wrapping ethclient; handler uses Client.Dial(node.Http) → BlockNumber → Close. No driver dispatch; local and remote nodes share the same code path because both have an `Http` field.

**Tech Stack:** Go 1.25, `github.com/ethereum/go-ethereum/ethclient`, existing probe/state/wire infra, `net/http/httptest` for mocks.

Spec: `docs/superpowers/specs/2026-04-24-sprint-3b-2a-remote-driver.md`

---

## Commit Discipline

- English, no Co-Authored-By, no "Generated with Claude Code", no emoji.
- One commit per TDD cycle or logical grouping.
- go.mod + go.sum changes land with their first consumer (Task 1).

## File Structure

**Create:**
- `network/internal/drivers/remote/client.go`
- `network/internal/drivers/remote/client_test.go`
- `tests/unit/tests/node-block-number.sh`

**Modify:**
- `network/go.mod`, `network/go.sum` (via `go get`)
- `network/schema/command.json` — add `"node.block_number"`
- `network/internal/types/command_gen.go` — regenerated
- `network/cmd/chainbench-net/handlers.go` — resolveNode helper, node.block_number handler, NOT_SUPPORTED guard on node.stop/start/restart/tail_log
- `network/cmd/chainbench-net/handlers_test.go` — new tests
- `network/cmd/chainbench-net/e2e_test.go` — attach + block_number E2E
- `docs/VISION_AND_ROADMAP.md` — mark 3b.2a complete

---

## Task 1 — go-ethereum dep + drivers/remote/client.go

**Files:**
- Modify: `network/go.mod`, `network/go.sum`
- Create: `network/internal/drivers/remote/client.go`
- Create: `network/internal/drivers/remote/client_test.go`

- [ ] **Step 1: Add go-ethereum dep**

```bash
cd network && go get github.com/ethereum/go-ethereum/ethclient@latest
```

This pulls the transitive graph. Expect go.sum to grow by ~30+ lines.

- [ ] **Step 2: Write failing test for remote.Client**

```go
// network/internal/drivers/remote/client_test.go
package remote

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"
)

// mockRPC returns a JSON-RPC server responding to eth_blockNumber with the given hex.
// Unknown methods return -32601.
func mockRPC(t *testing.T, responses map[string]string) *httptest.Server {
    t.Helper()
    return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req struct{ Method string; ID json.RawMessage }
        _ = json.NewDecoder(r.Body).Decode(&req)
        w.Header().Set("Content-Type", "application/json")
        if result, ok := responses[req.Method]; ok {
            _, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"result":"` + result + `"}`))
            return
        }
        _, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"error":{"code":-32601,"message":"method not found"}}`))
    }))
}

func TestClient_BlockNumber(t *testing.T) {
    srv := mockRPC(t, map[string]string{"eth_blockNumber": "0x10"})
    defer srv.Close()

    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    c, err := Dial(ctx, srv.URL)
    if err != nil {
        t.Fatalf("Dial: %v", err)
    }
    defer c.Close()

    bn, err := c.BlockNumber(ctx)
    if err != nil {
        t.Fatalf("BlockNumber: %v", err)
    }
    if bn != 16 {
        t.Errorf("BlockNumber = %d, want 16", bn)
    }
}

func TestClient_DialRejectsBadURL(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), time.Second)
    defer cancel()
    _, err := Dial(ctx, "not-a-url")
    if err == nil {
        t.Fatal("expected Dial error for malformed URL")
    }
}

func TestClient_BlockNumber_RPCError(t *testing.T) {
    srv := mockRPC(t, map[string]string{}) // no methods — every call fails
    defer srv.Close()
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()
    c, err := Dial(ctx, srv.URL)
    if err != nil {
        t.Fatalf("Dial: %v", err)
    }
    defer c.Close()
    _, err = c.BlockNumber(ctx)
    if err == nil {
        t.Fatal("expected error when server returns method-not-found")
    }
}
```

- [ ] **Step 3: Run test to verify failure**

```bash
cd network && go test ./internal/drivers/remote/... -v
```
Expected: FAIL — `undefined: Dial`.

- [ ] **Step 4: Implement client.go**

```go
// network/internal/drivers/remote/client.go
//
// Thin wrapper around ethclient. Exists as a seam for future auth transport
// injection (Sprint 3b.2b) and method grouping (tx.send etc. in Sprint 4).
// Handlers talk to remote.Client, not ethclient directly.
package remote

import (
    "context"
    "fmt"

    "github.com/ethereum/go-ethereum/ethclient"
)

// Client is a read-only Ethereum RPC client backed by go-ethereum's ethclient.
// Construct via Dial and Close when done.
type Client struct {
    rpc *ethclient.Client
}

// Dial opens a JSON-RPC client against the given HTTP(S) URL.
func Dial(ctx context.Context, url string) (*Client, error) {
    rpc, err := ethclient.DialContext(ctx, url)
    if err != nil {
        return nil, fmt.Errorf("remote.Dial %q: %w", url, err)
    }
    return &Client{rpc: rpc}, nil
}

// BlockNumber returns the current head block number as reported by the endpoint.
func (c *Client) BlockNumber(ctx context.Context) (uint64, error) {
    bn, err := c.rpc.BlockNumber(ctx)
    if err != nil {
        return 0, fmt.Errorf("remote.BlockNumber: %w", err)
    }
    return bn, nil
}

// Close releases the underlying HTTP/RPC connection.
func (c *Client) Close() {
    if c != nil && c.rpc != nil {
        c.rpc.Close()
    }
}
```

- [ ] **Step 5: Run tests**

```bash
cd network && go test ./internal/drivers/remote/... -v -cover
```
Expected: all PASS, coverage ≥80%.

- [ ] **Step 6: Run full suite to catch dep-introduction regressions**

```bash
cd network && go build ./... && go test ./... -count=1
```
Expected: all packages build and test green.

- [ ] **Step 7: Commit**

```bash
git add network/go.mod network/go.sum network/internal/drivers/remote/
git commit -m "feat(drivers/remote): introduce ethclient-backed Client

Adds github.com/ethereum/go-ethereum/ethclient dependency and a minimal
remote.Client wrapper (Dial, BlockNumber, Close). Thin seam so future
auth Transport injection (3b.2b) and tx.send (Sprint 4) extend without
touching handler code paths.

go-stablenet / wbft / wemix nodes speak standard JSON-RPC for eth_*,
so the upstream ethclient works unchanged against them. Chain-specific
namespaces (istanbul_*, wemix_*) continue to use raw rpc.Client.Call
per the Sprint 3a probe pattern.

Heavy transitive deps expected — go.sum grows accordingly."
```

---

## Task 2 — resolveNode + node.block_number handler + guards

**Files:**
- Modify: `network/schema/command.json`
- Modify: `network/internal/types/command_gen.go` (regenerated)
- Modify: `network/cmd/chainbench-net/handlers.go`
- Modify: `network/cmd/chainbench-net/handlers_test.go`

- [ ] **Step 1: Add node.block_number to schema**

Edit `network/schema/command.json`. Add `"node.block_number"` to the command enum.

- [ ] **Step 2: Regenerate types**

```bash
cd network && go generate ./...
```

Verify `command_gen.go` has a new `CommandCommandNodeBlockNumber` const.

- [ ] **Step 3: Write failing handler tests**

Append to `handlers_test.go`:

```go
func TestHandleNodeBlockNumber_RemoteHappy(t *testing.T) {
    // Mock a JSON-RPC server that returns eth_blockNumber=0x10.
    rpcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req struct{ Method string; ID json.RawMessage }
        _ = json.NewDecoder(r.Body).Decode(&req)
        w.Header().Set("Content-Type", "application/json")
        if req.Method == "eth_blockNumber" {
            _, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"result":"0x10"}`))
            return
        }
        _, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"error":{"code":-32601,"message":"nf"}}`))
    }))
    defer rpcSrv.Close()

    // Pre-populate a remote network state via SaveRemote.
    stateDir := t.TempDir()
    net := &types.Network{
        Name: "testnet", ChainType: "ethereum", ChainId: 1,
        Nodes: []types.Node{{Id: "node1", Provider: "remote", Http: rpcSrv.URL}},
    }
    if err := state.SaveRemote(stateDir, net); err != nil {
        t.Fatal(err)
    }

    h := newHandleNodeBlockNumber(stateDir)
    args, _ := json.Marshal(map[string]any{"network": "testnet", "node_id": "node1"})
    bus, _ := newTestBus(t)
    data, err := h(args, bus)
    if err != nil {
        t.Fatalf("handler: %v", err)
    }
    if data["network"] != "testnet" {
        t.Errorf("network = %v", data["network"])
    }
    if data["node_id"] != "node1" {
        t.Errorf("node_id = %v", data["node_id"])
    }
    if bn, ok := data["block_number"].(uint64); !ok || bn != 16 {
        if f, fok := data["block_number"].(float64); fok && f == 16 {
            // JSON round-trip may produce float64; accept both.
        } else {
            t.Errorf("block_number = %v (type %T)", data["block_number"], data["block_number"])
        }
    }
}

func TestHandleNodeBlockNumber_MissingNodeID(t *testing.T) {
    h := newHandleNodeBlockNumber(t.TempDir())
    args, _ := json.Marshal(map[string]any{"network": "testnet"})
    bus, _ := newTestBus(t)
    _, err := h(args, bus)
    var api *APIError
    if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
        t.Errorf("want INVALID_ARGS, got %v", err)
    }
}

func TestHandleNodeBlockNumber_UnknownNetwork(t *testing.T) {
    h := newHandleNodeBlockNumber(t.TempDir())
    args, _ := json.Marshal(map[string]any{"network": "ghost", "node_id": "node1"})
    bus, _ := newTestBus(t)
    _, err := h(args, bus)
    var api *APIError
    if !errors.As(err, &api) || string(api.Code) != "UPSTREAM_ERROR" {
        t.Errorf("want UPSTREAM_ERROR, got %v", err)
    }
}

func TestHandleNodeBlockNumber_UnknownNode(t *testing.T) {
    stateDir := t.TempDir()
    net := &types.Network{
        Name: "tn", ChainType: "ethereum", ChainId: 1,
        Nodes: []types.Node{{Id: "node1", Provider: "remote", Http: "http://127.0.0.1:1"}},
    }
    _ = state.SaveRemote(stateDir, net)

    h := newHandleNodeBlockNumber(stateDir)
    args, _ := json.Marshal(map[string]any{"network": "tn", "node_id": "node9"})
    bus, _ := newTestBus(t)
    _, err := h(args, bus)
    var api *APIError
    if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
        t.Errorf("want INVALID_ARGS, got %v", err)
    }
}

// Local-only handler guard: node.stop with network:"remote" → NOT_SUPPORTED.
func TestHandleNodeStop_RejectsNonLocalNetwork(t *testing.T) {
    stateDir, chainbenchDir := setupCmdStubDir(t)
    h := newHandleNodeStop(stateDir, chainbenchDir)
    args, _ := json.Marshal(map[string]any{"network": "remote-foo", "node_id": "node1"})
    bus, _ := newTestBus(t)
    _, err := h(args, bus)
    var api *APIError
    if !errors.As(err, &api) || string(api.Code) != "NOT_SUPPORTED" {
        t.Errorf("want NOT_SUPPORTED, got %v", err)
    }
}
```

- [ ] **Step 4: Verify RED**

```bash
cd network && go test ./cmd/chainbench-net/... -run 'TestHandleNodeBlockNumber|TestHandleNodeStop_RejectsNonLocal' -v
```
Expected: FAIL — `undefined: newHandleNodeBlockNumber`.

- [ ] **Step 5: Implement resolveNode + handler + guards**

Edit `handlers.go`:

Add import if missing: `"github.com/0xmhha/chainbench/network/internal/drivers/remote"`.

Register handler:
```go
"node.block_number": newHandleNodeBlockNumber(stateDir),
```

Add `resolveNode` helper near `resolveNodeID`:

```go
// resolveNode resolves (networkName, nodeID) to the network + node pair.
// networkName=="" defaults to "local". Non-local names pattern-checked at the
// boundary (handler cost; state layer re-validates). Returns:
//   INVALID_ARGS  — empty/malformed node_id, bad network name pattern,
//                   node not in resolved network
//   UPSTREAM_ERROR — state.LoadActive failure (missing pids.json or
//                   networks/<name>.json)
func resolveNode(stateDir, networkName, nodeID string) (*types.Network, *types.Node, error) {
    if nodeID == "" {
        return nil, nil, NewInvalidArgs("args.node_id is required")
    }
    if networkName == "" {
        networkName = "local"
    }
    if networkName != "local" && !state.IsValidRemoteName(networkName) {
        return nil, nil, NewInvalidArgs(fmt.Sprintf("args.network must be 'local' or match [a-z0-9][a-z0-9_-]*: %q", networkName))
    }
    net, err := state.LoadActive(state.LoadActiveOptions{StateDir: stateDir, Name: networkName})
    if err != nil {
        return nil, nil, NewUpstream("failed to load network state", err)
    }
    for i := range net.Nodes {
        if net.Nodes[i].Id == nodeID {
            return net, &net.Nodes[i], nil
        }
    }
    return nil, nil, NewInvalidArgs(fmt.Sprintf("node_id %q not found in network %q", nodeID, networkName))
}
```

Add `newHandleNodeBlockNumber` near `newHandleNetworkProbe`:

```go
// newHandleNodeBlockNumber returns a handler that opens an ethclient
// connection to the resolved node's HTTP endpoint and returns the current
// head block number. Works uniformly across local and remote networks
// because both populate types.Node.Http.
//
// Args: {network?: "local"|"<remote-name>", node_id: "nodeN"}
// Returns: {network, node_id, block_number}
func newHandleNodeBlockNumber(stateDir string) Handler {
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
        networkName, node, err := resolveNodeToName(stateDir, req.Network, req.NodeID)
        if err != nil {
            return nil, err
        }
        ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
        defer cancel()
        client, err := remote.Dial(ctx, node.Http)
        if err != nil {
            return nil, NewUpstream(fmt.Sprintf("dial %s", node.Http), err)
        }
        defer client.Close()
        bn, err := client.BlockNumber(ctx)
        if err != nil {
            return nil, NewUpstream("eth_blockNumber", err)
        }
        return map[string]any{
            "network":      networkName,
            "node_id":      node.Id,
            "block_number": bn,
        }, nil
    }
}

// resolveNodeToName wraps resolveNode returning (networkName, node, err).
// Handler needs the resolved name string (post-default) for its result.
func resolveNodeToName(stateDir, networkName, nodeID string) (string, *types.Node, error) {
    if networkName == "" {
        networkName = "local"
    }
    net, node, err := resolveNode(stateDir, networkName, nodeID)
    if err != nil {
        return "", nil, err
    }
    return net.Name, node, nil
}
```

Add NOT_SUPPORTED guard to local-only handlers. Each of `node.stop/start/restart/tail_log` gets a parsed-but-ignored `Network` field in the req struct. Example pattern applied to `newHandleNodeStop`:

```go
// Inside newHandleNodeStop's returned closure, BEFORE calling resolveNodeID:
var pre struct {
    Network string `json:"network"`
}
if len(args) > 0 {
    _ = json.Unmarshal(args, &pre) // best-effort; main parse is in resolveNodeID
}
if pre.Network != "" && pre.Network != "local" {
    return nil, NewNotSupported(fmt.Sprintf("node.stop is only supported on the local network (got %q)", pre.Network))
}
```

Apply identical block to `newHandleNodeStart`, `newHandleNodeRestart`, and `newHandleNodeTailLog` (customize the command name in the message).

- [ ] **Step 6: Run handler tests**

```bash
cd network && go test ./cmd/chainbench-net/... -v
```
Expected: all PASS.

- [ ] **Step 7: Full suite regression check**

```bash
cd network && go test ./... -count=1
```

- [ ] **Step 8: Commit**

```bash
git add network/schema/command.json network/internal/types/command_gen.go network/cmd/chainbench-net/handlers.go network/cmd/chainbench-net/handlers_test.go
git commit -m "feat(network-net): node.block_number via ethclient + resolveNode

Adds the first remote-capable node command. Handler uses remote.Client
(Sprint 3b.2a Task 1) to call eth_blockNumber against the node's HTTP
URL, uniformly for local and remote networks.

New resolveNode helper absorbs M4 partially — takes an explicit network
name (defaults to 'local'). The existing resolveNodeID / resolveNodeIDFromString
remain untouched for local-only lifecycle handlers.

Local-only handlers (node.stop/start/restart/tail_log) gain a NOT_SUPPORTED
guard for non-local network args, so remote attachments can't accidentally
invoke shell-based operations."
```

---

## Task 3 — Go E2E

**Files:**
- Modify: `network/cmd/chainbench-net/e2e_test.go`

- [ ] **Step 1: Write failing E2E**

Append:

```go
func TestE2E_NodeBlockNumber_AgainstAttachedRemote(t *testing.T) {
    rpcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req struct{ Method string; ID json.RawMessage }
        _ = json.NewDecoder(r.Body).Decode(&req)
        w.Header().Set("Content-Type", "application/json")
        switch req.Method {
        case "eth_chainId":
            _, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"result":"0x1"}`))
        case "istanbul_getValidators", "wemix_getReward":
            _, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"error":{"code":-32601,"message":"nf"}}`))
        case "eth_blockNumber":
            _, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"result":"0x2a"}`))
        default:
            _, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"error":{"code":-32601,"message":"nf"}}`))
        }
    }))
    defer rpcSrv.Close()

    stateDir := t.TempDir()
    t.Setenv("CHAINBENCH_STATE_DIR", stateDir)

    // attach
    var stdout, stderr bytes.Buffer
    root := newRootCmd()
    attachCmd := fmt.Sprintf(`{"command":"network.attach","args":{"rpc_url":%q,"name":"remote-e2e"}}`, rpcSrv.URL)
    root.SetIn(strings.NewReader(attachCmd))
    root.SetOut(&stdout)
    root.SetErr(&stderr)
    root.SetArgs([]string{"run"})
    if err := root.Execute(); err != nil {
        t.Fatalf("attach: %v\nstderr: %s", err, stderr.String())
    }

    // block_number
    stdout.Reset()
    stderr.Reset()
    root2 := newRootCmd()
    bnCmd := `{"command":"node.block_number","args":{"network":"remote-e2e","node_id":"node1"}}`
    root2.SetIn(strings.NewReader(bnCmd))
    root2.SetOut(&stdout)
    root2.SetErr(&stderr)
    root2.SetArgs([]string{"run"})
    if err := root2.Execute(); err != nil {
        t.Fatalf("block_number: %v\nstderr: %s", err, stderr.String())
    }

    // Find result terminator.
    var resultLine []byte
    scanner := bufio.NewScanner(&stdout)
    for scanner.Scan() {
        line := append([]byte(nil), scanner.Bytes()...)
        if err := schema.ValidateBytes("event", line); err != nil {
            t.Fatalf("schema: %v\nline: %s", err, line)
        }
        msg, _ := wire.DecodeMessage(line)
        if _, ok := msg.(wire.ResultMessage); ok {
            resultLine = line
        }
    }
    if resultLine == nil {
        t.Fatal("no result terminator")
    }
    var res struct {
        Ok   bool           `json:"ok"`
        Data map[string]any `json:"data"`
    }
    _ = json.Unmarshal(resultLine, &res)
    if !res.Ok {
        t.Fatalf("not ok: %s", resultLine)
    }
    if res.Data["network"] != "remote-e2e" {
        t.Errorf("network = %v", res.Data["network"])
    }
    if bn, ok := res.Data["block_number"].(float64); !ok || bn != 42 {
        t.Errorf("block_number = %v (type %T), want 42", res.Data["block_number"], res.Data["block_number"])
    }
}
```

- [ ] **Step 2: Run + commit**

```bash
cd network && go test ./cmd/chainbench-net/... -run TestE2E_NodeBlockNumber -v
cd network && go test ./... -count=1
git add network/cmd/chainbench-net/e2e_test.go
git commit -m "test(network-net): E2E for node.block_number on attached remote

Attaches a mock RPC as 'remote-e2e', then calls node.block_number against
it and asserts the ethclient path surfaces eth_blockNumber correctly
through the wire terminator."
```

---

## Task 4 — Bash E2E

**Files:**
- Create: `tests/unit/tests/node-block-number.sh`

- [ ] **Step 1: Create bash test**

Use `tests/unit/tests/network-attach.sh` as template. Mock needs to also handle `eth_blockNumber`. Assertions:
- attach succeeds
- block_number for attached remote returns a numeric value (any non-empty)
- jq extract `.block_number` matches the mock's 0x2a = 42

Skeleton (adapt from network-attach.sh):
```bash
#!/usr/bin/env bash
set -euo pipefail
# ... (same harness boilerplate: SCRIPT_DIR, CHAINBENCH_DIR, TMPDIR_ROOT, STATE_DIR, binary build, ephemeral port, python mock)
# Mock must respond to:
#   eth_chainId              -> "0x1"
#   istanbul_getValidators   -> -32601
#   eth_blockNumber          -> "0x2a"
#   anything else            -> -32601

# attach
data="$(cb_net_call "network.attach" "{\"rpc_url\":\"http://127.0.0.1:${PORT}\",\"name\":\"bash-bn\"}")"
assert_eq "$(jq -r .name <<<"$data")" "bash-bn" "attach name"

# block_number
data2="$(cb_net_call "node.block_number" '{"network":"bash-bn","node_id":"node1"}')"
assert_eq "$(jq -r .block_number <<<"$data2")" "42" "block_number matches 0x2a"
assert_eq "$(jq -r .network <<<"$data2")" "bash-bn" "block_number network echo"

unit_summary
```

- [ ] **Step 2: Run + commit**

```bash
bash tests/unit/tests/node-block-number.sh
bash tests/unit/run.sh  # must stay 100%
git add tests/unit/tests/node-block-number.sh
git commit -m "test(bash): add node-block-number.sh E2E

Subprocess-level roundtrip: attach a mock RPC, then call node.block_number
and assert the ethclient path surfaces the expected block number."
```

---

## Task 5 — Final review + roadmap

- [ ] **Step 1: Full matrix**

```bash
cd network && go test ./... -count=1
cd .. && bash tests/unit/run.sh
```

- [ ] **Step 2: Update roadmap**

Edit `docs/VISION_AND_ROADMAP.md` §6 Sprint 3:

```
- [x] `drivers/remote` (read-only) + `node.block_number` — Sprint 3b.2a 완료 (2026-04-24); 인증(API key/JWT)은 3b.2b 에서
```

Keep the previous `drivers/remote` pending line replaced.

- [ ] **Step 3: Commit**

```bash
git add docs/VISION_AND_ROADMAP.md
git commit -m "docs: mark Sprint 3b.2a complete; auth tagged 3b.2b"
```

- [ ] **Step 4: Report**

Report counts, coverage, commit range, deferrals.
