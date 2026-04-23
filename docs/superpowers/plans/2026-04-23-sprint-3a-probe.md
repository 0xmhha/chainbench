# Sprint 3a — chain_type probe Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Read-only RPC probe that returns `{chain_type, chain_id, namespaces}` for an unknown EVM endpoint. Enables `network attach <url>` auto-classification in Sprint 3b.

**Architecture:** New `network/internal/probe/` package with table-driven chain signature detection via minimal JSON-RPC POST. New `network.probe` wire handler. No new schema, no new deps.

**Tech Stack:** Go 1.25, `net/http`, `encoding/json`, `net/http/httptest` for mocks, existing wire/events/errors infra.

Spec: `docs/superpowers/specs/2026-04-23-sprint-3a-probe.md`

---

## Commit Discipline

- English commit messages, no Co-Authored-By, no "Generated with Claude Code" attribution, no emoji.
- One commit per TDD RED+GREEN cycle (or per logically-scoped change for refactors).
- Test code and implementation code committed together when they land in the same cycle.

## File Structure

**Create:**
- `network/internal/probe/probe.go` — Detect() entry, Options, Result
- `network/internal/probe/rpc.go` — jsonRPCCall helper
- `network/internal/probe/signatures.go` — chain signature table
- `network/internal/probe/probe_test.go` — table-driven tests

**Modify:**
- `network/cmd/chainbench-net/handlers.go` — add newHandleNetworkProbe, register in allHandlers
- `network/cmd/chainbench-net/e2e_test.go` — add probe E2E case

**Create (bash):**
- `tests/unit/tests/network-probe.sh` — bash client test

---

## Task 1 — probe package core (RED → GREEN)

**Files:**
- Create: `network/internal/probe/probe.go`
- Create: `network/internal/probe/rpc.go`
- Create: `network/internal/probe/signatures.go`
- Create: `network/internal/probe/probe_test.go`

- [ ] **Step 1: Write failing test for `Detect` (stablenet happy path)**

Create `probe_test.go` with table-driven harness. First entry: stablenet — mock server handles `eth_chainId` → `"0x205b"` (8283), `istanbul_getValidators` → `[]`, `wemix_getReward` → method-not-found.

```go
package probe

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"
)

type rpcRequest struct {
    Method string        `json:"method"`
    Params []interface{} `json:"params"`
    ID     int           `json:"id"`
}

type rpcError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
}

type rpcResponse struct {
    Result interface{} `json:"result,omitempty"`
    Error  *rpcError   `json:"error,omitempty"`
    ID     int         `json:"id"`
}

// mockRPC returns an httptest.Server that dispatches by method name.
// handlers[method] returns (result, errCode). errCode 0 = success (result only).
// Missing method → -32601 method-not-found.
func mockRPC(t *testing.T, handlers map[string]func() (interface{}, int)) *httptest.Server {
    t.Helper()
    return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req rpcRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, err.Error(), http.StatusBadRequest)
            return
        }
        resp := rpcResponse{ID: req.ID}
        if h, ok := handlers[req.Method]; ok {
            result, code := h()
            if code == 0 {
                resp.Result = result
            } else {
                resp.Error = &rpcError{Code: code, Message: "mock error"}
            }
        } else {
            resp.Error = &rpcError{Code: -32601, Message: "method not found"}
        }
        w.Header().Set("Content-Type", "application/json")
        _ = json.NewEncoder(w).Encode(resp)
    }))
}

func TestDetect(t *testing.T) {
    cases := []struct {
        name           string
        handlers       map[string]func() (interface{}, int)
        override       string
        wantChainType  string
        wantChainID    int64
        wantNamespaces []string
        wantOverridden bool
    }{
        {
            name: "stablenet via istanbul + chain_id 8283",
            handlers: map[string]func() (interface{}, int){
                "eth_chainId":              func() (interface{}, int) { return "0x205b", 0 },
                "istanbul_getValidators":   func() (interface{}, int) { return []string{}, 0 },
            },
            wantChainType:  "stablenet",
            wantChainID:    8283,
            wantNamespaces: []string{"istanbul"},
        },
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            srv := mockRPC(t, tc.handlers)
            defer srv.Close()
            res, err := Detect(context.Background(), Options{
                RPCURL:   srv.URL,
                Timeout:  2 * time.Second,
                Override: tc.override,
            })
            if err != nil {
                t.Fatalf("Detect: %v", err)
            }
            if res.ChainType != tc.wantChainType {
                t.Errorf("ChainType = %q, want %q", res.ChainType, tc.wantChainType)
            }
            if res.ChainID != tc.wantChainID {
                t.Errorf("ChainID = %d, want %d", res.ChainID, tc.wantChainID)
            }
            if res.Overridden != tc.wantOverridden {
                t.Errorf("Overridden = %v, want %v", res.Overridden, tc.wantOverridden)
            }
        })
    }
}
```

