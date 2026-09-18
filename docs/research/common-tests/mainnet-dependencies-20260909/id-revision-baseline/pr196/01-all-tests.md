# 전체 원본 테스트 목록

문서 시나리오 74행과 JSON 파일 189개, 총 263개 원본 항목을 보존했다. 서로 겹치는 기능이 있으므로 독립된 테스트 263개라는 뜻은 아니다. 문서의 한 행이 참조하는 복수 ID와 스크립트는 document-catalog.json에 원문 그대로 있다.

| 관리 ID | 원본/테스트 | 우선순위 | go-wemix 적용 |
|---|---|---|---|
| DOC-C-001 | [Legacy Tx (Type 0x0) 전송 및 실행](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| DOC-C-002 | [Dynamic Fee Tx (Type 0x2, EIP-1559)](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| DOC-C-003 | [Access List Tx (Type 0x1, EIP-2930)](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| DOC-C-004 | [수수료 대납 Tx (Type 0x16) 정상 처리](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| DOC-C-005 | [대납 Tx Sender 서명 변조 거부](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| DOC-C-006 | [대납 Tx FeePayer 서명 변조 거부](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| DOC-C-007 | [FeePayer 잔액 부족시 실행 실패](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| DOC-C-008 | [Nonce 순서 보장 (Nonce Ordering)](../sources/common_test_scenarios.md) | P0 | 명세 이식 |
| DOC-C-009 | [TipCap 미달 (Underpriced) 거부](../sources/common_test_scenarios.md) | P1 | 조건부 |
| DOC-C-010 | [잔액 부족 (Insufficient Funds) 거부](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| DOC-C-011 | [Gas Limit 초과 거부](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| DOC-C-012 | [Effective GasPrice 노드 간 일관성](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| DOC-C-013 | [Queued/Pending 트랜잭션 교체 (Replacement Tx)](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| DOC-C-014 | [스마트 컨트랙트 배포](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| DOC-C-015 | [컨트랙트 상태 변경 함수 호출](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| DOC-C-016 | [`eth_call` View/Pure 함수 조회](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| DOC-C-017 | [`eth_estimateGas` 가스 추정](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| DOC-C-018 | [`eth_call` 실행 중 Revert 메세지 반환](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| DOC-C-019 | [Revert 트랜잭션 (상태 롤백 & 가스 환불)](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| DOC-C-020 | [Out-of-Gas 트랜잭션 (가스 Limit 전액 소모)](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| DOC-C-021 | [`eth_blockNumber`](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| DOC-C-022 | [`eth_getBalance`](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| DOC-C-023 | [`eth_sendRawTransaction`](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| DOC-C-024 | [`eth_getLogs`](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| DOC-C-025 | [`eth_chainId`](../sources/common_test_scenarios.md) | P3 | 명세 이식 |
| DOC-C-026 | [WebSocket `eth_subscribe("newHeads")`](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| DOC-C-027 | [WebSocket `eth_subscribe("logs")`](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| DOC-C-028 | [`eth_getBlockByNumber`](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| DOC-C-029 | [`eth_getBlockByHash`](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| DOC-C-030 | [`eth_getTransactionByHash`](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| DOC-C-031 | [`eth_getTransactionReceipt`](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| DOC-C-032 | [`eth_getTransactionCount`](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| DOC-C-033 | [`eth_gasPrice`](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| DOC-C-034 | [`eth_maxPriorityFeePerGas`](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| DOC-C-035 | [`eth_feeHistory`](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| DOC-C-036 | [`txpool_status`](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| DOC-C-037 | [`txpool_content`](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| DOC-C-038 | [FeePayer 서명 RPC 헬스체크 (`eth_signRawFeeDelegateTransaction`)](../sources/common_test_scenarios.md) | P3 | 명세 이식 |
| DOC-C-039 | [제네시스 초기화 (Genesis Initialization)](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| DOC-C-040 | [Full Sync 동기화](../sources/common_test_scenarios.md) | P0 | 명세 이식 |
| DOC-C-041 | [Snap Sync 동기화](../sources/common_test_scenarios.md) | P1 | 조건부 |
| DOC-C-042 | [노드 재기동 후 동기화 유지](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| DOC-C-043 | [P2P Bootnode 피어 연결](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| DOC-C-044 | [Block Downloader 경로 동기화](../sources/common_test_scenarios.md) | P0 | 명세 이식 |
| DOC-C-045 | [Block Fetcher 경로 동기화](../sources/common_test_scenarios.md) | P0 | 명세 이식 |
| DOC-D-001 | [`wemix_getBriocheBlockReward` 하위 호환성](../sources/dual_chain_test_scenarios.md) | P2 | 조건부 |
| DOC-D-002 | [블록 생산 주기 1초](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| DOC-D-003 | [블록 Finalize 및 CommittedSeal 존재](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| DOC-D-004 | [에폭 전환 시 검증자 집합 갱신](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| DOC-D-005 | [블록 헤더 WBFTExtra 필드 반영](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| DOC-D-006 | [View Change / 라운드 체인지](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| DOC-D-007 | [라운드 체인지 후 블록 연결](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| DOC-D-008 | [RoundRobin Proposer 정책](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| DOC-D-009 | [PrevCommittedSeal 수집](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| DOC-D-010 | [PrevPreparedSeal 수집](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| DOC-D-011 | [RandaoReveal / MixDigest](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| DOC-D-012 | [쿼럼 미달 블록 수락 거부](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| DOC-D-013 | [1/3 미만 장애 시 합의 지속](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| DOC-D-014 | [1/3 이상 장애 시 합의 중단](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| DOC-D-015 | [쿼럼 계산 — validator 3개 (전원 필요)](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| DOC-D-016 | [쿼럼 계산 — validator 6개, 1개 장애](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| DOC-D-017 | [쿼럼 계산 — validator 6개, 2개 장애](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| DOC-D-018 | [`istanbul_nodeAddress`](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| DOC-D-019 | [`istanbul_getValidators`](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| DOC-D-020 | [`istanbul_getCommitSignersFromBlock`](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| DOC-D-021 | [`istanbul_getWbftExtraInfo`](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| DOC-D-022 | [`istanbul_status`](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| DOC-D-023 | [`istanbul_isValidator`](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| DOC-D-024 | [SetCode Tx (Type 0x4, EIP-7702)](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| DOC-D-025 | [EIP-7702 AuthorizationList estimateGas 비용](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| DOC-D-026 | [EIP-7702 estimateGas baseline](../sources/dual_chain_test_scenarios.md) | P2 | 조건부 |
| DOC-D-027 | [secp256r1 프리컴파일 — 유효 서명](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| DOC-D-028 | [secp256r1 프리컴파일 — 무효 서명](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| DOC-D-029 | [secp256r1 프리컴파일 — 잘못된 입력 길이](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| TC-001 | [basic-consensus](../sources/chainbench/tests/tc/basic/01-basic-consensus.json) | P1 | JSON 수정 필요 |
| TC-002 | [basic-peers](../sources/chainbench/tests/tc/basic/02-basic-peers.json) | P1 | JSON 수정 필요 |
| TC-003 | [basic-rpc-health](../sources/chainbench/tests/tc/basic/03-basic-rpc-health.json) | P1 | JSON 수정 필요 |
| TC-004 | [basic-sync](../sources/chainbench/tests/tc/basic/04-basic-sync.json) | P1 | JSON 수정 필요 |
| TC-005 | [basic-tx-send](../sources/chainbench/tests/tc/basic/05-basic-tx-send.json) | P1 | JSON 수정 필요 |
| TC-006 | [basic-txpool-propagation](../sources/chainbench/tests/tc/basic/06-basic-txpool-propagation.json) | P1 | JSON 수정 필요 |
| TC-007 | [fault-network-partition](../sources/chainbench/tests/tc/fault/01-fault-network-partition.json) | P1 | JSON 수정 필요 |
| TC-008 | [fault-node-crash](../sources/chainbench/tests/tc/fault/02-fault-node-crash.json) | P1 | JSON 수정 필요 |
| TC-009 | [fault-node-recover](../sources/chainbench/tests/tc/fault/03-fault-node-recover.json) | P1 | JSON 수정 필요 |
| TC-010 | [fault-p2p-topology](../sources/chainbench/tests/tc/fault/04-fault-p2p-topology.json) | P1 | JSON 수정 필요 |
| TC-011 | [fault-two-down](../sources/chainbench/tests/tc/fault/05-fault-two-down.json) | P1 | JSON 수정 필요 |
| TC-012 | [fault-txpool-leader-change](../sources/chainbench/tests/tc/fault/06-fault-txpool-leader-change.json) | P1 | JSON 수정 필요 |
| TC-013 | [legacy-gasprice-below-min-rejected](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/12-legacy-gasprice-below-min-rejected.json) | P1 | JSON 수정 필요 |
| TC-014 | [accesslist-gasprice-below-min-rejected](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/13-accesslist-gasprice-below-min-rejected.json) | P1 | JSON 수정 필요 |
| TC-015 | [feecap-below-min-rejected](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/14-feecap-below-min-rejected.json) | P1 | JSON 수정 필요 |
| TC-016 | [effective-gas-price-regular-bp-en](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json) | P1 | JSON 수정 필요 |
| TC-017 | [signature-compat-across-swap](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json) | P1 | JSON 수정 필요 |
| TC-018 | [genesis-mismatch-refuses-to-start](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json) | P1 | JSON 수정 필요 |
| TC-019 | [genesis-block-hash-consistent](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/04-genesis-block-hash-consistent.json) | P1 | JSON 수정 필요 |
| TC-020 | [block-transactions-field](../sources/chainbench/tests/tc/go-stablenet/regression/api/01-block-transactions-field.json) | P1 | JSON 수정 필요 |
| TC-021 | [block-by-hash-consistency](../sources/chainbench/tests/tc/go-stablenet/regression/api/02-block-by-hash-consistency.json) | P1 | JSON 수정 필요 |
| TC-022 | [transaction-by-hash-fields](../sources/chainbench/tests/tc/go-stablenet/regression/api/03-transaction-by-hash-fields.json) | P1 | JSON 수정 필요 |
| TC-023 | [transaction-receipt-fields](../sources/chainbench/tests/tc/go-stablenet/regression/api/04-transaction-receipt-fields.json) | P1 | JSON 수정 필요 |
| TC-024 | [transaction-count-increments](../sources/chainbench/tests/tc/go-stablenet/regression/api/05-transaction-count-increments.json) | P1 | JSON 수정 필요 |
| TC-025 | [txpool-status](../sources/chainbench/tests/tc/go-stablenet/regression/api/18-txpool-status.json) | P1 | JSON 수정 필요 |
| TC-026 | [txpool-content-well-formed](../sources/chainbench/tests/tc/go-stablenet/regression/api/19-txpool-content-well-formed.json) | P1 | JSON 수정 필요 |
| TC-027 | [fee-delegate-sign-rpc-present](../sources/chainbench/tests/tc/go-stablenet/regression/api/21-fee-delegate-sign-rpc-present.json) | P3 | JSON 수정 필요 |
| TC-028 | [legacy-transfer](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/08-legacy-transfer.json) | P1 | JSON 수정 필요 |
| TC-029 | [dynamic-fee-tx](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/09-dynamic-fee-tx.json) | P1 | JSON 수정 필요 |
| TC-030 | [access-list-tx](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/10-access-list-tx.json) | P1 | JSON 수정 필요 |
| TC-031 | [nonce-ordering](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/11-nonce-ordering.json) | P1 | JSON 수정 필요 |
| TC-032 | [out-of-order-nonces-mine](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json) | P1 | JSON 수정 필요 |
| TC-033 | [dynamic-fee-below-basefee-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/12-dynamic-fee-below-basefee-rejected.json) | P1 | JSON 수정 필요 |
| TC-034 | [insufficient-funds-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/14-insufficient-funds-rejected.json) | P1 | JSON 수정 필요 |
| TC-035 | [gas-limit-exceeds-block-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/15-gas-limit-exceeds-block-rejected.json) | P1 | JSON 수정 필요 |
| TC-036 | [effective-gas-price](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/16-effective-gas-price.json) | P1 | JSON 수정 필요 |
| TC-037 | [replacement-tx](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/17-replacement-tx.json) | P1 | JSON 수정 필요 |
| TC-038 | [same-nonce-replacement](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/17b-same-nonce-replacement.json) | P1 | JSON 수정 필요 |
| TC-039 | [contract-roundtrip](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/19-contract-roundtrip.json) | P1 | JSON 수정 필요 |
| TC-040 | [eth-call-revert-returns-error](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json) | P1 | JSON 수정 필요 |
| TC-041 | [revert-tx-status-zero](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/24-revert-tx-status-zero.json) | P1 | JSON 수정 필요 |
| TC-042 | [out-of-gas-consumes-all](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json) | P1 | JSON 수정 필요 |
| TC-043 | [value-transfer](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/27b-value-transfer.json) | P1 | JSON 수정 필요 |
| TC-044 | [contract-event-emitted](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/36-contract-event-emitted.json) | P1 | JSON 수정 필요 |
| TC-045 | [fee-delegated-transfer](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json) | P1 | JSON 수정 필요 |
| TC-046 | [fd-sender-sig-invalid-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json) | P1 | JSON 수정 필요 |
| TC-047 | [fd-feepayer-sig-invalid-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json) | P1 | JSON 수정 필요 |
| TC-048 | [feepayer-insufficient-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json) | P1 | JSON 수정 필요 |
| TC-049 | [fee-delegated-sender-sig-invalid-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json) | P1 | JSON 수정 필요 |
| TC-050 | [fee-delegated-feepayer-sig-invalid-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json) | P1 | JSON 수정 필요 |
| TC-051 | [fee-delegated-unfunded-feepayer-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json) | P1 | JSON 수정 필요 |
| TC-052 | [wemix-chain-up](../sources/chainbench/tests/tc/go-wemix/chain-up/01-wemix-chain-up.json) | P1 | 그대로 실행 후보 |
| TC-053 | [wemix-chain-up-15](../sources/chainbench/tests/tc/go-wemix/chain-up/02-wemix-chain-up-15.json) | P1 | 그대로 실행 후보 |
| TC-054 | [wemix-node-crash](../sources/chainbench/tests/tc/go-wemix/fault/01-wemix-node-crash.json) | P1 | 그대로 실행 후보 |
| TC-055 | [wemix-brioche-block-reward](../sources/chainbench/tests/tc/go-wemix/rpc/01-wemix-brioche-block-reward.json) | P2 | 그대로 실행 후보 |
| TC-056 | [wemix-tx-and-contract](../sources/chainbench/tests/tc/go-wemix/tx/01-wemix-tx-and-contract.json) | P1 | 그대로 실행 후보 |
| TC-057 | [stress-block-time](../sources/chainbench/tests/tc/stress/01-stress-block-time.json) | P1 | JSON 수정 필요 |
| TC-058 | [stress-tx-flood](../sources/chainbench/tests/tc/stress/02-stress-tx-flood.json) | P1 | JSON 수정 필요 |
| TC-059 | [gas-price-positive](../sources/chainbench/tests/tc/go-stablenet/regression/api/07-gas-price-positive.json) | P2 | JSON 수정 필요 |
| TC-060 | [fee-history-well-formed](../sources/chainbench/tests/tc/go-stablenet/regression/api/09-fee-history-well-formed.json) | P2 | JSON 수정 필요 |
| TC-061 | [admin-peers-populated](../sources/chainbench/tests/tc/go-stablenet/regression/api/20-admin-peers-populated.json) | P2 | JSON 수정 필요 |
| TC-062 | [chain-not-syncing](../sources/chainbench/tests/tc/go-stablenet/regression/api/24-chain-not-syncing.json) | P2 | JSON 수정 필요 |
| TC-063 | [stablenet-chain-up](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/01-stablenet-chain-up.json) | P2 | JSON 수정 필요 |
| TC-064 | [estimate-gas](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/22-estimate-gas.json) | P2 | JSON 수정 필요 |
| TC-065 | [genesis-balance](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/27-genesis-balance.json) | P2 | JSON 수정 필요 |
| TC-066 | [logs-query-well-formed](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/29-logs-query-well-formed.json) | P2 | JSON 수정 필요 |
| TC-067 | [chain-id](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/30-chain-id.json) | P3 | JSON 수정 필요 |
| TC-068 | [ws-subscribe-new-heads](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/31-ws-subscribe-new-heads.json) | P2 | JSON 수정 필요 |
| TC-069 | [ws-subscribe-logs](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/32-ws-subscribe-logs.json) | P2 | JSON 수정 필요 |
| TC-070 | [stablenet-chain-up-15](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/33-stablenet-chain-up-15.json) | P2 | JSON 수정 필요 |
| TC-071 | [stablenet-proxied-pn-routing](../sources/chainbench/tests/tc/go-stablenet/topology/01-proxied-pn-routing.json) | P2 | 조건부 |
| TC-072 | [stablenet-negative-tx-revert](../sources/chainbench/tests/tc/go-stablenet/tx/01-negative-tx-revert.json) | P2 | JSON 수정 필요 |
| TC-073 | [wbft-chain-up](../sources/chainbench/tests/tc/go-wbft/chain-up/01-wbft-chain-up.json) | P2 | JSON 수정 필요 |
| TC-074 | [wbft-chain-up-15](../sources/chainbench/tests/tc/go-wbft/chain-up/02-wbft-chain-up-15.json) | P2 | JSON 수정 필요 |
| TC-075 | [e1-mixed-producers](../sources/chainbench/tests/tc/go-wbft/consensus/01-e1-mixed-producers.json) | P2 | JSON 수정 필요 |
| TC-076 | [wbft-node-crash](../sources/chainbench/tests/tc/go-wbft/fault/01-wbft-node-crash.json) | P2 | JSON 수정 필요 |
| TC-077 | [wbft-tx-and-contract](../sources/chainbench/tests/tc/go-wbft/tx/01-wbft-tx-and-contract.json) | P2 | JSON 수정 필요 |
| TC-078 | [wemix-wbft-handoff](../sources/chainbench/tests/tc/go-wemix/handoff/01-wemix-wbft-handoff.json) | P2 | 조건부 |
| TC-079 | [remote-rpc-health](../sources/chainbench/tests/tc/remote/01-remote-rpc-health.json) | P2 | JSON 수정 필요 |
| TC-080 | [remote-chain-info](../sources/chainbench/tests/tc/remote/02-remote-chain-info.json) | P2 | JSON 수정 필요 |
| TC-081 | [remote-balance-check](../sources/chainbench/tests/tc/remote/03-remote-balance-check.json) | P2 | JSON 수정 필요 |
| TC-082 | [sample-minimal-value-transfer](../sources/chainbench/tests/tc/samples/01-sample-minimal.json) | P2 | JSON 수정 필요 |
| TC-083 | [sample-lifecycle-node-restart](../sources/chainbench/tests/tc/samples/02-sample-lifecycle.json) | P2 | JSON 수정 필요 |
| TC-084 | [stablenet-derived-vocabulary](../sources/chainbench/tests/tc/go-stablenet/vocabulary/01-derived-address-and-checksum.json) | P3 | JSON 수정 필요 |
| TC-085 | [stablenet-faucet-funds](../sources/chainbench/tests/tc/go-stablenet/vocabulary/02-faucet-funds-account.json) | P3 | 조건부 |
| TC-086 | [stablenet-register-contract](../sources/chainbench/tests/tc/go-stablenet/vocabulary/03-register-contract.json) | P3 | JSON 수정 필요 |
| TC-087 | [basic-wbft-consensus](../sources/chainbench/tests/tc/basic/07-basic-wbft-consensus.json) | P3 | 제외 |
| TC-088 | [stablenet-delayed-fork](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json) | P3 | 제외 |
| TC-089 | [govminter-v2-code](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json) | P3 | 제외 |
| TC-090 | [burn-cancel-refundable](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json) | P3 | 제외 |
| TC-091 | [burn-reject-refundable](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json) | P3 | 제외 |
| TC-092 | [burn-expire-refundable](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json) | P3 | 제외 |
| TC-093 | [burn-execute-no-refundable](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json) | P3 | 제외 |
| TC-094 | [claim-burn-refund-succeeds](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json) | P3 | 제외 |
| TC-095 | [claim-zero-refund-reverts](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json) | P3 | 제외 |
| TC-096 | [claim-burn-refund-double-reverts](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json) | P3 | 제외 |
| TC-097 | [prealloc-preserved-across-boho](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json) | P3 | 제외 |
| TC-098 | [boho-chain-config-active](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/15-boho-chain-config-active.json) | P3 | 제외 |
| TC-099 | [anzeon-active-before-boho](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/16-anzeon-active-before-boho.json) | P3 | 제외 |
| TC-100 | [estimategas-authorizationlist-cost](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json) | P3 | 제외 |
| TC-101 | [upgrade-registry-order](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json) | P3 | 제외 |
| TC-102 | [v1-params-init-storage](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json) | P3 | 제외 |
| TC-103 | [burn-refund-events](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json) | P3 | 제외 |
| TC-104 | [effective-gas-price-authorized-bp-en](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json) | P3 | 제외 |
| TC-105 | [effective-gas-price-regular](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json) | P1 | JSON 수정 필요 |
| TC-106 | [auth-tx-event-last-bp-en](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json) | P3 | 제외 |
| TC-107 | [authorized-extra-bit-synced](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/01-authorized-extra-bit-synced.json) | P3 | 제외 |
| TC-108 | [blacklisted-extra-bit-synced](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/01b-blacklisted-extra-bit-synced.json) | P3 | 제외 |
| TC-109 | [stablenet-account-extra](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json) | P3 | 제외 |
| TC-110 | [extra-union-merge](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json) | P3 | 제외 |
| TC-111 | [dual-status-extra](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json) | P3 | 제외 |
| TC-112 | [extra-balance-preserved](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/05-extra-balance-preserved.json) | P3 | 제외 |
| TC-113 | [invalid-extra-reject](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json) | P3 | 제외 |
| TC-114 | [extra-state-across-delayed-boho](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json) | P3 | 제외 |
| TC-115 | [unsupported-system-contract-version](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json) | P3 | 제외 |
| TC-116 | [authorized-accounts-no-space](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/01-authorized-accounts-no-space.json) | P3 | 제외 |
| TC-117 | [authorized-accounts-space](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/02-authorized-accounts-space.json) | P3 | 제외 |
| TC-118 | [authorized-accounts-trim](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/03-authorized-accounts-trim.json) | P3 | 제외 |
| TC-119 | [authorized-accounts-empty-item](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/04-authorized-accounts-empty-item.json) | P3 | 제외 |
| TC-120 | [authorized-accounts-single](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/05-authorized-accounts-single.json) | P3 | 제외 |
| TC-121 | [authorized-accounts-empty](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/06-authorized-accounts-empty.json) | P3 | 제외 |
| TC-122 | [regular-account-gastip-forced](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json) | P3 | 제외 |
| TC-123 | [authorized-account-gastip-free](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json) | P3 | 제외 |
| TC-124 | [anzeon-basefee-increase](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json) | P1 | JSON 수정 필요 |
| TC-125 | [anzeon-basefee-stable](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json) | P1 | JSON 수정 필요 |
| TC-126 | [anzeon-basefee-decrease](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json) | P1 | JSON 수정 필요 |
| TC-127 | [basefee-minimum](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/06-basefee-minimum.json) | P1 | JSON 수정 필요 |
| TC-128 | [basefee-maximum](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/07-basefee-maximum.json) | P1 | JSON 수정 필요 |
| TC-129 | [feecap-above-min-accepted](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json) | P1 | JSON 수정 필요 |
| TC-130 | [feecap-exact-min-accepted](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json) | P1 | JSON 수정 필요 |
| TC-131 | [gaslimit-exceeded-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json) | P1 | JSON 수정 필요 |
| TC-132 | [system-contracts-deployed](../sources/chainbench/tests/tc/go-stablenet/regression/api/06-system-contracts-deployed.json) | P3 | 제외 |
| TC-133 | [gas-price-equals-basefee-plus-tip](../sources/chainbench/tests/tc/go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json) | P2 | JSON 수정 필요 |
| TC-134 | [max-priority-fee-equals-gastip](../sources/chainbench/tests/tc/go-stablenet/regression/api/08-max-priority-fee-equals-gastip.json) | P3 | 제외 |
| TC-135 | [estimate-gas-token-transfer](../sources/chainbench/tests/tc/go-stablenet/regression/api/10-estimate-gas-token-transfer.json) | P3 | 제외 |
| TC-136 | [node-address-returned](../sources/chainbench/tests/tc/go-stablenet/regression/api/11-node-address-returned.json) | P3 | 제외 |
| TC-137 | [validator-set-nonempty](../sources/chainbench/tests/tc/go-stablenet/regression/api/12-validator-set-nonempty.json) | P3 | 제외 |
| TC-138 | [validator-set-count](../sources/chainbench/tests/tc/go-stablenet/regression/api/12b-validator-set-count.json) | P3 | 제외 |
| TC-139 | [commit-signers-quorum](../sources/chainbench/tests/tc/go-stablenet/regression/api/13-commit-signers-quorum.json) | P3 | 제외 |
| TC-140 | [wbft-extra-info-fields](../sources/chainbench/tests/tc/go-stablenet/regression/api/14-wbft-extra-info-fields.json) | P3 | 제외 |
| TC-141 | [istanbul-status-fields](../sources/chainbench/tests/tc/go-stablenet/regression/api/15-istanbul-status-fields.json) | P3 | 제외 |
| TC-142 | [is-validator-flags](../sources/chainbench/tests/tc/go-stablenet/regression/api/16-is-validator-flags.json) | P3 | 제외 |
| TC-143 | [token-total-supply-readable](../sources/chainbench/tests/tc/go-stablenet/regression/api/22-token-total-supply-readable.json) | P3 | 제외 |
| TC-144 | [token-approve-sets-allowance](../sources/chainbench/tests/tc/go-stablenet/regression/api/23-token-approve-sets-allowance.json) | P3 | 제외 |
| TC-145 | [sender-blacklisted-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json) | P3 | 제외 |
| TC-146 | [recipient-blacklisted-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json) | P3 | 제외 |
| TC-147 | [feepayer-blacklisted-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json) | P3 | 제외 |
| TC-148 | [address-unblacklisted-event](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json) | P3 | 제외 |
| TC-149 | [zero-address-transfer-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json) | P3 | 제외 |
| TC-150 | [precompile-transfer-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json) | P3 | 제외 |
| TC-151 | [account-blacklist-readable](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/07-account-blacklist-readable.json) | P3 | 제외 |
| TC-152 | [account-authorization-readable](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/08-account-authorization-readable.json) | P3 | 제외 |
| TC-153 | [authorized-tx-executed-event](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json) | P3 | 제외 |
| TC-154 | [set-code-delegation](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/18-set-code-delegation.json) | P3 | 제외 |
| TC-155 | [native-coin-adapter-code](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/01-native-coin-adapter-code.json) | P3 | 제외 |
| TC-156 | [token-transfer-emits-event](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json) | P3 | 제외 |
| TC-157 | [token-balance-readable](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/02-token-balance-readable.json) | P3 | 제외 |
| TC-158 | [token-transfer-from-moves-balance](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json) | P3 | 제외 |
| TC-159 | [mint-transfer-event](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/04-mint-transfer-event.json) | P3 | 제외 |
| TC-160 | [burn-transfer-event](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/05-burn-transfer-event.json) | P3 | 제외 |
| TC-161 | [mint-proposal-executes](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/06-mint-proposal-executes.json) | P3 | 제외 |
| TC-162 | [burn-proposal-executes](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/07-burn-proposal-executes.json) | P3 | 제외 |
| TC-163 | [quorum-deficient-stays-voting](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json) | P3 | 제외 |
| TC-164 | [validator-metadata-readable](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/09-validator-metadata-readable.json) | P3 | 제외 |
| TC-165 | [gastip-governance-updates-header](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json) | P3 | 제외 |
| TC-166 | [proposal-expiry-transitions](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json) | P3 | 제외 |
| TC-167 | [configure-minter-proposal-executes](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json) | P3 | 제외 |
| TC-168 | [remove-minter-executes](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/13-remove-minter-executes.json) | P3 | 제외 |
| TC-169 | [masterminter-member-add-remove](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json) | P3 | 제외 |
| TC-170 | [non-member-configure-minter-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json) | P3 | 제외 |
| TC-171 | [blacklist-proposal-executes](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json) | P3 | 제외 |
| TC-172 | [authorize-proposal-executes](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json) | P3 | 제외 |
| TC-173 | [unauthorize-proposal-executes](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json) | P3 | 제외 |
| TC-174 | [direct-blacklist-call-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json) | P3 | 제외 |
| TC-175 | [authorized-account-added-event](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/23-authorized-account-added-event.json) | P3 | 제외 |
| TC-176 | [token-metadata](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/25-token-metadata.json) | P3 | 제외 |
| TC-177 | [minter-status-readable](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/28-minter-status-readable.json) | P3 | 제외 |
| TC-178 | [block-period-one-second](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/01-block-period-one-second.json) | P2 | JSON 수정 필요 |
| TC-179 | [wbft-seals-quorum](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/02-wbft-seals-quorum.json) | P3 | 제외 |
| TC-180 | [epoch-transition-carries-epoch-info](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json) | P3 | 제외 |
| TC-181 | [validator-add-member-executes](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/04-validator-add-member-executes.json) | P3 | 제외 |
| TC-182 | [validator-add-member-epoch-activates](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json) | P3 | 제외 |
| TC-183 | [validator-remove-member-executes](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/05-validator-remove-member-executes.json) | P3 | 제외 |
| TC-184 | [prev-seals-quorum](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/11-prev-seals-quorum.json) | P3 | 제외 |
| TC-185 | [randao-and-mixdigest-present](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/13-randao-and-mixdigest-present.json) | P3 | 제외 |
| TC-186 | [stablenet-gastip-field](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/14-stablenet-gastip-field.json) | P3 | 제외 |
| TC-187 | [secp256r1-precompile-valid](../sources/chainbench/tests/tc/go-wbft/accounts/01-secp256r1-precompile-valid.json) | P3 | 제외 |
| TC-188 | [secp256r1-precompile-invalid](../sources/chainbench/tests/tc/go-wbft/accounts/02-secp256r1-precompile-invalid.json) | P3 | 제외 |
| TC-189 | [secp256r1-precompile-short-input](../sources/chainbench/tests/tc/go-wbft/accounts/03-secp256r1-precompile-short-input.json) | P3 | 제외 |
