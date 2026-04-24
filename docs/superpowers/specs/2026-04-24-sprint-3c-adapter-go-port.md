# Sprint 3c — Adapter Go Port Design Spec

> 2026-04-24 · Sub-sprint of VISION §6 Sprint 3
> Scope: port the `stablenet` bash adapter to Go. Add `wbft` + `wemix`
> skeletons. Bash adapters stay in place — Go adapter is additive, available
> to future `chainbench-net` subcommands.

## 1. Goal

Expose the adapter layer to Go callers so future `chainbench-net` commands
(e.g. `network.init`, `network.bootstrap`) can generate genesis / per-node
TOML config without shelling out to Python + bash. Establish a clean
`Adapter` interface that wbft + wemix can fill in incrementally.

## 2. Non-Goals

- **Replacing the bash `chainbench init` flow** — bash stays as the current
  entry point. Go adapter is additive.
- **Full wbft / wemix implementation** — skeletons only. Genesis + TOML
  generation return `ErrNotImplemented` until a real chain variant needs them.
- **New `chainbench-net` subcommand** — 3c ships the library layer; the
  init/bootstrap wire handlers come in a follow-up.
- **Template redesign** — keep the `__PLACEHOLDER__` convention used by the
  current `templates/genesis.template.json` and `templates/node.template.toml`.
- **Byte-identical output** with bash adapters — JSON deep-compare is the
  equivalence contract; key ordering and whitespace may differ.

## 3. Package Layout

```
network/internal/adapters/
├── adapter.go             # Adapter interface, Load(chainType) factory,
│                          # shared error sentinels
├── adapter_test.go        # factory + interface contract tests
├── stablenet/
│   ├── stablenet.go       # adapter impl
│   ├── stablenet_test.go  # unit tests
│   └── stablenet_contract_test.go  # JSON-equivalence vs bash output
├── wbft/
│   ├── wbft.go            # skeleton — flags + namespace; generate stubs
│   └── wbft_test.go
└── wemix/
    ├── wemix.go
    └── wemix_test.go
```

## 4. Interface

```go
package adapters

import "context"

type Role string

const (
    RoleValidator Role = "validator"
    RoleEndpoint  Role = "endpoint"
)

// Profile captures the JSON-decoded profile input used by GenerateGenesis.
// Exact fields are chain-type-dependent; Profile is a discriminated shape
// that each adapter interprets according to its own schema.
type Profile map[string]any

// Metadata captures the JSON-decoded node metadata (addresses, public keys,
// BLS keys, etc.).
type Metadata map[string]any

// GenesisInput bundles the inputs GenerateGenesis needs. TemplateBytes is the
// raw template file contents (caller reads from disk); OutputPath is where the
// adapter writes the generated genesis.json.
type GenesisInput struct {
    Profile       Profile
    Metadata      Metadata
    TemplateBytes []byte
    OutputPath    string
    NumValidators int
    BaseP2P       int
}

// TomlInput bundles inputs for GenerateToml. WriteDir is the directory where
// per-node config_node<N>.toml files land.
type TomlInput struct {
    Metadata       Metadata
    TemplateBytes  []byte
    WriteDir       string
    TotalNodes     int
    NumValidators  int
    BaseP2P        int
    BaseHTTP       int
    BaseWS         int
    BaseAuth       int
    BaseMetrics    int
}

// Adapter is the chain-type-specific strategy for initializing a local
// network. Each adapter knows how to fill its chain's template placeholders.
type Adapter interface {
    // GenerateGenesis writes a genesis.json to input.OutputPath. Returns
    // ErrNotImplemented for adapters (wbft, wemix) still under development.
    GenerateGenesis(ctx context.Context, input GenesisInput) error

    // GenerateToml writes per-node config_node<N>.toml files into
    // input.WriteDir. Returns ErrNotImplemented for skeleton adapters.
    GenerateToml(ctx context.Context, input TomlInput) error

    // ExtraStartFlags returns chain-specific flags for the given role.
    // Flags are space-separated in the returned string (bash-compatible).
    ExtraStartFlags(role Role) string

    // ConsensusRpcNamespace returns the RPC namespace this chain exposes for
    // consensus queries (e.g. "istanbul" for stablenet/wbft, "wemix" for wemix).
    ConsensusRpcNamespace() string
}

// Load returns the Adapter implementation for the given chain type, or
// ErrUnknownChainType if the type has no registered adapter.
func Load(chainType string) (Adapter, error)
```

### 4.1 Error sentinels

```go
var (
    ErrUnknownChainType = errors.New("adapters: unknown chain type")
    ErrNotImplemented   = errors.New("adapters: operation not implemented for this chain type")
)
```

## 5. stablenet Implementation

Port the Python logic from `lib/adapters/stablenet.sh` directly into Go:

### 5.1 GenerateGenesis

- Decode profile for `chain.chain_id`, `genesis.overrides.wbft.*`,
  `genesis.overrides.systemContracts.*`, `genesis.overrides.alloc`.
