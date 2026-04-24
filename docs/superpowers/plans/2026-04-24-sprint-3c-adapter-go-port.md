# Sprint 3c — Adapter Go Port Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development.

**Goal:** Ship a Go `adapters` package with `Adapter` interface, full `stablenet` implementation (GenerateGenesis + GenerateToml + ExtraStartFlags + ConsensusRpcNamespace), and `wbft` / `wemix` skeletons. Equivalence verified via pinned golden files. Bash adapters remain in place; Go adapter is additive.

**Architecture:** New package at `network/internal/adapters/` with per-chain-type subpackages. `Load(chainType)` factory returns the right adapter. stablenet port mirrors the Python logic in `lib/adapters/stablenet.sh` with direct `strings.Replace` for template substitution and `encoding/json` for structured pieces.

**Tech Stack:** Go 1.25 stdlib (`encoding/json`, `strings`, `fmt`, `os`), TOML parsing via `github.com/pelletier/go-toml/v2` (new dep, lightweight) for contract-test parsing only.

Spec: `docs/superpowers/specs/2026-04-24-sprint-3c-adapter-go-port.md`

---

## Commit Discipline

- English, no Co-Authored-By, no "Generated with Claude Code", no emoji.

## File Structure

**Create:**
- `network/internal/adapters/adapter.go`
- `network/internal/adapters/adapter_test.go`
- `network/internal/adapters/stablenet/stablenet.go`
- `network/internal/adapters/stablenet/stablenet_test.go`
- `network/internal/adapters/stablenet/stablenet_contract_test.go`
- `network/internal/adapters/wbft/wbft.go`
- `network/internal/adapters/wbft/wbft_test.go`
- `network/internal/adapters/wemix/wemix.go`
- `network/internal/adapters/wemix/wemix_test.go`
- `network/internal/adapters/testdata/profile.json` (fixture)
- `network/internal/adapters/testdata/metadata.json` (fixture)
- `network/internal/adapters/testdata/golden/stablenet-genesis.json`
- `network/internal/adapters/testdata/golden/stablenet-config-node1.toml`
- `network/internal/adapters/testdata/golden/stablenet-config-node2.toml`

**Modify:**
- `network/go.mod`, `network/go.sum` — add toml parser dep
- `docs/VISION_AND_ROADMAP.md` — mark 3c complete

---

## Task 1 — Adapter interface + Load factory + wbft/wemix skeletons

**Files:**
- Create: `network/internal/adapters/adapter.go`
- Create: `network/internal/adapters/adapter_test.go`
- Create: `network/internal/adapters/wbft/wbft.go`
- Create: `network/internal/adapters/wbft/wbft_test.go`
- Create: `network/internal/adapters/wemix/wemix.go`
- Create: `network/internal/adapters/wemix/wemix_test.go`

- [ ] **Step 1: Write failing test for Load factory**

```go
// network/internal/adapters/adapter_test.go
package adapters_test

import (
    "errors"
    "testing"

    "github.com/0xmhha/chainbench/network/internal/adapters"
)

func TestLoad_Stablenet(t *testing.T) {
    a, err := adapters.Load("stablenet")
    if err != nil {
        t.Fatalf("Load(stablenet): %v", err)
    }
    if a == nil {
        t.Fatal("Load returned nil adapter")
    }
    if ns := a.ConsensusRpcNamespace(); ns != "istanbul" {
        t.Errorf("stablenet namespace = %q, want istanbul", ns)
    }
}

func TestLoad_WBFT(t *testing.T) {
    a, err := adapters.Load("wbft")
    if err != nil {
        t.Fatalf("Load(wbft): %v", err)
    }
    if ns := a.ConsensusRpcNamespace(); ns != "istanbul" {
        t.Errorf("wbft namespace = %q, want istanbul", ns)
    }
}

func TestLoad_Wemix(t *testing.T) {
    a, err := adapters.Load("wemix")
    if err != nil {
        t.Fatalf("Load(wemix): %v", err)
    }
    if ns := a.ConsensusRpcNamespace(); ns != "wemix" {
        t.Errorf("wemix namespace = %q, want wemix", ns)
    }
}

func TestLoad_UnknownChainType(t *testing.T) {
    _, err := adapters.Load("ethereum")
    if !errors.Is(err, adapters.ErrUnknownChainType) {
        t.Errorf("err = %v, want ErrUnknownChainType", err)
    }
}
```

