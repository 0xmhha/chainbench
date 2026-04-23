# Sprint 3b — Remote Network Attach Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**Goal:** Ship `network.attach <url>` — reuses Sprint 3a probe to classify an unknown EVM endpoint, persists it as a named remote network in `state/networks/<name>.json`, and extends `network.load` to resolve any name (not just "local").

**Architecture:** New `state.SaveRemote` + atomic file write; extended `state.LoadActive` routing on Name; new `newHandleNetworkAttach` handler composing probe + state. No new third-party deps. No driver, no auth, no resolveNodeID refactor — those arrive in 3b.2 with the first remote-node command.

**Tech Stack:** Go 1.25, existing probe/wire/events/state/types packages, `encoding/json`.

Spec: `docs/superpowers/specs/2026-04-23-sprint-3b-attach.md`

---

## Commit Discipline

- English commit messages, no Co-Authored-By, no "Generated with Claude Code", no emoji.
- One commit per TDD RED+GREEN cycle or per logically-scoped change.

## File Structure

**Create:**
- `network/internal/state/remote.go` — SaveRemote, LoadRemote, helpers
- `network/internal/state/remote_test.go` — TDD roundtrip + atomicity
- `tests/unit/tests/network-attach.sh` — bash client test

**Modify:**
- `network/internal/state/network.go` — extend LoadActive to route by name
- `network/internal/state/network_test.go` — add routing cases
- `network/schema/command.json` — add `"network.attach"` to command enum
- `network/cmd/chainbench-net/handlers.go` — newHandleNetworkAttach + register
- `network/cmd/chainbench-net/handlers_test.go` — handler tests
- `network/cmd/chainbench-net/e2e_test.go` — E2E test
- `docs/VISION_AND_ROADMAP.md` — mark Sprint 3b attach item complete

---

## Task 1 — state.SaveRemote + LoadActive routing

**Files:**
- Create: `network/internal/state/remote.go`
- Create: `network/internal/state/remote_test.go`
- Modify: `network/internal/state/network.go`
- Modify: `network/internal/state/network_test.go`

- [ ] **Step 1: Write failing test for SaveRemote**

```go
// remote_test.go
package state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/network/internal/types"
)

func TestSaveRemote_WritesNetworksFile(t *testing.T) {
	dir := t.TempDir()
	net := &types.Network{
		Name:      "sepolia",
		ChainType: types.NetworkChainType("ethereum"),
		ChainId:   11155111,
		Nodes: []types.Node{{
			Id:       "node1",
			Provider: types.NodeProvider("remote"),
			Http:     "https://rpc.sepolia.test",
		}},
	}
	if err := SaveRemote(dir, net); err != nil {
		t.Fatalf("SaveRemote: %v", err)
	}
	path := filepath.Join(dir, "networks", "sepolia.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("empty file written")
	}
}

func TestSaveRemote_RejectsLocal(t *testing.T) {
	dir := t.TempDir()
	net := &types.Network{Name: "local", ChainType: "stablenet", ChainId: 8283}
	if err := SaveRemote(dir, net); err == nil {
		t.Fatal("expected error for reserved name 'local'")
	}
}

func TestSaveRemote_RejectsBadName(t *testing.T) {
	dir := t.TempDir()
	// Note: schema pattern ^[a-z0-9][a-z0-9_-]*$ permits a trailing hyphen, so
	// "trailing-" is NOT a bad name. Test only cases the schema actually forbids.
	cases := []string{"", "Has-Upper", "has/slash", "..", ".hidden"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			net := &types.Network{Name: name, ChainType: "ethereum", ChainId: 1}
			if err := SaveRemote(dir, net); err == nil {
				t.Errorf("expected error for bad name %q", name)
			}
		})
	}
}

func TestSaveRemote_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	orig := &types.Network{
		Name:      "mynet",
		ChainType: types.NetworkChainType("wbft"),
		ChainId:   31337,
		Nodes: []types.Node{{
			Id:       "node1",
			Provider: types.NodeProvider("remote"),
			Http:     "https://rpc.example.com",
		}},
	}
	if err := SaveRemote(dir, orig); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := loadRemote(dir, "mynet")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if got.Name != orig.Name || got.ChainId != orig.ChainId || string(got.ChainType) != string(orig.ChainType) {
		t.Errorf("roundtrip mismatch: got %+v want %+v", got, orig)
	}
	if len(got.Nodes) != 1 || got.Nodes[0].Http != orig.Nodes[0].Http {
		t.Errorf("nodes mismatch: got %+v", got.Nodes)
	}
}

func TestLoadRemote_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := loadRemote(dir, "missing")
	if err == nil {
		t.Fatal("expected error for missing network")
	}
}

func TestSaveRemote_AtomicOnExistingFile(t *testing.T) {
	dir := t.TempDir()
	orig := &types.Network{Name: "foo", ChainType: "ethereum", ChainId: 1, Nodes: []types.Node{{Id: "node1", Provider: "remote", Http: "http://a"}}}
	if err := SaveRemote(dir, orig); err != nil {
		t.Fatal(err)
	}
	// Overwrite with new content
	updated := *orig
	updated.Nodes = []types.Node{{Id: "node1", Provider: "remote", Http: "http://b"}}
	if err := SaveRemote(dir, &updated); err != nil {
		t.Fatal(err)
	}
	got, err := loadRemote(dir, "foo")
	if err != nil {
		t.Fatal(err)
	}
	if got.Nodes[0].Http != "http://b" {
		t.Errorf("overwrite failed: %q", got.Nodes[0].Http)
	}
	// No orphan temp file
	entries, _ := os.ReadDir(filepath.Join(dir, "networks"))
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Errorf("orphan temp file: %s", e.Name())
		}
	}
}
```

