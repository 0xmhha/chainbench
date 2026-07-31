# wemix4 test-case port tracker (Phase 6)

Phase 6 of the wemix4 migration ports the `tests/wemix4/{NODE,WBFT,RPC,GOV,TX}`
shell cases into the chainbench Go **testkit** (`tests/wbft/...`, run by
`pkg/core/pipeline/testrun`) — NOT a blind 1:1 re-port.

A large share of wemix4's coverage **already exists** in the testkit corpus
(`tests/wbft/consensus`, `tests/wbft/accounts`, `tests/api`), ported earlier
from the ethereum/wbft regression suites. Phase 6 is therefore driven by a **gap
analysis**: port only what is not already covered, and defer what needs
machinery the testkit does not yet have (multi-node fault injection, a deployed
wemix governance).

Status legend: **covered** (an existing testkit case already asserts it) ·
**ported** (added in Phase 6) · **deferred** (needs new machinery — see reason).

## Remaining work (prioritized)

Everything else in wemix4 is ported or already covered. What is left, most
tractable first:

1. ~~NODE-004 snap sync~~ — **ported** (`TestE2E_WbftSnapSync`): wbft boot with a
   `snap`-sync endpoint, open a ~90-block gap, confirm re-sync + matching
   hash/stateRoot + `eth_syncing` false.
2. ~~NODE-006 / NODE-007 genesis NCP-parse~~ — **ported**
   (`TestE2E_WbftGenesisNCPWhitespace` / `TestE2E_WbftGenesisEmptyNCP`): `gwemix
   init` accepts whitespace-padded NCP addresses (trimmed) and rejects an empty
   NCP list ("govNCP is configured but no initial NCPs provided").
3. ~~GOV-012 / GOV-013 / GOV-014 block-reward + reward claims~~ — **ported**:
   register a *producing* validator (node2) as a staker via a distinct operator
   (node3); `previewReward` grows block over block (GOV-012); the operator claims
   and its balance rises (GOV-013); a delegator (node4) accrues its share and
   claims (GOV-014). Live-verified.
4. GOV-021 delayed fee change — **ported** (`TestWemixGovernanceFeeChangeDelayedE2E`:
   with a delegator, requestChangingFee records a pending request and does NOT
   apply the fee immediately; the execute-after-`changeFeeDelay` half is not
   waited out). GOV-023 credential expiry still needs the unbonding window. *Partly done.*
5. **GOV-005 / GOV-009 / GOV-010 + RPC-023** — need a full wemix network where
   governance drives the validator set at epoch transitions; the minimal handoff
   runs a static validator set (see the epoch note). *Blocked by target.*
6. **NODE-002 data migration** — import a pre-fork wemix datadir fixture and
   migrate; needs that fixture. *Deferred.*
7. **RPC-008 wemix_getBriocheBlockReward** — needs the genesis configured with a
   `brioche` halving object (a dedicated setup, to be provided) and the wemix RPC
   namespace on a node that has it. **Do LAST.**

## RPC (23)

| wemix4 | maps to | status |
|---|---|---|
| RPC-001 eth_blockNumber | chain-advancing (implicit in every case; block_period) | covered |
| RPC-002 get block by number/hash | block-by-hash-consistency, block-transactions-field | covered |
| RPC-003 istanbul_getValidators | validator-set-count / -nonempty | covered |
| RPC-004 getCommitSignersFromBlock | commit-signers-quorum | covered |
| RPC-005 getWbftExtraInfo (normal) | wbft-extra-info-fields | covered |
| RPC-006 istanbul_status | istanbul-status-fields | covered |
| RPC-007 getTransactionReceipt | transaction-receipt-fields | covered |
| RPC-008 wemix_getBriocheBlockReward | — | **deferred — do LAST** (the `wemix_getBriocheBlockReward` RPC registers only when the genesis carries a `brioche` halving-config object — distinct from the `briocheBlock` fork switch; the go-wemix template ships only `briocheBlock`. There is a dedicated way to configure brioche for this test — pending that setup; lowest priority) |
| RPC-009 eth_call governance read | e2e TestWemixGovernanceE2E (GovStaking.totalStaking) | **ported** (e2e) |
| RPC-010 istanbul_nodeAddress | node-address-returned | covered |
| RPC-011 istanbul_isValidator | is-validator-flags | covered |
| RPC-012 eth_getBalance | genesis-balance | covered |
| RPC-013 eth_chainId | chain-id | covered |
| RPC-014 eth_getLogs | logs-query-well-formed | covered |
| RPC-015 eth_getTransactionCount | transaction-count-increments | covered |
| RPC-016 eth_gasPrice | gas-price-positive | covered |
| RPC-017 eth_feeHistory | fee-history-well-formed | covered |
| RPC-018 txpool_status/content | txpool-status, txpool-content-well-formed | covered |
| RPC-019 admin_peers | e2e TestE2E_WbftAdminPeers | **ported** (e2e) |
| RPC-020 eth_subscribe newHeads | anzeon ws_subscribe (a4-06) | covered |
| RPC-021 eth_subscribe logs | anzeon ws_subscribe (a4-07) | covered |
| RPC-022 getWbftExtraInfo (epoch) | epoch-transition-carries-epoch-info | covered |
| RPC-023 isValidator after epoch | — | **blocked by target** (validator set is static; see epoch note) |