- [ ] **Step 2: Implement adapter.go**

```go
// Package adapters defines the chain-type-specific strategy for initializing
// a local chainbench network. Bash adapters under lib/adapters/ remain the
// current execution path for `chainbench init`; this package is additive,
// available to future chainbench-net subcommands.
package adapters

import (
    "context"
    "errors"

    "github.com/0xmhha/chainbench/network/internal/adapters/stablenet"
    "github.com/0xmhha/chainbench/network/internal/adapters/wbft"
    "github.com/0xmhha/chainbench/network/internal/adapters/wemix"
)

type Role string

const (
    RoleValidator Role = "validator"
    RoleEndpoint  Role = "endpoint"
)

type Profile map[string]any
type Metadata map[string]any

type GenesisInput struct {
    Profile       Profile
    Metadata      Metadata
    TemplateBytes []byte
    OutputPath    string
    NumValidators int
    BaseP2P       int
}

type TomlInput struct {
    Metadata      Metadata
    TemplateBytes []byte
    WriteDir      string
    TotalNodes    int
    NumValidators int
    BaseP2P       int
    BaseHTTP      int
    BaseWS        int
    BaseAuth      int
    BaseMetrics   int
}

type Adapter interface {
    GenerateGenesis(ctx context.Context, input GenesisInput) error
    GenerateToml(ctx context.Context, input TomlInput) error
    ExtraStartFlags(role Role) string
    ConsensusRpcNamespace() string
}

var (
    ErrUnknownChainType = errors.New("adapters: unknown chain type")
    ErrNotImplemented   = errors.New("adapters: operation not implemented for this chain type")
)

// Load returns the Adapter registered for chainType, or ErrUnknownChainType
// when no adapter handles it.
func Load(chainType string) (Adapter, error) {
    switch chainType {
    case "stablenet":
        return stablenet.New(), nil
    case "wbft":
        return wbft.New(), nil
    case "wemix":
        return wemix.New(), nil
    default:
        return nil, ErrUnknownChainType
    }
}
```

**Note:** this file imports the three subpackages, which means Tasks 1 implements
`wbft.New()`, `wemix.New()` first (skeletons), and Task 2 implements
`stablenet.New()`. For the factory test, Task 1 must provide a stub
`stablenet.New()` that returns ErrNotImplemented for Generate* — Task 2 fleshes
it out.

- [ ] **Step 3: Implement wbft skeleton**

```go
// network/internal/adapters/wbft/wbft.go
package wbft

import (
    "context"
    "errors"
)

type Adapter struct{}

// sentinel proxy — callers import adapters.ErrNotImplemented directly.
// This package re-exports via ErrUnsupported for local use; adapters.Load
// returns this same sentinel through its own error wrapping.
var errNotImplemented = errors.New("adapters/wbft: operation not implemented (placeholder)")

func New() *Adapter { return &Adapter{} }

func (*Adapter) GenerateGenesis(ctx context.Context, _ any) error { return errNotImplemented }
func (*Adapter) GenerateToml(ctx context.Context, _ any) error    { return errNotImplemented }
func (*Adapter) ExtraStartFlags(role string) string               { return "--allow-insecure-unlock" }
func (*Adapter) ConsensusRpcNamespace() string                    { return "istanbul" }
```

**Problem with this shape:** wbft's Adapter methods don't match the interface
signatures (uses `any` / `string` instead of GenesisInput / TomlInput / Role).
Fix: import the parent package types.