- [ ] **Step 2: Run test to verify failure**

```bash
cd network && go test ./internal/state/... -run TestSaveRemote -v
```
Expected: FAIL — `undefined: SaveRemote`.

- [ ] **Step 3: Implement remote.go**

```go
// network/internal/state/remote.go
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/0xmhha/chainbench/network/internal/types"
)

// remoteNameRE mirrors network.json's name pattern: ^[a-z0-9][a-z0-9_-]*$
// Enforced at handler boundary and re-checked here (defense-in-depth against
// path traversal / reserved-name misuse).
var remoteNameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// ErrReservedName reports an attempt to save a network under the reserved "local" name.
var ErrReservedName = errors.New("state: 'local' is reserved for the local network")

// ErrInvalidName reports a network name that violates the schema pattern.
var ErrInvalidName = errors.New("state: network name must match [a-z0-9][a-z0-9_-]*")

// SaveRemote persists a remote attached network under
// <stateDir>/networks/<name>.json. The write is atomic (temp file + rename).
// Overwriting an existing file is allowed.
func SaveRemote(stateDir string, net *types.Network) error {
	if net == nil {
		return fmt.Errorf("state: nil network")
	}
	if net.Name == "local" {
		return ErrReservedName
	}
	if !remoteNameRE.MatchString(net.Name) {
		return fmt.Errorf("%w: %q", ErrInvalidName, net.Name)
	}
	dir := filepath.Join(stateDir, "networks")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("state: mkdir networks: %w", err)
	}
	raw, err := json.MarshalIndent(net, "", "  ")
	if err != nil {
		return fmt.Errorf("state: marshal network: %w", err)
	}
	finalPath := filepath.Join(dir, net.Name+".json")
	tmpPath := finalPath + ".tmp"
	if err := os.WriteFile(tmpPath, raw, 0o644); err != nil {
		return fmt.Errorf("state: write temp: %w", err)
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("state: rename: %w", err)
	}
	return nil
}

// loadRemote is package-private; external callers use LoadActive which
// routes on Name.
func loadRemote(stateDir, name string) (*types.Network, error) {
	if name == "local" {
		return nil, ErrReservedName
	}
	if !remoteNameRE.MatchString(name) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidName, name)
	}
	path := filepath.Join(stateDir, "networks", name+".json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: no attached network named %q", ErrStateNotFound, name)
		}
		return nil, fmt.Errorf("state: read %s: %w", path, err)
	}
	var net types.Network
	if err := json.Unmarshal(raw, &net); err != nil {
		return nil, fmt.Errorf("state: decode %s: %w", path, err)
	}
	return &net, nil
}
```

