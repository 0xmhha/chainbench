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
| TX-002 basefee-reject | — | deferred (needs sub-basefee send helper) |
| TX-003 dynamic-fee tx | dynamic-fee-tx | covered |
| TX-004 fee delegation | fee-delegated-transfer | covered |
| TX-005 contract deploy | contract-roundtrip | covered |
| TX-006 legacy tx | legacy-transfer | covered |
| TX-007 access-list tx | access-list-tx | covered |
| TX-008 setcode tx (7702) | set-code-delegation | covered |
| TX-009 secp256r1 valid | secp256r1-precompile-valid | **ported** |
| TX-010 nonce ordering | — | deferred (queued-nonce send helper) |
| TX-011 insufficient funds | insufficient-funds-rejected | covered |
| TX-012 gas limit exceeded | — | deferred (over-block-gas send helper) |
| TX-013 tx replacement | — | deferred (same-nonce replacement helper) |
| TX-014 FD sender sig invalid | — | deferred (malformed dual-sig helper) |
| TX-015 FD feepayer sig invalid | — | deferred (malformed dual-sig helper) |
| TX-016 FD feepayer insufficient | — | deferred (funded second key) |
| TX-017 contract revert | eth-call-revert-returns-error | covered |
| TX-018 contract out-of-gas | tx-errors (partial) | covered |
| TX-019 secp256r1 invalid | secp256r1-precompile-invalid | **ported** |
| TX-020 secp256r1 short input | secp256r1-precompile-short-input | **ported** |

## WBFT (13)

| wemix4 | maps to | status |
|---|---|---|
| WBFT-001 finalize/committedSeal | wbft-seals-quorum | covered |
| WBFT-002 block period 1s | block-period-one-second | covered |
| WBFT-003 view change | — | deferred (needs proposer-stall fault injection) |
| WBFT-005 epoch transition | epoch-transition-carries-epoch-info | covered |
| WBFT-006 round-robin proposer | — | deferred (needs multi-block proposer tracking) |
| WBFT-007 fault < 1/3 | — | deferred (needs node-stop fault injection) |
| WBFT-008 fault ≥ 1/3 | — | deferred (needs node-stop fault injection) |
| WBFT-009 prev seal | prev-seals-quorum | covered |
| WBFT-010 randao/mixdigest | randao-and-mixdigest-present | **ported** |
| WBFT-011 quorum 3 validators | — | deferred (needs fault injection) |
| WBFT-012 quorum 6, 1 fault | — | deferred (needs fault injection) |
| WBFT-013 quorum 6, 2 fault | — | deferred (needs fault injection) |

## GOV (22)

All GOV cases exercise the **deployed wemix governance** (GovConfig, GovStaking,
GovNCP, delegation, rewards, emergency mode). A fresh go-wbft testkit network has
no wemix governance, so these need a wemix→wbft handoff chain (or the remote
cluster, post-`remote bootstrap`) as the target. **Deferred** as a group until a
governance-bearing target is wired into the testkit runner. (The anzeon suite
covers a *different* governance and does not apply.)

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
- **Batch 2 (this PR)** — WBFT-010 randao/mixDigest:
  `tests/wbft/consensus/randao_mixdigest.go`. Reads the head block's
  `randaoReveal` (WBFT extra) and non-zero `mixHash` (MixDigest).
- **Next candidates (testkit-portable, no new machinery):** the TX send-helper
  gaps (basefee-reject, gaslimit, replacement, nonce, FD-failure) once a small
  set of raw-send helpers lands.
- **Blocked on machinery:** the fault-injection WBFT cases (node-stop) and the
  GOV group (governance-bearing target).