Go-module-wise, adapters/wbft cannot import adapters (import cycle — adapters
imports wbft). Solution: put the interface + types in a shared internal
package, or define GenesisInput/TomlInput/Role in each subpackage.

**Revised approach**: move types into `adapters/spec/` (leaf package with no
imports of siblings), have `adapters/` and `adapters/stablenet`, `adapters/wbft`,
`adapters/wemix` all import from `adapters/spec/`.

```
network/internal/adapters/
├── adapter.go             # Load() factory; imports spec + subpackages
├── adapter_test.go
├── spec/
│   ├── types.go           # Role, GenesisInput, TomlInput, Adapter interface,
│                          #   ErrNotImplemented, ErrUnknownChainType
│   └── types_test.go      # (optional — no behavior to test yet)
├── stablenet/
│   ├── stablenet.go       # imports spec
│   └── stablenet_test.go
├── wbft/
│   ├── wbft.go            # imports spec
│   └── wbft_test.go
└── wemix/
    ├── wemix.go           # imports spec
    └── wemix_test.go
```

**Revised step sequence:** implement `spec/types.go` first, then subpackages,
then the top-level `adapter.go` factory.

- [ ] **Step 3a: Create spec/types.go**

```go
// Package spec holds shared adapter types + interface + error sentinels.
// Lives at the leaf of the import graph so each chain adapter and the top
// level Load() can import it without cycles.
package spec

import (
    "context"
    "errors"
)

type Role string

const (
    RoleValidator Role = "validator"
    RoleEndpoint  Role = "endpoint"
)

type Profile map[string]any
type Metadata map[string]any

type GenesisInput struct {
    Profile       Profile
    Metadata      Metadata
    TemplateBytes []byte
    OutputPath    string
    NumValidators int
    BaseP2P       int
}

type TomlInput struct {
    Metadata      Metadata
    TemplateBytes []byte
    WriteDir      string
    TotalNodes    int
    NumValidators int
    BaseP2P       int
    BaseHTTP      int
    BaseWS        int
    BaseAuth      int
    BaseMetrics   int
}

type Adapter interface {
    GenerateGenesis(ctx context.Context, input GenesisInput) error
    GenerateToml(ctx context.Context, input TomlInput) error
    ExtraStartFlags(role Role) string
    ConsensusRpcNamespace() string
}

var (
    ErrUnknownChainType = errors.New("adapters: unknown chain type")
    ErrNotImplemented   = errors.New("adapters: operation not implemented for this chain type")
)
```

- [ ] **Step 3b: Create wbft/wbft.go**

```go
package wbft

import (
    "context"

    "github.com/0xmhha/chainbench/network/internal/adapters/spec"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

func (*Adapter) GenerateGenesis(ctx context.Context, _ spec.GenesisInput) error {
    return spec.ErrNotImplemented
}

func (*Adapter) GenerateToml(ctx context.Context, _ spec.TomlInput) error {
    return spec.ErrNotImplemented
}

func (*Adapter) ExtraStartFlags(spec.Role) string { return "--allow-insecure-unlock" }

func (*Adapter) ConsensusRpcNamespace() string { return "istanbul" }
```

- [ ] **Step 3c: Create wemix/wemix.go**

Same pattern as wbft; `ConsensusRpcNamespace()` returns `"wemix"`.

- [ ] **Step 3d: Create stub stablenet/stablenet.go**

```go
package stablenet

import (
    "context"

    "github.com/0xmhha/chainbench/network/internal/adapters/spec"
)

type Adapter struct{}

func New() *Adapter { return &Adapter{} }

// TODO (Task 2): real implementation.
func (*Adapter) GenerateGenesis(ctx context.Context, _ spec.GenesisInput) error {
    return spec.ErrNotImplemented
}

func (*Adapter) GenerateToml(ctx context.Context, _ spec.TomlInput) error {
    return spec.ErrNotImplemented
}

func (*Adapter) ExtraStartFlags(role spec.Role) string {
    flags := "--allow-insecure-unlock --rpc.enabledeprecatedpersonal --rpc.allow-unprotected-txs"
    if role == spec.RoleValidator {
        flags += " --mine"
    }
    return flags
}

func (*Adapter) ConsensusRpcNamespace() string { return "istanbul" }
```