- [ ] **Step 4: Run remote tests**

```bash
cd network && go test ./internal/state/... -run TestSaveRemote -v
cd network && go test ./internal/state/... -run TestLoadRemote -v
```
Expected: PASS.

- [ ] **Step 5: Extend LoadActive to route by name**

Modify `network/internal/state/network.go`. Replace the early body of `LoadActive` so that a non-"local" name delegates to `loadRemote`:

```go
func LoadActive(opts LoadActiveOptions) (*types.Network, error) {
	stateDir := opts.StateDir
	if stateDir == "" {
		stateDir = "state"
	}
	name := opts.Name
	if name == "" {
		name = "local"
	}

	if name != "local" {
		return loadRemote(stateDir, name)
	}

	// existing local path unchanged below...
	pids, err := ReadPIDsFile(filepath.Join(stateDir, "pids.json"))
	// ... (keep the rest of the function as-is)
}
```

- [ ] **Step 6: Add routing test to network_test.go**

Append:
```go
func TestLoadActive_RoutesNonLocalToRemote(t *testing.T) {
	dir := t.TempDir()
	orig := &types.Network{
		Name:      "sepolia",
		ChainType: types.NetworkChainType("ethereum"),
		ChainId:   11155111,
		Nodes: []types.Node{{
			Id:       "node1",
			Provider: types.NodeProvider("remote"),
			Http:     "https://rpc.sepolia.test",
		}},
	}
	if err := SaveRemote(dir, orig); err != nil {
		t.Fatal(err)
	}
	got, err := LoadActive(LoadActiveOptions{StateDir: dir, Name: "sepolia"})
	if err != nil {
		t.Fatalf("LoadActive: %v", err)
	}
	if got.Name != "sepolia" || got.ChainId != 11155111 {
		t.Errorf("got %+v", got)
	}
}

func TestLoadActive_UnknownNameIsNotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadActive(LoadActiveOptions{StateDir: dir, Name: "ghost"})
	if err == nil {
		t.Fatal("expected error for unknown name")
	}
	if !errors.Is(err, ErrStateNotFound) {
		t.Errorf("err = %v, want wrapped ErrStateNotFound", err)
	}
}
```

Add `"errors"` to imports if not already present.

- [ ] **Step 7: Run full state suite**

```bash
cd network && go test ./internal/state/... -v
```
Expected: all tests PASS. No regression in pre-existing local tests.

- [ ] **Step 8: Commit**

```bash
git add network/internal/state/
git commit -m "feat(state): add SaveRemote + route LoadActive by network name

New state/remote.go persists attached remote networks to
<state-dir>/networks/<name>.json with atomic writes. LoadActive now
routes non-\"local\" names through loadRemote. Name validation uses
the network.json schema pattern [a-z0-9][a-z0-9_-]* plus a reserved-
name check for \"local\".

No new deps. Pure refactor on LoadActive for local networks."
```

---

## Task 2 — network.attach handler + schema

**Files:**
- Modify: `network/schema/command.json`
- Modify: `network/cmd/chainbench-net/handlers.go`
- Modify: `network/cmd/chainbench-net/handlers_test.go`

- [ ] **Step 1: Add network.attach to command enum**

Edit `network/schema/command.json` — add `"network.attach"` to the command enum array. Keep alphabetical-ish order (match existing placement patterns).

- [ ] **Step 2: Regenerate Go types if the schema feeds them**

Check if the schema code-generator runs on build:
```bash
cd network && grep -r "go:generate" .
```
If `command.go`-generated types include an enum, regenerate:
```bash
cd network && go generate ./...
```
If no generated enum exists (command name is a raw string in handlers), no regen needed.

- [ ] **Step 3: Write failing handler test**

Add to `network/cmd/chainbench-net/handlers_test.go`:

```go
// Use the existing newStablenetMockRPC / newTestBus helpers.

func TestHandleNetworkAttach_HappyPath(t *testing.T) {
	srv := newStablenetMockRPC(t)
	defer srv.Close()

	stateDir := t.TempDir()
	h := newHandleNetworkAttach(stateDir)
	args, _ := json.Marshal(map[string]any{
		"rpc_url": srv.URL,
		"name":    "testnet",
	})
	bus, _ := newTestBus(t)
	data, err := h(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if data["name"] != "testnet" {
		t.Errorf("name = %v", data["name"])
	}
	if data["chain_type"] != "stablenet" {
		t.Errorf("chain_type = %v", data["chain_type"])
	}
	if data["created"] != true {
		t.Errorf("created = %v, want true", data["created"])
	}
	// File must exist.
	if _, err := os.Stat(filepath.Join(stateDir, "networks", "testnet.json")); err != nil {
		t.Errorf("state file missing: %v", err)
	}
}

func TestHandleNetworkAttach_SecondCallOverwrites(t *testing.T) {
	srv := newStablenetMockRPC(t)
	defer srv.Close()
	stateDir := t.TempDir()
	h := newHandleNetworkAttach(stateDir)
	args, _ := json.Marshal(map[string]any{"rpc_url": srv.URL, "name": "testnet"})
	bus, _ := newTestBus(t)

	if _, err := h(args, bus); err != nil {
		t.Fatal(err)
	}
	data, err := h(args, bus)
	if err != nil {
		t.Fatal(err)
	}
	if data["created"] != false {
		t.Errorf("second call created = %v, want false", data["created"])
	}
}

func TestHandleNetworkAttach_RejectsLocalName(t *testing.T) {
	h := newHandleNetworkAttach(t.TempDir())
	args, _ := json.Marshal(map[string]any{"rpc_url": "http://x", "name": "local"})
	bus, _ := newTestBus(t)
	_, err := h(args, bus)
	if err == nil {
		t.Fatal("expected INVALID_ARGS for reserved name 'local'")
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "INVALID_ARGS" {
		t.Errorf("err = %v, want INVALID_ARGS", err)
	}
}

func TestHandleNetworkAttach_RejectsBadName(t *testing.T) {
	h := newHandleNetworkAttach(t.TempDir())
	args, _ := json.Marshal(map[string]any{"rpc_url": "http://x", "name": "Has-Upper"})
	bus, _ := newTestBus(t)
	_, err := h(args, bus)
	if err == nil {
		t.Fatal("expected INVALID_ARGS for bad name")
	}
}

func TestHandleNetworkAttach_MissingURL(t *testing.T) {
	h := newHandleNetworkAttach(t.TempDir())
	args, _ := json.Marshal(map[string]any{"name": "testnet"})
	bus, _ := newTestBus(t)
	_, err := h(args, bus)
	if err == nil {
		t.Fatal("expected INVALID_ARGS for missing rpc_url")
	}
}

func TestHandleNetworkAttach_MissingName(t *testing.T) {
	h := newHandleNetworkAttach(t.TempDir())
	args, _ := json.Marshal(map[string]any{"rpc_url": "http://x"})
	bus, _ := newTestBus(t)
	_, err := h(args, bus)
	if err == nil {
		t.Fatal("expected INVALID_ARGS for missing name")
	}
}

func TestHandleNetworkAttach_UpstreamFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	}))
	defer srv.Close()
	stateDir := t.TempDir()
	h := newHandleNetworkAttach(stateDir)
	args, _ := json.Marshal(map[string]any{"rpc_url": srv.URL, "name": "testnet"})
	bus, _ := newTestBus(t)
	_, err := h(args, bus)
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.Code != "UPSTREAM_ERROR" {
		t.Errorf("err = %v, want UPSTREAM_ERROR", err)
	}
	// No state file written.
	if _, err := os.Stat(filepath.Join(stateDir, "networks", "testnet.json")); !os.IsNotExist(err) {
		t.Errorf("state file should not exist: %v", err)
	}
}
```

- [ ] **Step 4: Implement newHandleNetworkAttach**

Edit `handlers.go`. Add to imports if missing: `os`, `path/filepath` (likely already in). Register in `allHandlers`:

```go
"network.attach": newHandleNetworkAttach(stateDir),
```

