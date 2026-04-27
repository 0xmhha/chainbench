# Adapter Contract

> **Status:** active — bash adapter inventory (LocalDriver backing) + Go adapter
> port progress (Sprint 3c, 2026-04-24). The bash surface in §1/§2 still
> backs `lib/cmd_*.sh`; the Go track in §4 is the long-term home of the
> Network Abstraction adapter axis (§5.17 of VISION_AND_ROADMAP).
> Regenerate the bash tables below with:
>
>     scripts/inventory/list-adapter-functions.sh

## 1. Current adapter surface (stablenet — authoritative)

These are the four functions every chain adapter must provide today. Each is
called from `lib/cmd_*.sh` through the dispatcher in `lib/chain_adapter.sh`.

| Function | Arity | Purpose |
|---|---|---|
| `adapter_generate_genesis <profile_json> <template> <out> <meta> <num_validators> <base_p2p>` | 6 | Produce `genesis.json` and sidecar metadata for the chosen chain |
| `adapter_generate_toml <profile_json> <node_idx> <out>` | 3 | Produce the per-node TOML config the binary reads at startup |
| `adapter_extra_start_flags` | 0 | Chain-specific CLI flags appended to the node launch command |
| `adapter_consensus_rpc_namespace` | 0 | Name of the RPC namespace exposing validator/consensus methods (e.g. `istanbul`) |

## 2. Per-chain implementation status

```
CHAIN      FUNCTION                                 LOCATION                                           STATUS
stablenet  adapter_generate_genesis                 lib/adapters/stablenet.sh:14                       real
stablenet  adapter_generate_toml                    lib/adapters/stablenet.sh:124                      real
stablenet  adapter_extra_start_flags                lib/adapters/stablenet.sh:203                      real
stablenet  adapter_consensus_rpc_namespace          lib/adapters/stablenet.sh:216                      real
wbft       adapter_generate_genesis                 lib/adapters/wbft.sh:13                            stub
wbft       adapter_generate_toml                    lib/adapters/wbft.sh:14                            stub
wbft       adapter_extra_start_flags                lib/adapters/wbft.sh:15                            real
wbft       adapter_consensus_rpc_namespace          lib/adapters/wbft.sh:16                            real
wemix      adapter_generate_genesis                 lib/adapters/wemix.sh:14                           stub
wemix      adapter_generate_toml                    lib/adapters/wemix.sh:15                           stub
wemix      adapter_extra_start_flags                lib/adapters/wemix.sh:16                           real
wemix      adapter_consensus_rpc_namespace          lib/adapters/wemix.sh:17                           real
```

## 3. Gaps — functions the HAL contract will need but adapters do not expose today

Derived from §5.15 event catalog and §5.17 HAL interface.

| Proposed function | Why needed | First consumer |
|---|---|---|
| `adapter_binary_name` | Remove `gstable` hardcoding from `cmd_start/stop/node` (see `docs/HARDCODING_AUDIT.md`) | LocalDriver subprocess launch |
| `adapter_datadir_layout <node_idx>` | Compute per-node data dir path without shell assumptions | LocalDriver + log tail |
| `adapter_log_file_path <node_idx>` | Locate the appender-rotated file driver-side | `node.tail_log` |
| `adapter_consensus_validator_rpc_method` | Uniform access to `istanbul_getValidators` / `clique_getSigners` / `wemix_*` | `network.capabilities`, consensus tests |
| `adapter_supported_tx_types` | Gate chain-specific tx types (0x16, 0x04) for Layer 2 tx_builder | `tx.send` composite |
| `adapter_probe_markers` | RPC method names whose presence identifies the chain type | `network.probe` (§5.17 Q2) |

## 4. Go adapter track (Sprint 3c, 2026-04-24)

The Network Abstraction adapter axis lives at `network/internal/adapters/`.
The interface and factory are landed; per-chain implementations are partial.

| Chain | `GenerateGenesis` | `GenerateToml` | Notes |
|---|---|---|---|
| stablenet | real (Go port — golden-pinned) | real (Go port — golden-pinned) | Sprint 3c |
| wbft | skeleton (returns `ErrNotImplemented`) | skeleton | real impl pending wbft consumer |
| wemix | skeleton | skeleton | real impl pending wemix consumer |

Interface (see `network/internal/adapters/spec/`):

- `Adapter.GenerateGenesis(profile, template, out, meta, numValidators, baseP2P) error`
- `Adapter.GenerateToml(profile, nodeIdx, out) error`
- `Load(chainType string) (Adapter, error)` — factory that maps chain id → impl

The bash adapter (§1) still backs `LocalDriver` for `init/start/stop/node`
flows; the Go adapter is invoked from new wire handlers as the network
binary takes over those flows (§5.12 M4). Both tracks coexist until M4
fully lands.

## 5. Migration guidance

Each remaining gap (§3) becomes a method on the Go `Adapter` interface plus
(if it crosses the Go/bash boundary) a flag in `network/schema/network.json`.
See §5.12 M0–M4 of the roadmap for the sequencing.