- [ ] **Step 3e: Create top-level adapter.go factory**

```go
package adapters

import (
    "github.com/0xmhha/chainbench/network/internal/adapters/spec"
    "github.com/0xmhha/chainbench/network/internal/adapters/stablenet"
    "github.com/0xmhha/chainbench/network/internal/adapters/wbft"
    "github.com/0xmhha/chainbench/network/internal/adapters/wemix"
)

// Re-export shared types so callers only need one import.
type (
    Adapter      = spec.Adapter
    GenesisInput = spec.GenesisInput
    TomlInput    = spec.TomlInput
    Role         = spec.Role
    Profile      = spec.Profile
    Metadata     = spec.Metadata
)

const (
    RoleValidator = spec.RoleValidator
    RoleEndpoint  = spec.RoleEndpoint
)

var (
    ErrUnknownChainType = spec.ErrUnknownChainType
    ErrNotImplemented   = spec.ErrNotImplemented
)

func Load(chainType string) (Adapter, error) {
    switch chainType {
    case "stablenet":
        return stablenet.New(), nil
    case "wbft":
        return wbft.New(), nil
    case "wemix":
        return wemix.New(), nil
    default:
        return nil, ErrUnknownChainType
    }
}
```

- [ ] **Step 4: Add skeleton subpackage tests**

Each of wbft/wemix/stablenet gets a minimal test file:

```go
// network/internal/adapters/wbft/wbft_test.go
package wbft

import (
    "context"
    "errors"
    "testing"

    "github.com/0xmhha/chainbench/network/internal/adapters/spec"
)

func TestAdapter_ConsensusRpcNamespace(t *testing.T) {
    if ns := New().ConsensusRpcNamespace(); ns != "istanbul" {
        t.Errorf("got %q, want istanbul", ns)
    }
}

func TestAdapter_ExtraStartFlags(t *testing.T) {
    want := "--allow-insecure-unlock"
    if flags := New().ExtraStartFlags(spec.RoleValidator); flags != want {
        t.Errorf("got %q, want %q", flags, want)
    }
}

func TestAdapter_GenerateGenesis_NotImplemented(t *testing.T) {
    err := New().GenerateGenesis(context.Background(), spec.GenesisInput{})
    if !errors.Is(err, spec.ErrNotImplemented) {
        t.Errorf("err = %v, want ErrNotImplemented", err)
    }
}

func TestAdapter_GenerateToml_NotImplemented(t *testing.T) {
    err := New().GenerateToml(context.Background(), spec.TomlInput{})
    if !errors.Is(err, spec.ErrNotImplemented) {
        t.Errorf("err = %v, want ErrNotImplemented", err)
    }
}
```

Replicate for wemix (with `"wemix"` namespace). For stablenet the skeleton tests
will be superseded in Task 2 — include only the role-flags + namespace tests for now.

- [ ] **Step 5: Run tests**

```bash
cd network && go test ./internal/adapters/... -v -count=1 -timeout=30s
```

Expected: Load factory test + skeleton tests PASS.

- [ ] **Step 6: Full suite regression check**

```bash
cd network && go test ./... -count=1 -timeout=60s
go -C network vet ./...
gofmt -l network/
```

- [ ] **Step 7: Commit**

