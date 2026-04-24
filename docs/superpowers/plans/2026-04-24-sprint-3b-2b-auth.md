# Sprint 3b.2b — Remote RPC Auth (Minimal) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Scaffold API-key / JWT auth on `remote.Client`. Wire through `network.attach` (persist Node.Auth) and `node.block_number` (use auth when Node.Auth present). Auth material stays in env vars, never in state files or logs.

**Architecture:** `http.RoundTripper` wrapping around the default HTTP transport; injected via `rpc.WithHTTPClient` into ethclient's dial path. `AuthFromNode(node, envLookup)` reads `types.Node.Auth` + env and returns a RoundTripper.

**Tech Stack:** Go 1.25, existing ethclient infra, new `network/internal/drivers/remote/auth.go`.

Spec: `docs/superpowers/specs/2026-04-24-sprint-3b-2b-auth.md`

---

## Commit Discipline

- English, no Co-Authored-By, no "Generated with Claude Code", no emoji.

## File Structure

**Create:**
- `network/internal/drivers/remote/auth.go`
- `network/internal/drivers/remote/auth_test.go`
- `tests/unit/tests/node-block-number-auth.sh`

**Modify:**
- `network/internal/drivers/remote/client.go` — add `DialWithOptions`, have `Dial` delegate
- `network/internal/drivers/remote/client_test.go` — add `DialWithOptions` Transport test
- `network/cmd/chainbench-net/handlers.go` — `network.attach` accepts `auth` arg; `node.block_number` uses `AuthFromNode`
- `network/cmd/chainbench-net/handlers_test.go` — auth roundtrip test
- `network/cmd/chainbench-net/e2e_test.go` — auth E2E
- `docs/VISION_AND_ROADMAP.md` — mark 3b.2b complete

---

## Task 1 — remote.auth + DialWithOptions

**Files:**
- Create: `network/internal/drivers/remote/auth.go`
- Create: `network/internal/drivers/remote/auth_test.go`
- Modify: `network/internal/drivers/remote/client.go`
- Modify: `network/internal/drivers/remote/client_test.go`

- [ ] **Step 1: Write failing auth_test.go**

```go
package remote

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/network/internal/types"
)

func TestAPIKeyTransport_InjectsHeader(t *testing.T) {
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("X-Api-Key")
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	rt := APIKeyTransport(nil, "X-Api-Key", "secret123")
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()
	if gotHeader != "secret123" {
		t.Errorf("X-Api-Key = %q, want secret123", gotHeader)
	}
}

func TestBearerTokenTransport_InjectsAuthorization(t *testing.T) {
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("Authorization")
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	rt := BearerTokenTransport(nil, "eyJabc.def")
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip: %v", err)
	}
	resp.Body.Close()
	if gotHeader != "Bearer eyJabc.def" {
		t.Errorf("Authorization = %q, want Bearer eyJabc.def", gotHeader)
	}
}

func TestAuthFromNode_NilAuthReturnsNil(t *testing.T) {
	node := &types.Node{Id: "node1", Http: "http://x"}
	rt, err := AuthFromNode(node, func(string) string { return "" })
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if rt != nil {
		t.Errorf("expected nil RoundTripper for nil Auth, got %T", rt)
	}
}

func TestAuthFromNode_APIKey(t *testing.T) {
	// types.Auth is map[string]interface{} (no typed union from generator).
	node := &types.Node{Id: "node1", Http: "http://x", Auth: types.Auth{"type": "api-key", "env": "TEST_KEY"}}
	envs := map[string]string{"TEST_KEY": "abc"}
	rt, err := AuthFromNode(node, func(k string) string { return envs[k] })
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if rt == nil {
		t.Fatal("expected non-nil RoundTripper")
	}

	// Exercise it.
	var gotHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotHeader = r.Header.Get("Authorization")
		w.Write([]byte("ok"))
	}))
	defer srv.Close()
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if gotHeader != "abc" {
		t.Errorf("got header %q, want abc", gotHeader)
	}
}

func TestAuthFromNode_EmptyEnvIsError(t *testing.T) {
	node := &types.Node{Id: "node1", Http: "http://x", Auth: types.Auth{"type": "api-key", "env": "MISSING_KEY"}}
	_, err := AuthFromNode(node, func(string) string { return "" })
	if err == nil {
		t.Fatal("expected error for empty env value")
	}
	// Error must mention the env name but never contain a fake value we could
	// mistake for leaked material.
	if !strings.Contains(err.Error(), "MISSING_KEY") {
		t.Errorf("err should reference env name: %v", err)
	}
}
```

