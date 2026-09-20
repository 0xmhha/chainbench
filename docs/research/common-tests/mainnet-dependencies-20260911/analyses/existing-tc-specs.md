# 기존 명세 ID 대응표 (2026-09-11)

실행 ID(JSON id)와 Confluence 명세 ID의 대응이다. 파일 해시가 2026-09-09 보존본과 같은 187개와 description만 바뀐 2개는 그때의 대응을 출발점으로 쓰고, 신규 6개는 이번에 대응했다. 2026-09-11에 Confluence Chainbench 폴더 페이지 29개를 직접 읽어(`../sources/confluence/`) 대응된 명세 ID 전부가 실제 페이지에 있는지 확인했다(상태 `confluence-verified`, 부분 대응은 `confluence-verified-partial`). 대응은 명세 전체를 구현했다는 뜻이 아니며, `[부분 대응]`은 명세 흐름의 일부만 검증한다는 뜻이다. 명세 ID 단위의 판정은 [Confluence 명세 ID 판정](confluence-spec-catalog.md)에 있다.

| 분석 ID | 파일 | 실행 ID | 명세 ID | 상태 | 근거 |
|---|---|---|---|---|---|
| TC-001 | basic/01-basic-consensus.json | basic-consensus | basic-consensus | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-002 | basic/02-basic-peers.json | basic-peers | basic-peers | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-003 | basic/03-basic-rpc-health.json | basic-rpc-health | basic-rpc-health | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-004 | basic/04-basic-sync.json | basic-sync | basic-sync | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-005 | basic/05-basic-tx-send.json | basic-tx-send | basic-tx-send | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-006 | basic/06-basic-txpool-propagation.json | basic-txpool-propagation | basic-txpool-propagation | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-007 | basic/07-basic-wbft-consensus.json | basic-wbft-consensus | basic-wbft-consensus | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-008 | fault/01-fault-network-partition.json | fault-network-partition | fault-network-partition | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-009 | fault/02-fault-node-crash.json | fault-node-crash | fault-node-crash | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-010 | fault/03-fault-node-recover.json | fault-node-recover | fault-node-recover | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-011 | fault/04-fault-p2p-topology.json | fault-p2p-topology | fault-p2p-topology | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-012 | fault/05-fault-two-down.json | fault-two-down | fault-two-down | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-013 | fault/06-fault-txpool-leader-change.json | fault-txpool-leader-change | fault-txpool-leader-change | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-014 | go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json | stablenet-delayed-fork | stablenet-delayed-fork | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-015 | go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json | govminter-v2-code | TC-5-2-05 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-016 | go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json | burn-cancel-refundable | TC-1-1-01 / TC-1-1-10 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-017 | go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json | burn-reject-refundable | TC-1-1-02 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-018 | go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json | burn-expire-refundable | TC-1-1-03 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-019 | go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json | burn-execute-no-refundable | TC-1-1-04 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-020 | go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json | claim-burn-refund-succeeds | TC-1-1-05 / TC-1-1-09 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-021 | go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json | claim-zero-refund-reverts | TC-1-1-06 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-022 | go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json | claim-burn-refund-double-reverts | TC-1-1-07 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-023 | go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json | prealloc-preserved-across-boho | TC-1-1-11 / TC-1-1-12 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-024 | go-stablenet/post-v1.0.0-change/common-all/12-legacy-gasprice-below-min-rejected.json | legacy-gasprice-below-min-rejected | TC-1-3-04 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-025 | go-stablenet/post-v1.0.0-change/common-all/13-accesslist-gasprice-below-min-rejected.json | accesslist-gasprice-below-min-rejected | TC-1-3-05 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-026 | go-stablenet/post-v1.0.0-change/common-all/14-feecap-below-min-rejected.json | feecap-below-min-rejected | TC-1-3-06 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-027 | go-stablenet/post-v1.0.0-change/common-all/15-boho-chain-config-active.json | boho-chain-config-active | TC-4-1-01 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-028 | go-stablenet/post-v1.0.0-change/common-all/16-anzeon-active-before-boho.json | anzeon-active-before-boho | TC-4-1-02 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-029 | go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json | estimategas-authorizationlist-cost | TC-4-2-01 / TC-4-2-03 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-030 | go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json | upgrade-registry-order | TC-5-2-01 / TC-5-2-02 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-031 | go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json | v1-params-init-storage | TC-5-2-04 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-032 | go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json | burn-refund-events | TC-1-1-09 / TC-1-1-10 | confluence-verified-partial | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-033 | go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json | effective-gas-price-authorized-bp-en | TC-4-6-01 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-034 | go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json | effective-gas-price-regular | TC-4-6-02 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-035 | go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json | effective-gas-price-regular-bp-en | TC-4-6-02 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-036 | go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json | auth-tx-event-last-bp-en | TC-4-6-04 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-037 | go-stablenet/post-v1.0.0-change/extra-state/01-authorized-extra-bit-synced.json | authorized-extra-bit-synced | TC-4-5-01 / TC-4-5-02 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-038 | go-stablenet/post-v1.0.0-change/extra-state/01b-blacklisted-extra-bit-synced.json | blacklisted-extra-bit-synced | TC-4-5-02 | confluence-verified-partial | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-039 | go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json | stablenet-account-extra | TC-4-5-01 / TC-4-5-02 / TC-4-5-07 | confluence-verified-partial | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-040 | go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json | extra-union-merge | TC-4-5-05 / TC-4-5-06 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-041 | go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json | dual-status-extra | TC-4-5-07 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-042 | go-stablenet/post-v1.0.0-change/extra-state/05-extra-balance-preserved.json | extra-balance-preserved | TC-4-5-08 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-043 | go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json | invalid-extra-reject | TC-4-5-09 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-044 | go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json | extra-state-across-delayed-boho | TC-4-5-10 / TC-4-5-11 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-045 | go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json | signature-compat-across-swap | TC-3-1-04 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-046 | go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json | genesis-mismatch-refuses-to-start | TC-4-1-03 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-047 | go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json | unsupported-system-contract-version | TC-5-2-06 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-048 | go-stablenet/post-v1.0.0-change/stand-alone/04-genesis-block-hash-consistent.json | genesis-block-hash-consistent | TC-5-3-01 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-049 | go-stablenet/post-v1.0.0-change/string-handling/01-authorized-accounts-no-space.json | authorized-accounts-no-space | TC-4-3-01 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-050 | go-stablenet/post-v1.0.0-change/string-handling/02-authorized-accounts-space.json | authorized-accounts-space | TC-4-3-02 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-051 | go-stablenet/post-v1.0.0-change/string-handling/03-authorized-accounts-trim.json | authorized-accounts-trim | TC-4-3-03 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-052 | go-stablenet/post-v1.0.0-change/string-handling/04-authorized-accounts-empty-item.json | authorized-accounts-empty-item | TC-4-3-04 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-053 | go-stablenet/post-v1.0.0-change/string-handling/05-authorized-accounts-single.json | authorized-accounts-single | TC-4-3-05 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-054 | go-stablenet/post-v1.0.0-change/string-handling/06-authorized-accounts-empty.json | authorized-accounts-empty | TC-4-3-06 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-055 | go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json | regular-account-gastip-forced | RT-C-01 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-056 | go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json | authorized-account-gastip-free | RT-C-02 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-057 | go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json | anzeon-basefee-increase | RT-C-03 | confluence-verified-partial | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-058 | go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json | anzeon-basefee-stable | RT-C-04 | confluence-verified-partial | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-059 | go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json | anzeon-basefee-decrease | RT-C-05 | confluence-verified-partial | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-060 | go-stablenet/regression/anzeon/06-basefee-minimum.json | basefee-minimum | RT-C-06 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-061 | go-stablenet/regression/anzeon/07-basefee-maximum.json | basefee-maximum | RT-C-07 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-062 | go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json | feecap-above-min-accepted | TC-1-3-03 | confluence-verified-partial | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-063 | go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json | feecap-exact-min-accepted | TC-1-3-02 | confluence-verified-partial | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-064 | go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json | gaslimit-exceeded-rejected | RT-A-2-07 / TX-012 | confluence-verified-partial | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-065 | go-stablenet/regression/api/01-block-transactions-field.json | block-transactions-field | RT-G-1-01 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-066 | go-stablenet/regression/api/02-block-by-hash-consistency.json | block-by-hash-consistency | RT-G-1-02 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-067 | go-stablenet/regression/api/03-transaction-by-hash-fields.json | transaction-by-hash-fields | RT-G-1-03 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-068 | go-stablenet/regression/api/04-transaction-receipt-fields.json | transaction-receipt-fields | RT-G-1-04 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-069 | go-stablenet/regression/api/05-transaction-count-increments.json | transaction-count-increments | RT-G-1-05 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-070 | go-stablenet/regression/api/06-system-contracts-deployed.json | system-contracts-deployed | RT-G-1-06 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-071 | go-stablenet/regression/api/07-gas-price-positive.json | gas-price-positive | RT-G-2-01 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-072 | go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json | gas-price-equals-basefee-plus-tip | RT-G-2-01 | confluence-verified-partial | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-073 | go-stablenet/regression/api/08-max-priority-fee-equals-gastip.json | max-priority-fee-equals-gastip | RT-G-2-02 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-074 | go-stablenet/regression/api/09-fee-history-well-formed.json | fee-history-well-formed | RT-G-2-03 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-075 | go-stablenet/regression/api/10-estimate-gas-token-transfer.json | estimate-gas-token-transfer | RT-G-2-04 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-076 | go-stablenet/regression/api/11-node-address-returned.json | node-address-returned | RT-G-3-01 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-077 | go-stablenet/regression/api/12-validator-set-nonempty.json | validator-set-nonempty | RT-G-3-02 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-078 | go-stablenet/regression/api/12b-validator-set-count.json | validator-set-count | RT-B-07 | confluence-verified-partial | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-079 | go-stablenet/regression/api/13-commit-signers-quorum.json | commit-signers-quorum | RT-G-3-03 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-080 | go-stablenet/regression/api/14-wbft-extra-info-fields.json | wbft-extra-info-fields | RT-G-3-04 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-081 | go-stablenet/regression/api/15-istanbul-status-fields.json | istanbul-status-fields | RT-G-3-05 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-082 | go-stablenet/regression/api/16-is-validator-flags.json | is-validator-flags | RT-G-3-06 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-083 | go-stablenet/regression/api/18-txpool-status.json | txpool-status | RT-G-4-02 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-084 | go-stablenet/regression/api/19-txpool-content-well-formed.json | txpool-content-well-formed | RT-G-4-03 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-085 | go-stablenet/regression/api/20-admin-peers-populated.json | admin-peers-populated | RT-A-1-05 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-086 | go-stablenet/regression/api/21-fee-delegate-sign-rpc-present.json | fee-delegate-sign-rpc-present | RT-G-5-01 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-087 | go-stablenet/regression/api/22-token-total-supply-readable.json | token-total-supply-readable | RT-G-5-02 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-088 | go-stablenet/regression/api/23-token-approve-sets-allowance.json | token-approve-sets-allowance | RT-G-5-03 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-089 | go-stablenet/regression/api/24-chain-not-syncing.json | chain-not-syncing | chain-not-syncing | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-090 | go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json | sender-blacklisted-rejected | RT-E-01 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-091 | go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json | recipient-blacklisted-rejected | RT-E-02 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-092 | go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json | feepayer-blacklisted-rejected | RT-E-03 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-093 | go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json | address-unblacklisted-event | RT-E-04 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-094 | go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json | zero-address-transfer-rejected | RT-E-05 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-095 | go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json | precompile-transfer-rejected | RT-E-06 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-096 | go-stablenet/regression/blacklist-authorized/07-account-blacklist-readable.json | account-blacklist-readable | RT-E-07 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-097 | go-stablenet/regression/blacklist-authorized/08-account-authorization-readable.json | account-authorization-readable | RT-E-08 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-098 | go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json | authorized-tx-executed-event | RT-E-09 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-099 | go-stablenet/regression/ethereum/01-stablenet-chain-up.json | stablenet-chain-up | stablenet-chain-up | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-100 | go-stablenet/regression/ethereum/08-legacy-transfer.json | legacy-transfer | RT-A-2-01 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-101 | go-stablenet/regression/ethereum/09-dynamic-fee-tx.json | dynamic-fee-tx | RT-A-2-02 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-102 | go-stablenet/regression/ethereum/10-access-list-tx.json | access-list-tx | RT-A-2-03 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-103 | go-stablenet/regression/ethereum/11-nonce-ordering.json | nonce-ordering | RT-A-2-04 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-104 | go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json | out-of-order-nonces-mine | TX-010 | confluence-verified-partial | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-105 | go-stablenet/regression/ethereum/12-dynamic-fee-below-basefee-rejected.json | dynamic-fee-below-basefee-rejected | RT-A-2-05a | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-106 | go-stablenet/regression/ethereum/14-insufficient-funds-rejected.json | insufficient-funds-rejected | RT-A-2-06 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-107 | go-stablenet/regression/ethereum/15-gas-limit-exceeds-block-rejected.json | gas-limit-exceeds-block-rejected | RT-A-2-07 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-108 | go-stablenet/regression/ethereum/16-effective-gas-price.json | effective-gas-price | RT-A-2-08 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-109 | go-stablenet/regression/ethereum/17-replacement-tx.json | replacement-tx | RT-A-2-09 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-110 | go-stablenet/regression/ethereum/17b-same-nonce-replacement.json | same-nonce-replacement | TX-013 | confluence-verified-partial | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-111 | go-stablenet/regression/ethereum/18-set-code-delegation.json | set-code-delegation | RT-A-2-10 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-112 | go-stablenet/regression/ethereum/19-contract-roundtrip.json | contract-roundtrip | RT-A-3-01 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-113 | go-stablenet/regression/ethereum/22-estimate-gas.json | estimate-gas | RT-A-3-04 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-114 | go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json | eth-call-revert-returns-error | RT-A-3-05 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-115 | go-stablenet/regression/ethereum/24-revert-tx-status-zero.json | revert-tx-status-zero | RT-A-3-06 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-116 | go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json | out-of-gas-consumes-all | RT-A-3-07 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-117 | go-stablenet/regression/ethereum/27-genesis-balance.json | genesis-balance | RT-A-4-02 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-118 | go-stablenet/regression/ethereum/27b-value-transfer.json | value-transfer | RT-A-4-03 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-119 | go-stablenet/regression/ethereum/29-logs-query-well-formed.json | logs-query-well-formed | RT-A-4-04 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-120 | go-stablenet/regression/ethereum/30-chain-id.json | chain-id | RT-A-1-01 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-121 | go-stablenet/regression/ethereum/31-ws-subscribe-new-heads.json | ws-subscribe-new-heads | RT-A-4-06 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-122 | go-stablenet/regression/ethereum/32-ws-subscribe-logs.json | ws-subscribe-logs | RT-A-4-07 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-123 | go-stablenet/regression/ethereum/33-stablenet-chain-up-15.json | stablenet-chain-up-15 | stablenet-chain-up-15 | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-124 | go-stablenet/regression/ethereum/36-contract-event-emitted.json | contract-event-emitted | RT-A-4-04 / RPC-014 | confluence-verified-partial | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-125 | go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json | fee-delegated-transfer | RT-D-01 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-126 | go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json | fd-sender-sig-invalid-rejected | RT-D-03 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-127 | go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json | fd-feepayer-sig-invalid-rejected | RT-D-04 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-128 | go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json | feepayer-insufficient-rejected | RT-D-05 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-129 | go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json | fee-delegated-sender-sig-invalid-rejected | RT-D-03 / TX-014 | confluence-verified-partial | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-130 | go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json | fee-delegated-feepayer-sig-invalid-rejected | RT-D-04 / TX-015 | confluence-verified-partial | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-131 | go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json | fee-delegated-unfunded-feepayer-rejected | RT-D-05 / TX-016 | confluence-verified-partial | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-132 | go-stablenet/regression/system-contracts/01-native-coin-adapter-code.json | native-coin-adapter-code | RT-F-1-01 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-133 | go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json | token-transfer-emits-event | RT-F-1-01 | confluence-verified-partial | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-134 | go-stablenet/regression/system-contracts/02-token-balance-readable.json | token-balance-readable | RT-F-1-02 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-135 | go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json | token-transfer-from-moves-balance | RT-F-1-03 | confluence-verified-partial | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-136 | go-stablenet/regression/system-contracts/04-mint-transfer-event.json | mint-transfer-event | RT-F-1-04 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-137 | go-stablenet/regression/system-contracts/05-burn-transfer-event.json | burn-transfer-event | RT-F-1-05 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-138 | go-stablenet/regression/system-contracts/06-mint-proposal-executes.json | mint-proposal-executes | RT-F-2-01 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-139 | go-stablenet/regression/system-contracts/07-burn-proposal-executes.json | burn-proposal-executes | RT-F-2-02 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-140 | go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json | quorum-deficient-stays-voting | RT-F-2-03 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-141 | go-stablenet/regression/system-contracts/09-validator-metadata-readable.json | validator-metadata-readable | RT-F-3-04 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-142 | go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json | gastip-governance-updates-header | RT-B-06 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-143 | go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json | proposal-expiry-transitions | RT-F-3-06 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-144 | go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json | configure-minter-proposal-executes | RT-F-4-01 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-145 | go-stablenet/regression/system-contracts/13-remove-minter-executes.json | remove-minter-executes | RT-F-4-02 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-146 | go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json | masterminter-member-add-remove | RT-F-4-03 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-147 | go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json | non-member-configure-minter-rejected | RT-F-4-04 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-148 | go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json | blacklist-proposal-executes | RT-F-5-01 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-149 | go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json | authorize-proposal-executes | RT-F-5-03 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-150 | go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json | unauthorize-proposal-executes | RT-F-5-04 / RT-F-5-09 | confluence-verified-partial | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-151 | go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json | direct-blacklist-call-rejected | RT-F-5-05 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-152 | go-stablenet/regression/system-contracts/23-authorized-account-added-event.json | authorized-account-added-event | RT-F-5-08 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-153 | go-stablenet/regression/system-contracts/25-token-metadata.json | token-metadata | token-metadata | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-154 | go-stablenet/regression/system-contracts/28-minter-status-readable.json | minter-status-readable | minter-status-readable | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-155 | go-stablenet/regression/wbft/01-block-period-one-second.json | block-period-one-second | RT-B-01 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-156 | go-stablenet/regression/wbft/02-wbft-seals-quorum.json | wbft-seals-quorum | RT-B-02 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-157 | go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json | epoch-transition-carries-epoch-info | RT-B-03 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-158 | go-stablenet/regression/wbft/04-validator-add-member-executes.json | validator-add-member-executes | RT-B-04 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-159 | go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json | validator-add-member-epoch-activates | RT-B-04 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-160 | go-stablenet/regression/wbft/05-validator-remove-member-executes.json | validator-remove-member-executes | RT-B-05 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-161 | go-stablenet/regression/wbft/11-prev-seals-quorum.json | prev-seals-quorum | RT-B-11 | confluence-verified | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-162 | go-stablenet/regression/wbft/13-randao-and-mixdigest-present.json | randao-and-mixdigest-present | WBFT-010 | confluence-verified-partial | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-163 | go-stablenet/regression/wbft/14-stablenet-gastip-field.json | stablenet-gastip-field | RT-B-06 | confluence-verified-partial | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-164 | go-stablenet/topology/01-proxied-pn-routing.json | stablenet-proxied-pn-routing | stablenet-proxied-pn-routing | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-165 | go-stablenet/tx/01-negative-tx-revert.json | stablenet-negative-tx-revert | stablenet-negative-tx-revert | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-166 | go-stablenet/vocabulary/01-derived-address-and-checksum.json | stablenet-derived-vocabulary | stablenet-derived-vocabulary | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-167 | go-stablenet/vocabulary/02-faucet-funds-account.json | stablenet-faucet-funds | stablenet-faucet-funds | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-168 | go-stablenet/vocabulary/03-metric-head-block.json | stablenet-metric-head-block | stablenet-metric-head-block [명세 ID 미확인] | unconfirmed | 2026-09-11 신규 대응 |
| TC-169 | go-stablenet/vocabulary/03-register-contract.json | stablenet-register-contract | stablenet-register-contract | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-170 | go-wbft/accounts/01-secp256r1-precompile-valid.json | secp256r1-precompile-valid | TX-009 | confluence-verified-partial | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-171 | go-wbft/accounts/02-secp256r1-precompile-invalid.json | secp256r1-precompile-invalid | TX-019 | confluence-verified-partial | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-172 | go-wbft/accounts/03-secp256r1-precompile-short-input.json | secp256r1-precompile-short-input | TX-020 | confluence-verified-partial | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-173 | go-wbft/chain-up/01-wbft-chain-up.json | wbft-chain-up | wbft-chain-up | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-174 | go-wbft/chain-up/02-wbft-chain-up-15.json | wbft-chain-up-15 | wbft-chain-up-15 | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-175 | go-wbft/consensus/01-e1-mixed-producers.json | e1-mixed-producers | e1-mixed-producers | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-176 | go-wbft/fault/01-wbft-node-crash.json | wbft-node-crash | wbft-node-crash | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-177 | go-wbft/network/01-wbft-proxied-routing.json | wbft-proxied-routing | wbft-proxied-routing [명세 ID 미확인] | unconfirmed | 2026-09-11 신규 대응 |
| TC-178 | go-wbft/tx/01-wbft-tx-and-contract.json | wbft-tx-and-contract | wbft-tx-and-contract | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-179 | go-wbft/tx/02-wbft-insufficient-funds-rejected.json | wbft-insufficient-funds-rejected | RT-A-2-06 / TX-011 [부분 대응] | confluence-verified-partial | 2026-09-11 신규 대응 |
| TC-180 | go-wbft/tx/03-wbft-revert-status-zero.json | wbft-revert-status-zero | RT-A-3-06 / TX-017 [부분 대응] | confluence-verified-partial | 2026-09-11 신규 대응 |
| TC-181 | go-wemix/chain-up/01-wemix-chain-up.json | wemix-chain-up | wemix-chain-up | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-182 | go-wemix/chain-up/02-wemix-chain-up-15.json | wemix-chain-up-15 | wemix-chain-up-15 | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-183 | go-wemix/fault/01-wemix-node-crash.json | wemix-node-crash | wemix-node-crash | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-184 | go-wemix/handoff/01-wemix-wbft-handoff.json | wemix-wbft-handoff | wemix-wbft-handoff | unconfirmed | 2026-09-09 대응표 승계 (2026-09-11 #384 이후 파일 변경, steps 재확인) |
| TC-185 | go-wemix/rpc/01-wemix-brioche-block-reward.json | wemix-brioche-block-reward | BRIOCHE-02 / RPC-008 | confluence-verified-partial | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-186 | go-wemix/tx/01-wemix-tx-and-contract.json | wemix-tx-and-contract | wemix-tx-and-contract | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-187 | go-wemix/tx/02-wemix-insufficient-funds-rejected.json | wemix-insufficient-funds-rejected | RT-A-2-06 / TX-011 [부분 대응] | confluence-verified-partial | 2026-09-11 신규 대응 |
| TC-188 | go-wemix/tx/03-wemix-revert-status-zero.json | wemix-revert-status-zero | RT-A-3-06 / TX-017 [부분 대응] | confluence-verified-partial | 2026-09-11 신규 대응 |
| TC-189 | remote/01-remote-rpc-health.json | remote-rpc-health | remote-rpc-health | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-190 | remote/02-remote-chain-info.json | remote-chain-info | remote-chain-info | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-191 | remote/03-remote-balance-check.json | remote-balance-check | remote-balance-check | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-192 | samples/01-sample-minimal.json | sample-minimal-value-transfer | sample-minimal-value-transfer | unconfirmed | 2026-09-09 대응표 승계 (description만 변경, steps 동일 확인) |
| TC-193 | samples/02-sample-lifecycle.json | sample-lifecycle-node-restart | sample-lifecycle-node-restart | unconfirmed | 2026-09-09 대응표 승계 (description만 변경, steps 동일 확인) |
| TC-194 | stress/01-stress-block-time.json | stress-block-time | stress-block-time | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |
| TC-195 | stress/02-stress-tx-flood.json | stress-tx-flood | stress-tx-flood | unconfirmed | 2026-09-09 대응표 승계 (파일 해시 동일) |

상태별: unconfirmed 43건, confluence-verified 122건, confluence-verified-partial 30건. unconfirmed 43건은 명세 ID가 없는 하네스·기본 케이스이며 PR196 144 목록이 그중 37건에 PR196-TC 관리 ID를 부여했다.
