# Sprint 2b.1 network.load Command Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the first end-to-end wire-compliant command for `chainbench-net`: `network.load`. Input on stdin (`{"command":"network.load","args":{"name":"local"}}`), output NDJSON with a Network object terminator, structured logs on stderr, exit code per VISION §5.

**Architecture:** One new package `network/internal/state/` parses `state/pids.json` + `state/current-profile.yaml` into `types.Network`. The cobra `run` subcommand on `chainbench-net` reads a command envelope via Sprint 2a's `wire.DecodeCommand`, dispatches by `command` field to a handler that returns `(data, error)`, and emits `result` via `wire.Emitter`. `APIError` + `exitCode` map errors to the schema's `ResultErrorCode` + OS exit code.

**Tech Stack:** Go 1.25, cobra (already pinned), `gopkg.in/yaml.v3` (new direct dep), Sprint 2a's `network/internal/wire` and `network/internal/events`, schema validator from `network/schema`.

**Spec reference:** `docs/superpowers/specs/2026-04-22-sprint-2b1-network-load-design.md`.

---

## File Structure

**Created in this plan:**
- `network/internal/state/doc.go`
- `network/internal/state/profile.go`
- `network/internal/state/profile_test.go`
- `network/internal/state/pids.go`
- `network/internal/state/pids_test.go`
- `network/internal/state/network.go`
- `network/internal/state/network_test.go`
- `network/internal/state/testdata/profile-default.yaml`
- `network/internal/state/testdata/pids-default.json`
- `network/cmd/chainbench-net/errors.go`
- `network/cmd/chainbench-net/errors_test.go`
- `network/cmd/chainbench-net/handlers.go`
- `network/cmd/chainbench-net/handlers_test.go`
- `network/cmd/chainbench-net/run.go`
- `network/cmd/chainbench-net/run_test.go`
- `network/cmd/chainbench-net/e2e_test.go`
- `network/cmd/chainbench-net/testdata/pids.json`
- `network/cmd/chainbench-net/testdata/current-profile.yaml`

**Modified:**
- `network/go.mod` (adds `gopkg.in/yaml.v3`)
- `network/go.sum`
- `network/cmd/chainbench-net/main.go` (add `run` subcommand registration + exit code wiring)

---

## Task 1: `internal/state/profile.go` — YAML profile parser

**Files:**
- Create: `network/internal/state/doc.go`
- Create: `network/internal/state/profile.go`
- Create: `network/internal/state/profile_test.go`
- Create: `network/internal/state/testdata/profile-default.yaml`

- [ ] **Step 1.1: Add YAML dependency**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network
go get gopkg.in/yaml.v3@latest
```
Expected: `go.mod` gets `gopkg.in/yaml.v3`; `go.sum` updated.

- [ ] **Step 1.2: Create test fixture**

Create `network/internal/state/testdata/profile-default.yaml`:

```yaml
name: default

chain:
  binary: gstable
  binary_path: ""
  network_id: 8283
  chain_id: 8283

nodes:
  validators: 4
  endpoints: 1

ports:
  base_p2p: 30301
  base_http: 8501
  base_ws: 9501
```

- [ ] **Step 1.3: Write failing test**

Create `network/internal/state/profile_test.go`:

```go
package state

import (
	"bytes"
	"os"
	"testing"
)

func TestParseProfile_ValidFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/profile-default.yaml")
	if err != nil {
		t.Fatal(err)
	}
	p, err := ParseProfile(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if p.Name != "default" {
		t.Errorf("name: got %q, want default", p.Name)
	}
	if p.Chain.Binary != "gstable" {
		t.Errorf("binary: got %q", p.Chain.Binary)
	}
	if p.Chain.ChainID != 8283 {
		t.Errorf("chain_id: got %d, want 8283", p.Chain.ChainID)
	}
	if p.Chain.NetworkID != 8283 {
		t.Errorf("network_id: got %d", p.Chain.NetworkID)
	}
	if p.Nodes.Validators != 4 || p.Nodes.Endpoints != 1 {
		t.Errorf("nodes: got %+v", p.Nodes)
	}
	if p.Ports.BaseHTTP != 8501 || p.Ports.BaseWS != 9501 || p.Ports.BaseP2P != 30301 {
		t.Errorf("ports: got %+v", p.Ports)
	}
}

func TestParseProfile_Empty(t *testing.T) {
	p, err := ParseProfile(bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("empty should parse to zero-value profile, got: %v", err)
	}
	if p.Name != "" || p.Chain.ChainID != 0 {
		t.Errorf("expected zero-value profile, got %+v", p)
	}
}

func TestParseProfile_MalformedYAML(t *testing.T) {
	_, err := ParseProfile(bytes.NewReader([]byte("chain: [this is not valid")))
	if err == nil {
		t.Fatal("expected error for malformed YAML")
	}
}