- [ ] **Step 2: Run test to verify it fails (no Detect yet)**

```bash
cd network && go test ./internal/probe/...
```
Expected: FAIL with `undefined: Detect`.

- [ ] **Step 3: Write minimal rpc.go JSON-RPC client**

```go
// network/internal/probe/rpc.go
package probe

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
)

type jsonRPCError struct {
    Code    int    `json:"code"`
    Message string `json:"message"`
}

type jsonRPCResponse struct {
    JSONRPC string           `json:"jsonrpc"`
    ID      int              `json:"id"`
    Result  json.RawMessage  `json:"result,omitempty"`
    Error   *jsonRPCError    `json:"error,omitempty"`
}

// jsonRPCCall fires a single JSON-RPC POST and returns the Response.
// Network errors, non-200 HTTP, and malformed JSON bubble up as err.
// RPC-level errors (response.Error != nil) are returned in the response — caller decides.
func jsonRPCCall(ctx context.Context, client *http.Client, url, method string, params []interface{}) (*jsonRPCResponse, error) {
    body, err := json.Marshal(map[string]interface{}{
        "jsonrpc": "2.0",
        "id":      1,
        "method":  method,
        "params":  params,
    })
    if err != nil {
        return nil, fmt.Errorf("marshal rpc body: %w", err)
    }
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
    if err != nil {
        return nil, fmt.Errorf("build rpc request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("rpc http: %w", err)
    }
    defer resp.Body.Close()
    raw, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("read rpc body: %w", err)
    }
    if resp.StatusCode != http.StatusOK {
        return nil, fmt.Errorf("rpc http status %d: %s", resp.StatusCode, string(raw))
    }
    var out jsonRPCResponse
    if err := json.Unmarshal(raw, &out); err != nil {
        return nil, fmt.Errorf("decode rpc response: %w", err)
    }
    return &out, nil
}
```

- [ ] **Step 4: Write minimal signatures.go table**

```go
// network/internal/probe/signatures.go
package probe

// chainSignature describes how to detect one chain family.
// Order matters: evaluate top-to-bottom, first match wins.
// "istanbul_*" namespace is shared by stablenet and wbft; disambiguate via knownChainIDs.
type chainSignature struct {
    chainType     string
    namespace     string // namespace name reported in Result.Namespaces
    probeMethod   string // RPC method to hit; "" = skip (fallback rule)
    knownChainIDs map[int64]bool // non-nil = require chainID membership
}

var signatures = []chainSignature{
    {
        chainType:   "wemix",
        namespace:   "wemix",
        probeMethod: "wemix_getReward",
    },
    {
        chainType:     "stablenet",
        namespace:     "istanbul",
        probeMethod:   "istanbul_getValidators",
        knownChainIDs: map[int64]bool{8283: true},
    },
    {
        chainType:   "wbft",
        namespace:   "istanbul",
        probeMethod: "istanbul_getValidators",
    },
    // ethereum: implicit fallback, no probe.
}

// isKnownOverride returns true if the supplied override string maps to a known chain_type.
func isKnownOverride(s string) bool {
    switch s {
    case "stablenet", "wbft", "wemix", "ethereum":
        return true
    default:
        return false
    }
}
```