```bash
git add network/internal/adapters/
git commit -m "feat(adapters): introduce Adapter interface + Load factory + skeletons

New network/internal/adapters package exposes a chain-type-agnostic
init layer for future chainbench-net subcommands. The Adapter interface
defines GenerateGenesis / GenerateToml / ExtraStartFlags /
ConsensusRpcNamespace; shared types live in a leaf 'spec' subpackage
to avoid import cycles between the Load factory and per-chain impls.

stablenet / wbft / wemix adapters exist as skeletons — ExtraStartFlags
and ConsensusRpcNamespace return the bash-adapter-equivalent values;
GenerateGenesis / GenerateToml return spec.ErrNotImplemented pending
the Task 2 port. Bash adapters under lib/adapters/ remain the active
'chainbench init' path — the Go surface is additive."
```

---

## Task 2 — stablenet GenerateGenesis port

**Files:**
- Modify: `network/internal/adapters/stablenet/stablenet.go`
- Create: `network/internal/adapters/stablenet/stablenet_test.go`
- Create: `network/internal/adapters/testdata/profile.json`
- Create: `network/internal/adapters/testdata/metadata.json`

- [ ] **Step 1: Create test fixtures**

`testdata/profile.json` — minimal profile that exercises overrides:

```json
{
  "chain": { "chain_id": 8283 },
  "genesis": {
    "overrides": {
      "wbft": {
        "requestTimeoutSeconds": 2,
        "blockPeriodSeconds": 1,
        "epochLength": 140,
        "proposerPolicy": 0
      },
      "systemContracts": {
        "govValidator": { "params": {} },
        "govMinter":    { "params": {} }
      },
      "alloc": {
        "0x1234567890abcdef1234567890abcdef12345678": "0xDEADBEEF"
      },
      "extraData": "0x00"
    }
  }
}
```

`testdata/metadata.json` — 2 nodes with deterministic addresses/keys:

```json
{
  "nodes": [
    {
      "index":        1,
      "address":      "0xAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
      "publicKey":    "aaaaaaaaaaaa",
      "blsPublicKey": "0xaaaa"
    },
    {
      "index":        2,
      "address":      "0xBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB",
      "publicKey":    "bbbbbbbbbbbb",
      "blsPublicKey": "0xbbbb"
    }
  ],
  "extraData": "0xfeedface"
}
```

- [ ] **Step 2: Write failing test**

```go
// network/internal/adapters/stablenet/stablenet_test.go
package stablenet_test

import (
    "context"
    "encoding/json"
    "os"
    "path/filepath"
    "testing"

    "github.com/0xmhha/chainbench/network/internal/adapters/spec"
    "github.com/0xmhha/chainbench/network/internal/adapters/stablenet"
)

func loadFixture(t *testing.T, name string) []byte {
    t.Helper()
    data, err := os.ReadFile(filepath.Join("..", "testdata", name))
    if err != nil {
        t.Fatalf("read %s: %v", name, err)
    }
    return data
}

func TestGenerateGenesis_Happy(t *testing.T) {
    profileRaw := loadFixture(t, "profile.json")
    metadataRaw := loadFixture(t, "metadata.json")
    templateRaw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "templates", "genesis.template.json"))
    if err != nil {
        t.Fatalf("load genesis template: %v", err)
    }

    var profile spec.Profile
    _ = json.Unmarshal(profileRaw, &profile)
    var metadata spec.Metadata
    _ = json.Unmarshal(metadataRaw, &metadata)

    outputPath := filepath.Join(t.TempDir(), "genesis.json")
    in := spec.GenesisInput{
        Profile:       profile,
        Metadata:      metadata,
        TemplateBytes: templateRaw,
        OutputPath:    outputPath,
        NumValidators: 2,
        BaseP2P:       30303,
    }
    if err := stablenet.New().GenerateGenesis(context.Background(), in); err != nil {
        t.Fatalf("GenerateGenesis: %v", err)
    }

    raw, err := os.ReadFile(outputPath)
    if err != nil {
        t.Fatalf("read output: %v", err)
    }
    var got map[string]any
    if err := json.Unmarshal(raw, &got); err != nil {
        t.Fatalf("output is not valid JSON: %v", err)
    }

    // chainId present inside config.
    cfg, ok := got["config"].(map[string]any)
    if !ok {
        t.Fatalf("config missing / wrong type: %T", got["config"])
    }
    if cid, _ := cfg["chainId"].(float64); cid != 8283 {
        t.Errorf("chainId = %v, want 8283", cfg["chainId"])
    }

    // alloc has the user-specified override + default balances for 2 nodes +
    // the CREATE2 proxy address.
    alloc, ok := got["alloc"].(map[string]any)
    if !ok {
        t.Fatalf("alloc missing")
    }
    if _, ok := alloc["1234567890abcdef1234567890abcdef12345678"]; !ok {
        t.Errorf("user-specified alloc entry missing")
    }
    if _, ok := alloc["4e59b44847b379578588920cA78FbF26c0B4956C"]; !ok {
        // note: bash lowercases the key check but preserves original casing in
        // the output; this test allows either case. Adjust test logic below
        // if Go port chose a specific case.
        lowered := "4e59b44847b379578588920ca78fbf26c0b4956c"
        if _, ok := alloc[lowered]; !ok {
            t.Errorf("CREATE2 proxy alloc entry missing (expected key 4e59b44...)")
        }
    }

    // extraData — metadata extraData takes precedence over genesis overrides.
    if ed, _ := got["extraData"].(string); ed != "0xfeedface" {
        t.Errorf("extraData = %q, want 0xfeedface", ed)
    }
}

func TestGenerateGenesis_EmptyTemplate(t *testing.T) {
    in := spec.GenesisInput{
        TemplateBytes: nil,
        OutputPath:    filepath.Join(t.TempDir(), "out.json"),
        NumValidators: 1,
    }
    if err := stablenet.New().GenerateGenesis(context.Background(), in); err == nil {
        t.Fatal("expected error for empty template")
    }
}
```