func TestReadProfileFile_MissingFile(t *testing.T) {
	_, err := ReadProfileFile("testdata/does-not-exist.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestReadProfileFile_Valid(t *testing.T) {
	p, err := ReadProfileFile("testdata/profile-default.yaml")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if p.Chain.ChainID != 8283 {
		t.Errorf("chain_id: got %d", p.Chain.ChainID)
	}
}
```

- [ ] **Step 1.4: Run test — expect fail (undefined)**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/state/... 2>&1 | tail -10
```
Expected: FAIL — `no Go files` or `undefined: ParseProfile`.

- [ ] **Step 1.5: Write `doc.go`**

Create `network/internal/state/doc.go`:

```go
// Package state reads on-disk chain state (state/pids.json,
// state/current-profile.yaml) and produces a types.Network suitable for
// wire emission by the network.load command and future commands.
//
// This package is read-only — it does not mutate state files. Side-effecting
// commands (start/stop/restart) live under drivers/ in later sprints.
package state
```

- [ ] **Step 1.6: Write `profile.go`**

Create `network/internal/state/profile.go`:

```go
package state

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// Profile is the subset of profile YAML fields the state package needs.
// Extend as more handlers require more data.
type Profile struct {
	Name  string       `yaml:"name"`
	Chain ChainBlock   `yaml:"chain"`
	Nodes NodesBlock   `yaml:"nodes"`
	Ports PortsBlock   `yaml:"ports"`
}

// ChainBlock mirrors the "chain:" section of a profile.
type ChainBlock struct {
	Binary     string `yaml:"binary"`
	BinaryPath string `yaml:"binary_path"`
	ChainID    int64  `yaml:"chain_id"`
	NetworkID  int64  `yaml:"network_id"`
	Type       string `yaml:"type"` // optional; default applied at consumption time
}

// NodesBlock mirrors the "nodes:" section.
type NodesBlock struct {
	Validators int `yaml:"validators"`
	Endpoints  int `yaml:"endpoints"`
}

// PortsBlock mirrors the "ports:" section.
type PortsBlock struct {
	BaseP2P  int `yaml:"base_p2p"`
	BaseHTTP int `yaml:"base_http"`
	BaseWS   int `yaml:"base_ws"`
}

// ParseProfile decodes a profile YAML from r into a Profile. Missing fields
// get zero values; consumers apply defaults.
func ParseProfile(r io.Reader) (*Profile, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("state: read profile: %w", err)
	}
	var p Profile
	if len(data) == 0 {
		return &p, nil
	}
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("state: parse profile: %w", err)
	}
	return &p, nil
}

// ReadProfileFile opens path and delegates to ParseProfile.
func ReadProfileFile(path string) (*Profile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("state: open profile: %w", err)
	}
	defer f.Close()
	return ParseProfile(f)
}
```

- [ ] **Step 1.7: Run test — expect pass**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/state/... -v -run TestParseProfile && go test ./internal/state/... -v -run TestReadProfileFile
```
Expected: 5 subtests PASS.

- [ ] **Step 1.8: Build/vet/fmt**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go build ./... && go vet ./... && gofmt -l internal/state/
```
Expected: exit 0, gofmt empty.

- [ ] **Step 1.9: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/go.mod network/go.sum network/internal/state/doc.go network/internal/state/profile.go network/internal/state/profile_test.go network/internal/state/testdata/profile-default.yaml
git commit -m "network: add state profile YAML parser"
```

---

## Task 2: `internal/state/pids.go` — pids.json parser

**Files:**
- Create: `network/internal/state/pids.go`
- Create: `network/internal/state/pids_test.go`
- Create: `network/internal/state/testdata/pids-default.json`

- [ ] **Step 2.1: Create test fixture**

Create `network/internal/state/testdata/pids-default.json`:

```json
{
  "chain_id": "local-default-20260420084840",
  "profile": "default",
  "started_at": "2026-04-20T08:48:40Z",
  "nodes": {
    "1": {
      "pid": 23350,
      "type": "validator",
      "p2p_port": 30301,
      "http_port": 8501,
      "ws_port": 9501,
      "auth_port": 8551,
      "metrics_port": 6061,
      "status": "running",
      "log_file": "/tmp/node-data/logs/node1.log",
      "binary": "/opt/gstable",
      "datadir": "/tmp/node-data/node1"
    },
    "5": {
      "pid": 23354,
      "type": "endpoint",
      "p2p_port": 30305,
      "http_port": 8505,
      "ws_port": 9505,
      "auth_port": 8555,
      "metrics_port": 6065,
      "status": "running",
      "log_file": "/tmp/node-data/logs/node5.log",
      "binary": "/opt/gstable",
      "datadir": "/tmp/node-data/node5"
    }
  }
}
```

- [ ] **Step 2.2: Write failing test**

Create `network/internal/state/pids_test.go`:

```go
package state

import (
	"bytes"
	"os"
	"testing"
)

func TestParsePIDs_ValidFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/pids-default.json")
	if err != nil {
		t.Fatal(err)
	}
	p, err := ParsePIDs(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if p.Profile != "default" {
		t.Errorf("profile: got %q", p.Profile)
	}
	if p.ChainID != "local-default-20260420084840" {
		t.Errorf("chain_id: got %q", p.ChainID)
	}
	if len(p.Nodes) != 2 {
		t.Fatalf("nodes count: got %d, want 2", len(p.Nodes))
	}
	n1, ok := p.Nodes["1"]
	if !ok {
		t.Fatal("node 1 missing")
	}
	if n1.Type != "validator" || n1.HTTPPort != 8501 || n1.WSPort != 9501 {
		t.Errorf("node 1: got %+v", n1)
	}
	n5, ok := p.Nodes["5"]
	if !ok {
		t.Fatal("node 5 missing")
	}
	if n5.Type != "endpoint" || n5.HTTPPort != 8505 {
		t.Errorf("node 5: got %+v", n5)
	}
}

func TestParsePIDs_MalformedJSON(t *testing.T) {
	_, err := ParsePIDs(bytes.NewReader([]byte(`{"chain_id":`)))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestParsePIDs_EmptyNodes(t *testing.T) {
	p, err := ParsePIDs(bytes.NewReader([]byte(`{"chain_id":"x","profile":"y","nodes":{}}`)))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(p.Nodes) != 0 {
		t.Errorf("nodes: got %d, want 0", len(p.Nodes))
	}
}

func TestReadPIDsFile_MissingFile(t *testing.T) {
	_, err := ReadPIDsFile("testdata/does-not-exist.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestReadPIDsFile_Valid(t *testing.T) {
	p, err := ReadPIDsFile("testdata/pids-default.json")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(p.Nodes) != 2 {
		t.Errorf("nodes: got %d", len(p.Nodes))
	}
}
```

- [ ] **Step 2.3: Run test — expect fail**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/state/... -run TestParsePIDs 2>&1 | tail -5
```
Expected: FAIL — `undefined: ParsePIDs`.

- [ ] **Step 2.4: Write `pids.go`**

Create `network/internal/state/pids.go`:

```go
package state

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// PIDsFile mirrors the shape of state/pids.json.
type PIDsFile struct {
	ChainID   string              `json:"chain_id"`
	Profile   string              `json:"profile"`
	StartedAt string              `json:"started_at"`
	Nodes     map[string]NodeInfo `json:"nodes"`
}

// NodeInfo mirrors a single entry under state/pids.json ".nodes".
// Fields not needed by the Network mapping are intentionally omitted
// (e.g., saved_args).
type NodeInfo struct {
	PID         int    `json:"pid"`
	Type        string `json:"type"`
	P2PPort     int    `json:"p2p_port"`
	HTTPPort    int    `json:"http_port"`
	WSPort      int    `json:"ws_port"`
	AuthPort    int    `json:"auth_port"`
	MetricsPort int    `json:"metrics_port"`
	Status      string `json:"status"`
	LogFile     string `json:"log_file"`
	Binary      string `json:"binary"`
	Datadir     string `json:"datadir"`
}

// ParsePIDs decodes state/pids.json from r.
func ParsePIDs(r io.Reader) (*PIDsFile, error) {
	dec := json.NewDecoder(r)
	var p PIDsFile
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("state: parse pids.json: %w", err)
	}
	if p.Nodes == nil {
		p.Nodes = map[string]NodeInfo{}
	}
	return &p, nil
}

// ReadPIDsFile opens path and delegates to ParsePIDs.
func ReadPIDsFile(path string) (*PIDsFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("state: open pids.json: %w", err)
	}
	defer f.Close()
	return ParsePIDs(f)
}
```

- [ ] **Step 2.5: Run test — expect pass**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/state/... -v -run 'TestParsePIDs|TestReadPIDsFile'
```
Expected: 5 subtests PASS.

- [ ] **Step 2.6: Full state package + vet + fmt**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/state/... && go vet ./... && gofmt -l internal/state/
```
Expected: all OK.

- [ ] **Step 2.7: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/internal/state/pids.go network/internal/state/pids_test.go network/internal/state/testdata/pids-default.json
git commit -m "network: add state pids.json parser"
```

---

## Task 3: `internal/state/network.go` — LoadActive (combine + schema cross-check)

**Files:**
- Create: `network/internal/state/network.go`
- Create: `network/internal/state/network_test.go`

- [ ] **Step 3.1: Write failing test**

Create `network/internal/state/network_test.go`:

```go
package state_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/network/internal/state"
	"github.com/0xmhha/chainbench/network/schema"
)

// setupStateDir creates a temporary state/ directory with fixture files and
// returns its path.
func setupStateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	copy := func(src, dst string) {
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, dst), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	copy("testdata/pids-default.json", "pids.json")
	copy("testdata/profile-default.yaml", "current-profile.yaml")
	return dir
}

func TestLoadActive_HappyPath(t *testing.T) {
	dir := setupStateDir(t)
	net, err := state.LoadActive(state.LoadActiveOptions{StateDir: dir, Name: "local"})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if string(net.Name) != "local" {
		t.Errorf("name: got %q, want local", net.Name)
	}
	if string(net.ChainType) != "stablenet" {
		t.Errorf("chain_type: got %q, want stablenet (default)", net.ChainType)
	}
	if net.ChainId != 8283 {
		t.Errorf("chain_id: got %d, want 8283", net.ChainId)
	}
	if len(net.Nodes) != 2 {
		t.Fatalf("nodes: got %d, want 2", len(net.Nodes))
	}
	// Nodes must be sorted by numeric key ascending.
	if net.Nodes[0].Id != "node1" {
		t.Errorf("nodes[0].id: got %q, want node1", net.Nodes[0].Id)
	}
	if net.Nodes[1].Id != "node5" {
		t.Errorf("nodes[1].id: got %q, want node5", net.Nodes[1].Id)
	}
	// Provider and URLs.
	if string(net.Nodes[0].Provider) != "local" {
		t.Errorf("provider: got %q", net.Nodes[0].Provider)
	}
	if net.Nodes[0].Http != "http://127.0.0.1:8501" {
		t.Errorf("http: got %q", net.Nodes[0].Http)
	}
	if net.Nodes[0].Ws == nil || *net.Nodes[0].Ws != "ws://127.0.0.1:9501" {
		t.Errorf("ws: got %v", net.Nodes[0].Ws)
	}
	// Role mapping.
	if net.Nodes[0].Role == nil || string(*net.Nodes[0].Role) != "validator" {
		t.Errorf("role: got %v", net.Nodes[0].Role)
	}
	if net.Nodes[1].Role == nil || string(*net.Nodes[1].Role) != "endpoint" {
		t.Errorf("role: got %v", net.Nodes[1].Role)
	}
}

func TestLoadActive_OutputValidatesAgainstSchema(t *testing.T) {
	dir := setupStateDir(t)
	net, err := state.LoadActive(state.LoadActiveOptions{StateDir: dir, Name: "local"})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	raw, err := json.Marshal(net)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := schema.ValidateBytes("network", raw); err != nil {
		t.Fatalf("schema validation: %v\nraw: %s", err, raw)
	}
}

func TestLoadActive_DefaultsStateDirAndName(t *testing.T) {
	dir := setupStateDir(t)
	// StateDir provided but Name empty → default "local".
	net, err := state.LoadActive(state.LoadActiveOptions{StateDir: dir})
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if string(net.Name) != "local" {
		t.Errorf("name default: got %q", net.Name)
	}
}

func TestLoadActive_MissingPIDs(t *testing.T) {
	dir := t.TempDir()
	// profile exists, pids.json missing
	data, _ := os.ReadFile("testdata/profile-default.yaml")
	_ = os.WriteFile(filepath.Join(dir, "current-profile.yaml"), data, 0o644)
	_, err := state.LoadActive(state.LoadActiveOptions{StateDir: dir, Name: "local"})
	if err == nil {
		t.Fatal("expected error when pids.json is missing")
	}
}

func TestLoadActive_MissingProfile(t *testing.T) {
	dir := t.TempDir()
	data, _ := os.ReadFile("testdata/pids-default.json")
	_ = os.WriteFile(filepath.Join(dir, "pids.json"), data, 0o644)
	_, err := state.LoadActive(state.LoadActiveOptions{StateDir: dir, Name: "local"})
	if err == nil {
		t.Fatal("expected error when profile is missing")
	}
}
```

- [ ] **Step 3.2: Run test — expect fail**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/state/... -run TestLoadActive 2>&1 | tail -5
```
Expected: FAIL — `undefined: LoadActive`.

- [ ] **Step 3.3: Write `network.go`**

Create `network/internal/state/network.go`:

```go
package state

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/0xmhha/chainbench/network/internal/types"
)

// LoadActiveOptions controls how state files map to a Network.
type LoadActiveOptions struct {
	// StateDir is the directory containing pids.json and current-profile.yaml.
	// Empty means "state/" relative to the process working directory.
	StateDir string
	// Name is Network.name. Empty defaults to "local".
	Name string
}

// ErrStateNotFound is returned when required state files are absent.
// Callers in the command layer map this to UPSTREAM_ERROR.
var ErrStateNotFound = errors.New("state: active chain state not found")

// LoadActive reads pids.json + current-profile.yaml under opts.StateDir and
// builds a types.Network. Nodes are sorted by numeric pids key ascending.
func LoadActive(opts LoadActiveOptions) (*types.Network, error) {
	stateDir := opts.StateDir
	if stateDir == "" {
		stateDir = "state"
	}
	name := opts.Name
	if name == "" {
		name = "local"
	}

	pids, err := ReadPIDsFile(filepath.Join(stateDir, "pids.json"))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrStateNotFound, err)
	}
	profile, err := ReadProfileFile(filepath.Join(stateDir, "current-profile.yaml"))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrStateNotFound, err)
	}

	chainType := profile.Chain.Type
	if chainType == "" {
		chainType = "stablenet"
	}

	nodes, err := buildNodes(pids)
	if err != nil {
		return nil, err
	}

	net := &types.Network{
		Name:      types.NetworkName(name),
		ChainType: types.NetworkChainType(chainType),
		ChainId:   profile.Chain.ChainID,
		Nodes:     nodes,
	}
	return net, nil
}

func buildNodes(p *PIDsFile) ([]types.Node, error) {
	keys := make([]string, 0, len(p.Nodes))
	for k := range p.Nodes {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, _ := strconv.Atoi(keys[i])
		b, _ := strconv.Atoi(keys[j])
		return a < b
	})

	out := make([]types.Node, 0, len(keys))
	for _, k := range keys {
		info := p.Nodes[k]
		id := "node" + k
		role := mapRole(info.Type)
		ws := fmt.Sprintf("ws://127.0.0.1:%d", info.WSPort)
		provider := types.NodeProvider("local")
		meta := types.NodeProviderMeta{"pid_key": id}
		node := types.Node{
			Id:            id,
			Provider:      provider,
			Http:          fmt.Sprintf("http://127.0.0.1:%d", info.HTTPPort),
			Ws:            &ws,
			Role:          &role,
			ProviderMeta:  meta,
		}
		out = append(out, node)
	}
	return out, nil
}

func mapRole(t string) types.NodeRole {
	switch t {
	case "validator":
		return types.NodeRole("validator")
	case "endpoint":
		return types.NodeRole("endpoint")
	default:
		return types.NodeRole("observer")
	}
}
```

> **Note:** Exact field names (`Ws`, `Role`, `ProviderMeta`, `Http`, `ChainId`, `Name`, etc.) depend on the generated `types` package. If the generator produced different names (e.g., `WS` all-caps, `HTTPUrl`), update the code to match. Verify by reading `network/internal/types/network_gen.go` before the first build. The test asserts on the getters it thinks the struct exposes — if compilation fails, the test may need adjustments to access the correct fields; if so, keep assertion semantics identical (ids, URLs, roles).

- [ ] **Step 3.4: Run test — expect pass (may need field-name adjustment)**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./internal/state/... -v -run TestLoadActive 2>&1 | tail -20
```

If compile fails due to field-name mismatch:
1. Read `network/internal/types/network_gen.go` — find exact `Node` / `Network` struct field names.
2. Update `network.go` and `network_test.go` accordingly. Preserve assertion semantics (same IDs, URLs, etc.).
3. Re-run. Expected 5 subtests PASS.

- [ ] **Step 3.5: Full state package race + coverage**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test -race -cover ./internal/state/...
```
Expected: PASS, coverage ≥ 85%.

- [ ] **Step 3.6: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/internal/state/network.go network/internal/state/network_test.go
git commit -m "network: add state LoadActive network builder with schema cross-check"
```

---

## Task 4: `cmd/chainbench-net/errors.go` — APIError + exitCode

**Files:**
- Create: `network/cmd/chainbench-net/errors.go`
- Create: `network/cmd/chainbench-net/errors_test.go`

- [ ] **Step 4.1: Write failing test**

Create `network/cmd/chainbench-net/errors_test.go`:

```go
package main

import (
	"errors"
	"testing"

	"github.com/0xmhha/chainbench/network/internal/types"
)

func TestAPIError_ErrorMessage(t *testing.T) {
	e := &APIError{Code: types.ResultErrorCode("INVALID_ARGS"), Message: "bad name"}
	if e.Error() == "" {
		t.Error("Error() returned empty string")
	}
}

func TestAPIError_Unwrap(t *testing.T) {
	cause := errors.New("underlying")
	e := &APIError{Code: types.ResultErrorCode("UPSTREAM_ERROR"), Message: "upstream failed", Cause: cause}
	if !errors.Is(e, cause) {
		t.Error("errors.Is should find cause")
	}
}

func TestExitCode_NilIsZero(t *testing.T) {
	if got := exitCode(nil); got != 0 {
		t.Errorf("nil: got %d, want 0", got)
	}
}

func TestExitCode_MapsErrorCodes(t *testing.T) {
	cases := map[string]int{
		"NOT_SUPPORTED":  2,
		"PROTOCOL_ERROR": 3,
		"INVALID_ARGS":   1,
		"UPSTREAM_ERROR": 1,
		"INTERNAL":       1,
	}
	for code, want := range cases {
		e := &APIError{Code: types.ResultErrorCode(code), Message: "x"}
		if got := exitCode(e); got != want {
			t.Errorf("code %s: got %d, want %d", code, got, want)
		}
	}
}

func TestExitCode_GenericErrorTreatedAsInternal(t *testing.T) {
	if got := exitCode(errors.New("generic")); got != 1 {
		t.Errorf("generic: got %d, want 1", got)
	}
}

func TestNewInvalidArgs(t *testing.T) {
	e := NewInvalidArgs("bad")
	if string(e.Code) != "INVALID_ARGS" || e.Message != "bad" {
		t.Errorf("got %+v", e)
	}
}

func TestNewUpstream_WrapsCase(t *testing.T) {
	cause := errors.New("x")
	e := NewUpstream("disk gone", cause)
	if string(e.Code) != "UPSTREAM_ERROR" {
		t.Errorf("code: got %q", e.Code)
	}
	if !errors.Is(e, cause) {
		t.Error("should wrap cause")
	}
}
```

- [ ] **Step 4.2: Run — expect fail**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./cmd/chainbench-net/... -run 'TestAPIError|TestExitCode|TestNew' 2>&1 | tail -5
```
Expected: FAIL — `undefined: APIError`, `undefined: exitCode`, etc.

- [ ] **Step 4.3: Write `errors.go`**

Create `network/cmd/chainbench-net/errors.go`:

```go
package main

import (
	"errors"
	"fmt"

	"github.com/0xmhha/chainbench/network/internal/types"
)

// APIError is a typed error carrying a wire-level result error code plus a
// user-facing message. Handlers return this when a specific code is meaningful;
// any other error is treated as INTERNAL by the dispatcher.
type APIError struct {
	Code    types.ResultErrorCode
	Message string
	Cause   error
}

func (e *APIError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *APIError) Unwrap() error { return e.Cause }

// Constructors for the common codes.

func NewInvalidArgs(message string) *APIError {
	return &APIError{Code: types.ResultErrorCode("INVALID_ARGS"), Message: message}
}

func NewNotSupported(message string) *APIError {
	return &APIError{Code: types.ResultErrorCode("NOT_SUPPORTED"), Message: message}
}

func NewUpstream(message string, cause error) *APIError {
	return &APIError{Code: types.ResultErrorCode("UPSTREAM_ERROR"), Message: message, Cause: cause}
}

func NewProtocolError(message string, cause error) *APIError {
	return &APIError{Code: types.ResultErrorCode("PROTOCOL_ERROR"), Message: message, Cause: cause}
}

func NewInternal(message string, cause error) *APIError {
	return &APIError{Code: types.ResultErrorCode("INTERNAL"), Message: message, Cause: cause}
}

// exitCode maps an error (possibly nil) to an OS exit code per VISION §5.
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var api *APIError
	if errors.As(err, &api) {
		switch string(api.Code) {
		case "NOT_SUPPORTED":
			return 2
		case "PROTOCOL_ERROR":
			return 3
		case "INVALID_ARGS", "UPSTREAM_ERROR", "INTERNAL":
			return 1
		}
	}
	return 1
}
```

- [ ] **Step 4.4: Run — expect pass**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./cmd/chainbench-net/... -v -run 'TestAPIError|TestExitCode|TestNew'
```
Expected: 8 subtests PASS.

- [ ] **Step 4.5: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/cmd/chainbench-net/errors.go network/cmd/chainbench-net/errors_test.go
git commit -m "network: add cmd error type with exit code mapping"
```

---

## Task 5: `cmd/chainbench-net/handlers.go` — network.load handler

**Files:**
- Create: `network/cmd/chainbench-net/handlers.go`
- Create: `network/cmd/chainbench-net/handlers_test.go`
- Create: `network/cmd/chainbench-net/testdata/pids.json`
- Create: `network/cmd/chainbench-net/testdata/current-profile.yaml`

- [ ] **Step 5.1: Create cmd testdata fixtures**

Copy the state package fixtures into cmd testdata (handler tests need them).

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
mkdir -p network/cmd/chainbench-net/testdata
cp network/internal/state/testdata/pids-default.json     network/cmd/chainbench-net/testdata/pids.json
cp network/internal/state/testdata/profile-default.yaml  network/cmd/chainbench-net/testdata/current-profile.yaml
```

Verify:
```bash
ls network/cmd/chainbench-net/testdata/
```
Expected: `current-profile.yaml  pids.json`.

- [ ] **Step 5.2: Write failing test**

Create `network/cmd/chainbench-net/handlers_test.go`:

```go
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/network/internal/events"
	"github.com/0xmhha/chainbench/network/internal/types"
	"github.com/0xmhha/chainbench/network/internal/wire"
)