- [ ] **Step 5: Write minimal probe.go Detect**

```go
// network/internal/probe/probe.go
package probe

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "strconv"
    "strings"
    "time"
)

const defaultTimeout = 5 * time.Second

type Options struct {
    RPCURL   string
    Timeout  time.Duration
    Override string
    Client   *http.Client
}

type Result struct {
    ChainType  string   `json:"chain_type"`
    ChainID    int64    `json:"chain_id"`
    RPCURL     string   `json:"rpc_url"`
    Namespaces []string `json:"namespaces"`
    Overridden bool     `json:"overridden"`
    Warnings   []string `json:"warnings"`
}

// Detect probes an RPC endpoint for chain_type and chain_id.
// Returns (*Result, nil) on success; (nil, error) on unrecoverable failures.
// Caller wraps error as an APIError (INVALID_ARGS / UPSTREAM_ERROR) per context.
func Detect(ctx context.Context, opts Options) (*Result, error) {
    if opts.RPCURL == "" {
        return nil, fmt.Errorf("rpc_url required")
    }
    parsed, err := url.Parse(opts.RPCURL)
    if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
        return nil, fmt.Errorf("rpc_url must be http(s): %q", opts.RPCURL)
    }
    timeout := opts.Timeout
    if timeout <= 0 {
        timeout = defaultTimeout
    }
    client := opts.Client
    if client == nil {
        client = &http.Client{Timeout: timeout}
    }
    ctx, cancel := context.WithTimeout(ctx, timeout)
    defer cancel()

    chainID, err := fetchChainID(ctx, client, opts.RPCURL)
    if err != nil {
        return nil, fmt.Errorf("eth_chainId: %w", err)
    }

    result := &Result{
        ChainID:    chainID,
        RPCURL:     opts.RPCURL,
        Namespaces: []string{},
        Warnings:   []string{},
    }

    if opts.Override != "" {
        if !isKnownOverride(opts.Override) {
            return nil, fmt.Errorf("unknown override %q", opts.Override)
        }
        result.ChainType = opts.Override
        result.Overridden = true
        return result, nil
    }

    for _, sig := range signatures {
        if sig.probeMethod == "" {
            continue
        }
        ok, _ := probeMethod(ctx, client, opts.RPCURL, sig.probeMethod)
        if !ok {
            continue
        }
        if sig.knownChainIDs != nil && !sig.knownChainIDs[chainID] {
            continue
        }
        result.ChainType = sig.chainType
        result.Namespaces = appendUnique(result.Namespaces, sig.namespace)
        return result, nil
    }

    // Disambiguation second pass: if a non-id-gated istanbul signature exists and we
    // saw istanbul but did not match stablenet's id gate, fall through to wbft.
    for _, sig := range signatures {
        if sig.knownChainIDs != nil || sig.probeMethod == "" {
            continue
        }
        ok, _ := probeMethod(ctx, client, opts.RPCURL, sig.probeMethod)
        if !ok {
            continue
        }
        result.ChainType = sig.chainType
        result.Namespaces = appendUnique(result.Namespaces, sig.namespace)
        return result, nil
    }

    result.ChainType = "ethereum"
    return result, nil
}

func fetchChainID(ctx context.Context, client *http.Client, url string) (int64, error) {
    resp, err := jsonRPCCall(ctx, client, url, "eth_chainId", []interface{}{})
    if err != nil {
        return 0, err
    }
    if resp.Error != nil {
        return 0, fmt.Errorf("rpc error %d: %s", resp.Error.Code, resp.Error.Message)
    }
    var hex string
    if err := json.Unmarshal(resp.Result, &hex); err != nil {
        return 0, fmt.Errorf("chainId not a string: %w", err)
    }
    hex = strings.TrimPrefix(hex, "0x")
    id, err := strconv.ParseInt(hex, 16, 64)
    if err != nil {
        return 0, fmt.Errorf("chainId parse %q: %w", hex, err)
    }
    return id, nil
}

func probeMethod(ctx context.Context, client *http.Client, url, method string) (bool, *jsonRPCResponse) {
    resp, err := jsonRPCCall(ctx, client, url, method, []interface{}{})
    if err != nil {
        return false, nil
    }
    if resp.Error != nil {
        return false, resp
    }
    return true, resp
}

func appendUnique(xs []string, s string) []string {
    for _, x := range xs {
        if x == s {
            return xs
        }
    }
    return append(xs, s)
}
```