Add implementation near `newHandleNetworkProbe`:

```go
// newHandleNetworkAttach returns a handler that probes a remote RPC endpoint
// via probe.Detect, constructs a types.Network entry, and persists it under
// <state-dir>/networks/<name>.json. Subsequent network.load calls can resolve
// the network by name.
//
// Args: {"rpc_url": "...", "name": "...", "override": "..."?}
// Returns: {name, chain_type, chain_id, rpc_url, nodes, created}
func newHandleNetworkAttach(stateDir string) Handler {
	return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
		var req struct {
			RPCURL   string `json:"rpc_url"`
			Name     string `json:"name"`
			Override string `json:"override"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
			}
		}
		if req.RPCURL == "" {
			return nil, NewInvalidArgs("args.rpc_url is required")
		}
		if req.Name == "" {
			return nil, NewInvalidArgs("args.name is required")
		}
		if req.Name == "local" {
			return nil, NewInvalidArgs("args.name 'local' is reserved")
		}
		// Pre-check pattern so we don't attempt probe for a structurally-invalid name.
		if !state.IsValidRemoteName(req.Name) {
			return nil, NewInvalidArgs(fmt.Sprintf("args.name must match [a-z0-9][a-z0-9_-]*: %q", req.Name))
		}

		probeResult, err := probe.Detect(context.Background(), probe.Options{
			RPCURL:   req.RPCURL,
			Override: req.Override,
		})
		if err != nil {
			if probe.IsInputError(err) {
				return nil, NewInvalidArgs(err.Error())
			}
			return nil, NewUpstream("probe failed", err)
		}

		// Detect prior existence to set created=true/false.
		path := filepath.Join(stateDir, "networks", req.Name+".json")
		_, statErr := os.Stat(path)
		created := os.IsNotExist(statErr)

		net := &types.Network{
			Name:      req.Name,
			ChainType: types.NetworkChainType(probeResult.ChainType),
			ChainId:   int(probeResult.ChainID),
			Nodes: []types.Node{{
				Id:       "node1",
				Provider: types.NodeProvider("remote"),
				Http:     req.RPCURL,
			}},
		}
		if err := state.SaveRemote(stateDir, net); err != nil {
			return nil, NewUpstream("save remote state", err)
		}

		raw, err := json.Marshal(net)
		if err != nil {
			return nil, NewInternal("marshal network", err)
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, NewInternal("unmarshal network", err)
		}
		data["rpc_url"] = req.RPCURL
		data["created"] = created
		return data, nil
	}
}
```

- [ ] **Step 5: Export IsValidRemoteName from state**

In `network/internal/state/remote.go`, add:

```go
// IsValidRemoteName reports whether s is a structurally-valid remote network name.
// Matches network.json schema pattern; rejects "local".
func IsValidRemoteName(s string) bool {
	return s != "local" && remoteNameRE.MatchString(s)
}
```

- [ ] **Step 6: Add imports to handlers.go if missing**

Ensure `"os"`, `"path/filepath"`, `"github.com/0xmhha/chainbench/network/internal/probe"`, `"github.com/0xmhha/chainbench/network/internal/state"`, `"github.com/0xmhha/chainbench/network/internal/types"` are present.

- [ ] **Step 7: Run tests**

```bash
cd network && go test ./cmd/chainbench-net/... -run TestHandleNetworkAttach -v
cd network && go test ./... -count=1
```
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add network/schema/command.json network/cmd/chainbench-net/handlers.go network/cmd/chainbench-net/handlers_test.go network/internal/state/remote.go
git commit -m "feat(network-net): add network.attach handler

Wires probe.Detect + state.SaveRemote into a single command that
persists an attached remote network under state/networks/<name>.json.
Returns created=true on first save, false on overwrite. Rejects name
'local' (reserved) and bad patterns at handler boundary.

Schema: add 'network.attach' to command.json enum."
```

---

## Task 3 — Go E2E test

**Files:**
- Modify: `network/cmd/chainbench-net/e2e_test.go`

- [ ] **Step 1: Write E2E test**