// setupCmdStateDir builds a temp state dir populated from cmd testdata.
func setupCmdStateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"pids.json", "current-profile.yaml"} {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func newTestBus(t *testing.T) (*events.Bus, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	return events.NewBus(wire.NewEmitter(&buf)), &buf
}

func TestHandleNetworkLoad_HappyPath(t *testing.T) {
	dir := setupCmdStateDir(t)
	handler := newHandleNetworkLoad(dir)
	bus, _ := newTestBus(t)
	defer bus.Close()

	args, _ := json.Marshal(map[string]any{"name": "local"})
	data, err := handler(args, bus)
	if err != nil {
		t.Fatalf("handler: %v", err)
	}
	if data["name"] != "local" {
		t.Errorf("name: got %v", data["name"])
	}
	nodes, ok := data["nodes"].([]any)
	if !ok || len(nodes) != 2 {
		t.Errorf("nodes: got %v", data["nodes"])
	}
}

func TestHandleNetworkLoad_WrongName(t *testing.T) {
	dir := setupCmdStateDir(t)
	handler := newHandleNetworkLoad(dir)
	bus, _ := newTestBus(t)
	defer bus.Close()

	args, _ := json.Marshal(map[string]any{"name": "mainnet"})
	_, err := handler(args, bus)
	if err == nil {
		t.Fatal("expected error")
	}
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want APIError INVALID_ARGS, got %v", err)
	}
}