- [ ] **Step 6: Run test to verify it passes**

```bash
cd network && go test ./internal/probe/... -v
```
Expected: PASS for `TestDetect/stablenet via istanbul + chain_id 8283`.

- [ ] **Step 7: Extend test cases for full coverage**

Append to `TestDetect` cases:
```go
{
    name: "wbft via istanbul + non-stablenet chain_id",
    handlers: map[string]func() (interface{}, int){
        "eth_chainId":            func() (interface{}, int) { return "0x7a69", 0 }, // 31337
        "istanbul_getValidators": func() (interface{}, int) { return []string{}, 0 },
    },
    wantChainType:  "wbft",
    wantChainID:    31337,
    wantNamespaces: []string{"istanbul"},
},
{
    name: "wemix via wemix namespace",
    handlers: map[string]func() (interface{}, int){
        "eth_chainId":     func() (interface{}, int) { return "0x3e9", 0 }, // 1001
        "wemix_getReward": func() (interface{}, int) { return "0x0", 0 },
    },
    wantChainType:  "wemix",
    wantChainID:    1001,
    wantNamespaces: []string{"wemix"},
},
{
    name: "ethereum fallback",
    handlers: map[string]func() (interface{}, int){
        "eth_chainId": func() (interface{}, int) { return "0x1", 0 },
    },
    wantChainType:  "ethereum",
    wantChainID:    1,
    wantNamespaces: []string{},
},
{
    name: "override stablenet short-circuits",
    handlers: map[string]func() (interface{}, int){
        "eth_chainId": func() (interface{}, int) { return "0x1", 0 }, // mismatched id
    },
    override:       "stablenet",
    wantChainType:  "stablenet",
    wantChainID:    1,
    wantNamespaces: []string{},
    wantOverridden: true,
},
```

- [ ] **Step 8: Add negative tests**

Separate test functions (not table) for error paths:
```go
func TestDetect_EthChainIDFails(t *testing.T) {
    srv := mockRPC(t, map[string]func() (interface{}, int){
        "eth_chainId": func() (interface{}, int) { return nil, -32000 },
    })
    defer srv.Close()
    _, err := Detect(context.Background(), Options{RPCURL: srv.URL, Timeout: time.Second})
    if err == nil {
        t.Fatal("expected error for eth_chainId RPC error")
    }
}

func TestDetect_RejectsNonHTTP(t *testing.T) {
    _, err := Detect(context.Background(), Options{RPCURL: "ws://x", Timeout: time.Second})
    if err == nil {
        t.Fatal("expected error for non-http scheme")
    }
}

func TestDetect_UnknownOverride(t *testing.T) {
    srv := mockRPC(t, map[string]func() (interface{}, int){
        "eth_chainId": func() (interface{}, int) { return "0x1", 0 },
    })
    defer srv.Close()
    _, err := Detect(context.Background(), Options{RPCURL: srv.URL, Override: "fakechain"})
    if err == nil {
        t.Fatal("expected error for unknown override")
    }
}
```

- [ ] **Step 9: Run full probe suite**

```bash
cd network && go test ./internal/probe/... -v -cover
```
Expected: all tests PASS, coverage ≥85%.

- [ ] **Step 10: Commit**