- [ ] **Step 3: Implement GenerateGenesis**

Port the Python logic from `lib/adapters/stablenet.sh:14-118` to Go. Key steps:

1. Decode profile + metadata (already `map[string]any` — drill in with type assertions + zero-value defaults).
2. Extract values: chain_id, WBFT params, system contracts, alloc overrides, extraData.
3. Slice metadata.nodes to `[:num_validators]`; extract addresses + BLS keys.
4. Build members CSV (`strings.Join`) and inject into `govValidator`, `govMinter`, `govMasterMinter`, `govCouncil` params.
5. Build alloc map: default balance (`"0x84595161401484a000000"`) for each node + user-specified overrides + CREATE2 proxy contract.
6. Determine final extraData: metadata's takes precedence, falls back to override, falls back to `"0x00"`.
7. Do `strings.Replace` on template for each placeholder.
8. Validate result parses as JSON.
9. Write to OutputPath.

Skeleton:
```go
package stablenet

import (
    "context"
    "encoding/json"
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/0xmhha/chainbench/network/internal/adapters/spec"
)

const create2ProxyAddress = "4e59b44847b379578588920cA78FbF26c0B4956C"
const defaultAllocBalance = "0x84595161401484a000000"

func (*Adapter) GenerateGenesis(ctx context.Context, in spec.GenesisInput) error {
    if len(in.TemplateBytes) == 0 {
        return fmt.Errorf("stablenet.GenerateGenesis: template required")
    }
    if in.NumValidators <= 0 {
        return fmt.Errorf("stablenet.GenerateGenesis: num_validators must be > 0")
    }
    // ...extract chainId, wbft params, etc via getMap / getString / getInt helpers
    // (handle profile.chain.chain_id = 8283 default)
    // ...build validators, blsKeys, membersCSV
    // ...build systemContracts + alloc
    // ...replace placeholders
    // ...validate JSON
    // ...write file
    return nil
}

// Small helpers that don't panic on missing keys / wrong types; fall back to
// sensible defaults. Mirrors the bash + Python leniency.
func getMap(m map[string]any, key string) map[string]any { ... }
func getInt(m map[string]any, key string, def int64) int64 { ... }
func getString(m map[string]any, key string, def string) string { ... }
```