func TestHandleNetworkLoad_MissingArgs(t *testing.T) {
	dir := setupCmdStateDir(t)
	handler := newHandleNetworkLoad(dir)
	bus, _ := newTestBus(t)
	defer bus.Close()

	args, _ := json.Marshal(map[string]any{}) // name omitted
	_, err := handler(args, bus)
	if err == nil {
		t.Fatal("expected error")
	}
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INVALID_ARGS" {
		t.Errorf("want INVALID_ARGS, got %v", err)
	}
}

func TestHandleNetworkLoad_StateMissing(t *testing.T) {
	dir := t.TempDir() // empty
	handler := newHandleNetworkLoad(dir)
	bus, _ := newTestBus(t)
	defer bus.Close()

	args, _ := json.Marshal(map[string]any{"name": "local"})
	_, err := handler(args, bus)
	if err == nil {
		t.Fatal("expected error")
	}
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "UPSTREAM_ERROR" {
		t.Errorf("want UPSTREAM_ERROR, got %v", err)
	}
}

func TestAllHandlers_IncludesNetworkLoad(t *testing.T) {
	handlers := allHandlers("whatever")
	if _, ok := handlers["network.load"]; !ok {
		t.Error("allHandlers missing network.load")
	}
}

// Keep the types import referenced for future table-expansion tests.
var _ = types.ResultErrorCode("NOT_SUPPORTED")
```

- [ ] **Step 5.3: Run — expect fail**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./cmd/chainbench-net/... -run 'TestHandleNetworkLoad|TestAllHandlers' 2>&1 | tail -10
```
Expected: FAIL — `undefined: newHandleNetworkLoad`, `undefined: allHandlers`, `undefined: Handler`.