```bash
git add network/internal/probe/
git commit -m "feat(network/probe): add chain_type probe package

Table-driven detection using eth_chainId + namespace probes
(istanbul_getValidators, wemix_getReward). Distinguishes stablenet
from wbft via known chain_id set. Falls back to ethereum.
Supports user-provided chain_type override.

No go-ethereum dep; uses net/http + encoding/json directly."
```

---

## Task 2 — network.probe handler in chainbench-net

**Files:**
- Modify: `network/cmd/chainbench-net/handlers.go`

- [ ] **Step 1: Add failing unit test for handler** (in `cmd/chainbench-net/handlers_test.go` — create if missing)

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/0xmhha/chainbench/network/internal/events"
)

func TestHandleNetworkProbe_StablenetOK(t *testing.T) {
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req struct{ Method string `json:"method"` }
        _ = json.NewDecoder(r.Body).Decode(&req)
        w.Header().Set("Content-Type", "application/json")
        switch req.Method {
        case "eth_chainId":
            fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":"0x205b"}`)
        case "istanbul_getValidators":
            fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":[]}`)
        default:
            fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"not found"}}`)
        }
    }))
    defer srv.Close()

    h := newHandleNetworkProbe()
    args, _ := json.Marshal(map[string]interface{}{"rpc_url": srv.URL})
    bus, _ := events.NewBusForTest()
    data, err := h(args, bus)
    if err != nil {
        t.Fatalf("handler err: %v", err)
    }
    if data["chain_type"] != "stablenet" {
        t.Errorf("chain_type = %v, want stablenet", data["chain_type"])
    }
    if int64(data["chain_id"].(float64)) != 8283 {
        t.Errorf("chain_id = %v, want 8283", data["chain_id"])
    }
}

func TestHandleNetworkProbe_MissingURL(t *testing.T) {
    h := newHandleNetworkProbe()
    bus, _ := events.NewBusForTest()
    _, err := h(json.RawMessage(`{}`), bus)
    if err == nil {
        t.Fatal("expected INVALID_ARGS error")
    }
}
```

- [ ] **Step 2: Check if events.NewBusForTest exists; if not, use existing Bus constructor**

```bash
cd network && grep -n "func NewBus" internal/events/*.go
```
If `NewBusForTest` not present, substitute with the existing constructor in the test (e.g., `events.NewBus(nil)` or equivalent). Adjust test to match actual API.

- [ ] **Step 3: Run test to verify it fails**

```bash
cd network && go test ./cmd/chainbench-net/... -run TestHandleNetworkProbe -v
```
Expected: FAIL — `newHandleNetworkProbe undefined`.

- [ ] **Step 4: Implement handler**

Modify `network/cmd/chainbench-net/handlers.go`:

```go
// Add to imports:
// "time"
// "github.com/0xmhha/chainbench/network/internal/probe"

// Add to allHandlers() map:
//   "network.probe": newHandleNetworkProbe(),

// New function:
func newHandleNetworkProbe() Handler {
    return func(args json.RawMessage, bus *events.Bus) (map[string]any, error) {
        var req struct {
            RPCURL    string `json:"rpc_url"`
            TimeoutMs *int   `json:"timeout_ms"`
            Override  string `json:"override"`
        }
        if len(args) > 0 {
            if err := json.Unmarshal(args, &req); err != nil {
                return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
            }
        }
        if req.RPCURL == "" {
            return nil, NewInvalidArgs("args.rpc_url is required")
        }
        opts := probe.Options{
            RPCURL:   req.RPCURL,
            Override: req.Override,
        }
        if req.TimeoutMs != nil {
            if *req.TimeoutMs < 100 || *req.TimeoutMs > 60000 {
                return nil, NewInvalidArgs(fmt.Sprintf("args.timeout_ms must be 100..60000, got %d", *req.TimeoutMs))
            }
            opts.Timeout = time.Duration(*req.TimeoutMs) * time.Millisecond
        }

        result, err := probe.Detect(context.Background(), opts)
        if err != nil {
            // probe package exposes sentinel errors for input-validation failures.
            // errors.Is classification (via probe.IsInputError) avoids fragile
            // substring matching on error messages.
            if probe.IsInputError(err) {
                return nil, NewInvalidArgs(err.Error())
            }
            return nil, NewUpstream("probe failed", err)
        }

        raw, err := json.Marshal(result)
        if err != nil {
            return nil, NewInternal("marshal result", err)
        }
        var data map[string]any
        if err := json.Unmarshal(raw, &data); err != nil {
            return nil, NewInternal("unmarshal result", err)
        }
        return data, nil
    }
}
```