**Note on types**: `types.Auth` is `map[string]interface{}` (go-jsonschema
did NOT emit a tagged union for the oneOf; it fell back to a loose map).
Tests construct it as `types.Auth{"type": "api-key", "env": "KEY_NAME"}`
directly. Node.Auth field is `Auth` (not a pointer), and is nil-safe on
`len(node.Auth) == 0`.

- [ ] **Step 2: Run test to verify it fails**

```bash
cd network && go test ./internal/drivers/remote/... -run 'TestAPIKey|TestBearer|TestAuthFromNode' -v
```

Expected: compile failure due to undefined `APIKeyTransport`, `BearerTokenTransport`, `AuthFromNode`.

- [ ] **Step 3: Confirm types.Auth shape (already known)**

```bash
grep -n "type Auth\|Auth " network/internal/types/network_gen.go
```

Expected output confirms `type Auth map[string]interface{}`. No helper
constructors to find — tests use map literals directly.

- [ ] **Step 4: Implement auth.go**

**Important context on the generated types.Auth**: go-jsonschema did NOT emit a
tagged union for the `Auth` oneOf in `network.json`. Instead, `network_gen.go`
declares `type Auth map[string]interface{}` — a loose map keyed by JSON field
names (`type`, `env`, `header`, `user`, etc.). `types.Node.Auth` is that map.
No `Auth0 / Auth1 / NodeAuthFromApiKey` helpers exist. Branch on
`auth["type"].(string)` directly.

```go
// Package remote — auth helpers.
package remote

import (
	"fmt"
	"net/http"

	"github.com/0xmhha/chainbench/network/internal/types"
)

// headerTransport is a RoundTripper that clones the request and sets a header
// before delegating. Keeping this private forces callers through the typed
// constructors below (APIKeyTransport / BearerTokenTransport).
type headerTransport struct {
	base   http.RoundTripper
	header string
	value  string
}

func (t *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone per Go net/http contract — RoundTrippers must not mutate the
	// request they receive.
	clone := req.Clone(req.Context())
	clone.Header.Set(t.header, t.value)
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(clone)
}

// APIKeyTransport wraps base to inject "<header>: <value>" on every request.
// Passing nil for base uses http.DefaultTransport.
func APIKeyTransport(base http.RoundTripper, header, value string) http.RoundTripper {
	if header == "" {
		header = "Authorization"
	}
	return &headerTransport{base: base, header: header, value: value}
}

// BearerTokenTransport wraps base to inject "Authorization: Bearer <token>".
func BearerTokenTransport(base http.RoundTripper, token string) http.RoundTripper {
	return &headerTransport{base: base, header: "Authorization", value: "Bearer " + token}
}

// AuthFromNode reads node.Auth (generated as map[string]any by go-jsonschema)
// and returns a RoundTripper matching the configured type. Returns (nil, nil)
// when node.Auth is empty (unauthenticated). envLookup is injected for
// testability; production callers pass os.Getenv.
//
// Auth material never appears in returned errors. If the env var is unset or
// empty, the error references the variable name only.
func AuthFromNode(node *types.Node, envLookup func(string) string) (http.RoundTripper, error) {
	if node == nil || len(node.Auth) == 0 {
		return nil, nil
	}
	rawType, ok := node.Auth["type"].(string)
	if !ok || rawType == "" {
		return nil, fmt.Errorf("remote.AuthFromNode: missing or non-string 'type' field")
	}
	switch rawType {
	case "api-key":
		envName, _ := node.Auth["env"].(string)
		if envName == "" {
			return nil, fmt.Errorf("remote.AuthFromNode(api-key): 'env' field is required")
		}
		value := envLookup(envName)
		if value == "" {
			return nil, fmt.Errorf("remote.AuthFromNode(api-key): env var %q is empty", envName)
		}
		header, _ := node.Auth["header"].(string) // optional; APIKeyTransport defaults
		return APIKeyTransport(nil, header, value), nil
	case "jwt":
		envName, _ := node.Auth["env"].(string)
		if envName == "" {
			return nil, fmt.Errorf("remote.AuthFromNode(jwt): 'env' field is required")
		}
		token := envLookup(envName)
		if token == "" {
			return nil, fmt.Errorf("remote.AuthFromNode(jwt): env var %q is empty", envName)
		}
		return BearerTokenTransport(nil, token), nil
	case "ssh-password":
		// SSH auth belongs to SSHRemoteDriver (future), not RPC client.
		return nil, fmt.Errorf("remote.AuthFromNode: 'ssh-password' auth not applicable to RPC client")
	default:
		return nil, fmt.Errorf("remote.AuthFromNode: unknown auth type %q", rawType)
	}
}
```

