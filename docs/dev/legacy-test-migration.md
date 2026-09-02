# Legacy shell test-suite → DSL migration

> Source: `~/Work/github/packages/chainbench/tests` (~420 shell test scripts).
> Target: this project's v2 DSL cases (`tests/cases/`) run by `chainbench run`.
> This doc is the map both sides were parsed into and the plan for porting the rest.
> All local categories (basic, fault, stress, anzeon) are ported and live-verified
> (§5–§5d); the remote/stablenet/wemix4 remainder is covered by specs + Go e2e, with
> a small set of wemix4 items still blocked on chain-side prerequisites (§7).

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
| `fault/two-down.sh` | `two-down.json` | waitBlock 3 → sameBlockHash → stopNode node3 + node4 → `blockHalt` node1 (advance ≤ 1 over 12s: 2/4 cannot seal) → startNode node3 (3/4 quorum) → blockAdvance node1 → startNode node4 |
| `fault/p2p-topology.sh` | `p2p-topology.json` | hub-spoke via `partition [[node2],[node3],[node4]]` (each spoke its own group severs spoke↔spoke; node1, ungrouped, keeps all links) → peerCount node1 ≥ 2 → blockAdvance (consensus relays through hub) → tx through hub mined → healPartition → re-converge |

The last two needed the `blockHalt` assertion (§7) and the insight that hub-spoke
is just `partition` with singleton spoke groups — no new peering primitive. All
four fault cases now run green.

## 5b. Second ported slice — `basic` (live-verified)

`tests/cases/basic/` + env `basic-stablenet` (stablenet, 4 bp + 1 en). All seven
ran green against a real `gstable` 5-node network in one suite:

| legacy | DSL case | check |
|---|---|---|
| `basic/rpc-health.sh` | `rpc-health.json` | every node's `blockNumber` ≥ 0 |
| `basic/peers.sh` | `peers.json` | every node's `peerCount` ≥ 1 |
| `basic/sync.sh` | `sync.json` | `sameBlockHash` — nodes agree on the head |
| `basic/consensus.sh` | `consensus.json` | waitBlock 20 → `blockAdvance` → `sameBlockHash` |
| `basic/wbft-consensus.sh` | `wbft-consensus.json` | `istanbul_getValidators` ≥ 4, `istanbul_getWbftExtraInfo("@latest").prevCommittedSeal.sealers` ≥ 3 |
| `basic/tx-send.sh` | `tx-send.json` | send a value tx → recipient `balanceAt` > 0 |
| `basic/txpool-propagation.sh` | `txpool-propagation.json` | tx sent to node1 is mined and visible on node2 |

## 5c. Third ported slice — `anzeon` dynamic base fee (live-verified)

`tests/cases/anzeon/` + env `anzeon-stablenet` (stablenet, 4 bp, preset keys).
These were the one genuine regression gap (§6 item 2): the base fee moves ±2% a
block by block gas usage, which no static spec can drive. The new `load` action
(§7) deploys a gas-burner sized to a percent of the block gas limit, one burn per
block, sustained across blocks — so the base fee climbs (or, left idle, falls).
All three ran green against a real `gstable` 4-node network:

| legacy | DSL case | procedure |
|---|---|---|
| `anzeon/03-basefee-increase.sh` | `basefee-increase.json` | read baseFee → `load` fillPercent 25 blocks 6 (usage > 20%) → baseFee rose above the baseline (`Greater $bf0`) |
| `anzeon/04-basefee-stable.sh` | `basefee-stable.json` | read baseFee → `load` fillPercent 10 blocks 6 (usage 6-20%) → baseFee unchanged (`Equal $bf0`) — the control: moderate load must NOT raise it |
| `anzeon/05-basefee-decrease.sh` | `basefee-decrease.json` | `load` fillPercent 25 blocks 6 → read peak → idle empty blocks (waitBlock) → baseFee fell below the peak (`Less $peak`) |