- [ ] **Step 5: Register handler in allHandlers**

Edit the `allHandlers` map literal to add:
```go
"network.probe": newHandleNetworkProbe(),
```

- [ ] **Step 6: Run handler tests**

```bash
cd network && go test ./cmd/chainbench-net/... -run TestHandleNetworkProbe -v
```
Expected: PASS.

- [ ] **Step 7: Run full Go suite to catch regressions**

```bash
cd network && go test ./... -count=1
```
Expected: all PASS.

- [ ] **Step 8: Commit**

```bash
git add network/cmd/chainbench-net/handlers.go network/cmd/chainbench-net/handlers_test.go
git commit -m "feat(network-net): wire network.probe handler to probe package

Accepts {rpc_url, timeout_ms?, override?}, returns probe.Result as
a plain map. Maps probe-level errors to INVALID_ARGS (caller input)
or UPSTREAM_ERROR (endpoint failure)."
```

---

## Task 3 — E2E test via wire protocol

**Files:**
- Modify: `network/cmd/chainbench-net/e2e_test.go`

- [ ] **Step 1: Read existing e2e_test.go structure**

```bash
cd network && head -80 cmd/chainbench-net/e2e_test.go
```
Identify helper used to spawn the binary and feed stdin. Plan will use the same helper.

- [ ] **Step 2: Add failing E2E test**

Append to `e2e_test.go` (names match existing patterns):

```go
func TestE2E_NetworkProbe(t *testing.T) {
    rpcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        var req struct{ Method string `json:"method"` }
        _ = json.NewDecoder(r.Body).Decode(&req)
        w.Header().Set("Content-Type", "application/json")
        switch req.Method {
        case "eth_chainId":
            fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":"0x205b"}`)
        case "istanbul_getValidators":
            fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":[]}`)
        default:
            fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"nf"}}`)
        }
    }))
    defer rpcSrv.Close()

    // Use the existing spawn helper (adapt signature as needed by reading current code).
    cmd := fmt.Sprintf(`{"command":"network.probe","args":{"rpc_url":%q}}`, rpcSrv.URL)
    out := runCLIOnce(t, cmd) // assumes existing helper; rename if different
    result := parseResultTerminator(t, out)

    if !result.Ok {
        t.Fatalf("result.ok = false: %+v", result.Error)
    }
    if result.Data["chain_type"] != "stablenet" {
        t.Errorf("chain_type = %v, want stablenet", result.Data["chain_type"])
    }
}
```

Adjust `runCLIOnce` / `parseResultTerminator` to match whatever the existing e2e file uses
(read the file first; do not invent names).

- [ ] **Step 3: Run E2E**

```bash
cd network && go test ./cmd/chainbench-net/... -run TestE2E_NetworkProbe -v
```

- [ ] **Step 4: Commit**

```bash
git add network/cmd/chainbench-net/e2e_test.go
git commit -m "test(network-net): E2E probe test via wire protocol

Spawns chainbench-net, feeds network.probe command via stdin,
validates NDJSON result terminator against a httptest mock RPC
serving stablenet-shaped responses."
```

---

## Task 4 — bash client test

**Files:**
- Create: `tests/unit/tests/network-probe.sh`

- [ ] **Step 1: Read an existing bash test to match style**