Fill in the body. Keep it under ~150 lines. Reference the Python source for exact semantics.

- [ ] **Step 4: Run tests**

```bash
cd network && go test ./internal/adapters/stablenet/... -v -count=1 -timeout=30s
```

Iterate until green.

- [ ] **Step 5: Commit**

```bash
git add network/internal/adapters/stablenet/ network/internal/adapters/testdata/
git commit -m "feat(adapters/stablenet): port GenerateGenesis to Go

Direct port of lib/adapters/stablenet.sh's Python genesis generator.
Reads profile overrides (wbft params, system contracts, alloc),
computes validator + BLS key CSVs from metadata, substitutes template
placeholders via strings.Replace, validates the result parses as JSON,
writes to input.OutputPath.

Tests cover the happy path (chainId, alloc, extraData precedence) and
the empty-template guard. Fixtures live in testdata/ for Task 3
(GenerateToml) and Task 4 (golden-file contract) reuse."
```

---

## Task 3 — stablenet GenerateToml port

**Files:**
- Modify: `network/internal/adapters/stablenet/stablenet.go`
- Modify: `network/internal/adapters/stablenet/stablenet_test.go`

- [ ] **Step 1: Write failing test**

```go
func TestGenerateToml_Happy(t *testing.T) {
    metadataRaw := loadFixture(t, "metadata.json")
    templateRaw, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "templates", "node.template.toml"))
    if err != nil {
        t.Fatalf("load node template: %v", err)
    }
    var metadata spec.Metadata
    _ = json.Unmarshal(metadataRaw, &metadata)

    dir := t.TempDir()
    in := spec.TomlInput{
        Metadata:      metadata,
        TemplateBytes: templateRaw,
        WriteDir:      dir,
        TotalNodes:    2,
        NumValidators: 2,
        BaseP2P:       30303,
        BaseHTTP:      8545,
        BaseWS:        8546,
        BaseAuth:      8551,
        BaseMetrics:   6060,
    }
    if err := stablenet.New().GenerateToml(context.Background(), in); err != nil {
        t.Fatalf("GenerateToml: %v", err)
    }

    for i := 1; i <= 2; i++ {
        name := fmt.Sprintf("config_node%d.toml", i)
        path := filepath.Join(dir, name)
        data, err := os.ReadFile(path)
        if err != nil {
            t.Errorf("%s: %v", name, err)
            continue
        }
        s := string(data)
        // Assert per-node HTTP port is sequential
        wantPort := fmt.Sprintf("HTTPPort = %d", 8545+i-1)
        if !strings.Contains(s, wantPort) {
            t.Errorf("%s missing %q", name, wantPort)
        }
        // Assert static nodes block present (both enodes)
        if !strings.Contains(s, "StaticNodes") {
            t.Errorf("%s missing StaticNodes", name)
        }
    }
}
```

- [ ] **Step 2: Implement GenerateToml**

Port `lib/adapters/stablenet.sh:124-196`:

1. Extract `metadata.nodes[:total]`, build enode static-nodes block.
2. For i in 1..total:
   - Compute per-node ports.
   - Determine miner section (validators only).
   - Build keystore dir path and ethstats URL.
   - Replace placeholders, write `config_node<i>.toml`.

~60 lines of Go.

- [ ] **Step 3: Run tests + commit**

```bash
cd network && go test ./internal/adapters/stablenet/... -v -count=1 -timeout=30s
```

```bash
git add network/internal/adapters/stablenet/
git commit -m "feat(adapters/stablenet): port GenerateToml to Go

Port of lib/adapters/stablenet.sh's per-node TOML generator. Builds
the shared static-nodes block from metadata, then for each node
composes a per-node config_node<N>.toml with sequential ports,
role-appropriate miner section, and keystore directory path.

Tests verify: correct file count, sequential HTTP ports, StaticNodes
block present in each output."
```