## TX (20)

| wemix4 | maps to | status |
|---|---|---|
| TX-001 normal tx | value-transfer, effective-gas-price | covered |
| TX-002 basefee-reject | dynamic-fee-below-basefee-rejected | **ported** |
| TX-003 dynamic-fee tx | dynamic-fee-tx | covered |
| TX-004 fee delegation | fee-delegated-transfer | covered |
| TX-005 contract deploy | contract-roundtrip | covered |
| TX-006 legacy tx | legacy-transfer | covered |
| TX-007 access-list tx | access-list-tx | covered |
| TX-008 setcode tx (7702) | set-code-delegation | covered |
| TX-009 secp256r1 valid | secp256r1-precompile-valid | **ported** |
| TX-010 nonce ordering | out-of-order-nonces-mine | **ported** |
| TX-011 insufficient funds | insufficient-funds-rejected | covered |
| TX-012 gas limit exceeded | gas-limit-exceeds-block-rejected | **ported** |
| TX-013 tx replacement | same-nonce-replacement | **ported** |
| TX-014 FD sender sig invalid | fee-delegated-sender-sig-invalid-rejected | **ported** |
| TX-015 FD feepayer sig invalid | fee-delegated-feepayer-sig-invalid-rejected | **ported** |
| TX-016 FD feepayer insufficient | fee-delegated-unfunded-feepayer-rejected | **ported** |
| TX-017 contract revert | eth-call-revert-returns-error | covered |
| TX-018 contract out-of-gas | tx-errors (partial) | covered |
| TX-019 secp256r1 invalid | secp256r1-precompile-invalid | **ported** |
| TX-020 secp256r1 short input | secp256r1-precompile-short-input | **ported** |

## WBFT (13)

| wemix4 | maps to | status |
|---|---|---|
| WBFT-001 finalize/committedSeal | wbft-seals-quorum | covered |
| WBFT-002 block period 1s | block-period-one-second | covered |
| WBFT-003 view change | e2e TestE2E_WbftViewChange | **ported** (e2e) |
| WBFT-005 epoch transition | epoch-transition-carries-epoch-info | covered |
| WBFT-006 round-robin proposer | e2e TestE2E_WbftRoundRobinProposer | **ported** (e2e) |
| WBFT-007 fault < 1/3 | e2e TestE2E_WbftFaultTolerance | **ported** (e2e) |
| WBFT-008 fault ≥ 1/3 | e2e TestE2E_WbftFaultHalt | **ported** (e2e) |
| WBFT-009 prev seal | prev-seals-quorum | covered |
| WBFT-010 randao/mixdigest | randao-and-mixdigest-present | **ported** |
| WBFT-011 quorum 3 validators | e2e TestE2E_WbftQuorumAllRequired | **ported** (e2e) |
| WBFT-012 quorum 6, 1 fault | e2e TestE2E_WbftQuorum6of6Tolerates1 | **ported** (e2e, generated preset) |
| WBFT-013 quorum 6, 2 fault | e2e TestE2E_WbftQuorum6of6Halts2 | **ported** (e2e, generated preset) |

## GOV (22)

