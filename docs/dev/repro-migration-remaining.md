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

## Remaining (5 bash scripts) — each blocked on real prerequisite work

These are NOT simple ports; each needs a separate track. Verified against
gstable v1.0.1 AND v1.1.0 (same results) — not a chain-version issue.

### 1. `stablenet-delayed-fork.sh` — testkit-case ↔ chain boho-behavior mismatch
- Boots stablenet with `--set genesis.overrides.bohoBlock=N` (verified: the
  rendered genesis has `"bohoBlock": N` and gstable's config reads
  `json:"bohoBlock"`, so the override key is correct — not a chainbench bug).
- The chain crosses block N, but the gated cases time out (2m):
  `govminter-code-changes-at-boho`, `p256-inactive-before-boho`,
  `prealloc-preserved-across-boho` FAIL; `anzeon-active-before-boho` PASSES.
- Root cause: the boho activation effects the testkit cases assert (GovMinter-v2
  code injection, P-256 precompile at 0x100) do not manifest as expected in the
  built gstable. Both v1.0.1 and v1.1.0 fail identically.
- **Needed:** reconcile the testkit cases (`tests/anzeon/hardfork_reads.go`,
  `fork_transition.go`, etc.) with the actual go-stablenet boho implementation
  (addresses, activation semantics) — or a gstable build matching the cases.

### 2. `stablenet-account-extra.sh` — overlay fixture bug
- Genesis init FAILS: `cannot unmarshal array into … govCouncil.params of type
  string`. Fails on both v1.0.1 and v1.1.0.
- The overlay `internal/chains/stablenet/overlays/account-extra.json` merges
  `govCouncil.params.{authorizedAddresses,blacklistedAddresses}` (arrays) — but
  the base template's `govCouncil.params` is a flat string-map
  (`members/quorum/expiry/…`). The intent is to seed account **Extra bits
  (62/63)** in the alloc, not to put arrays in govCouncil.params.
- **Needed:** rewrite the overlay to set the Extra-bit account state in the
  genesis `alloc` in the format go-stablenet expects (investigate
  `account.Extra` genesis representation), then port to Go with `bootOverlay` +
  `runCase` (the harness already supports this — see proposal-expiry).

### 3. `stablenet-basefee-dynamics.sh` — SUPERSEDED by the DSL anzeon cases
- ~~Needs a burst of many txs into ONE block to move baseFee.~~ Resolved a
  different way: the DSL `load` action (`internal/testhelper/load.go`) deploys a
  single gas-burner sized to a percent of the block gas limit, so one tx fills the
  block — no explicit-nonce burst needed. The base-fee dynamics are now covered by
  `tests/cases/anzeon/basefee-{increase,stable,decrease}.json`, live-verified
  against a real gstable network (see `legacy-test-migration.md` §5c).
- No Go-e2e port of this script is needed; it can be retired from `run-all.sh`.

### 4. `wemix-chain.sh` (scenario 1, pure wemix) — framework gap
- `chainbench setup --launch --chain wemix` does NOT run the governance-etcd
  bootstrap standalone (only the upgrade/handoff path drives poa bootstrap via
  `internal/consensus/poa` + a `Bootstrap` callback). The bash script does the raw
  wemix genesis + init + launch + `deploy-governance` + `admin.etcdInit()`.
- **Needed:** wire a standalone wemix bootstrap into `chainbench setup` (drive
  `internal/consensus/poa.BootstrapPlan` steps for a non-handoff wemix network), then
  port as a Go test. (Lower priority — the handoff test already exercises the
  full wemix+etcd+governance bring-up.)

### 5. `layer2-attach.sh` — external resource
- Needs an already-running Layer-2 RPC endpoint (`L2_RPC`). Chain-agnostic
  read/write cases already work via `chainbench test --rpc <L2>`.
- **Needed:** a live L2 endpoint to verify against; the Go port is otherwise
  straightforward (attach + run rpc-only cases).

## Notes
- `tests/repro/run-all.sh` still orchestrates the not-yet-ported bash scripts;
  it shrinks as each lands and is retired when the last one ports.
- The `bootOverlay` / `runCase` / `capabilities` harness helpers (added for
  proposal-expiry) are the infra for #2 once its overlay is fixed.