---

## Task 4 — Golden-file contract tests

**Files:**
- Create: `network/internal/adapters/stablenet/stablenet_contract_test.go`
- Create: `network/internal/adapters/testdata/golden/stablenet-genesis.json`
- Create: `network/internal/adapters/testdata/golden/stablenet-config-node1.toml`
- Create: `network/internal/adapters/testdata/golden/stablenet-config-node2.toml`
- Modify: `network/go.mod`, `network/go.sum` (add go-toml dep)

- [ ] **Step 1: Capture golden output from Go adapter**

Run `GenerateGenesis` + `GenerateToml` with the same fixtures as Task 2/3 tests. Copy their outputs to `testdata/golden/`. Regenerate script lives at `network/internal/adapters/stablenet/regen_golden_test.go`:

```go
//go:build regen_golden

package stablenet_test

// Run with: go test -tags regen_golden ./internal/adapters/stablenet/...
// Regenerates the golden files under testdata/golden/. Commit the result.
func TestRegenGolden(t *testing.T) { ... }
```

OR — simpler — use `-update` flag idiom:

```go
var updateGolden = flag.Bool("update", false, "update golden files")
```

Either pattern works. Pick one and document in the file.

- [ ] **Step 2: Write contract tests**

```go
// stablenet_contract_test.go
package stablenet_test

func TestContract_GenesisMatchesGolden(t *testing.T) {
    // Run GenerateGenesis
    // Read golden + generated
    // Deep-compare via reflect.DeepEqual(map[string]any, map[string]any)
    // On mismatch: pretty-print both for diff visibility
}

func TestContract_TomlMatchesGolden(t *testing.T) {
    // For each generated config_nodeN.toml:
    //   Parse both (generated + golden) via go-toml/v2 into map[string]any
    //   Deep-compare
}
```

- [ ] **Step 3: Add go-toml dep**

```bash
cd network && go get github.com/pelletier/go-toml/v2@latest
go mod tidy
```

- [ ] **Step 4: Verify goldens stored + tests pass**

```bash
cd network && go test ./internal/adapters/... -v -count=1 -timeout=30s
```

- [ ] **Step 5: Commit**

```bash
git add network/go.mod network/go.sum network/internal/adapters/testdata/golden/ network/internal/adapters/stablenet/stablenet_contract_test.go
git commit -m "test(adapters/stablenet): pin genesis + toml golden files

Golden-file contract tests lock the Go adapter output shape. Genesis
compares via JSON deep-equal; TOML parses both files via go-toml/v2
and compares the resulting map[string]any. An -update flag on the
test binary regenerates the goldens when the bash adapter or template
changes (regenerate, diff, commit).

Byte-level equivalence with the bash+Python output is an explicit
non-goal; JSON/TOML semantic equivalence is the contract."
```

---

## Task 5 — Final review + roadmap

- [ ] **Step 1: Full matrix**

```bash
cd network && go test ./... -count=1 -timeout=60s
cd .. && bash tests/unit/run.sh
```
Expected: all green.

- [ ] **Step 2: Update roadmap**

Edit `docs/VISION_AND_ROADMAP.md`:
- Mark `adapters/stablenet` Go port complete.
- Note wbft + wemix skeletons shipped; real impls deferred.

```
- [x] `adapters/stablenet` Go 포팅 (genesis + TOML) + `adapters/wbft` / `adapters/wemix` 스켈레톤 + `Adapter` 인터페이스 + `Load` 팩토리 — Sprint 3c 완료 (2026-04-24); wbft / wemix 실구현 + `network.init` wire handler 는 후속
```

- [ ] **Step 3: Commit**

```bash
git add docs/VISION_AND_ROADMAP.md
git commit -m "docs: mark Sprint 3c complete — adapter Go port"
```

- [ ] **Step 4: Report**

Commit range, test counts, coverage on the new package, any deferrals (wbft real impl, wemix real impl, chainbench-net network.init handler).