Corresponding tests in `auth_test.go` should use map literals directly:

```go
func TestAuthFromNode_APIKey(t *testing.T) {
	node := &types.Node{
		Id: "node1", Http: "http://x",
		Auth: types.Auth{"type": "api-key", "env": "TEST_KEY"},
	}
	envs := map[string]string{"TEST_KEY": "abc"}
	rt, err := AuthFromNode(node, func(k string) string { return envs[k] })
	// ... etc. Adapt Step 1's test block to use types.Auth{...} map literals.
}
```

- [ ] **Step 5: Implement DialWithOptions**

Edit `client.go`:

```go
import "net/http"

type DialOptions struct {
	Transport http.RoundTripper
}

// Dial is equivalent to DialWithOptions(ctx, url, DialOptions{}).
func Dial(ctx context.Context, url string) (*Client, error) {
	return DialWithOptions(ctx, url, DialOptions{})
}

// DialWithOptions opens an ethclient-backed Client with optional transport
// injection (used for auth). Passing a zero DialOptions is equivalent to Dial.
func DialWithOptions(ctx context.Context, url string, opts DialOptions) (*Client, error) {
	// If no Transport override, use ethclient.DialContext directly (simpler).
	if opts.Transport == nil {
		rpc, err := ethclient.DialContext(ctx, url)
		if err != nil {
			return nil, fmt.Errorf("remote.Dial %q: %w", url, err)
		}
		return &Client{rpc: rpc}, nil
	}
	// Custom transport path: build an *http.Client and wire via rpc.DialOptions.
	httpClient := &http.Client{Transport: opts.Transport}
	rpcClient, err := rpc.DialOptions(ctx, url, rpc.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("remote.DialWithOptions %q: %w", url, err)
	}
	return &Client{rpc: ethclient.NewClient(rpcClient)}, nil
}
```

Add imports: `"github.com/ethereum/go-ethereum/rpc"` and `"net/http"`.

- [ ] **Step 6: Add DialWithOptions test in client_test.go**

```go
func TestDialWithOptions_TransportInjected(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("X-Test-Auth")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x5"}`))
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	rt := APIKeyTransport(nil, "X-Test-Auth", "mykey")
	c, err := DialWithOptions(ctx, srv.URL, DialOptions{Transport: rt})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	bn, err := c.BlockNumber(ctx)
	if err != nil {
		t.Fatalf("BlockNumber: %v", err)
	}
	if bn != 5 {
		t.Errorf("bn = %d, want 5", bn)
	}
	if gotAuth != "mykey" {
		t.Errorf("header not injected: %q", gotAuth)
	}
}
```

- [ ] **Step 7: Run tests, fix AuthFromNode union implementation**

```bash
cd network && go test ./internal/drivers/remote/... -v
```

Iterate on `AuthFromNode` until all cases pass. Likely shape based on how
go-jsonschema emits oneOf unions — you may see a struct with discriminator
fields populated via a generated `AsX() / IsX()` helper, or separate
`auth.Type` + type-specific fields. Whichever exists, branch on it.

- [ ] **Step 8: Run full Go suite**

```bash
cd network && go test ./... -count=1 -timeout=60s
go -C network vet ./...
gofmt -l network/
```

- [ ] **Step 9: Commit**

```bash
git add network/internal/drivers/remote/
git commit -m "feat(drivers/remote): add API key / JWT auth via RoundTripper

New remote/auth.go introduces APIKeyTransport, BearerTokenTransport,
and AuthFromNode helpers. remote.DialWithOptions accepts a custom
http.RoundTripper, wired via rpc.WithHTTPClient into the ethclient
dial path. remote.Dial is now a thin wrapper over DialWithOptions
with zero options (unchanged behavior for existing callers).