All GOV cases exercise the **deployed wemix governance** (GovConfig, GovStaking,
GovNCP, delegation, rewards, emergency mode). Key fact confirmed live: the
governance system contracts are deployed at the **Croissant fork block**, so they
live on the **go-wbft successor** chain (the go-wemix producer stops at
croissant-1 and never sees them) at the fixed addresses `0x..1000` GovConfig /
`0x..1001` GovStaking / `0x..1002` GovRewardee / `0x..1003` GovNCP. The target is
therefore the `chainbench upgrade run` handoff (or the remote cluster,
post-`remote bootstrap`). (The anzeon suite covers a *different* governance and
does not apply.)

| wemix4 | maps to | status |
|---|---|---|
| GOV-001 contract deploy | e2e TestWemixGovernanceE2E | **ported** (e2e) |
| GOV-002 config params | e2e TestWemixGovernanceE2E | **ported** (e2e) |
| GOV-006 NCP add proposal + vote | e2e TestWemixGovernanceNCPAddE2E | **ported** (e2e) |
| GOV-007 NCP remove by vote | e2e TestWemixGovernanceNCPLifecycleE2E | **ported** (e2e) |
| GOV-008 NCP self-exit (immediate) | e2e TestWemixGovernanceNCPLifecycleE2E | **ported** (e2e) |
| GOV-024 NCP staker-only validator | (merged into GOV-009 upstream) | n/a |
| GOV-003 staker register | e2e TestWemixGovernanceRegisterStakerE2E | **ported** (e2e) |
| GOV-011 delegation | e2e TestWemixGovernanceDelegateE2E | **ported** (e2e) |
| GOV-004 unstake | e2e TestWemixGovernanceUnstakeE2E | **ported** (e2e) |
| GOV-020 fee change (immediate, no delegators) | e2e TestWemixGovernanceFeeChangeE2E | **ported** (e2e) |
| GOV-022 claim theft guard | e2e TestWemixGovernanceClaimGuardE2E | **ported** (e2e) |
| GOV-015 unstake below minimum rejected | e2e TestWemixGovernanceUnstakeMinimumGuardE2E | **ported** (e2e) |
| GOV-016 inactive→active (unstake then re-stake) | e2e TestWemixGovernanceReactivateE2E | **ported** (e2e) |
| GOV-017 emergency mode blocks staking | e2e TestWemixGovernanceEmergencyModeE2E | **ported** (e2e) |
| GOV-005 staker→validator reflection | — | **blocked by target** (static validator set; see epoch note) |
| GOV-009 validator change | — | **blocked by target** (static validator set; see epoch note) |
| GOV-010 stabilization stage | — | **blocked by target** (no epochInfo at epoch boundaries; see epoch note) |
| GOV-012 block reward accumulation | e2e TestWemixGovernanceBlockRewardE2E | **ported** (e2e) |
| GOV-013 operator claim | e2e TestWemixGovernanceOperatorClaimE2E | **ported** (e2e) |
| GOV-014 delegator claim | e2e TestWemixGovernanceDelegatorClaimE2E | **ported** (e2e) |
| GOV-021 fee change (delayed request) | e2e TestWemixGovernanceFeeChangeDelayedE2E | **ported** (e2e) |
| GOV-023 credential expiry | — | deferred (needs the unbonding/expiry window to elapse) |

### The write path, and what is reachable (confirmed live)

The **GovNCP write path is reachable**: on the handoff successor the initial NCP
is the **preset node-1 account** (`0xc17d…`), whose **raw key ships in keys/preset**
(no keystore decryption needed); the other preset validator accounts (node2/node3)
supply additional NCP votes. Quorum is `ceil(2*ncpCount/3)`. The full NCP
lifecycle is ported and live-verified: GOV-006 add (propose→vote), GOV-007 remove
by vote of the others, and GOV-008 immediate self-exit (proposing to remove
oneself executes without a vote) — `add node2 → add node3 → remove node3 → node2
self-exit`, with `ncpCount` walking 1→2→3→2→1 and `isNCP` flipping at each step.
GOV-024 is upstream-merged into GOV-009 (a no-op pass) and does not apply.

