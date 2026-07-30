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
| RPC-008 wemix_getBriocheBlockReward | — | deferred (cross-binary equality go-wemix vs go-wbft; needs the handoff chain, not a fresh go-wbft net) |
| RPC-009 eth_call governance read | — | deferred (needs deployed wemix governance) |
| RPC-010 istanbul_nodeAddress | node-address-returned | covered |
| RPC-011 istanbul_isValidator | is-validator-flags | covered |
| RPC-012 eth_getBalance | genesis-balance | covered |
| RPC-013 eth_chainId | chain-id | covered |
| RPC-014 eth_getLogs | logs-query-well-formed | covered |
| RPC-015 eth_getTransactionCount | transaction-count-increments | covered |
| RPC-016 eth_gasPrice | gas-price-positive | covered |
| RPC-017 eth_feeHistory | fee-history-well-formed | covered |
| RPC-018 txpool_status/content | txpool-status, txpool-content-well-formed | covered |
| RPC-019 admin_peers | — | deferred (needs a real multi-node peer set) |
| RPC-020 eth_subscribe newHeads | anzeon ws_subscribe (a4-06) | covered |
| RPC-021 eth_subscribe logs | anzeon ws_subscribe (a4-07) | covered |
| RPC-022 getWbftExtraInfo (epoch) | epoch-transition-carries-epoch-info | covered |
| RPC-023 isValidator after epoch | — | deferred (needs epoch validator-set change) |

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
| WBFT-011 quorum 3 validators | (subsumed by WBFT-007/008 boundary) | deferred (n-variant of the ported quorum boundary) |
| WBFT-012 quorum 6, 1 fault | (n=6 variant) | deferred (n-variant; harness now exists) |
| WBFT-013 quorum 6, 2 fault | (n=6 variant) | deferred (n-variant; harness now exists) |

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
| GOV-005/010/012/015-017 (validator/staking/stabilization/emergency) | — | follow-up (build on the registered staker) |
| GOV-013/014 (reward claims) | — | follow-up (need accrued rewards on the registered staker) |
| GOV-009 validator change | — | follow-up (build on the registered staker) |
| GOV-020/021 (fee change) | — | follow-up (GovStaking write by the registered staker's operator) |
| GOV-022/023 (claim theft guard / credential expiry) | — | follow-up |

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
`getDelegatedAmount(staker)` grows by the amount). The remaining dependent flows
(GOV-004/005/009/010–017 staking/rewards, GOV-013/014 claims, GOV-020/021 fee
change) build on a registered staker and are follow-ups on this machinery.

## NODE (7)

| wemix4 | maps to | status |
|---|---|---|
| NODE-001 hardfork transition | e2e stablenet_hardfork_swap, upgrade_run e2e | covered (e2e) |
| NODE-002 wemix data migration | — | deferred (needs pre-fork datadir fixture) |
| NODE-003 full sync | e2e sync_gap | covered (e2e) |
| NODE-004 snap sync | — | deferred (snap-mode e2e variant) |
| NODE-005 node restart | e2e sync_gap (restart path) | covered (e2e) |
| NODE-006 ncp whitespace genesis | — | deferred (genesis-parse unit test) |
| NODE-007 empty ncp genesis | — | deferred (genesis-parse unit test) |

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
- **Batch 12 (this PR)** — GOV-004 unstake:
  `TestWemixGovernanceUnstakeE2E` (same file) — the operator unstakes its full
  stake; `getStakerAmount(staker)` drops from `minimumStaking` to 0 (a partial
  unstake below the minimum is rejected, so a full unstake is the deactivation
  path). The withdrawal credential then matures over the unbonding period, which
  the test does not wait out. Live-verified against the go-wemix + go-wbft binaries.
- **Remaining:** the n-variant quorum WBFT cases (011/012/013), the GOV staking
  **dependents** (GOV-005/009/010/012/015–017 validator/staking/stabilization,
  GOV-013/014 reward claims, GOV-020/021 fee change, GOV-022/023 guards) — all
  follow-ups on the registered-staker machinery, some needing accrued rewards or a
  long unbonding/fee delay to fully complete — and RPC-008/009/019/023 + the NODE
  ops cases. The GOV read path, the NCP-governance write path, and the staking
  register/delegate/unstake writes are ported.