- [ ] **Step 5.4: Write `handlers.go`**

Create `network/cmd/chainbench-net/handlers.go`:

```go
package main

import (
	"encoding/json"
	"fmt"

	"github.com/0xmhha/chainbench/network/internal/events"
	"github.com/0xmhha/chainbench/network/internal/state"
)

// Handler is the common signature for all command handlers.
// args is the raw JSON of cmd.args from the wire envelope; bus is available
// for mid-flight progress/events. Returns (data, nil) on success — data is
// wrapped into EmitResult(true, data). Returns (_, *APIError) for typed
// failures; any other error is treated as INTERNAL by the dispatcher.
type Handler func(args json.RawMessage, bus *events.Bus) (map[string]any, error)

// allHandlers builds the command → handler dispatch table. stateDir is
// bound via closure into handlers that need it.
func allHandlers(stateDir string) map[string]Handler {
	return map[string]Handler{
		"network.load": newHandleNetworkLoad(stateDir),
	}
}

// newHandleNetworkLoad returns the "network.load" handler closing over stateDir.
func newHandleNetworkLoad(stateDir string) Handler {
	return func(args json.RawMessage, bus *events.Bus) (map[string]any, error) {
		var req struct {
			Name string `json:"name"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
			}
		}
		if req.Name == "" {
			return nil, NewInvalidArgs("args.name is required")
		}
		if req.Name != "local" {
			return nil, NewInvalidArgs(fmt.Sprintf("only 'local' supported (got %q)", req.Name))
		}

		net, err := state.LoadActive(state.LoadActiveOptions{StateDir: stateDir, Name: req.Name})
		if err != nil {
			return nil, NewUpstream("failed to load active state", err)
		}

		// Marshal through JSON so the result is a plain map[string]any matching
		// the generated schema layout.
		raw, err := json.Marshal(net)
		if err != nil {
			return nil, NewInternal("marshal network", err)
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, NewInternal("unmarshal network", err)
		}
		return data, nil
	}
}
```

- [ ] **Step 5.5: Run — expect pass**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./cmd/chainbench-net/... -v -run 'TestHandleNetworkLoad|TestAllHandlers'
```
Expected: 5 subtests PASS.