**The staking flows — machinery built, GOV-003 ported.** `registerStaker(uint256,
address,address,uint256,bytes,bytes)` is reachable on the handoff successor. The
earlier revert was a false lead: it was NOT the council gate — `GovNCP` is the
`govCouncil`, and its `inspectOperation` returns `!emergencyMode`, so it permits
operations by default. The real requirements (from `GovStaking.sol`) are:

- `msg.value == amount`, and `minimumStaking <= amount <= maximumStaking`;
- the operator (`msg.sender`) must DIFFER from the staker (`"operator cannot be
  staker"`) — the original probe used the same address, hence the revert;
- a valid BLS public key + proof-of-possession for the staker.

The BLS pubkey/PoP for each preset node are derived once from the committed
nodekeys via the go-wbft `bootnode -writeaddress` tool and now ship in
`keys/preset/metadata.json` (`blsPublicKey` + `blsPoP`), so tests need no extra
binary. GOV-003 is ported (`TestWemixGovernanceRegisterStakerE2E`: operator=node2,
staker=node1, amount=minimumStaking → `isStaker` true, `stakerByOperator` maps
back). GOV-011 delegation builds on it (`TestWemixGovernanceDelegateE2E`: node3
delegates to the active staker via `delegate(staker,amount)` payable →
`getDelegatedAmount(staker)` grows by the amount).

### Epoch / validator-set note (why GOV-005/009/010 + RPC-023 are blocked)

Probed live on the handoff successor: `istanbul_getValidators` returns the same
four croissant.init validators at every block — including across epoch boundaries
(epochLength 100; checked at blocks 99/100/101 and 199/200/201) — and
`istanbul_getWbftExtraInfo.epochInfo` is populated only at the fork block (20),
null at the epoch boundaries. So the chainbench minimal handoff runs a **static
validator set**; governance staker registration does not add a node to the
block-producing set, and there is no per-epoch validator rotation to observe.
GOV-005 (staker→validator), GOV-009 (validator change), GOV-010 (stabilization
via epochInfo), and RPC-023 (isValidator after epoch) therefore cannot be
reproduced here — they need a full wemix network where governance drives the
validator set at epoch transitions.

### Handoff reliability (test infrastructure)

The GOV/staking e2e all boot a wemix→wbft handoff via `runGovHandoff`. The
go-wemix producer's embedded etcd intermittently fails to bootstrap — it enters a
join-failure loop and the chain halts at ~block 10, before the fork — so a single
launch is flaky (worse on a long-lived machine). Two pieces make it reliable:

- **`pkg/core/procman`** — a process-lifecycle manager that tracks every launched
  node PID (parsed from the launch output / nodeset.json) and tears them down
  verifiably (SIGTERM → SIGKILL → confirm gone, reporting any leak). This replaces
  the best-effort `pkill -f <datadir>` sweeps that were leaving orphans holding
  etcd's ports and poisoning subsequent boots.
- **retry in `runGovHandoff`** — up to `govHandoffAttempts` launches; each failed
  attempt is torn down cleanly via procman (no orphans) before the next. All GOV
  handoff tests route through it (the earlier inline boots were migrated).

## NODE (7)

| wemix4 | maps to | status |
|---|---|---|
| NODE-001 hardfork transition | e2e stablenet_hardfork_swap, upgrade_run e2e | covered (e2e) |
| NODE-002 wemix data migration | — | deferred (needs pre-fork datadir fixture) |
| NODE-003 full sync | e2e sync_gap | covered (e2e) |
| NODE-004 snap sync | e2e TestE2E_WbftSnapSync | **ported** (e2e) |
| NODE-005 node restart | e2e sync_gap (restart path) | covered (e2e) |
| NODE-006 ncp whitespace genesis | e2e TestE2E_WbftGenesisNCPWhitespace | **ported** (e2e) |
| NODE-007 empty ncp genesis | e2e TestE2E_WbftGenesisEmptyNCP | **ported** (e2e) |

## Batches

- **Batch 1** — TX-009/019/020 secp256r1 (RIP-7212 P256VERIFY) precompile:
  `tests/wbft/accounts/secp256r1_precompile.go`. Deterministic, read-only, no
  funding — genuinely new capability coverage.
- **Batch 2** — WBFT-010 randao/mixDigest:
  `tests/wbft/consensus/randao_mixdigest.go`. Reads the head block's
  `randaoReveal` (WBFT extra) and non-zero `mixHash` (MixDigest).
