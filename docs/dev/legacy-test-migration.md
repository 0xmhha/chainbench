# Legacy shell test-suite → DSL migration

> Source: `~/Work/github/packages/chainbench/tests` (~420 shell test scripts).
> Target: this project's v2 DSL cases (`tests/cases/`) run by `chainbench run`.
> This doc is the map both sides were parsed into and the plan for porting the rest.
> First slice ported and live-verified: the `fault` category (see §5).

## 1. Scope

| category | scripts | paradigm | chain / binary |
|---|---|---|---|
| basic | 7 | local | stablenet `gstable` |
| fault | 6 | local | stablenet `gstable` |
| remote | 4 | attach (external) | any (read-only) |
| stress | 2 | local | stablenet `gstable` |
| stablenet | 306 (regression 222 + post-v1.0.0-change 80) | remote/SSH | `gstable` + HF variants |
| wemix4 | 95 (TX 20, RPC 23, GOV 22, WBFT 12, NODE 7 …) | remote/SSH | `gwemix3` (poa) → `gwemix4` (wbft) |
| lib / unit | 16 / 24 | — | harness internals, NOT chain tests |

Two execution paradigms in the legacy suite:
- **local** (basic/fault/stress) — nodes are local processes the parent tool manages; config from a `profiles/*.yaml`; node started as a **private network** (`gstable init --datadir <dd> <genesis>` then `gstable --datadir … --networkid …`), never `--testnet`/`--mainnet`.
- **remote/SSH** (stablenet/wemix4) — nodes are pre-provisioned servers driven over `ssh`; binary is a symlink switched per test; genesis + `config.toml` uploaded and SHA-256 verified; network selected by `testnet|private|mainnet` positional arg (the **network/genesis**, not the binary).

Both map onto this project's compose model: `env` declares a private network (chain + binary + topology + genesis overlay); the DSL never uses `--testnet`/`--mainnet`.

## 2. Already ported

Most of the suite is already covered — do NOT re-port these; port the gap.
- `tests/specs/` (122 v1 specs): accounts, api, consensus, gas-policy, hardfork, network, system-contracts.
- `tests/cases/` (v2): stablenet/wbft/wemix chain-up (+15-node), handoff.
- Go e2e (`//go:build e2e`): the consensus/fault/upgrade tiers (`tests/e2e/`, `cmd/chainbench/*_e2e_test.go`).
- Trackers: `wemix4-port-tracker.md` (wemix4 ~62 ported / ~8 deferred), `stablenet-post-v1.0.0-change-test-catalog.md`.

The **gap** is what those trackers mark deferred plus categories with no DSL yet — the `fault` category had none, which is why it is the first slice here.

## 3. Vocabulary map (legacy lib → DSL)

The legacy `lib/` functions are the action+assertion vocabulary; each maps to a DSL verb.

| legacy (lib) | DSL step |
|---|---|
| `send_tx` / `cb_send_tx` / `send_raw_tx` | `do: sendTx` (`from`/`to`/`value`/`key`; `expect: receipt\|revert\|reject`) |
| `cb_deploy_contract` | `do: deployContract` |
| `newAccount` (local sign) | `do: newAccount` (`save` addr, `saveKey` privkey) |
| `wait_for_block` | `do: waitBlock` (`target`, `timeout`, `pollInterval`, `on`) |
| `wait_for_condition` | `do: waitFor` |
| `chainbench.sh node stop/start/restart` | `do: stopNode` / `startNode` / `restartNode` (`on`) |
| `admin_remove_peer` cross-group loop | `do: partition` (`groups: [[…],[…]]`) |
| `admin_add_peer` restore loop | `do: healPartition` |
| `block_number` | `expect: blockNumber` |
| `check_sync` (all nodes agree) | `expect: sameBlockHash`; production → `expect: blockAdvance` |
| `peer_count` | `expect: peerCount` |
| `get_balance` / `get_nonce` / `get_code` | `expect: balanceAt` / `nonceAt` / `codeAt` |
| `eth_call` | `expect: call` |
| `get_receipt` / status | `expect: txStatus` / `txMined` / `receiptLog` |
| `rpc <t> <method>` (istanbul_*, admin_*, txpool_*) | `expect: rpc` (alias of `rpcCall`) with `method`/`params`/`select` |
| `get_base_fee` / `get_gas_price` | `expect: baseFee` / `gasPrice` |
| `assert_eq/gt/ge/true/contains` | `expect … compare Equal/Greater/GreaterOrEqual/…`, `is` |