The three together characterise the Anzeon band: > 20% raises, 6-20% is neutral,
empty blocks lower. `read baseFee` (the `baseFee` source doubles as a reader)
captures the before value; the expect compares against it (`is: "$bf0"`). This is
the first case to reference a saved value from an *expect* statement, which
exposed a v2 gap in offline `validate` (`Unresolved` walked v1's Steps then
Assertions and never saw a step's `save`); it now walks the unified `Sequence`.

The base-fee min/max clamps (`anzeon/06`, `07`) stay covered by the `gas-policy`
specs; the gas-tip cases (`anzeon/01`, `02`) by `regular-account-gastip-forced` /
`authorized-account-gastip-free`. Only the three dynamic-adjustment cases needed
porting.

## 5d. Fourth ported slice — `stress` (live-verified)

`tests/cases/stress/` + env `stress-stablenet` (stablenet, 4 bp, preset keys).
Both ran green against a real `gstable` 4-node network:

| legacy | DSL case | check |
|---|---|---|
| `stress/block-time.sh` | `block-time.json` | waitBlock 20 → `blockInterval` blocks 15 maxSeconds 60 (average interval over the last 15 blocks is bounded and positive) |
| `stress/tx-flood.sh` | `tx-flood.json` | `load` fillPercent 30 blocks 15 (15 burn txs, each confirmed — load errors if one fails to mine) → `blockAdvance` (chain kept sealing under sustained ~1/3-block load) |

`blockInterval` samples block timestamps and checks the mean gap; `tx-flood` reuses
the `load` action for throughput (every burn is confirmed, so 15 blocks = 15 mined
txs under load) rather than the legacy's 100 independent value txs.

## 6. Phased plan for the rest

Port category-by-category, each slice live-verified against the from-source binary, gap-checked against §2 first:

1. **basic (7)** — ✅ done (§5b). **stress (2)** — ✅ done (§5d): `stress/block-time` uses the new `blockInterval` assertion (avg interval over a window), `stress/tx-flood` uses the `load` action (sustained per-block burn txs, all confirmed). **fault (6)** — ✅ done (§5): all six, including two-down and p2p-topology. **remote (4)** — the same reads as `basic` (chainId, blockNumber, peerCount, balance, tx) but attach-mode against an external chain; the DSL already runs any case attach-mode via `chainbench run --chain <c> --rpc <url> <case>`, so a `basic` case IS the remote test pointed at an external node — no separate cases needed, only external-chain env vars at run time.
2. **stablenet regression (~88 tests: api 23, ethereum 32, system-contracts 24, wbft 12, blacklist-authorized 9, anzeon 7, fee-delegation 4)** — gap-checked: essentially **already covered**, since `tests/specs/` was derived from this suite. Confirmed by cross-referencing each behavior:
   - **api (23)** — every RPC (getBlock*, getTx*, getCode, getTxCount, gasPrice, maxPriorityFee, feeHistory, estimateGas, all `istanbul_*`, txpool_*, admin_peers, signRawFeeDelegate, totalSupply, allowance) has a spec or a DSL source; `net_peerCount` is now covered by `basic/peers`.
   - **ethereum (32)** — the tx-type matrix, contract deploy/call, eth_call/revert, out-of-gas, get-logs, chain-id, ws-subscribe are in `accounts`/`api`/`gas-policy` specs; **node-restart** is `fault/node-recover`; **full/snap sync, downloader, block-fetcher** are Go e2e (`TestE2E_StablenetSyncGap`, `TestE2E_WbftSnapSync`).
   - **wbft (12)** — block-period, extra-seal, epoch, add/remove-validator, gastip-header-sync, get-validators, quorum-deficient, prev-committed/prepared-seal are in `consensus` specs; **round-change / post-round-change** are Go e2e (`TestE2E_WbftViewChange`, `TestE2E_WbftRoundRobinProposer`).
   - **system-contracts (24) / blacklist-authorized (9) / fee-delegation (4)** — covered by the `system-contracts` (45) and `accounts` specs (native transfer, approve/transferFrom, mint/burn proposals, gov lifecycle, blacklist/authorize + events, fee-delegation valid + tampered).
   - **anzeon (7)** — gastip-forced/free, min/max baseFee are in `gas-policy` specs. The dynamic `basefee-increase` / `basefee-stable` / `basefee-decrease` cases (±2% by block gas usage) are now **ported and live-verified** (§5c) via the new `load` action — the last genuine regression gap is closed.
3. **stablenet post-v1.0.0-change (80)** — hardfork/boho behaviors; the catalog (`stablenet-post-v1.0.0-change-test-catalog.md`) and the `tests/repro`→Go-e2e doc record these as ported to Go e2e (`TestE2E_StablenetHardforkSwap`, delayed-fork, account-extra) or blocked on chain-side prerequisites (`repro-migration-remaining.md` §1–5). No DSL-tractable gap without those prerequisites.
4. **wemix4 (95)** — `wemix4-port-tracker.md`: ~62 ported (DSL + Go e2e), ~8 deferred needing new machinery (e.g. RPC-008 brioche-reward genesis config). Finish those against the tracker.

**Net for items 2–4:** the local categories (basic, fault, stress, anzeon) are now
**fully ported** (§5–§5d); the rest is covered by `tests/specs` + Go e2e. The only
remaining DSL-tractable work is the ~8 tracked wemix4 items, most blocked on
chain-side prerequisites (see §7 and `wemix4-port-tracker.md`).

## 7. New DSL machinery the migration needs

All four primitives the earlier slices called for are now built and live-verified:

- ~~a load primitive~~ — **done**: the `load` action (`internal/testhelper/load.go`) deploys a gas-burner sized to `fillPercent`% of the block gas limit, one burn per block for `blocks` blocks. Unblocked `anzeon` (§5c) and `stress/tx-flood` (§5d).
- ~~`blockHalt` assertion~~ — **done** (`internal/testhelper/blockprobe.go`): head advances ≤ `maxAdvance` over a `within` window — the negation of `blockAdvance`. Unblocked `fault/two-down` (§5).
- ~~hub-spoke peering~~ — **done, no new code**: hub-spoke is `partition` with singleton spoke groups (`[[node2],[node3],[node4]]`); the ungrouped hub keeps all links. Unblocked `fault/p2p-topology` (§5).
- ~~a block-interval assertion~~ — **done** (`blockprobe.go`): `blockInterval` samples the last `blocks` timestamps and bounds the mean gap. Unblocked `stress/block-time` (§5d).

The **wemix4** suite is now fully ported/covered (see `wemix4-port-tracker.md`):
RPC-008 `wemix_getBriocheBlockReward` was the last item, ported as
`tests/cases/wemix/brioche-block-reward.json` by injecting a `brioche`
halving-config object through `genesis.overlay` (no code change — the overlay
deep-merges into the poa genesis `config`). GOV-023 and WBFT-012/013 were already
Go e2e (the earlier "remaining" prose was stale).

The `tests/repro`→Go-e2e remainder (`repro-migration-remaining.md`) is down to a
single blocked script:
- `layer2-attach` — needs a live external L2 RPC endpoint to verify against;
  chain-agnostic attach-mode cases already run against any `--rpc <url>`.

Now resolved and ported to DSL cases here:
- `stablenet-delayed-fork` (`tests/cases/stablenet/delayed-fork.json`) — the boho
  effects (GovMinter-v2 code swap, P-256 at 0x100, prealloc preserved) are real;
  the old failures were genesis wiring (the genesis needs the full `boho` object,
  not just `bohoBlock`). The case reads across the fork with `rpcCall` at explicit
  block tags.
- `stablenet-account-extra` (`tests/cases/stablenet/account-extra.json`, plus the
  overlay fixture fix).
- `wemix-chain` scenario 1 (`tests/cases/wemix/tx-and-contract.json`).
- `stablenet-basefee-dynamics` (superseded by the anzeon cases, §5c).
