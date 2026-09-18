# 기존 TC 스펙과 ID 대응

분석 보고서의 TC-001~TC-189 및 DOC-C/D 번호는 분석용 별칭이다. 기존 TC ID나 실행 ID를 대체하지 않는다. 현재 JSON 189개에는 모두 고유한 최상위 id가 있고, 그중 122개 description에서 원래 명세 ID 형식이 확인된다. 나머지 67개는 description에 그 형식이 없다는 뜻이며 원래 명세 자체가 없다는 판정은 아니다.

원래 명세 번호는 RT-A/RT-B/RT-D 계열, StableNet의 TC-숫자-숫자-숫자 계열, WEMIX4의 TX/RPC/NODE/GOV/WBFT 계열로 나뉜다. 원래 번호 표현의 /·~ 축약은 보존하며 임의로 모두 확장하지 않는다. 같은 명세가 여러 JSON으로 나뉘거나 여러 명세가 하나의 JSON으로 통합되므로 일대일 대응을 강제하지 않는다.

## 찾은 기존 자료

| 자료 | 역할 |
|---|---|
| [tests/tc JSON](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/README.md) | 현재 실행 스펙. 최상위 id, description, env, steps가 기준 |
| [StableNet 원본 TC 카탈로그](/Users/wm-it-25_0220/Work/github/chainbench/docs/dev/stablenet-post-v1.0.0-change-test-catalog.md) | 원래 TC-1-1-01, TC-4-6-01 등의 번호와 시나리오. 과거 bash metadata 보존 자료 |
| [WEMIX4 이관 대응표](/Users/wm-it-25_0220/Work/github/chainbench/docs/dev/wemix4-port-tracker.md) | TX/RPC/NODE/GOV/WBFT 번호와 JSON 또는 Go 테스트 대응 |
| [레거시 포팅 감사](/Users/wm-it-25_0220/Work/github/chainbench/docs/dev/legacy-port-audit/03-port-audit.md) | 원래 RT/TC 번호, 이전 파일, 이관 대상, 동등·통합·부분 판정 |
| [SPECS.md](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/SPECS.md) | 과거 Go 함수에서 DSL로 옮긴 이력. 최신 실행 목록·기능 지원의 확정 근거로 단독 사용하지 않음 |

위 문서들은 이관 이력과 명세 번호를 찾는 근거다. 과거 covered/ported 표시를 현재 바이너리의 PASS로 승계하지 않는다. 예를 들어 WEMIX4 tracker의 TX-018→tx-errors 표기는 실제 JSON id가 없고, 포팅 감사 문서도 이 문제를 지적한다. 이 자료에서 정확히 확인한 연결과 미확인 연결을 구분한다.

## ID를 구분하는 예

| 분석 별칭 | 현재 실행 ID | 직접 기록된 원래 명세 ID | 추가 문서 대응 |
|---|---|---|---|
| TC-028 | legacy-transfer | RT-A-2-01 | WEMIX4 tracker의 TX-006 |
| DOC-C-001 | 실행 파일이 아닌 원본 문서 행 | RT-A-2-01, TX-006 | 같은 기능에 대한 문서 명세 |
| TC-007 | fault-network-partition | description에서 해당 ID 형식 미확인 | 번호를 새로 원래 TC인 것처럼 만들지 않음 |

burn-cancel-refundable은 JSON id 하나가 TC-1-1-01과 TC-1-1-10 두 원래 명세를 명시적으로 참조한다. 반대로 nonce-ordering과 out-of-order-nonces-mine처럼 같은 목적의 구현이 나뉜 경우도 있어 이름만으로 통합하지 않는다.

## 앞으로 사용할 식별 필드

- original_spec_ids: 출처 체계와 함께 보존한 원래 명세 ID. 예: StableNet regression / RT-A-2-01.
- runtime_id: 실제 JSON 최상위 id 또는 Go 테스트 함수. 실행·결과 매칭에 사용.
- source_file: 현재 실행 스펙 파일 경로.
- analysis_id: 이번 분석에만 쓰는 별칭. 기존 문서 링크를 보존하기 위해 남김.
- mapping_relation: 직접 참조, 문서 대응, 통합·분리, 부분 커버, 미확인을 구분.