```go
func TestE2E_NetworkAttach_ViaRootCommand(t *testing.T) {
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

	stateDir := t.TempDir()
	t.Setenv("CHAINBENCH_STATE_DIR", stateDir)

	var stdout, stderr bytes.Buffer
	root := newRootCmd()
	cmd := fmt.Sprintf(`{"command":"network.attach","args":{"rpc_url":%q,"name":"integration"}}`, rpcSrv.URL)
	root.SetIn(strings.NewReader(cmd))
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"run"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v\nstderr: %s", err, stderr.String())
	}

	// Find result terminator, assert shape.
	var resultLine []byte
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		if err := schema.ValidateBytes("event", line); err != nil {
			t.Fatalf("schema validation: %v\nline: %s", err, line)
		}
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
	if res.Data["chain_type"] != "stablenet" {
		t.Errorf("chain_type = %v", res.Data["chain_type"])
	}
	if res.Data["name"] != "integration" {
		t.Errorf("name = %v", res.Data["name"])
	}

	// Verify the state file landed on disk.
	if _, err := os.Stat(filepath.Join(stateDir, "networks", "integration.json")); err != nil {
		t.Errorf("state file missing: %v", err)
	}

	// Second attempt: network.load the same name and verify equivalence.
	stdout.Reset()
	stderr.Reset()
	root2 := newRootCmd()
	loadCmd := `{"command":"network.load","args":{"name":"integration"}}`
	root2.SetIn(strings.NewReader(loadCmd))
	root2.SetOut(&stdout)
	root2.SetErr(&stderr)
	root2.SetArgs([]string{"run"})
	if err := root2.Execute(); err != nil {
		t.Fatalf("load execute: %v\nstderr: %s", err, stderr.String())
	}
	scanner = bufio.NewScanner(&stdout)
	var loadResult []byte
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		if msg, _ := wire.DecodeMessage(line); msg != nil {
			if _, ok := msg.(wire.ResultMessage); ok {
				loadResult = line
			}
		}
	}
	if loadResult == nil {
		t.Fatal("no load result")
	}
	var loadRes struct {
		Ok   bool           `json:"ok"`
		Data map[string]any `json:"data"`
	}
	_ = json.Unmarshal(loadResult, &loadRes)
	if !loadRes.Ok {
		t.Fatalf("load not ok: %s", loadResult)
	}
	if loadRes.Data["name"] != "integration" {
		t.Errorf("load.name = %v", loadRes.Data["name"])
	}
}
```

- [ ] **Step 2: Run test**

```bash
cd network && go test ./cmd/chainbench-net/... -run TestE2E_NetworkAttach -v
```
Expected: PASS.

- [ ] **Step 3: Run full Go suite**

```bash
cd network && go test ./... -count=1
```
Expected: all PASS.

- [ ] **Step 4: Commit**

```bash
git add network/cmd/chainbench-net/e2e_test.go
git commit -m "test(network-net): E2E test for network.attach + load roundtrip

Drives cobra root command twice: first with network.attach, then with
network.load on the same name. Validates that the persisted state file
resolves correctly via the non-local LoadActive routing."
```

---

## Task 4 — bash client test

**Files:**
- Create: `tests/unit/tests/network-attach.sh`

- [ ] **Step 1: Create test file**

Use `tests/unit/tests/network-probe.sh` as the template (it already has the python mock, port picker, trap cleanup, and cb_net_call invocation pattern).