- Build `validators`, `blsPublicKeys` from `metadata.nodes[:num_validators]`.
- Compose system contracts (govValidator, govMinter, govMasterMinter, govCouncil).
- Build alloc map (default balance for each node + user-specified overrides).
- Ensure CREATE2 proxy contract at `0x4e59b44847b379578588920cA78FbF26c0B4956C`
  is injected (matches bash behavior).
- Replace template placeholders: `__CHAIN_ID__`, `__REQUEST_TIMEOUT_SECONDS__`,
  `__BLOCK_PERIOD_SECONDS__`, `__EPOCH_LENGTH__`, `__PROPOSER_POLICY__`,
  `__MAX_REQUEST_TIMEOUT_SECONDS__`, `__VALIDATORS_JSON__`,
  `__BLS_PUBLIC_KEYS_JSON__`, `__SYSTEM_CONTRACTS_JSON__`, `__ALLOC_JSON__`,
  `__EXTRA_DATA__`.
- Validate the final document parses as JSON.
- Write to `input.OutputPath`.

### 5.2 GenerateToml

- Decode metadata nodes, build enode strings for static-nodes.
- For each node 1..TotalNodes:
  - Compute per-node ports (P2P, HTTP, WS, Auth, Metrics).
  - Compose miner section (validators only).
  - Replace template placeholders, write to `config_node<N>.toml`.

### 5.3 ExtraStartFlags

- Common: `--allow-insecure-unlock --rpc.enabledeprecatedpersonal --rpc.allow-unprotected-txs`
- Validator adds: `--mine`

### 5.4 ConsensusRpcNamespace

Returns `"istanbul"`.

## 6. wbft / wemix Skeletons

### wbft
- `ExtraStartFlags(role)` → `"--allow-insecure-unlock"` (role-agnostic; matches
  bash stub)
- `ConsensusRpcNamespace()` → `"istanbul"` (bash stub returns this)
- `GenerateGenesis` / `GenerateToml` → `ErrNotImplemented`

### wemix
- `ExtraStartFlags(role)` → `"--allow-insecure-unlock"`
- `ConsensusRpcNamespace()` → `"wemix"`
- `GenerateGenesis` / `GenerateToml` → `ErrNotImplemented`

## 7. Equivalence Contract (stablenet)

`stablenet_contract_test.go` runs both implementations against the same fixture
inputs and asserts:

1. **Genesis**: generated JSON files parse to semantically equal `map[string]any`
   (deep compare, order-independent).
2. **TOML**: generated `config_node<N>.toml` files parse to semantically equal
   structured map (key-value, one level deep — TOML shape is flat).

Fixtures:
- Profile: a minimal in-memory profile with known chain_id, validator count.
- Metadata: N nodes with placeholder addresses / public keys.
- Template: existing `templates/genesis.template.json` and `templates/node.template.toml`.

Contract test runs the bash adapter via `bash -c 'source lib/adapters/stablenet.sh; adapter_generate_genesis ...'` (requires python3; fine on dev / CI).

If the bash comparison is too brittle for CI, fall back to a pinned golden
file: commit `testdata/golden/stablenet-genesis.json` and assert the Go output
matches it. Simpler, catches regressions against a known-good state.

**Decision**: ship the golden-file variant. Bash comparison runs manually by
maintainer when regenerating goldens. CI stays deterministic.

## 8. Testing Strategy

**Unit (each adapter's _test.go):**
- `Load("stablenet")` returns stablenet impl; unknown type returns
  `ErrUnknownChainType`.
- `ExtraStartFlags(Validator)` / `(Endpoint)` return expected flag strings.
- `ConsensusRpcNamespace()` returns expected string.
- `GenerateGenesis` / `GenerateToml` return `ErrNotImplemented` for skeletons.

**Unit (stablenet):**
- `GenerateGenesis` with fixture: output parses as valid JSON, has expected
  `config.chainId`, alloc contains expected addresses, extraData is propagated.
- `GenerateToml` with fixture: correct number of files written, each parses
  as TOML, per-node ports are sequential.

**Contract (stablenet):**
- Run Go `GenerateGenesis` on fixture → compare to pinned golden file.
- Run Go `GenerateToml` on fixture → compare each output file to pinned golden.

## 9. Deferred / Out-of-Scope

**From 3c:**
- `chainbench-net network.init` / `network.bootstrap` wire handlers.
- wbft real implementation (genesis + TOML generation).
- wemix real implementation.
- Replacing bash `chainbench init` with the Go path.
- Profile / metadata schema validation at the adapter boundary.

**Still deferred from earlier sprints:**
- 2b.3 M3, 2c M3, 2c M4, 3a Minor, 3b Minor (TOCTOU), 3b.2a I-2, 3b.2b Minor
- 3b.2c minors (tail_log M4 absorption, BadAddress variants, types.Auth typed refactor)
