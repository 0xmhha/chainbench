# 전체 원본 테스트 목록

> ID 표기는 Confluence 원래 명세 ID를 우선한다. 실행 ID는 별도 표기한다. ID 대응은 전체 명세 충족이나 PASS를 뜻하지 않는다. [ID 대응표](/Users/wm-it-25_0220/Work/github/chainbench/docs/research/common-tests/mainnet-dependencies-20260909/analyses/existing-tc-specs.md)

문서 시나리오 74행과 JSON 파일 189개, 총 263개 원본 항목을 보존했다. 서로 겹치는 기능이 있으므로 독립된 테스트 263개라는 뜻은 아니다. 문서의 한 행이 참조하는 복수 ID와 스크립트는 document-catalog.json에 원문 그대로 있다.

| 관리 ID | 원본/테스트 | 우선순위 | go-wemix 적용 |
|---|---|---|---|
| RT-A-2-01 / TX-006 | [Legacy Tx (Type 0x0) 전송 및 실행](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| RT-A-2-02 / TX-003 | [Dynamic Fee Tx (Type 0x2, EIP-1559)](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| RT-A-2-03 / TX-007 | [Access List Tx (Type 0x1, EIP-2930)](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| RT-D-01 / TX-004 | [수수료 대납 Tx (Type 0x16) 정상 처리](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| RT-D-03 / TX-014 | [대납 Tx Sender 서명 변조 거부](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| RT-D-04 / TX-015 | [대납 Tx FeePayer 서명 변조 거부](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| RT-D-05 / TX-016 | [FeePayer 잔액 부족시 실행 실패](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| RT-A-2-04 / TX-010 | [Nonce 순서 보장 (Nonce Ordering)](../sources/common_test_scenarios.md) | P0 | 명세 이식 |
| RT-A-2-05a | [TipCap 미달 (Underpriced) 거부](../sources/common_test_scenarios.md) | P1 | 조건부 |
| RT-A-2-06 / TX-011 | [잔액 부족 (Insufficient Funds) 거부](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| RT-A-2-07 / TX-012 | [Gas Limit 초과 거부](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| RT-A-2-08 | [Effective GasPrice 노드 간 일관성](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| RT-A-2-09 / TX-013 | [Queued/Pending 트랜잭션 교체 (Replacement Tx)](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| RT-A-3-01 / TX-005 | [스마트 컨트랙트 배포](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| RT-A-3-02 | [컨트랙트 상태 변경 함수 호출](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| RT-A-3-03 | [`eth_call` View/Pure 함수 조회](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| RT-A-3-04 | [`eth_estimateGas` 가스 추정](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| RT-A-3-05 | [`eth_call` 실행 중 Revert 메세지 반환](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| RT-A-3-06 / TX-017 | [Revert 트랜잭션 (상태 롤백 & 가스 환불)](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| RT-A-3-07 / TX-018 | [Out-of-Gas 트랜잭션 (가스 Limit 전액 소모)](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| RT-A-4-01 / RPC-001 | [`eth_blockNumber`](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| RT-A-4-02 / RPC-012 | [`eth_getBalance`](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| RT-A-4-03 | [`eth_sendRawTransaction`](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| RT-A-4-04 / RPC-014 | [`eth_getLogs`](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| RT-A-4-05 / RPC-013 | [`eth_chainId`](../sources/common_test_scenarios.md) | P3 | 명세 이식 |
| RT-A-4-06 / RPC-020 | [WebSocket `eth_subscribe("newHeads")`](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| RT-A-4-07 / RPC-021 | [WebSocket `eth_subscribe("logs")`](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| RT-G-1-01 / RPC-002 | [`eth_getBlockByNumber`](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| RT-G-1-02 / RPC-002 | [`eth_getBlockByHash`](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| RT-G-1-03 | [`eth_getTransactionByHash`](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| RT-G-1-04 / RPC-007 | [`eth_getTransactionReceipt`](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| RT-G-1-05 / RPC-015 | [`eth_getTransactionCount`](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| RT-G-2-01 / RPC-016 | [`eth_gasPrice`](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| RT-G-2-02 | [`eth_maxPriorityFeePerGas`](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| RT-G-2-03 / RPC-017 | [`eth_feeHistory`](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| RT-G-4-02 / RPC-018 | [`txpool_status`](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| RT-G-4-03 | [`txpool_content`](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| RT-G-5-01 | [FeePayer 서명 RPC 헬스체크 (`eth_signRawFeeDelegateTransaction`)](../sources/common_test_scenarios.md) | P3 | 명세 이식 |
| RT-A-1-01 | [제네시스 초기화 (Genesis Initialization)](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| RT-A-1-02 / NODE-003 | [Full Sync 동기화](../sources/common_test_scenarios.md) | P0 | 명세 이식 |
| RT-A-1-03 / NODE-004 | [Snap Sync 동기화](../sources/common_test_scenarios.md) | P1 | 조건부 |
| RT-A-1-04 / NODE-005 | [노드 재기동 후 동기화 유지](../sources/common_test_scenarios.md) | P1 | 명세 이식 |
| RT-A-1-05 | [P2P Bootnode 피어 연결](../sources/common_test_scenarios.md) | P2 | 명세 이식 |
| RT-A-1-06 | [Block Downloader 경로 동기화](../sources/common_test_scenarios.md) | P0 | 명세 이식 |
| RT-A-1-07 | [Block Fetcher 경로 동기화](../sources/common_test_scenarios.md) | P0 | 명세 이식 |
| RPC-008 | [`wemix_getBriocheBlockReward` 하위 호환성](../sources/dual_chain_test_scenarios.md) | P2 | 조건부 |
| RT-B-01 / WBFT-002 | [블록 생산 주기 1초](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| RT-B-02 / WBFT-001 | [블록 Finalize 및 CommittedSeal 존재](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| RT-B-03 / WBFT-005 | [에폭 전환 시 검증자 집합 갱신](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| RT-B-06 | [블록 헤더 WBFTExtra 필드 반영](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| RT-B-09 / WBFT-003 | [View Change / 라운드 체인지](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| RT-B-10 / WBFT-003 | [라운드 체인지 후 블록 연결](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| RT-B-06 / WBFT-006 | [RoundRobin Proposer 정책](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| RT-B-11 / WBFT-009 | [PrevCommittedSeal 수집](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| RT-B-12 / WBFT-009 | [PrevPreparedSeal 수집](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| WBFT-010 | [RandaoReveal / MixDigest](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| RT-B-08 | [쿼럼 미달 블록 수락 거부](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| WBFT-007 | [1/3 미만 장애 시 합의 지속](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| WBFT-008 | [1/3 이상 장애 시 합의 중단](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| WBFT-011 | [쿼럼 계산 — validator 3개 (전원 필요)](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| WBFT-012 | [쿼럼 계산 — validator 6개, 1개 장애](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| WBFT-013 | [쿼럼 계산 — validator 6개, 2개 장애](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| RT-G-3-01 / RPC-010 | [`istanbul_nodeAddress`](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| RT-G-3-02 / RPC-003 | [`istanbul_getValidators`](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| RT-G-3-03 / RPC-004 | [`istanbul_getCommitSignersFromBlock`](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| RT-G-3-04 / RPC-005 | [`istanbul_getWbftExtraInfo`](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| RT-G-3-05 / RPC-006 | [`istanbul_status`](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| RT-G-3-06 / RPC-011 | [`istanbul_isValidator`](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| RT-A-2-10 / TX-008 | [SetCode Tx (Type 0x4, EIP-7702)](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| TC-4-2-01 / TC-4-2-03 | [EIP-7702 AuthorizationList estimateGas 비용](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| TC-4-2-02 | [EIP-7702 estimateGas baseline](../sources/dual_chain_test_scenarios.md) | P2 | 조건부 |
| TC-1-2-01 / TC-1-2-02 / TC-1-2-03 / TC-1-2-04 / TC-1-2-05 / TC-1-2-06 / TX-009 | [secp256r1 프리컴파일 — 유효 서명](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| TC-1-2-01 / TC-1-2-02 / TC-1-2-03 / TC-1-2-04 / TC-1-2-05 / TC-1-2-06 / TX-019 | [secp256r1 프리컴파일 — 무효 서명](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| TC-1-2-01 / TC-1-2-02 / TC-1-2-03 / TC-1-2-04 / TC-1-2-05 / TC-1-2-06 / TX-020 | [secp256r1 프리컴파일 — 잘못된 입력 길이](../sources/dual_chain_test_scenarios.md) | P3 | 제외 |
| basic-consensus [명세 ID 미확인] | [basic-consensus](../sources/chainbench/tests/tc/basic/01-basic-consensus.json) | P1 | JSON 수정 필요 |
| basic-peers [명세 ID 미확인] | [basic-peers](../sources/chainbench/tests/tc/basic/02-basic-peers.json) | P1 | JSON 수정 필요 |
| basic-rpc-health [명세 ID 미확인] | [basic-rpc-health](../sources/chainbench/tests/tc/basic/03-basic-rpc-health.json) | P1 | JSON 수정 필요 |
| basic-sync [명세 ID 미확인] | [basic-sync](../sources/chainbench/tests/tc/basic/04-basic-sync.json) | P1 | JSON 수정 필요 |
| basic-tx-send [명세 ID 미확인] | [basic-tx-send](../sources/chainbench/tests/tc/basic/05-basic-tx-send.json) | P1 | JSON 수정 필요 |
| basic-txpool-propagation [명세 ID 미확인] | [basic-txpool-propagation](../sources/chainbench/tests/tc/basic/06-basic-txpool-propagation.json) | P1 | JSON 수정 필요 |
| fault-network-partition [명세 ID 미확인] | [fault-network-partition](../sources/chainbench/tests/tc/fault/01-fault-network-partition.json) | P1 | JSON 수정 필요 |
| fault-node-crash [명세 ID 미확인] | [fault-node-crash](../sources/chainbench/tests/tc/fault/02-fault-node-crash.json) | P1 | JSON 수정 필요 |
| fault-node-recover [명세 ID 미확인] | [fault-node-recover](../sources/chainbench/tests/tc/fault/03-fault-node-recover.json) | P1 | JSON 수정 필요 |
| fault-p2p-topology [명세 ID 미확인] | [fault-p2p-topology](../sources/chainbench/tests/tc/fault/04-fault-p2p-topology.json) | P1 | JSON 수정 필요 |
| fault-two-down [명세 ID 미확인] | [fault-two-down](../sources/chainbench/tests/tc/fault/05-fault-two-down.json) | P1 | JSON 수정 필요 |
| fault-txpool-leader-change [명세 ID 미확인] | [fault-txpool-leader-change](../sources/chainbench/tests/tc/fault/06-fault-txpool-leader-change.json) | P1 | JSON 수정 필요 |
| TC-1-3-04 <br> 실행: legacy-gasprice-below-min-rejected | [legacy-gasprice-below-min-rejected](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/12-legacy-gasprice-below-min-rejected.json) | P1 | JSON 수정 필요 |
| TC-1-3-05 <br> 실행: accesslist-gasprice-below-min-rejected | [accesslist-gasprice-below-min-rejected](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/13-accesslist-gasprice-below-min-rejected.json) | P1 | JSON 수정 필요 |
| TC-1-3-06 <br> 실행: feecap-below-min-rejected | [feecap-below-min-rejected](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/14-feecap-below-min-rejected.json) | P1 | JSON 수정 필요 |
| TC-4-6-02 <br> 실행: effective-gas-price-regular-bp-en | [effective-gas-price-regular-bp-en](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json) | P1 | JSON 수정 필요 |
| TC-3-1-04 <br> 실행: signature-compat-across-swap | [signature-compat-across-swap](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json) | P1 | JSON 수정 필요 |
| TC-4-1-03 <br> 실행: genesis-mismatch-refuses-to-start | [genesis-mismatch-refuses-to-start](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json) | P1 | JSON 수정 필요 |
| TC-5-3-01 <br> 실행: genesis-block-hash-consistent | [genesis-block-hash-consistent](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/04-genesis-block-hash-consistent.json) | P1 | JSON 수정 필요 |
| RT-G-1-01 <br> 실행: block-transactions-field | [block-transactions-field](../sources/chainbench/tests/tc/go-stablenet/regression/api/01-block-transactions-field.json) | P1 | JSON 수정 필요 |
| RT-G-1-02 <br> 실행: block-by-hash-consistency | [block-by-hash-consistency](../sources/chainbench/tests/tc/go-stablenet/regression/api/02-block-by-hash-consistency.json) | P1 | JSON 수정 필요 |
| RT-G-1-03 <br> 실행: transaction-by-hash-fields | [transaction-by-hash-fields](../sources/chainbench/tests/tc/go-stablenet/regression/api/03-transaction-by-hash-fields.json) | P1 | JSON 수정 필요 |
| RT-G-1-04 <br> 실행: transaction-receipt-fields | [transaction-receipt-fields](../sources/chainbench/tests/tc/go-stablenet/regression/api/04-transaction-receipt-fields.json) | P1 | JSON 수정 필요 |
| RT-G-1-05 <br> 실행: transaction-count-increments | [transaction-count-increments](../sources/chainbench/tests/tc/go-stablenet/regression/api/05-transaction-count-increments.json) | P1 | JSON 수정 필요 |
| RT-G-4-02 <br> 실행: txpool-status | [txpool-status](../sources/chainbench/tests/tc/go-stablenet/regression/api/18-txpool-status.json) | P1 | JSON 수정 필요 |
| RT-G-4-03 <br> 실행: txpool-content-well-formed | [txpool-content-well-formed](../sources/chainbench/tests/tc/go-stablenet/regression/api/19-txpool-content-well-formed.json) | P1 | JSON 수정 필요 |
| RT-G-5-01 <br> 실행: fee-delegate-sign-rpc-present | [fee-delegate-sign-rpc-present](../sources/chainbench/tests/tc/go-stablenet/regression/api/21-fee-delegate-sign-rpc-present.json) | P3 | JSON 수정 필요 |
| RT-A-2-01 <br> 실행: legacy-transfer | [legacy-transfer](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/08-legacy-transfer.json) | P1 | JSON 수정 필요 |
| RT-A-2-02 <br> 실행: dynamic-fee-tx | [dynamic-fee-tx](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/09-dynamic-fee-tx.json) | P1 | JSON 수정 필요 |
| RT-A-2-03 <br> 실행: access-list-tx | [access-list-tx](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/10-access-list-tx.json) | P1 | JSON 수정 필요 |
| RT-A-2-04 <br> 실행: nonce-ordering | [nonce-ordering](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/11-nonce-ordering.json) | P1 | JSON 수정 필요 |
| TX-010 [부분 대응] <br> 실행: out-of-order-nonces-mine | [out-of-order-nonces-mine](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json) | P1 | JSON 수정 필요 |
| RT-A-2-05a <br> 실행: dynamic-fee-below-basefee-rejected | [dynamic-fee-below-basefee-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/12-dynamic-fee-below-basefee-rejected.json) | P1 | JSON 수정 필요 |
| RT-A-2-06 <br> 실행: insufficient-funds-rejected | [insufficient-funds-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/14-insufficient-funds-rejected.json) | P1 | JSON 수정 필요 |
| RT-A-2-07 <br> 실행: gas-limit-exceeds-block-rejected | [gas-limit-exceeds-block-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/15-gas-limit-exceeds-block-rejected.json) | P1 | JSON 수정 필요 |
| RT-A-2-08 <br> 실행: effective-gas-price | [effective-gas-price](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/16-effective-gas-price.json) | P1 | JSON 수정 필요 |
| RT-A-2-09 <br> 실행: replacement-tx | [replacement-tx](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/17-replacement-tx.json) | P1 | JSON 수정 필요 |
| TX-013 [부분 대응] <br> 실행: same-nonce-replacement | [same-nonce-replacement](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/17b-same-nonce-replacement.json) | P1 | JSON 수정 필요 |
| RT-A-3-01 <br> 실행: contract-roundtrip | [contract-roundtrip](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/19-contract-roundtrip.json) | P1 | JSON 수정 필요 |
| RT-A-3-05 <br> 실행: eth-call-revert-returns-error | [eth-call-revert-returns-error](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json) | P1 | JSON 수정 필요 |
| RT-A-3-06 <br> 실행: revert-tx-status-zero | [revert-tx-status-zero](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/24-revert-tx-status-zero.json) | P1 | JSON 수정 필요 |
| RT-A-3-07 <br> 실행: out-of-gas-consumes-all | [out-of-gas-consumes-all](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json) | P1 | JSON 수정 필요 |
| RT-A-4-03 <br> 실행: value-transfer | [value-transfer](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/27b-value-transfer.json) | P1 | JSON 수정 필요 |
| RT-A-4-04 / RPC-014 [부분 대응] <br> 실행: contract-event-emitted | [contract-event-emitted](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/36-contract-event-emitted.json) | P1 | JSON 수정 필요 |
| RT-D-01 <br> 실행: fee-delegated-transfer | [fee-delegated-transfer](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json) | P1 | JSON 수정 필요 |
| RT-D-03 <br> 실행: fd-sender-sig-invalid-rejected | [fd-sender-sig-invalid-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json) | P1 | JSON 수정 필요 |
| RT-D-04 <br> 실행: fd-feepayer-sig-invalid-rejected | [fd-feepayer-sig-invalid-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json) | P1 | JSON 수정 필요 |
| RT-D-05 <br> 실행: feepayer-insufficient-rejected | [feepayer-insufficient-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json) | P1 | JSON 수정 필요 |
| RT-D-03 / TX-014 [부분 대응] <br> 실행: fee-delegated-sender-sig-invalid-rejected | [fee-delegated-sender-sig-invalid-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json) | P1 | JSON 수정 필요 |
| RT-D-04 / TX-015 [부분 대응] <br> 실행: fee-delegated-feepayer-sig-invalid-rejected | [fee-delegated-feepayer-sig-invalid-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json) | P1 | JSON 수정 필요 |
| RT-D-05 / TX-016 [부분 대응] <br> 실행: fee-delegated-unfunded-feepayer-rejected | [fee-delegated-unfunded-feepayer-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json) | P1 | JSON 수정 필요 |
| wemix-chain-up [명세 ID 미확인] | [wemix-chain-up](../sources/chainbench/tests/tc/go-wemix/chain-up/01-wemix-chain-up.json) | P1 | 그대로 실행 후보 |
| wemix-chain-up-15 [명세 ID 미확인] | [wemix-chain-up-15](../sources/chainbench/tests/tc/go-wemix/chain-up/02-wemix-chain-up-15.json) | P1 | 그대로 실행 후보 |
| wemix-node-crash [명세 ID 미확인] | [wemix-node-crash](../sources/chainbench/tests/tc/go-wemix/fault/01-wemix-node-crash.json) | P1 | 그대로 실행 후보 |
| BRIOCHE-02 / RPC-008 [부분 대응] <br> 실행: wemix-brioche-block-reward | [wemix-brioche-block-reward](../sources/chainbench/tests/tc/go-wemix/rpc/01-wemix-brioche-block-reward.json) | P2 | 그대로 실행 후보 |
| wemix-tx-and-contract [명세 ID 미확인] | [wemix-tx-and-contract](../sources/chainbench/tests/tc/go-wemix/tx/01-wemix-tx-and-contract.json) | P1 | 그대로 실행 후보 |
| stress-block-time [명세 ID 미확인] | [stress-block-time](../sources/chainbench/tests/tc/stress/01-stress-block-time.json) | P1 | JSON 수정 필요 |
| stress-tx-flood [명세 ID 미확인] | [stress-tx-flood](../sources/chainbench/tests/tc/stress/02-stress-tx-flood.json) | P1 | JSON 수정 필요 |
| RT-G-2-01 <br> 실행: gas-price-positive | [gas-price-positive](../sources/chainbench/tests/tc/go-stablenet/regression/api/07-gas-price-positive.json) | P2 | JSON 수정 필요 |
| RT-G-2-03 <br> 실행: fee-history-well-formed | [fee-history-well-formed](../sources/chainbench/tests/tc/go-stablenet/regression/api/09-fee-history-well-formed.json) | P2 | JSON 수정 필요 |
| RT-A-1-05 <br> 실행: admin-peers-populated | [admin-peers-populated](../sources/chainbench/tests/tc/go-stablenet/regression/api/20-admin-peers-populated.json) | P2 | JSON 수정 필요 |
| chain-not-syncing [명세 ID 미확인] | [chain-not-syncing](../sources/chainbench/tests/tc/go-stablenet/regression/api/24-chain-not-syncing.json) | P2 | JSON 수정 필요 |
| stablenet-chain-up [명세 ID 미확인] | [stablenet-chain-up](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/01-stablenet-chain-up.json) | P2 | JSON 수정 필요 |
| RT-A-3-04 <br> 실행: estimate-gas | [estimate-gas](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/22-estimate-gas.json) | P2 | JSON 수정 필요 |
| RT-A-4-02 <br> 실행: genesis-balance | [genesis-balance](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/27-genesis-balance.json) | P2 | JSON 수정 필요 |
| RT-A-4-04 <br> 실행: logs-query-well-formed | [logs-query-well-formed](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/29-logs-query-well-formed.json) | P2 | JSON 수정 필요 |
| RT-A-1-01 <br> 실행: chain-id | [chain-id](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/30-chain-id.json) | P3 | JSON 수정 필요 |
| RT-A-4-06 <br> 실행: ws-subscribe-new-heads | [ws-subscribe-new-heads](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/31-ws-subscribe-new-heads.json) | P2 | JSON 수정 필요 |
| RT-A-4-07 <br> 실행: ws-subscribe-logs | [ws-subscribe-logs](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/32-ws-subscribe-logs.json) | P2 | JSON 수정 필요 |
| stablenet-chain-up-15 [명세 ID 미확인] | [stablenet-chain-up-15](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/33-stablenet-chain-up-15.json) | P2 | JSON 수정 필요 |
| stablenet-proxied-pn-routing [명세 ID 미확인] | [stablenet-proxied-pn-routing](../sources/chainbench/tests/tc/go-stablenet/topology/01-proxied-pn-routing.json) | P2 | 조건부 |
| stablenet-negative-tx-revert [명세 ID 미확인] | [stablenet-negative-tx-revert](../sources/chainbench/tests/tc/go-stablenet/tx/01-negative-tx-revert.json) | P2 | JSON 수정 필요 |
| wbft-chain-up [명세 ID 미확인] | [wbft-chain-up](../sources/chainbench/tests/tc/go-wbft/chain-up/01-wbft-chain-up.json) | P2 | JSON 수정 필요 |
| wbft-chain-up-15 [명세 ID 미확인] | [wbft-chain-up-15](../sources/chainbench/tests/tc/go-wbft/chain-up/02-wbft-chain-up-15.json) | P2 | JSON 수정 필요 |
| e1-mixed-producers [명세 ID 미확인] | [e1-mixed-producers](../sources/chainbench/tests/tc/go-wbft/consensus/01-e1-mixed-producers.json) | P2 | JSON 수정 필요 |
| wbft-node-crash [명세 ID 미확인] | [wbft-node-crash](../sources/chainbench/tests/tc/go-wbft/fault/01-wbft-node-crash.json) | P2 | JSON 수정 필요 |
| wbft-tx-and-contract [명세 ID 미확인] | [wbft-tx-and-contract](../sources/chainbench/tests/tc/go-wbft/tx/01-wbft-tx-and-contract.json) | P2 | JSON 수정 필요 |
| wemix-wbft-handoff [명세 ID 미확인] | [wemix-wbft-handoff](../sources/chainbench/tests/tc/go-wemix/handoff/01-wemix-wbft-handoff.json) | P2 | 조건부 |
| remote-rpc-health [명세 ID 미확인] | [remote-rpc-health](../sources/chainbench/tests/tc/remote/01-remote-rpc-health.json) | P2 | JSON 수정 필요 |
| remote-chain-info [명세 ID 미확인] | [remote-chain-info](../sources/chainbench/tests/tc/remote/02-remote-chain-info.json) | P2 | JSON 수정 필요 |
| remote-balance-check [명세 ID 미확인] | [remote-balance-check](../sources/chainbench/tests/tc/remote/03-remote-balance-check.json) | P2 | JSON 수정 필요 |
| sample-minimal-value-transfer [명세 ID 미확인] | [sample-minimal-value-transfer](../sources/chainbench/tests/tc/samples/01-sample-minimal.json) | P2 | JSON 수정 필요 |
| sample-lifecycle-node-restart [명세 ID 미확인] | [sample-lifecycle-node-restart](../sources/chainbench/tests/tc/samples/02-sample-lifecycle.json) | P2 | JSON 수정 필요 |
| stablenet-derived-vocabulary [명세 ID 미확인] | [stablenet-derived-vocabulary](../sources/chainbench/tests/tc/go-stablenet/vocabulary/01-derived-address-and-checksum.json) | P3 | JSON 수정 필요 |
| stablenet-faucet-funds [명세 ID 미확인] | [stablenet-faucet-funds](../sources/chainbench/tests/tc/go-stablenet/vocabulary/02-faucet-funds-account.json) | P3 | 조건부 |
| stablenet-register-contract [명세 ID 미확인] | [stablenet-register-contract](../sources/chainbench/tests/tc/go-stablenet/vocabulary/03-register-contract.json) | P3 | JSON 수정 필요 |
| basic-wbft-consensus [명세 ID 미확인] | [basic-wbft-consensus](../sources/chainbench/tests/tc/basic/07-basic-wbft-consensus.json) | P3 | 제외 |
| stablenet-delayed-fork [명세 ID 미확인] | [stablenet-delayed-fork](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json) | P3 | 제외 |
| TC-5-2-05 <br> 실행: govminter-v2-code | [govminter-v2-code](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json) | P3 | 제외 |
| TC-1-1-01 / TC-1-1-10 <br> 실행: burn-cancel-refundable | [burn-cancel-refundable](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json) | P3 | 제외 |
| TC-1-1-02 <br> 실행: burn-reject-refundable | [burn-reject-refundable](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json) | P3 | 제외 |
| TC-1-1-03 <br> 실행: burn-expire-refundable | [burn-expire-refundable](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json) | P3 | 제외 |
| TC-1-1-04 <br> 실행: burn-execute-no-refundable | [burn-execute-no-refundable](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json) | P3 | 제외 |
| TC-1-1-05 / TC-1-1-09 <br> 실행: claim-burn-refund-succeeds | [claim-burn-refund-succeeds](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json) | P3 | 제외 |
| TC-1-1-06 <br> 실행: claim-zero-refund-reverts | [claim-zero-refund-reverts](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json) | P3 | 제외 |
| TC-1-1-07 <br> 실행: claim-burn-refund-double-reverts | [claim-burn-refund-double-reverts](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json) | P3 | 제외 |
| TC-1-1-11 / TC-1-1-12 <br> 실행: prealloc-preserved-across-boho | [prealloc-preserved-across-boho](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json) | P3 | 제외 |
| TC-4-1-01 <br> 실행: boho-chain-config-active | [boho-chain-config-active](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/15-boho-chain-config-active.json) | P3 | 제외 |
| TC-4-1-02 <br> 실행: anzeon-active-before-boho | [anzeon-active-before-boho](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/16-anzeon-active-before-boho.json) | P3 | 제외 |
| TC-4-2-01 / TC-4-2-03 <br> 실행: estimategas-authorizationlist-cost | [estimategas-authorizationlist-cost](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json) | P3 | 제외 |
| TC-5-2-01 / TC-5-2-02 <br> 실행: upgrade-registry-order | [upgrade-registry-order](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json) | P3 | 제외 |
| TC-5-2-04 <br> 실행: v1-params-init-storage | [v1-params-init-storage](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json) | P3 | 제외 |
| TC-1-1-09 / TC-1-1-10 [부분 대응] <br> 실행: burn-refund-events | [burn-refund-events](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json) | P3 | 제외 |
| TC-4-6-01 <br> 실행: effective-gas-price-authorized-bp-en | [effective-gas-price-authorized-bp-en](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json) | P3 | 제외 |
| TC-4-6-02 <br> 실행: effective-gas-price-regular | [effective-gas-price-regular](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json) | P1 | JSON 수정 필요 |
| TC-4-6-04 <br> 실행: auth-tx-event-last-bp-en | [auth-tx-event-last-bp-en](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json) | P3 | 제외 |
| TC-4-5-01 / TC-4-5-02 <br> 실행: authorized-extra-bit-synced | [authorized-extra-bit-synced](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/01-authorized-extra-bit-synced.json) | P3 | 제외 |
| TC-4-5-02 [부분 대응] <br> 실행: blacklisted-extra-bit-synced | [blacklisted-extra-bit-synced](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/01b-blacklisted-extra-bit-synced.json) | P3 | 제외 |
| TC-4-5-01 / TC-4-5-02 / TC-4-5-07 [부분 대응] <br> 실행: stablenet-account-extra | [stablenet-account-extra](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json) | P3 | 제외 |
| TC-4-5-05 / TC-4-5-06 <br> 실행: extra-union-merge | [extra-union-merge](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json) | P3 | 제외 |
| TC-4-5-07 <br> 실행: dual-status-extra | [dual-status-extra](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json) | P3 | 제외 |
| TC-4-5-08 <br> 실행: extra-balance-preserved | [extra-balance-preserved](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/05-extra-balance-preserved.json) | P3 | 제외 |
| TC-4-5-09 <br> 실행: invalid-extra-reject | [invalid-extra-reject](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json) | P3 | 제외 |
| TC-4-5-10 / TC-4-5-11 <br> 실행: extra-state-across-delayed-boho | [extra-state-across-delayed-boho](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json) | P3 | 제외 |
| TC-5-2-06 <br> 실행: unsupported-system-contract-version | [unsupported-system-contract-version](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json) | P3 | 제외 |
| TC-4-3-01 <br> 실행: authorized-accounts-no-space | [authorized-accounts-no-space](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/01-authorized-accounts-no-space.json) | P3 | 제외 |
| TC-4-3-02 <br> 실행: authorized-accounts-space | [authorized-accounts-space](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/02-authorized-accounts-space.json) | P3 | 제외 |
| TC-4-3-03 <br> 실행: authorized-accounts-trim | [authorized-accounts-trim](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/03-authorized-accounts-trim.json) | P3 | 제외 |
| TC-4-3-04 <br> 실행: authorized-accounts-empty-item | [authorized-accounts-empty-item](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/04-authorized-accounts-empty-item.json) | P3 | 제외 |
| TC-4-3-05 <br> 실행: authorized-accounts-single | [authorized-accounts-single](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/05-authorized-accounts-single.json) | P3 | 제외 |
| TC-4-3-06 <br> 실행: authorized-accounts-empty | [authorized-accounts-empty](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/06-authorized-accounts-empty.json) | P3 | 제외 |
| RT-C-01 <br> 실행: regular-account-gastip-forced | [regular-account-gastip-forced](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json) | P3 | 제외 |
| RT-C-02 <br> 실행: authorized-account-gastip-free | [authorized-account-gastip-free](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json) | P3 | 제외 |
| RT-C-03 [부분 대응] <br> 실행: anzeon-basefee-increase | [anzeon-basefee-increase](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json) | P1 | JSON 수정 필요 |
| RT-C-04 [부분 대응] <br> 실행: anzeon-basefee-stable | [anzeon-basefee-stable](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json) | P1 | JSON 수정 필요 |
| RT-C-05 [부분 대응] <br> 실행: anzeon-basefee-decrease | [anzeon-basefee-decrease](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json) | P1 | JSON 수정 필요 |
| RT-C-06 <br> 실행: basefee-minimum | [basefee-minimum](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/06-basefee-minimum.json) | P1 | JSON 수정 필요 |
| RT-C-07 <br> 실행: basefee-maximum | [basefee-maximum](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/07-basefee-maximum.json) | P1 | JSON 수정 필요 |
| TC-1-3-03 [부분 대응] <br> 실행: feecap-above-min-accepted | [feecap-above-min-accepted](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json) | P1 | JSON 수정 필요 |
| TC-1-3-02 [부분 대응] <br> 실행: feecap-exact-min-accepted | [feecap-exact-min-accepted](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json) | P1 | JSON 수정 필요 |
| RT-A-2-07 / TX-012 [부분 대응] <br> 실행: gaslimit-exceeded-rejected | [gaslimit-exceeded-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json) | P1 | JSON 수정 필요 |
| RT-G-1-06 <br> 실행: system-contracts-deployed | [system-contracts-deployed](../sources/chainbench/tests/tc/go-stablenet/regression/api/06-system-contracts-deployed.json) | P3 | 제외 |
| RT-G-2-01 [부분 대응] <br> 실행: gas-price-equals-basefee-plus-tip | [gas-price-equals-basefee-plus-tip](../sources/chainbench/tests/tc/go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json) | P2 | JSON 수정 필요 |
| RT-G-2-02 <br> 실행: max-priority-fee-equals-gastip | [max-priority-fee-equals-gastip](../sources/chainbench/tests/tc/go-stablenet/regression/api/08-max-priority-fee-equals-gastip.json) | P3 | 제외 |
| RT-G-2-04 <br> 실행: estimate-gas-token-transfer | [estimate-gas-token-transfer](../sources/chainbench/tests/tc/go-stablenet/regression/api/10-estimate-gas-token-transfer.json) | P3 | 제외 |
| RT-G-3-01 <br> 실행: node-address-returned | [node-address-returned](../sources/chainbench/tests/tc/go-stablenet/regression/api/11-node-address-returned.json) | P3 | 제외 |
| RT-G-3-02 <br> 실행: validator-set-nonempty | [validator-set-nonempty](../sources/chainbench/tests/tc/go-stablenet/regression/api/12-validator-set-nonempty.json) | P3 | 제외 |
| RT-B-07 [부분 대응] <br> 실행: validator-set-count | [validator-set-count](../sources/chainbench/tests/tc/go-stablenet/regression/api/12b-validator-set-count.json) | P3 | 제외 |
| RT-G-3-03 <br> 실행: commit-signers-quorum | [commit-signers-quorum](../sources/chainbench/tests/tc/go-stablenet/regression/api/13-commit-signers-quorum.json) | P3 | 제외 |
| RT-G-3-04 <br> 실행: wbft-extra-info-fields | [wbft-extra-info-fields](../sources/chainbench/tests/tc/go-stablenet/regression/api/14-wbft-extra-info-fields.json) | P3 | 제외 |
| RT-G-3-05 <br> 실행: istanbul-status-fields | [istanbul-status-fields](../sources/chainbench/tests/tc/go-stablenet/regression/api/15-istanbul-status-fields.json) | P3 | 제외 |
| RT-G-3-06 <br> 실행: is-validator-flags | [is-validator-flags](../sources/chainbench/tests/tc/go-stablenet/regression/api/16-is-validator-flags.json) | P3 | 제외 |
| RT-G-5-02 <br> 실행: token-total-supply-readable | [token-total-supply-readable](../sources/chainbench/tests/tc/go-stablenet/regression/api/22-token-total-supply-readable.json) | P3 | 제외 |
| RT-G-5-03 <br> 실행: token-approve-sets-allowance | [token-approve-sets-allowance](../sources/chainbench/tests/tc/go-stablenet/regression/api/23-token-approve-sets-allowance.json) | P3 | 제외 |
| RT-E-01 <br> 실행: sender-blacklisted-rejected | [sender-blacklisted-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json) | P3 | 제외 |
| RT-E-02 <br> 실행: recipient-blacklisted-rejected | [recipient-blacklisted-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json) | P3 | 제외 |
| RT-E-03 <br> 실행: feepayer-blacklisted-rejected | [feepayer-blacklisted-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json) | P3 | 제외 |
| RT-E-04 <br> 실행: address-unblacklisted-event | [address-unblacklisted-event](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json) | P3 | 제외 |
| RT-E-05 <br> 실행: zero-address-transfer-rejected | [zero-address-transfer-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json) | P3 | 제외 |
| RT-E-06 <br> 실행: precompile-transfer-rejected | [precompile-transfer-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json) | P3 | 제외 |
| RT-E-07 <br> 실행: account-blacklist-readable | [account-blacklist-readable](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/07-account-blacklist-readable.json) | P3 | 제외 |
| RT-E-08 <br> 실행: account-authorization-readable | [account-authorization-readable](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/08-account-authorization-readable.json) | P3 | 제외 |
| RT-E-09 <br> 실행: authorized-tx-executed-event | [authorized-tx-executed-event](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json) | P3 | 제외 |
| RT-A-2-10 <br> 실행: set-code-delegation | [set-code-delegation](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/18-set-code-delegation.json) | P3 | 제외 |
| RT-F-1-01 <br> 실행: native-coin-adapter-code | [native-coin-adapter-code](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/01-native-coin-adapter-code.json) | P3 | 제외 |
| RT-F-1-01 [부분 대응] <br> 실행: token-transfer-emits-event | [token-transfer-emits-event](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json) | P3 | 제외 |
| RT-F-1-02 <br> 실행: token-balance-readable | [token-balance-readable](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/02-token-balance-readable.json) | P3 | 제외 |
| RT-F-1-03 [부분 대응] <br> 실행: token-transfer-from-moves-balance | [token-transfer-from-moves-balance](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json) | P3 | 제외 |
| RT-F-1-04 <br> 실행: mint-transfer-event | [mint-transfer-event](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/04-mint-transfer-event.json) | P3 | 제외 |
| RT-F-1-05 <br> 실행: burn-transfer-event | [burn-transfer-event](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/05-burn-transfer-event.json) | P3 | 제외 |
| RT-F-2-01 <br> 실행: mint-proposal-executes | [mint-proposal-executes](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/06-mint-proposal-executes.json) | P3 | 제외 |
| RT-F-2-02 <br> 실행: burn-proposal-executes | [burn-proposal-executes](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/07-burn-proposal-executes.json) | P3 | 제외 |
| RT-F-2-03 <br> 실행: quorum-deficient-stays-voting | [quorum-deficient-stays-voting](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json) | P3 | 제외 |
| RT-F-3-04 <br> 실행: validator-metadata-readable | [validator-metadata-readable](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/09-validator-metadata-readable.json) | P3 | 제외 |
| RT-B-06 <br> 실행: gastip-governance-updates-header | [gastip-governance-updates-header](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json) | P3 | 제외 |
| RT-F-3-06 <br> 실행: proposal-expiry-transitions | [proposal-expiry-transitions](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json) | P3 | 제외 |
| RT-F-4-01 <br> 실행: configure-minter-proposal-executes | [configure-minter-proposal-executes](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json) | P3 | 제외 |
| RT-F-4-02 <br> 실행: remove-minter-executes | [remove-minter-executes](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/13-remove-minter-executes.json) | P3 | 제외 |
| RT-F-4-03 <br> 실행: masterminter-member-add-remove | [masterminter-member-add-remove](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json) | P3 | 제외 |
| RT-F-4-04 <br> 실행: non-member-configure-minter-rejected | [non-member-configure-minter-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json) | P3 | 제외 |
| RT-F-5-01 <br> 실행: blacklist-proposal-executes | [blacklist-proposal-executes](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json) | P3 | 제외 |
| RT-F-5-03 <br> 실행: authorize-proposal-executes | [authorize-proposal-executes](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json) | P3 | 제외 |
| RT-F-5-04 / RT-F-5-09 [부분 대응] <br> 실행: unauthorize-proposal-executes | [unauthorize-proposal-executes](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json) | P3 | 제외 |
| RT-F-5-05 <br> 실행: direct-blacklist-call-rejected | [direct-blacklist-call-rejected](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json) | P3 | 제외 |
| RT-F-5-08 <br> 실행: authorized-account-added-event | [authorized-account-added-event](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/23-authorized-account-added-event.json) | P3 | 제외 |
| token-metadata [명세 ID 미확인] | [token-metadata](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/25-token-metadata.json) | P3 | 제외 |
| minter-status-readable [명세 ID 미확인] | [minter-status-readable](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/28-minter-status-readable.json) | P3 | 제외 |
| RT-B-01 <br> 실행: block-period-one-second | [block-period-one-second](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/01-block-period-one-second.json) | P2 | JSON 수정 필요 |
| RT-B-02 <br> 실행: wbft-seals-quorum | [wbft-seals-quorum](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/02-wbft-seals-quorum.json) | P3 | 제외 |
| RT-B-03 <br> 실행: epoch-transition-carries-epoch-info | [epoch-transition-carries-epoch-info](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json) | P3 | 제외 |
| RT-B-04 <br> 실행: validator-add-member-executes | [validator-add-member-executes](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/04-validator-add-member-executes.json) | P3 | 제외 |
| RT-B-04 <br> 실행: validator-add-member-epoch-activates | [validator-add-member-epoch-activates](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json) | P3 | 제외 |
| RT-B-05 <br> 실행: validator-remove-member-executes | [validator-remove-member-executes](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/05-validator-remove-member-executes.json) | P3 | 제외 |
| RT-B-11 <br> 실행: prev-seals-quorum | [prev-seals-quorum](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/11-prev-seals-quorum.json) | P3 | 제외 |
| WBFT-010 [부분 대응] <br> 실행: randao-and-mixdigest-present | [randao-and-mixdigest-present](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/13-randao-and-mixdigest-present.json) | P3 | 제외 |
| RT-B-06 [부분 대응] <br> 실행: stablenet-gastip-field | [stablenet-gastip-field](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/14-stablenet-gastip-field.json) | P3 | 제외 |
| TX-009 [부분 대응] <br> 실행: secp256r1-precompile-valid | [secp256r1-precompile-valid](../sources/chainbench/tests/tc/go-wbft/accounts/01-secp256r1-precompile-valid.json) | P3 | 제외 |
| TX-019 [부분 대응] <br> 실행: secp256r1-precompile-invalid | [secp256r1-precompile-invalid](../sources/chainbench/tests/tc/go-wbft/accounts/02-secp256r1-precompile-invalid.json) | P3 | 제외 |
| TX-020 [부분 대응] <br> 실행: secp256r1-precompile-short-input | [secp256r1-precompile-short-input](../sources/chainbench/tests/tc/go-wbft/accounts/03-secp256r1-precompile-short-input.json) | P3 | 제외 |
