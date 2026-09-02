# tests/repro → Go e2e migration — remaining work

> Status as of 2026-07-29. The live-verification tier is being ported from
> `tests/repro/*.sh` (bash+python) to Go gated e2e tests under `tests/e2e/`
> (`//go:build e2e`) + `cmd/chainbench/upgrade_run_e2e_test.go`. See
> `tests/e2e/README.md` for how the ported tests run.

## Done (8 scenarios, all live-verified against from-source binaries)

| Scenario | Test | Binary |
|---|---|---|
| stablenet chain (block/tx/contract) | `TestE2E_StablenetChain` | gstable |
| stablenet binary-swap hardfork | `TestE2E_StablenetHardforkSwap` | gstable (+POST_FORK_BIN) |
| WBFT consensus lifecycle (b-08/09/10, a1-04) | `TestE2E_StablenetConsensusLifecycle` | gstable |
| near-head block propagation (a1-07) | `TestE2E_StablenetBlockPropagation` | gstable |
| endpoint re-sync (a1-02/03/06) | `TestE2E_StablenetSyncGap` | gstable |
| proposal expiry → Expired (f3-06) | `TestE2E_StablenetProposalExpiry` | gstable (overlay) |
| fresh wbft chain | `TestE2E_WbftChain` | gwbft (go-wbft's `gwemix`) |
| wemix→wbft handoff + post-fork state/tx/contract | `TestUpgradeRunE2E` (cmd/chainbench) | gwemix + gwbft (embedded etcd) |

Binaries build from `/Users/…/Work/github/chain/{go-stablenet,go-wbft,go-wemix}`
(`make gstable` / `make gwemix` / `make gwemix USE_ROCKSDB=NO`). **gwemix embeds
etcd** — no external etcd needed. Funded key: `keys/preset` node1 nodekey (public
test fixture) is genesis-funded. See memory `live-verification-setup`.

## Remaining — nothing to port

Of the original 5, four are ported to DSL cases (#1 delayed-fork, #2
account-extra, #3 basefee-dynamics, #4 wemix-chain). The fifth, `layer2-attach`
(#5), is NOT a porting target: the reference suite has no L2 scripts (only a
design note), chainbench does not support L2 as a chain family, and the "L2 E2E"
cases are the existing chain-agnostic RPC cases run attach-mode against any
endpoint. So the legacy→DSL migration is effectively complete. The five sections
below are kept for the root-cause record.

### 1. `stablenet-delayed-fork.sh` — RESOLVED (root cause was genesis wiring, now ported)
- The boho effects DO exist in go-stablenet: the GovMinter-v2 code swap
  (`consensus/wbft/engine/engine.go` `processFinalize` → `SetCode` at the boho
  block), the P-256 precompile at 0x100 (`core/vm/contracts.go`
  `PrecompiledContractsBoho`, gated on `rules.IsBoho`), and prealloc preservation
  (only code is swapped, no state cleared).
- Root cause of the old failures: setting `bohoBlock=N` alone is not enough. The
  GovMinter swap is gated by `params/config.go:1114` on `c.Boho != nil &&
  c.Boho.SystemContracts != nil`, so the genesis must ALSO carry the `boho` object
  (`boho.systemContracts.govMinter` version `v2`). With only `bohoBlock` set,
  `CollectUpgrades` yields no upgrade and the code never changes.
- **Ported** as `tests/cases/stablenet/delayed-fork.json` (env
  `delayed-fork-stablenet`): the overlay sets `bohoBlock: 3` AND the `boho`
  govMinter-v2 object. The case reads across the fork with `rpcCall` at explicit
  block tags (`eth_getCode`/`eth_call`/`eth_getBalance` at `0x1` vs `latest`) and
  asserts all three effects — GovMinter code changes v1→v2, P-256 at 0x100 is
  inactive (`0x`) pre-boho and returns `0x…01` post-boho (a real RIP-7212 vector),
  and a prealloc balance is unchanged. Live-verified.

### 2. `stablenet-account-extra.sh` — RESOLVED (overlay fixed + ported to a DSL case)
- Root cause confirmed: the overlay's broken half was the
  `config.anzeon.systemContracts.govCouncil.params.{authorizedAddresses,blacklistedAddresses}`
  ARRAYS, which collide with the base template's flat string-map `govCouncil.params`
  (`cannot unmarshal array into a string field`). The `alloc.*.extra` half was
  already correct.
- go-stablenet represents the Extra bits as a top-level `"extra"` field on each
  alloc entry (`core/types/account.go` `Account.Extra uint64`, JSON
  `math.HexOrDecimal64`): bit 62 `0x4000000000000000` = authorized, bit 63
  `0x8000000000000000` = blacklisted. `statedb.SetExtra` applies them at
  genesis-init and `ValidateExtra` rejects bits outside that mask.
- **Fixed:** `internal/chains/stablenet/overlays/account-extra.json` now carries
  only the alloc `extra` bits (the invalid `config.anzeon` block is removed), so
  genesis init succeeds and the AccountManager (0x…B00003) reads the seeded
  status. **Ported** as `tests/cases/stablenet/account-extra.json` — a self-contained
  DSL case that seeds the three accounts via `genesis.overlay` and asserts
  `isAuthorized`/`isBlacklisted` (incl. the dual account) return 1. Live-verified.

### 3. `stablenet-basefee-dynamics.sh` — SUPERSEDED by the DSL anzeon cases
- ~~Needs a burst of many txs into ONE block to move baseFee.~~ Resolved a
  different way: the DSL `load` action (`internal/testhelper/load.go`) deploys a
  single gas-burner sized to a percent of the block gas limit, so one tx fills the
  block — no explicit-nonce burst needed. The base-fee dynamics are now covered by
  `tests/cases/anzeon/basefee-{increase,stable,decrease}.json`, live-verified
  against a real gstable network (see `legacy-test-migration.md` §5c).
- No Go-e2e port of this script is needed; it can be retired from `run-all.sh`.

### 4. `wemix-chain.sh` (scenario 1, pure wemix) — RESOLVED via the DSL run path
- ~~`chainbench setup` does not bootstrap standalone wemix.~~ The DSL `chainbench
  run` path DOES: it drives the poa governance-etcd bootstrap for a non-handoff
  wemix network (confirmed by `tests/cases/wemix/chain-up.json`,
  `brioche-block-reward.json`, and now `tx-and-contract.json`, all live-verified
  against a real gwemix 4-validator network).
- Scenario 1's block/tx/contract verification is ported as
  `tests/cases/wemix/tx-and-contract.json`: on a standalone wemix network it sends
  a value tx (receipt + recipient balance), deploys the roundtrip contract
  (`codeAt` non-empty), and `eth_call`s it (returns 42). The sender is funded with
  a `genesis.overlay` alloc entry (wemix ships an empty alloc), node-signed like
  the stablenet cases.
- No standalone-bootstrap wiring into `chainbench setup` is needed for the port;
  the run path covers it.

### 5. `layer2-attach.sh` — NOT a porting target (no L2 support, no legacy script)
- There is nothing to port. The reference suite has NO z-layer2 scripts — only a
  design note (`stablenet/regression/docs/z-layer2-e2e.md`) listing RT-Z-01..05.
  "Layer 2 E2E" is not an L2 chain implementation; it is generic RPC operations
  (chain state, event logs, contract call, tx, fee delegation) run against an L2.
- chainbench does not support L2 as a chain family (no L2 binary/genesis/consensus)
  and does not need to. Its chain-agnostic cases (block-progression,
  gas-price-positive, chain-not-syncing, logs-query-well-formed, fee-history-…,
  etc. — empty ChainCompat, gated only on `rpc`) already run against ANY endpoint
  via attach mode (`chainbench test --rpc <url>`).
- "External L2" = an already-running L2 that chainbench did NOT launch and does
  not manage; it is treated as a black-box RPC endpoint. `layer2-attach.sh` is
  just a convenience wrapper that points the existing generic cases at an `L2_RPC`
  and checks they run (skip=0). It requires a live L2 URL only to execute — it is
  not a DSL/Go port of a legacy test, because no such legacy test exists.

## Notes
- `tests/repro/run-all.sh` still orchestrates the not-yet-ported bash scripts;
  it shrinks as each lands and is retired when the last one ports.
- The `bootOverlay` / `runCase` / `capabilities` harness helpers (added for
  proposal-expiry) are the infra for #2 once its overlay is fixed.