Structure map:
- `# ---chainbench-meta---` (id/name/category/tags/preconditions) → the case `id` + `requires` (capabilities) + `applicableChains`.
- `ensure_chain_environment private` → the `env` (private network, genesis, topology).
- `NN-test-*.sh` body (pure actions+asserts) → the case `steps` (do/expect sequence).
- No per-test teardown in shared-fixture tests; a compose run tears its own network down.

## 4. Node addressing

Legacy uses numeric-local (`"1"` via `pids.json`) or alias-remote (`@stablenet-bp1`). The DSL uses one selector on each step's `on`: `node<N>` (index) or role-ordinal (`bp1`, `en1`), resolved by the environment. Fault/lifecycle verbs need node control, so they run in **compose** mode (`chainbench run --workspace-dir`), not attach.

## 5. First ported slice — `fault` (live-verified)

`tests/cases/fault/` + env `tests/cases/env/fault-stablenet.env.json` (stablenet, 4 bp, preset keys). Each ran green against a real `gstable` 4-node network:

| legacy | DSL case | procedure |
|---|---|---|
| `fault/node-crash.sh` | `node-crash.json` | waitBlock 3 → sameBlockHash → stopNode node3 → blockAdvance node1 (3/4 keeps producing) → restartNode node3 → waitBlock node3 → sameBlockHash |
| `fault/network-partition.sh` | `network-partition.json` | partition {node1,node2}\|{node3,node4} → peerCount ≤2 each side → healPartition → node3 catches up → sameBlockHash |
| `fault/node-recover.sh` | `node-recover.json` | stopNode node3 → node1 runs ahead → restartNode node3 → node3 catches up → sameBlockHash |
| `fault/txpool-leader-change.sh` | `txpool-leader-change.json` | send a tx → stopNode node1 (leader) → blockAdvance node2 → txpool_status.pending == 0x0 → startNode node1 |

Deferred (need new DSL machinery, tracked below):
- `fault/two-down.sh` — asserts consensus **halts** with 2/4 down. Needs a `blockHalt`-style assertion (head does NOT advance within a window); `blockAdvance`'s negation is not expressible today.
- `fault/p2p-topology.sh` — hub-spoke topology. `partition` models disjoint groups, not a hub with spokes.

## 6. Phased plan for the rest

Port category-by-category, each slice live-verified against the from-source binary, gap-checked against §2 first:

1. **basic (7)** — smoke: rpc-health, peers, sync, consensus, tx-send, txpool-propagation. Trivial `expect` reads + one `sendTx`.
2. **stress (2)** — tx-flood, block-time. Needs a load primitive or a bounded `sendTx` loop.
3. **stablenet regression (222)** — the functional matrix (api 46, system-contracts 48, ethereum 64, wbft 24, blacklist-authorized 18, anzeon 14, fee-delegation 8). Cross-ref each against `tests/specs/` (most api/system-contracts/consensus already exist); port the remainder.
4. **stablenet post-v1.0.0-change (80)** — hardfork/boho behaviors; cross-ref the catalog doc.
5. **wemix4 (95)** — finish the ~8 deferred in the tracker (e.g. RPC-008 brioche reward config).
6. **remote (4)** — attach-mode reads against an external chain (`chainbench run --rpc`).

## 7. New DSL machinery the migration needs

- `blockHalt` assertion — head stays within a window (for fault halt, two-down).
- hub-spoke / star peering for `partition` (for p2p-topology).
- `sendTx`-loop / load primitive (for stress tx-flood).
- Whatever the deferred wemix4 items need (see `wemix4-port-tracker.md`).