```bash
#!/usr/bin/env bash
# tests/unit/tests/network-attach.sh
# Covers the network.attach wire handler end-to-end:
#   attach via cb_net_call, then load the same name back and verify the
#   persisted state file round-trips.
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
if ! ( cd "${CHAINBENCH_DIR}/network" && go build -o "${BINARY}" ./cmd/chainbench-net ) >/dev/null 2>&1; then
  echo "FATAL: failed to build chainbench-net" >&2
  rm -rf "${TMPDIR_ROOT}"
  exit 1
fi
export CHAINBENCH_NET_BIN="${BINARY}"

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
        try:
            req = json.loads(self.rfile.read(n))
        except Exception:
            req = {}
        method = req.get('method')
        rid = req.get('id')
        if method == 'eth_chainId':
            body = {'jsonrpc': '2.0', 'id': rid, 'result': '0x205b'}
        elif method == 'istanbul_getValidators':
            body = {'jsonrpc': '2.0', 'id': rid, 'result': []}
        else:
            body = {'jsonrpc': '2.0', 'id': rid,
                    'error': {'code': -32601, 'message': 'method not found'}}
        raw = json.dumps(body).encode()
        self.send_response(200)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Content-Length', str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)
    def log_message(self, *a, **k):
        pass
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
  if (echo > "/dev/tcp/127.0.0.1/${PORT}") 2>/dev/null; then
    mock_ready=1
    break
  fi
  sleep 0.1
done
if [[ "${mock_ready}" -ne 1 ]]; then
  echo "FATAL: mock RPC server failed to listen on 127.0.0.1:${PORT}" >&2
  cat "${MOCK_LOG}" >&2 || true
  exit 1
fi

# shellcheck disable=SC1091
source "${CHAINBENCH_DIR}/lib/network_client.sh"

describe "network.attach: persists remote network and round-trips via load"
rc=0
data="$(cb_net_call "network.attach" "{\"rpc_url\":\"http://127.0.0.1:${PORT}\",\"name\":\"bash-attach\"}" 2>/dev/null)" || rc=$?
assert_eq "$rc" "0" "attach success exit code"
assert_eq "$(jq -r .chain_type <<<"$data")" "stablenet"   "attach chain_type"
assert_eq "$(jq -r .name       <<<"$data")" "bash-attach" "attach name"
assert_eq "$(jq -r .created    <<<"$data")" "true"        "attach created flag"

# State file must exist.
if [[ ! -f "${STATE_DIR}/networks/bash-attach.json" ]]; then
  echo "FATAL: state file missing" >&2
  exit 1
fi

describe "network.load: resolves attached remote by name"
rc=0
loaded="$(cb_net_call "network.load" '{"name":"bash-attach"}' 2>/dev/null)" || rc=$?
assert_eq "$rc" "0" "load success exit code"
assert_eq "$(jq -r .name <<<"$loaded")" "bash-attach" "load name"
assert_eq "$(jq -r .chain_id <<<"$loaded")" "8283" "load chain_id"

unit_summary
```

- [ ] **Step 2: Run test**

```bash
bash tests/unit/tests/network-attach.sh
```
Expected: all assertions PASS.

- [ ] **Step 3: Run full bash suite**

```bash
bash tests/unit/run.sh
```
Expected: full pass count increases by one; no regressions.

- [ ] **Step 4: Commit**

```bash
git add tests/unit/tests/network-attach.sh
git commit -m "test(bash): add network-attach.sh unit test

Attaches a remote network via cb_net_call against a python JSON-RPC
mock, asserts the persisted shape, then loads the same name back
and asserts equivalence. Covers the round-trip that Sprint 3b ships."
```

---

## Task 5 — Final review + roadmap

- [ ] **Step 1: Full test matrix**

```bash
cd network && go test ./... -count=1
cd .. && bash tests/unit/run.sh
```
Expected: green.

- [ ] **Step 2: Update roadmap**

Edit `docs/VISION_AND_ROADMAP.md` §6 Sprint 3 checklist. The current `drivers/remote` item is *not* complete (deferred to 3b.2). Add a new line marking attach complete; keep drivers/remote unchecked:

```
- [x] `network.attach` + state routing (Sprint 3b 완료, 2026-04-23) — RemoteDriver 및 auth 은 3b.2 로 분리
- [ ] `drivers/remote` — `go-ethereum/ethclient` 기반 + API key / JWT 인증 (S6) — Sprint 3b.2 예정
```

- [ ] **Step 3: Commit**

```bash
git add docs/VISION_AND_ROADMAP.md
git commit -m "docs: mark Sprint 3b attach complete; 3b.2 tracks RemoteDriver"
```

- [ ] **Step 4: Report**

Report to controller:
- Test counts (Go + bash)
- Coverage for new packages
- Commit SHA range
- Any deferrals (expected: RemoteDriver + auth + M4 all still deferred to 3b.2)