- **Batch 3** — TX-policy rejections/ordering:
  `tests/wbft/accounts/tx_rejections.go` — TX-002 (below-basefee reject),
  TX-012 (over-block-gas reject), TX-010 (out-of-order nonces mine), TX-013
  (same-nonce replacement), TX-016 (unfunded fee payer reject). Built on the
  existing `SendDynamicFeeTx`/`SendDynamicFeeGas`/`SendFeeDelegated` API — no
  new machinery.
- **Batch 4** — TX-014/015 fee-delegation invalid-signature:
  `tests/wbft/accounts/tx_fd_sig.go`. Reuses the existing
  `accounts.EncodeFeeDelegatedTampered` builder (sender/feepayer signature
  corruption) and submits via `eth_sendRawTransaction`, expecting rejection.
- **Batch 5** — WBFT fault tolerance (the first fault-injection port):
  `tests/e2e/wbft_fault_test.go` — WBFT-007 (`TestE2E_WbftFaultTolerance`:
  4 validators, stop 1 → 3/4 == quorum → consensus continues, node rejoins and
  re-syncs) and WBFT-008 (`TestE2E_WbftFaultHalt`: stop 2 → 2/4 below quorum →
  production halts, restart resumes). Uses the e2e harness node-stop machinery;
  live-verified against the go-wbft binary. This establishes the fault-injection
  harness the remaining WBFT cases build on.
- **Batch 6** — WBFT proposer behavior:
  `tests/e2e/wbft_proposer_test.go` — WBFT-006 (`TestE2E_WbftRoundRobinProposer`:
  the miner rotates across a block window — ≥3 distinct proposers over 16 blocks
  on a 4-validator net) and WBFT-003 (`TestE2E_WbftViewChange`: killing the
  proposer triggers a round change; the chain keeps advancing from a surviving
  validator with parentHash links intact, then the node restarts). Live-verified
  against the go-wbft binary.
- **Batch 7** — GOV read cases on the handoff successor:
  `cmd/chainbench/upgrade_gov_e2e_test.go` (`TestWemixGovernanceE2E`) — drives the
  `upgrade run` handoff and asserts GOV-001 (all four governance contracts carry
  code at `0x..1000-1003`) and GOV-002 (GovConfig params > 0) on the go-wbft
  successor. Live-verified: governance is deployed at the fork block, on the
  successor, not on the wemix producer.
- **Batch 8** — first GOV WRITE flow:
  `cmd/chainbench/upgrade_gov_write_e2e_test.go` (`TestWemixGovernanceNCPAddE2E`) —
  GOV-006 NCP add-proposal + vote. As the sole NCP (preset node-1, raw key), it
  proposes `newProposalToAddNCP`, votes the ballot through (quorum 1), and asserts
  the candidate becomes an NCP (`ncpCount` 1→2, `isNCP` false→true). Live-verified
  against the go-wemix + go-wbft binaries. Proves the GovNCP write path.
- **Batch 9** — the rest of the reachable NCP write flows:
  `cmd/chainbench/upgrade_gov_ncp_lifecycle_e2e_test.go`
  (`TestWemixGovernanceNCPLifecycleE2E`) — GOV-007 (remove an NCP by vote of the
  others) and GOV-008 (immediate self-exit), driven as one NCP lifecycle
  (add→add→remove→self-exit) with multi-NCP quorum voting by the preset validator
  accounts. Live-verified against the go-wemix + go-wbft binaries.
- **Batch 10** — the staking machinery + GOV-003:
  `cmd/chainbench/upgrade_gov_staking_e2e_test.go`
  (`TestWemixGovernanceRegisterStakerE2E`) — `GovStaking.registerStaker` with
  operator=node2, staker=node1, amount=minimumStaking, and the staker's BLS
  pubkey/PoP (now shipped in `keys/preset/metadata.json`). Asserts `isStaker` and
  `stakerByOperator`. Live-verified against the go-wemix + go-wbft binaries. This
  unblocks the dependent staking flows.
- **Batch 11** — GOV-011 delegation:
  `TestWemixGovernanceDelegateE2E` (same file) — after registering the staker,
  node3 delegates to it via `delegate(staker,amount)` payable, and
  `getDelegatedAmount(staker)` grows by the delegated amount. Reuses the extracted
  `stakingRegister` helper. Live-verified against the go-wemix + go-wbft binaries.