- [ ] **Step 5.6: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/cmd/chainbench-net/handlers.go network/cmd/chainbench-net/handlers_test.go network/cmd/chainbench-net/testdata/
git commit -m "network: add cmd network.load handler"
```

---

## Task 6: `cmd/chainbench-net/run.go` — run subcommand + dispatcher

**Files:**
- Create: `network/cmd/chainbench-net/run.go`
- Create: `network/cmd/chainbench-net/run_test.go`

- [ ] **Step 6.1: Write failing test**

Create `network/cmd/chainbench-net/run_test.go`:

```go
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/network/internal/events"
	"github.com/0xmhha/chainbench/network/internal/wire"
)

// setupRunStateDir reuses cmd testdata via an in-process temp dir.
func setupRunStateDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, name := range []string{"pids.json", "current-profile.yaml"} {
		data, _ := os.ReadFile(filepath.Join("testdata", name))
		_ = os.WriteFile(filepath.Join(dir, name), data, 0o644)
	}
	return dir
}

func TestRunOnce_HappyPath_NetworkLoad(t *testing.T) {
	dir := setupRunStateDir(t)
	stdin := strings.NewReader(`{"command":"network.load","args":{"name":"local"}}`)
	var stdout, stderr bytes.Buffer
	handlers := allHandlers(dir)

	err := runOnce(stdin, &stdout, &stderr, handlers)
	if err != nil {
		t.Fatalf("runOnce: %v", err)
	}

	// Last line must be a successful result; decode all lines and verify.
	var terminator *wire.ResultMessage
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		msg, derr := wire.DecodeMessage(scanner.Bytes())
		if derr != nil {
			t.Fatalf("decode line %q: %v", scanner.Bytes(), derr)
		}
		if rm, ok := msg.(wire.ResultMessage); ok {
			rm := rm
			terminator = &rm
		}
	}
	if terminator == nil {
		t.Fatal("no result line in output")
	}
	if terminator.Ok != true {
		t.Fatalf("ok: got %v", terminator.Ok)
	}
	if terminator.Data == nil {
		t.Fatal("result.data is nil")
	}
}

