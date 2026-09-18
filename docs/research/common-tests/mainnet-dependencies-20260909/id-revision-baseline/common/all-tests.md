# 전체 테스트 목록

문서 45 + 29행, JSON 189개 = 263개 원본 항목이다. 중복 시나리오도 원본 ID를 보존했다. JSON 파일명/설명보다 실제 do/expect를 기준으로 분류했다. 189개 JSON 및 두 문서는 이전 분석 스냅샷과 바이트 동일하다. 이전 PR196 우선순위는 본 작업에 사용하지 않는다.

`exact-config`: 시나리오의 공통 assertion을 유지하고 기존 환경 adapter에 profile 값을 제공하는 후보. 그대로 실행 가능하다는 뜻은 아니다. `adapter-required`: DSL 명세 구현 또는 합의·수수료·서명 등 별도 동작 필요. `chain-specific`: 원문 목적을 세 체인 공통으로 충족할 수 없음. `unknown`: 외부 서비스 등 확인되지 않은 선행조건. 명세에 이름만 있는 shell script는 실행 파일이 확보된 것으로 세지 않았다.


원본 테스트 입력은 이전 보존본과 동일하지만 하네스 분석은 현재 dirty working tree를 보존한 helper 기준이다. 이전 helper와 동일하다는 뜻은 아니다. TC128은 WBFT에 고정 baseFee 최대 clamp가 없어 공통에서 제외했다.