- **Batch 12** — GOV-004 unstake:
  `TestWemixGovernanceUnstakeE2E` (same file) — the operator unstakes its full
  stake; `getStakerAmount(staker)` drops from `minimumStaking` to 0 (a partial
  unstake below the minimum is rejected, so a full unstake is the deactivation
  path). The withdrawal credential then matures over the unbonding period, which
  the test does not wait out. Live-verified against the go-wemix + go-wbft binaries.
- **Batch 13** — GOV-020 fee change (immediate path):
  `TestWemixGovernanceFeeChangeE2E` (same file) — with no delegators,
  `requestChangingFee(rate)` applies the new fee immediately (with delegators it
  becomes a delayed request). Registers a staker with feeRate 0, the operator
  requests a new rate, and `stakerInfo.feeRate` (word 3 of the struct getter)
  updates at once. Live-verified against the go-wemix + go-wbft binaries.
- **Batch 14** — GOV-022 claim theft guard:
  `TestWemixGovernanceClaimGuardE2E` (same file) — a third party (node3) that is
  neither the staker's operator nor a delegator calls `claim(staker,false)`; the
  guard resolves the caller to a user with no stake/pending reward and reverts
  ("no reward to claim"), and the staker's rewardee balance is untouched. Needs no
  accrued rewards. Live-verified against the go-wemix + go-wbft binaries.
- **Batch 15** — GOV-015 unstake-below-minimum guard:
  `TestWemixGovernanceUnstakeMinimumGuardE2E` (same file) — a staker registered at
  exactly `minimumStaking` cannot be partially unstaked; unstaking 1 wei would
  leave `minimumStaking-1` (neither `>= minimum` nor 0), so `unstake` reverts
  ("amount must equal balance to deactivate staker") and the stake is unchanged.
  Live-verified against the go-wemix + go-wbft binaries.
- **Infra** — after the handoff boot proved flaky (the go-wemix producer's
  embedded etcd intermittently fails to bootstrap), `runGovHandoff` gained a retry
  backed by `pkg/core/procman` (tracked-PID, verified teardown → no orphans). All
  GOV handoff tests route through it. Also added config-driven local topology
  (`pkg/core/topology`, `setup --topology`).
- **Batch 16** — GOV-016 + GOV-017 (bundled):
  `TestWemixGovernanceReactivateE2E` (GOV-016: a full unstake deactivates the
  staker — `isStaker` false — and `stake()` reactivates it — `isStaker` true, the
  staker staying registered throughout) and `TestWemixGovernanceEmergencyModeE2E`
  (GOV-017: the NCP council enters emergency mode via propose+vote, which blocks a
  guarded `stake()` via `inspectWithCouncil`, then leaves it and the flag clears).
  Both live-verified on the now-reliable handoff.
- **Batch 17 (this PR)** — WBFT-011 + RPC-019 (bundled, reliable wbft boot):
  `TestE2E_WbftQuorumAllRequired` (WBFT-011: at n=3 the quorum is the whole set —
  wemix WBFT quorum = floor(2n/3)+1 = 3 — so stopping one validator halts
  production; restarting resumes it; live-confirmed the strict quorum) and
  `TestE2E_WbftAdminPeers` (RPC-019: on a multi-node network `admin_peers` reports
  a non-empty peer set). The n=6 quorum variants (WBFT-012/013) stay blocked on a
  6+ node preset (the current preset ships 5). Both live-verified.
- **Remaining:** the n=6 quorum WBFT cases (012/013, need a larger preset), the GOV staking
  **dependents** that need machinery this harness doesn't produce quickly —
  GOV-005/009/010 + RPC-023 (validator-set reflection / stabilization) are BLOCKED
  by the static validator set (see the epoch note), GOV-012/013/014 (block reward + reward claims,
  need accrued rewards), GOV-021 (delayed fee change) / GOV-023 (credential
  expiry) (long fee/expiry windows) — and RPC-008/009/019/023 + the NODE ops
  cases. The GOV read path, the NCP-governance write path, and the staking
  register/delegate/unstake/fee-change/reactivate writes plus the claim,
  unstake-minimum, and emergency-mode guards are ported.