이번에는 스펙과 ID의 위치를 찾고 대응표를 추가했다. 원래 JSON id나 기존 분석 번호를 일괄 변경하지 않았다. 원본 업무용 TC 정의서 자체가 모두 복원되었다는 뜻은 아니다.

## JSON 189개 실행 ID 전수 대응

원래 명세란은 JSON description에 명시된 표현만 옮겼다. WEMIX4 연결은 tracker에서 현재 id가 정확히 등장하는 행만 추가했으며 동작 동등성을 새로 검증한 것은 아니다. 문서 74행까지 포함한 263개 대응은 existing-id-crosswalk.json에 있다.

| 분석 별칭 | 실행 ID | 원래 명세 표현 | WEMIX4 문서 연결 |
|---|---|---|---|
| TC-001 | [basic-consensus](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/basic/01-basic-consensus.json) | 직접 참조 미확인 | — |
| TC-002 | [basic-peers](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/basic/02-basic-peers.json) | 직접 참조 미확인 | — |
| TC-003 | [basic-rpc-health](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/basic/03-basic-rpc-health.json) | 직접 참조 미확인 | — |
| TC-004 | [basic-sync](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/basic/04-basic-sync.json) | 직접 참조 미확인 | — |
| TC-005 | [basic-tx-send](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/basic/05-basic-tx-send.json) | 직접 참조 미확인 | — |
| TC-006 | [basic-txpool-propagation](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/basic/06-basic-txpool-propagation.json) | 직접 참조 미확인 | — |
| TC-007 | [fault-network-partition](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/fault/01-fault-network-partition.json) | 직접 참조 미확인 | — |
| TC-008 | [fault-node-crash](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/fault/02-fault-node-crash.json) | 직접 참조 미확인 | — |
| TC-009 | [fault-node-recover](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/fault/03-fault-node-recover.json) | 직접 참조 미확인 | — |
| TC-010 | [fault-p2p-topology](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/fault/04-fault-p2p-topology.json) | 직접 참조 미확인 | — |
| TC-011 | [fault-two-down](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/fault/05-fault-two-down.json) | 직접 참조 미확인 | — |
| TC-012 | [fault-txpool-leader-change](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/fault/06-fault-txpool-leader-change.json) | 직접 참조 미확인 | — |
| TC-013 | [legacy-gasprice-below-min-rejected](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/12-legacy-gasprice-below-min-rejected.json) | TC-1-3-04 | — |
| TC-014 | [accesslist-gasprice-below-min-rejected](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/13-accesslist-gasprice-below-min-rejected.json) | TC-1-3-05 | — |
| TC-015 | [feecap-below-min-rejected](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/14-feecap-below-min-rejected.json) | TC-1-3-06 | — |
| TC-016 | [effective-gas-price-regular-bp-en](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json) | TC-4-6-02 | — |
| TC-017 | [signature-compat-across-swap](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json) | TC-3-1-04 | — |
| TC-018 | [genesis-mismatch-refuses-to-start](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json) | TC-4-1-03 | — |
| TC-019 | [genesis-block-hash-consistent](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/04-genesis-block-hash-consistent.json) | TC-5-3-01 | — |
| TC-020 | [block-transactions-field](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/api/01-block-transactions-field.json) | RT-G-1-01 | RPC-002 |
| TC-021 | [block-by-hash-consistency](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/api/02-block-by-hash-consistency.json) | RT-G-1-02 | RPC-002 |
| TC-022 | [transaction-by-hash-fields](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/api/03-transaction-by-hash-fields.json) | RT-G-1-03 | — |
| TC-023 | [transaction-receipt-fields](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/api/04-transaction-receipt-fields.json) | RT-G-1-04 | RPC-007 |
| TC-024 | [transaction-count-increments](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/api/05-transaction-count-increments.json) | RT-G-1-05 | RPC-015 |
| TC-025 | [txpool-status](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/api/18-txpool-status.json) | RT-G-4-02 | RPC-018 |
| TC-026 | [txpool-content-well-formed](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/api/19-txpool-content-well-formed.json) | RT-G-4-03 | RPC-018 |
| TC-027 | [fee-delegate-sign-rpc-present](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/api/21-fee-delegate-sign-rpc-present.json) | RT-G-5-01 | — |
| TC-028 | [legacy-transfer](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/ethereum/08-legacy-transfer.json) | RT-A-2-01 | TX-006 |
| TC-029 | [dynamic-fee-tx](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/ethereum/09-dynamic-fee-tx.json) | RT-A-2-02 | TX-003 |
| TC-030 | [access-list-tx](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/ethereum/10-access-list-tx.json) | RT-A-2-03 | TX-007 |
| TC-031 | [nonce-ordering](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/ethereum/11-nonce-ordering.json) | RT-A-2-04 | — |
| TC-032 | [out-of-order-nonces-mine](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json) | 직접 참조 미확인 | TX-010 |
| TC-033 | [dynamic-fee-below-basefee-rejected](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/ethereum/12-dynamic-fee-below-basefee-rejected.json) | RT-A-2-05a | TX-002 |
| TC-034 | [insufficient-funds-rejected](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/ethereum/14-insufficient-funds-rejected.json) | RT-A-2-06 | TX-011 |
| TC-035 | [gas-limit-exceeds-block-rejected](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/ethereum/15-gas-limit-exceeds-block-rejected.json) | RT-A-2-07 | TX-012 |
| TC-036 | [effective-gas-price](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/ethereum/16-effective-gas-price.json) | RT-A-2-08 | TX-001 |
| TC-037 | [replacement-tx](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/ethereum/17-replacement-tx.json) | RT-A-2-09 | — |
| TC-038 | [same-nonce-replacement](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/ethereum/17b-same-nonce-replacement.json) | 직접 참조 미확인 | TX-013 |
| TC-039 | [contract-roundtrip](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/ethereum/19-contract-roundtrip.json) | RT-A-3-01 | TX-005 |
| TC-040 | [eth-call-revert-returns-error](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json) | RT-A-3-05 | TX-017 |
| TC-041 | [revert-tx-status-zero](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/ethereum/24-revert-tx-status-zero.json) | RT-A-3-06 | — |
| TC-042 | [out-of-gas-consumes-all](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json) | RT-A-3-07 | — |
| TC-043 | [value-transfer](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/ethereum/27b-value-transfer.json) | RT-A-4-03 | TX-001 |
| TC-044 | [contract-event-emitted](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/ethereum/36-contract-event-emitted.json) | 직접 참조 미확인 | — |
| TC-045 | [fee-delegated-transfer](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json) | RT-D-01 | TX-004 |
| TC-046 | [fd-sender-sig-invalid-rejected](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json) | RT-D-03 | — |
| TC-047 | [fd-feepayer-sig-invalid-rejected](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json) | RT-D-04 | — |
| TC-048 | [feepayer-insufficient-rejected](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json) | RT-D-05 | — |
| TC-049 | [fee-delegated-sender-sig-invalid-rejected](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json) | 직접 참조 미확인 | TX-014 |
| TC-050 | [fee-delegated-feepayer-sig-invalid-rejected](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json) | 직접 참조 미확인 | TX-015 |
| TC-051 | [fee-delegated-unfunded-feepayer-rejected](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json) | 직접 참조 미확인 | TX-016 |
| TC-052 | [wemix-chain-up](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-wemix/chain-up/01-wemix-chain-up.json) | 직접 참조 미확인 | — |
| TC-053 | [wemix-chain-up-15](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-wemix/chain-up/02-wemix-chain-up-15.json) | 직접 참조 미확인 | — |
| TC-054 | [wemix-node-crash](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-wemix/fault/01-wemix-node-crash.json) | 직접 참조 미확인 | — |
| TC-055 | [wemix-brioche-block-reward](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-wemix/rpc/01-wemix-brioche-block-reward.json) | 직접 참조 미확인 | — |
| TC-056 | [wemix-tx-and-contract](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-wemix/tx/01-wemix-tx-and-contract.json) | 직접 참조 미확인 | — |
| TC-057 | [stress-block-time](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/stress/01-stress-block-time.json) | 직접 참조 미확인 | — |
| TC-058 | [stress-tx-flood](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/stress/02-stress-tx-flood.json) | 직접 참조 미확인 | — |
| TC-059 | [gas-price-positive](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/api/07-gas-price-positive.json) | RT-G-2-01 | RPC-016 |
| TC-060 | [fee-history-well-formed](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/api/09-fee-history-well-formed.json) | RT-G-2-03 | RPC-017 |
| TC-061 | [admin-peers-populated](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/api/20-admin-peers-populated.json) | RT-A-1-05 | — |
| TC-062 | [chain-not-syncing](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/api/24-chain-not-syncing.json) | 직접 참조 미확인 | — |
| TC-063 | [stablenet-chain-up](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/ethereum/01-stablenet-chain-up.json) | 직접 참조 미확인 | — |
| TC-064 | [estimate-gas](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/ethereum/22-estimate-gas.json) | RT-A-3-04 | — |
| TC-065 | [genesis-balance](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/ethereum/27-genesis-balance.json) | RT-A-4-02 | RPC-012 |
| TC-066 | [logs-query-well-formed](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/ethereum/29-logs-query-well-formed.json) | RT-A-4-04 | RPC-014 |
| TC-067 | [chain-id](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/ethereum/30-chain-id.json) | RT-A-1-01 | RPC-013 |
| TC-068 | [ws-subscribe-new-heads](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/ethereum/31-ws-subscribe-new-heads.json) | RT-A-4-06 | — |
| TC-069 | [ws-subscribe-logs](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/ethereum/32-ws-subscribe-logs.json) | RT-A-4-07 | — |
| TC-070 | [stablenet-chain-up-15](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/ethereum/33-stablenet-chain-up-15.json) | 직접 참조 미확인 | — |
| TC-071 | [stablenet-proxied-pn-routing](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/topology/01-proxied-pn-routing.json) | 직접 참조 미확인 | — |
| TC-072 | [stablenet-negative-tx-revert](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/tx/01-negative-tx-revert.json) | 직접 참조 미확인 | — |
| TC-073 | [wbft-chain-up](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-wbft/chain-up/01-wbft-chain-up.json) | 직접 참조 미확인 | — |
| TC-074 | [wbft-chain-up-15](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-wbft/chain-up/02-wbft-chain-up-15.json) | 직접 참조 미확인 | — |
| TC-075 | [e1-mixed-producers](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-wbft/consensus/01-e1-mixed-producers.json) | 직접 참조 미확인 | — |
| TC-076 | [wbft-node-crash](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-wbft/fault/01-wbft-node-crash.json) | 직접 참조 미확인 | — |
| TC-077 | [wbft-tx-and-contract](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-wbft/tx/01-wbft-tx-and-contract.json) | 직접 참조 미확인 | — |
| TC-078 | [wemix-wbft-handoff](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-wemix/handoff/01-wemix-wbft-handoff.json) | 직접 참조 미확인 | — |
| TC-079 | [remote-rpc-health](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/remote/01-remote-rpc-health.json) | 직접 참조 미확인 | — |
| TC-080 | [remote-chain-info](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/remote/02-remote-chain-info.json) | 직접 참조 미확인 | — |
| TC-081 | [remote-balance-check](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/remote/03-remote-balance-check.json) | 직접 참조 미확인 | — |
| TC-082 | [sample-minimal-value-transfer](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/samples/01-sample-minimal.json) | 직접 참조 미확인 | — |
| TC-083 | [sample-lifecycle-node-restart](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/samples/02-sample-lifecycle.json) | 직접 참조 미확인 | — |
| TC-084 | [stablenet-derived-vocabulary](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/vocabulary/01-derived-address-and-checksum.json) | 직접 참조 미확인 | — |
| TC-085 | [stablenet-faucet-funds](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/vocabulary/02-faucet-funds-account.json) | 직접 참조 미확인 | — |
| TC-086 | [stablenet-register-contract](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/vocabulary/03-register-contract.json) | 직접 참조 미확인 | — |
| TC-087 | [basic-wbft-consensus](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/basic/07-basic-wbft-consensus.json) | 직접 참조 미확인 | — |
| TC-088 | [stablenet-delayed-fork](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json) | 직접 참조 미확인 | — |
| TC-089 | [govminter-v2-code](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json) | TC-5-2-05 | — |
| TC-090 | [burn-cancel-refundable](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json) | TC-1-1-01, TC-1-1-10 | — |
| TC-091 | [burn-reject-refundable](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json) | TC-1-1-02 | — |
| TC-092 | [burn-expire-refundable](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json) | TC-1-1-03 | — |
| TC-093 | [burn-execute-no-refundable](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json) | TC-1-1-04 | — |
| TC-094 | [claim-burn-refund-succeeds](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json) | TC-1-1-05, TC-1-1-09 | — |
| TC-095 | [claim-zero-refund-reverts](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json) | TC-1-1-06 | — |
| TC-096 | [claim-burn-refund-double-reverts](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json) | TC-1-1-07 | — |
| TC-097 | [prealloc-preserved-across-boho](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json) | TC-1-1-11/12 | — |
| TC-098 | [boho-chain-config-active](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/15-boho-chain-config-active.json) | TC-4-1-01 | — |
| TC-099 | [anzeon-active-before-boho](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/16-anzeon-active-before-boho.json) | TC-4-1-02 | — |
| TC-100 | [estimategas-authorizationlist-cost](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json) | TC-4-2-01/03 | — |
| TC-101 | [upgrade-registry-order](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json) | TC-5-2-01/02 | — |
| TC-102 | [v1-params-init-storage](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json) | TC-5-2-04 | — |
| TC-103 | [burn-refund-events](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json) | 직접 참조 미확인 | — |
| TC-104 | [effective-gas-price-authorized-bp-en](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json) | TC-4-6-01 | — |
| TC-105 | [effective-gas-price-regular](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json) | TC-4-6-02 | — |
| TC-106 | [auth-tx-event-last-bp-en](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json) | TC-4-6-04 | — |
| TC-107 | [authorized-extra-bit-synced](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/01-authorized-extra-bit-synced.json) | TC-4-5-01, TC-4-5-02 | — |
| TC-108 | [blacklisted-extra-bit-synced](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/01b-blacklisted-extra-bit-synced.json) | 직접 참조 미확인 | — |
| TC-109 | [stablenet-account-extra](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json) | 직접 참조 미확인 | — |
| TC-110 | [extra-union-merge](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json) | TC-4-5-05/06 | — |
| TC-111 | [dual-status-extra](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json) | TC-4-5-07 | — |
| TC-112 | [extra-balance-preserved](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/05-extra-balance-preserved.json) | TC-4-5-08 | — |
| TC-113 | [invalid-extra-reject](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json) | TC-4-5-09 | — |
| TC-114 | [extra-state-across-delayed-boho](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json) | TC-4-5-10/11 | — |
| TC-115 | [unsupported-system-contract-version](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json) | TC-5-2-06 | — |
| TC-116 | [authorized-accounts-no-space](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/01-authorized-accounts-no-space.json) | TC-4-3-01 | — |
| TC-117 | [authorized-accounts-space](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/02-authorized-accounts-space.json) | TC-4-3-02 | — |
| TC-118 | [authorized-accounts-trim](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/03-authorized-accounts-trim.json) | TC-4-3-03 | — |
| TC-119 | [authorized-accounts-empty-item](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/04-authorized-accounts-empty-item.json) | TC-4-3-04 | — |
| TC-120 | [authorized-accounts-single](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/05-authorized-accounts-single.json) | TC-4-3-05 | — |
| TC-121 | [authorized-accounts-empty](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/06-authorized-accounts-empty.json) | TC-4-3-06 | — |
| TC-122 | [regular-account-gastip-forced](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json) | RT-C-01 | — |
| TC-123 | [authorized-account-gastip-free](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json) | RT-C-02 | — |
| TC-124 | [anzeon-basefee-increase](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json) | 직접 참조 미확인 | — |
| TC-125 | [anzeon-basefee-stable](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json) | 직접 참조 미확인 | — |
| TC-126 | [anzeon-basefee-decrease](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json) | 직접 참조 미확인 | — |
| TC-127 | [basefee-minimum](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/anzeon/06-basefee-minimum.json) | RT-C-06 | — |
| TC-128 | [basefee-maximum](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/anzeon/07-basefee-maximum.json) | RT-C-07 | — |
| TC-129 | [feecap-above-min-accepted](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json) | 직접 참조 미확인 | — |
| TC-130 | [feecap-exact-min-accepted](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json) | 직접 참조 미확인 | — |
| TC-131 | [gaslimit-exceeded-rejected](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json) | 직접 참조 미확인 | — |
| TC-132 | [system-contracts-deployed](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/api/06-system-contracts-deployed.json) | RT-G-1-06 | — |
| TC-133 | [gas-price-equals-basefee-plus-tip](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json) | 직접 참조 미확인 | — |
| TC-134 | [max-priority-fee-equals-gastip](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/api/08-max-priority-fee-equals-gastip.json) | RT-G-2-02 | — |
| TC-135 | [estimate-gas-token-transfer](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/api/10-estimate-gas-token-transfer.json) | RT-G-2-04 | — |
| TC-136 | [node-address-returned](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/api/11-node-address-returned.json) | RT-G-3-01 | RPC-010 |
| TC-137 | [validator-set-nonempty](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/api/12-validator-set-nonempty.json) | RT-G-3-02 | — |
| TC-138 | [validator-set-count](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/api/12b-validator-set-count.json) | 직접 참조 미확인 | RPC-003 |
| TC-139 | [commit-signers-quorum](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/api/13-commit-signers-quorum.json) | RT-G-3-03 | RPC-004 |
| TC-140 | [wbft-extra-info-fields](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/api/14-wbft-extra-info-fields.json) | RT-G-3-04 | RPC-005 |
| TC-141 | [istanbul-status-fields](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/api/15-istanbul-status-fields.json) | RT-G-3-05 | RPC-006 |
| TC-142 | [is-validator-flags](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/api/16-is-validator-flags.json) | RT-G-3-06 | RPC-011 |
| TC-143 | [token-total-supply-readable](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/api/22-token-total-supply-readable.json) | RT-G-5-02 | — |
| TC-144 | [token-approve-sets-allowance](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/api/23-token-approve-sets-allowance.json) | RT-G-5-03 | — |
| TC-145 | [sender-blacklisted-rejected](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json) | RT-E-01 | — |
| TC-146 | [recipient-blacklisted-rejected](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json) | RT-E-02 | — |
| TC-147 | [feepayer-blacklisted-rejected](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json) | RT-E-03 | — |
| TC-148 | [address-unblacklisted-event](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json) | RT-E-04 | — |
| TC-149 | [zero-address-transfer-rejected](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json) | RT-E-05 | — |
| TC-150 | [precompile-transfer-rejected](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json) | RT-E-06 | — |
| TC-151 | [account-blacklist-readable](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/07-account-blacklist-readable.json) | RT-E-07 | — |
| TC-152 | [account-authorization-readable](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/08-account-authorization-readable.json) | RT-E-08 | — |
| TC-153 | [authorized-tx-executed-event](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json) | RT-E-09 | — |
| TC-154 | [set-code-delegation](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/ethereum/18-set-code-delegation.json) | RT-A-2-10 | TX-008 |
| TC-155 | [native-coin-adapter-code](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/system-contracts/01-native-coin-adapter-code.json) | RT-F-1-01 | — |
| TC-156 | [token-transfer-emits-event](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json) | 직접 참조 미확인 | — |
| TC-157 | [token-balance-readable](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/system-contracts/02-token-balance-readable.json) | RT-F-1-02 | — |
| TC-158 | [token-transfer-from-moves-balance](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json) | 직접 참조 미확인 | — |
| TC-159 | [mint-transfer-event](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/system-contracts/04-mint-transfer-event.json) | RT-F-1-04 | — |
| TC-160 | [burn-transfer-event](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/system-contracts/05-burn-transfer-event.json) | RT-F-1-05 | — |
| TC-161 | [mint-proposal-executes](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/system-contracts/06-mint-proposal-executes.json) | RT-F-2-01 | — |
| TC-162 | [burn-proposal-executes](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/system-contracts/07-burn-proposal-executes.json) | RT-F-2-02 | — |
| TC-163 | [quorum-deficient-stays-voting](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json) | RT-F-2-03 | — |
| TC-164 | [validator-metadata-readable](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/system-contracts/09-validator-metadata-readable.json) | RT-F-3-04 | — |
| TC-165 | [gastip-governance-updates-header](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json) | RT-B-06 | — |
| TC-166 | [proposal-expiry-transitions](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json) | RT-F-3-06 | — |
| TC-167 | [configure-minter-proposal-executes](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json) | RT-F-4-01 | — |
| TC-168 | [remove-minter-executes](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/system-contracts/13-remove-minter-executes.json) | RT-F-4-02 | — |
| TC-169 | [masterminter-member-add-remove](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json) | RT-F-4-03 | — |
| TC-170 | [non-member-configure-minter-rejected](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json) | RT-F-4-04 | — |
| TC-171 | [blacklist-proposal-executes](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json) | RT-F-5-01 | — |
| TC-172 | [authorize-proposal-executes](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json) | RT-F-5-03 | — |
| TC-173 | [unauthorize-proposal-executes](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json) | 직접 참조 미확인 | — |
| TC-174 | [direct-blacklist-call-rejected](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json) | RT-F-5-05 | — |
| TC-175 | [authorized-account-added-event](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/system-contracts/23-authorized-account-added-event.json) | RT-F-5-08 | — |
| TC-176 | [token-metadata](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/system-contracts/25-token-metadata.json) | 직접 참조 미확인 | — |
| TC-177 | [minter-status-readable](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/system-contracts/28-minter-status-readable.json) | 직접 참조 미확인 | — |
| TC-178 | [block-period-one-second](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/wbft/01-block-period-one-second.json) | RT-B-01 | WBFT-002 |
| TC-179 | [wbft-seals-quorum](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/wbft/02-wbft-seals-quorum.json) | RT-B-02 | WBFT-001 |
| TC-180 | [epoch-transition-carries-epoch-info](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json) | RT-B-03 | RPC-022, WBFT-005 |
| TC-181 | [validator-add-member-executes](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/wbft/04-validator-add-member-executes.json) | RT-B-04 | — |
| TC-182 | [validator-add-member-epoch-activates](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json) | RT-B-04 | — |
| TC-183 | [validator-remove-member-executes](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/wbft/05-validator-remove-member-executes.json) | RT-B-05 | — |
| TC-184 | [prev-seals-quorum](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/wbft/11-prev-seals-quorum.json) | RT-B-11 | WBFT-009 |
| TC-185 | [randao-and-mixdigest-present](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/wbft/13-randao-and-mixdigest-present.json) | 직접 참조 미확인 | WBFT-010 |
| TC-186 | [stablenet-gastip-field](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-stablenet/regression/wbft/14-stablenet-gastip-field.json) | 직접 참조 미확인 | — |
| TC-187 | [secp256r1-precompile-valid](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-wbft/accounts/01-secp256r1-precompile-valid.json) | 직접 참조 미확인 | TX-009 |
| TC-188 | [secp256r1-precompile-invalid](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-wbft/accounts/02-secp256r1-precompile-invalid.json) | 직접 참조 미확인 | TX-019 |
| TC-189 | [secp256r1-precompile-short-input](/Users/wm-it-25_0220/Work/github/chainbench/tests/tc/go-wbft/accounts/03-secp256r1-precompile-short-input.json) | 직접 참조 미확인 | TX-020 |
