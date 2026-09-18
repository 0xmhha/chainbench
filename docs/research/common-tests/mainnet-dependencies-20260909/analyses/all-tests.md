# 전체 테스트 목록

> ID 표기는 Confluence 원래 명세 ID를 우선한다. 실행 ID는 별도 표기한다. ID 대응은 전체 명세 충족이나 PASS를 뜻하지 않는다. [ID 대응표](existing-tc-specs.md)

문서 45 + 29행, JSON 189개 = 263개 원본 항목이다. 중복 시나리오도 원본 ID를 보존했다. JSON 파일명/설명보다 실제 do/expect를 기준으로 분류했다. 189개 JSON 및 두 문서는 이전 분석 스냅샷과 바이트 동일하다. 이전 PR196 우선순위는 본 작업에 사용하지 않는다.

`exact-config`: 시나리오의 공통 assertion을 유지하고 기존 환경 adapter에 profile 값을 제공하는 후보. 그대로 실행 가능하다는 뜻은 아니다. `adapter-required`: DSL 명세 구현 또는 합의·수수료·서명 등 별도 동작 필요. `chain-specific`: 원문 목적을 세 체인 공통으로 충족할 수 없음. `unknown`: 외부 서비스 등 확인되지 않은 선행조건. 명세에 이름만 있는 shell script는 실행 파일이 확보된 것으로 세지 않았다.


원본 테스트 입력은 이전 보존본과 동일하지만 하네스 분석은 현재 dirty working tree를 보존한 helper 기준이다. 이전 helper와 동일하다는 뜻은 아니다. RT-C-07 (분석 TC-128)은 WBFT에 고정 baseFee 최대 clamp가 없어 공통에서 제외했다.


