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
| GOV-003 staker register | — | blocked (see below) |
| GOV-004/005/010-017 (staking/unstake/rewards/emergency) | — | blocked (depend on a registered staker) |
| GOV-011/013/014 (delegation/claims) | — | blocked (depend on a registered staker) |
| GOV-007/008/009/024 (NCP remove/self-exit/validator change) | — | follow-up (reachable via the NCP write path, like GOV-006) |
| GOV-020/021 (fee change) | — | follow-up (GovStaking write by an existing staker) |
| GOV-022/023 (claim theft guard / credential expiry) | — | follow-up |

### The write path, and what is reachable (confirmed live)

The **GovNCP write path is reachable**: on the handoff successor the sole NCP is
the **preset node-1 account** (`0xc17d…`), whose **raw key ships in keys/preset**
(no keystore decryption needed). Quorum is `ceil(2*ncpCount/3)`; with one NCP it
is 1, so that account can propose and vote a ballot through on its own. GOV-006
(add-NCP propose→vote→execute) is ported this way and live-verified (ncpCount
1→2, `isNCP(new)` false→true). The other NCP-governance writes
(GOV-007/008/009/024) are the same shape and are natural follow-ups.

**Blocked: the staking flows.** `registerStaker(uint256,address,address,uint256,
bytes,bytes)` reverts for an arbitrary funded EOA — a staker must first be an
approved node, and it also needs a BLS pubkey+PoP (derivable via the `bootnode`
tool, which is not built by default). So GOV-003 and everything that depends on a
registered staker (GOV-004/005/010–017 staking/rewards, GOV-011/013/014
delegation/claims) are blocked on: (1) the add-node-then-register sequence and
(2) BLS-PoP derivation machinery. These are the largest remaining GOV chunk.

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
- **Batch 8 (this PR)** — first GOV WRITE flow:
  `cmd/chainbench/upgrade_gov_write_e2e_test.go` (`TestWemixGovernanceNCPAddE2E`) —
  GOV-006 NCP add-proposal + vote. As the sole NCP (preset node-1, raw key), it
  proposes `newProposalToAddNCP`, votes the ballot through (quorum 1), and asserts
  the candidate becomes an NCP (`ncpCount` 1→2, `isNCP` false→true). Live-verified
  against the go-wemix + go-wbft binaries. Proves the GovNCP write path.
- **Remaining:** the n-variant quorum WBFT cases (011/012/013), the rest of the
  GOV **write flows** — the NCP-governance ones (GOV-007/008/009/024, same shape
  as GOV-006) as follow-ups, and the **staking flows** (GOV-003 registerStaker and
  its dependents) blocked on the add-node-then-register sequence + BLS-PoP
  derivation (see the GOV section) — plus RPC-008/009/019/023 and the NODE ops
  cases. The GOV read path and the first write flow are ported; the NCP-governance
  writes are unblocked follow-ups, the staking writes need more machinery.