func TestRunOnce_MalformedStdin_ProtocolError(t *testing.T) {
	stdin := strings.NewReader(`not json`)
	var stdout, stderr bytes.Buffer
	err := runOnce(stdin, &stdout, &stderr, map[string]Handler{})
	if err == nil {
		t.Fatal("expected error")
	}
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "PROTOCOL_ERROR" {
		t.Errorf("want PROTOCOL_ERROR, got %v", err)
	}
	// Result terminator must still have been emitted.
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	last := lines[len(lines)-1]
	var envelope struct {
		Type string `json:"type"`
		OK   bool   `json:"ok"`
	}
	_ = json.Unmarshal([]byte(last), &envelope)
	if envelope.Type != "result" || envelope.OK != false {
		t.Errorf("last line not an error result: %q", last)
	}
}

func TestRunOnce_UnknownCommand_NotSupported(t *testing.T) {
	// command.load is valid enum but no handler registered — simulate
	// by passing an empty handler table.
	stdin := strings.NewReader(`{"command":"network.load","args":{}}`)
	var stdout, stderr bytes.Buffer
	err := runOnce(stdin, &stdout, &stderr, map[string]Handler{})
	if err == nil {
		t.Fatal("expected error")
	}
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "NOT_SUPPORTED" {
		t.Errorf("want NOT_SUPPORTED, got %v", err)
	}
}

func TestRunOnce_HandlerPanic_Internal(t *testing.T) {
	stdin := strings.NewReader(`{"command":"network.load","args":{"name":"local"}}`)
	var stdout, stderr bytes.Buffer
	panicking := map[string]Handler{
		"network.load": func(args json.RawMessage, bus *events.Bus) (map[string]any, error) {
			panic("boom")
		},
	}
	err := runOnce(stdin, &stdout, &stderr, panicking)
	if err == nil {
		t.Fatal("expected error")
	}
	var api *APIError
	if !errors.As(err, &api) || string(api.Code) != "INTERNAL" {
		t.Errorf("want INTERNAL, got %v", err)
	}
}
```

- [ ] **Step 6.2: Run — expect fail**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./cmd/chainbench-net/... -run TestRunOnce 2>&1 | tail -5
```
Expected: FAIL — `undefined: runOnce`, `undefined: newRunCmd`.

- [ ] **Step 6.3: Write `run.go`**

Create `network/cmd/chainbench-net/run.go`:

```go
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/network/internal/events"
	"github.com/0xmhha/chainbench/network/internal/types"
	"github.com/0xmhha/chainbench/network/internal/wire"
)

// newRunCmd builds the `run` subcommand. It reads a wire command envelope
// from stdin, dispatches to a handler, and emits a result NDJSON terminator
// on stdout. Structured logs go to stderr.
func newRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:           "run",
		Short:         "Execute one wire command envelope from stdin",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			stateDir := os.Getenv("CHAINBENCH_STATE_DIR")
			if stateDir == "" {
				stateDir = "state"
			}
			return runOnce(cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr(), allHandlers(stateDir))
		},
	}
}

// runOnce decodes one wire command envelope from stdin, dispatches to a
// handler, and emits a result terminator on stdout. Structured logs go to
// stderr. Returns the error (if any) so the caller can map to an exit code.
// Safe against handler panics via deferred recover.
func runOnce(stdin io.Reader, stdout, stderr io.Writer, handlers map[string]Handler) (returnErr error) {
	wire.SetupLoggerTo(stderr, slog.LevelInfo)
	emitter := wire.NewEmitter(stdout)
	bus := events.NewBus(emitter)
	defer bus.Close()

	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprintf("panic: %v", r)
			slog.Error("handler panic", "panic", r)
			_ = emitter.EmitResultError(types.ResultErrorCode("INTERNAL"), msg)
			returnErr = NewInternal(msg, nil)
		}
	}()

	cmd, err := wire.DecodeCommand(stdin)
	if err != nil {
		_ = emitter.EmitResultError(types.ResultErrorCode("PROTOCOL_ERROR"), err.Error())
		return NewProtocolError("decode command envelope", err)
	}

	handler, ok := handlers[string(cmd.Command)]
	if !ok {
		msg := fmt.Sprintf("no handler for command %q", cmd.Command)
		_ = emitter.EmitResultError(types.ResultErrorCode("NOT_SUPPORTED"), msg)
		return NewNotSupported(msg)
	}

	raw, _ := json.Marshal(cmd.Args)

	data, err := handler(raw, bus)
	if err != nil {
		var api *APIError
		if errors.As(err, &api) {
			_ = emitter.EmitResultError(api.Code, api.Message)
			return err
		}
		_ = emitter.EmitResultError(types.ResultErrorCode("INTERNAL"), err.Error())
		return NewInternal(err.Error(), err)
	}

	if emitErr := emitter.EmitResult(true, data); emitErr != nil {
		return fmt.Errorf("emit result: %w", emitErr)
	}
	return nil
}
```

- [ ] **Step 6.4: Run — expect pass**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./cmd/chainbench-net/... -v -run TestRunOnce
```
Expected: 4 subtests PASS (TestRunOnce_HappyPath_NetworkLoad, TestRunOnce_MalformedStdin_ProtocolError, TestRunOnce_UnknownCommand_NotSupported, TestRunOnce_HandlerPanic_Internal).

- [ ] **Step 6.5: Race + build/vet/fmt**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test -race ./cmd/chainbench-net/... && go build ./... && go vet ./... && gofmt -l cmd/chainbench-net/
```
Expected: all clean.

- [ ] **Step 6.6: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/cmd/chainbench-net/run.go network/cmd/chainbench-net/run_test.go
git commit -m "network: add cmd run subcommand with wire dispatcher and panic recovery"
```

---

## Task 7: Wire `run` into root + main exit code + E2E test

**Files:**
- Modify: `network/cmd/chainbench-net/main.go`
- Create: `network/cmd/chainbench-net/e2e_test.go`

- [ ] **Step 7.1: Write E2E failing test**

Create `network/cmd/chainbench-net/e2e_test.go`:

```go
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/network/internal/wire"
	"github.com/0xmhha/chainbench/network/schema"
)