Auth material stays in env vars; AuthFromNode takes the env var name
from types.Node.Auth and resolves via an injected lookup function.
Empty env values produce an error that references the variable NAME
only — never the value. Errors do not leak to stdout/stderr per the
existing slog boundary."
```

---

## Task 2 — network.attach accepts auth + node.block_number uses it

**Files:**
- Modify: `network/cmd/chainbench-net/handlers.go`
- Modify: `network/cmd/chainbench-net/handlers_test.go`

- [ ] **Step 1: Write failing test — attach with auth persists it**

Append to `handlers_test.go`:

```go
func TestHandleNetworkAttach_WithAuth(t *testing.T) {
	rpcSrv := newStablenetMockRPC(t)
	defer rpcSrv.Close()

	stateDir := t.TempDir()
	h := newHandleNetworkAttach(stateDir)
	args, _ := json.Marshal(map[string]any{
		"rpc_url": rpcSrv.URL,
		"name":    "protected",
		"auth": map[string]any{
			"type":   "api-key",
			"header": "X-Api-Key",
			"env":    "MY_KEY",
		},
	})
	bus, _ := newTestBus(t)
	_, err := h(args, bus)
	if err != nil {
		t.Fatalf("attach: %v", err)
	}
	// Verify state file has Auth populated but NOT the raw key.
	raw, err := os.ReadFile(filepath.Join(stateDir, "networks", "protected.json"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "\"env\": \"MY_KEY\"") && !strings.Contains(s, `"env":"MY_KEY"`) {
		t.Errorf("auth.env not persisted: %s", s)
	}
}
```

- [ ] **Step 2: Extend `newHandleNetworkAttach` to accept `auth`**

Inside the handler closure:

```go
var req struct {
    RPCURL   string          `json:"rpc_url"`
    Name     string          `json:"name"`
    Override string          `json:"override"`
    Auth     json.RawMessage `json:"auth"` // optional; raw passthrough to types.NodeAuth
}
```

After probe + network construction but before save, decode `req.Auth` into
`types.NodeAuth` (use the same Unmarshal trick the generated union supports).
Attach it to `net.Nodes[0].Auth`.

Pseudocode:
```go
if len(req.Auth) > 0 {
    var auth types.NodeAuth
    if err := json.Unmarshal(req.Auth, &auth); err != nil {
        return nil, NewInvalidArgs(fmt.Sprintf("args.auth: %v", err))
    }
    net.Nodes[0].Auth = &auth
}
```

Inspect the generated `NodeAuth` type first — if it's a pointer-union with
a marker-interface, the Unmarshal path may look different.

- [ ] **Step 3: Write failing test — block_number uses auth**

```go
func TestHandleNodeBlockNumber_UsesAuth(t *testing.T) {
    var gotKey string
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        gotKey = r.Header.Get("X-Api-Key")
        var req struct{ Method string; ID json.RawMessage }
        _ = json.NewDecoder(r.Body).Decode(&req)
        w.Header().Set("Content-Type", "application/json")
        if req.Method == "eth_blockNumber" {
            _, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"result":"0x7"}`))
            return
        }
        _, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"error":{"code":-32601,"message":"nf"}}`))
    }))
    defer srv.Close()

    stateDir := t.TempDir()
    // Save a network with Auth set.
    net := &types.Network{
        Name: "protected", ChainType: "ethereum", ChainId: 1,
        Nodes: []types.Node{{Id: "node1", Provider: "remote", Http: srv.URL,
            /* Auth: set via test helper matching generated union shape */}},
    }
    if err := state.SaveRemote(stateDir, net); err != nil { t.Fatal(err) }

    t.Setenv("TEST_AUTH_KEY", "mysecret")

    h := newHandleNodeBlockNumber(stateDir)
    args, _ := json.Marshal(map[string]any{"network": "protected", "node_id": "node1"})
    bus, _ := newTestBus(t)
    _, err := h(args, bus)
    if err != nil {
        t.Fatalf("handler: %v", err)
    }
    if gotKey != "mysecret" {
        t.Errorf("expected auth header injected, got %q", gotKey)
    }
}
```

Fill in the Auth field using whatever constructor helper the generator produces.

- [ ] **Step 4: Modify `newHandleNodeBlockNumber` to use AuthFromNode**

```go
var rt http.RoundTripper
if node.Auth != nil {
    got, err := remote.AuthFromNode(&node, os.Getenv)
    if err != nil {
        return nil, NewUpstream("auth setup", err)
    }
    rt = got
}
client, err := remote.DialWithOptions(ctx, node.Http, remote.DialOptions{Transport: rt})
if err != nil {
    return nil, NewUpstream(fmt.Sprintf("dial %s", node.Http), err)
}
```

Add imports: `"net/http"`, `"os"` (likely already imported).

- [ ] **Step 5: Run both tests + full suite**

```bash
cd network && go test ./cmd/chainbench-net/... -run 'TestHandleNetworkAttach_WithAuth|TestHandleNodeBlockNumber_UsesAuth' -v
cd network && go test ./... -count=1 -timeout=60s
```

- [ ] **Step 6: Commit**

```bash
git add network/cmd/chainbench-net/handlers.go network/cmd/chainbench-net/handlers_test.go
git commit -m "feat(network-net): network.attach accepts auth; node.block_number honors it

network.attach now accepts an optional 'auth' object matching the
network.json Auth schema. The raw JSON is attached to Node.Auth and
persisted. State file holds only the env var NAME, never the value.