```bash
ls tests/unit/tests/ | head -20
cat tests/unit/tests/network-wire-protocol.sh | head -60
```

- [ ] **Step 2: Create `network-probe.sh`**

Use the same structure as `network-wire-protocol.sh`:
- Build or locate chainbench-net binary.
- Launch a python one-shot HTTP server dispatching on request body `method` field.
- Call `cb_net_call "network.probe" '{"rpc_url":"http://127.0.0.1:PORT"}'`.
- Assert `chain_type`, `chain_id` via jq.

```bash
#!/usr/bin/env bash
# tests/unit/tests/network-probe.sh - network.probe wire-protocol test
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${SCRIPT_DIR}/lib/assert.sh"

CHAINBENCH_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
export CHAINBENCH_DIR

# Ensure chainbench-net is built (reuse pattern from network-wire-protocol.sh).
# Launch mock RPC server on an ephemeral port.
PORT="$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')"

python3 - "$PORT" <<'PYEOF' &
import sys, json, http.server, socketserver
port = int(sys.argv[1])
class H(http.server.BaseHTTPRequestHandler):
    def do_POST(self):
        n = int(self.headers.get('Content-Length', 0))
        req = json.loads(self.rfile.read(n))
        method = req.get('method')
        if method == 'eth_chainId':
            body = {'jsonrpc':'2.0','id':req.get('id'),'result':'0x205b'}
        elif method == 'istanbul_getValidators':
            body = {'jsonrpc':'2.0','id':req.get('id'),'result':[]}
        else:
            body = {'jsonrpc':'2.0','id':req.get('id'),'error':{'code':-32601,'message':'nf'}}
        raw = json.dumps(body).encode()
        self.send_response(200)
        self.send_header('Content-Type','application/json')
        self.send_header('Content-Length', str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)
    def log_message(self, *a, **k): pass
with socketserver.TCPServer(("127.0.0.1", port), H) as srv:
    srv.serve_forever()
PYEOF
MOCK_PID=$!
trap 'kill $MOCK_PID 2>/dev/null || true' EXIT
sleep 0.3

source "${CHAINBENCH_DIR}/lib/network_client.sh"

describe "network.probe: returns stablenet + chain_id 8283"
data="$(cb_net_call "network.probe" "{\"rpc_url\":\"http://127.0.0.1:${PORT}\"}")"
assert_eq "$(jq -r .chain_type <<<"$data")" "stablenet" "chain_type"
assert_eq "$(jq -r .chain_id <<<"$data")"   "8283"      "chain_id"

unit_summary
```

- [ ] **Step 3: Run test**

```bash
bash tests/unit/run.sh network-probe
```
Expected: PASS (2 assertions).

- [ ] **Step 4: Commit**

```bash
git add tests/unit/tests/network-probe.sh
git commit -m "test(bash): add network-probe.sh unit test

Spawns python JSON-RPC mock, calls cb_net_call network.probe,
asserts chain_type and chain_id via jq."
```

---

## Task 5 — Final review + roadmap update

- [ ] **Step 1: Re-run full test matrix**

```bash
cd network && go test ./... -count=1
cd .. && bash tests/unit/run.sh
```
Expected: Go suite green; bash unit suite 100% pass.

- [ ] **Step 2: Update VISION_AND_ROADMAP.md**

In §6 Sprint 3 checklist, mark the probe item:
```
- [x] `probe` 패키지 + `network probe <url>` 커맨드 (S7 자동+수동)
```

(Leave remote and adapter items unchecked — 3b and 3c.)

- [ ] **Step 3: Commit roadmap update**

```bash
git add docs/VISION_AND_ROADMAP.md
git commit -m "docs: mark Sprint 3a probe complete in roadmap"
```

- [ ] **Step 4: Report summary**

Report to controller:
- Files created/modified (list)
- Test counts (Go + bash)
- Coverage for probe package
- Any deferrals surfaced during implementation