| ID | 테스트 | 공통성 | go-wemix | family |
|---|---|---|---|---|
| DOC-C-001 | [Legacy Tx (Type 0x0) 전송 및 실행](../sources/common_test_scenarios.md) | adapter-required | port-required | transaction-types |
| DOC-C-002 | [Dynamic Fee Tx (Type 0x2, EIP-1559)](../sources/common_test_scenarios.md) | adapter-required | port-required | transaction-types |
| DOC-C-003 | [Access List Tx (Type 0x1, EIP-2930)](../sources/common_test_scenarios.md) | adapter-required | port-required | transaction-types |
| DOC-C-004 | [수수료 대납 Tx (Type 0x16) 정상 처리](../sources/common_test_scenarios.md) | adapter-required | port-required | fee-delegation |
| DOC-C-005 | [대납 Tx Sender 서명 변조 거부](../sources/common_test_scenarios.md) | adapter-required | port-required | fee-delegation |
| DOC-C-006 | [대납 Tx FeePayer 서명 변조 거부](../sources/common_test_scenarios.md) | adapter-required | port-required | fee-delegation |
| DOC-C-007 | [FeePayer 잔액 부족시 실행 실패](../sources/common_test_scenarios.md) | adapter-required | port-required | fee-delegation |
| DOC-C-008 | [Nonce 순서 보장 (Nonce Ordering)](../sources/common_test_scenarios.md) | adapter-required | port-required | nonce-replacement |
| DOC-C-009 | [TipCap 미달 (Underpriced) 거부](../sources/common_test_scenarios.md) | adapter-required | conditional | fee-policy |
| DOC-C-010 | [잔액 부족 (Insufficient Funds) 거부](../sources/common_test_scenarios.md) | adapter-required | port-required | invalid-transaction |
| DOC-C-011 | [Gas Limit 초과 거부](../sources/common_test_scenarios.md) | adapter-required | port-required | invalid-transaction |
| DOC-C-012 | [Effective GasPrice 노드 간 일관성](../sources/common_test_scenarios.md) | adapter-required | port-required | fee-observation |
| DOC-C-013 | [Queued/Pending 트랜잭션 교체 (Replacement Tx)](../sources/common_test_scenarios.md) | adapter-required | port-required | nonce-replacement |
| DOC-C-014 | [스마트 컨트랙트 배포](../sources/common_test_scenarios.md) | adapter-required | port-required | evm-contract |
| DOC-C-015 | [컨트랙트 상태 변경 함수 호출](../sources/common_test_scenarios.md) | adapter-required | port-required | evm-contract |
| DOC-C-016 | [`eth_call` View/Pure 함수 조회](../sources/common_test_scenarios.md) | adapter-required | port-required | evm-contract |
| DOC-C-017 | [`eth_estimateGas` 가스 추정](../sources/common_test_scenarios.md) | adapter-required | port-required | evm-contract |
| DOC-C-018 | [`eth_call` 실행 중 Revert 메세지 반환](../sources/common_test_scenarios.md) | adapter-required | port-required | evm-contract |
| DOC-C-019 | [Revert 트랜잭션 (상태 롤백 & 가스 환불)](../sources/common_test_scenarios.md) | adapter-required | port-required | evm-contract |
| DOC-C-020 | [Out-of-Gas 트랜잭션 (가스 Limit 전액 소모)](../sources/common_test_scenarios.md) | adapter-required | port-required | evm-contract |
| DOC-C-021 | [`eth_blockNumber`](../sources/common_test_scenarios.md) | adapter-required | port-required | rpc-basics |
| DOC-C-022 | [`eth_getBalance`](../sources/common_test_scenarios.md) | adapter-required | port-required | rpc-basics |
| DOC-C-023 | [`eth_sendRawTransaction`](../sources/common_test_scenarios.md) | adapter-required | port-required | rpc-basics |
| DOC-C-024 | [`eth_getLogs`](../sources/common_test_scenarios.md) | adapter-required | port-required | logs-subscription |
| DOC-C-025 | [`eth_chainId`](../sources/common_test_scenarios.md) | adapter-required | port-required | rpc-basics |
| DOC-C-026 | [WebSocket `eth_subscribe("newHeads")`](../sources/common_test_scenarios.md) | adapter-required | port-required | logs-subscription |
| DOC-C-027 | [WebSocket `eth_subscribe("logs")`](../sources/common_test_scenarios.md) | adapter-required | port-required | logs-subscription |
| DOC-C-028 | [`eth_getBlockByNumber`](../sources/common_test_scenarios.md) | adapter-required | port-required | rpc-basics |
| DOC-C-029 | [`eth_getBlockByHash`](../sources/common_test_scenarios.md) | adapter-required | port-required | rpc-basics |
| DOC-C-030 | [`eth_getTransactionByHash`](../sources/common_test_scenarios.md) | adapter-required | port-required | rpc-basics |
| DOC-C-031 | [`eth_getTransactionReceipt`](../sources/common_test_scenarios.md) | adapter-required | port-required | rpc-basics |
| DOC-C-032 | [`eth_getTransactionCount`](../sources/common_test_scenarios.md) | adapter-required | port-required | rpc-basics |
| DOC-C-033 | [`eth_gasPrice`](../sources/common_test_scenarios.md) | adapter-required | port-required | fee-observation |
| DOC-C-034 | [`eth_maxPriorityFeePerGas`](../sources/common_test_scenarios.md) | adapter-required | port-required | fee-observation |
| DOC-C-035 | [`eth_feeHistory`](../sources/common_test_scenarios.md) | adapter-required | port-required | fee-observation |
| DOC-C-036 | [`txpool_status`](../sources/common_test_scenarios.md) | adapter-required | port-required | txpool |
| DOC-C-037 | [`txpool_content`](../sources/common_test_scenarios.md) | adapter-required | port-required | txpool |
| DOC-C-038 | [FeePayer 서명 RPC 헬스체크 (`eth_signRawFeeDelegateTransaction`)](../sources/common_test_scenarios.md) | adapter-required | port-required | fee-delegation |
| DOC-C-039 | [제네시스 초기화 (Genesis Initialization)](../sources/common_test_scenarios.md) | adapter-required | port-required | genesis-upgrade |
| DOC-C-040 | [Full Sync 동기화](../sources/common_test_scenarios.md) | adapter-required | port-required | sync-lifecycle |
| DOC-C-041 | [Snap Sync 동기화](../sources/common_test_scenarios.md) | adapter-required | conditional | sync-lifecycle |
| DOC-C-042 | [노드 재기동 후 동기화 유지](../sources/common_test_scenarios.md) | adapter-required | port-required | sync-lifecycle |
| DOC-C-043 | [P2P Bootnode 피어 연결](../sources/common_test_scenarios.md) | adapter-required | port-required | sync-lifecycle |
| DOC-C-044 | [Block Downloader 경로 동기화](../sources/common_test_scenarios.md) | adapter-required | port-required | sync-lifecycle |
| DOC-C-045 | [Block Fetcher 경로 동기화](../sources/common_test_scenarios.md) | adapter-required | port-required | sync-lifecycle |
| DOC-D-001 | [`wemix_getBriocheBlockReward` 하위 호환성](../sources/dual_chain_test_scenarios.md) | chain-specific | conditional | reward-compatibility |
| DOC-D-002 | [블록 생산 주기 1초](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| DOC-D-003 | [블록 Finalize 및 CommittedSeal 존재](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| DOC-D-004 | [에폭 전환 시 검증자 집합 갱신](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| DOC-D-005 | [블록 헤더 WBFTExtra 필드 반영](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| DOC-D-006 | [View Change / 라운드 체인지](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| DOC-D-007 | [라운드 체인지 후 블록 연결](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| DOC-D-008 | [RoundRobin Proposer 정책](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| DOC-D-009 | [PrevCommittedSeal 수집](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| DOC-D-010 | [PrevPreparedSeal 수집](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| DOC-D-011 | [RandaoReveal / MixDigest](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| DOC-D-012 | [쿼럼 미달 블록 수락 거부](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| DOC-D-013 | [1/3 미만 장애 시 합의 지속](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| DOC-D-014 | [1/3 이상 장애 시 합의 중단](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| DOC-D-015 | [쿼럼 계산 — validator 3개 (전원 필요)](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| DOC-D-016 | [쿼럼 계산 — validator 6개, 1개 장애](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| DOC-D-017 | [쿼럼 계산 — validator 6개, 2개 장애](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| DOC-D-018 | [`istanbul_nodeAddress`](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| DOC-D-019 | [`istanbul_getValidators`](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| DOC-D-020 | [`istanbul_getCommitSignersFromBlock`](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| DOC-D-021 | [`istanbul_getWbftExtraInfo`](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| DOC-D-022 | [`istanbul_status`](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| DOC-D-023 | [`istanbul_isValidator`](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | wbft-specific |
| DOC-D-024 | [SetCode Tx (Type 0x4, EIP-7702)](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | modern-evm-specific |
| DOC-D-025 | [EIP-7702 AuthorizationList estimateGas 비용](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | modern-evm-specific |
| DOC-D-026 | [EIP-7702 estimateGas baseline](../sources/dual_chain_test_scenarios.md) | adapter-required | conditional | evm-contract |
| DOC-D-027 | [secp256r1 프리컴파일 — 유효 서명](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | modern-evm-specific |
| DOC-D-028 | [secp256r1 프리컴파일 — 무효 서명](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | modern-evm-specific |
| DOC-D-029 | [secp256r1 프리컴파일 — 잘못된 입력 길이](../sources/dual_chain_test_scenarios.md) | chain-specific | not-applicable | modern-evm-specific |
| TC-001 | [basic-consensus](../sources/chainbench/tests/tc/basic/01-basic-consensus.json) | adapter-required | port-required | sync-lifecycle |
| TC-002 | [basic-peers](../sources/chainbench/tests/tc/basic/02-basic-peers.json) | adapter-required | port-required | sync-lifecycle |
| TC-003 | [basic-rpc-health](../sources/chainbench/tests/tc/basic/03-basic-rpc-health.json) | exact-config | port-required | rpc-basics |
| TC-004 | [basic-sync](../sources/chainbench/tests/tc/basic/04-basic-sync.json) | adapter-required | port-required | sync-lifecycle |
| TC-005 | [basic-tx-send](../sources/chainbench/tests/tc/basic/05-basic-tx-send.json) | exact-config | port-required | rpc-basics |
| TC-006 | [basic-txpool-propagation](../sources/chainbench/tests/tc/basic/06-basic-txpool-propagation.json) | exact-config | port-required | txpool |
| TC-007 | [fault-network-partition](../sources/chainbench/tests/tc/fault/01-fault-network-partition.json) | adapter-required | port-required | fault-topology |
| TC-008 | [fault-node-crash](../sources/chainbench/tests/tc/fault/02-fault-node-crash.json) | adapter-required | port-required | fault-topology |
| TC-009 | [fault-node-recover](../sources/chainbench/tests/tc/fault/03-fault-node-recover.json) | adapter-required | port-required | fault-topology |
| TC-010 | [fault-p2p-topology](../sources/chainbench/tests/tc/fault/04-fault-p2p-topology.json) | adapter-required | port-required | fault-topology |
| TC-011 | [fault-two-down](../sources/chainbench/tests/tc/fault/05-fault-two-down.json) | adapter-required | port-required | fault-topology |
| TC-012 | [fault-txpool-leader-change](../sources/chainbench/tests/tc/fault/06-fault-txpool-leader-change.json) | adapter-required | port-required | fault-topology |
| TC-013 | [legacy-gasprice-below-min-rejected](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/12-legacy-gasprice-below-min-rejected.json) | adapter-required | port-required | fee-policy |
| TC-014 | [accesslist-gasprice-below-min-rejected](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/13-accesslist-gasprice-below-min-rejected.json) | adapter-required | port-required | fee-policy |
| TC-015 | [feecap-below-min-rejected](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/14-feecap-below-min-rejected.json) | adapter-required | port-required | fee-policy |
| TC-016 | [effective-gas-price-regular-bp-en](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json) | exact-config | port-required | fee-observation |
| TC-017 | [signature-compat-across-swap](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json) | adapter-required | port-required | genesis-upgrade |
| TC-018 | [genesis-mismatch-refuses-to-start](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json) | adapter-required | port-required | genesis-upgrade |
| TC-019 | [genesis-block-hash-consistent](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/04-genesis-block-hash-consistent.json) | adapter-required | port-required | genesis-upgrade |
| TC-020 | [block-transactions-field](../sources/chainbench/tests/tc/go-stablenet/regression/api/01-block-transactions-field.json) | exact-config | port-required | rpc-basics |
| TC-021 | [block-by-hash-consistency](../sources/chainbench/tests/tc/go-stablenet/regression/api/02-block-by-hash-consistency.json) | exact-config | port-required | rpc-basics |
| TC-022 | [transaction-by-hash-fields](../sources/chainbench/tests/tc/go-stablenet/regression/api/03-transaction-by-hash-fields.json) | exact-config | port-required | rpc-basics |
| TC-023 | [transaction-receipt-fields](../sources/chainbench/tests/tc/go-stablenet/regression/api/04-transaction-receipt-fields.json) | exact-config | port-required | rpc-basics |
| TC-024 | [transaction-count-increments](../sources/chainbench/tests/tc/go-stablenet/regression/api/05-transaction-count-increments.json) | exact-config | port-required | rpc-basics |
| TC-025 | [txpool-status](../sources/chainbench/tests/tc/go-stablenet/regression/api/18-txpool-status.json) | exact-config | port-required | txpool |
| TC-026 | [txpool-content-well-formed](../sources/chainbench/tests/tc/go-stablenet/regression/api/19-txpool-content-well-formed.json) | exact-config | port-required | txpool |
| TC-027 | [fee-delegate-sign-rpc-present](../sources/chainbench/tests/tc/go-stablenet/regression/api/21-fee-delegate-sign-rpc-present.json) | adapter-required | port-required | fee-delegation |
| TC-028 | [legacy-transfer](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/08-legacy-transfer.json) | adapter-required | port-required | transaction-types |
| TC-029 | [dynamic-fee-tx](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/09-dynamic-fee-tx.json) | adapter-required | port-required | transaction-types |
| TC-030 | [access-list-tx](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/10-access-list-tx.json) | exact-config | port-required | transaction-types |
| TC-031 | [nonce-ordering](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/11-nonce-ordering.json) | exact-config | port-required | nonce-replacement |
| TC-032 | [out-of-order-nonces-mine](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json) | exact-config | port-required | nonce-replacement |
| TC-033 | [dynamic-fee-below-basefee-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/12-dynamic-fee-below-basefee-rejected.json) | adapter-required | port-required | fee-policy |
| TC-034 | [insufficient-funds-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/14-insufficient-funds-rejected.json) | exact-config | port-required | invalid-transaction |
| TC-035 | [gas-limit-exceeds-block-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/15-gas-limit-exceeds-block-rejected.json) | exact-config | port-required | invalid-transaction |
| TC-036 | [effective-gas-price](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/16-effective-gas-price.json) | exact-config | port-required | fee-observation |
| TC-037 | [replacement-tx](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/17-replacement-tx.json) | exact-config | port-required | nonce-replacement |
| TC-038 | [same-nonce-replacement](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/17b-same-nonce-replacement.json) | exact-config | port-required | nonce-replacement |
| TC-039 | [contract-roundtrip](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/19-contract-roundtrip.json) | exact-config | port-required | evm-contract |
| TC-040 | [eth-call-revert-returns-error](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json) | exact-config | port-required | evm-contract |
| TC-041 | [revert-tx-status-zero](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/24-revert-tx-status-zero.json) | exact-config | port-required | evm-contract |
| TC-042 | [out-of-gas-consumes-all](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json) | exact-config | port-required | evm-contract |
| TC-043 | [value-transfer](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/27b-value-transfer.json) | exact-config | port-required | transaction-types |
| TC-044 | [contract-event-emitted](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/36-contract-event-emitted.json) | exact-config | port-required | evm-contract |
| TC-045 | [fee-delegated-transfer](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json) | adapter-required | port-required | fee-delegation |
| TC-046 | [fd-sender-sig-invalid-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json) | adapter-required | port-required | fee-delegation |
| TC-047 | [fd-feepayer-sig-invalid-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json) | adapter-required | port-required | fee-delegation |
| TC-048 | [feepayer-insufficient-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json) | adapter-required | port-required | fee-delegation |
| TC-049 | [fee-delegated-sender-sig-invalid-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json) | adapter-required | port-required | fee-delegation |
| TC-050 | [fee-delegated-feepayer-sig-invalid-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json) | adapter-required | port-required | fee-delegation |
| TC-051 | [fee-delegated-unfunded-feepayer-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json) | adapter-required | port-required | fee-delegation |
| TC-052 | [wemix-chain-up](../sources/chainbench/tests/tc/go-wemix/chain-up/01-wemix-chain-up.json) | adapter-required | existing-case | sync-lifecycle |
| TC-053 | [wemix-chain-up-15](../sources/chainbench/tests/tc/go-wemix/chain-up/02-wemix-chain-up-15.json) | adapter-required | existing-case | sync-lifecycle |
| TC-054 | [wemix-node-crash](../sources/chainbench/tests/tc/go-wemix/fault/01-wemix-node-crash.json) | adapter-required | existing-case | fault-topology |
| TC-055 | [wemix-brioche-block-reward](../sources/chainbench/tests/tc/go-wemix/rpc/01-wemix-brioche-block-reward.json) | chain-specific | existing-case | reward-compatibility |
| TC-056 | [wemix-tx-and-contract](../sources/chainbench/tests/tc/go-wemix/tx/01-wemix-tx-and-contract.json) | exact-config | existing-case | evm-contract |
| TC-057 | [stress-block-time](../sources/chainbench/tests/tc/stress/01-stress-block-time.json) | adapter-required | port-required | stress |
| TC-058 | [stress-tx-flood](../sources/chainbench/tests/tc/stress/02-stress-tx-flood.json) | adapter-required | port-required | stress |
| TC-059 | [gas-price-positive](../sources/chainbench/tests/tc/go-stablenet/regression/api/07-gas-price-positive.json) | exact-config | port-required | fee-observation |
| TC-060 | [fee-history-well-formed](../sources/chainbench/tests/tc/go-stablenet/regression/api/09-fee-history-well-formed.json) | exact-config | port-required | fee-observation |
| TC-061 | [admin-peers-populated](../sources/chainbench/tests/tc/go-stablenet/regression/api/20-admin-peers-populated.json) | adapter-required | port-required | sync-lifecycle |
| TC-062 | [chain-not-syncing](../sources/chainbench/tests/tc/go-stablenet/regression/api/24-chain-not-syncing.json) | adapter-required | port-required | sync-lifecycle |
| TC-063 | [stablenet-chain-up](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/01-stablenet-chain-up.json) | adapter-required | port-required | sync-lifecycle |
| TC-064 | [estimate-gas](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/22-estimate-gas.json) | exact-config | port-required | evm-contract |
| TC-065 | [genesis-balance](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/27-genesis-balance.json) | exact-config | port-required | rpc-basics |
| TC-066 | [logs-query-well-formed](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/29-logs-query-well-formed.json) | exact-config | port-required | logs-subscription |
| TC-067 | [chain-id](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/30-chain-id.json) | exact-config | port-required | rpc-basics |
| TC-068 | [ws-subscribe-new-heads](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/31-ws-subscribe-new-heads.json) | exact-config | port-required | logs-subscription |
| TC-069 | [ws-subscribe-logs](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/32-ws-subscribe-logs.json) | exact-config | port-required | logs-subscription |
| TC-070 | [stablenet-chain-up-15](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/33-stablenet-chain-up-15.json) | adapter-required | port-required | sync-lifecycle |
| TC-071 | [stablenet-proxied-pn-routing](../sources/chainbench/tests/tc/go-stablenet/topology/01-proxied-pn-routing.json) | adapter-required | conditional | fault-topology |
| TC-072 | [stablenet-negative-tx-revert](../sources/chainbench/tests/tc/go-stablenet/tx/01-negative-tx-revert.json) | exact-config | port-required | evm-contract |
| TC-073 | [wbft-chain-up](../sources/chainbench/tests/tc/go-wbft/chain-up/01-wbft-chain-up.json) | adapter-required | port-required | sync-lifecycle |
| TC-074 | [wbft-chain-up-15](../sources/chainbench/tests/tc/go-wbft/chain-up/02-wbft-chain-up-15.json) | adapter-required | port-required | sync-lifecycle |
| TC-075 | [e1-mixed-producers](../sources/chainbench/tests/tc/go-wbft/consensus/01-e1-mixed-producers.json) | chain-specific | port-required | wbft-specific |
| TC-076 | [wbft-node-crash](../sources/chainbench/tests/tc/go-wbft/fault/01-wbft-node-crash.json) | adapter-required | port-required | fault-topology |
| TC-077 | [wbft-tx-and-contract](../sources/chainbench/tests/tc/go-wbft/tx/01-wbft-tx-and-contract.json) | exact-config | port-required | evm-contract |
| TC-078 | [wemix-wbft-handoff](../sources/chainbench/tests/tc/go-wemix/handoff/01-wemix-wbft-handoff.json) | chain-specific | conditional | handoff-specific |
| TC-079 | [remote-rpc-health](../sources/chainbench/tests/tc/remote/01-remote-rpc-health.json) | exact-config | port-required | rpc-basics |
| TC-080 | [remote-chain-info](../sources/chainbench/tests/tc/remote/02-remote-chain-info.json) | exact-config | port-required | rpc-basics |
| TC-081 | [remote-balance-check](../sources/chainbench/tests/tc/remote/03-remote-balance-check.json) | exact-config | port-required | rpc-basics |
| TC-082 | [sample-minimal-value-transfer](../sources/chainbench/tests/tc/samples/01-sample-minimal.json) | exact-config | port-required | transaction-types |
| TC-083 | [sample-lifecycle-node-restart](../sources/chainbench/tests/tc/samples/02-sample-lifecycle.json) | adapter-required | port-required | fault-topology |
| TC-084 | [stablenet-derived-vocabulary](../sources/chainbench/tests/tc/go-stablenet/vocabulary/01-derived-address-and-checksum.json) | exact-config | port-required | harness-vocabulary |
| TC-085 | [stablenet-faucet-funds](../sources/chainbench/tests/tc/go-stablenet/vocabulary/02-faucet-funds-account.json) | unknown | conditional | external-faucet |
| TC-086 | [stablenet-register-contract](../sources/chainbench/tests/tc/go-stablenet/vocabulary/03-register-contract.json) | exact-config | port-required | evm-contract |
| TC-087 | [basic-wbft-consensus](../sources/chainbench/tests/tc/basic/07-basic-wbft-consensus.json) | chain-specific | not-applicable | stablenet-specific |
| TC-088 | [stablenet-delayed-fork](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json) | chain-specific | not-applicable | stablenet-specific |
| TC-089 | [govminter-v2-code](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json) | chain-specific | not-applicable | stablenet-specific |
| TC-090 | [burn-cancel-refundable](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json) | chain-specific | not-applicable | stablenet-specific |
| TC-091 | [burn-reject-refundable](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json) | chain-specific | not-applicable | stablenet-specific |
| TC-092 | [burn-expire-refundable](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json) | chain-specific | not-applicable | stablenet-specific |
| TC-093 | [burn-execute-no-refundable](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json) | chain-specific | not-applicable | stablenet-specific |
| TC-094 | [claim-burn-refund-succeeds](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json) | chain-specific | not-applicable | stablenet-specific |
| TC-095 | [claim-zero-refund-reverts](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json) | chain-specific | not-applicable | stablenet-specific |
| TC-096 | [claim-burn-refund-double-reverts](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json) | chain-specific | not-applicable | stablenet-specific |
| TC-097 | [prealloc-preserved-across-boho](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json) | chain-specific | not-applicable | stablenet-specific |
| TC-098 | [boho-chain-config-active](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/15-boho-chain-config-active.json) | chain-specific | not-applicable | stablenet-specific |
| TC-099 | [anzeon-active-before-boho](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/16-anzeon-active-before-boho.json) | chain-specific | not-applicable | stablenet-specific |
| TC-100 | [estimategas-authorizationlist-cost](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json) | chain-specific | not-applicable | modern-evm-specific |
| TC-101 | [upgrade-registry-order](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json) | chain-specific | not-applicable | stablenet-specific |
| TC-102 | [v1-params-init-storage](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json) | chain-specific | not-applicable | stablenet-specific |
| TC-103 | [burn-refund-events](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json) | chain-specific | not-applicable | stablenet-specific |
| TC-104 | [effective-gas-price-authorized-bp-en](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json) | chain-specific | not-applicable | stablenet-specific |
| TC-105 | [effective-gas-price-regular](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json) | adapter-required | port-required | fee-observation |
| TC-106 | [auth-tx-event-last-bp-en](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json) | chain-specific | not-applicable | stablenet-specific |
| TC-107 | [authorized-extra-bit-synced](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/01-authorized-extra-bit-synced.json) | chain-specific | not-applicable | stablenet-specific |
| TC-108 | [blacklisted-extra-bit-synced](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/01b-blacklisted-extra-bit-synced.json) | chain-specific | not-applicable | stablenet-specific |
| TC-109 | [stablenet-account-extra](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json) | chain-specific | not-applicable | stablenet-specific |
| TC-110 | [extra-union-merge](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json) | chain-specific | not-applicable | stablenet-specific |
| TC-111 | [dual-status-extra](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json) | chain-specific | not-applicable | stablenet-specific |
| TC-112 | [extra-balance-preserved](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/05-extra-balance-preserved.json) | chain-specific | not-applicable | stablenet-specific |
| TC-113 | [invalid-extra-reject](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json) | chain-specific | not-applicable | stablenet-specific |
| TC-114 | [extra-state-across-delayed-boho](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json) | chain-specific | not-applicable | stablenet-specific |
| TC-115 | [unsupported-system-contract-version](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json) | chain-specific | not-applicable | stablenet-specific |
| TC-116 | [authorized-accounts-no-space](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/01-authorized-accounts-no-space.json) | chain-specific | not-applicable | stablenet-specific |
| TC-117 | [authorized-accounts-space](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/02-authorized-accounts-space.json) | chain-specific | not-applicable | stablenet-specific |
| TC-118 | [authorized-accounts-trim](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/03-authorized-accounts-trim.json) | chain-specific | not-applicable | stablenet-specific |
| TC-119 | [authorized-accounts-empty-item](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/04-authorized-accounts-empty-item.json) | chain-specific | not-applicable | stablenet-specific |
| TC-120 | [authorized-accounts-single](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/05-authorized-accounts-single.json) | chain-specific | not-applicable | stablenet-specific |
| TC-121 | [authorized-accounts-empty](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/06-authorized-accounts-empty.json) | chain-specific | not-applicable | stablenet-specific |
| TC-122 | [regular-account-gastip-forced](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json) | chain-specific | not-applicable | stablenet-specific |
| TC-123 | [authorized-account-gastip-free](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json) | chain-specific | not-applicable | stablenet-specific |
| TC-124 | [anzeon-basefee-increase](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json) | adapter-required | port-required | fee-policy |
| TC-125 | [anzeon-basefee-stable](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json) | adapter-required | port-required | fee-policy |
| TC-126 | [anzeon-basefee-decrease](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json) | adapter-required | port-required | fee-policy |
| TC-127 | [basefee-minimum](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/06-basefee-minimum.json) | adapter-required | port-required | fee-policy |
| TC-128 | [basefee-maximum](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/07-basefee-maximum.json) | chain-specific | port-required | fee-policy |
| TC-129 | [feecap-above-min-accepted](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json) | adapter-required | port-required | fee-policy |
| TC-130 | [feecap-exact-min-accepted](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json) | adapter-required | port-required | fee-policy |
| TC-131 | [gaslimit-exceeded-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json) | adapter-required | port-required | invalid-transaction |
| TC-132 | [system-contracts-deployed](../sources/chainbench/tests/tc/go-stablenet/regression/api/06-system-contracts-deployed.json) | chain-specific | not-applicable | stablenet-specific |
| TC-133 | [gas-price-equals-basefee-plus-tip](../sources/chainbench/tests/tc/go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json) | adapter-required | port-required | fee-observation |
| TC-134 | [max-priority-fee-equals-gastip](../sources/chainbench/tests/tc/go-stablenet/regression/api/08-max-priority-fee-equals-gastip.json) | chain-specific | not-applicable | stablenet-specific |
| TC-135 | [estimate-gas-token-transfer](../sources/chainbench/tests/tc/go-stablenet/regression/api/10-estimate-gas-token-transfer.json) | chain-specific | not-applicable | stablenet-specific |
| TC-136 | [node-address-returned](../sources/chainbench/tests/tc/go-stablenet/regression/api/11-node-address-returned.json) | chain-specific | not-applicable | wbft-specific |
| TC-137 | [validator-set-nonempty](../sources/chainbench/tests/tc/go-stablenet/regression/api/12-validator-set-nonempty.json) | chain-specific | not-applicable | wbft-specific |
| TC-138 | [validator-set-count](../sources/chainbench/tests/tc/go-stablenet/regression/api/12b-validator-set-count.json) | chain-specific | not-applicable | wbft-specific |
| TC-139 | [commit-signers-quorum](../sources/chainbench/tests/tc/go-stablenet/regression/api/13-commit-signers-quorum.json) | chain-specific | not-applicable | wbft-specific |
| TC-140 | [wbft-extra-info-fields](../sources/chainbench/tests/tc/go-stablenet/regression/api/14-wbft-extra-info-fields.json) | chain-specific | not-applicable | wbft-specific |
| TC-141 | [istanbul-status-fields](../sources/chainbench/tests/tc/go-stablenet/regression/api/15-istanbul-status-fields.json) | chain-specific | not-applicable | wbft-specific |
| TC-142 | [is-validator-flags](../sources/chainbench/tests/tc/go-stablenet/regression/api/16-is-validator-flags.json) | chain-specific | not-applicable | wbft-specific |
| TC-143 | [token-total-supply-readable](../sources/chainbench/tests/tc/go-stablenet/regression/api/22-token-total-supply-readable.json) | chain-specific | not-applicable | stablenet-specific |
| TC-144 | [token-approve-sets-allowance](../sources/chainbench/tests/tc/go-stablenet/regression/api/23-token-approve-sets-allowance.json) | chain-specific | not-applicable | stablenet-specific |
| TC-145 | [sender-blacklisted-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json) | chain-specific | not-applicable | stablenet-specific |
| TC-146 | [recipient-blacklisted-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json) | chain-specific | not-applicable | stablenet-specific |
| TC-147 | [feepayer-blacklisted-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json) | chain-specific | not-applicable | stablenet-specific |
| TC-148 | [address-unblacklisted-event](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json) | chain-specific | not-applicable | stablenet-specific |
| TC-149 | [zero-address-transfer-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json) | chain-specific | not-applicable | stablenet-specific |
| TC-150 | [precompile-transfer-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json) | chain-specific | not-applicable | stablenet-specific |
| TC-151 | [account-blacklist-readable](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/07-account-blacklist-readable.json) | chain-specific | not-applicable | stablenet-specific |
| TC-152 | [account-authorization-readable](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/08-account-authorization-readable.json) | chain-specific | not-applicable | stablenet-specific |
| TC-153 | [authorized-tx-executed-event](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json) | chain-specific | not-applicable | stablenet-specific |
| TC-154 | [set-code-delegation](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/18-set-code-delegation.json) | chain-specific | not-applicable | modern-evm-specific |
| TC-155 | [native-coin-adapter-code](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/01-native-coin-adapter-code.json) | chain-specific | not-applicable | stablenet-specific |
| TC-156 | [token-transfer-emits-event](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json) | chain-specific | not-applicable | stablenet-specific |
| TC-157 | [token-balance-readable](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/02-token-balance-readable.json) | chain-specific | not-applicable | stablenet-specific |
| TC-158 | [token-transfer-from-moves-balance](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json) | chain-specific | not-applicable | stablenet-specific |
| TC-159 | [mint-transfer-event](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/04-mint-transfer-event.json) | chain-specific | not-applicable | stablenet-specific |
| TC-160 | [burn-transfer-event](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/05-burn-transfer-event.json) | chain-specific | not-applicable | stablenet-specific |
| TC-161 | [mint-proposal-executes](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/06-mint-proposal-executes.json) | chain-specific | not-applicable | stablenet-specific |
| TC-162 | [burn-proposal-executes](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/07-burn-proposal-executes.json) | chain-specific | not-applicable | stablenet-specific |
| TC-163 | [quorum-deficient-stays-voting](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json) | chain-specific | not-applicable | stablenet-specific |
| TC-164 | [validator-metadata-readable](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/09-validator-metadata-readable.json) | chain-specific | not-applicable | stablenet-specific |
| TC-165 | [gastip-governance-updates-header](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json) | chain-specific | not-applicable | stablenet-specific |
| TC-166 | [proposal-expiry-transitions](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json) | chain-specific | not-applicable | stablenet-specific |
| TC-167 | [configure-minter-proposal-executes](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json) | chain-specific | not-applicable | stablenet-specific |
| TC-168 | [remove-minter-executes](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/13-remove-minter-executes.json) | chain-specific | not-applicable | stablenet-specific |
| TC-169 | [masterminter-member-add-remove](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json) | chain-specific | not-applicable | stablenet-specific |
| TC-170 | [non-member-configure-minter-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json) | chain-specific | not-applicable | stablenet-specific |
| TC-171 | [blacklist-proposal-executes](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json) | chain-specific | not-applicable | stablenet-specific |
| TC-172 | [authorize-proposal-executes](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json) | chain-specific | not-applicable | stablenet-specific |
| TC-173 | [unauthorize-proposal-executes](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json) | chain-specific | not-applicable | stablenet-specific |
| TC-174 | [direct-blacklist-call-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json) | chain-specific | not-applicable | stablenet-specific |
| TC-175 | [authorized-account-added-event](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/23-authorized-account-added-event.json) | chain-specific | not-applicable | stablenet-specific |
| TC-176 | [token-metadata](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/25-token-metadata.json) | chain-specific | not-applicable | stablenet-specific |
| TC-177 | [minter-status-readable](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/28-minter-status-readable.json) | chain-specific | not-applicable | stablenet-specific |
| TC-178 | [block-period-one-second](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/01-block-period-one-second.json) | adapter-required | port-required | sync-lifecycle |
| TC-179 | [wbft-seals-quorum](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/02-wbft-seals-quorum.json) | chain-specific | not-applicable | wbft-specific |
| TC-180 | [epoch-transition-carries-epoch-info](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json) | chain-specific | not-applicable | wbft-specific |
| TC-181 | [validator-add-member-executes](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/04-validator-add-member-executes.json) | chain-specific | not-applicable | wbft-specific |
| TC-182 | [validator-add-member-epoch-activates](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json) | chain-specific | not-applicable | wbft-specific |
| TC-183 | [validator-remove-member-executes](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/05-validator-remove-member-executes.json) | chain-specific | not-applicable | wbft-specific |
| TC-184 | [prev-seals-quorum](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/11-prev-seals-quorum.json) | chain-specific | not-applicable | wbft-specific |
| TC-185 | [randao-and-mixdigest-present](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/13-randao-and-mixdigest-present.json) | chain-specific | not-applicable | wbft-specific |
| TC-186 | [stablenet-gastip-field](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/14-stablenet-gastip-field.json) | chain-specific | not-applicable | wbft-specific |
| TC-187 | [secp256r1-precompile-valid](../sources/chainbench/tests/tc/go-wbft/accounts/01-secp256r1-precompile-valid.json) | chain-specific | not-applicable | modern-evm-specific |
| TC-188 | [secp256r1-precompile-invalid](../sources/chainbench/tests/tc/go-wbft/accounts/02-secp256r1-precompile-invalid.json) | chain-specific | not-applicable | modern-evm-specific |
| TC-189 | [secp256r1-precompile-short-input](../sources/chainbench/tests/tc/go-wbft/accounts/03-secp256r1-precompile-short-input.json) | chain-specific | not-applicable | modern-evm-specific |

의존 필드 정밀 재검토: JSON의 정확한 key와 step 문맥으로 분류했다. schemaVersion/description은 제외하며, 일반 sendTx의 to는 수신 계정으로 기록한다. env.capabilities/requires/applicableChains는 별도 capability_dependencies에 두었다. source_fields와 field_bindings에 literal/binding/role/config-reference를 구분하고 실제 값은 생략했다. 빈 목록은 환경 상속 또는 사전 의존이 없다는 뜻이 아니다. 세부 행은 all-tests.json을 참조한다.