node.block_number uses remote.AuthFromNode when the resolved node
has Auth set, constructing an injected RoundTripper that adds the
required header (API key) or Bearer token (JWT). Unauthenticated
nodes take the existing bare Dial path."
```

---

## Task 3 — E2E coverage

**Files:**
- Modify: `network/cmd/chainbench-net/e2e_test.go`
- Create: `tests/unit/tests/node-block-number-auth.sh`

- [ ] **Step 1: Go E2E**

Append to `e2e_test.go`:

```go
func TestE2E_NodeBlockNumber_WithAuth(t *testing.T) {
    var gotAuth string
    rpcSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        gotAuth = r.Header.Get("X-Api-Key")
        var req struct{ Method string `json:"method"`; ID json.RawMessage `json:"id"` }
        _ = json.NewDecoder(r.Body).Decode(&req)
        w.Header().Set("Content-Type", "application/json")
        switch req.Method {
        case "eth_chainId":
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x1"}`, req.ID)
        case "eth_blockNumber":
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":"0x99"}`, req.ID)
        case "istanbul_getValidators", "wemix_getReward":
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
        default:
            fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32601,"message":"nf"}}`, req.ID)
        }
    }))
    defer rpcSrv.Close()

    stateDir := t.TempDir()
    t.Setenv("CHAINBENCH_STATE_DIR", stateDir)
    t.Setenv("AUTH_E2E_KEY", "e2e-secret")

    // attach with auth
    var stdout, stderr bytes.Buffer
    root := newRootCmd()
    attachCmd := fmt.Sprintf(
        `{"command":"network.attach","args":{"rpc_url":%q,"name":"auth-e2e","auth":{"type":"api-key","header":"X-Api-Key","env":"AUTH_E2E_KEY"}}}`,
        rpcSrv.URL,
    )
    root.SetIn(strings.NewReader(attachCmd))
    root.SetOut(&stdout); root.SetErr(&stderr); root.SetArgs([]string{"run"})
    if err := root.Execute(); err != nil {
        t.Fatalf("attach: %v stderr=%s", err, stderr.String())
    }

    // block_number
    stdout.Reset(); stderr.Reset()
    root2 := newRootCmd()
    root2.SetIn(strings.NewReader(`{"command":"node.block_number","args":{"network":"auth-e2e","node_id":"node1"}}`))
    root2.SetOut(&stdout); root2.SetErr(&stderr); root2.SetArgs([]string{"run"})
    if err := root2.Execute(); err != nil {
        t.Fatalf("block_number: %v stderr=%s", err, stderr.String())
    }

    // ... parse result terminator as in prior E2Es ...
    // assert result.ok == true, block_number == 153 (0x99), gotAuth == "e2e-secret"
    if gotAuth != "e2e-secret" {
        t.Errorf("server didn't see the key: %q", gotAuth)
    }
}
```

Flesh out the terminator parsing per the existing pattern in this file.

- [ ] **Step 2: Bash test**

Create `tests/unit/tests/node-block-number-auth.sh` matching the style of
`tests/unit/tests/node-block-number.sh`. The mock must require
`Authorization: <value>` where value comes from an env var you set before
calling cb_net_call. Attach with `auth` arg, then block_number.

- [ ] **Step 3: Run + commit**

```bash
cd network && go test ./cmd/chainbench-net/... -run TestE2E_NodeBlockNumber_WithAuth -v
bash tests/unit/tests/node-block-number-auth.sh
bash tests/unit/run.sh

git add network/cmd/chainbench-net/e2e_test.go tests/unit/tests/node-block-number-auth.sh
git commit -m "test: E2E + bash coverage for auth on node.block_number

Go E2E attaches a mock RPC with api-key auth (env-injected), then
calls node.block_number and asserts the server receives the configured
header. Bash test covers the subprocess path with the same roundtrip."
```

---

## Task 4 — Final review + roadmap

- [ ] **Step 1: Full matrix + commit roadmap update**

```bash
cd network && go test ./... -count=1 -timeout=60s
cd .. && bash tests/unit/run.sh
```

Edit `docs/VISION_AND_ROADMAP.md` Sprint 3 entry — mark auth complete:

```
- [x] Remote auth (API key / JWT via RoundTripper) — Sprint 3b.2b 완료 (2026-04-24)
```

Add line under the 3b.2a entry.

```bash
git add docs/VISION_AND_ROADMAP.md
git commit -m "docs: mark Sprint 3b.2b complete — remote auth scaffolding"
```

- [ ] **Step 2: Report**

Test counts, commit range, auth type coverage (api-key + jwt), any deferrals.