| ID | 테스트 | 공통성 | go-wemix | family |
|---|---|---|---|---|
| RT-A-2-01 / TX-006 | [Legacy Tx (Type 0x0) 전송 및 실행](../sources/common_test_scenarios.md) | adapter-required | port-required | transaction-types |
| RT-A-2-02 / TX-003 | [Dynamic Fee Tx (Type 0x2, EIP-1559)](../sources/common_test_scenarios.md) | adapter-required | port-required | transaction-types |
| RT-A-2-03 / TX-007 | [Access List Tx (Type 0x1, EIP-2930)](../sources/common_test_scenarios.md) | adapter-required | port-required | transaction-types |
| RT-D-01 / TX-004 | [수수료 대납 Tx (Type 0x16) 정상 처리](../sources/common_test_scenarios.md) | adapter-required | port-required | fee-delegation |
| RT-D-03 / TX-014 | [대납 Tx Sender 서명 변조 거부](../sources/common_test_scenarios.md) | adapter-required | port-required | fee-delegation |
| RT-D-04 / TX-015 | [대납 Tx FeePayer 서명 변조 거부](../sources/common_test_scenarios.md) | adapter-required | port-required | fee-delegation |
| RT-D-05 / TX-016 | [FeePayer 잔액 부족시 실행 실패](../sources/common_test_scenarios.md) | adapter-required | port-required | fee-delegation |
| RT-A-2-04 / TX-010 | [Nonce 순서 보장 (Nonce Ordering)](../sources/common_test_scenarios.md) | adapter-required | port-required | nonce-replacement |
| RT-A-2-05a | [TipCap 미달 (Underpriced) 거부](../sources/common_test_scenarios.md) | adapter-required | conditional | fee-policy |
| RT-A-2-06 / TX-011 | [잔액 부족 (Insufficient Funds) 거부](../sources/common_test_scenarios.md) | adapter-required | port-required | invalid-transaction |
| RT-A-2-07 / TX-012 | [Gas Limit 초과 거부](../sources/common_test_scenarios.md) | adapter-required | port-required | invalid-transaction |
| RT-A-2-08 | [Effective GasPrice 노드 간 일관성](../sources/common_test_scenarios.md) | adapter-required | port-required | fee-observation |
| RT-A-2-09 / TX-013 | [Queued/Pending 트랜잭션 교체 (Replacement Tx)](../sources/common_test_scenarios.md) | adapter-required | port-required | nonce-replacement |
| RT-A-3-01 / TX-005 | [스마트 컨트랙트 배포](../sources/common_test_scenarios.md) | adapter-required | port-required | evm-contract |
| RT-A-3-02 | [컨트랙트 상태 변경 함수 호출](../sources/common_test_scenarios.md) | adapter-required | port-required | evm-contract |
| RT-A-3-03 | [`eth_call` View/Pure 함수 조회](../sources/common_test_scenarios.md) | adapter-required | port-required | evm-contract |
| RT-A-3-04 | [`eth_estimateGas` 가스 추정](../sources/common_test_scenarios.md) | adapter-required | port-required | evm-contract |
| RT-A-3-05 | [`eth_call` 실행 중 Revert 메세지 반환](../sources/common_test_scenarios.md) | adapter-required | port-required | evm-contract |
| RT-A-3-06 / TX-017 | [Revert 트랜잭션 (상태 롤백 & 가스 환불)](../sources/common_test_scenarios.md) | adapter-required | port-required | evm-contract |
| RT-A-3-07 / TX-018 | [Out-of-Gas 트랜잭션 (가스 Limit 전액 소모)](../sources/common_test_scenarios.md) | adapter-required | port-required | evm-contract |
| RT-A-4-01 / RPC-001 | [`eth_blockNumber`](../sources/common_test_scenarios.md) | adapter-required | port-required | rpc-basics |
| RT-A-4-02 / RPC-012 | [`eth_getBalance`](../sources/common_test_scenarios.md) | adapter-required | port-required | rpc-basics |
| RT-A-4-03 | [`eth_sendRawTransaction`](../sources/common_test_scenarios.md) | adapter-required | port-required | rpc-basics |
| RT-A-4-04 / RPC-014 | [`eth_getLogs`](../sources/common_test_scenarios.md) | adapter-required | port-required | logs-subscription |
| RT-A-4-05 / RPC-013 | [`eth_chainId`](../sources/common_test_scenarios.md) | adapter-required | port-required | rpc-basics |
| RT-A-4-06 / RPC-020 | [WebSocket `eth_subscribe("newHeads")`](../sources/common_test_scenarios.md) | adapter-required | port-required | logs-subscription |
| RT-A-4-07 / RPC-021 | [WebSocket `eth_subscribe("logs")`](../sources/common_test_scenarios.md) | adapter-required | port-required | logs-subscription |
| RT-G-1-01 / RPC-002 | [`eth_getBlockByNumber`](../sources/common_test_scenarios.md) | adapter-required | port-required | rpc-basics |
| RT-G-1-02 / RPC-002 | [`eth_getBlockByHash`](../sources/common_test_scenarios.md) | adapter-required | port-required | rpc-basics |
| RT-G-1-03 | [`eth_getTransactionByHash`](../sources/common_test_scenarios.md) | adapter-required | port-required | rpc-basics |
| RT-G-1-04 / RPC-007 | [`eth_getTransactionReceipt`](../sources/common_test_scenarios.md) | adapter-required | port-required | rpc-basics |
| RT-G-1-05 / RPC-015 | [`eth_getTransactionCount`](../sources/common_test_scenarios.md) | adapter-required | port-required | rpc-basics |
| RT-G-2-01 / RPC-016 | [`eth_gasPrice`](../sources/common_test_scenarios.md) | adapter-required | port-required | fee-observation |
| RT-G-2-02 | [`eth_maxPriorityFeePerGas`](../sources/common_test_scenarios.md) | adapter-required | port-required | fee-observation |
| RT-G-2-03 / RPC-017 | [`eth_feeHistory`](../sources/common_test_scenarios.md) | adapter-required | port-required | fee-observation |
| RT-G-4-02 / RPC-018 | [`txpool_status`](../sources/common_test_scenarios.md) | adapter-required | port-required | txpool |
| RT-G-4-03 | [`txpool_content`](../sources/common_test_scenarios.md) | adapter-required | port-required | txpool |
| RT-G-5-01 | [FeePayer 서명 RPC 헬스체크 (`eth_signRawFeeDelegateTransaction`)](../sources/common_test_scenarios.md) | adapter-required | port-required | fee-delegation |
| RT-A-1-01 | [제네시스 초기화 (Genesis Initialization)](../sources/common_test_scenarios.md) | adapter-required | port-required | genesis-upgrade |
| RT-A-1-02 / NODE-003 | [Full Sync 동기화](../sources/common_test_scenarios.md) | adapter-required | port-required | sync-lifecycle |
| RT-A-1-03 / NODE-004 | [Snap Sync 동기화](../sources/common_test_scenarios.md) | adapter-required | conditional | sync-lifecycle |
| RT-A-1-04 / NODE-005 | [노드 재기동 후 동기화 유지](../sources/common_test_scenarios.md) | adapter-required | port-required | sync-lifecycle |
| RT-A-1-05 | [P2P Bootnode 피어 연결](../sources/common_test_scenarios.md) | adapter-required | port-required | sync-lifecycle |
| RT-A-1-06 | [Block Downloader 경로 동기화](../sources/common_test_scenarios.md) | adapter-required | port-required | sync-lifecycle |
| RT-A-1-07 | [Block Fetcher 경로 동기화](../sources/common_test_scenarios.md) | adapter-required | port-required | sync-lifecycle |
| RPC-008 | [`wemix_getBriocheBlockReward` 하위 호환성](../sources/dual_chain_test_scenarios.md) | chain-specific | conditional | reward-compatibility |
| RT-B-01 / WBFT-002 | [블록 생산 주기 1초](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| RT-B-02 / WBFT-001 | [블록 Finalize 및 CommittedSeal 존재](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| RT-B-03 / WBFT-005 | [에폭 전환 시 검증자 집합 갱신](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| RT-B-06 | [블록 헤더 WBFTExtra 필드 반영](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| RT-B-09 / WBFT-003 | [View Change / 라운드 체인지](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| RT-B-10 / WBFT-003 | [라운드 체인지 후 블록 연결](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| RT-B-06 / WBFT-006 | [RoundRobin Proposer 정책](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| RT-B-11 / WBFT-009 | [PrevCommittedSeal 수집](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| RT-B-12 / WBFT-009 | [PrevPreparedSeal 수집](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| WBFT-010 | [RandaoReveal / MixDigest](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| RT-B-08 | [쿼럼 미달 블록 수락 거부](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| WBFT-007 | [1/3 미만 장애 시 합의 지속](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| WBFT-008 | [1/3 이상 장애 시 합의 중단](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| WBFT-011 | [쿼럼 계산 — validator 3개 (전원 필요)](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| WBFT-012 | [쿼럼 계산 — validator 6개, 1개 장애](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| WBFT-013 | [쿼럼 계산 — validator 6개, 2개 장애](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| RT-G-3-01 / RPC-010 | [`istanbul_nodeAddress`](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| RT-G-3-02 / RPC-003 | [`istanbul_getValidators`](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| RT-G-3-03 / RPC-004 | [`istanbul_getCommitSignersFromBlock`](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| RT-G-3-04 / RPC-005 | [`istanbul_getWbftExtraInfo`](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| RT-G-3-05 / RPC-006 | [`istanbul_status`](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| RT-G-3-06 / RPC-011 | [`istanbul_isValidator`](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| RT-A-2-10 / TX-008 | [SetCode Tx (Type 0x4, EIP-7702)](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | modern-evm-specific |
| TC-4-2-01 / TC-4-2-03 | [EIP-7702 AuthorizationList estimateGas 비용](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | modern-evm-specific |
| TC-4-2-02 | [EIP-7702 estimateGas baseline](../sources/dual_chain_test_scenarios.md) | adapter-required | conditional | evm-contract |
| TC-1-2-01 / TC-1-2-02 / TC-1-2-03 / TC-1-2-04 / TC-1-2-05 / TC-1-2-06 / TX-009 | [secp256r1 프리컴파일 — 유효 서명](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | modern-evm-specific |
| TC-1-2-01 / TC-1-2-02 / TC-1-2-03 / TC-1-2-04 / TC-1-2-05 / TC-1-2-06 / TX-019 | [secp256r1 프리컴파일 — 무효 서명](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | modern-evm-specific |
| TC-1-2-01 / TC-1-2-02 / TC-1-2-03 / TC-1-2-04 / TC-1-2-05 / TC-1-2-06 / TX-020 | [secp256r1 프리컴파일 — 잘못된 입력 길이](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | modern-evm-specific |
| basic-consensus [명세 ID 미확인] | [basic-consensus](../sources/chainbench/tests/tc/basic/01-basic-consensus.json) | adapter-required | port-required | sync-lifecycle |
| basic-peers [명세 ID 미확인] | [basic-peers](../sources/chainbench/tests/tc/basic/02-basic-peers.json) | adapter-required | port-required | sync-lifecycle |
| basic-rpc-health [명세 ID 미확인] | [basic-rpc-health](../sources/chainbench/tests/tc/basic/03-basic-rpc-health.json) | exact-config | port-required | rpc-basics |
| basic-sync [명세 ID 미확인] | [basic-sync](../sources/chainbench/tests/tc/basic/04-basic-sync.json) | adapter-required | port-required | sync-lifecycle |
| basic-tx-send [명세 ID 미확인] | [basic-tx-send](../sources/chainbench/tests/tc/basic/05-basic-tx-send.json) | exact-config | port-required | rpc-basics |
| basic-txpool-propagation [명세 ID 미확인] | [basic-txpool-propagation](../sources/chainbench/tests/tc/basic/06-basic-txpool-propagation.json) | exact-config | port-required | txpool |
| fault-network-partition [명세 ID 미확인] | [fault-network-partition](../sources/chainbench/tests/tc/fault/01-fault-network-partition.json) | adapter-required | port-required | fault-topology |
| fault-node-crash [명세 ID 미확인] | [fault-node-crash](../sources/chainbench/tests/tc/fault/02-fault-node-crash.json) | adapter-required | port-required | fault-topology |
| fault-node-recover [명세 ID 미확인] | [fault-node-recover](../sources/chainbench/tests/tc/fault/03-fault-node-recover.json) | adapter-required | port-required | fault-topology |
| fault-p2p-topology [명세 ID 미확인] | [fault-p2p-topology](../sources/chainbench/tests/tc/fault/04-fault-p2p-topology.json) | adapter-required | port-required | fault-topology |
| fault-two-down [명세 ID 미확인] | [fault-two-down](../sources/chainbench/tests/tc/fault/05-fault-two-down.json) | adapter-required | port-required | fault-topology |
| fault-txpool-leader-change [명세 ID 미확인] | [fault-txpool-leader-change](../sources/chainbench/tests/tc/fault/06-fault-txpool-leader-change.json) | adapter-required | port-required | fault-topology |
| TC-1-3-04 <br> 실행: legacy-gasprice-below-min-rejected | [legacy-gasprice-below-min-rejected](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/12-legacy-gasprice-below-min-rejected.json) | adapter-required | port-required | fee-policy |
| TC-1-3-05 <br> 실행: accesslist-gasprice-below-min-rejected | [accesslist-gasprice-below-min-rejected](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/13-accesslist-gasprice-below-min-rejected.json) | adapter-required | port-required | fee-policy |
| TC-1-3-06 <br> 실행: feecap-below-min-rejected | [feecap-below-min-rejected](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/14-feecap-below-min-rejected.json) | adapter-required | port-required | fee-policy |
| TC-4-6-02 <br> 실행: effective-gas-price-regular-bp-en | [effective-gas-price-regular-bp-en](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json) | exact-config | port-required | fee-observation |
| TC-3-1-04 <br> 실행: signature-compat-across-swap | [signature-compat-across-swap](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json) | adapter-required | port-required | genesis-upgrade |
| TC-4-1-03 <br> 실행: genesis-mismatch-refuses-to-start | [genesis-mismatch-refuses-to-start](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json) | adapter-required | port-required | genesis-upgrade |
| TC-5-3-01 <br> 실행: genesis-block-hash-consistent | [genesis-block-hash-consistent](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/04-genesis-block-hash-consistent.json) | adapter-required | port-required | genesis-upgrade |
| RT-G-1-01 <br> 실행: block-transactions-field | [block-transactions-field](../sources/chainbench/tests/tc/go-stablenet/regression/api/01-block-transactions-field.json) | exact-config | port-required | rpc-basics |
| RT-G-1-02 <br> 실행: block-by-hash-consistency | [block-by-hash-consistency](../sources/chainbench/tests/tc/go-stablenet/regression/api/02-block-by-hash-consistency.json) | exact-config | port-required | rpc-basics |
| RT-G-1-03 <br> 실행: transaction-by-hash-fields | [transaction-by-hash-fields](../sources/chainbench/tests/tc/go-stablenet/regression/api/03-transaction-by-hash-fields.json) | exact-config | port-required | rpc-basics |
| RT-G-1-04 <br> 실행: transaction-receipt-fields | [transaction-receipt-fields](../sources/chainbench/tests/tc/go-stablenet/regression/api/04-transaction-receipt-fields.json) | exact-config | port-required | rpc-basics |
| RT-G-1-05 <br> 실행: transaction-count-increments | [transaction-count-increments](../sources/chainbench/tests/tc/go-stablenet/regression/api/05-transaction-count-increments.json) | exact-config | port-required | rpc-basics |
| RT-G-4-02 <br> 실행: txpool-status | [txpool-status](../sources/chainbench/tests/tc/go-stablenet/regression/api/18-txpool-status.json) | exact-config | port-required | txpool |
| RT-G-4-03 <br> 실행: txpool-content-well-formed | [txpool-content-well-formed](../sources/chainbench/tests/tc/go-stablenet/regression/api/19-txpool-content-well-formed.json) | exact-config | port-required | txpool |
| RT-G-5-01 <br> 실행: fee-delegate-sign-rpc-present | [fee-delegate-sign-rpc-present](../sources/chainbench/tests/tc/go-stablenet/regression/api/21-fee-delegate-sign-rpc-present.json) | adapter-required | port-required | fee-delegation |
| RT-A-2-01 <br> 실행: legacy-transfer | [legacy-transfer](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/08-legacy-transfer.json) | adapter-required | port-required | transaction-types |
| RT-A-2-02 <br> 실행: dynamic-fee-tx | [dynamic-fee-tx](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/09-dynamic-fee-tx.json) | adapter-required | port-required | transaction-types |
| RT-A-2-03 <br> 실행: access-list-tx | [access-list-tx](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/10-access-list-tx.json) | exact-config | port-required | transaction-types |
| RT-A-2-04 <br> 실행: nonce-ordering | [nonce-ordering](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/11-nonce-ordering.json) | exact-config | port-required | nonce-replacement |
| TX-010 [부분 대응] <br> 실행: out-of-order-nonces-mine | [out-of-order-nonces-mine](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json) | exact-config | port-required | nonce-replacement |
| RT-A-2-05a <br> 실행: dynamic-fee-below-basefee-rejected | [dynamic-fee-below-basefee-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/12-dynamic-fee-below-basefee-rejected.json) | adapter-required | port-required | fee-policy |
| RT-A-2-06 <br> 실행: insufficient-funds-rejected | [insufficient-funds-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/14-insufficient-funds-rejected.json) | exact-config | port-required | invalid-transaction |
| RT-A-2-07 <br> 실행: gas-limit-exceeds-block-rejected | [gas-limit-exceeds-block-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/15-gas-limit-exceeds-block-rejected.json) | exact-config | port-required | invalid-transaction |
| RT-A-2-08 <br> 실행: effective-gas-price | [effective-gas-price](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/16-effective-gas-price.json) | exact-config | port-required | fee-observation |
| RT-A-2-09 <br> 실행: replacement-tx | [replacement-tx](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/17-replacement-tx.json) | exact-config | port-required | nonce-replacement |
| TX-013 [부분 대응] <br> 실행: same-nonce-replacement | [same-nonce-replacement](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/17b-same-nonce-replacement.json) | exact-config | port-required | nonce-replacement |
| RT-A-3-01 <br> 실행: contract-roundtrip | [contract-roundtrip](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/19-contract-roundtrip.json) | exact-config | port-required | evm-contract |
| RT-A-3-05 <br> 실행: eth-call-revert-returns-error | [eth-call-revert-returns-error](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json) | exact-config | port-required | evm-contract |
| RT-A-3-06 <br> 실행: revert-tx-status-zero | [revert-tx-status-zero](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/24-revert-tx-status-zero.json) | exact-config | port-required | evm-contract |
| RT-A-3-07 <br> 실행: out-of-gas-consumes-all | [out-of-gas-consumes-all](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json) | exact-config | port-required | evm-contract |
| RT-A-4-03 <br> 실행: value-transfer | [value-transfer](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/27b-value-transfer.json) | exact-config | port-required | transaction-types |
| RT-A-4-04 / RPC-014 [부분 대응] <br> 실행: contract-event-emitted | [contract-event-emitted](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/36-contract-event-emitted.json) | exact-config | port-required | evm-contract |
| RT-D-01 <br> 실행: fee-delegated-transfer | [fee-delegated-transfer](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json) | adapter-required | port-required | fee-delegation |
| RT-D-03 <br> 실행: fd-sender-sig-invalid-rejected | [fd-sender-sig-invalid-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json) | adapter-required | port-required | fee-delegation |
| RT-D-04 <br> 실행: fd-feepayer-sig-invalid-rejected | [fd-feepayer-sig-invalid-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json) | adapter-required | port-required | fee-delegation |
| RT-D-05 <br> 실행: feepayer-insufficient-rejected | [feepayer-insufficient-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json) | adapter-required | port-required | fee-delegation |
| RT-D-03 / TX-014 [부분 대응] <br> 실행: fee-delegated-sender-sig-invalid-rejected | [fee-delegated-sender-sig-invalid-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json) | adapter-required | port-required | fee-delegation |
| RT-D-04 / TX-015 [부분 대응] <br> 실행: fee-delegated-feepayer-sig-invalid-rejected | [fee-delegated-feepayer-sig-invalid-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json) | adapter-required | port-required | fee-delegation |
| RT-D-05 / TX-016 [부분 대응] <br> 실행: fee-delegated-unfunded-feepayer-rejected | [fee-delegated-unfunded-feepayer-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json) | adapter-required | port-required | fee-delegation |
| wemix-chain-up [명세 ID 미확인] | [wemix-chain-up](../sources/chainbench/tests/tc/go-wemix/chain-up/01-wemix-chain-up.json) | adapter-required | existing-case | sync-lifecycle |
| wemix-chain-up-15 [명세 ID 미확인] | [wemix-chain-up-15](../sources/chainbench/tests/tc/go-wemix/chain-up/02-wemix-chain-up-15.json) | adapter-required | existing-case | sync-lifecycle |
| wemix-node-crash [명세 ID 미확인] | [wemix-node-crash](../sources/chainbench/tests/tc/go-wemix/fault/01-wemix-node-crash.json) | adapter-required | existing-case | fault-topology |
| BRIOCHE-02 / RPC-008 [부분 대응] <br> 실행: wemix-brioche-block-reward | [wemix-brioche-block-reward](../sources/chainbench/tests/tc/go-wemix/rpc/01-wemix-brioche-block-reward.json) | chain-specific | existing-case | reward-compatibility |
| wemix-tx-and-contract [명세 ID 미확인] | [wemix-tx-and-contract](../sources/chainbench/tests/tc/go-wemix/tx/01-wemix-tx-and-contract.json) | exact-config | existing-case | evm-contract |
| stress-block-time [명세 ID 미확인] | [stress-block-time](../sources/chainbench/tests/tc/stress/01-stress-block-time.json) | adapter-required | port-required | stress |
| stress-tx-flood [명세 ID 미확인] | [stress-tx-flood](../sources/chainbench/tests/tc/stress/02-stress-tx-flood.json) | adapter-required | port-required | stress |
| RT-G-2-01 <br> 실행: gas-price-positive | [gas-price-positive](../sources/chainbench/tests/tc/go-stablenet/regression/api/07-gas-price-positive.json) | exact-config | port-required | fee-observation |
| RT-G-2-03 <br> 실행: fee-history-well-formed | [fee-history-well-formed](../sources/chainbench/tests/tc/go-stablenet/regression/api/09-fee-history-well-formed.json) | exact-config | port-required | fee-observation |
| RT-A-1-05 <br> 실행: admin-peers-populated | [admin-peers-populated](../sources/chainbench/tests/tc/go-stablenet/regression/api/20-admin-peers-populated.json) | adapter-required | port-required | sync-lifecycle |
| chain-not-syncing [명세 ID 미확인] | [chain-not-syncing](../sources/chainbench/tests/tc/go-stablenet/regression/api/24-chain-not-syncing.json) | adapter-required | port-required | sync-lifecycle |
| stablenet-chain-up [명세 ID 미확인] | [stablenet-chain-up](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/01-stablenet-chain-up.json) | adapter-required | port-required | sync-lifecycle |
| RT-A-3-04 <br> 실행: estimate-gas | [estimate-gas](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/22-estimate-gas.json) | exact-config | port-required | evm-contract |
| RT-A-4-02 <br> 실행: genesis-balance | [genesis-balance](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/27-genesis-balance.json) | exact-config | port-required | rpc-basics |
| RT-A-4-04 <br> 실행: logs-query-well-formed | [logs-query-well-formed](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/29-logs-query-well-formed.json) | exact-config | port-required | logs-subscription |
| RT-A-1-01 <br> 실행: chain-id | [chain-id](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/30-chain-id.json) | exact-config | port-required | rpc-basics |
| RT-A-4-06 <br> 실행: ws-subscribe-new-heads | [ws-subscribe-new-heads](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/31-ws-subscribe-new-heads.json) | exact-config | port-required | logs-subscription |
| RT-A-4-07 <br> 실행: ws-subscribe-logs | [ws-subscribe-logs](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/32-ws-subscribe-logs.json) | exact-config | port-required | logs-subscription |
| stablenet-chain-up-15 [명세 ID 미확인] | [stablenet-chain-up-15](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/33-stablenet-chain-up-15.json) | adapter-required | port-required | sync-lifecycle |
| stablenet-proxied-pn-routing [명세 ID 미확인] | [stablenet-proxied-pn-routing](../sources/chainbench/tests/tc/go-stablenet/topology/01-proxied-pn-routing.json) | adapter-required | conditional | fault-topology |
| stablenet-negative-tx-revert [명세 ID 미확인] | [stablenet-negative-tx-revert](../sources/chainbench/tests/tc/go-stablenet/tx/01-negative-tx-revert.json) | exact-config | port-required | evm-contract |
| wbft-chain-up [명세 ID 미확인] | [wbft-chain-up](../sources/chainbench/tests/tc/go-wbft/chain-up/01-wbft-chain-up.json) | adapter-required | port-required | sync-lifecycle |
| wbft-chain-up-15 [명세 ID 미확인] | [wbft-chain-up-15](../sources/chainbench/tests/tc/go-wbft/chain-up/02-wbft-chain-up-15.json) | adapter-required | port-required | sync-lifecycle |
| e1-mixed-producers [명세 ID 미확인] | [e1-mixed-producers](../sources/chainbench/tests/tc/go-wbft/consensus/01-e1-mixed-producers.json) | chain-specific | port-required | wbft-specific |
| wbft-node-crash [명세 ID 미확인] | [wbft-node-crash](../sources/chainbench/tests/tc/go-wbft/fault/01-wbft-node-crash.json) | adapter-required | port-required | fault-topology |
| wbft-tx-and-contract [명세 ID 미확인] | [wbft-tx-and-contract](../sources/chainbench/tests/tc/go-wbft/tx/01-wbft-tx-and-contract.json) | exact-config | port-required | evm-contract |
| wemix-wbft-handoff [명세 ID 미확인] | [wemix-wbft-handoff](../sources/chainbench/tests/tc/go-wemix/handoff/01-wemix-wbft-handoff.json) | chain-specific | conditional | handoff-specific |
| remote-rpc-health [명세 ID 미확인] | [remote-rpc-health](../sources/chainbench/tests/tc/remote/01-remote-rpc-health.json) | exact-config | port-required | rpc-basics |
| remote-chain-info [명세 ID 미확인] | [remote-chain-info](../sources/chainbench/tests/tc/remote/02-remote-chain-info.json) | exact-config | port-required | rpc-basics |
| remote-balance-check [명세 ID 미확인] | [remote-balance-check](../sources/chainbench/tests/tc/remote/03-remote-balance-check.json) | exact-config | port-required | rpc-basics |
| sample-minimal-value-transfer [명세 ID 미확인] | [sample-minimal-value-transfer](../sources/chainbench/tests/tc/samples/01-sample-minimal.json) | exact-config | port-required | transaction-types |
| sample-lifecycle-node-restart [명세 ID 미확인] | [sample-lifecycle-node-restart](../sources/chainbench/tests/tc/samples/02-sample-lifecycle.json) | adapter-required | port-required | fault-topology |
| stablenet-derived-vocabulary [명세 ID 미확인] | [stablenet-derived-vocabulary](../sources/chainbench/tests/tc/go-stablenet/vocabulary/01-derived-address-and-checksum.json) | exact-config | port-required | harness-vocabulary |
| stablenet-faucet-funds [명세 ID 미확인] | [stablenet-faucet-funds](../sources/chainbench/tests/tc/go-stablenet/vocabulary/02-faucet-funds-account.json) | unknown | conditional | external-faucet |
| stablenet-register-contract [명세 ID 미확인] | [stablenet-register-contract](../sources/chainbench/tests/tc/go-stablenet/vocabulary/03-register-contract.json) | exact-config | port-required | evm-contract |
| basic-wbft-consensus [명세 ID 미확인] | [basic-wbft-consensus](../sources/chainbench/tests/tc/basic/07-basic-wbft-consensus.json) | chain-specific | not-applicable | stablenet-specific |
| stablenet-delayed-fork [명세 ID 미확인] | [stablenet-delayed-fork](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json) | chain-specific | not-applicable | stablenet-specific |
| TC-5-2-05 <br> 실행: govminter-v2-code | [govminter-v2-code](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json) | chain-specific | not-applicable | stablenet-specific |
| TC-1-1-01 / TC-1-1-10 <br> 실행: burn-cancel-refundable | [burn-cancel-refundable](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json) | chain-specific | not-applicable | stablenet-specific |
| TC-1-1-02 <br> 실행: burn-reject-refundable | [burn-reject-refundable](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json) | chain-specific | not-applicable | stablenet-specific |
| TC-1-1-03 <br> 실행: burn-expire-refundable | [burn-expire-refundable](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json) | chain-specific | not-applicable | stablenet-specific |
| TC-1-1-04 <br> 실행: burn-execute-no-refundable | [burn-execute-no-refundable](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json) | chain-specific | not-applicable | stablenet-specific |
| TC-1-1-05 / TC-1-1-09 <br> 실행: claim-burn-refund-succeeds | [claim-burn-refund-succeeds](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json) | chain-specific | not-applicable | stablenet-specific |
| TC-1-1-06 <br> 실행: claim-zero-refund-reverts | [claim-zero-refund-reverts](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json) | chain-specific | not-applicable | stablenet-specific |
| TC-1-1-07 <br> 실행: claim-burn-refund-double-reverts | [claim-burn-refund-double-reverts](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json) | chain-specific | not-applicable | stablenet-specific |
| TC-1-1-11 / TC-1-1-12 <br> 실행: prealloc-preserved-across-boho | [prealloc-preserved-across-boho](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json) | chain-specific | not-applicable | stablenet-specific |
| TC-4-1-01 <br> 실행: boho-chain-config-active | [boho-chain-config-active](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/15-boho-chain-config-active.json) | chain-specific | not-applicable | stablenet-specific |
| TC-4-1-02 <br> 실행: anzeon-active-before-boho | [anzeon-active-before-boho](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/16-anzeon-active-before-boho.json) | chain-specific | not-applicable | stablenet-specific |
| TC-4-2-01 / TC-4-2-03 <br> 실행: estimategas-authorizationlist-cost | [estimategas-authorizationlist-cost](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json) | chain-specific | not-applicable | modern-evm-specific |
| TC-5-2-01 / TC-5-2-02 <br> 실행: upgrade-registry-order | [upgrade-registry-order](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json) | chain-specific | not-applicable | stablenet-specific |
| TC-5-2-04 <br> 실행: v1-params-init-storage | [v1-params-init-storage](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json) | chain-specific | not-applicable | stablenet-specific |
| TC-1-1-09 / TC-1-1-10 [부분 대응] <br> 실행: burn-refund-events | [burn-refund-events](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json) | chain-specific | not-applicable | stablenet-specific |
| TC-4-6-01 <br> 실행: effective-gas-price-authorized-bp-en | [effective-gas-price-authorized-bp-en](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json) | chain-specific | not-applicable | stablenet-specific |
| TC-4-6-02 <br> 실행: effective-gas-price-regular | [effective-gas-price-regular](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json) | adapter-required | port-required | fee-observation |
| TC-4-6-04 <br> 실행: auth-tx-event-last-bp-en | [auth-tx-event-last-bp-en](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json) | chain-specific | not-applicable | stablenet-specific |
| TC-4-5-01 / TC-4-5-02 <br> 실행: authorized-extra-bit-synced | [authorized-extra-bit-synced](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/01-authorized-extra-bit-synced.json) | chain-specific | not-applicable | stablenet-specific |
| TC-4-5-02 [부분 대응] <br> 실행: blacklisted-extra-bit-synced | [blacklisted-extra-bit-synced](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/01b-blacklisted-extra-bit-synced.json) | chain-specific | not-applicable | stablenet-specific |
| TC-4-5-01 / TC-4-5-02 / TC-4-5-07 [부분 대응] <br> 실행: stablenet-account-extra | [stablenet-account-extra](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json) | chain-specific | not-applicable | stablenet-specific |
| TC-4-5-05 / TC-4-5-06 <br> 실행: extra-union-merge | [extra-union-merge](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json) | chain-specific | not-applicable | stablenet-specific |
| TC-4-5-07 <br> 실행: dual-status-extra | [dual-status-extra](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json) | chain-specific | not-applicable | stablenet-specific |
| TC-4-5-08 <br> 실행: extra-balance-preserved | [extra-balance-preserved](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/05-extra-balance-preserved.json) | chain-specific | not-applicable | stablenet-specific |
| TC-4-5-09 <br> 실행: invalid-extra-reject | [invalid-extra-reject](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json) | chain-specific | not-applicable | stablenet-specific |
| TC-4-5-10 / TC-4-5-11 <br> 실행: extra-state-across-delayed-boho | [extra-state-across-delayed-boho](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json) | chain-specific | not-applicable | stablenet-specific |
| TC-5-2-06 <br> 실행: unsupported-system-contract-version | [unsupported-system-contract-version](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json) | chain-specific | not-applicable | stablenet-specific |
| TC-4-3-01 <br> 실행: authorized-accounts-no-space | [authorized-accounts-no-space](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/01-authorized-accounts-no-space.json) | chain-specific | not-applicable | stablenet-specific |
| TC-4-3-02 <br> 실행: authorized-accounts-space | [authorized-accounts-space](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/02-authorized-accounts-space.json) | chain-specific | not-applicable | stablenet-specific |
| TC-4-3-03 <br> 실행: authorized-accounts-trim | [authorized-accounts-trim](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/03-authorized-accounts-trim.json) | chain-specific | not-applicable | stablenet-specific |
| TC-4-3-04 <br> 실행: authorized-accounts-empty-item | [authorized-accounts-empty-item](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/04-authorized-accounts-empty-item.json) | chain-specific | not-applicable | stablenet-specific |
| TC-4-3-05 <br> 실행: authorized-accounts-single | [authorized-accounts-single](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/05-authorized-accounts-single.json) | chain-specific | not-applicable | stablenet-specific |
| TC-4-3-06 <br> 실행: authorized-accounts-empty | [authorized-accounts-empty](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/06-authorized-accounts-empty.json) | chain-specific | not-applicable | stablenet-specific |
| RT-C-01 <br> 실행: regular-account-gastip-forced | [regular-account-gastip-forced](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json) | chain-specific | not-applicable | stablenet-specific |
| RT-C-02 <br> 실행: authorized-account-gastip-free | [authorized-account-gastip-free](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json) | chain-specific | not-applicable | stablenet-specific |
| RT-C-03 [부분 대응] <br> 실행: anzeon-basefee-increase | [anzeon-basefee-increase](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json) | adapter-required | port-required | fee-policy |
| RT-C-04 [부분 대응] <br> 실행: anzeon-basefee-stable | [anzeon-basefee-stable](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json) | adapter-required | port-required | fee-policy |
| RT-C-05 [부분 대응] <br> 실행: anzeon-basefee-decrease | [anzeon-basefee-decrease](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json) | adapter-required | port-required | fee-policy |
| RT-C-06 <br> 실행: basefee-minimum | [basefee-minimum](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/06-basefee-minimum.json) | adapter-required | port-required | fee-policy |
| RT-C-07 <br> 실행: basefee-maximum | [basefee-maximum](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/07-basefee-maximum.json) | chain-specific | port-required | fee-policy |
| TC-1-3-03 [부분 대응] <br> 실행: feecap-above-min-accepted | [feecap-above-min-accepted](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json) | adapter-required | port-required | fee-policy |
| TC-1-3-02 [부분 대응] <br> 실행: feecap-exact-min-accepted | [feecap-exact-min-accepted](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json) | adapter-required | port-required | fee-policy |
| RT-A-2-07 / TX-012 [부분 대응] <br> 실행: gaslimit-exceeded-rejected | [gaslimit-exceeded-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json) | adapter-required | port-required | invalid-transaction |
| RT-G-1-06 <br> 실행: system-contracts-deployed | [system-contracts-deployed](../sources/chainbench/tests/tc/go-stablenet/regression/api/06-system-contracts-deployed.json) | chain-specific | not-applicable | stablenet-specific |
| RT-G-2-01 [부분 대응] <br> 실행: gas-price-equals-basefee-plus-tip | [gas-price-equals-basefee-plus-tip](../sources/chainbench/tests/tc/go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json) | adapter-required | port-required | fee-observation |
| RT-G-2-02 <br> 실행: max-priority-fee-equals-gastip | [max-priority-fee-equals-gastip](../sources/chainbench/tests/tc/go-stablenet/regression/api/08-max-priority-fee-equals-gastip.json) | chain-specific | not-applicable | stablenet-specific |
| RT-G-2-04 <br> 실행: estimate-gas-token-transfer | [estimate-gas-token-transfer](../sources/chainbench/tests/tc/go-stablenet/regression/api/10-estimate-gas-token-transfer.json) | chain-specific | not-applicable | stablenet-specific |
| RT-G-3-01 <br> 실행: node-address-returned | [node-address-returned](../sources/chainbench/tests/tc/go-stablenet/regression/api/11-node-address-returned.json) | chain-specific | not-applicable | wbft-specific |
| RT-G-3-02 <br> 실행: validator-set-nonempty | [validator-set-nonempty](../sources/chainbench/tests/tc/go-stablenet/regression/api/12-validator-set-nonempty.json) | chain-specific | not-applicable | wbft-specific |
| RT-B-07 [부분 대응] <br> 실행: validator-set-count | [validator-set-count](../sources/chainbench/tests/tc/go-stablenet/regression/api/12b-validator-set-count.json) | chain-specific | not-applicable | wbft-specific |
| RT-G-3-03 <br> 실행: commit-signers-quorum | [commit-signers-quorum](../sources/chainbench/tests/tc/go-stablenet/regression/api/13-commit-signers-quorum.json) | chain-specific | not-applicable | wbft-specific |
| RT-G-3-04 <br> 실행: wbft-extra-info-fields | [wbft-extra-info-fields](../sources/chainbench/tests/tc/go-stablenet/regression/api/14-wbft-extra-info-fields.json) | chain-specific | not-applicable | wbft-specific |
| RT-G-3-05 <br> 실행: istanbul-status-fields | [istanbul-status-fields](../sources/chainbench/tests/tc/go-stablenet/regression/api/15-istanbul-status-fields.json) | chain-specific | not-applicable | wbft-specific |
| RT-G-3-06 <br> 실행: is-validator-flags | [is-validator-flags](../sources/chainbench/tests/tc/go-stablenet/regression/api/16-is-validator-flags.json) | chain-specific | not-applicable | wbft-specific |
| RT-G-5-02 <br> 실행: token-total-supply-readable | [token-total-supply-readable](../sources/chainbench/tests/tc/go-stablenet/regression/api/22-token-total-supply-readable.json) | chain-specific | not-applicable | stablenet-specific |
| RT-G-5-03 <br> 실행: token-approve-sets-allowance | [token-approve-sets-allowance](../sources/chainbench/tests/tc/go-stablenet/regression/api/23-token-approve-sets-allowance.json) | chain-specific | not-applicable | stablenet-specific |
| RT-E-01 <br> 실행: sender-blacklisted-rejected | [sender-blacklisted-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json) | chain-specific | not-applicable | stablenet-specific |
| RT-E-02 <br> 실행: recipient-blacklisted-rejected | [recipient-blacklisted-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json) | chain-specific | not-applicable | stablenet-specific |
| RT-E-03 <br> 실행: feepayer-blacklisted-rejected | [feepayer-blacklisted-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json) | chain-specific | not-applicable | stablenet-specific |
| RT-E-04 <br> 실행: address-unblacklisted-event | [address-unblacklisted-event](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json) | chain-specific | not-applicable | stablenet-specific |
| RT-E-05 <br> 실행: zero-address-transfer-rejected | [zero-address-transfer-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json) | chain-specific | not-applicable | stablenet-specific |
| RT-E-06 <br> 실행: precompile-transfer-rejected | [precompile-transfer-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json) | chain-specific | not-applicable | stablenet-specific |
| RT-E-07 <br> 실행: account-blacklist-readable | [account-blacklist-readable](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/07-account-blacklist-readable.json) | chain-specific | not-applicable | stablenet-specific |
| RT-E-08 <br> 실행: account-authorization-readable | [account-authorization-readable](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/08-account-authorization-readable.json) | chain-specific | not-applicable | stablenet-specific |
| RT-E-09 <br> 실행: authorized-tx-executed-event | [authorized-tx-executed-event](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json) | chain-specific | not-applicable | stablenet-specific |
| RT-A-2-10 <br> 실행: set-code-delegation | [set-code-delegation](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/18-set-code-delegation.json) | chain-specific | not-applicable | modern-evm-specific |
| RT-F-1-01 <br> 실행: native-coin-adapter-code | [native-coin-adapter-code](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/01-native-coin-adapter-code.json) | chain-specific | not-applicable | stablenet-specific |
| RT-F-1-01 [부분 대응] <br> 실행: token-transfer-emits-event | [token-transfer-emits-event](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json) | chain-specific | not-applicable | stablenet-specific |
| RT-F-1-02 <br> 실행: token-balance-readable | [token-balance-readable](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/02-token-balance-readable.json) | chain-specific | not-applicable | stablenet-specific |
| RT-F-1-03 [부분 대응] <br> 실행: token-transfer-from-moves-balance | [token-transfer-from-moves-balance](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json) | chain-specific | not-applicable | stablenet-specific |
| RT-F-1-04 <br> 실행: mint-transfer-event | [mint-transfer-event](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/04-mint-transfer-event.json) | chain-specific | not-applicable | stablenet-specific |
| RT-F-1-05 <br> 실행: burn-transfer-event | [burn-transfer-event](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/05-burn-transfer-event.json) | chain-specific | not-applicable | stablenet-specific |
| RT-F-2-01 <br> 실행: mint-proposal-executes | [mint-proposal-executes](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/06-mint-proposal-executes.json) | chain-specific | not-applicable | stablenet-specific |
| RT-F-2-02 <br> 실행: burn-proposal-executes | [burn-proposal-executes](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/07-burn-proposal-executes.json) | chain-specific | not-applicable | stablenet-specific |
| RT-F-2-03 <br> 실행: quorum-deficient-stays-voting | [quorum-deficient-stays-voting](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json) | chain-specific | not-applicable | stablenet-specific |
| RT-F-3-04 <br> 실행: validator-metadata-readable | [validator-metadata-readable](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/09-validator-metadata-readable.json) | chain-specific | not-applicable | stablenet-specific |
| RT-B-06 <br> 실행: gastip-governance-updates-header | [gastip-governance-updates-header](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json) | chain-specific | not-applicable | stablenet-specific |
| RT-F-3-06 <br> 실행: proposal-expiry-transitions | [proposal-expiry-transitions](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json) | chain-specific | not-applicable | stablenet-specific |
| RT-F-4-01 <br> 실행: configure-minter-proposal-executes | [configure-minter-proposal-executes](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json) | chain-specific | not-applicable | stablenet-specific |
| RT-F-4-02 <br> 실행: remove-minter-executes | [remove-minter-executes](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/13-remove-minter-executes.json) | chain-specific | not-applicable | stablenet-specific |
| RT-F-4-03 <br> 실행: masterminter-member-add-remove | [masterminter-member-add-remove](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json) | chain-specific | not-applicable | stablenet-specific |
| RT-F-4-04 <br> 실행: non-member-configure-minter-rejected | [non-member-configure-minter-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json) | chain-specific | not-applicable | stablenet-specific |
| RT-F-5-01 <br> 실행: blacklist-proposal-executes | [blacklist-proposal-executes](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json) | chain-specific | not-applicable | stablenet-specific |
| RT-F-5-03 <br> 실행: authorize-proposal-executes | [authorize-proposal-executes](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json) | chain-specific | not-applicable | stablenet-specific |
| RT-F-5-04 / RT-F-5-09 [부분 대응] <br> 실행: unauthorize-proposal-executes | [unauthorize-proposal-executes](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json) | chain-specific | not-applicable | stablenet-specific |
| RT-F-5-05 <br> 실행: direct-blacklist-call-rejected | [direct-blacklist-call-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json) | chain-specific | not-applicable | stablenet-specific |
| RT-F-5-08 <br> 실행: authorized-account-added-event | [authorized-account-added-event](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/23-authorized-account-added-event.json) | chain-specific | not-applicable | stablenet-specific |
| token-metadata [명세 ID 미확인] | [token-metadata](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/25-token-metadata.json) | chain-specific | not-applicable | stablenet-specific |
| minter-status-readable [명세 ID 미확인] | [minter-status-readable](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/28-minter-status-readable.json) | chain-specific | not-applicable | stablenet-specific |
| RT-B-01 <br> 실행: block-period-one-second | [block-period-one-second](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/01-block-period-one-second.json) | adapter-required | port-required | sync-lifecycle |
| RT-B-02 <br> 실행: wbft-seals-quorum | [wbft-seals-quorum](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/02-wbft-seals-quorum.json) | chain-specific | not-applicable | wbft-specific |
| RT-B-03 <br> 실행: epoch-transition-carries-epoch-info | [epoch-transition-carries-epoch-info](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json) | chain-specific | not-applicable | wbft-specific |
| RT-B-04 <br> 실행: validator-add-member-executes | [validator-add-member-executes](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/04-validator-add-member-executes.json) | chain-specific | not-applicable | wbft-specific |
| RT-B-04 <br> 실행: validator-add-member-epoch-activates | [validator-add-member-epoch-activates](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json) | chain-specific | not-applicable | wbft-specific |
| RT-B-05 <br> 실행: validator-remove-member-executes | [validator-remove-member-executes](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/05-validator-remove-member-executes.json) | chain-specific | not-applicable | wbft-specific |
| RT-B-11 <br> 실행: prev-seals-quorum | [prev-seals-quorum](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/11-prev-seals-quorum.json) | chain-specific | not-applicable | wbft-specific |
| WBFT-010 [부분 대응] <br> 실행: randao-and-mixdigest-present | [randao-and-mixdigest-present](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/13-randao-and-mixdigest-present.json) | chain-specific | not-applicable | wbft-specific |
| RT-B-06 [부분 대응] <br> 실행: stablenet-gastip-field | [stablenet-gastip-field](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/14-stablenet-gastip-field.json) | chain-specific | not-applicable | wbft-specific |
| TX-009 [부분 대응] <br> 실행: secp256r1-precompile-valid | [secp256r1-precompile-valid](../sources/chainbench/tests/tc/go-wbft/accounts/01-secp256r1-precompile-valid.json) | chain-specific | not-applicable | modern-evm-specific |
| TX-019 [부분 대응] <br> 실행: secp256r1-precompile-invalid | [secp256r1-precompile-invalid](../sources/chainbench/tests/tc/go-wbft/accounts/02-secp256r1-precompile-invalid.json) | chain-specific | not-applicable | modern-evm-specific |
| TX-020 [부분 대응] <br> 실행: secp256r1-precompile-short-input | [secp256r1-precompile-short-input](../sources/chainbench/tests/tc/go-wbft/accounts/03-secp256r1-precompile-short-input.json) | chain-specific | not-applicable | modern-evm-specific |

의존 필드 정밀 재검토: JSON의 정확한 key와 step 문맥으로 분류했다. schemaVersion/description은 제외하며, 일반 sendTx의 to는 수신 계정으로 기록한다. env.capabilities/requires/applicableChains는 별도 capability_dependencies에 두었다. source_fields와 field_bindings에 literal/binding/role/config-reference를 구분하고 실제 값은 생략했다. 빈 목록은 환경 상속 또는 사전 의존이 없다는 뜻이 아니다. 세부 행은 all-tests.json을 참조한다.