func TestE2E_NetworkLoad_ViaRootCommand(t *testing.T) {
	// Prepare state dir.
	dir := t.TempDir()
	for _, name := range []string{"pids.json", "current-profile.yaml"} {
		data, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("CHAINBENCH_STATE_DIR", dir)

	// Drive cobra root via in-memory IO.
	var stdout, stderr bytes.Buffer
	root := newRootCmd()
	root.SetIn(strings.NewReader(`{"command":"network.load","args":{"name":"local"}}`))
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"run"})

	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v\nstderr: %s", err, stderr.String())
	}

	// Collect lines; find the result terminator.
	var resultLine []byte
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		msg, derr := wire.DecodeMessage(line)
		if derr != nil {
			t.Fatalf("decode %q: %v", line, derr)
		}
		if _, ok := msg.(wire.ResultMessage); ok {
			resultLine = line
		}
	}
	if resultLine == nil {
		t.Fatal("no result line in output")
	}

	// Parse terminator and cross-validate its data against the network schema.
	var res struct {
		Type string                 `json:"type"`
		Ok   bool                   `json:"ok"`
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(resultLine, &res); err != nil {
		t.Fatalf("unmarshal terminator: %v", err)
	}
	if !res.Ok {
		t.Fatalf("not ok: %s", resultLine)
	}
	raw, err := json.Marshal(res.Data)
	if err != nil {
		t.Fatalf("marshal data: %v", err)
	}
	if err := schema.ValidateBytes("network", raw); err != nil {
		t.Fatalf("network schema validation failed: %v\nraw: %s", err, raw)
	}

	// Schema validity also on the full terminator line against event schema.
	if err := schema.ValidateBytes("event", resultLine); err != nil {
		t.Fatalf("event schema validation failed: %v\nline: %s", err, resultLine)
	}
}

func TestE2E_ExitCodeViaAPIError(t *testing.T) {
	// Verify that an intentional handler error propagates through Execute()
	// and that main's exitCode() maps it correctly. Since we can't os.Exit
	// in a test, we call exitCode directly on the returned error.
	var stdout, stderr bytes.Buffer
	root := newRootCmd()
	root.SetIn(strings.NewReader(`not json`)) // triggers PROTOCOL_ERROR
	root.SetOut(&stdout)
	root.SetErr(&stderr)
	root.SetArgs([]string{"run"})

	err := root.Execute()
	if err == nil {
		t.Fatal("expected error")
	}
	if code := exitCode(err); code != 3 {
		t.Errorf("exit code: got %d, want 3 (PROTOCOL_ERROR)", code)
	}
}
```

- [ ] **Step 7.2: Run — expect fail (run subcommand not registered)**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test ./cmd/chainbench-net/... -run TestE2E 2>&1 | tail -5
```
Expected: FAIL — `unknown command "run"` or equivalent, because `newRootCmd` does not include `run`.

- [ ] **Step 7.3: Update `main.go`**

Read current `network/cmd/chainbench-net/main.go` and update two sections:

Replace:
```go
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "chainbench-net",
		Short:         "Network abstraction layer for chainbench",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newVersionCmd())
	return root
}
```
with:
```go
func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "chainbench-net",
		Short:         "Network abstraction layer for chainbench",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(newVersionCmd())
	root.AddCommand(newRunCmd())
	return root
}
```

Replace:
```go
func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```
with:
```go
func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(exitCode(err))
	}
}
```

- [ ] **Step 7.4: Run E2E + full package — expect pass**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test -race ./cmd/chainbench-net/...
```
Expected: all tests pass (existing version + 8 error/handler/run + 2 E2E).

- [ ] **Step 7.5: Manual smoke test of built binary**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
go build -o /tmp/chainbench-net-smoke ./network/cmd/chainbench-net
# Use the real state directory — should succeed if chain is up, or emit UPSTREAM_ERROR otherwise.
echo '{"command":"network.load","args":{"name":"local"}}' | /tmp/chainbench-net-smoke run
echo "exit=$?"
# Clean up.
rm /tmp/chainbench-net-smoke
```
Expected: one line of NDJSON `{"type":"result", ...}`. Exit code 0 if state exists; 1 with UPSTREAM_ERROR otherwise. Either outcome proves the pipeline works.

- [ ] **Step 7.6: Coverage check**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network && go test -cover ./cmd/chainbench-net/... ./internal/state/...
```
Expected: `state` ≥ 85%, `cmd/chainbench-net` ≥ 80%.

- [ ] **Step 7.7: Commit**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench
git add network/cmd/chainbench-net/main.go network/cmd/chainbench-net/e2e_test.go
git commit -m "network: wire run subcommand into root with end-to-end test"
```

---

## Final verification

- [ ] **Module-wide green**

```bash
cd /Users/wm-it-22-00661/Work/github/tools/chainbench/network
go build ./... && go test -race ./... && go vet ./... && gofmt -l .
go build -tags tools ./...
```
Expected: all clean.

- [ ] **Commit list**

```bash
git log --oneline b3fa1af..HEAD
```
Expected 7 commits:
1. `network: add state profile YAML parser`
2. `network: add state pids.json parser`
3. `network: add state LoadActive network builder with schema cross-check`
4. `network: add cmd error type with exit code mapping`
5. `network: add cmd network.load handler`
6. `network: add cmd run subcommand with wire dispatcher and panic recovery`
7. `network: wire run subcommand into root with end-to-end test`

---

## Out of scope (explicit)

- LocalDriver subprocess wrapping (Sprint 2b.2)
- `node.rpc` / `node.start` / `node.stop` / `node.tail_log` / `network.probe` (Sprint 2b.2+)
- bash client `lib/network_client.sh` (Sprint 2c)
- `network.load` for names other than "local" (Sprint 2b.3 with `state/networks/*.json`)
- Signer / keystore (Sprint 4)
