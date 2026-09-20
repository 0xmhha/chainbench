# tests/tc 전체 목록

JSON 189개. 상태: adapt 89, conditional 3, direct 5, excluded 92. 문서 3개는 테스트 수에 포함하지 않는다.

direct는 원문 환경이 wemix인 구현이라는 뜻이며 실행 성공을 뜻하지 않는다. adapt는 환경/오류정책/기대조건 수정 후 실행 가능, conditional은 추가 바이너리·외부 환경·지원 확인 필요, excluded는 이번 PoA 회귀 범위에서 제외한다. P0는 기본 릴리스 차단 점검, P1은 핵심 회귀, P2는 확장 호환성, P3는 보조/대상 외다. 모든 테스트는 미실행이다.

## 우선순위별 전수 목록

| 순위 | 상태 | 파일 / ID | 원문 chain | requires | 내용 및 적용 판정 |
|---|---|---|---|---|---|
| P1 | adapt | [tests/tc/basic/01-basic-consensus.json](../sources/chainbench/tests/tc/basic/01-basic-consensus.json) / `basic-consensus` | stablenet | rpc, consensus | Verify blocks are being produced and all validators participate (원본 basic/consensus.sh) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/basic/02-basic-peers.json](../sources/chainbench/tests/tc/basic/02-basic-peers.json) / `basic-peers` | stablenet | rpc | Verify all nodes have proper peer connectivity (원본 basic/peers.sh) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/basic/03-basic-rpc-health.json](../sources/chainbench/tests/tc/basic/03-basic-rpc-health.json) / `basic-rpc-health` | stablenet | rpc | Verify all node RPC endpoints are responding (원본 basic/rpc-health.sh) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/basic/04-basic-sync.json](../sources/chainbench/tests/tc/basic/04-basic-sync.json) / `basic-sync` | stablenet | rpc, consensus | Verify all running nodes have synchronized block heights (원본 basic/sync.sh) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/basic/05-basic-tx-send.json](../sources/chainbench/tests/tc/basic/05-basic-tx-send.json) / `basic-tx-send` | stablenet | rpc | Send a transaction and verify it gets included in a block (원본 basic/tx-send.sh) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/basic/06-basic-txpool-propagation.json](../sources/chainbench/tests/tc/basic/06-basic-txpool-propagation.json) / `basic-txpool-propagation` | stablenet | rpc | Verify TX propagation across nodes and txpool drain under load (원본 basic/txpool-propagation.sh) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/fault/01-fault-network-partition.json](../sources/chainbench/tests/tc/fault/01-fault-network-partition.json) / `fault-network-partition` | stablenet | rpc, consensus, process | Simulate network partition via admin_removePeer - verify consensus halts and recovers after heal (원본 fault/network-partition.sh) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/fault/02-fault-node-crash.json](../sources/chainbench/tests/tc/fault/02-fault-node-crash.json) / `fault-node-crash` | stablenet | rpc, consensus, process | Stop 1 validator and verify consensus continues with 3/4 (원본 fault/node-crash.sh) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/fault/03-fault-node-recover.json](../sources/chainbench/tests/tc/fault/03-fault-node-recover.json) / `fault-node-recover` | stablenet | rpc, consensus, process | Stop a node, wait, restart, and measure sync time (원본 fault/node-recover.sh) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/fault/04-fault-p2p-topology.json](../sources/chainbench/tests/tc/fault/04-fault-p2p-topology.json) / `fault-p2p-topology` | stablenet | rpc, consensus, process | Test consensus and TX propagation under restricted hub-spoke P2P topology (원본 fault/p2p-topology.sh) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/fault/05-fault-two-down.json](../sources/chainbench/tests/tc/fault/05-fault-two-down.json) / `fault-two-down` | stablenet | rpc, consensus, process | Stop 2/4 validators - consensus should halt, recover when 1 returns (원본 fault/two-down.sh) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/fault/06-fault-txpool-leader-change.json](../sources/chainbench/tests/tc/fault/06-fault-txpool-leader-change.json) / `fault-txpool-leader-change` | stablenet | rpc, consensus, process | Verify pending transactions survive leader node failure and get processed by remaining validators (원본 fault/txpool-leader-change.sh) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/go-stablenet/post-v1.0.0-change/common-all/12-legacy-gasprice-below-min-rejected.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/12-legacy-gasprice-below-min-rejected.json) / `legacy-gasprice-below-min-rejected` | stablenet | rpc | TC-1-3-04 — LegacyTx gasPrice 최소 가스비 하한선 미만 거부 검증 (원본 post-v1.0.0-change/common-all/12-test-legacy-gasprice-below-min-revert) — 트랜잭션 하한 거부 의도는 유효하나 Stablenet 최소 수수료 정책에 묶임 |
| P1 | adapt | [tests/tc/go-stablenet/post-v1.0.0-change/common-all/13-accesslist-gasprice-below-min-rejected.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/13-accesslist-gasprice-below-min-rejected.json) / `accesslist-gasprice-below-min-rejected` | stablenet | rpc | TC-1-3-05 — AccessListTx gasPrice 최소 가스비 하한선 미만 거부 검증 (원본 post-v1.0.0-change/common-all/13-test-accesslist-gasprice-below-min-revert) — 트랜잭션 하한 거부 의도는 유효하나 Stablenet 최소 수수료 정책에 묶임 |
| P1 | adapt | [tests/tc/go-stablenet/post-v1.0.0-change/common-all/14-feecap-below-min-rejected.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/14-feecap-below-min-rejected.json) / `feecap-below-min-rejected` | stablenet | rpc | TC-1-3-06 — DynamicFeeTx gasTipCap 최소값 미만 거부 검증 (원본 post-v1.0.0-change/common-all/14-test-dynamic-fee-tipcap-below-min-revert) — 트랜잭션 하한 거부 의도는 유효하나 Stablenet 최소 수수료 정책에 묶임 |
| P1 | adapt | [tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json) / `effective-gas-price-regular-bp-en` | stablenet | rpc | TC-4-6-02 — 일반 계정 tx 의 effectiveGasPrice 가 두 노드에서 같다 — 일반 계정 영수증 effectiveGasPrice 노드 간 일치 검증 |
| P1 | adapt | [tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json) / `signature-compat-across-swap` | stablenet | rpc, consensus, process | TC-3-1-04 — 노드가 다른 바이너리로 재기동해도 기존 tx 의 blockNumber·status·from·to 가 보존된다 — genesis/바이너리 교체/기존 데이터 보존 의도는 PoA에서도 유효 |
| P1 | adapt | [tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json) / `genesis-mismatch-refuses-to-start` | stablenet | rpc, consensus, process | TC-4-1-03 — 이 체인과 다른 genesis 로 빌드된 바이너리는 GenesisMismatch 로 기동에 실패한다 — genesis/바이너리 교체/기존 데이터 보존 의도는 PoA에서도 유효 |
| P1 | adapt | [tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/04-genesis-block-hash-consistent.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/04-genesis-block-hash-consistent.json) / `genesis-block-hash-consistent` | stablenet | rpc | TC-5-3-01: block 0 해시가 모든 노드에서 같고 parentHash 가 0 이다. 레거시는 릴리스 고정 해시와 비교했으나, 사설망은 env 마다 genesis 가 달라 노드 간 일치와 genesis 형태로 검증한다. — genesis/바이너리 교체/기존 데이터 보존 의도는 PoA에서도 유효 |
| P1 | adapt | [tests/tc/go-stablenet/regression/api/01-block-transactions-field.json](../sources/chainbench/tests/tc/go-stablenet/regression/api/01-block-transactions-field.json) / `block-transactions-field` | stablenet | rpc | RT-G-1-01 — eth_getBlockByNumber(latest) (원본 regression/api/01-test-get-block-by-number) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/go-stablenet/regression/api/02-block-by-hash-consistency.json](../sources/chainbench/tests/tc/go-stablenet/regression/api/02-block-by-hash-consistency.json) / `block-by-hash-consistency` | stablenet | rpc | RT-G-1-02 — eth_getBlockByHash (원본 regression/api/02-test-get-block-by-hash) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/go-stablenet/regression/api/03-transaction-by-hash-fields.json](../sources/chainbench/tests/tc/go-stablenet/regression/api/03-transaction-by-hash-fields.json) / `transaction-by-hash-fields` | stablenet | rpc | RT-G-1-03 — eth_getTransactionByHash (원본 regression/api/03-test-get-tx-by-hash) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/go-stablenet/regression/api/04-transaction-receipt-fields.json](../sources/chainbench/tests/tc/go-stablenet/regression/api/04-transaction-receipt-fields.json) / `transaction-receipt-fields` | stablenet | rpc | RT-G-1-04 — eth_getTransactionReceipt (PR #70 fix 확인) (원본 regression/api/04-test-get-tx-receipt) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/go-stablenet/regression/api/05-transaction-count-increments.json](../sources/chainbench/tests/tc/go-stablenet/regression/api/05-transaction-count-increments.json) / `transaction-count-increments` | stablenet | rpc | RT-G-1-05 — eth_getTransactionCount (nonce 조회) (원본 regression/api/05-test-get-tx-count) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/go-stablenet/regression/api/18-txpool-status.json](../sources/chainbench/tests/tc/go-stablenet/regression/api/18-txpool-status.json) / `txpool-status` | stablenet | rpc | RT-G-4-02 — txpool_status: pending(연속 nonce) + queued(nonce gap) 분리 (원본 regression/api/18-test-txpool-status) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/go-stablenet/regression/api/19-txpool-content-well-formed.json](../sources/chainbench/tests/tc/go-stablenet/regression/api/19-txpool-content-well-formed.json) / `txpool-content-well-formed` | stablenet | rpc | RT-G-4-03 — txpool_content: pending/queued 분리 내용 확인 (원본 regression/api/19-test-txpool-content) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/go-stablenet/regression/ethereum/08-legacy-transfer.json](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/08-legacy-transfer.json) / `legacy-transfer` | stablenet | rpc | RT-A-2-01 — Legacy Tx (type 0x0) 발행 (원본 regression/ethereum/08-test-legacy-tx) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/go-stablenet/regression/ethereum/09-dynamic-fee-tx.json](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/09-dynamic-fee-tx.json) / `dynamic-fee-tx` | stablenet | rpc | RT-A-2-02 — EIP-1559 DynamicFeeTx (type 0x2) 발행 (원본 regression/ethereum/09-test-dynamic-fee-tx) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/go-stablenet/regression/ethereum/10-access-list-tx.json](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/10-access-list-tx.json) / `access-list-tx` | stablenet | rpc | RT-A-2-03: eth_createAccessList 로 노드가 만든 접근 목록을 붙여 type 0x01 트랜잭션을 보낸다. — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/go-stablenet/regression/ethereum/11-nonce-ordering.json](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/11-nonce-ordering.json) / `nonce-ordering` | stablenet | rpc | RT-A-2-04 — Nonce 순서 보장 (원본 regression/ethereum/11-test-nonce-ordering) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json) / `out-of-order-nonces-mine` | stablenet | rpc | (description 없음; id 및 steps 참조) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/go-stablenet/regression/ethereum/12-dynamic-fee-below-basefee-rejected.json](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/12-dynamic-fee-below-basefee-rejected.json) / `dynamic-fee-below-basefee-rejected` | stablenet | rpc | RT-A-2-05a — GasTipCap < MinTip tx 거부 검증 (원본 regression/ethereum/12-test-tipcap-underpriced) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/go-stablenet/regression/ethereum/14-insufficient-funds-rejected.json](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/14-insufficient-funds-rejected.json) / `insufficient-funds-rejected` | stablenet | rpc | RT-A-2-06 — 잔액 부족 tx 거부 (원본 regression/ethereum/14-test-insufficient-funds) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/go-stablenet/regression/ethereum/15-gas-limit-exceeds-block-rejected.json](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/15-gas-limit-exceeds-block-rejected.json) / `gas-limit-exceeds-block-rejected` | stablenet | rpc | RT-A-2-07 — Gas Limit 초과 tx 거부 (블록 gas limit 초과) (원본 regression/ethereum/15-test-gaslimit-exceeded) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/go-stablenet/regression/ethereum/16-effective-gas-price.json](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/16-effective-gas-price.json) / `effective-gas-price` | stablenet | rpc | RT-A-2-08 — eth_getTransactionReceipt의 effectiveGasPrice 검증 (원본 regression/ethereum/16-test-effective-gas-price) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/go-stablenet/regression/ethereum/17-replacement-tx.json](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/17-replacement-tx.json) / `replacement-tx` | stablenet | rpc | RT-A-2-09 — 동일 nonce, 더 높은 GasFeeCap으로 tx 교체 (원본 regression/ethereum/17-test-replacement-tx) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/go-stablenet/regression/ethereum/17b-same-nonce-replacement.json](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/17b-same-nonce-replacement.json) / `same-nonce-replacement` | stablenet | rpc | (description 없음; id 및 steps 참조) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/go-stablenet/regression/ethereum/19-contract-roundtrip.json](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/19-contract-roundtrip.json) / `contract-roundtrip` | stablenet | rpc | RT-A-3-01 — 컨트랙트 배포 (원본 regression/ethereum/19-test-contract-deploy) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json) / `eth-call-revert-returns-error` | stablenet | rpc | RT-A-3-05 — eth_call로 revert하는 함수 호출 시 에러 반환 (원본 regression/ethereum/23-test-eth-call-revert) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/go-stablenet/regression/ethereum/24-revert-tx-status-zero.json](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/24-revert-tx-status-zero.json) / `revert-tx-status-zero` | stablenet | rpc | RT-A-3-06 — revert tx: receipt.status == 0x0, gasUsed만 차감 검증 (원본 regression/ethereum/24-test-revert-tx) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json) / `out-of-gas-consumes-all` | stablenet | rpc | RT-A-3-07 — out-of-gas tx: gasUsed == gasLimit, 잔액 전량 차감 검증 (원본 regression/ethereum/25-test-out-of-gas) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/go-stablenet/regression/ethereum/27b-value-transfer.json](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/27b-value-transfer.json) / `value-transfer` | stablenet | rpc | RT-A-4-03 — eth_sendRawTransaction 서명된 tx 전파 (원본 regression/ethereum/28-test-send-raw-tx) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/go-stablenet/regression/ethereum/36-contract-event-emitted.json](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/36-contract-event-emitted.json) / `contract-event-emitted` | stablenet | rpc | (description 없음; id 및 steps 참조) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P1 | adapt | [tests/tc/go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json) / `fee-delegated-transfer` | stablenet | rpc | RT-D-01 — FeeDelegateDynamicFeeTx (type 0x16) 정상 처리 (원본 regression/fee-delegation/01-test-fee-delegate-normal) — go-wemix는 FeeDelegateDynamicFeeTx(type 22)를 지원 |
| P1 | adapt | [tests/tc/go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json) / `fd-sender-sig-invalid-rejected` | stablenet | rpc | RT-D-03 — Sender 서명 변조 시 거부 (원본 regression/fee-delegation/02-test-sender-sig-invalid) — go-wemix는 FeeDelegateDynamicFeeTx(type 22)를 지원 |
| P1 | adapt | [tests/tc/go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json) / `fd-feepayer-sig-invalid-rejected` | stablenet | rpc | RT-D-04 — FeePayer 서명 변조 시 거부 (원본 regression/fee-delegation/03-test-feepayer-sig-invalid) — go-wemix는 FeeDelegateDynamicFeeTx(type 22)를 지원 |
| P1 | adapt | [tests/tc/go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json) / `feepayer-insufficient-rejected` | stablenet | rpc | RT-D-05 — FeePayer 잔액 부족 시 거부 (원본 regression/fee-delegation/04-test-feepayer-insufficient) — go-wemix는 FeeDelegateDynamicFeeTx(type 22)를 지원 |
| P1 | adapt | [tests/tc/go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json) / `fee-delegated-sender-sig-invalid-rejected` | stablenet | rpc | (description 없음; id 및 steps 참조) — go-wemix는 FeeDelegateDynamicFeeTx(type 22)를 지원 |
| P1 | adapt | [tests/tc/go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json) / `fee-delegated-feepayer-sig-invalid-rejected` | stablenet | rpc | (description 없음; id 및 steps 참조) — go-wemix는 FeeDelegateDynamicFeeTx(type 22)를 지원 |
| P1 | adapt | [tests/tc/go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json](../sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json) / `fee-delegated-unfunded-feepayer-rejected` | stablenet | rpc | (description 없음; id 및 steps 참조) — go-wemix는 FeeDelegateDynamicFeeTx(type 22)를 지원 |
| P1 | direct | [tests/tc/go-wemix/chain-up/01-wemix-chain-up.json](../sources/chainbench/tests/tc/go-wemix/chain-up/01-wemix-chain-up.json) / `wemix-chain-up` | wemix | rpc | (description 없음; id 및 steps 참조) — wemix 환경이 명시된 구현 테스트; 미실행 |
| P1 | direct | [tests/tc/go-wemix/chain-up/02-wemix-chain-up-15.json](../sources/chainbench/tests/tc/go-wemix/chain-up/02-wemix-chain-up-15.json) / `wemix-chain-up-15` | wemix | rpc, consensus | (description 없음; id 및 steps 참조) — wemix 환경이 명시된 구현 테스트; 미실행 |
| P1 | direct | [tests/tc/go-wemix/fault/01-wemix-node-crash.json](../sources/chainbench/tests/tc/go-wemix/fault/01-wemix-node-crash.json) / `wemix-node-crash` | wemix | rpc, consensus, process | Stop one wemix (poa) producer and verify block production continues, then restart it (WA25 — wemix had no fault coverage). — wemix 환경이 명시된 구현 테스트; 미실행 |
| P1 | direct | [tests/tc/go-wemix/tx/01-wemix-tx-and-contract.json](../sources/chainbench/tests/tc/go-wemix/tx/01-wemix-tx-and-contract.json) / `wemix-tx-and-contract` | wemix | rpc | (description 없음; id 및 steps 참조) — wemix 환경이 명시된 구현 테스트; 미실행 |
| P1 | adapt | [tests/tc/stress/01-stress-block-time.json](../sources/chainbench/tests/tc/stress/01-stress-block-time.json) / `stress-block-time` | stablenet | rpc, consensus | Measure block production time statistics over last 100 blocks (원본 stress/block-time.sh) — 부하 중 생산 지속/간격 회귀에 유효하나 블록 바이트 크기 경계는 검증하지 않음 |
| P1 | adapt | [tests/tc/stress/02-stress-tx-flood.json](../sources/chainbench/tests/tc/stress/02-stress-tx-flood.json) / `stress-tx-flood` | stablenet | rpc, consensus | Send N transactions rapidly and measure throughput (원본 stress/tx-flood.sh) — 부하 중 생산 지속/간격 회귀에 유효하나 블록 바이트 크기 경계는 검증하지 않음 |
| P1 | adapt | [tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json) / `effective-gas-price-regular` | stablenet | rpc | TC-4-6-02 — EffectiveGasPrice for regular (non-authorized) account (BP vs snap-sync EN comparison) (원본 post-v1.0.0-change/effectivegasprice/02-test-regular-account) — 공통 실행/RPC 동작이 존재하며 기존 adapt와 동일 기준으로 기대값을 go-wemix 정책으로 이식할 수 있다. |
| P1 | adapt | [tests/tc/go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json) / `anzeon-basefee-increase` | stablenet | rpc | (description 없음; id 및 steps 참조) — 공통 실행/RPC 동작이 존재하며 기존 adapt와 동일 기준으로 기대값을 go-wemix 정책으로 이식할 수 있다. |
| P1 | adapt | [tests/tc/go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json) / `anzeon-basefee-stable` | stablenet | rpc | (description 없음; id 및 steps 참조) — 공통 실행/RPC 동작이 존재하며 기존 adapt와 동일 기준으로 기대값을 go-wemix 정책으로 이식할 수 있다. |
| P1 | adapt | [tests/tc/go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json) / `anzeon-basefee-decrease` | stablenet | rpc | (description 없음; id 및 steps 참조) — 공통 실행/RPC 동작이 존재하며 기존 adapt와 동일 기준으로 기대값을 go-wemix 정책으로 이식할 수 있다. |
| P1 | adapt | [tests/tc/go-stablenet/regression/anzeon/06-basefee-minimum.json](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/06-basefee-minimum.json) / `basefee-minimum` | stablenet | rpc | RT-C-06 — baseFee가 MinBaseFee(20 Gwei) 아래로 내려가지 않음 (원본 regression/anzeon/06-test-min-basefee) — 공통 실행/RPC 동작이 존재하며 기존 adapt와 동일 기준으로 기대값을 go-wemix 정책으로 이식할 수 있다. |
| P1 | adapt | [tests/tc/go-stablenet/regression/anzeon/07-basefee-maximum.json](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/07-basefee-maximum.json) / `basefee-maximum` | stablenet | rpc | RT-C-07 — baseFee가 MaxBaseFee(20,000,000 Gwei) 상한을 초과하지 않음 (원본 regression/anzeon/07-test-max-basefee) — 공통 실행/RPC 동작이 존재하며 기존 adapt와 동일 기준으로 기대값을 go-wemix 정책으로 이식할 수 있다. |
| P1 | adapt | [tests/tc/go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json) / `feecap-above-min-accepted` | stablenet | rpc | (description 없음; id 및 steps 참조) — 공통 실행/RPC 동작이 존재하며 기존 adapt와 동일 기준으로 기대값을 go-wemix 정책으로 이식할 수 있다. |
| P1 | adapt | [tests/tc/go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json) / `feecap-exact-min-accepted` | stablenet | rpc | (description 없음; id 및 steps 참조) — 공통 실행/RPC 동작이 존재하며 기존 adapt와 동일 기준으로 기대값을 go-wemix 정책으로 이식할 수 있다. |
| P1 | adapt | [tests/tc/go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json) / `gaslimit-exceeded-rejected` | stablenet | rpc | (description 없음; id 및 steps 참조) — 공통 실행/RPC 동작이 존재하며 기존 adapt와 동일 기준으로 기대값을 go-wemix 정책으로 이식할 수 있다. |
| P2 | direct | [tests/tc/go-wemix/rpc/01-wemix-brioche-block-reward.json](../sources/chainbench/tests/tc/go-wemix/rpc/01-wemix-brioche-block-reward.json) / `wemix-brioche-block-reward` | wemix | rpc | (description 없음; id 및 steps 참조) — wemix 환경이 명시된 구현 테스트; 미실행 |
| P2 | adapt | [tests/tc/go-stablenet/regression/api/07-gas-price-positive.json](../sources/chainbench/tests/tc/go-stablenet/regression/api/07-gas-price-positive.json) / `gas-price-positive` | stablenet | rpc | RT-G-2-01 — eth_gasPrice == baseFee + GasTip (원본 regression/api/07-test-gas-price) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P2 | adapt | [tests/tc/go-stablenet/regression/api/09-fee-history-well-formed.json](../sources/chainbench/tests/tc/go-stablenet/regression/api/09-fee-history-well-formed.json) / `fee-history-well-formed` | stablenet | rpc | RT-G-2-03 — eth_feeHistory (원본 regression/api/09-test-fee-history) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P2 | adapt | [tests/tc/go-stablenet/regression/api/20-admin-peers-populated.json](../sources/chainbench/tests/tc/go-stablenet/regression/api/20-admin-peers-populated.json) / `admin-peers-populated` | stablenet | rpc | RT-A-1-05 — P2P 피어 연결 확인 (원본 regression/ethereum/05-test-p2p-peers) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P2 | adapt | [tests/tc/go-stablenet/regression/api/24-chain-not-syncing.json](../sources/chainbench/tests/tc/go-stablenet/regression/api/24-chain-not-syncing.json) / `chain-not-syncing` | stablenet | rpc | (description 없음; id 및 steps 참조) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P2 | adapt | [tests/tc/go-stablenet/regression/ethereum/01-stablenet-chain-up.json](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/01-stablenet-chain-up.json) / `stablenet-chain-up` | stablenet | rpc, consensus | (description 없음; id 및 steps 참조) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P2 | adapt | [tests/tc/go-stablenet/regression/ethereum/22-estimate-gas.json](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/22-estimate-gas.json) / `estimate-gas` | stablenet | rpc | RT-A-3-04 — eth_estimateGas 정상 동작 (원본 regression/ethereum/22-test-estimate-gas) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P2 | adapt | [tests/tc/go-stablenet/regression/ethereum/27-genesis-balance.json](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/27-genesis-balance.json) / `genesis-balance` | stablenet | rpc | RT-A-4-02 — eth_getBalance 정상 조회 (원본 regression/ethereum/27-test-eth-get-balance) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P2 | adapt | [tests/tc/go-stablenet/regression/ethereum/29-logs-query-well-formed.json](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/29-logs-query-well-formed.json) / `logs-query-well-formed` | stablenet | rpc | RT-A-4-04 — eth_getLogs 이벤트 로그 조회 (원본 regression/ethereum/29-test-eth-get-logs) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P2 | adapt | [tests/tc/go-stablenet/regression/ethereum/31-ws-subscribe-new-heads.json](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/31-ws-subscribe-new-heads.json) / `ws-subscribe-new-heads` | stablenet | rpc, ws | RT-A-4-06 — eth_subscribe(newHeads) WebSocket 구독 (원본 regression/ethereum/31-test-ws-subscribe-heads) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P2 | adapt | [tests/tc/go-stablenet/regression/ethereum/32-ws-subscribe-logs.json](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/32-ws-subscribe-logs.json) / `ws-subscribe-logs` | stablenet | rpc, ws | RT-A-4-07 — eth_subscribe(logs) WebSocket 구독 (원본 regression/ethereum/32-test-ws-subscribe-logs) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P2 | adapt | [tests/tc/go-stablenet/regression/ethereum/33-stablenet-chain-up-15.json](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/33-stablenet-chain-up-15.json) / `stablenet-chain-up-15` | stablenet | rpc, consensus | (description 없음; id 및 steps 참조) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P2 | conditional | [tests/tc/go-stablenet/topology/01-proxied-pn-routing.json](../sources/chainbench/tests/tc/go-stablenet/topology/01-proxied-pn-routing.json) / `stablenet-proxied-pn-routing` | stablenet | rpc, consensus | Compose the proxied tier (bp <-> pn <-> en) and prove the endpoint stays in sync through the pn: under proxied peering an endpoint never dials a producer, so a rising head on en confirms the pn relays blocks to it (WA25). The pn/proxied compose was verified live on the docker fleet; chain sync is a proven assertion pattern. — proxied PN routing 기능의 wemix 토폴로지 지원 확인이 선행되어야 함 |
| P2 | adapt | [tests/tc/go-stablenet/tx/01-negative-tx-revert.json](../sources/chainbench/tests/tc/go-stablenet/tx/01-negative-tx-revert.json) / `stablenet-negative-tx-revert` | stablenet | rpc | Negative path (WA25): deploy a contract whose runtime always reverts, then send it a transaction and require that it reverts (mined with status 0x0). Also send a plain value transfer and require it succeeds, so a false pass would need both to be wrong. — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P2 | adapt | [tests/tc/go-wbft/chain-up/01-wbft-chain-up.json](../sources/chainbench/tests/tc/go-wbft/chain-up/01-wbft-chain-up.json) / `wbft-chain-up` | wbft | rpc, consensus | (description 없음; id 및 steps 참조) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P2 | adapt | [tests/tc/go-wbft/chain-up/02-wbft-chain-up-15.json](../sources/chainbench/tests/tc/go-wbft/chain-up/02-wbft-chain-up-15.json) / `wbft-chain-up-15` | wbft | rpc, consensus | (description 없음; id 및 steps 참조) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P2 | adapt | [tests/tc/go-wbft/consensus/01-e1-mixed-producers.json](../sources/chainbench/tests/tc/go-wbft/consensus/01-e1-mixed-producers.json) / `e1-mixed-producers` | stablenet | rpc, consensus | (description 없음; id 및 steps 참조) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P2 | adapt | [tests/tc/go-wbft/fault/01-wbft-node-crash.json](../sources/chainbench/tests/tc/go-wbft/fault/01-wbft-node-crash.json) / `wbft-node-crash` | wbft | rpc, consensus, process | Stop one wbft validator and verify consensus continues 3/4, then restart it (WA25 — wbft had no fault coverage). — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P2 | adapt | [tests/tc/go-wbft/tx/01-wbft-tx-and-contract.json](../sources/chainbench/tests/tc/go-wbft/tx/01-wbft-tx-and-contract.json) / `wbft-tx-and-contract` | wbft | rpc | wbft value transfer and contract deploy/call: fund a fresh account, deploy a returner contract, and read it back (WA25 — wbft had no tx coverage). — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P2 | conditional | [tests/tc/go-wemix/handoff/01-wemix-wbft-handoff.json](../sources/chainbench/tests/tc/go-wemix/handoff/01-wemix-wbft-handoff.json) / `wemix-wbft-handoff` | wbft | rpc | (description 없음; id 및 steps 참조) — 별도 go-wbft 바이너리와 업그레이드 프로필이 필요한 통합 테스트 |
| P2 | adapt | [tests/tc/remote/01-remote-rpc-health.json](../sources/chainbench/tests/tc/remote/01-remote-rpc-health.json) / `remote-rpc-health` | stablenet | rpc | 레거시 remote/rpc-health: 붙은 엔드포인트가 살아 있고 기본 RPC 셋이 응답한다. chainbench run --rpc <endpoint> 로 실행한다. — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P2 | adapt | [tests/tc/remote/02-remote-chain-info.json](../sources/chainbench/tests/tc/remote/02-remote-chain-info.json) / `remote-chain-info` | stablenet | rpc | 레거시 remote/chain-info: 붙은 체인이 chainId 를 보고하고 동기화가 끝나 있다. — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P2 | adapt | [tests/tc/remote/03-remote-balance-check.json](../sources/chainbench/tests/tc/remote/03-remote-balance-check.json) / `remote-balance-check` | stablenet | rpc | 레거시 remote/balance-check: 잔액 조회가 16진 수량으로 돌아온다. 레거시 기본값과 같이 0 주소를 본다 — 어느 체인에나 있고 값이 변하지 않는다. — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P2 | adapt | [tests/tc/samples/01-sample-minimal.json](../sources/chainbench/tests/tc/samples/01-sample-minimal.json) / `sample-minimal-value-transfer` | stablenet | rpc | v1 스펙 샘플. 이미 떠 있는 체인에 붙어 실행한다. steps 로 값을 모으고 assertions 로 판정한다. chainbench validate tests/tc/samples/01-sample-spec.json — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P2 | adapt | [tests/tc/samples/02-sample-lifecycle.json](../sources/chainbench/tests/tc/samples/02-sample-lifecycle.json) / `sample-lifecycle-node-restart` | stablenet | rpc, consensus, process | 작성 샘플 — 노드를 멈췄다 살리고 체인이 이어지는지 확인한다 (docs/dev/dsl-authoring-guide.md) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P2 | adapt | [tests/tc/go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json](../sources/chainbench/tests/tc/go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json) / `gas-price-equals-basefee-plus-tip` | stablenet | rpc | (description 없음; id 및 steps 참조) — 공통 실행/RPC 동작이 존재하며 기존 adapt와 동일 기준으로 기대값을 go-wemix 정책으로 이식할 수 있다. |
| P2 | adapt | [tests/tc/go-stablenet/regression/wbft/01-block-period-one-second.json](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/01-block-period-one-second.json) / `block-period-one-second` | stablenet | rpc | RT-B-01 — 블록 생산 주기 1초 간격 (원본 regression/wbft/01-test-block-period) — 공통 실행/RPC 동작이 존재하며 기존 adapt와 동일 기준으로 기대값을 go-wemix 정책으로 이식할 수 있다. |
| P3 | adapt | [tests/tc/go-stablenet/regression/api/21-fee-delegate-sign-rpc-present.json](../sources/chainbench/tests/tc/go-stablenet/regression/api/21-fee-delegate-sign-rpc-present.json) / `fee-delegate-sign-rpc-present` | stablenet | rpc | RT-G-5-01 — eth_signRawFeeDelegateTransaction (원본 regression/api/21-test-sign-raw-fee-delegate) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P3 | adapt | [tests/tc/go-stablenet/regression/ethereum/30-chain-id.json](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/30-chain-id.json) / `chain-id` | stablenet | rpc | RT-A-1-01 — 제네시스 블록으로 노드 초기화 (원본 regression/ethereum/01-test-genesis-init) — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P3 | adapt | [tests/tc/go-stablenet/vocabulary/01-derived-address-and-checksum.json](../sources/chainbench/tests/tc/go-stablenet/vocabulary/01-derived-address-and-checksum.json) / `stablenet-derived-vocabulary` | stablenet | rpc | Exercise the pure-derivation vocabulary that no case used before (WA24): createAddress computes the CREATE address from a deployer and nonce, and contractChecksum hashes bytecode. Both are deterministic — the expected values are fixed, so this is safe against a live chain as well as offline. — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P3 | conditional | [tests/tc/go-stablenet/vocabulary/02-faucet-funds-account.json](../sources/chainbench/tests/tc/go-stablenet/vocabulary/02-faucet-funds-account.json) / `stablenet-faucet-funds` | stablenet | rpc | Exercise the faucet action (WA24): top up a freshly generated account and read back its balance. LIVE-UNVERIFIED: authored offline (passes validate), needs a fleet run to confirm the target node's coinbase funder is unlocked. — faucet 외부 서비스 또는 자금 공급 설정 필요; 원문 LIVE-UNVERIFIED |
| P3 | adapt | [tests/tc/go-stablenet/vocabulary/03-register-contract.json](../sources/chainbench/tests/tc/go-stablenet/vocabulary/03-register-contract.json) / `stablenet-register-contract` | stablenet | rpc | Exercise the registerContract action (WA24): deploy a contract that accepts any call, then send an encoded call to it through registerContract and confirm the tx is mined without reverting. — 공통 Ethereum 실행/RPC 동작으로 이식 가능 |
| P3 | excluded | [tests/tc/basic/07-basic-wbft-consensus.json](../sources/chainbench/tests/tc/basic/07-basic-wbft-consensus.json) / `basic-wbft-consensus` | stablenet | rpc, consensus | Verify WBFT protocol properties - validator participation, round stability, commit seals (원본 basic/wbft-consensus.sh) — WBFT seals/validator API 전용 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json) / `stablenet-delayed-fork` | stablenet | rpc | (description 없음; id 및 steps 참조) — Boho/Anzeon·GovMinter·EIP-7702 관련 테스트 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json) / `govminter-v2-code` | stablenet | rpc | TC-5-2-05: GovMinter v2 업그레이드는 코드만 교체하고 잔액은 건드리지 않는다. 하드포크 전후의 코드가 다르고 잔액이 같은지를 함께 본다. — Boho/Anzeon·GovMinter·EIP-7702 관련 테스트 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json) / `burn-cancel-refundable` | stablenet | rpc | TC-1-1-01, TC-1-1-10 — 소각 제안 취소 → refundableBalance 이동 및 BurnDepositRefunded 이벤트 검증 (원본 post-v1.0.0-change/common-all/03-test-burn-cancel-refund) — Boho/Anzeon·GovMinter·EIP-7702 관련 테스트 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json) / `burn-reject-refundable` | stablenet | rpc | TC-1-1-02 — 소각 제안 거부(reject) → refundableBalance 이동 → claimBurnRefund 정상 출금 검증 (원본 post-v1.0.0-change/common-all/04-test-burn-reject-refund) — Boho/Anzeon·GovMinter·EIP-7702 관련 테스트 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json) / `burn-expire-refundable` | stablenet | rpc, short-expiry | TC-1-1-03: 소각 제안이 만료되면 GovMinter 로 옮겨진 예치금이 환불 가능 잔액이 된다. proposeBurn 직후 GovMinter 잔액 증가, 만료 후 상태 Expired(5), refundableBalance 증가분, BurnDepositRefunded 이벤트를 확인한다. — Boho/Anzeon·GovMinter·EIP-7702 관련 테스트 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json) / `burn-execute-no-refundable` | stablenet | rpc | TC-1-1-04 — 소각 제안 승인(approve) → 자동 실행(execute) → refundableBalance == 0 검증 (원본 post-v1.0.0-change/common-all/06-test-burn-execute-no-refund) — Boho/Anzeon·GovMinter·EIP-7702 관련 테스트 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json) / `claim-burn-refund-succeeds` | stablenet | rpc | TC-1-1-05, TC-1-1-09 — claimBurnRefund 정상 출금 및 BurnRefundClaimed 이벤트 검증 (원본 post-v1.0.0-change/common-all/07-test-claim-refund-success) — Boho/Anzeon·GovMinter·EIP-7702 관련 테스트 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json) / `claim-zero-refund-reverts` | stablenet | rpc | TC-1-1-06 — refundableBalance 0 계정 claimBurnRefund revert 검증 (원본 post-v1.0.0-change/common-all/08-test-claim-refund-zero-revert) — Boho/Anzeon·GovMinter·EIP-7702 관련 테스트 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json) / `claim-burn-refund-double-reverts` | stablenet | rpc | TC-1-1-07 — claimBurnRefund 중복 호출 revert 검증 (원본 post-v1.0.0-change/common-all/09-test-claim-refund-double-revert) — Boho/Anzeon·GovMinter·EIP-7702 관련 테스트 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json) / `prealloc-preserved-across-boho` | stablenet | rpc | TC-1-1-11/12: 하드포크가 prealloc 계정의 잔액·nonce 를 보존하고, 시스템 컨트랙트의 스토리지 슬롯도 그대로 둔다. — Boho/Anzeon·GovMinter·EIP-7702 관련 테스트 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/common-all/15-boho-chain-config-active.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/15-boho-chain-config-active.json) / `boho-chain-config-active` | stablenet | rpc | TC-4-1-01 — Boho hardfork chain config verification (원본 post-v1.0.0-change/common-all/15-test-chain-config-boho) — Boho/Anzeon·GovMinter·EIP-7702 관련 테스트 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/common-all/16-anzeon-active-before-boho.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/16-anzeon-active-before-boho.json) / `anzeon-active-before-boho` | stablenet | rpc | TC-4-1-02 — Boho 하드포크 값의 체인 설정 반영 및 런타임 활성화 검증 (원본 post-v1.0.0-change/common-all/16-test-boho-chain-config-activation) — Boho/Anzeon·GovMinter·EIP-7702 관련 테스트 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json) / `estimategas-authorizationlist-cost` | stablenet | rpc | TC-4-2-01/03: EIP-7702 authorizationList 를 1건·2건 붙였을 때 eth_estimateGas 가 그만큼 늘어난다. 경계값은 preActions 에서 이름을 붙여 한 번씩만 적는다. 상한은 노드의 estimateGas 오차 1.5% 정책에서 나온다 — 절대 상한은 하한 x 1.015, 증가분 상한은 절대 상한에서 baseline(21000)을 뺀 값이다. — Boho/Anzeon·GovMinter·EIP-7702 관련 테스트 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json) / `upgrade-registry-order` | stablenet | rpc | TC-5-2-01/02/03: block 0 에 Anzeon baseline 이 등록돼 있고, BohoBlock(100) 전까지 중간 업그레이드가 없으며, BohoBlock 에서 GovMinter 코드가 교체된다. — Boho/Anzeon·GovMinter·EIP-7702 관련 테스트 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json) / `v1-params-init-storage` | stablenet | rpc | TC-5-2-04: v1 시스템 컨트랙트의 Params 가 genesis 에서 초기화된다. GovValidator gasTip 슬롯(0x39)과 GovMinter quorum 슬롯(0x04)이 0이 아니어야 한다. — Boho/Anzeon·GovMinter·EIP-7702 관련 테스트 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json) / `burn-refund-events` | stablenet | rpc | (description 없음; id 및 steps 참조) — Boho/Anzeon·GovMinter·EIP-7702 관련 테스트 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json) / `effective-gas-price-authorized-bp-en` | stablenet | rpc | TC-4-6-01 — 인가 계정 tx 의 effectiveGasPrice 가 블록 생산 노드와 snap-sync 엔드포인트에서 같다 — 인가 계정 특수 정책 또는 WBFT GasTip 계산에 의존 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json) / `auth-tx-event-last-bp-en` | stablenet | rpc | TC-4-6-04 — AuthorizedTxExecuted 가 영수증 로그의 마지막이고, 두 노드가 같은 값을 보고한다 — 인가 계정 특수 정책 또는 WBFT GasTip 계산에 의존 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/extra-state/01-authorized-extra-bit-synced.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/01-authorized-extra-bit-synced.json) / `authorized-extra-bit-synced` | stablenet | rpc, account-extra | TC-4-5-01,TC-4-5-02 — Account Extra alloc bits reflected in AccountManager (authorized + blacklisted) (원본 post-v1.0.0-change/extra-state/01-test-extra-alloc-to-contract) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/extra-state/01b-blacklisted-extra-bit-synced.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/01b-blacklisted-extra-bit-synced.json) / `blacklisted-extra-bit-synced` | stablenet | rpc, account-extra | (description 없음; id 및 steps 참조) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json) / `stablenet-account-extra` | stablenet | rpc | (description 없음; id 및 steps 참조) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json) / `extra-union-merge` | stablenet | rpc | TC-4-5-05/06: alloc.Extra 와 GovCouncil params 가 서로 다른 계정을 인가하면 합집합이 되고, 양쪽에 다 있는 계정은 중복 없이 한 번만 센다 (3개, 4개가 아님). — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json) / `dual-status-extra` | stablenet | rpc, account-extra | TC-4-5-07 — 동일 주소 dual-status — authorized AND blacklisted 동시 반영 (원본 post-v1.0.0-change/extra-state/04-test-extra-dual-status) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/extra-state/05-extra-balance-preserved.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/05-extra-balance-preserved.json) / `extra-balance-preserved` | stablenet | rpc, account-extra | TC-4-5-08 — 동기화 시 무관 계정 잔액 보존 (원본 post-v1.0.0-change/extra-state/05-test-extra-balance-preserved) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json) / `invalid-extra-reject` | stablenet | rpc, consensus, process | TC-4-5-09 — 미정의 Extra 비트를 가진 genesis 를 받은 노드는 부팅에 실패한다 — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json) / `extra-state-across-delayed-boho` | stablenet | rpc | TC-4-5-10/11/12 — 하드포크가 지연돼도 alloc.Extra 가 AccountManager 에 반영되고 잔액은 보존된다 — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json) / `unsupported-system-contract-version` | stablenet | rpc, consensus, process | TC-5-2-06 — genesis 가 지원하지 않는 시스템 컨트랙트 버전을 선언하면 BohoBlock 을 커밋하지 못하고 멈춘다 — Boho 시스템 계약 버전 registry 실패 검증 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/string-handling/01-authorized-accounts-no-space.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/01-authorized-accounts-no-space.json) / `authorized-accounts-no-space` | stablenet | rpc | TC-4-3-01: GovCouncil authorizedAccounts splitAndTrim — 공백 없음 "0xaaa,0xbbb,0xccc" → 3 — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/string-handling/02-authorized-accounts-space.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/02-authorized-accounts-space.json) / `authorized-accounts-space` | stablenet | rpc | TC-4-3-02: GovCouncil authorizedAccounts splitAndTrim — 항목 사이 공백 "0xaaa, 0xbbb, 0xccc" → 3 — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/string-handling/03-authorized-accounts-trim.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/03-authorized-accounts-trim.json) / `authorized-accounts-trim` | stablenet | rpc | TC-4-3-03: GovCouncil authorizedAccounts splitAndTrim — 앞뒤 공백 " 0xaaa , 0xbbb " → 2 — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/string-handling/04-authorized-accounts-empty-item.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/04-authorized-accounts-empty-item.json) / `authorized-accounts-empty-item` | stablenet | rpc | TC-4-3-04: GovCouncil authorizedAccounts splitAndTrim — 빈 항목 "0xaaa,,0xbbb" → 2 — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/string-handling/05-authorized-accounts-single.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/05-authorized-accounts-single.json) / `authorized-accounts-single` | stablenet | rpc | TC-4-3-05: GovCouncil authorizedAccounts splitAndTrim — 단일 항목 "0xaaa" → 1 — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/post-v1.0.0-change/string-handling/06-authorized-accounts-empty.json](../sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/06-authorized-accounts-empty.json) / `authorized-accounts-empty` | stablenet | rpc | TC-4-3-06: GovCouncil authorizedAccounts splitAndTrim — 빈 문자열 "" → 0 — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json) / `regular-account-gastip-forced` | stablenet | rpc | RT-C-01 — 일반 계정의 tipCap이 header.GasTip()으로 강제 대체됨 (원본 regression/anzeon/01-test-regular-account-gastip-forced) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json](../sources/chainbench/tests/tc/go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json) / `authorized-account-gastip-free` | stablenet | rpc | RT-C-02 — Authorized 계정은 tipCap을 자유롭게 설정할 수 있음 (원본 regression/anzeon/02-test-authorized-account-gastip-free) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/api/06-system-contracts-deployed.json](../sources/chainbench/tests/tc/go-stablenet/regression/api/06-system-contracts-deployed.json) / `system-contracts-deployed` | stablenet | rpc | RT-G-1-06 — eth_getCode on NativeCoinAdapter (0x1000) (원본 regression/api/06-test-get-code-system) — Istanbul/WBFT API 또는 Stablenet 시스템 계약에 종속 |
| P3 | excluded | [tests/tc/go-stablenet/regression/api/08-max-priority-fee-equals-gastip.json](../sources/chainbench/tests/tc/go-stablenet/regression/api/08-max-priority-fee-equals-gastip.json) / `max-priority-fee-equals-gastip` | stablenet | rpc | RT-G-2-02 — eth_maxPriorityFeePerGas == WBFTExtra.GasTip (원본 regression/api/08-test-max-priority-fee) — Istanbul/WBFT API 또는 Stablenet 시스템 계약에 종속 |
| P3 | excluded | [tests/tc/go-stablenet/regression/api/10-estimate-gas-token-transfer.json](../sources/chainbench/tests/tc/go-stablenet/regression/api/10-estimate-gas-token-transfer.json) / `estimate-gas-token-transfer` | stablenet | rpc | RT-G-2-04 — eth_estimateGas (NativeCoinAdapter.transfer) (원본 regression/api/10-test-estimate-system-call) — Istanbul/WBFT API 또는 Stablenet 시스템 계약에 종속 |
| P3 | excluded | [tests/tc/go-stablenet/regression/api/11-node-address-returned.json](../sources/chainbench/tests/tc/go-stablenet/regression/api/11-node-address-returned.json) / `node-address-returned` | stablenet | rpc, consensus | RT-G-3-01 — istanbul_nodeAddress (원본 regression/api/11-test-node-address) — Istanbul/WBFT API 또는 Stablenet 시스템 계약에 종속 |
| P3 | excluded | [tests/tc/go-stablenet/regression/api/12-validator-set-nonempty.json](../sources/chainbench/tests/tc/go-stablenet/regression/api/12-validator-set-nonempty.json) / `validator-set-nonempty` | stablenet | rpc, consensus | RT-G-3-02 — istanbul_getValidators (원본 regression/api/12-test-get-validators) — Istanbul/WBFT API 또는 Stablenet 시스템 계약에 종속 |
| P3 | excluded | [tests/tc/go-stablenet/regression/api/12b-validator-set-count.json](../sources/chainbench/tests/tc/go-stablenet/regression/api/12b-validator-set-count.json) / `validator-set-count` | stablenet | rpc, consensus | (description 없음; id 및 steps 참조) — Istanbul/WBFT API 또는 Stablenet 시스템 계약에 종속 |
| P3 | excluded | [tests/tc/go-stablenet/regression/api/13-commit-signers-quorum.json](../sources/chainbench/tests/tc/go-stablenet/regression/api/13-commit-signers-quorum.json) / `commit-signers-quorum` | stablenet | rpc, consensus | RT-G-3-03 — istanbul_getCommitSignersFromBlock (원본 regression/api/13-test-get-commit-signers) — Istanbul/WBFT API 또는 Stablenet 시스템 계약에 종속 |
| P3 | excluded | [tests/tc/go-stablenet/regression/api/14-wbft-extra-info-fields.json](../sources/chainbench/tests/tc/go-stablenet/regression/api/14-wbft-extra-info-fields.json) / `wbft-extra-info-fields` | stablenet | rpc | RT-G-3-04 — istanbul_getWbftExtraInfo (원본 regression/api/14-test-get-wbft-extra) — Istanbul/WBFT API 또는 Stablenet 시스템 계약에 종속 |
| P3 | excluded | [tests/tc/go-stablenet/regression/api/15-istanbul-status-fields.json](../sources/chainbench/tests/tc/go-stablenet/regression/api/15-istanbul-status-fields.json) / `istanbul-status-fields` | stablenet | rpc | RT-G-3-05 — istanbul_status (원본 regression/api/15-test-istanbul-status) — Istanbul/WBFT API 또는 Stablenet 시스템 계약에 종속 |
| P3 | excluded | [tests/tc/go-stablenet/regression/api/16-is-validator-flags.json](../sources/chainbench/tests/tc/go-stablenet/regression/api/16-is-validator-flags.json) / `is-validator-flags` | stablenet | rpc, consensus | RT-G-3-06 — istanbul_isValidator (원본 regression/api/16-test-is-validator) — Istanbul/WBFT API 또는 Stablenet 시스템 계약에 종속 |
| P3 | excluded | [tests/tc/go-stablenet/regression/api/22-token-total-supply-readable.json](../sources/chainbench/tests/tc/go-stablenet/regression/api/22-token-total-supply-readable.json) / `token-total-supply-readable` | stablenet | rpc | RT-G-5-02 — eth_call NativeCoinAdapter.totalSupply (원본 regression/api/22-test-total-supply) — Istanbul/WBFT API 또는 Stablenet 시스템 계약에 종속 |
| P3 | excluded | [tests/tc/go-stablenet/regression/api/23-token-approve-sets-allowance.json](../sources/chainbench/tests/tc/go-stablenet/regression/api/23-token-approve-sets-allowance.json) / `token-approve-sets-allowance` | stablenet | rpc | RT-G-5-03 — eth_call NativeCoinAdapter.allowance (원본 regression/api/23-test-allowance) — Istanbul/WBFT API 또는 Stablenet 시스템 계약에 종속 |
| P3 | excluded | [tests/tc/go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json) / `sender-blacklisted-rejected` | stablenet | rpc | RT-E-01 — 블랙리스트 계정이 Sender인 tx 거부 (ErrBlacklistedAccount) (원본 regression/blacklist-authorized/01-test-sender-blacklisted) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json) / `recipient-blacklisted-rejected` | stablenet | rpc | RT-E-02 — 블랙리스트 계정이 Recipient인 tx 거부 (원본 regression/blacklist-authorized/02-test-recipient-blacklisted) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json) / `feepayer-blacklisted-rejected` | stablenet | rpc | RT-E-03 — FeePayer가 블랙리스트 계정이면 거부 (원본 regression/blacklist-authorized/03-test-feepayer-blacklisted) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json) / `address-unblacklisted-event` | stablenet | rpc | RT-E-04 — 블랙리스트 해제 거버넌스 흐름 검증 (원본 regression/blacklist-authorized/04-test-unblacklist) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json) / `zero-address-transfer-rejected` | stablenet | rpc | RT-E-05: 0x0 주소로의 전송은 제출 단계에서 거부된다 (ErrZeroAddressTransfer). — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json) / `precompile-transfer-rejected` | stablenet | rpc | RT-E-06: 프리컴파일·시스템 컨트랙트 주소로의 값 전송은 5개 주소 모두 거부된다. — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/blacklist-authorized/07-account-blacklist-readable.json](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/07-account-blacklist-readable.json) / `account-blacklist-readable` | stablenet | rpc | RT-E-07 — AccountManager.isBlacklisted() 조회 검증 (원본 regression/blacklist-authorized/07-test-is-blacklisted) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/blacklist-authorized/08-account-authorization-readable.json](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/08-account-authorization-readable.json) / `account-authorization-readable` | stablenet | rpc | RT-E-08 — AccountManager.isAuthorized() 조회 검증 (원본 regression/blacklist-authorized/08-test-is-authorized) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json](../sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json) / `authorized-tx-executed-event` | stablenet | rpc | RT-E-09 — AuthorizedTxExecuted 이벤트 발생 검증 (원본 regression/blacklist-authorized/09-test-authorized-tx-executed) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/ethereum/18-set-code-delegation.json](../sources/chainbench/tests/tc/go-stablenet/regression/ethereum/18-set-code-delegation.json) / `set-code-delegation` | stablenet | rpc | RT-A-2-10 — SetCodeTx (type 0x4 / EIP-7702) 계정 코드 위임 (원본 regression/ethereum/18-test-setcode-tx) — type 4 EIP-7702는 go-wemix 지원 transaction type에 없음 |
| P3 | excluded | [tests/tc/go-stablenet/regression/system-contracts/01-native-coin-adapter-code.json](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/01-native-coin-adapter-code.json) / `native-coin-adapter-code` | stablenet | rpc | RT-F-1-01 — NativeCoinAdapter.transfer → 기본 코인 전송과 동일 (원본 regression/system-contracts/01-test-native-transfer) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json) / `token-transfer-emits-event` | stablenet | rpc | (description 없음; id 및 steps 참조) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/system-contracts/02-token-balance-readable.json](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/02-token-balance-readable.json) / `token-balance-readable` | stablenet | rpc | RT-F-1-02 — NativeCoinAdapter.balanceOf == eth_getBalance (원본 regression/system-contracts/02-test-balance-of) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json) / `token-transfer-from-moves-balance` | stablenet | rpc | (description 없음; id 및 steps 참조) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/system-contracts/04-mint-transfer-event.json](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/04-mint-transfer-event.json) / `mint-transfer-event` | stablenet | rpc | RT-F-1-04 — Mint 실행 시 Transfer(0x0 → beneficiary) 이벤트 발생 (원본 regression/system-contracts/04-test-mint-transfer-event) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/system-contracts/05-burn-transfer-event.json](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/05-burn-transfer-event.json) / `burn-transfer-event` | stablenet | rpc | RT-F-1-05 — Burn 실행 시 Transfer(account → 0x0) 이벤트 발생 (원본 regression/system-contracts/05-test-burn-transfer-event) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/system-contracts/06-mint-proposal-executes.json](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/06-mint-proposal-executes.json) / `mint-proposal-executes` | stablenet | rpc | RT-F-2-01 — 코인 발행: proposeMint(proofData) → 승인 → execute (원본 regression/system-contracts/06-test-mint-proposal) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/system-contracts/07-burn-proposal-executes.json](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/07-burn-proposal-executes.json) / `burn-proposal-executes` | stablenet | rpc | RT-F-2-02 — 코인 소각: proposeBurn(proofData) payable → 승인 → execute (원본 regression/system-contracts/07-test-burn-proposal) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json) / `quorum-deficient-stays-voting` | stablenet | rpc | RT-F-2-03 — quorum 미달 → proposal 상태 Voting 유지, 발행 미실행 (원본 regression/system-contracts/08-test-quorum-deficient) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/system-contracts/09-validator-metadata-readable.json](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/09-validator-metadata-readable.json) / `validator-metadata-readable` | stablenet | rpc | RT-F-3-04 — validatorList() + validatorToOperator(v) + validatorToBlsKey(v) 다중 호출 (원본 regression/system-contracts/09-test-validator-metadata) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json) / `gastip-governance-updates-header` | stablenet | rpc | RT-B-06 — GasTip 거버넌스 변경 → 블록 헤더 WBFTExtra.GasTip 반영 검증 + 원복 (원본 regression/wbft/06-test-gastip-header-sync) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json) / `proposal-expiry-transitions` | stablenet | rpc, short-expiry | RT-F-3-06 — proposal expiry 초과 → Expired 상태 전환 → execute 불가 (원본 regression/system-contracts/11-test-proposal-expiry) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json) / `configure-minter-proposal-executes` | stablenet | rpc | RT-F-4-01 — GovMasterMinter.proposeConfigureMinter(address, uint256) → 승인 → execute (원본 regression/system-contracts/12-test-add-minter) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/system-contracts/13-remove-minter-executes.json](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/13-remove-minter-executes.json) / `remove-minter-executes` | stablenet | rpc | RT-F-4-02 — GovMasterMinter.proposeRemoveMinter(address) → 승인 → execute (원본 regression/system-contracts/13-test-remove-minter) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json) / `masterminter-member-add-remove` | stablenet | rpc | RT-F-4-03 — GovMasterMinter 자체 멤버 추가/제거 (proposeAddMember, proposeRemoveMember) (원본 regression/system-contracts/14-test-masterminter-self-member) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json) / `non-member-configure-minter-rejected` | stablenet | rpc | RT-F-4-04 — 비멤버 계정의 GovMasterMinter.proposeConfigureMinter 호출 거부 (원본 regression/system-contracts/15-test-non-member-rejected) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json) / `blacklist-proposal-executes` | stablenet | rpc | RT-F-5-01 — GovCouncil blacklist proposal → execute → isBlacklisted == true (원본 regression/system-contracts/16-test-blacklist) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json) / `authorize-proposal-executes` | stablenet | rpc | RT-F-5-03 — GovCouncil authorized account proposal → execute → isAuthorized == true (원본 regression/system-contracts/18-test-authorize) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json) / `unauthorize-proposal-executes` | stablenet | rpc | f5-04-unauthorize (원본 regression/system-contracts/19-test-unauthorize) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json) / `direct-blacklist-call-rejected` | stablenet | rpc | RT-F-5-05 — 비멤버가 AccountManager.blacklist 직접 호출 → revert (원본 regression/system-contracts/20-test-direct-blacklist-rejected) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/system-contracts/23-authorized-account-added-event.json](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/23-authorized-account-added-event.json) / `authorized-account-added-event` | stablenet | rpc | RT-F-5-08 — AuthorizedAccountAdded 이벤트 (원본 regression/system-contracts/23-test-authorized-account-added-event) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/system-contracts/25-token-metadata.json](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/25-token-metadata.json) / `token-metadata` | stablenet | rpc | (description 없음; id 및 steps 참조) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/system-contracts/28-minter-status-readable.json](../sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/28-minter-status-readable.json) / `minter-status-readable` | stablenet | rpc | (description 없음; id 및 steps 참조) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/wbft/02-wbft-seals-quorum.json](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/02-wbft-seals-quorum.json) / `wbft-seals-quorum` | stablenet | rpc, consensus | RT-B-02 — WBFTExtra에 Committed Seal + Prepared Seal 모두 존재하고 quorum 이상 (원본 regression/wbft/02-test-wbft-extra-seal) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json) / `epoch-transition-carries-epoch-info` | stablenet | rpc, consensus | RT-B-03 — 에폭 전환 — 검증자 집합 갱신 (원본 regression/wbft/03-test-epoch-transition) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/wbft/04-validator-add-member-executes.json](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/04-validator-add-member-executes.json) / `validator-add-member-executes` | stablenet | rpc | RT-B-04 — 신규 검증자 추가 및 에폭 합의 참여 확인 (원본 regression/wbft/04-test-add-validator) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json) / `validator-add-member-epoch-activates` | stablenet | rpc, consensus | RT-B-04 — 멤버로 추가된 노드가 에폭 경계에서 실제 검증자 집합에 들어간다 — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/wbft/05-validator-remove-member-executes.json](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/05-validator-remove-member-executes.json) / `validator-remove-member-executes` | stablenet | rpc | RT-B-05: proposeRemoveMember 가 실행되면 GovValidator 멤버에서 빠진다. 제거 대상을 먼저 추가해 자기 완결로 만든다. — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/wbft/11-prev-seals-quorum.json](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/11-prev-seals-quorum.json) / `prev-seals-quorum` | stablenet | rpc, consensus | RT-B-11 — 블록 N+1의 PrevCommittedSeal이 블록 N의 committers를 포함 (원본 regression/wbft/11-test-prev-committed-seal) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/wbft/13-randao-and-mixdigest-present.json](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/13-randao-and-mixdigest-present.json) / `randao-and-mixdigest-present` | stablenet | rpc | (description 없음; id 및 steps 참조) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-stablenet/regression/wbft/14-stablenet-gastip-field.json](../sources/chainbench/tests/tc/go-stablenet/regression/wbft/14-stablenet-gastip-field.json) / `stablenet-gastip-field` | stablenet | rpc | (description 없음; id 및 steps 참조) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-wbft/accounts/01-secp256r1-precompile-valid.json](../sources/chainbench/tests/tc/go-wbft/accounts/01-secp256r1-precompile-valid.json) / `secp256r1-precompile-valid` | wbft | rpc | (description 없음; id 및 steps 참조) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-wbft/accounts/02-secp256r1-precompile-invalid.json](../sources/chainbench/tests/tc/go-wbft/accounts/02-secp256r1-precompile-invalid.json) / `secp256r1-precompile-invalid` | wbft | rpc | (description 없음; id 및 steps 참조) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |
| P3 | excluded | [tests/tc/go-wbft/accounts/03-secp256r1-precompile-short-input.json](../sources/chainbench/tests/tc/go-wbft/accounts/03-secp256r1-precompile-short-input.json) / `secp256r1-precompile-short-input` | wbft | rpc | (description 없음; id 및 steps 참조) — WBFT/Anzeon 또는 별도 시스템 계약·account extra·P256 전용 조건 |

## 파일별 실제 do/expect와 수정 조건

### basic-consensus — tests/tc/basic/01-basic-consensus.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/basic/01-basic-consensus.json:29`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음. 제목의 전체 validator 참여 여부는 검사하지 않음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=20; timeout="180s"
2. expect="blockAdvance"; on="node1"; timeout="60s"; pollInterval="2s"
3. expect="sameBlockHash"; block="latest"

### basic-peers — tests/tc/basic/02-basic-peers.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/basic/02-basic-peers.json:28`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=2; timeout="120s"
2. expect="peerCount"; on="node1"; compare="GreaterOrEqual"; is="1"
3. expect="peerCount"; on="node2"; compare="GreaterOrEqual"; is="1"
4. expect="peerCount"; on="node3"; compare="GreaterOrEqual"; is="1"
5. expect="peerCount"; on="node4"; compare="GreaterOrEqual"; is="1"
6. expect="peerCount"; on="en1"; compare="GreaterOrEqual"; is="1"

### basic-rpc-health — tests/tc/basic/03-basic-rpc-health.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/basic/03-basic-rpc-health.json:28`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=2; timeout="120s"
2. expect="blockNumber"; on="node1"; compare="GreaterOrEqual"; is="0"
3. expect="blockNumber"; on="node2"; compare="GreaterOrEqual"; is="0"
4. expect="blockNumber"; on="node3"; compare="GreaterOrEqual"; is="0"
5. expect="blockNumber"; on="node4"; compare="GreaterOrEqual"; is="0"
6. expect="blockNumber"; on="en1"; compare="GreaterOrEqual"; is="0"

### basic-sync — tests/tc/basic/04-basic-sync.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/basic/04-basic-sync.json:29`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=3; timeout="120s"
2. expect="sameBlockHash"; block="latest"

### basic-tx-send — tests/tc/basic/05-basic-tx-send.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/basic/05-basic-tx-send.json:28`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=2; timeout="120s"
2. do="newAccount"; save="r"; saveKey="rk"
3. do="sendTx"; on="node1"; from="0xc17d493883eaa3b4cceb0f214b273392d562f9d8"; to="$r"; value="1"; expect="receipt"
4. expect="balanceAt"; on="node1"; address="$r"; compare="Greater"; is="0"

### basic-txpool-propagation — tests/tc/basic/06-basic-txpool-propagation.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/basic/06-basic-txpool-propagation.json:28`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음. 한 건 receipt와 node2 잔액만 검사; txpool 전파/배출 부하 직접 검사 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=2; timeout="120s"
2. do="newAccount"; save="r"; saveKey="rk"
3. do="sendTx"; on="node1"; from="0xc17d493883eaa3b4cceb0f214b273392d562f9d8"; to="$r"; value="1"; expect="receipt"
4. expect="balanceAt"; on="node2"; address="$r"; compare="Greater"; is="0"

### fault-network-partition — tests/tc/fault/01-fault-network-partition.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/fault/01-fault-network-partition.json:29`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. 장애 노드/etcd leader·quorum 및 P2P 단절 범위를 PoA 기준으로 재정의한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음. 설명은 halt지만 blockHalt assert 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=3; timeout="120s"
2. expect="sameBlockHash"; block="latest"
3. do="partition"; groups=[["node1","node2"],["node3","node4"]]
4. expect="peerCount"; on="node1"; compare="LessOrEqual"; is="2"
5. expect="peerCount"; on="node3"; compare="LessOrEqual"; is="2"
6. do="healPartition"
7. do="waitBlock"; on="node3"; target=8; timeout="150s"
8. expect="sameBlockHash"; block="latest"

### fault-node-crash — tests/tc/fault/02-fault-node-crash.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/fault/02-fault-node-crash.json:29`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. 장애 노드/etcd leader·quorum 및 P2P 단절 범위를 PoA 기준으로 재정의한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=3; timeout="120s"
2. expect="sameBlockHash"; block="latest"
3. do="stopNode"; on="node3"
4. expect="blockAdvance"; on="node1"; timeout="90s"; pollInterval="2s"
5. do="restartNode"; on="node3"
6. do="waitBlock"; on="node3"; target=8; timeout="120s"
7. expect="sameBlockHash"; block="latest"

### fault-node-recover — tests/tc/fault/03-fault-node-recover.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/fault/03-fault-node-recover.json:29`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. 장애 노드/etcd leader·quorum 및 P2P 단절 범위를 PoA 기준으로 재정의한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음. 복귀 블록 대기 timeout은 있으나 sync 소요시간 계측/assert 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=3; timeout="120s"
2. do="stopNode"; on="node3"
3. do="waitBlock"; on="node1"; target=12; timeout="150s"
4. do="restartNode"; on="node3"
5. do="waitBlock"; on="node3"; target=12; timeout="150s"
6. expect="sameBlockHash"; block="latest"

### fault-p2p-topology — tests/tc/fault/04-fault-p2p-topology.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/fault/04-fault-p2p-topology.json:29`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. 장애 노드/etcd leader·quorum 및 P2P 단절 범위를 PoA 기준으로 재정의한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=3; timeout="120s"
2. expect="sameBlockHash"; block="latest"
3. do="partition"; groups=[["node2"],["node3"],["node4"]]
4. expect="peerCount"; on="node1"; compare="GreaterOrEqual"; is="2"
5. expect="blockAdvance"; on="node1"; timeout="90s"; pollInterval="2s"
6. do="newAccount"; save="r"; saveKey="rk"
7. do="sendTx"; on="node1"; from="0xc17d493883eaa3b4cceb0f214b273392d562f9d8"; to="$r"; value="1"; expect="receipt"
8. expect="balanceAt"; on="node1"; address="$r"; compare="Greater"; is="0"
9. do="healPartition"
10. expect="blockAdvance"; on="node4"; timeout="90s"; pollInterval="2s"
11. expect="sameBlockHash"; block="latest"

### fault-two-down — tests/tc/fault/05-fault-two-down.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/fault/05-fault-two-down.json:29`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. 장애 노드/etcd leader·quorum 및 P2P 단절 범위를 PoA 기준으로 재정의한다. 2/4 중단 중 blockHalt 기대는 WBFT 설명을 근거로 적용하지 말고 etcd 멤버 수·다수 조건으로 재검증한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=3; timeout="120s"
2. expect="sameBlockHash"; block="latest"
3. do="stopNode"; on="node3"
4. do="stopNode"; on="node4"
5. expect="blockHalt"; on="node1"; within="12s"; maxAdvance=1
6. do="startNode"; on="node3"
7. expect="blockAdvance"; on="node1"; timeout="90s"; pollInterval="2s"
8. do="startNode"; on="node4"

### fault-txpool-leader-change — tests/tc/fault/06-fault-txpool-leader-change.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/fault/06-fault-txpool-leader-change.json:29`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. 장애 노드/etcd leader·quorum 및 P2P 단절 범위를 PoA 기준으로 재정의한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음. 장애 전에 receipt 대기; 미포함 pending tx 보존을 재현하지 않음. node1이 실제 leader라는 assert 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=3; timeout="120s"
2. do="newAccount"; save="acct"; saveKey="acctKey"
3. do="sendTx"; on="node1"; from="0xc17d493883eaa3b4cceb0f214b273392d562f9d8"; to="$acct"; value="1"; expect="receipt"
4. do="stopNode"; on="node1"
5. expect="blockAdvance"; on="node2"; timeout="90s"; pollInterval="2s"
6. expect="rpc"; on="node2"; method="txpool_status"; select="pending"; compare="Equal"; is="0x0"
7. do="startNode"; on="node1"

### legacy-gasprice-below-min-rejected — tests/tc/go-stablenet/post-v1.0.0-change/common-all/12-legacy-gasprice-below-min-rejected.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/12-legacy-gasprice-below-min-rejected.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. 고정 최소 gasPrice/tipCap과 reject 조건을 go-wemix 정책으로 다시 정한다. top-level applicableChains에 wemix를 포함한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="sendTx"; expect="reject"; from="node1"; gas="21000"; gasPrice="1"; to="0x00000000000000000000000000000000C0FFEE0E"; value="1"
2. compare="GreaterOrEqual"; expect="blockNumber"; is="0x1"

### accesslist-gasprice-below-min-rejected — tests/tc/go-stablenet/post-v1.0.0-change/common-all/13-accesslist-gasprice-below-min-rejected.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/13-accesslist-gasprice-below-min-rejected.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. 고정 최소 gasPrice/tipCap과 reject 조건을 go-wemix 정책으로 다시 정한다. top-level applicableChains에 wemix를 포함한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. accessList=[]; do="sendTx"; expect="reject"; from="node1"; gas="21000"; gasPrice="1"; to="0x00000000000000000000000000000000C0FFEE0E"; value="1"
2. compare="GreaterOrEqual"; expect="blockNumber"; is="0x1"

### feecap-below-min-rejected — tests/tc/go-stablenet/post-v1.0.0-change/common-all/14-feecap-below-min-rejected.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/14-feecap-below-min-rejected.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. 고정 최소 gasPrice/tipCap과 reject 조건을 go-wemix 정책으로 다시 정한다. top-level applicableChains에 wemix를 포함한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="sendTx"; expect="reject"; from="node1"; gas="21000"; maxFeePerGas="1"; maxPriorityFeePerGas="1"; to="0x00000000000000000000000000000000C0FFEE0E"; value="1"
2. compare="GreaterOrEqual"; expect="blockNumber"; is="0x1"

### effective-gas-price-regular-bp-en — tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/02b-effective-gas-price-regular-bp-en.json:29`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. 동일 확정 블록에서 BP/EN 영수증을 비교한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=2; timeout="120s"
2. do="sendTx"; on="node1"; from="node1"; to="0x70997970C51812dc3A010C7d01b50e0d17dc79C8"; value="1"; gas="21000"; save="txHash"; expect="receipt"
3. do="read"; on="node1"; source="rpcCall"; method="eth_getTransactionReceipt"; params=["$txHash"]; select="blockNumber"; save="txBlock"
4. do="waitFor"; on="en1"; source="rpcCall"; method="eth_blockNumber"; params=[]; compare="GreaterOrEqual"; expected="$txBlock"; timeout="120s"; pollInterval="2s"
5. do="read"; on="node1"; source="rpcCall"; method="eth_getTransactionReceipt"; params=["$txHash"]; select="effectiveGasPrice"; save="egpBP"
6. expect="rpc"; on="en1"; method="eth_getTransactionReceipt"; params=["$txHash"]; select="effectiveGasPrice"; compare="Equal"; is="$egpBP"

### signature-compat-across-swap — tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/01b-signature-compat-across-swap.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. swap 대상은 이전/패치 go-wemix로 설정; genesis mismatch는 다른 genesis fixture를 준비하고 실제 기동 실패 및 DB 보존을 검증.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=3; timeout="120s"
2. do="sendTx"; on="node1"; from="node1"; to="0x70997970C51812dc3A010C7d01b50e0d17dc79C8"; value="1"; gas="21000"; save="legacyTx"; expect="receipt"
3. do="read"; on="node1"; source="rpcCall"; method="eth_getTransactionReceipt"; params=["$legacyTx"]; select="blockNumber"; save="blockBefore"
4. do="read"; on="node1"; source="rpcCall"; method="eth_getTransactionReceipt"; params=["$legacyTx"]; select="status"; save="statusBefore"
5. do="read"; on="node1"; source="rpcCall"; method="eth_getTransactionByHash"; params=["$legacyTx"]; select="from"; save="fromBefore"
6. do="read"; on="node1"; source="rpcCall"; method="eth_getTransactionByHash"; params=["$legacyTx"]; select="to"; save="toBefore"
7. do="swapNode"; on="en1"; binary="upgrade"; purpose="signature-compat"
8. do="waitFor"; on="en1"; source="rpcCall"; method="eth_blockNumber"; params=[]; compare="GreaterOrEqual"; expected="$blockBefore"; timeout="180s"; pollInterval="2s"
9. expect="rpc"; on="en1"; method="eth_getTransactionReceipt"; params=["$legacyTx"]; select="blockNumber"; compare="Equal"; is="$blockBefore"
10. expect="rpc"; on="en1"; method="eth_getTransactionReceipt"; params=["$legacyTx"]; select="status"; compare="Equal"; is="$statusBefore"
11. expect="rpc"; on="en1"; method="eth_getTransactionByHash"; params=["$legacyTx"]; select="from"; compare="Equal"; is="$fromBefore"
12. expect="rpc"; on="en1"; method="eth_getTransactionByHash"; params=["$legacyTx"]; select="to"; compare="Equal"; is="$toBefore"
13. do="readNodeLog"; on="en1"; save="upgradedNodeLog"

### genesis-mismatch-refuses-to-start — tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/02-genesis-mismatch.json:30`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. swap 대상은 이전/패치 go-wemix로 설정; genesis mismatch는 다른 genesis fixture를 준비하고 실제 기동 실패 및 DB 보존을 검증.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=3; timeout="120s"
2. expect="sameBlockHash"; block="latest"
3. do="swapNode"; on="node4"; binary="mismatch"; purpose="genesis-mismatch"; expect="fail"; reason="genesis"; save="mismatchEvidence"
4. do="readNodeLog"; on="node4"; save="node4Log"
5. expect="blockAdvance"; on="node1"; timeout="60s"; pollInterval="2s"

### genesis-block-hash-consistent — tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/04-genesis-block-hash-consistent.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/04-genesis-block-hash-consistent.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. swap 대상은 이전/패치 go-wemix로 설정; genesis mismatch는 다른 genesis fixture를 준비하고 실제 기동 실패 및 DB 보존을 검증. top-level applicableChains에 wemix를 포함한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="read"; method="eth_getBlockByNumber"; params=["0x0",false]; save="genesisHash"; select="hash"; source="rpcCall"
2. compare="NotEqual"; expect="rpcCall"; is="0x"; method="eth_getBlockByNumber"; params=["0x0",false]; select="hash"
3. block="0x0"; expect="sameBlockHash"
4. compare="Equal"; expect="rpcCall"; is="0x0000000000000000000000000000000000000000000000000000000000000000"; method="eth_getBlockByNumber"; params=["0x0",false]; select="parentHash"

### block-transactions-field — tests/tc/go-stablenet/regression/api/01-block-transactions-field.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/api/01-block-transactions-field.json:30`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="Regexp"; expect="rpcCall"; is="^0x[0-9a-fA-F]+$"; method="eth_getBlockByNumber"; params=["latest",false]; select="number"
2. compare="Regexp"; expect="rpcCall"; is="^0x[0-9a-fA-F]{64}$"; method="eth_getBlockByNumber"; params=["latest",false]; select="hash"
3. compare="NotNil"; expect="rpcCall"; is=null; method="eth_getBlockByNumber"; params=["latest",false]; select="transactions"

### block-by-hash-consistency — tests/tc/go-stablenet/regression/api/02-block-by-hash-consistency.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/api/02-block-by-hash-consistency.json:30`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="read"; method="eth_getBlockByNumber"; params=["latest",false]; save="hash"; select="hash"; source="rpcCall"
2. do="read"; method="eth_getBlockByNumber"; params=["latest",false]; save="number"; select="number"; source="rpcCall"
3. expect="rpcCall"; is="$hash"; method="eth_getBlockByHash"; params=["$hash",false]; select="hash"
4. expect="rpcCall"; is="$number"; method="eth_getBlockByHash"; params=["$hash",false]; select="number"

### transaction-by-hash-fields — tests/tc/go-stablenet/regression/api/03-transaction-by-hash-fields.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/api/03-transaction-by-hash-fields.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. top-level applicableChains에 wemix를 포함한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="sendTx"; from="node1"; gas="21000"; save="hash"; to="0x00000000000000000000000000000000C0FFEE05"; value="1"
2. compare="EqualCI"; expect="rpcCall"; is="0xc17d493883eaa3b4cceb0f214b273392d562f9d8"; method="eth_getTransactionByHash"; params=["$hash"]; select="from"
3. compare="EqualCI"; expect="rpcCall"; is="0x00000000000000000000000000000000C0FFEE05"; method="eth_getTransactionByHash"; params=["$hash"]; select="to"
4. compare="NotNil"; expect="rpcCall"; method="eth_getTransactionByHash"; params=["$hash"]; select="blockNumber"
5. compare="NotNil"; expect="rpcCall"; method="eth_getTransactionByHash"; params=["$hash"]; select="value"

### transaction-receipt-fields — tests/tc/go-stablenet/regression/api/04-transaction-receipt-fields.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/api/04-transaction-receipt-fields.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. top-level applicableChains에 wemix를 포함한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="sendTx"; from="node1"; gas="21000"; save="hash"; to="0x00000000000000000000000000000000C0FFEE06"; value="1"
2. compare="Equal"; expect="rpcCall"; is="0x1"; method="eth_getTransactionReceipt"; params=["$hash"]; select="status"
3. compare="NotNil"; expect="rpcCall"; method="eth_getTransactionReceipt"; params=["$hash"]; select="effectiveGasPrice"
4. compare="NotNil"; expect="rpcCall"; method="eth_getTransactionReceipt"; params=["$hash"]; select="logs"

### transaction-count-increments — tests/tc/go-stablenet/regression/api/05-transaction-count-increments.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/api/05-transaction-count-increments.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. top-level applicableChains에 wemix를 포함한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="read"; method="eth_getTransactionCount"; params=["0xc17d493883eaa3b4cceb0f214b273392d562f9d8","latest"]; save="before"; source="rpcCall"
2. do="sendTx"; from="node1"; gas="21000"; save="hash"; to="0x00000000000000000000000000000000C0FFEE04"; value="1"
3. do="read"; method="eth_getTransactionCount"; params=["0xc17d493883eaa3b4cceb0f214b273392d562f9d8","latest"]; save="after"; source="rpcCall"
4. expect="txStatus"; hash="$hash"; is="0x1"
5. compare="Equal"; expect="derive"; is="1"; of=["$after","$before"]; op="diff"

### txpool-status — tests/tc/go-stablenet/regression/api/18-txpool-status.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/api/18-txpool-status.json:30`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="Regexp"; expect="rpcCall"; is="^0x[0-9a-fA-F]+$"; method="txpool_status"; select="pending"
2. compare="Regexp"; expect="rpcCall"; is="^0x[0-9a-fA-F]+$"; method="txpool_status"; select="queued"

### txpool-content-well-formed — tests/tc/go-stablenet/regression/api/19-txpool-content-well-formed.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/api/19-txpool-content-well-formed.json:30`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="NotNil"; expect="rpcCall"; is=null; method="txpool_content"; select="pending"
2. compare="NotNil"; expect="rpcCall"; is=null; method="txpool_content"; select="queued"

### fee-delegate-sign-rpc-present — tests/tc/go-stablenet/regression/api/21-fee-delegate-sign-rpc-present.json

- `P3` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/api/21-fee-delegate-sign-rpc-present.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. top-level applicableChains에 wemix를 포함한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. expect="methodPresent"; method="eth_signRawFeeDelegateTransaction"; params=[{"from":"0x00000000000000000000000000000000C0FFEE08"},"0x00"]

### legacy-transfer — tests/tc/go-stablenet/regression/ethereum/08-legacy-transfer.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/ethereum/08-legacy-transfer.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. istanbul_getWbftExtraInfo 기반 fee 계산을 go-wemix fee RPC/정책으로 교체한다. top-level applicableChains에 wemix를 포함한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="read"; method="eth_blockNumber"; save="n"; source="rpcCall"
2. do="read"; method="eth_getBlockByNumber"; params=["$n",false]; save="base"; select="baseFeePerGas"; source="rpcCall"
3. do="read"; method="istanbul_getWbftExtraInfo"; params=["$n"]; save="tip"; select="gasTip"; source="rpcCall"
4. do="read"; of=["$base","$base","$tip"]; op="sum"; save="gp"; source="derive"
5. do="read"; method="eth_getBalance"; params=["0x00000000000000000000000000000000C0FFEE03","latest"]; save="before"; source="rpcCall"
6. do="sendTx"; from="node1"; gas="21000"; gasPrice="$gp"; save="hash"; to="0x00000000000000000000000000000000C0FFEE03"; value="1000000000000000000"
7. do="read"; method="eth_getBalance"; params=["0x00000000000000000000000000000000C0FFEE03","latest"]; save="after"; source="rpcCall"
8. expect="txStatus"; hash="$hash"; is="0x1"
9. compare="Equal"; expect="rpcCall"; is="0x0"; method="eth_getTransactionByHash"; params=["$hash"]; select="type"
10. compare="Equal"; expect="derive"; is="1000000000000000000"; of=["$after","$before"]; op="diff"

### dynamic-fee-tx — tests/tc/go-stablenet/regression/ethereum/09-dynamic-fee-tx.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/ethereum/09-dynamic-fee-tx.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. istanbul_getWbftExtraInfo 기반 fee 계산을 go-wemix fee RPC/정책으로 교체한다. top-level applicableChains에 wemix를 포함한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="read"; method="eth_blockNumber"; save="n"; source="rpcCall"
2. do="read"; method="eth_getBlockByNumber"; params=["$n",false]; save="base"; select="baseFeePerGas"; source="rpcCall"
3. do="read"; method="istanbul_getWbftExtraInfo"; params=["$n"]; save="tip"; select="gasTip"; source="rpcCall"
4. do="read"; of=["$base","$base","$tip"]; op="sum"; save="feecap"; source="derive"
5. do="sendTx"; from="node1"; gas="21000"; maxFeePerGas="$feecap"; maxPriorityFeePerGas="$tip"; save="hash"; to="0x00000000000000000000000000000000C0FFEE0C"; value="1"
6. expect="txStatus"; hash="$hash"; is="0x1"
7. compare="Equal"; expect="rpcCall"; is="0x2"; method="eth_getTransactionByHash"; params=["$hash"]; select="type"

### access-list-tx — tests/tc/go-stablenet/regression/ethereum/10-access-list-tx.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/ethereum/10-access-list-tx.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. top-level applicableChains에 wemix를 포함한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="read"; save="base"; source="baseFee"
2. do="read"; of=["$base","$base"]; op="sum"; save="gp"; source="derive"
3. do="read"; method="eth_createAccessList"; params=[{"from":"0xc17d493883eaa3b4cceb0f214b273392d562f9d8","to":"0x00000000000000000000000000000000C0FFEE0D","value":"0x1"},"latest"]; save="createdList"; select="accessList"; source="rpcCall"
4. accessList="$createdList"; do="sendTx"; from="node1"; gas="21000"; gasPrice="$gp"; save="hash"; to="0x00000000000000000000000000000000C0FFEE0D"; value="1"
5. expect="txStatus"; hash="$hash"; is="0x1"
6. compare="Equal"; expect="rpcCall"; is="0x1"; method="eth_getTransactionByHash"; params=["$hash"]; select="type"

### nonce-ordering — tests/tc/go-stablenet/regression/ethereum/11-nonce-ordering.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/ethereum/11-nonce-ordering.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. top-level applicableChains에 wemix를 포함한다. env.topology.en=1을 추가하여 on=en1 참조를 충족한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="newAccount"; save="acct"; saveKey="acctKey"
2. do="sendTx"; from="node1"; gas="21000"; on="node1"; save="fundHash"; to="$acct"; value="10000000000000000000"
3. do="sendTx"; key="$acctKey"; maxFeePerGas="100000000000000"; maxPriorityFeePerGas="30000000000000"; nonce="2"; on="en1"; to="0x00000000000000000000000000000000C0FFEE10"; value="1"; wait=false
4. do="sendTx"; key="$acctKey"; maxFeePerGas="100000000000000"; maxPriorityFeePerGas="30000000000000"; nonce="1"; on="en1"; to="0x00000000000000000000000000000000C0FFEE10"; value="1"; wait=false
5. do="sendTx"; key="$acctKey"; maxFeePerGas="100000000000000"; maxPriorityFeePerGas="30000000000000"; nonce="0"; on="en1"; to="0x00000000000000000000000000000000C0FFEE10"; value="1"; wait=false
6. address="$acct"; compare="Equal"; do="waitFor"; expected="3"; pollInterval="2s"; source="nonceAt"; timeout="90s"
7. expect="txStatus"; hash="$fundHash"; is="0x1"
8. address="$acct"; compare="Equal"; expect="nonceAt"; is="3"

### out-of-order-nonces-mine — tests/tc/go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/ethereum/11b-out-of-order-nonces-mine.json:30`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. top-level applicableChains에 wemix를 포함한다. env.topology.en=1을 추가하여 on=en1 참조를 충족한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="newAccount"; save="acct"; saveKey="acctKey"
2. do="sendTx"; from="node1"; gas="21000"; on="node1"; save="fundHash"; to="$acct"; value="10000000000000000000"
3. do="sendTx"; key="$acctKey"; maxFeePerGas="100000000000000"; maxPriorityFeePerGas="30000000000000"; nonce="2"; on="en1"; to="0x00000000000000000000000000000000C0FFEE11"; value="1"; wait=false
4. do="sendTx"; key="$acctKey"; maxFeePerGas="100000000000000"; maxPriorityFeePerGas="30000000000000"; nonce="1"; on="en1"; to="0x00000000000000000000000000000000C0FFEE11"; value="1"; wait=false
5. do="sendTx"; key="$acctKey"; maxFeePerGas="100000000000000"; maxPriorityFeePerGas="30000000000000"; nonce="0"; on="en1"; to="0x00000000000000000000000000000000C0FFEE11"; value="1"; wait=false
6. address="$acct"; compare="Equal"; do="waitFor"; expected="3"; pollInterval="2s"; source="nonceAt"; timeout="90s"
7. expect="txStatus"; hash="$fundHash"; is="0x1"
8. address="$acct"; compare="Equal"; expect="nonceAt"; is="3"

### dynamic-fee-below-basefee-rejected — tests/tc/go-stablenet/regression/ethereum/12-dynamic-fee-below-basefee-rejected.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/ethereum/12-dynamic-fee-below-basefee-rejected.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. 파일명은 baseFee지만 원문 의도는 MinTip 거부: feeCap<baseFee와 tip 하한을 별개로 검증한다. top-level applicableChains에 wemix를 포함한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="sendTx"; expect="reject"; from="node1"; gas="21000"; maxFeePerGas="1"; maxPriorityFeePerGas="1"; to="0x00000000000000000000000000000000C0FFEE10"; value="1"
2. compare="GreaterOrEqual"; expect="blockNumber"; is="0x1"

### insufficient-funds-rejected — tests/tc/go-stablenet/regression/ethereum/14-insufficient-funds-rejected.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/ethereum/14-insufficient-funds-rejected.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. top-level applicableChains에 wemix를 포함한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. address="node1"; do="read"; save="bal"; source="balanceAt"
2. do="read"; of=["$bal","1000000000000000000"]; op="sum"; save="over"; source="derive"
3. do="sendTx"; expect="reject"; from="node1"; gas="21000"; to="0x00000000000000000000000000000000C0FFEE07"; value="$over"
4. compare="GreaterOrEqual"; expect="blockNumber"; is="0x1"

### gas-limit-exceeds-block-rejected — tests/tc/go-stablenet/regression/ethereum/15-gas-limit-exceeds-block-rejected.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/ethereum/15-gas-limit-exceeds-block-rejected.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. top-level applicableChains에 wemix를 포함한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="read"; method="eth_getBlockByNumber"; params=["latest",false]; save="gl"; select="gasLimit"; source="rpcCall"
2. do="read"; of=["$gl","1"]; op="sum"; save="over"; source="derive"
3. do="sendTx"; expect="reject"; from="node1"; gas="$over"; to="0x00000000000000000000000000000000C0FFEE10"; value="1"
4. compare="GreaterOrEqual"; expect="blockNumber"; is="0x1"

### effective-gas-price — tests/tc/go-stablenet/regression/ethereum/16-effective-gas-price.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/ethereum/16-effective-gas-price.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. top-level applicableChains에 wemix를 포함한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="sendTx"; from="node1"; gas="21000"; save="hash"; to="0x00000000000000000000000000000000C0FFEE0B"; value="1"
2. expect="txStatus"; hash="$hash"; is="0x1"
3. compare="Greater"; expect="rpcCall"; is="0"; method="eth_getTransactionReceipt"; params=["$hash"]; select="effectiveGasPrice"
4. compare="GreaterOrEqual"; expect="rpcCall"; is="21000"; method="eth_getTransactionReceipt"; params=["$hash"]; select="gasUsed"

### replacement-tx — tests/tc/go-stablenet/regression/ethereum/17-replacement-tx.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/ethereum/17-replacement-tx.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. top-level applicableChains에 wemix를 포함한다. env.topology.en=1을 추가하여 on=en1 참조를 충족한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="newAccount"; save="acct"; saveKey="acctKey"
2. do="sendTx"; from="node1"; gas="21000"; on="node1"; save="fundHash"; to="$acct"; value="10000000000000000000"
3. do="sendTx"; key="$acctKey"; maxFeePerGas="100000000000000"; maxPriorityFeePerGas="30000000000000"; nonce="1"; on="en1"; save="tx1"; to="0x00000000000000000000000000000000C0FFEE12"; value="1"; wait=false
4. do="sendTx"; key="$acctKey"; maxFeePerGas="120000000000000"; maxPriorityFeePerGas="36000000000000"; nonce="1"; on="en1"; save="tx2"; to="0x00000000000000000000000000000000C0FFEE12"; value="1"; wait=false
5. do="sendTx"; key="$acctKey"; maxFeePerGas="100000000000000"; maxPriorityFeePerGas="30000000000000"; nonce="0"; on="en1"; save="tx3"; to="0x00000000000000000000000000000000C0FFEE12"; value="1"; wait=false
6. compare="Equal"; do="waitFor"; expected="true"; hash="$tx2"; pollInterval="2s"; source="txMined"; timeout="90s"
7. expect="txStatus"; hash="$fundHash"; is="0x1"
8. expect="txMined"; hash="$tx2"; is="true"
9. expect="txMined"; hash="$tx1"; is="false"

### same-nonce-replacement — tests/tc/go-stablenet/regression/ethereum/17b-same-nonce-replacement.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/ethereum/17b-same-nonce-replacement.json:30`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. top-level applicableChains에 wemix를 포함한다. env.topology.en=1을 추가하여 on=en1 참조를 충족한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="newAccount"; save="acct"; saveKey="acctKey"
2. do="sendTx"; from="node1"; gas="21000"; on="node1"; save="fundHash"; to="$acct"; value="10000000000000000000"
3. do="sendTx"; key="$acctKey"; maxFeePerGas="100000000000000"; maxPriorityFeePerGas="30000000000000"; nonce="1"; on="en1"; save="tx1"; to="0x00000000000000000000000000000000C0FFEE13"; value="1"; wait=false
4. do="sendTx"; key="$acctKey"; maxFeePerGas="120000000000000"; maxPriorityFeePerGas="36000000000000"; nonce="1"; on="en1"; save="tx2"; to="0x00000000000000000000000000000000C0FFEE13"; value="1"; wait=false
5. do="sendTx"; key="$acctKey"; maxFeePerGas="100000000000000"; maxPriorityFeePerGas="30000000000000"; nonce="0"; on="en1"; save="tx3"; to="0x00000000000000000000000000000000C0FFEE13"; value="1"; wait=false
6. compare="Equal"; do="waitFor"; expected="true"; hash="$tx2"; pollInterval="2s"; source="txMined"; timeout="90s"
7. expect="txStatus"; hash="$fundHash"; is="0x1"
8. expect="txMined"; hash="$tx2"; is="true"
9. expect="txMined"; hash="$tx1"; is="false"

### contract-roundtrip — tests/tc/go-stablenet/regression/ethereum/19-contract-roundtrip.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/ethereum/19-contract-roundtrip.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. top-level applicableChains에 wemix를 포함한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. bytecode="0x600a600c600039600a6000f3602a60005260206000f3"; do="deployContract"; from="node1"; gas="200000"; save="contract"
2. address="$contract"; compare="NotEqual"; expect="codeAt"; is="0x"
3. compare="Equal"; data="0x"; expect="call"; is="0x000000000000000000000000000000000000000000000000000000000000002a"; to="$contract"

### eth-call-revert-returns-error — tests/tc/go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/ethereum/23-eth-call-revert-returns-error.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. top-level applicableChains에 wemix를 포함한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. data=hex payload (132 bytes; 원문 참조); do="sendTx"; from="node1"; gas="300000"; on="node1"; save="dhash"
2. do="read"; method="eth_getTransactionReceipt"; params=["$dhash"]; save="addr"; select="contractAddress"; source="rpcCall"
3. address="$addr"; compare="NotEqual"; do="waitFor"; expected="0x"; pollInterval="1s"; source="codeAt"; timeout="60s"
4. expect="txStatus"; hash="$dhash"; is="0x1"
5. data="0xa9cc4718"; expect="callError"; to="$addr"

### revert-tx-status-zero — tests/tc/go-stablenet/regression/ethereum/24-revert-tx-status-zero.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/ethereum/24-revert-tx-status-zero.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. top-level applicableChains에 wemix를 포함한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음. revert status 검증은 있으나 잔액 gas-only 차감 검증 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. data="0x600580600b6000396000f360006000fd"; do="sendTx"; from="node1"; gas="200000"; save="dhash"
2. do="read"; method="eth_getTransactionReceipt"; params=["$dhash"]; save="addr"; select="contractAddress"; source="rpcCall"
3. do="sendTx"; expect="revert"; from="node1"; gas="100000"; save="chash"; to="$addr"
4. expect="txStatus"; hash="$dhash"; is="0x1"
5. compare="Equal"; expect="rpcCall"; is="0x0"; method="eth_getTransactionReceipt"; params=["$chash"]; select="status"

### out-of-gas-consumes-all — tests/tc/go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/ethereum/25-out-of-gas-consumes-all.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. top-level applicableChains에 wemix를 포함한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음. gasUsed 비교는 있으나 제목의 잔액 전량 차감 검증 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. data="0x600480600b6000396000f35b600056"; do="sendTx"; from="node1"; gas="200000"; save="dhash"
2. do="read"; method="eth_getTransactionReceipt"; params=["$dhash"]; save="addr"; select="contractAddress"; source="rpcCall"
3. do="sendTx"; expect="revert"; from="node1"; gas="50000"; save="chash"; to="$addr"
4. do="read"; method="eth_getTransactionReceipt"; params=["$chash"]; save="used"; select="gasUsed"; source="rpcCall"
5. expect="txStatus"; hash="$dhash"; is="0x1"
6. compare="Equal"; expect="derive"; is="0"; of=["$used","50000"]; op="diff"

### value-transfer — tests/tc/go-stablenet/regression/ethereum/27b-value-transfer.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/ethereum/27b-value-transfer.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. top-level applicableChains에 wemix를 포함한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="read"; method="eth_getBalance"; params=["0x00000000000000000000000000000000C0FFEE01","latest"]; save="before"; source="rpcCall"
2. do="sendTx"; from="node1"; gas="21000"; save="hash"; to="0x00000000000000000000000000000000C0FFEE01"; value="1000000000000000000"
3. do="read"; method="eth_getBalance"; params=["0x00000000000000000000000000000000C0FFEE01","latest"]; save="after"; source="rpcCall"
4. expect="txStatus"; hash="$hash"; is="0x1"
5. compare="Equal"; expect="derive"; is="1000000000000000000"; of=["$after","$before"]; op="diff"

### contract-event-emitted — tests/tc/go-stablenet/regression/ethereum/36-contract-event-emitted.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/ethereum/36-contract-event-emitted.json:30`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. top-level applicableChains에 wemix를 포함한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. data="0x6027600c60003960276000f37f111111111111111111111111111111111111111111111111111111111111111160006000a100"; do="sendTx"; from="node1"; gas="200000"; save="dhash"
2. do="read"; method="eth_getTransactionReceipt"; params=["$dhash"]; save="addr"; select="contractAddress"; source="rpcCall"
3. do="sendTx"; from="node1"; gas="100000"; save="ehash"; to="$addr"
4. do="read"; method="eth_getTransactionReceipt"; params=["$ehash"]; save="blk"; select="blockNumber"; source="rpcCall"
5. expect="txStatus"; hash="$ehash"; is="0x1"
6. address="$addr"; compare="Equal"; expect="logs"; fromBlock="$blk"; is="0x1111111111111111111111111111111111111111111111111111111111111111"; select="topic0"; toBlock="$blk"; topics=["0x1111111111111111111111111111111111111111111111111111111111111111"]

### fee-delegated-transfer — tests/tc/go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/01-fee-delegated-transfer.json:38`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. fee payer 서명·비용 차감 및 tamper 거부 이유가 동일한지 확인한다. top-level applicableChains에 wemix를 포함한다. env.topology.en=1을 추가하여 on=en1 참조를 충족한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/pr-head/core/types/transaction.go:198`, `sources/pr-head/internal/ethapi/api.go:2404`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="newAccount"; save="sender"; saveKey="senderKey"
2. do="newAccount"; save="feePayer"; saveKey="feePayerKey"
3. do="sendTx"; from="node1"; gas="21000"; on="node1"; save="fundSenderHash"; to="$sender"; value="10000000000000000000"
4. do="sendTx"; from="node1"; gas="21000"; on="node1"; save="fundFeeHash"; to="$feePayer"; value="10000000000000000000"
5. do="sendTx"; feePayerKey="$feePayerKey"; key="$senderKey"; on="en1"; save="hash"; to="0x00000000000000000000000000000000C0FFEE02"; value="1000000000000000000"
6. expect="txStatus"; hash="$fundSenderHash"; is="0x1"
7. expect="txStatus"; hash="$fundFeeHash"; is="0x1"
8. expect="txStatus"; hash="$hash"; is="0x1"
9. address="0x00000000000000000000000000000000C0FFEE02"; compare="Equal"; expect="balanceAt"; is="1000000000000000000"

### fd-sender-sig-invalid-rejected — tests/tc/go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/02-fd-sender-sig-invalid-rejected.json:38`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. fee payer 서명·비용 차감 및 tamper 거부 이유가 동일한지 확인한다. top-level applicableChains에 wemix를 포함한다. env.topology.en=1을 추가하여 on=en1 참조를 충족한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/pr-head/core/types/transaction.go:198`, `sources/pr-head/internal/ethapi/api.go:2404`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="newAccount"; save="acct"; saveKey="acctKey"
2. do="sendTx"; from="node1"; gas="21000"; on="node1"; save="fundHash"; to="$acct"; value="10000000000000000000"
3. do="sendRawTampered"; feePayerKey="$acctKey"; on="en1"; senderKey="$acctKey"; to="0x00000000000000000000000000000000C0FFEE20"; value="1"; which="sender"
4. expect="txStatus"; hash="$fundHash"; is="0x1"
5. compare="GreaterOrEqual"; expect="blockNumber"; is="1"

### fd-feepayer-sig-invalid-rejected — tests/tc/go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/03-fd-feepayer-sig-invalid-rejected.json:38`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. fee payer 서명·비용 차감 및 tamper 거부 이유가 동일한지 확인한다. top-level applicableChains에 wemix를 포함한다. env.topology.en=1을 추가하여 on=en1 참조를 충족한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/pr-head/core/types/transaction.go:198`, `sources/pr-head/internal/ethapi/api.go:2404`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="newAccount"; save="acct"; saveKey="acctKey"
2. do="sendTx"; from="node1"; gas="21000"; on="node1"; save="fundHash"; to="$acct"; value="10000000000000000000"
3. do="sendRawTampered"; feePayerKey="$acctKey"; on="en1"; senderKey="$acctKey"; to="0x00000000000000000000000000000000C0FFEE21"; value="1"; which="feepayer"
4. expect="txStatus"; hash="$fundHash"; is="0x1"
5. compare="GreaterOrEqual"; expect="blockNumber"; is="1"

### feepayer-insufficient-rejected — tests/tc/go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/04-feepayer-insufficient-rejected.json:38`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. fee payer 서명·비용 차감 및 tamper 거부 이유가 동일한지 확인한다. top-level applicableChains에 wemix를 포함한다. env.topology.en=1을 추가하여 on=en1 참조를 충족한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/pr-head/core/types/transaction.go:198`, `sources/pr-head/internal/ethapi/api.go:2404`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="newAccount"; save="sender"; saveKey="senderKey"
2. do="newAccount"; save="feePayer"; saveKey="feePayerKey"
3. do="sendTx"; from="node1"; gas="21000"; on="node1"; save="fundHash"; to="$sender"; value="10000000000000000000"
4. do="sendTx"; expect="reject"; feePayerKey="$feePayerKey"; key="$senderKey"; on="en1"; to="0x70997970C51812dc3A010C7d01b50e0d17dc79C8"; value="1"
5. expect="txStatus"; hash="$fundHash"; is="0x1"
6. compare="GreaterOrEqual"; expect="blockNumber"; is="1"

### fee-delegated-sender-sig-invalid-rejected — tests/tc/go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/05-fee-delegated-sender-sig-invalid-rejected.json:37`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. fee payer 서명·비용 차감 및 tamper 거부 이유가 동일한지 확인한다. top-level applicableChains에 wemix를 포함한다. env.topology.en=1을 추가하여 on=en1 참조를 충족한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/pr-head/core/types/transaction.go:198`, `sources/pr-head/internal/ethapi/api.go:2404`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="newAccount"; save="acct"; saveKey="acctKey"
2. do="sendTx"; from="node1"; gas="21000"; on="node1"; save="fundHash"; to="$acct"; value="10000000000000000000"
3. do="sendRawTampered"; feePayerKey="$acctKey"; on="en1"; senderKey="$acctKey"; to="0x00000000000000000000000000000000C0FFEE22"; value="1"; which="sender"
4. expect="txStatus"; hash="$fundHash"; is="0x1"
5. compare="GreaterOrEqual"; expect="blockNumber"; is="1"

### fee-delegated-feepayer-sig-invalid-rejected — tests/tc/go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/06-fee-delegated-feepayer-sig-invalid-rejected.json:37`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. fee payer 서명·비용 차감 및 tamper 거부 이유가 동일한지 확인한다. top-level applicableChains에 wemix를 포함한다. env.topology.en=1을 추가하여 on=en1 참조를 충족한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/pr-head/core/types/transaction.go:198`, `sources/pr-head/internal/ethapi/api.go:2404`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="newAccount"; save="acct"; saveKey="acctKey"
2. do="sendTx"; from="node1"; gas="21000"; on="node1"; save="fundHash"; to="$acct"; value="10000000000000000000"
3. do="sendRawTampered"; feePayerKey="$acctKey"; on="en1"; senderKey="$acctKey"; to="0x00000000000000000000000000000000C0FFEE23"; value="1"; which="feepayer"
4. expect="txStatus"; hash="$fundHash"; is="0x1"
5. compare="GreaterOrEqual"; expect="blockNumber"; is="1"

### fee-delegated-unfunded-feepayer-rejected — tests/tc/go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/fee-delegation/07-fee-delegated-unfunded-feepayer-rejected.json:37`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. fee payer 서명·비용 차감 및 tamper 거부 이유가 동일한지 확인한다. top-level applicableChains에 wemix를 포함한다. env.topology.en=1을 추가하여 on=en1 참조를 충족한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/pr-head/core/types/transaction.go:198`, `sources/pr-head/internal/ethapi/api.go:2404`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="newAccount"; save="sender"; saveKey="senderKey"
2. do="newAccount"; save="feePayer"; saveKey="feePayerKey"
3. do="sendTx"; from="node1"; gas="21000"; on="node1"; save="fundHash"; to="$sender"; value="10000000000000000000"
4. do="sendTx"; expect="reject"; feePayerKey="$feePayerKey"; key="$senderKey"; on="en1"; to="0x00000000000000000000000000000000C0FFEE10"; value="1"
5. expect="txStatus"; hash="$fundHash"; is="0x1"
6. compare="GreaterOrEqual"; expect="blockNumber"; is="1"

### wemix-chain-up — tests/tc/go-wemix/chain-up/01-wemix-chain-up.json

- `P1` / `direct` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-wemix/chain-up/01-wemix-chain-up.json:26`
- 적용: PR 패치 바이너리 및 preset/generated keys·자금·PoA/etcd 준비. docker15 경로는 해당 실행 환경에서만 유효.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=2; timeout="180s"
2. expect="chainId"; compare="Greater"; is="0"
3. expect="blockNumber"; compare="GreaterOrEqual"; is="2"
4. expect="rpc"; method="net_peerCount"; compare="NotEqual"; is="0x0"

### wemix-chain-up-15 — tests/tc/go-wemix/chain-up/02-wemix-chain-up-15.json

- `P1` / `direct` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-wemix/chain-up/02-wemix-chain-up-15.json:27`
- 적용: PR 패치 바이너리 및 preset/generated keys·자금·PoA/etcd 준비. docker15 경로는 해당 실행 환경에서만 유효.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=2; timeout="300s"
2. expect="chainId"; compare="Greater"; is="0"
3. expect="blockNumber"; compare="GreaterOrEqual"; is="2"
4. expect="rpc"; method="net_peerCount"; compare="NotEqual"; is="0x0"

### wemix-node-crash — tests/tc/go-wemix/fault/01-wemix-node-crash.json

- `P1` / `direct` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-wemix/fault/01-wemix-node-crash.json:13`
- 적용: PR 패치 바이너리 및 preset/generated keys·자금·PoA/etcd 준비. docker15 경로는 해당 실행 환경에서만 유효.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음. restart 후 동기화/동일 hash assert 없음; 실제 leader 중단 보장 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=3; timeout="300s"
2. do="stopNode"; on="node3"
3. expect="blockAdvance"; on="node1"; timeout="180s"; pollInterval="3s"
4. do="restartNode"; on="node3"

### wemix-brioche-block-reward — tests/tc/go-wemix/rpc/01-wemix-brioche-block-reward.json

- `P2` / `direct` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-wemix/rpc/01-wemix-brioche-block-reward.json:39`
- 적용: PR 패치 바이너리 및 preset/generated keys·자금·PoA/etcd 준비. docker15 경로는 해당 실행 환경에서만 유효.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=2; timeout="180s"
2. do="read"; source="rpcCall"; method="wemix_getBriocheBlockReward"; params=["0x5"]; save="pre"
3. expect="rpc"; method="wemix_getBriocheBlockReward"; params=["0x5"]; compare="Greater"; is="0"
4. expect="rpc"; method="wemix_getBriocheBlockReward"; params=["0xf"]; compare="Less"; is="$pre"

### wemix-tx-and-contract — tests/tc/go-wemix/tx/01-wemix-tx-and-contract.json

- `P1` / `direct` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-wemix/tx/01-wemix-tx-and-contract.json:35`
- 적용: PR 패치 바이너리 및 preset/generated keys·자금·PoA/etcd 준비. docker15 경로는 해당 실행 환경에서만 유효.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=3; timeout="180s"
2. do="newAccount"; save="r"; saveKey="rk"
3. do="sendTx"; on="node1"; from="0xc17d493883eaa3b4cceb0f214b273392d562f9d8"; to="$r"; value="1"; expect="receipt"
4. expect="balanceAt"; on="node1"; address="$r"; compare="Greater"; is="0"
5. do="deployContract"; on="node1"; from="0xc17d493883eaa3b4cceb0f214b273392d562f9d8"; bytecode="0x600a600c600039600a6000f3602a60005260206000f3"; gas="300000"; save="c"
6. expect="codeAt"; on="node1"; address="$c"; compare="NotEqual"; is="0x"
7. expect="call"; on="node1"; to="$c"; data="0x"; compare="Equal"; is="0x000000000000000000000000000000000000000000000000000000000000002a"

### stress-block-time — tests/tc/stress/01-stress-block-time.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/stress/01-stress-block-time.json:28`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음. 설명의 100블록 통계 대신 blocks=15/maxSeconds=60 검사.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/chainbench/internal/testhelper/load.go:20`, `sources/chainbench/internal/testhelper/load.go:69`

1. do="waitBlock"; target=20; timeout="180s"
2. expect="blockInterval"; on="node1"; blocks=15; maxSeconds=60

### stress-tx-flood — tests/tc/stress/02-stress-tx-flood.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/stress/02-stress-tx-flood.json:28`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음. load는 짧은 gas-burn initcode 한 건/블록; 30% gas×15블록, 동시 flood/TPS/대형 RLP 검증 아님.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/chainbench/internal/testhelper/load.go:20`, `sources/chainbench/internal/testhelper/load.go:69`

1. do="waitBlock"; target=2; timeout="60s"
2. do="load"; from="0xc17d493883eaa3b4cceb0f214b273392d562f9d8"; fillPercent=30; blocks=15; timeout="90s"
3. expect="blockAdvance"; on="node1"; timeout="60s"; pollInterval="2s"

### gas-price-positive — tests/tc/go-stablenet/regression/api/07-gas-price-positive.json

- `P2` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/api/07-gas-price-positive.json:30`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="Greater"; expect="gasPrice"; is="0"

### fee-history-well-formed — tests/tc/go-stablenet/regression/api/09-fee-history-well-formed.json

- `P2` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/api/09-fee-history-well-formed.json:30`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="Regexp"; expect="rpcCall"; is="^0x[0-9a-fA-F]+$"; method="eth_feeHistory"; params=["0x10","latest",[]]; select="oldestBlock"
2. compare="NotNil"; expect="rpcCall"; is=null; method="eth_feeHistory"; params=["0x10","latest",[]]; select="baseFeePerGas"

### admin-peers-populated — tests/tc/go-stablenet/regression/api/20-admin-peers-populated.json

- `P2` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/api/20-admin-peers-populated.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="GreaterOrEqual"; expect="rpcCall"; is="1"; method="admin_peers"; select="#"
2. compare="NotEqual"; expect="rpcCall"; is=""; method="admin_peers"; select="0.id"

### chain-not-syncing — tests/tc/go-stablenet/regression/api/24-chain-not-syncing.json

- `P2` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/api/24-chain-not-syncing.json:29`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. expect="rpcCall"; is=false; method="eth_syncing"

### stablenet-chain-up — tests/tc/go-stablenet/regression/ethereum/01-stablenet-chain-up.json

- `P2` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/ethereum/01-stablenet-chain-up.json:27`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. istanbul_getValidators expect를 PoA 참여/etcd 검증으로 교체한다. istanbul_getWbftExtraInfo 기반 fee 계산을 go-wemix fee RPC/정책으로 교체한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=2; timeout="120s"
2. expect="chainId"; compare="Greater"; is="0"
3. expect="blockNumber"; compare="GreaterOrEqual"; is="2"
4. expect="rpc"; method="istanbul_getValidators"; params=["latest"]; compare="Len"; is=4
5. expect="rpc"; method="net_peerCount"; compare="NotEqual"; is="0x0"
6. expect="sameBlockHash"; block="0x0"

### estimate-gas — tests/tc/go-stablenet/regression/ethereum/22-estimate-gas.json

- `P2` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/ethereum/22-estimate-gas.json:30`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="GreaterOrEqual"; data="0x"; expect="estimateGas"; is="21000"; to="0x0000000000000000000000000000000000000001"

### genesis-balance — tests/tc/go-stablenet/regression/ethereum/27-genesis-balance.json

- `P2` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/ethereum/27-genesis-balance.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. top-level applicableChains에 wemix를 포함한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. address="node1"; compare="Greater"; expect="balanceAt"; is="0"

### logs-query-well-formed — tests/tc/go-stablenet/regression/ethereum/29-logs-query-well-formed.json

- `P2` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/ethereum/29-logs-query-well-formed.json:30`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="GreaterOrEqual"; expect="logs"; fromBlock="latest"; is="0"; select="count"; toBlock="latest"

### chain-id — tests/tc/go-stablenet/regression/ethereum/30-chain-id.json

- `P3` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/ethereum/30-chain-id.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. top-level applicableChains에 wemix를 포함한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. compare="Regexp"; expect="rpcCall"; is="^0x[0-9a-fA-F]+$"; method="eth_chainId"
2. compare="Greater"; expect="chainId"; is="0"

### ws-subscribe-new-heads — tests/tc/go-stablenet/regression/ethereum/31-ws-subscribe-new-heads.json

- `P2` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/ethereum/31-ws-subscribe-new-heads.json:33`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. top-level applicableChains에 wemix를 포함한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. count=1; event="newHeads"; expect="wsSubscribe"; on="bp1"; timeout="30s"

### ws-subscribe-logs — tests/tc/go-stablenet/regression/ethereum/32-ws-subscribe-logs.json

- `P2` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/ethereum/32-ws-subscribe-logs.json:33`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. top-level applicableChains에 wemix를 포함한다. env.topology.en=1을 추가하여 on=en1 참조를 충족한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. data="0x6027600c60003960276000f37f111111111111111111111111111111111111111111111111111111111111111160006000a100"; do="sendTx"; from="node1"; gas="200000"; on="node1"; save="dhash"
2. do="read"; method="eth_getTransactionReceipt"; params=["$dhash"]; save="addr"; select="contractAddress"; source="rpcCall"
3. address="$addr"; do="wsOpen"; event="logs"; on="en1"; save="logsub"
4. do="sendTx"; from="node1"; gas="100000"; on="node1"; save="ehash"; to="$addr"
5. expect="txStatus"; hash="$dhash"; is="0x1"
6. expect="txStatus"; hash="$ehash"; is="0x1"
7. count=1; expect="wsCollected"; sub="$logsub"; timeout="30s"

### stablenet-chain-up-15 — tests/tc/go-stablenet/regression/ethereum/33-stablenet-chain-up-15.json

- `P2` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/ethereum/33-stablenet-chain-up-15.json:28`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. istanbul_getValidators expect를 PoA 참여/etcd 검증으로 교체한다. istanbul_getWbftExtraInfo 기반 fee 계산을 go-wemix fee RPC/정책으로 교체한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=2; timeout="180s"
2. expect="chainId"; compare="Greater"; is="0"
3. expect="blockNumber"; compare="GreaterOrEqual"; is="2"
4. expect="rpc"; method="istanbul_getValidators"; params=["latest"]; compare="Len"; is=13
5. expect="rpc"; method="net_peerCount"; compare="NotEqual"; is="0x0"

### stablenet-proxied-pn-routing — tests/tc/go-stablenet/topology/01-proxied-pn-routing.json

- `P2` / `conditional` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/topology/01-proxied-pn-routing.json:30`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. PoA 노드/peer 라우팅 지원이 확인되어야 실행 대상으로 승격.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=2; timeout="180s"
2. do="waitFor"; source="blockNumber"; on="en1"; compare="GreaterOrEqual"; expected=1; timeout="120s"
3. expect="blockNumber"; on="en1"; compare="GreaterOrEqual"; is=1

### stablenet-negative-tx-revert — tests/tc/go-stablenet/tx/01-negative-tx-revert.json

- `P2` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/tx/01-negative-tx-revert.json:13`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=2; timeout="120s"
2. do="deployContract"; on="node1"; from="0xc17d493883eaa3b4cceb0f214b273392d562f9d8"; bytecode="0x600580600b6000396000f360006000fd"; gas="300000"; save="rv"
3. expect="codeAt"; on="node1"; address="$rv"; compare="Equal"; is="0x60006000fd"
4. do="sendTx"; on="node1"; from="0xc17d493883eaa3b4cceb0f214b273392d562f9d8"; to="$rv"; gas="100000"; expect="revert"

### wbft-chain-up — tests/tc/go-wbft/chain-up/01-wbft-chain-up.json

- `P2` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-wbft/chain-up/01-wbft-chain-up.json:27`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. Istanbul validator 검사를 PoA/etcd 참여 조건으로 교체하고 3/4 WBFT 설명을 제거한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=2; timeout="120s"
2. expect="chainId"; compare="Greater"; is="0"
3. expect="blockNumber"; compare="GreaterOrEqual"; is="2"
4. expect="rpc"; method="istanbul_getValidators"; params=["latest"]; compare="Len"; is=4
5. expect="rpc"; method="net_peerCount"; compare="NotEqual"; is="0x0"

### wbft-chain-up-15 — tests/tc/go-wbft/chain-up/02-wbft-chain-up-15.json

- `P2` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-wbft/chain-up/02-wbft-chain-up-15.json:28`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. Istanbul validator 검사를 PoA/etcd 참여 조건으로 교체하고 3/4 WBFT 설명을 제거한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=2; timeout="180s"
2. expect="chainId"; compare="Greater"; is="0"
3. expect="blockNumber"; compare="GreaterOrEqual"; is="2"
4. expect="rpc"; method="istanbul_getValidators"; params=["latest"]; compare="Len"; is=13
5. expect="rpc"; method="net_peerCount"; compare="NotEqual"; is="0x0"

### e1-mixed-producers — tests/tc/go-wbft/consensus/01-e1-mixed-producers.json

- `P2` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-wbft/consensus/01-e1-mixed-producers.json:44`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. Istanbul validator 검사를 PoA/etcd 참여 조건으로 교체하고 3/4 WBFT 설명을 제거한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=3; timeout="120s"
2. expect="rpc"; method="istanbul_getValidators"; params=["latest"]; compare="Len"; is=3
3. expect="blockAdvance"; on="node1"; timeout="60s"; pollInterval="2s"

### wbft-node-crash — tests/tc/go-wbft/fault/01-wbft-node-crash.json

- `P2` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-wbft/fault/01-wbft-node-crash.json:13`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. Istanbul validator 검사를 PoA/etcd 참여 조건으로 교체하고 3/4 WBFT 설명을 제거한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음. restart 후 동기화/동일 hash assert 없음; 실제 leader 중단 보장 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=3; timeout="180s"
2. do="stopNode"; on="node3"
3. expect="blockAdvance"; on="node1"; timeout="120s"; pollInterval="2s"
4. do="restartNode"; on="node3"

### wbft-tx-and-contract — tests/tc/go-wbft/tx/01-wbft-tx-and-contract.json

- `P2` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-wbft/tx/01-wbft-tx-and-contract.json:14`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. Istanbul validator 검사를 PoA/etcd 참여 조건으로 교체하고 3/4 WBFT 설명을 제거한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=3; timeout="180s"
2. do="newAccount"; save="r"; saveKey="rk"
3. do="sendTx"; on="node1"; from="0xc17d493883eaa3b4cceb0f214b273392d562f9d8"; to="$r"; value="1"; expect="receipt"
4. expect="balanceAt"; on="node1"; address="$r"; compare="Greater"; is="0"
5. do="deployContract"; on="node1"; from="0xc17d493883eaa3b4cceb0f214b273392d562f9d8"; bytecode="0x600a600c600039600a6000f3602a60005260206000f3"; gas="300000"; save="c"
6. expect="codeAt"; on="node1"; address="$c"; compare="NotEqual"; is="0x"
7. expect="call"; on="node1"; to="$c"; data="0x"; compare="Equal"; is="0x000000000000000000000000000000000000000000000000000000000000002a"

### wemix-wbft-handoff — tests/tc/go-wemix/handoff/01-wemix-wbft-handoff.json

- `P2` / `conditional` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-wemix/handoff/01-wemix-wbft-handoff.json:31`
- 적용: GWEMIX_BIN/GWBFT_BIN/GOWEMIX_TEMPLATE과 전환 높이·해당 브랜치 호환성을 확인한다. PR 단독 회귀와 분리한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=22; timeout="240s"
2. expect="blockNumber"; compare="Greater"; is="20"
3. expect="rpc"; method="eth_getBlockByNumber"; params=["0x15",false]; select="miner"; compare="NotEqual"; is="0xf9593d358b373d354a348c00887b914b408f6984"
4. expect="rpc"; method="istanbul_getValidators"; params=["latest"]; compare="Len"; is=4

### remote-rpc-health — tests/tc/remote/01-remote-rpc-health.json

- `P2` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/remote/01-remote-rpc-health.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. top-level applicableChains에 wemix를 포함한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. compare="GreaterOrEqual"; expect="blockNumber"; is="0x0"
2. compare="NotEqual"; expect="rpcCall"; is=""; method="web3_clientVersion"
3. compare="NotEqual"; expect="rpcCall"; is=""; method="net_peerCount"

### remote-chain-info — tests/tc/remote/02-remote-chain-info.json

- `P2` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/remote/02-remote-chain-info.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. top-level applicableChains에 wemix를 포함한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. compare="Greater"; expect="chainId"; is="0"
2. expect="rpcCall"; is=false; method="eth_syncing"
3. compare="NotEqual"; expect="rpcCall"; is=""; method="eth_getBlockByNumber"; params=["latest",false]; select="hash"

### remote-balance-check — tests/tc/remote/03-remote-balance-check.json

- `P2` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/remote/03-remote-balance-check.json:31`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. top-level applicableChains에 wemix를 포함한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. address="0x0000000000000000000000000000000000000000"; compare="GreaterOrEqual"; expect="balanceAt"; is="0"
2. compare="NotEqual"; expect="rpcCall"; is=""; method="eth_getBalance"; params=["0x0000000000000000000000000000000000000000","latest"]

### sample-minimal-value-transfer — tests/tc/samples/01-sample-minimal.json

- `P2` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/samples/01-sample-minimal.json:44`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. sample의 bohoBlock overlay 제거 및 실제 목적에 맞는 기대조건 보강. top-level applicableChains에 wemix를 포함한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. address="node2"; do="read"; save="balBefore"; source="balanceAt"
2. do="sendTx"; expect="receipt"; from="node1"; gas="$transferGas"; on="node1"; save="txHash"; to="node2"; value="1000000000000000000"
3. do="read"; method="eth_getTransactionReceipt"; params=["$txHash"]; save="txBlock"; select="blockNumber"; source="rpcCall"
4. expect="txStatus"; hash="$txHash"; is="0x1"
5. compare="Greater"; expect="derive"; is="0"; of=["$txBlock","0"]; op="diff"
6. address="node2"; compare="Greater"; expect="balanceAt"; is="$balBefore"

### sample-lifecycle-node-restart — tests/tc/samples/02-sample-lifecycle.json

- `P2` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/samples/02-sample-lifecycle.json:42`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. sample의 bohoBlock overlay 제거 및 실제 목적에 맞는 기대조건 보강.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=3; timeout="120s"
2. expect="sameBlockHash"; block="latest"
3. do="newAccount"; save="acct"; saveKey="acctKey"
4. do="sendTx"; on="node1"; from="dev1"; to="$acct"; value="1000000000000000000"; gas="21000"; expect="receipt"
5. expect="balanceAt"; on="node1"; address="$acct"; compare="Greater"; is="0"
6. do="stopNode"; on="en1"
7. expect="blockAdvance"; on="node1"; timeout="60s"; pollInterval="2s"
8. do="startNode"; on="en1"
9. do="waitBlock"; on="en1"; target=8; timeout="120s"
10. expect="sameBlockHash"; block="latest"
11. do="readNodeLog"; on="en1"; maxBytes=4096; save="en1Log"

### stablenet-derived-vocabulary — tests/tc/go-stablenet/vocabulary/01-derived-address-and-checksum.json

- `P3` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/vocabulary/01-derived-address-and-checksum.json:27`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=1; timeout="120s"
2. expect="createAddress"; deployer="0x1111111111111111111111111111111111111111"; nonce=0; is="0x8f7a45ebde059392e46a46dcc14ab24681a961ea"
3. expect="contractChecksum"; bytecode="0x6001"; is="sha256:9e67b12fd8c58953460459cad7a6d4dd7d6d57594affce8206d1397c9c4db543"

### stablenet-faucet-funds — tests/tc/go-stablenet/vocabulary/02-faucet-funds-account.json

- `P3` / `conditional` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/vocabulary/02-faucet-funds-account.json:27`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다. go-wemix faucet backend/자금원 준비.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=1; timeout="120s"
2. do="newAccount"; save="acct"; saveKey="acctKey"
3. do="faucet"; to="$acct"; amount="1000000000000000000"; timeout="60s"
4. expect="balanceAt"; address="$acct"; compare="GreaterOrEqual"; is="1000000000000000000"

### stablenet-register-contract — tests/tc/go-stablenet/vocabulary/03-register-contract.json

- `P3` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/vocabulary/03-register-contract.json:13`
- 적용: env.chain=wemix, gwemix 바이너리·PoA/etcd genesis·노드 역할·자금/서명 계정·수수료·timeout을 맞춘다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=2; timeout="120s"
2. do="deployContract"; on="node1"; from="0xc17d493883eaa3b4cceb0f214b273392d562f9d8"; bytecode="0x600a600c600039600a6000f3602a60005260206000f3"; gas="300000"; save="c"
3. do="registerContract"; on="node1"; to="$c"; data="0x01"; gas="100000"; save="reg"
4. expect="txStatus"; on="node1"; hash="$reg"; is="0x1"

### basic-wbft-consensus — tests/tc/basic/07-basic-wbft-consensus.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/basic/07-basic-wbft-consensus.json:29`
- 적용: PoA/etcd 합의 검증으로 새로 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=5; timeout="120s"
2. expect="rpc"; method="istanbul_getValidators"; params=["latest"]; select="#"; compare="GreaterOrEqual"; is="4"
3. expect="rpc"; method="istanbul_getWbftExtraInfo"; params=["@latest"]; select="prevCommittedSeal.sealers.#"; compare="GreaterOrEqual"; is="3"

### stablenet-delayed-fork — tests/tc/go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/01-stablenet-delayed-fork.json:41`
- 적용: 해당 포크/계약은 PR-head go-wemix 대상이 아님.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=6; timeout="120s"
2. do="read"; source="rpcCall"; method="eth_getCode"; params=["0x0000000000000000000000000000000000001003","0x1"]; save="govMinterPreBoho"
3. expect="rpc"; method="eth_getCode"; params=["0x0000000000000000000000000000000000001003","0x1"]; compare="NotEqual"; is="0x"
4. expect="rpc"; method="eth_getCode"; params=["0x0000000000000000000000000000000000001003","latest"]; compare="NotEqual"; is="$govMinterPreBoho"
5. expect="rpc"; method="eth_call"; params=[{"to":"0x0000000000000000000000000000000000000100","data":"0x4cee90eb86eaa050036147a12d49004b6b9c72bd725d39d4785011fe190f0b4da73bd4903f0ce3b639bbbf6e8e80d16931ff4bcf5993d58468e8fb19086e8cac36dbcd03009df8c59286b162af3bd7fcc0450c9aa81be5d10d312af6c66b1d604aebd3099c618202fcfe16ae7770b0c49ab5eadf74b754204a3bb6060e44eff37618b065f9832de4ca6ca971a7a1adc826d0f7c00181a5fb2ddf79ae00b4e10e"},"0x1"]; compare="Equal"; is="0x"
6. expect="rpc"; method="eth_call"; params=[{"to":"0x0000000000000000000000000000000000000100","data":"0x4cee90eb86eaa050036147a12d49004b6b9c72bd725d39d4785011fe190f0b4da73bd4903f0ce3b639bbbf6e8e80d16931ff4bcf5993d58468e8fb19086e8cac36dbcd03009df8c59286b162af3bd7fcc0450c9aa81be5d10d312af6c66b1d604aebd3099c618202fcfe16ae7770b0c49ab5eadf74b754204a3bb6060e44eff37618b065f9832de4ca6ca971a7a1adc826d0f7c00181a5fb2ddf79ae00b4e10e"},"latest"]; compare="Equal"; is="0x0000000000000000000000000000000000000000000000000000000000000001"
7. do="read"; source="rpcCall"; method="eth_getBalance"; params=["0xc17d493883eaa3b4cceb0f214b273392d562f9d8","0x1"]; save="preallocPreBoho"
8. expect="rpc"; method="eth_getBalance"; params=["0xc17d493883eaa3b4cceb0f214b273392d562f9d8","latest"]; compare="Equal"; is="$preallocPreBoho"

### govminter-v2-code — tests/tc/go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/02-govminter-v2-code.json:31`
- 적용: 해당 포크/계약은 PR-head go-wemix 대상이 아님.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="read"; method="eth_getBalance"; params=["0x0000000000000000000000000000000000001003","0x1"]; save="balPreBoho"; source="rpcCall"
2. do="read"; method="eth_getCode"; params=["0x0000000000000000000000000000000000001003","0x1"]; save="codePreBoho"; source="rpcCall"
3. compare="GreaterOrEqual"; do="waitFor"; expected="11"; pollInterval="2s"; source="blockNumber"; timeout="120s"
4. address="0x0000000000000000000000000000000000001003"; compare="NotEqual"; expect="codeAt"; is="0x"
5. compare="NotEqual"; expect="rpcCall"; is="$codePreBoho"; method="eth_getCode"; params=["0x0000000000000000000000000000000000001003","latest"]
6. compare="Equal"; expect="rpcCall"; is="$balPreBoho"; method="eth_getBalance"; params=["0x0000000000000000000000000000000000001003","latest"]

### burn-cancel-refundable — tests/tc/go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/03-burn-cancel-refundable.json:31`
- 적용: 해당 포크/계약은 PR-head go-wemix 대상이 아님.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="read"; of=["0x2493a84a8f83cb87fdcbe0bb3b2d313f69a58d3c"]; op="abiCall"; save="refBalCall"; selector="0xb03d36cd"; source="derive"
2. data="$refBalCall"; do="read"; save="refBefore"; source="call"; to="0x0000000000000000000000000000000000001003"
3. data=hex payload (452 bytes; 원문 참조); do="sendTx"; from="node2"; gas="0xb71b0"; on="node2"; save="burnHash"; to="0x0000000000000000000000000000000000001003"; value="0xde0b6b3a7640000"
4. do="read"; hash="$burnHash"; save="cancelPid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
5. do="read"; of=["$cancelPid"]; op="abiCall"; save="cancelData"; selector="0xe0a8f6f5"; source="derive"
6. data="$cancelData"; do="sendTx"; from="node2"; gas="0xb71b0"; on="node2"; save="cancelHash"; to="0x0000000000000000000000000000000000001003"
7. data="$refBalCall"; do="read"; save="refAfter"; source="call"; to="0x0000000000000000000000000000000000001003"
8. expect="txStatus"; hash="$burnHash"; is="0x1"
9. expect="txStatus"; hash="$cancelHash"; is="0x1"
10. compare="Equal"; expect="derive"; is="1000000000000000000"; of=["$refAfter","$refBefore"]; op="diff"

### burn-reject-refundable — tests/tc/go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/04-burn-reject-refundable.json:31`
- 적용: 해당 포크/계약은 PR-head go-wemix 대상이 아님.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="read"; of=["0x8c4a10b9108d49b9d23f764464090831d9c17764"]; op="abiCall"; save="refBalCall"; selector="0xb03d36cd"; source="derive"
2. data="$refBalCall"; do="read"; save="refBefore"; source="call"; to="0x0000000000000000000000000000000000001003"
3. data=hex payload (452 bytes; 원문 참조); do="sendTx"; from="node3"; gas="0xb71b0"; on="node3"; save="burnHash"; to="0x0000000000000000000000000000000000001003"; value="0xde0b6b3a7640000"
4. do="read"; hash="$burnHash"; save="burnPid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
5. do="read"; of=["$burnPid"]; op="abiCall"; save="disData"; selector="0xc8541fe0"; source="derive"
6. data="$disData"; do="sendTx"; from="node1"; gas="0xb71b0"; on="node1"; save="dis1Hash"; to="0x0000000000000000000000000000000000001003"
7. data="$disData"; do="sendTx"; from="node2"; gas="0xb71b0"; on="node2"; save="dis2Hash"; to="0x0000000000000000000000000000000000001003"
8. data="$disData"; do="sendTx"; from="node4"; gas="0xb71b0"; on="node4"; save="dis3Hash"; to="0x0000000000000000000000000000000000001003"
9. do="read"; of=["$burnPid"]; op="abiCall"; save="proposalsCall"; selector="0x013cf08b"; source="derive"
10. data="$proposalsCall"; do="read"; save="proposalsRet"; source="call"; to="0x0000000000000000000000000000000000001003"
11. data="$refBalCall"; do="read"; save="refAfter"; source="call"; to="0x0000000000000000000000000000000000001003"
12. expect="txStatus"; hash="$burnHash"; is="0x1"
13. expect="txStatus"; hash="$dis1Hash"; is="0x1"
14. expect="txStatus"; hash="$dis2Hash"; is="0x1"
15. expect="txStatus"; hash="$dis3Hash"; is="0x1"
16. compare="Equal"; expect="derive"; index=9; is="0x0000000000000000000000000000000000000000000000000000000000000007"; of=["$proposalsRet"]; op="word"
17. compare="Equal"; expect="derive"; is="1000000000000000000"; of=["$refAfter","$refBefore"]; op="diff"

### burn-expire-refundable — tests/tc/go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/05-burn-expire-refundable.json:33`
- 적용: 해당 포크/계약은 PR-head go-wemix 대상이 아님.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="read"; of=["0x8c4a10b9108d49b9d23f764464090831d9c17764"]; op="abiCall"; save="refBalCall"; selector="0xb03d36cd"; source="derive"
2. data="$refBalCall"; do="read"; save="refBefore"; source="call"; to="0x0000000000000000000000000000000000001003"
3. address="0x0000000000000000000000000000000000001003"; do="read"; save="govBefore"; source="balanceAt"
4. data=hex payload (451 bytes; 원문 참조); do="sendTx"; from="node3"; gas="0xb71b0"; on="node3"; save="burnHash"; to="0x0000000000000000000000000000000000001003"; value="0xde0b6b3a7640000"
5. address="0x0000000000000000000000000000000000001003"; do="read"; save="govAfterBurn"; source="balanceAt"
6. do="read"; hash="$burnHash"; save="burnPid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
7. do="read"; method="eth_getBlockByNumber"; params=["@latest",false]; save="t0"; select="timestamp"; source="rpcCall"
8. do="read"; of=["$t0","40"]; op="sum"; save="expiryTarget"; source="derive"
9. compare="GreaterOrEqual"; do="waitFor"; expected="$expiryTarget"; method="eth_getBlockByNumber"; params=["@latest",false]; pollInterval="2s"; select="timestamp"; source="rpcCall"; timeout="90s"
10. do="read"; of=["$burnPid"]; op="abiCall"; save="expData"; selector="0xe1b526b0"; source="derive"
11. data="$expData"; do="sendTx"; from="node1"; gas="0xb71b0"; on="node1"; save="expHash"; to="0x0000000000000000000000000000000000001003"
12. do="read"; method="eth_getTransactionReceipt"; params=["$expHash"]; save="expBlk"; select="blockNumber"; source="rpcCall"
13. do="read"; of=["$burnPid"]; op="abiCall"; save="proposalsCall"; selector="0x013cf08b"; source="derive"
14. data="$proposalsCall"; do="read"; save="proposalsRet"; source="call"; to="0x0000000000000000000000000000000000001003"
15. data="$refBalCall"; do="read"; save="refAfter"; source="call"; to="0x0000000000000000000000000000000000001003"
16. expect="txStatus"; hash="$burnHash"; is="0x1"
17. expect="txStatus"; hash="$expHash"; is="0x1"
18. compare="Equal"; expect="derive"; is="1000000000000000000"; of=["$govAfterBurn","$govBefore"]; op="diff"
19. compare="Equal"; expect="derive"; index=9; is="0x0000000000000000000000000000000000000000000000000000000000000005"; of=["$proposalsRet"]; op="word"
20. compare="Equal"; expect="derive"; is="1000000000000000000"; of=["$refAfter","$refBefore"]; op="diff"
21. address="0x0000000000000000000000000000000000001003"; compare="GreaterOrEqual"; expect="logs"; fromBlock="$expBlk"; is="1"; select="count"; toBlock="$expBlk"; topics=["0x116044c8ec23de4088f4a50cf4f8a504ec887c439452af3be4f57f6c77bf2062"]

### burn-execute-no-refundable — tests/tc/go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/06-burn-execute-no-refundable.json:31`
- 적용: 해당 포크/계약은 PR-head go-wemix 대상이 아님.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="read"; of=["0xc17d493883eaa3b4cceb0f214b273392d562f9d8"]; op="abiCall"; save="refBalCall"; selector="0xb03d36cd"; source="derive"
2. data="$refBalCall"; do="read"; save="refBefore"; source="call"; to="0x0000000000000000000000000000000000001003"
3. data=hex payload (452 bytes; 원문 참조); do="sendTx"; from="node1"; gas="0xb71b0"; on="node1"; save="burnHash"; to="0x0000000000000000000000000000000000001003"; value="0xde0b6b3a7640000"
4. do="read"; hash="$burnHash"; save="burnPid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
5. do="read"; of=["$burnPid"]; op="abiCall"; save="approveData"; selector="0x98951b56"; source="derive"
6. data="$approveData"; do="sendTx"; from="node2"; gas="0xb71b0"; on="node2"; save="approveHash"; to="0x0000000000000000000000000000000000001003"
7. do="read"; of=["$burnPid"]; op="abiCall"; save="proposalsCall"; selector="0x013cf08b"; source="derive"
8. data="$proposalsCall"; do="read"; save="proposalsRet"; source="call"; to="0x0000000000000000000000000000000000001003"
9. data="$refBalCall"; do="read"; save="refAfter"; source="call"; to="0x0000000000000000000000000000000000001003"
10. expect="txStatus"; hash="$burnHash"; is="0x1"
11. expect="txStatus"; hash="$approveHash"; is="0x1"
12. compare="Equal"; expect="derive"; index=9; is="0x0000000000000000000000000000000000000000000000000000000000000003"; of=["$proposalsRet"]; op="word"
13. compare="Equal"; expect="derive"; is="0"; of=["$refAfter","$refBefore"]; op="diff"

### claim-burn-refund-succeeds — tests/tc/go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/07-claim-burn-refund-succeeds.json:31`
- 적용: 해당 포크/계약은 PR-head go-wemix 대상이 아님.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="read"; of=["0x8eb79036bc0f3aba136ef18b3a2fb8c1188939a6"]; op="abiCall"; save="refBalCall"; selector="0xb03d36cd"; source="derive"
2. data="$refBalCall"; do="read"; save="refBefore"; source="call"; to="0x0000000000000000000000000000000000001003"
3. data=hex payload (452 bytes; 원문 참조); do="sendTx"; from="node4"; gas="0xb71b0"; on="node4"; save="burnHash"; to="0x0000000000000000000000000000000000001003"; value="0xde0b6b3a7640000"
4. do="read"; hash="$burnHash"; save="burnPid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
5. do="read"; of=["$burnPid"]; op="abiCall"; save="cancelData"; selector="0xe0a8f6f5"; source="derive"
6. data="$cancelData"; do="sendTx"; from="node4"; gas="0xb71b0"; on="node4"; save="cancelHash"; to="0x0000000000000000000000000000000000001003"
7. data="$refBalCall"; do="read"; save="refMid"; source="call"; to="0x0000000000000000000000000000000000001003"
8. data="0x936834b9"; do="sendTx"; from="node4"; gas="0xb71b0"; on="node4"; save="claimHash"; to="0x0000000000000000000000000000000000001003"
9. data="$refBalCall"; do="read"; save="refAfter"; source="call"; to="0x0000000000000000000000000000000000001003"
10. expect="txStatus"; hash="$burnHash"; is="0x1"
11. expect="txStatus"; hash="$cancelHash"; is="0x1"
12. expect="txStatus"; hash="$claimHash"; is="0x1"
13. compare="Equal"; expect="derive"; is="1000000000000000000"; of=["$refMid","$refBefore"]; op="diff"
14. compare="Equal"; expect="derive"; index=0; is="0x0000000000000000000000000000000000000000000000000000000000000000"; of=["$refAfter"]; op="word"

### claim-zero-refund-reverts — tests/tc/go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/08-claim-zero-refund-reverts.json:31`
- 적용: 해당 포크/계약은 PR-head go-wemix 대상이 아님.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="newAccount"; save="acct"; saveKey="acctKey"
2. do="sendTx"; from="node1"; gas="21000"; on="node1"; save="fundHash"; to="$acct"; value="100000000000000000000"
3. data="0x936834b9"; do="sendTx"; expect="revert"; gas="0xb71b0"; key="$acctKey"; on="en1"; save="claimHash"; to="0x0000000000000000000000000000000000001003"
4. expect="txStatus"; hash="$fundHash"; is="0x1"

### claim-burn-refund-double-reverts — tests/tc/go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/09-claim-burn-refund-double-reverts.json:31`
- 적용: 해당 포크/계약은 PR-head go-wemix 대상이 아님.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. data=hex payload (452 bytes; 원문 참조); do="sendTx"; from="node2"; gas="0xb71b0"; on="node2"; save="burnHash"; to="0x0000000000000000000000000000000000001003"; value="0xde0b6b3a7640000"
2. do="read"; hash="$burnHash"; save="burnPid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
3. do="read"; of=["$burnPid"]; op="abiCall"; save="cancelData"; selector="0xe0a8f6f5"; source="derive"
4. data="$cancelData"; do="sendTx"; from="node2"; gas="0xb71b0"; on="node2"; save="cancelHash"; to="0x0000000000000000000000000000000000001003"
5. data="0x936834b9"; do="sendTx"; from="node2"; gas="0xb71b0"; on="node2"; save="claim1Hash"; to="0x0000000000000000000000000000000000001003"
6. data="0x936834b9"; do="sendTx"; expect="revert"; from="node2"; gas="0xb71b0"; on="node2"; save="claim2Hash"; to="0x0000000000000000000000000000000000001003"
7. expect="txStatus"; hash="$burnHash"; is="0x1"
8. expect="txStatus"; hash="$cancelHash"; is="0x1"
9. expect="txStatus"; hash="$claim1Hash"; is="0x1"

### prealloc-preserved-across-boho — tests/tc/go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/10-prealloc-preserved-across-boho.json:34`
- 적용: 해당 포크/계약은 PR-head go-wemix 대상이 아님.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="NotEqual"; do="waitFor"; expected="0x0"; method="eth_getBalance"; params=["0x71562b71999873db5b286df957af199ec94617f7","0x1"]; pollInterval="1s"; source="rpcCall"; timeout="60s"
2. do="read"; method="eth_getBalance"; params=["0x71562b71999873db5b286df957af199ec94617f7","0x1"]; save="balEarly"; source="rpcCall"
3. do="read"; method="eth_getStorageAt"; params=["0x0000000000000000000000000000000000001003","0x0000000000000000000000000000000000000000000000000000000000000004","0x1"]; save="storagePreBoho"; source="rpcCall"
4. compare="GreaterOrEqual"; do="waitFor"; expected="11"; pollInterval="2s"; source="blockNumber"; timeout="120s"
5. compare="Equal"; expect="rpcCall"; is="$balEarly"; method="eth_getBalance"; params=["0x71562b71999873db5b286df957af199ec94617f7","latest"]
6. compare="Equal"; expect="rpcCall"; is="0x0"; method="eth_getTransactionCount"; params=["0x71562b71999873db5b286df957af199ec94617f7","latest"]
7. compare="Equal"; expect="rpcCall"; is="$storagePreBoho"; method="eth_getStorageAt"; params=["0x0000000000000000000000000000000000001003","0x0000000000000000000000000000000000000000000000000000000000000004","latest"]

### boho-chain-config-active — tests/tc/go-stablenet/post-v1.0.0-change/common-all/15-boho-chain-config-active.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/15-boho-chain-config-active.json:31`
- 적용: 해당 포크/계약은 PR-head go-wemix 대상이 아님.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="Greater"; expect="blockNumber"; is="0"
2. compare="Greater"; expect="chainId"; is="0"
3. compare="Greater"; expect="baseFee"; is="0"

### anzeon-active-before-boho — tests/tc/go-stablenet/post-v1.0.0-change/common-all/16-anzeon-active-before-boho.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/16-anzeon-active-before-boho.json:34`
- 적용: 해당 포크/계약은 PR-head go-wemix 대상이 아님.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="GreaterOrEqual"; do="waitFor"; expected="1"; pollInterval="1s"; source="blockNumber"; timeout="60s"
2. compare="Regexp"; expect="rpcCall"; is="^0x[0-9a-fA-F]{100,}$"; method="eth_getCode"; params=["0x0000000000000000000000000000000000001001","0x1"]

### estimategas-authorizationlist-cost — tests/tc/go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/17-estimategas-authorizationlist-cost.json:116`
- 적용: 해당 포크/계약은 PR-head go-wemix 대상이 아님.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="newAccount"; save="delegate"; saveKey="delegateKey"
2. do="newAccount"; save="authority1"; saveKey="authority1Key"
3. do="newAccount"; save="authority2"; saveKey="authority2Key"
4. do="read"; method="eth_coinbase"; params=[]; save="sponsor"; source="rpcCall"
5. authorityKey="$authority1Key"; delegate="$delegate"; do="signAuthorization"; save="auth1"
6. authorityKey="$authority2Key"; delegate="$delegate"; do="signAuthorization"; save="auth2"
7. do="read"; method="eth_estimateGas"; params=[{"from":"$sponsor","to":"$sponsor"}]; save="gasBaseline"; source="rpcCall"
8. do="read"; method="eth_estimateGas"; params=[{"authorizationList":["$auth1"],"from":"$sponsor","to":"$sponsor"}]; save="gas1Auth"; source="rpcCall"
9. do="read"; method="eth_estimateGas"; params=[{"authorizationList":["$auth1","$auth2"],"from":"$sponsor","to":"$sponsor"}]; save="gas2Auth"; source="rpcCall"
10. compare="Equal"; expect="derive"; is="$gasBaselineExpected"; of=["$gasBaseline"]; op="sum"
11. compare="GreaterOrEqual"; expect="derive"; is="$gas1AuthMin"; of=["$gas1Auth"]; op="sum"
12. compare="LessOrEqual"; expect="derive"; is="$gas1AuthMax"; of=["$gas1Auth"]; op="sum"
13. compare="GreaterOrEqual"; expect="derive"; is="$gas2AuthMin"; of=["$gas2Auth"]; op="sum"
14. compare="LessOrEqual"; expect="derive"; is="$gas2AuthMax"; of=["$gas2Auth"]; op="sum"
15. compare="GreaterOrEqual"; expect="derive"; is="$authCost1Min"; of=["$gas1Auth","$gasBaseline"]; op="diff"
16. compare="LessOrEqual"; expect="derive"; is="$authCost1Max"; of=["$gas1Auth","$gasBaseline"]; op="diff"
17. compare="GreaterOrEqual"; expect="derive"; is="$authCost2Min"; of=["$gas2Auth","$gasBaseline"]; op="diff"
18. compare="LessOrEqual"; expect="derive"; is="$authCost2Max"; of=["$gas2Auth","$gasBaseline"]; op="diff"

### upgrade-registry-order — tests/tc/go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/19-upgrade-registry-order.json:31`
- 적용: 해당 포크/계약은 PR-head go-wemix 대상이 아님.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="read"; method="eth_getCode"; params=["0x0000000000000000000000000000000000001003","0x0"]; save="codeGenesis"; source="rpcCall"
2. compare="NotEqual"; expect="rpcCall"; is="0x"; method="eth_getCode"; params=["0x0000000000000000000000000000000000001003","0x0"]
3. compare="Equal"; expect="rpcCall"; is="$codeGenesis"; method="eth_getCode"; params=["0x0000000000000000000000000000000000001003","0x63"]
4. compare="NotEqual"; expect="rpcCall"; is="$codeGenesis"; method="eth_getCode"; params=["0x0000000000000000000000000000000000001003","0x64"]

### v1-params-init-storage — tests/tc/go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/20-v1-params-init-storage.json:31`
- 적용: 해당 포크/계약은 PR-head go-wemix 대상이 아님.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="NotEqual"; expect="rpcCall"; is="0x0000000000000000000000000000000000000000000000000000000000000000"; method="eth_getStorageAt"; params=["0x0000000000000000000000000000000000001001","0x0000000000000000000000000000000000000000000000000000000000000039","0x0"]
2. compare="NotEqual"; expect="rpcCall"; is="0x0000000000000000000000000000000000000000000000000000000000000000"; method="eth_getStorageAt"; params=["0x0000000000000000000000000000000000001003","0x0000000000000000000000000000000000000000000000000000000000000004","0x63"]
3. compare="NotEqual"; expect="rpcCall"; is="0x"; method="eth_getCode"; params=["0x0000000000000000000000000000000000001003","0x0"]

### burn-refund-events — tests/tc/go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/common-all/21-burn-refund-events.json:30`
- 적용: 해당 포크/계약은 PR-head go-wemix 대상이 아님.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. data=hex payload (452 bytes; 원문 참조); do="sendTx"; from="node4"; gas="0xb71b0"; on="node4"; save="burnHash"; to="0x0000000000000000000000000000000000001003"; value="0xde0b6b3a7640000"
2. do="read"; hash="$burnHash"; save="burnPid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
3. do="read"; of=["$burnPid"]; op="abiCall"; save="cancelData"; selector="0xe0a8f6f5"; source="derive"
4. data="$cancelData"; do="sendTx"; from="node4"; gas="0xb71b0"; on="node4"; save="cancelHash"; to="0x0000000000000000000000000000000000001003"
5. do="read"; method="eth_getTransactionReceipt"; params=["$cancelHash"]; save="cancelBlk"; select="blockNumber"; source="rpcCall"
6. data="0x936834b9"; do="sendTx"; from="node4"; gas="0xb71b0"; on="node4"; save="claimHash"; to="0x0000000000000000000000000000000000001003"
7. do="read"; method="eth_getTransactionReceipt"; params=["$claimHash"]; save="claimBlk"; select="blockNumber"; source="rpcCall"
8. expect="txStatus"; hash="$burnHash"; is="0x1"
9. expect="txStatus"; hash="$cancelHash"; is="0x1"
10. expect="txStatus"; hash="$claimHash"; is="0x1"
11. address="0x0000000000000000000000000000000000001003"; compare="GreaterOrEqual"; expect="logs"; fromBlock="$cancelBlk"; is="1"; select="count"; toBlock="$cancelBlk"; topics=["0x116044c8ec23de4088f4a50cf4f8a504ec887c439452af3be4f57f6c77bf2062"]
12. address="0x0000000000000000000000000000000000001003"; compare="GreaterOrEqual"; expect="logs"; fromBlock="$claimBlk"; is="1"; select="count"; toBlock="$claimBlk"; topics=["0x9543fa265d2616af3e7021d8b5a7d1271eb7bba960908675ce3bddaf60c1af24"]

### effective-gas-price-authorized-bp-en — tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/01-effective-gas-price-authorized-bp-en.json:29`
- 적용: 동일 이름만으로 go-wemix 수수료 의미와 같다고 볼 수 없다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=2; timeout="120s"
2. do="newAccount"; save="acct"; saveKey="acctKey"
3. do="sendTx"; on="node1"; from="node1"; to="$acct"; value="10000000000000000000"; gas="21000"; expect="receipt"
4. do="read"; source="derive"; op="abiCall"; selector="0x93a8bb99"; of=["$acct"]; save="authData"
5. do="sendTx"; on="node1"; from="node1"; to="0x0000000000000000000000000000000000001004"; gas="0x16e360"; data="$authData"; save="authHash"; expect="receipt"
6. do="read"; source="receiptLog"; hash="$authHash"; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"; topic=1; save="authPid"
7. do="read"; source="derive"; op="abiCall"; selector="0x98951b56"; of=["$authPid"]; save="authApprove"
8. do="sendTx"; on="node2"; from="node2"; to="0x0000000000000000000000000000000000001004"; gas="0x16e360"; data="$authApprove"; save="authApHash"; expect="receipt"
9. do="sendTx"; on="node1"; key="$acctKey"; to="0x70997970C51812dc3A010C7d01b50e0d17dc79C8"; value="1"; save="txHash"; expect="receipt"
10. do="read"; on="node1"; source="rpcCall"; method="eth_getTransactionReceipt"; params=["$txHash"]; select="blockNumber"; save="txBlock"
11. do="waitFor"; on="en1"; source="rpcCall"; method="eth_blockNumber"; params=[]; compare="GreaterOrEqual"; expected="$txBlock"; timeout="120s"; pollInterval="2s"
12. do="read"; on="node1"; source="rpcCall"; method="eth_getTransactionReceipt"; params=["$txHash"]; select="effectiveGasPrice"; save="egpBP"
13. do="read"; on="en1"; source="rpcCall"; method="eth_getTransactionReceipt"; params=["$txHash"]; select="effectiveGasPrice"; save="egpEN"
14. expect="rpc"; on="en1"; method="eth_getTransactionReceipt"; params=["$txHash"]; select="effectiveGasPrice"; compare="Equal"; is="$egpBP"

### effective-gas-price-regular — tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/02-effective-gas-price-regular.json:31`
- 적용: env.chain=wemix와 gwemix 바이너리, PoA genesis/계정/수수료를 준비하고 applicableChains에 wemix를 포함한다. 영수증 effectiveGasPrice를 해당 tx의 type 및 실제 gasPrice/feeCap/tipCap, 포함 블록 baseFee로 계산한다. TC-036과 중복으로 연결하고 BP/EN 비교는 실제 구현이 없음을 명시한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="sendTx"; from="node1"; gas="21000"; save="hash"; to="0x00000000000000000000000000000000C0FFEE11"; value="1"
2. do="read"; method="eth_getTransactionReceipt"; params=["$hash"]; save="egp"; select="effectiveGasPrice"; source="rpcCall"
3. do="read"; method="eth_getTransactionReceipt"; params=["$hash"]; save="blk"; select="blockNumber"; source="rpcCall"
4. do="read"; method="eth_getBlockByNumber"; params=["$blk",false]; save="base"; select="baseFeePerGas"; source="rpcCall"
5. do="read"; method="istanbul_getWbftExtraInfo"; params=["$blk"]; save="tip"; select="gasTip"; source="rpcCall"
6. compare="Equal"; expect="derive"; is="$egp"; of=["$base","$tip"]; op="sum"

### auth-tx-event-last-bp-en — tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/effectivegasprice/03-auth-tx-event-last-bp-en.json:29`
- 적용: 동일 이름만으로 go-wemix 수수료 의미와 같다고 볼 수 없다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=2; timeout="120s"
2. do="newAccount"; save="acct"; saveKey="acctKey"
3. do="sendTx"; on="node1"; from="node1"; to="$acct"; value="10000000000000000000"; gas="21000"; expect="receipt"
4. do="read"; source="derive"; op="abiCall"; selector="0x93a8bb99"; of=["$acct"]; save="authData"
5. do="sendTx"; on="node1"; from="node1"; to="0x0000000000000000000000000000000000001004"; gas="0x16e360"; data="$authData"; save="authHash"; expect="receipt"
6. do="read"; source="receiptLog"; hash="$authHash"; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"; topic=1; save="authPid"
7. do="read"; source="derive"; op="abiCall"; selector="0x98951b56"; of=["$authPid"]; save="authApprove"
8. do="sendTx"; on="node2"; from="node2"; to="0x0000000000000000000000000000000000001004"; gas="0x16e360"; data="$authApprove"; save="authApHash"; expect="receipt"
9. do="sendTx"; on="node1"; key="$acctKey"; to="0x70997970C51812dc3A010C7d01b50e0d17dc79C8"; value="1"; save="txHash"; expect="receipt"
10. do="read"; on="node1"; source="rpcCall"; method="eth_getTransactionReceipt"; params=["$txHash"]; select="blockNumber"; save="txBlock"
11. do="waitFor"; on="en1"; source="rpcCall"; method="eth_blockNumber"; params=[]; compare="GreaterOrEqual"; expected="$txBlock"; timeout="120s"; pollInterval="2s"
12. do="read"; on="node1"; source="rpcCall"; method="eth_getTransactionReceipt"; params=["$txHash"]; select="logs.#"; save="lastTopicBPCount"
13. do="read"; source="derive"; op="diff"; of=["$lastTopicBPCount","1"]; save="lastTopicBPLastIdx"
14. do="read"; on="node1"; source="rpcCall"; method="eth_getTransactionReceipt"; params=["$txHash"]; select="logs.${lastTopicBPLastIdx}.topics.0"; save="lastTopicBP"
15. do="read"; on="en1"; source="rpcCall"; method="eth_getTransactionReceipt"; params=["$txHash"]; select="logs.#"; save="lastTopicENCount"
16. do="read"; source="derive"; op="diff"; of=["$lastTopicENCount","1"]; save="lastTopicENLastIdx"
17. do="read"; on="en1"; source="rpcCall"; method="eth_getTransactionReceipt"; params=["$txHash"]; select="logs.${lastTopicENLastIdx}.topics.0"; save="lastTopicEN"
18. expect="rpc"; on="node1"; method="eth_getTransactionReceipt"; params=["$txHash"]; select="logs.${lastTopicBPLastIdx}.topics.0"; compare="Equal"; is="0x40e728a89c7f5b192cf1c1b747fb64d51d81c7a2b3ed4607b94d3a1e6a3e0373"
19. expect="rpc"; on="en1"; method="eth_getTransactionReceipt"; params=["$txHash"]; select="logs.${lastTopicENLastIdx}.topics.0"; compare="Equal"; is="$lastTopicBP"

### authorized-extra-bit-synced — tests/tc/go-stablenet/post-v1.0.0-change/extra-state/01-authorized-extra-bit-synced.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/01-authorized-extra-bit-synced.json:33`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="read"; of=["0x90F79bf6EB2c4f870365E785982E1f101E93b906"]; op="abiCall"; save="isAuthData"; selector="0xfe9fbb80"; source="derive"
2. compare="Equal"; data="$isAuthData"; expect="call"; is="0x0000000000000000000000000000000000000000000000000000000000000001"; to="0x0000000000000000000000000000000000B00003"

### blacklisted-extra-bit-synced — tests/tc/go-stablenet/post-v1.0.0-change/extra-state/01b-blacklisted-extra-bit-synced.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/01b-blacklisted-extra-bit-synced.json:32`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="read"; of=["0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"]; op="abiCall"; save="isBlData"; selector="0xfe575a87"; source="derive"
2. compare="Equal"; data="$isBlData"; expect="call"; is="0x0000000000000000000000000000000000000000000000000000000000000001"; to="0x0000000000000000000000000000000000B00003"

### stablenet-account-extra — tests/tc/go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/02-stablenet-account-extra.json:44`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=1; timeout="60s"
2. do="read"; source="derive"; op="abiCall"; selector="0xfe9fbb80"; of=["0x90F79bf6EB2c4f870365E785982E1f101E93b906"]; save="authCall"
3. expect="call"; on="node1"; to="0x0000000000000000000000000000000000B00003"; data="$authCall"; compare="Equal"; is="0x0000000000000000000000000000000000000000000000000000000000000001"
4. do="read"; source="derive"; op="abiCall"; selector="0xfe575a87"; of=["0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"]; save="blCall"
5. expect="call"; on="node1"; to="0x0000000000000000000000000000000000B00003"; data="$blCall"; compare="Equal"; is="0x0000000000000000000000000000000000000000000000000000000000000001"
6. do="read"; source="derive"; op="abiCall"; selector="0xfe9fbb80"; of=["0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc"]; save="dualAuth"
7. expect="call"; on="node1"; to="0x0000000000000000000000000000000000B00003"; data="$dualAuth"; compare="Equal"; is="0x0000000000000000000000000000000000000000000000000000000000000001"
8. do="read"; source="derive"; op="abiCall"; selector="0xfe575a87"; of=["0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc"]; save="dualBl"
9. expect="call"; on="node1"; to="0x0000000000000000000000000000000000B00003"; data="$dualBl"; compare="Equal"; is="0x0000000000000000000000000000000000000000000000000000000000000001"

### extra-union-merge — tests/tc/go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/03-extra-union-merge.json:56`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="read"; of=["0x90F79bf6EB2c4f870365E785982E1f101E93b906"]; op="abiCall"; save="allocOnly"; selector="0xfe9fbb80"; source="derive"
2. do="read"; of=["0x976EA74026E726554dB657fA54763abd0C3a0aa9"]; op="abiCall"; save="paramOnly"; selector="0xfe9fbb80"; source="derive"
3. do="read"; of=["0x14dC79964da2C08b23698B3D3cc7Ca32193d9955"]; op="abiCall"; save="bothSides"; selector="0xfe9fbb80"; source="derive"
4. compare="Equal"; data="$allocOnly"; expect="call"; is="0x0000000000000000000000000000000000000000000000000000000000000001"; to="0x0000000000000000000000000000000000B00003"
5. compare="Equal"; data="$paramOnly"; expect="call"; is="0x0000000000000000000000000000000000000000000000000000000000000001"; to="0x0000000000000000000000000000000000B00003"
6. compare="Equal"; data="$bothSides"; expect="call"; is="0x0000000000000000000000000000000000000000000000000000000000000001"; to="0x0000000000000000000000000000000000B00003"
7. compare="Equal"; data="0x6a6debd7"; expect="call"; is="0x0000000000000000000000000000000000000000000000000000000000000003"; to="0x0000000000000000000000000000000000001004"

### dual-status-extra — tests/tc/go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/04-dual-status-extra.json:33`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="read"; of=["0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc"]; op="abiCall"; save="isAuthData"; selector="0xfe9fbb80"; source="derive"
2. do="read"; of=["0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc"]; op="abiCall"; save="isBlData"; selector="0xfe575a87"; source="derive"
3. compare="Equal"; data="$isAuthData"; expect="call"; is="0x0000000000000000000000000000000000000000000000000000000000000001"; to="0x0000000000000000000000000000000000B00003"
4. compare="Equal"; data="$isBlData"; expect="call"; is="0x0000000000000000000000000000000000000000000000000000000000000001"; to="0x0000000000000000000000000000000000B00003"

### extra-balance-preserved — tests/tc/go-stablenet/post-v1.0.0-change/extra-state/05-extra-balance-preserved.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/05-extra-balance-preserved.json:33`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. address="0x90F79bf6EB2c4f870365E785982E1f101E93b906"; compare="Equal"; expect="balanceAt"; is="1000000000000000000"
2. address="0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"; compare="Equal"; expect="balanceAt"; is="1000000000000000000"

### invalid-extra-reject — tests/tc/go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/06-invalid-extra-reject.json:30`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=3; timeout="120s"
2. expect="rpc"; on="en1"; method="eth_blockNumber"; params=[]; compare="NotEqual"; is=null
3. do="swapNode"; on="en1"; purpose="invalid-extra"; genesisOverlay={"alloc":{"0x90F79bf6EB2c4f870365E785982E1f101E93b906":{"balance":"0xDE0B6B3A7640000","extra":"0x0000000000000001"}}}; expect="fail"; save="rejectEvidence"
4. do="readNodeLog"; on="en1"; save="victimLog"
5. expect="blockAdvance"; on="node1"; timeout="60s"; pollInterval="2s"

### extra-state-across-delayed-boho — tests/tc/go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/extra-state/07b-extra-state-across-delayed-boho.json:44`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=3; timeout="120s"
2. do="read"; source="rpcCall"; method="eth_getBalance"; params=["0x90F79bf6EB2c4f870365E785982E1f101E93b906","latest"]; save="authBalPre"
3. do="read"; source="rpcCall"; method="eth_getBalance"; params=["0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65","latest"]; save="blackBalPre"
4. do="waitBlock"; target=9; timeout="120s"
5. do="read"; source="derive"; op="abiCall"; selector="0xfe9fbb80"; of=["0x90F79bf6EB2c4f870365E785982E1f101E93b906"]; save="isAuthCall"
6. do="read"; source="derive"; op="abiCall"; selector="0xfe575a87"; of=["0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"]; save="isBlackCall"
7. expect="call"; to="0x0000000000000000000000000000000000B00003"; data="$isAuthCall"; compare="Equal"; is="0x0000000000000000000000000000000000000000000000000000000000000001"
8. expect="call"; to="0x0000000000000000000000000000000000B00003"; data="$isBlackCall"; compare="Equal"; is="0x0000000000000000000000000000000000000000000000000000000000000001"
9. expect="rpc"; method="eth_getBalance"; params=["0x90F79bf6EB2c4f870365E785982E1f101E93b906","latest"]; compare="Equal"; is="$authBalPre"
10. expect="rpc"; method="eth_getBalance"; params=["0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65","latest"]; compare="Equal"; is="$blackBalPre"

### unsupported-system-contract-version — tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/stand-alone/03-unsupported-version.json:44`
- 적용: go-wemix에 같은 업그레이드 registry 없음.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=5; timeout="120s"
2. do="read"; source="rpcCall"; method="eth_blockNumber"; params=[]; save="stalledAt"
3. do="readNodeLog"; on="node1"; save="node1Log"
4. expect="blockStalled"; on="node1"; timeout="20s"; pollInterval="2s"
5. expect="rpc"; method="eth_blockNumber"; params=[]; compare="Equal"; is="$stalledAt"
6. expect="rpc"; method="eth_getBlockByNumber"; params=["0x6",false]; compare="Equal"; is=null

### authorized-accounts-no-space — tests/tc/go-stablenet/post-v1.0.0-change/string-handling/01-authorized-accounts-no-space.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/01-authorized-accounts-no-space.json:46`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="Equal"; data="0x6a6debd7"; expect="call"; is="0x0000000000000000000000000000000000000000000000000000000000000003"; to="0x0000000000000000000000000000000000001004"

### authorized-accounts-space — tests/tc/go-stablenet/post-v1.0.0-change/string-handling/02-authorized-accounts-space.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/02-authorized-accounts-space.json:46`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="Equal"; data="0x6a6debd7"; expect="call"; is="0x0000000000000000000000000000000000000000000000000000000000000003"; to="0x0000000000000000000000000000000000001004"

### authorized-accounts-trim — tests/tc/go-stablenet/post-v1.0.0-change/string-handling/03-authorized-accounts-trim.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/03-authorized-accounts-trim.json:46`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="Equal"; data="0x6a6debd7"; expect="call"; is="0x0000000000000000000000000000000000000000000000000000000000000002"; to="0x0000000000000000000000000000000000001004"

### authorized-accounts-empty-item — tests/tc/go-stablenet/post-v1.0.0-change/string-handling/04-authorized-accounts-empty-item.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/04-authorized-accounts-empty-item.json:46`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="Equal"; data="0x6a6debd7"; expect="call"; is="0x0000000000000000000000000000000000000000000000000000000000000002"; to="0x0000000000000000000000000000000000001004"

### authorized-accounts-single — tests/tc/go-stablenet/post-v1.0.0-change/string-handling/05-authorized-accounts-single.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/05-authorized-accounts-single.json:46`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="Equal"; data="0x6a6debd7"; expect="call"; is="0x0000000000000000000000000000000000000000000000000000000000000001"; to="0x0000000000000000000000000000000000001004"

### authorized-accounts-empty — tests/tc/go-stablenet/post-v1.0.0-change/string-handling/06-authorized-accounts-empty.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/post-v1.0.0-change/string-handling/06-authorized-accounts-empty.json:46`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="Equal"; data="0x6a6debd7"; expect="call"; is="0x0000000000000000000000000000000000000000000000000000000000000000"; to="0x0000000000000000000000000000000000001004"

### regular-account-gastip-forced — tests/tc/go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/anzeon/01-regular-account-gastip-forced.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="read"; method="eth_blockNumber"; save="n"; source="rpcCall"
2. do="read"; method="istanbul_getWbftExtraInfo"; params=["$n"]; save="htip"; select="gasTip"; source="rpcCall"
3. do="read"; of=["$htip","$htip","$htip","$htip","$htip"]; op="sum"; save="hightip"; source="derive"
4. do="sendTx"; from="node1"; gas="21000"; maxFeePerGas="1000000000000000"; maxPriorityFeePerGas="$hightip"; save="hash"; to="0x70997970C51812dc3A010C7d01b50e0d17dc79C8"; value="1"
5. do="read"; method="eth_getTransactionReceipt"; params=["$hash"]; save="egp"; select="effectiveGasPrice"; source="rpcCall"
6. do="read"; method="eth_getTransactionReceipt"; params=["$hash"]; save="blk"; select="blockNumber"; source="rpcCall"
7. do="read"; method="eth_getBlockByNumber"; params=["$blk",false]; save="base"; select="baseFeePerGas"; source="rpcCall"
8. do="read"; method="istanbul_getWbftExtraInfo"; params=["$blk"]; save="itip"; select="gasTip"; source="rpcCall"
9. compare="Equal"; expect="derive"; is="$itip"; of=["$egp","$base"]; op="diff"

### authorized-account-gastip-free — tests/tc/go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/anzeon/02-authorized-account-gastip-free.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="newAccount"; save="acct"; saveKey="acctKey"
2. do="sendTx"; from="node1"; gas="21000"; on="node1"; save="fundHash"; to="$acct"; value="30000000000000000000"
3. do="read"; of=["$acct"]; op="abiCall"; save="authData"; selector="0x93a8bb99"; source="derive"
4. data="$authData"; do="sendTx"; from="node1"; gas="0x16e360"; on="node1"; save="authHash"; to="0x0000000000000000000000000000000000001004"
5. do="read"; hash="$authHash"; save="pid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
6. do="read"; of=["$pid"]; op="abiCall"; save="approveData"; selector="0x98951b56"; source="derive"
7. data="$approveData"; do="sendTx"; from="node2"; gas="0x16e360"; on="node2"; save="apHash"; to="0x0000000000000000000000000000000000001004"
8. do="read"; of=["$acct"]; op="abiCall"; save="isAuthData"; selector="0xfe9fbb80"; source="derive"
9. compare="Equal"; data="$isAuthData"; do="waitFor"; expected="0x0000000000000000000000000000000000000000000000000000000000000001"; pollInterval="1s"; source="call"; timeout="60s"; to="0x0000000000000000000000000000000000B00003"
10. do="read"; method="eth_blockNumber"; save="n"; source="rpcCall"
11. do="read"; method="istanbul_getWbftExtraInfo"; params=["$n"]; save="htip"; select="gasTip"; source="rpcCall"
12. do="read"; of=["$htip","$htip","$htip"]; op="sum"; save="customTip"; source="derive"
13. do="sendTx"; gas="21000"; key="$acctKey"; maxFeePerGas="1000000000000000"; maxPriorityFeePerGas="$customTip"; on="en1"; save="hash"; to="0x70997970C51812dc3A010C7d01b50e0d17dc79C8"; value="1"
14. do="read"; method="eth_getTransactionReceipt"; params=["$hash"]; save="egp"; select="effectiveGasPrice"; source="rpcCall"
15. do="read"; method="eth_getTransactionReceipt"; params=["$hash"]; save="blk"; select="blockNumber"; source="rpcCall"
16. do="read"; method="eth_getBlockByNumber"; params=["$blk",false]; save="base"; select="baseFeePerGas"; source="rpcCall"
17. expect="txStatus"; hash="$fundHash"; is="0x1"
18. expect="txStatus"; hash="$apHash"; is="0x1"
19. expect="txStatus"; hash="$hash"; is="0x1"
20. compare="Equal"; expect="derive"; is="$customTip"; of=["$egp","$base"]; op="diff"
21. expect="receiptLog"; hash="$hash"; is="0x40e728a89c7f5b192cf1c1b747fb64d51d81c7a2b3ed4607b94d3a1e6a3e0373"; topic=0; topic0="0x40e728a89c7f5b192cf1c1b747fb64d51d81c7a2b3ed4607b94d3a1e6a3e0373"

### anzeon-basefee-increase — tests/tc/go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/anzeon/03-anzeon-basefee-increase.json:26`
- 적용: env.chain=wemix와 gwemix 바이너리, PoA genesis/계정/수수료를 준비하고 applicableChains에 wemix를 포함한다. PoA governance gasTargetPercentage보다 높은 gasUsed를 실제로 확인한다. 단순 25% load로 증가를 단정하지 않고 maxBaseFee에서 떨어진 초기 baseFee를 쓴다. London 활성화 및 PoA governance 초기화 완료를 확인한다. Anzeon 지원 검증으로 세지 않는다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`, `sources/pr-head/consensus/misc/eip1559.go:66`

1. do="waitBlock"; target=2; timeout="60s"
2. do="read"; source="baseFee"; save="bf0"
3. do="load"; from="0xc17d493883eaa3b4cceb0f214b273392d562f9d8"; fillPercent=25; blocks=6; timeout="60s"
4. expect="baseFee"; compare="Greater"; is="$bf0"

### anzeon-basefee-stable — tests/tc/go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/anzeon/04-anzeon-basefee-stable.json:26`
- 적용: env.chain=wemix와 gwemix 바이너리, PoA genesis/계정/수수료를 준비하고 applicableChains에 wemix를 포함한다. gasUsed == gasLimit * gasTargetPercentage / 100인 부모 블록을 만들어 다음 baseFee 불변을 비교한다. 임의 10% load는 충분하지 않다. London 활성화 및 PoA governance 초기화 완료를 확인한다. Anzeon 지원 검증으로 세지 않는다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`, `sources/pr-head/consensus/misc/eip1559.go:66`

1. do="waitBlock"; target=2; timeout="60s"
2. do="read"; source="baseFee"; save="bf0"
3. do="load"; from="0xc17d493883eaa3b4cceb0f214b273392d562f9d8"; fillPercent=10; blocks=6; timeout="60s"
4. expect="baseFee"; compare="Equal"; is="$bf0"

### anzeon-basefee-decrease — tests/tc/go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/anzeon/05-anzeon-basefee-decrease.json:26`
- 적용: env.chain=wemix와 gwemix 바이너리, PoA genesis/계정/수수료를 준비하고 applicableChains에 wemix를 포함한다. 초기 baseFee가 최저값 1 wei보다 크고 목표보다 낮은 gasUsed인 부모 블록에서 다음 baseFee 감소를 검사한다. London 활성화 및 PoA governance 초기화 완료를 확인한다. Anzeon 지원 검증으로 세지 않는다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`, `sources/pr-head/consensus/misc/eip1559.go:66`

1. do="waitBlock"; target=2; timeout="60s"
2. do="load"; from="0xc17d493883eaa3b4cceb0f214b273392d562f9d8"; fillPercent=25; blocks=6; timeout="60s"
3. do="read"; source="baseFee"; save="peak"
4. do="waitBlock"; target=30; timeout="120s"
5. expect="baseFee"; compare="Less"; is="$peak"

### basefee-minimum — tests/tc/go-stablenet/regression/anzeon/06-basefee-minimum.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/anzeon/06-basefee-minimum.json:31`
- 적용: env.chain=wemix와 gwemix 바이너리, PoA genesis/계정/수수료를 준비하고 applicableChains에 wemix를 포함한다. 20,000,000,000,000 wei를 go-wemix 최저값 1 wei로 교체한다. 단순 한 번 읽는 assertion의 한계를 명시한다. London 활성화 및 PoA governance 초기화 완료를 확인한다. Anzeon 지원 검증으로 세지 않는다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`, `sources/pr-head/consensus/misc/eip1559.go:66`

1. compare="GreaterOrEqual"; expect="baseFee"; is="20000000000000"

### basefee-maximum — tests/tc/go-stablenet/regression/anzeon/07-basefee-maximum.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/anzeon/07-basefee-maximum.json:31`
- 적용: env.chain=wemix와 gwemix 바이너리, PoA genesis/계정/수수료를 준비하고 applicableChains에 wemix를 포함한다. Stablenet 고정 상한 대신 PoA governance maxBaseFee를 사용하고 상한 도달 조건을 만든다. London 활성화 및 PoA governance 초기화 완료를 확인한다. Anzeon 지원 검증으로 세지 않는다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`, `sources/pr-head/consensus/misc/eip1559.go:66`

1. compare="LessOrEqual"; expect="baseFee"; is="20000000000000000"

### feecap-above-min-accepted — tests/tc/go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/anzeon/08-feecap-above-min-accepted.json:30`
- 적용: env.chain=wemix와 gwemix 바이너리, PoA genesis/계정/수수료를 준비하고 applicableChains에 wemix를 포함한다. eth_maxPriorityFeePerGas 및 대상 노드 txpool 하한으로 tip을 준비한다. feeCap 여유와 수신 승인/채굴 성공을 검증한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="read"; method="eth_blockNumber"; save="n"; source="rpcCall"
2. do="read"; method="eth_getBlockByNumber"; params=["$n",false]; save="base"; select="baseFeePerGas"; source="rpcCall"
3. do="read"; method="istanbul_getWbftExtraInfo"; params=["$n"]; save="tip"; select="gasTip"; source="rpcCall"
4. do="read"; of=["$base","$base","$tip"]; op="sum"; save="feecap"; source="derive"
5. do="sendTx"; from="node1"; gas="21000"; maxFeePerGas="$feecap"; maxPriorityFeePerGas="$tip"; save="hash"; to="0x00000000000000000000000000000000C0FFEE0E"; value="1"
6. expect="txStatus"; hash="$hash"; is="0x1"

### feecap-exact-min-accepted — tests/tc/go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/anzeon/09-feecap-exact-min-accepted.json:30`
- 적용: env.chain=wemix와 gwemix 바이너리, PoA genesis/계정/수수료를 준비하고 applicableChains에 wemix를 포함한다. txpool의 해당 시점 baseFee + 유효 tip 경계로 설정한다. head 변화로 경계가 바뀌지 않도록 관측/채굴 제어한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="read"; method="eth_blockNumber"; save="n"; source="rpcCall"
2. do="read"; method="eth_getBlockByNumber"; params=["$n",false]; save="base"; select="baseFeePerGas"; source="rpcCall"
3. do="read"; method="istanbul_getWbftExtraInfo"; params=["$n"]; save="tip"; select="gasTip"; source="rpcCall"
4. do="read"; of=["$base","$tip"]; op="sum"; save="feecap"; source="derive"
5. do="sendTx"; from="node1"; gas="21000"; maxFeePerGas="$feecap"; maxPriorityFeePerGas="$tip"; save="hash"; to="0x00000000000000000000000000000000C0FFEE0E"; value="1"
6. expect="txStatus"; hash="$hash"; is="0x1"

### gaslimit-exceeded-rejected — tests/tc/go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json

- `P1` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/anzeon/11-gaslimit-exceeded-rejected.json:30`
- 적용: env.chain=wemix와 gwemix 바이너리, PoA genesis/계정/수수료를 준비하고 applicableChains에 wemix를 포함한다. Istanbul tip 읽기를 교체하고 나머지는 유효한 tx로 gasLimit+1에 따른 ErrGasLimit을 검사한다. 이미 채택한 TC-035와 동일 기능이다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="read"; method="eth_blockNumber"; save="n"; source="rpcCall"
2. do="read"; method="eth_getBlockByNumber"; params=["$n",false]; save="base"; select="baseFeePerGas"; source="rpcCall"
3. do="read"; method="istanbul_getWbftExtraInfo"; params=["$n"]; save="tip"; select="gasTip"; source="rpcCall"
4. do="read"; of=["$base","$tip"]; op="sum"; save="feecap"; source="derive"
5. do="read"; method="eth_getBlockByNumber"; params=["$n",false]; save="gl"; select="gasLimit"; source="rpcCall"
6. do="read"; of=["$gl","1"]; op="sum"; save="over"; source="derive"
7. do="sendTx"; expect="reject"; from="node1"; gas="$over"; maxFeePerGas="$feecap"; maxPriorityFeePerGas="$tip"; to="0x00000000000000000000000000000000C0FFEE0E"; value="1"
8. compare="GreaterOrEqual"; expect="blockNumber"; is="0x1"

### system-contracts-deployed — tests/tc/go-stablenet/regression/api/06-system-contracts-deployed.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/api/06-system-contracts-deployed.json:31`
- 적용: 일반 RPC 항목과 구분; 대상 계약/API가 달라 현재 케이스 제외.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. address="0x0000000000000000000000000000000000001000"; compare="NotEqual"; expect="codeAt"; is="0x"
2. address="0x0000000000000000000000000000000000001001"; compare="NotEqual"; expect="codeAt"; is="0x"
3. address="0x0000000000000000000000000000000000001002"; compare="NotEqual"; expect="codeAt"; is="0x"
4. address="0x0000000000000000000000000000000000001003"; compare="NotEqual"; expect="codeAt"; is="0x"
5. address="0x0000000000000000000000000000000000001004"; compare="NotEqual"; expect="codeAt"; is="0x"

### gas-price-equals-basefee-plus-tip — tests/tc/go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json

- `P2` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/api/07b-gas-price-equals-basefee-plus-tip.json:30`
- 적용: env.chain=wemix와 gwemix 바이너리, PoA genesis/계정/수수료를 준비하고 applicableChains에 wemix를 포함한다. Istanbul GasTip 대신 같은 head 시점의 eth_maxPriorityFeePerGas 값을 사용하여 eth_gasPrice == baseFee + suggestedTip을 비교한다. 샘플링 중 head 변경을 통제한다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="read"; save="gp"; source="gasPrice"
2. do="read"; method="eth_getBlockByNumber"; params=["latest",false]; save="n"; select="number"; source="rpcCall"
3. do="read"; method="eth_getBlockByNumber"; params=["$n",false]; save="base"; select="baseFeePerGas"; source="rpcCall"
4. do="read"; method="istanbul_getWbftExtraInfo"; params=["$n"]; save="tip"; select="gasTip"; source="rpcCall"
5. compare="Equal"; expect="derive"; is="$gp"; of=["$base","$tip"]; op="sum"

### max-priority-fee-equals-gastip — tests/tc/go-stablenet/regression/api/08-max-priority-fee-equals-gastip.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/api/08-max-priority-fee-equals-gastip.json:31`
- 적용: 일반 RPC 항목과 구분; 대상 계약/API가 달라 현재 케이스 제외.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="read"; method="eth_maxPriorityFeePerGas"; params=[]; save="mpf"; source="rpcCall"
2. do="read"; method="eth_getBlockByNumber"; params=["latest",false]; save="n"; select="number"; source="rpcCall"
3. do="read"; method="istanbul_getWbftExtraInfo"; params=["$n"]; save="tip"; select="gasTip"; source="rpcCall"
4. do="read"; of=["$tip"]; op="sum"; save="tipDec"; source="derive"
5. compare="Equal"; expect="derive"; is="$tipDec"; of=["$mpf"]; op="sum"

### estimate-gas-token-transfer — tests/tc/go-stablenet/regression/api/10-estimate-gas-token-transfer.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/api/10-estimate-gas-token-transfer.json:31`
- 적용: 일반 RPC 항목과 구분; 대상 계약/API가 달라 현재 케이스 제외.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="GreaterOrEqual"; data="0xa9059cbb00000000000000000000000000000000000000000000000000000000c0ffee0900000000000000000000000000000000000000000000000000000000000003e8"; expect="estimateGas"; from="node1"; is="21001"; to="0x0000000000000000000000000000000000001000"

### node-address-returned — tests/tc/go-stablenet/regression/api/11-node-address-returned.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/api/11-node-address-returned.json:33`
- 적용: 일반 RPC 항목과 구분; 대상 계약/API가 달라 현재 케이스 제외.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="Regexp"; expect="rpcCall"; is="^0x[0-9a-fA-F]{40}$"; method="istanbul_nodeAddress"; onEach=["bp1","bp2","bp3","bp4"]

### validator-set-nonempty — tests/tc/go-stablenet/regression/api/12-validator-set-nonempty.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/api/12-validator-set-nonempty.json:33`
- 적용: 일반 RPC 항목과 구분; 대상 계약/API가 달라 현재 케이스 제외.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="NotNil"; expect="rpcCall"; is=null; method="istanbul_getValidators"; params=["latest"]
2. compare="Len"; expect="rpcCall"; is=4; method="istanbul_getValidators"; params=["latest"]

### validator-set-count — tests/tc/go-stablenet/regression/api/12b-validator-set-count.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/api/12b-validator-set-count.json:32`
- 적용: 일반 RPC 항목과 구분; 대상 계약/API가 달라 현재 케이스 제외.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="GreaterOrEqual"; do="waitFor"; expected="1"; pollInterval="1s"; source="blockNumber"; timeout="60s"
2. compare="GreaterOrEqual"; expect="rpcCall"; is="4"; method="istanbul_getValidators"; params=["latest"]; select="#"

### commit-signers-quorum — tests/tc/go-stablenet/regression/api/13-commit-signers-quorum.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/api/13-commit-signers-quorum.json:33`
- 적용: 일반 RPC 항목과 구분; 대상 계약/API가 달라 현재 케이스 제외.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="read"; method="istanbul_getValidators"; params=["latest"]; save="validators"; source="rpcCall"
2. compare="NotNil"; expect="rpcCall"; is=null; method="istanbul_getCommitSignersFromBlock"; params=["latest"]
3. compare="Equal"; expect="rpcCall"; is="$validators"; method="istanbul_getValidators"; params=["latest"]

### wbft-extra-info-fields — tests/tc/go-stablenet/regression/api/14-wbft-extra-info-fields.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/api/14-wbft-extra-info-fields.json:31`
- 적용: 일반 RPC 항목과 구분; 대상 계약/API가 달라 현재 케이스 제외.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="read"; method="eth_blockNumber"; save="head"; source="rpcCall"
2. compare="NotNil"; expect="rpcCall"; is=null; method="istanbul_getWbftExtraInfo"; params=["$head"]; select="committedSeal"
3. compare="NotNil"; expect="rpcCall"; is=null; method="istanbul_getWbftExtraInfo"; params=["$head"]; select="preparedSeal"

### istanbul-status-fields — tests/tc/go-stablenet/regression/api/15-istanbul-status-fields.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/api/15-istanbul-status-fields.json:31`
- 적용: 일반 RPC 항목과 구분; 대상 계약/API가 달라 현재 케이스 제외.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="NotNil"; expect="rpcCall"; is=null; method="istanbul_status"; params=["0x1","0x2"]; select="sealerActivity"
2. compare="NotNil"; expect="rpcCall"; is=null; method="istanbul_status"; params=["0x1","0x2"]; select="author"
3. compare="NotNil"; expect="rpcCall"; is=null; method="istanbul_status"; params=["0x1","0x2"]; select="blockRange"
4. compare="NotNil"; expect="rpcCall"; is=null; method="istanbul_status"; params=["0x1","0x2"]; select="roundStats"

### is-validator-flags — tests/tc/go-stablenet/regression/api/16-is-validator-flags.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/api/16-is-validator-flags.json:33`
- 적용: 일반 RPC 항목과 구분; 대상 계약/API가 달라 현재 케이스 제외.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. expect="rpcCall"; is=true; method="istanbul_isValidator"; onEach=["bp1","bp2","bp3","bp4"]; params=["latest"]

### token-total-supply-readable — tests/tc/go-stablenet/regression/api/22-token-total-supply-readable.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/api/22-token-total-supply-readable.json:31`
- 적용: 일반 RPC 항목과 구분; 대상 계약/API가 달라 현재 케이스 제외.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="NotEqual"; data="0x18160ddd"; expect="call"; is="0x"; to="0x0000000000000000000000000000000000001000"

### token-approve-sets-allowance — tests/tc/go-stablenet/regression/api/23-token-approve-sets-allowance.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/api/23-token-approve-sets-allowance.json:31`
- 적용: 일반 RPC 항목과 구분; 대상 계약/API가 달라 현재 케이스 제외.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. data="0x095ea7b300000000000000000000000000000000000000000000000000000000c0ffee050000000000000000000000000000000000000000000000000000000000000064"; do="sendTx"; from="node1"; gas="100000"; save="hash"; to="0x0000000000000000000000000000000000001000"
2. do="read"; method="eth_getTransactionReceipt"; params=["$hash"]; save="blk"; select="blockNumber"; source="rpcCall"
3. expect="txStatus"; hash="$hash"; is="0x1"
4. address="0x0000000000000000000000000000000000001000"; compare="GreaterOrEqual"; expect="logs"; fromBlock="$blk"; is="1"; select="count"; toBlock="$blk"; topics=["0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925",null,"0x00000000000000000000000000000000000000000000000000000000c0ffee05"]
5. compare="Equal"; data="0xdd62ed3e000000000000000000000000c17d493883eaa3b4cceb0f214b273392d562f9d800000000000000000000000000000000000000000000000000000000c0ffee05"; expect="call"; is="0x0000000000000000000000000000000000000000000000000000000000000064"; to="0x0000000000000000000000000000000000001000"

### sender-blacklisted-rejected — tests/tc/go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/01-sender-blacklisted-rejected.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="newAccount"; save="acct"; saveKey="acctKey"
2. do="sendTx"; from="node1"; gas="21000"; on="node1"; save="fundHash"; to="$acct"; value="10000000000000000000"
3. do="read"; of=["$acct"]; op="abiCall"; save="addBlData"; selector="0x0d321273"; source="derive"
4. data="$addBlData"; do="sendTx"; from="node1"; gas="0x16e360"; on="node1"; save="addHash"; to="0x0000000000000000000000000000000000001004"
5. do="read"; hash="$addHash"; save="addPid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
6. do="read"; of=["$addPid"]; op="abiCall"; save="addApprove"; selector="0x98951b56"; source="derive"
7. data="$addApprove"; do="sendTx"; from="node2"; gas="0x16e360"; on="node2"; save="addApHash"; to="0x0000000000000000000000000000000000001004"
8. do="sendTx"; expect="reject"; key="$acctKey"; on="en1"; reason="blacklist"; to="0x00000000000000000000000000000000C0FFEE10"; value="1"
9. expect="txStatus"; hash="$fundHash"; is="0x1"
10. expect="txStatus"; hash="$addApHash"; is="0x1"

### recipient-blacklisted-rejected — tests/tc/go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/02-recipient-blacklisted-rejected.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. data="0x0d32127300000000000000000000000000000000000000000000000000000000c0ffee32"; do="sendTx"; from="node1"; gas="0x16e360"; on="node1"; save="addHash"; to="0x0000000000000000000000000000000000001004"
2. do="read"; hash="$addHash"; save="addPid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
3. do="read"; of=["$addPid"]; op="abiCall"; save="addApprove"; selector="0x98951b56"; source="derive"
4. data="$addApprove"; do="sendTx"; from="node2"; gas="0x16e360"; on="node2"; save="addApHash"; to="0x0000000000000000000000000000000000001004"
5. data="0xfe575a8700000000000000000000000000000000000000000000000000000000c0ffee32"; do="read"; save="isBl"; source="call"; to="0x0000000000000000000000000000000000B00003"
6. do="sendTx"; expect="reject"; from="node1"; gas="21000"; on="node1"; reason="blacklist"; to="0x00000000000000000000000000000000C0FFEE32"; value="1"
7. data="0x3d4c045200000000000000000000000000000000000000000000000000000000c0ffee32"; do="sendTx"; from="node1"; gas="0x16e360"; on="node1"; save="rmHash"; to="0x0000000000000000000000000000000000001004"
8. do="read"; hash="$rmHash"; save="rmPid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
9. do="read"; of=["$rmPid"]; op="abiCall"; save="rmApprove"; selector="0x98951b56"; source="derive"
10. data="$rmApprove"; do="sendTx"; from="node2"; gas="0x16e360"; on="node2"; save="rmApHash"; to="0x0000000000000000000000000000000000001004"
11. expect="txStatus"; hash="$addApHash"; is="0x1"
12. compare="Equal"; expect="derive"; index=0; is="0x0000000000000000000000000000000000000000000000000000000000000001"; of=["$isBl"]; op="word"
13. expect="txStatus"; hash="$rmApHash"; is="0x1"

### feepayer-blacklisted-rejected — tests/tc/go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/03-feepayer-blacklisted-rejected.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="newAccount"; save="sender"; saveKey="senderKey"
2. do="newAccount"; save="feePayer"; saveKey="feePayerKey"
3. do="sendTx"; from="node1"; gas="21000"; on="node1"; save="fundSenderHash"; to="$sender"; value="10000000000000000000"
4. do="sendTx"; from="node1"; gas="21000"; on="node1"; save="fundFeeHash"; to="$feePayer"; value="10000000000000000000"
5. do="read"; of=["$feePayer"]; op="abiCall"; save="addBlData"; selector="0x0d321273"; source="derive"
6. data="$addBlData"; do="sendTx"; from="node1"; gas="0x16e360"; on="node1"; save="addHash"; to="0x0000000000000000000000000000000000001004"
7. do="read"; hash="$addHash"; save="addPid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
8. do="read"; of=["$addPid"]; op="abiCall"; save="addApprove"; selector="0x98951b56"; source="derive"
9. data="$addApprove"; do="sendTx"; from="node2"; gas="0x16e360"; on="node2"; save="addApHash"; to="0x0000000000000000000000000000000000001004"
10. do="sendTx"; expect="reject"; feePayerKey="$feePayerKey"; key="$senderKey"; on="en1"; reason="blacklist"; to="0x00000000000000000000000000000000C0FFEE13"; value="1"
11. expect="txStatus"; hash="$fundSenderHash"; is="0x1"
12. expect="txStatus"; hash="$fundFeeHash"; is="0x1"
13. expect="txStatus"; hash="$addApHash"; is="0x1"

### address-unblacklisted-event — tests/tc/go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/04-address-unblacklisted-event.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. data="0x0d32127300000000000000000000000000000000000000000000000000000000c0ffee13"; do="sendTx"; from="node1"; gas="0x16e360"; on="node1"; save="addHash"; to="0x0000000000000000000000000000000000001004"
2. do="read"; hash="$addHash"; save="addPid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
3. do="read"; of=["$addPid"]; op="abiCall"; save="addApprove"; selector="0x98951b56"; source="derive"
4. data="$addApprove"; do="sendTx"; from="node2"; gas="0x16e360"; on="node2"; save="addApHash"; to="0x0000000000000000000000000000000000001004"
5. data="0x3d4c045200000000000000000000000000000000000000000000000000000000c0ffee13"; do="sendTx"; from="node1"; gas="0x16e360"; on="node1"; save="rmHash"; to="0x0000000000000000000000000000000000001004"
6. do="read"; hash="$rmHash"; save="rmPid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
7. do="read"; of=["$rmPid"]; op="abiCall"; save="rmApprove"; selector="0x98951b56"; source="derive"
8. data="$rmApprove"; do="sendTx"; from="node2"; gas="0x16e360"; on="node2"; save="rmApHash"; to="0x0000000000000000000000000000000000001004"
9. do="read"; method="eth_getTransactionReceipt"; params=["$rmApHash"]; save="rmApBlk"; select="blockNumber"; source="rpcCall"
10. expect="txStatus"; hash="$addApHash"; is="0x1"
11. expect="txStatus"; hash="$rmApHash"; is="0x1"
12. compare="Equal"; data="0xfe575a8700000000000000000000000000000000000000000000000000000000c0ffee13"; expect="call"; is="0x0000000000000000000000000000000000000000000000000000000000000000"; to="0x0000000000000000000000000000000000B00003"
13. address="0x0000000000000000000000000000000000001004"; compare="GreaterOrEqual"; expect="logs"; fromBlock="$rmApBlk"; is="1"; select="count"; toBlock="$rmApBlk"; topics=["0x5428c9b1fb7549fe41c6846061bfb4c682760e630052c3d880669ece73427cb7","0x00000000000000000000000000000000000000000000000000000000c0ffee13"]

### zero-address-transfer-rejected — tests/tc/go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/05-zero-address-transfer-rejected.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="sendTx"; expect="reject"; from="node1"; gas="100000"; on="node1"; reason="zero"; to="0x0000000000000000000000000000000000000000"; value="1"
2. compare="GreaterOrEqual"; expect="blockNumber"; is="0x1"

### precompile-transfer-rejected — tests/tc/go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/06-precompile-transfer-rejected.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="sendTx"; expect="reject"; from="node1"; gas="100000"; on="node1"; reason="precompile"; to="0x0000000000000000000000000000000000000001"; value="1"
2. do="sendTx"; expect="reject"; from="node1"; gas="100000"; on="node1"; reason="precompile"; to="0x0000000000000000000000000000000000000100"; value="1"
3. do="sendTx"; expect="reject"; from="node1"; gas="100000"; on="node1"; reason="precompile"; to="0x0000000000000000000000000000000000b00001"; value="1"
4. do="sendTx"; expect="reject"; from="node1"; gas="100000"; on="node1"; reason="precompile"; to="0x0000000000000000000000000000000000b00002"; value="1"
5. do="sendTx"; expect="reject"; from="node1"; gas="100000"; on="node1"; reason="precompile"; to="0x0000000000000000000000000000000000b00003"; value="1"
6. compare="GreaterOrEqual"; expect="blockNumber"; is="0x1"

### account-blacklist-readable — tests/tc/go-stablenet/regression/blacklist-authorized/07-account-blacklist-readable.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/07-account-blacklist-readable.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="NotEqual"; data="0xfe575a87000000000000000000000000c17d493883eaa3b4cceb0f214b273392d562f9d8"; expect="call"; is="0x"; to="0x0000000000000000000000000000000000B00003"

### account-authorization-readable — tests/tc/go-stablenet/regression/blacklist-authorized/08-account-authorization-readable.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/08-account-authorization-readable.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="NotEqual"; data="0xfe9fbb80000000000000000000000000c17d493883eaa3b4cceb0f214b273392d562f9d8"; expect="call"; is="0x"; to="0x0000000000000000000000000000000000B00003"

### authorized-tx-executed-event — tests/tc/go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/blacklist-authorized/09-authorized-tx-executed-event.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="newAccount"; save="acct"; saveKey="acctKey"
2. do="sendTx"; from="node1"; gas="21000"; on="node1"; save="fundHash"; to="$acct"; value="10000000000000000000"
3. do="read"; of=["$acct"]; op="abiCall"; save="authData"; selector="0x93a8bb99"; source="derive"
4. data="$authData"; do="sendTx"; from="node1"; gas="0x16e360"; on="node1"; save="authHash"; to="0x0000000000000000000000000000000000001004"
5. do="read"; hash="$authHash"; save="authPid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
6. do="read"; of=["$authPid"]; op="abiCall"; save="authApprove"; selector="0x98951b56"; source="derive"
7. data="$authApprove"; do="sendTx"; from="node2"; gas="0x16e360"; on="node2"; save="authApHash"; to="0x0000000000000000000000000000000000001004"
8. do="sendTx"; key="$acctKey"; on="en1"; save="txHash"; to="0x70997970C51812dc3A010C7d01b50e0d17dc79C8"; value="1"
9. expect="txStatus"; hash="$fundHash"; is="0x1"
10. expect="txStatus"; hash="$authApHash"; is="0x1"
11. expect="txStatus"; hash="$txHash"; is="0x1"
12. expect="receiptLog"; hash="$txHash"; is="0x40e728a89c7f5b192cf1c1b747fb64d51d81c7a2b3ed4607b94d3a1e6a3e0373"; topic=0; topic0="0x40e728a89c7f5b192cf1c1b747fb64d51d81c7a2b3ed4607b94d3a1e6a3e0373"

### set-code-delegation — tests/tc/go-stablenet/regression/ethereum/18-set-code-delegation.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/ethereum/18-set-code-delegation.json:38`
- 적용: 지원되는 type 0/1/2/22 검증과 구분.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="newAccount"; save="sponsor"; saveKey="sponsorKey"
2. do="newAccount"; save="authority"; saveKey="authorityKey"
3. do="sendTx"; from="node1"; gas="21000"; on="node1"; save="fundHash"; to="$sponsor"; value="30000000000000000000"
4. authorityKey="$authorityKey"; delegate="0x1111111111111111111111111111111111111111"; do="sendSetCode"; key="$sponsorKey"; on="en1"; save="scHash"
5. address="$authority"; compare="Equal"; do="waitFor"; expected="0xef01001111111111111111111111111111111111111111"; pollInterval="1s"; source="codeAt"; timeout="90s"
6. expect="txStatus"; hash="$fundHash"; is="0x1"
7. expect="txStatus"; hash="$scHash"; is="0x1"
8. compare="Equal"; expect="rpcCall"; is="0x4"; method="eth_getTransactionByHash"; params=["$scHash"]; select="type"
9. address="$authority"; compare="Equal"; expect="codeAt"; is="0xef01001111111111111111111111111111111111111111"

### native-coin-adapter-code — tests/tc/go-stablenet/regression/system-contracts/01-native-coin-adapter-code.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/01-native-coin-adapter-code.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. address="0x0000000000000000000000000000000000001000"; compare="NotEqual"; expect="codeAt"; is="0x"

### token-transfer-emits-event — tests/tc/go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/01b-token-transfer-emits-event.json:30`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. data="0xa9059cbb00000000000000000000000000000000000000000000000000000000c0ffee040000000000000000000000000000000000000000000000000000000000000001"; do="sendTx"; from="node1"; gas="100000"; save="hash"; to="0x0000000000000000000000000000000000001000"
2. do="read"; method="eth_getTransactionReceipt"; params=["$hash"]; save="blk"; select="blockNumber"; source="rpcCall"
3. expect="txStatus"; hash="$hash"; is="0x1"
4. address="0x0000000000000000000000000000000000001000"; compare="GreaterOrEqual"; expect="logs"; fromBlock="$blk"; is="1"; select="count"; toBlock="$blk"; topics=["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",null,"0x00000000000000000000000000000000000000000000000000000000c0ffee04"]
5. address="0x0000000000000000000000000000000000001000"; compare="Equal"; expect="logs"; fromBlock="$blk"; is="0x0000000000000000000000000000000000000000000000000000000000000001"; select="data"; toBlock="$blk"; topics=["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",null,"0x00000000000000000000000000000000000000000000000000000000c0ffee04"]

### token-balance-readable — tests/tc/go-stablenet/regression/system-contracts/02-token-balance-readable.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/02-token-balance-readable.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. data="0x70a08231000000000000000000000000c17d493883eaa3b4cceb0f214b273392d562f9d8"; do="read"; save="bal"; source="call"; to="0x0000000000000000000000000000000000001000"
2. compare="GreaterOrEqual"; data="0x18160ddd"; expect="call"; is="$bal"; to="0x0000000000000000000000000000000000001000"

### token-transfer-from-moves-balance — tests/tc/go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/03-token-transfer-from-moves-balance.json:30`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="newAccount"; save="owner"; saveKey="ownerKey"
2. do="newAccount"; save="spender"; saveKey="spenderKey"
3. do="newAccount"; save="recipient"; saveKey="recipientKey"
4. do="sendTx"; from="node1"; gas="21000"; on="node1"; save="fundOwnerHash"; to="$owner"; value="5000000000000000000"
5. do="sendTx"; from="node1"; gas="21000"; on="node1"; save="fundSpenderHash"; to="$spender"; value="10000000000000000000"
6. do="read"; of=["$spender","1000000"]; op="abiCall"; save="approveData"; selector="0x095ea7b3"; source="derive"
7. data="$approveData"; do="sendTx"; key="$ownerKey"; on="en1"; save="approveHash"; to="0x0000000000000000000000000000000000001000"
8. do="read"; of=["$owner","$recipient","1000000"]; op="abiCall"; save="transferFromData"; selector="0x23b872dd"; source="derive"
9. data="$transferFromData"; do="sendTx"; key="$spenderKey"; on="en1"; save="transferFromHash"; to="0x0000000000000000000000000000000000001000"
10. do="read"; of=["$recipient"]; op="abiCall"; save="balOfRecipient"; selector="0x70a08231"; source="derive"
11. expect="txStatus"; hash="$fundOwnerHash"; is="0x1"
12. expect="txStatus"; hash="$fundSpenderHash"; is="0x1"
13. expect="txStatus"; hash="$approveHash"; is="0x1"
14. expect="txStatus"; hash="$transferFromHash"; is="0x1"
15. compare="Equal"; data="$balOfRecipient"; expect="call"; is="0x00000000000000000000000000000000000000000000000000000000000f4240"; to="0x0000000000000000000000000000000000001000"

### mint-transfer-event — tests/tc/go-stablenet/regression/system-contracts/04-mint-transfer-event.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/04-mint-transfer-event.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. address="0x00000000000000000000000000000000C0FFEE30"; do="read"; save="b0"; source="balanceAt"
2. do="read"; of=["$b0","1000000000000000000"]; op="sum"; save="want"; source="derive"
3. data=hex payload (452 bytes; 원문 참조); do="sendTx"; from="node1"; gas="0x16e360"; on="node1"; save="propHash"; to="0x0000000000000000000000000000000000001003"
4. do="read"; hash="$propHash"; save="pid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
5. do="read"; of=["$pid"]; op="abiCall"; save="approveData"; selector="0x98951b56"; source="derive"
6. data="$approveData"; do="sendTx"; from="node2"; gas="0x16e360"; on="node2"; save="apHash"; to="0x0000000000000000000000000000000000001003"
7. do="read"; method="eth_getTransactionReceipt"; params=["$apHash"]; save="apBlk"; select="blockNumber"; source="rpcCall"
8. expect="txStatus"; hash="$propHash"; is="0x1"
9. expect="txStatus"; hash="$apHash"; is="0x1"
10. address="0x00000000000000000000000000000000C0FFEE30"; expect="balanceAt"; is="$want"
11. address="0x0000000000000000000000000000000000001000"; compare="GreaterOrEqual"; expect="logs"; fromBlock="$apBlk"; is="1"; select="count"; toBlock="$apBlk"; topics=["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef","0x0000000000000000000000000000000000000000000000000000000000000000","0x00000000000000000000000000000000000000000000000000000000c0ffee30"]

### burn-transfer-event — tests/tc/go-stablenet/regression/system-contracts/05-burn-transfer-event.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/05-burn-transfer-event.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. data=hex payload (452 bytes; 원문 참조); do="sendTx"; from="node1"; gas="0xb71b0"; on="node1"; save="burnHash"; to="0x0000000000000000000000000000000000001003"; value="0xde0b6b3a7640000"
2. do="read"; method="eth_getTransactionReceipt"; params=["$burnHash"]; save="burnBlk"; select="blockNumber"; source="rpcCall"
3. expect="txStatus"; hash="$burnHash"; is="0x1"
4. address="0x0000000000000000000000000000000000001000"; compare="GreaterOrEqual"; expect="logs"; fromBlock="$burnBlk"; is="1"; select="count"; toBlock="$burnBlk"; topics=["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef","0x000000000000000000000000c17d493883eaa3b4cceb0f214b273392d562f9d8","0x0000000000000000000000000000000000000000000000000000000000001003"]

### mint-proposal-executes — tests/tc/go-stablenet/regression/system-contracts/06-mint-proposal-executes.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/06-mint-proposal-executes.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. address="0x00000000000000000000000000000000C0FFEE05"; do="read"; save="b0"; source="balanceAt"
2. do="read"; of=["$b0","1000000000000000000"]; op="sum"; save="want"; source="derive"
3. data=hex payload (452 bytes; 원문 참조); do="sendTx"; from="node1"; gas="0x16e360"; on="node1"; save="propHash"; to="0x0000000000000000000000000000000000001003"
4. do="read"; hash="$propHash"; save="pid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
5. do="read"; of=["$pid"]; op="abiCall"; save="approveData"; selector="0x98951b56"; source="derive"
6. data="$approveData"; do="sendTx"; from="node2"; gas="0x16e360"; on="node2"; save="apHash"; to="0x0000000000000000000000000000000000001003"
7. expect="txStatus"; hash="$propHash"; is="0x1"
8. expect="txStatus"; hash="$apHash"; is="0x1"
9. address="0x00000000000000000000000000000000C0FFEE05"; expect="balanceAt"; is="$want"

### burn-proposal-executes — tests/tc/go-stablenet/regression/system-contracts/07-burn-proposal-executes.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/07-burn-proposal-executes.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. data=hex payload (452 bytes; 원문 참조); do="sendTx"; from="node1"; gas="0xb71b0"; on="node1"; save="burnHash"; to="0x0000000000000000000000000000000000001003"; value="0x38d7ea4c68000"
2. do="read"; hash="$burnHash"; save="burnPid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
3. do="read"; of=["$burnPid"]; op="abiCall"; save="burnApprove"; selector="0x98951b56"; source="derive"
4. data="$burnApprove"; do="sendTx"; from="node2"; gas="0xb71b0"; on="node2"; save="burnApHash"; to="0x0000000000000000000000000000000000001003"
5. do="read"; of=["$burnPid"]; op="abiCall"; save="proposalsCall"; selector="0x013cf08b"; source="derive"
6. data="$proposalsCall"; do="read"; save="proposalsRet"; source="call"; to="0x0000000000000000000000000000000000001003"
7. expect="txStatus"; hash="$burnHash"; is="0x1"
8. expect="txStatus"; hash="$burnApHash"; is="0x1"
9. compare="Equal"; expect="derive"; index=9; is="0x0000000000000000000000000000000000000000000000000000000000000003"; of=["$proposalsRet"]; op="word"

### quorum-deficient-stays-voting — tests/tc/go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/08-quorum-deficient-stays-voting.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. data=hex payload (452 bytes; 원문 참조); do="sendTx"; from="node1"; gas="0x16e360"; on="node1"; save="propHash"; to="0x0000000000000000000000000000000000001003"
2. do="read"; hash="$propHash"; save="pid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
3. do="read"; of=["$pid"]; op="abiCall"; save="execData"; selector="0x0d61b519"; source="derive"
4. data="$execData"; do="sendTx"; expect="revert"; from="node1"; gas="0x16e360"; on="node1"; to="0x0000000000000000000000000000000000001003"
5. do="read"; of=["$pid"]; op="abiCall"; save="proposalsCall"; selector="0x013cf08b"; source="derive"
6. data="$proposalsCall"; do="read"; save="proposalsRet"; source="call"; to="0x0000000000000000000000000000000000001003"
7. do="read"; of=["$pid"]; op="abiCall"; save="approveData"; selector="0x98951b56"; source="derive"
8. data="$approveData"; do="sendTx"; from="node2"; gas="0x16e360"; on="node2"; save="apHash"; to="0x0000000000000000000000000000000000001003"
9. expect="txStatus"; hash="$propHash"; is="0x1"
10. compare="Equal"; expect="derive"; index=9; is="0x0000000000000000000000000000000000000000000000000000000000000001"; of=["$proposalsRet"]; op="word"
11. expect="txStatus"; hash="$apHash"; is="0x1"

### validator-metadata-readable — tests/tc/go-stablenet/regression/system-contracts/09-validator-metadata-readable.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/09-validator-metadata-readable.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="NotEqual"; data="0x5890ef79"; expect="call"; is="0x"; to="0x0000000000000000000000000000000000001001"

### gastip-governance-updates-header — tests/tc/go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/10-gastip-governance-updates-header.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="read"; method="eth_blockNumber"; save="h0"; source="rpcCall"
2. do="read"; method="istanbul_getWbftExtraInfo"; params=["$h0"]; save="orig"; select="gasTip"; source="rpcCall"
3. do="read"; of=["25000000000000"]; op="abiCall"; save="propData"; selector="0xeeaf6816"; source="derive"
4. data="$propData"; do="sendTx"; from="node1"; gas="0x16e360"; on="node1"; save="propHash"; to="0x0000000000000000000000000000000000001001"
5. do="read"; hash="$propHash"; save="pid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
6. do="read"; of=["$pid"]; op="abiCall"; save="approveData"; selector="0x98951b56"; source="derive"
7. data="$approveData"; do="sendTx"; from="node2"; gas="0x16e360"; on="node2"; save="apHash"; to="0x0000000000000000000000000000000000001001"
8. compare="Equal"; do="waitFor"; expected="25000000000000"; method="istanbul_getWbftExtraInfo"; params=["@latest"]; pollInterval="1s"; select="gasTip"; source="rpcCall"; timeout="60s"
9. do="read"; of=["$orig"]; op="abiCall"; save="restoreData"; selector="0xeeaf6816"; source="derive"
10. data="$restoreData"; do="sendTx"; from="node1"; gas="0x16e360"; on="node1"; save="restoreHash"; to="0x0000000000000000000000000000000000001001"
11. do="read"; hash="$restoreHash"; save="pid2"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
12. do="read"; of=["$pid2"]; op="abiCall"; save="approveData2"; selector="0x98951b56"; source="derive"
13. data="$approveData2"; do="sendTx"; from="node2"; gas="0x16e360"; on="node2"; save="apHash2"; to="0x0000000000000000000000000000000000001001"
14. compare="Equal"; do="waitFor"; expected="$orig"; method="istanbul_getWbftExtraInfo"; params=["@latest"]; pollInterval="1s"; select="gasTip"; source="rpcCall"; timeout="60s"
15. expect="txStatus"; hash="$propHash"; is="0x1"
16. expect="txStatus"; hash="$apHash"; is="0x1"
17. expect="txStatus"; hash="$restoreHash"; is="0x1"
18. expect="txStatus"; hash="$apHash2"; is="0x1"
19. address="0x0000000000000000000000000000000000001001"; compare="GreaterOrEqual"; expect="logs"; fromBlock="earliest"; is="1"; select="count"; toBlock="latest"; topics=["0x535d54b8a14e2287f10efd7add38eca14bcf44eb31651abf8eec2499eba393a3"]

### proposal-expiry-transitions — tests/tc/go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/11-proposal-expiry-transitions.json:33`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="read"; of=["30000000000001"]; op="abiCall"; save="propData"; selector="0xeeaf6816"; source="derive"
2. data="$propData"; do="sendTx"; from="node1"; gas="0x16e360"; on="node1"; save="propHash"; to="0x0000000000000000000000000000000000001001"
3. do="read"; hash="$propHash"; save="pid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
4. do="read"; method="eth_getBlockByNumber"; params=["@latest",false]; save="t0"; select="timestamp"; source="rpcCall"
5. do="read"; of=["$t0","40"]; op="sum"; save="expiryTarget"; source="derive"
6. compare="GreaterOrEqual"; do="waitFor"; expected="$expiryTarget"; method="eth_getBlockByNumber"; params=["@latest",false]; pollInterval="2s"; select="timestamp"; source="rpcCall"; timeout="90s"
7. do="read"; of=["$pid"]; op="abiCall"; save="expData"; selector="0xe1b526b0"; source="derive"
8. data="$expData"; do="sendTx"; from="node1"; gas="0x16e360"; on="node1"; save="expHash"; to="0x0000000000000000000000000000000000001001"
9. do="read"; of=["$pid"]; op="abiCall"; save="proposalsCall"; selector="0x013cf08b"; source="derive"
10. data="$proposalsCall"; do="read"; save="proposalsRet"; source="call"; to="0x0000000000000000000000000000000000001001"
11. expect="txStatus"; hash="$propHash"; is="0x1"
12. expect="txStatus"; hash="$expHash"; is="0x1"
13. compare="Equal"; expect="derive"; index=9; is="0x0000000000000000000000000000000000000000000000000000000000000005"; of=["$proposalsRet"]; op="word"

### configure-minter-proposal-executes — tests/tc/go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/12-configure-minter-proposal-executes.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. data="0x898420a900000000000000000000000000000000000000000000000000000000c0ffee070000000000000000000000000000000000000000000000008ac7230489e80000"; do="sendTx"; from="node1"; gas="0x16e360"; on="node1"; save="propHash"; to="0x0000000000000000000000000000000000001002"
2. do="read"; hash="$propHash"; save="pid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
3. do="read"; of=["$pid"]; op="abiCall"; save="approveData"; selector="0x98951b56"; source="derive"
4. data="$approveData"; do="sendTx"; from="node2"; gas="0x16e360"; on="node2"; save="apHash"; to="0x0000000000000000000000000000000000001002"
5. expect="txStatus"; hash="$propHash"; is="0x1"
6. expect="txStatus"; hash="$apHash"; is="0x1"
7. compare="Equal"; data="0x8a6db9c300000000000000000000000000000000000000000000000000000000c0ffee07"; expect="call"; is="0x0000000000000000000000000000000000000000000000008ac7230489e80000"; to="0x0000000000000000000000000000000000001000"

### remove-minter-executes — tests/tc/go-stablenet/regression/system-contracts/13-remove-minter-executes.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/13-remove-minter-executes.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. data="0x898420a900000000000000000000000000000000000000000000000000000000c0ffee200000000000000000000000000000000000000000000000008ac7230489e80000"; do="sendTx"; from="node1"; gas="0x7a120"; on="node1"; save="cfgHash"; to="0x0000000000000000000000000000000000001002"
2. do="read"; hash="$cfgHash"; save="cfgPid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
3. do="read"; of=["$cfgPid"]; op="abiCall"; save="cfgApprove"; selector="0x98951b56"; source="derive"
4. data="$cfgApprove"; do="sendTx"; from="node2"; gas="0x7a120"; on="node2"; save="cfgApHash"; to="0x0000000000000000000000000000000000001002"
5. data="0xaa271e1a00000000000000000000000000000000000000000000000000000000c0ffee20"; do="read"; save="cfgIsMinter"; source="call"; to="0x0000000000000000000000000000000000001000"
6. data="0x9336411700000000000000000000000000000000000000000000000000000000c0ffee20"; do="sendTx"; from="node1"; gas="0x7a120"; on="node1"; save="rmHash"; to="0x0000000000000000000000000000000000001002"
7. do="read"; hash="$rmHash"; save="rmPid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
8. do="read"; of=["$rmPid"]; op="abiCall"; save="rmApprove"; selector="0x98951b56"; source="derive"
9. data="$rmApprove"; do="sendTx"; from="node2"; gas="0x7a120"; on="node2"; save="rmApHash"; to="0x0000000000000000000000000000000000001002"
10. expect="txStatus"; hash="$cfgApHash"; is="0x1"
11. expect="txStatus"; hash="$rmApHash"; is="0x1"
12. compare="Equal"; expect="derive"; index=0; is="0x0000000000000000000000000000000000000000000000000000000000000001"; of=["$cfgIsMinter"]; op="word"
13. compare="Equal"; data="0xaa271e1a00000000000000000000000000000000000000000000000000000000c0ffee20"; expect="call"; is="0x0000000000000000000000000000000000000000000000000000000000000000"; to="0x0000000000000000000000000000000000001000"

### masterminter-member-add-remove — tests/tc/go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/14-masterminter-member-add-remove.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="newAccount"; save="member"; saveKey="memberKey"
2. do="read"; of=["$member","3"]; op="abiCall"; save="addData"; selector="0x5c646aa6"; source="derive"
3. data="$addData"; do="sendTx"; from="node1"; gas="0x16e360"; on="node1"; save="addHash"; to="0x0000000000000000000000000000000000001002"
4. do="read"; hash="$addHash"; save="addPid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
5. do="read"; of=["$addPid"]; op="abiCall"; save="addApprove"; selector="0x98951b56"; source="derive"
6. data="$addApprove"; do="sendTx"; from="node2"; gas="0x16e360"; on="node2"; save="addApHash"; to="0x0000000000000000000000000000000000001002"
7. data="0x6ad89315"; do="read"; save="ver1"; source="call"; to="0x0000000000000000000000000000000000001002"
8. do="read"; of=["$member","$ver1"]; op="abiCall"; save="isMemData1"; selector="0x85752d03"; source="derive"
9. data="$isMemData1"; do="read"; save="isMem1"; source="call"; to="0x0000000000000000000000000000000000001002"
10. data="0x1703a018"; do="read"; save="q1"; source="call"; to="0x0000000000000000000000000000000000001002"
11. do="read"; of=["$member","2"]; op="abiCall"; save="rmData"; selector="0xbfbd7f4c"; source="derive"
12. data="$rmData"; do="sendTx"; from="node1"; gas="0x16e360"; on="node1"; save="rmHash"; to="0x0000000000000000000000000000000000001002"
13. do="read"; hash="$rmHash"; save="rmPid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
14. do="read"; of=["$rmPid"]; op="abiCall"; save="rmApprove"; selector="0x98951b56"; source="derive"
15. data="$rmApprove"; do="sendTx"; from="node2"; gas="0x16e360"; on="node2"; save="rmApHash2"; to="0x0000000000000000000000000000000000001002"
16. data="$rmApprove"; do="sendTx"; from="node3"; gas="0x16e360"; on="node3"; save="rmApHash3"; to="0x0000000000000000000000000000000000001002"
17. data="0x6ad89315"; do="read"; save="ver2"; source="call"; to="0x0000000000000000000000000000000000001002"
18. do="read"; of=["$member","$ver2"]; op="abiCall"; save="isMemData2"; selector="0x85752d03"; source="derive"
19. data="$isMemData2"; do="read"; save="isMem2"; source="call"; to="0x0000000000000000000000000000000000001002"
20. data="0x1703a018"; do="read"; save="q2"; source="call"; to="0x0000000000000000000000000000000000001002"
21. expect="txStatus"; hash="$addApHash"; is="0x1"
22. expect="txStatus"; hash="$rmApHash2"; is="0x1"
23. expect="txStatus"; hash="$rmApHash3"; is="0x1"
24. compare="Equal"; expect="derive"; index=0; is="0x0000000000000000000000000000000000000000000000000000000000000001"; of=["$isMem1"]; op="word"
25. compare="Equal"; expect="derive"; index=0; is="0x0000000000000000000000000000000000000000000000000000000000000003"; of=["$q1"]; op="word"
26. compare="Equal"; expect="derive"; index=0; is="0x0000000000000000000000000000000000000000000000000000000000000000"; of=["$isMem2"]; op="word"
27. compare="Equal"; expect="derive"; index=0; is="0x0000000000000000000000000000000000000000000000000000000000000002"; of=["$q2"]; op="word"

### non-member-configure-minter-rejected — tests/tc/go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/15-non-member-configure-minter-rejected.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="newAccount"; save="acct"; saveKey="acctKey"
2. do="sendTx"; from="node1"; gas="21000"; on="node1"; save="fundHash"; to="$acct"; value="10000000000000000000"
3. data="0x898420a900000000000000000000000000000000000000000000000000000000c0ffee410000000000000000000000000000000000000000000000008ac7230489e80000"; do="sendTx"; expect="reject"; key="$acctKey"; on="en1"; to="0x0000000000000000000000000000000000001002"
4. expect="txStatus"; hash="$fundHash"; is="0x1"

### blacklist-proposal-executes — tests/tc/go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/16-blacklist-proposal-executes.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. data="0x0d32127300000000000000000000000000000000000000000000000000000000c0ffee06"; do="sendTx"; from="node1"; gas="0x16e360"; on="node1"; save="propHash"; to="0x0000000000000000000000000000000000001004"
2. do="read"; hash="$propHash"; save="pid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
3. do="read"; of=["$pid"]; op="abiCall"; save="approveData"; selector="0x98951b56"; source="derive"
4. data="$approveData"; do="sendTx"; from="node2"; gas="0x16e360"; on="node2"; save="apHash"; to="0x0000000000000000000000000000000000001004"
5. do="read"; method="eth_getTransactionReceipt"; params=["$apHash"]; save="apBlk"; select="blockNumber"; source="rpcCall"
6. expect="txStatus"; hash="$propHash"; is="0x1"
7. expect="txStatus"; hash="$apHash"; is="0x1"
8. compare="Equal"; data="0xfe575a8700000000000000000000000000000000000000000000000000000000c0ffee06"; expect="call"; is="0x0000000000000000000000000000000000000000000000000000000000000001"; to="0x0000000000000000000000000000000000B00003"
9. address="0x0000000000000000000000000000000000001004"; compare="GreaterOrEqual"; expect="logs"; fromBlock="$apBlk"; is="1"; select="count"; toBlock="$apBlk"; topics=["0x1d2c7e6b911a4ad6c07d780ccd7e533749bff42c2b1d161c21f3fc7c830eb2cc","0x00000000000000000000000000000000000000000000000000000000c0ffee06"]

### authorize-proposal-executes — tests/tc/go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/18-authorize-proposal-executes.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. data="0x93a8bb9900000000000000000000000000000000000000000000000000000000c0ffee0a"; do="sendTx"; from="node1"; gas="0x16e360"; on="node1"; save="propHash"; to="0x0000000000000000000000000000000000001004"
2. do="read"; hash="$propHash"; save="pid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
3. do="read"; of=["$pid"]; op="abiCall"; save="approveData"; selector="0x98951b56"; source="derive"
4. data="$approveData"; do="sendTx"; from="node2"; gas="0x16e360"; on="node2"; save="apHash"; to="0x0000000000000000000000000000000000001004"
5. expect="txStatus"; hash="$propHash"; is="0x1"
6. expect="txStatus"; hash="$apHash"; is="0x1"
7. compare="Equal"; data="0xfe9fbb8000000000000000000000000000000000000000000000000000000000c0ffee0a"; expect="call"; is="0x0000000000000000000000000000000000000000000000000000000000000001"; to="0x0000000000000000000000000000000000B00003"

### unauthorize-proposal-executes — tests/tc/go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/19-unauthorize-proposal-executes.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. data="0x93a8bb9900000000000000000000000000000000000000000000000000000000c0ffee12"; do="sendTx"; from="node1"; gas="0x16e360"; on="node1"; save="addHash"; to="0x0000000000000000000000000000000000001004"
2. do="read"; hash="$addHash"; save="addPid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
3. do="read"; of=["$addPid"]; op="abiCall"; save="addApprove"; selector="0x98951b56"; source="derive"
4. data="$addApprove"; do="sendTx"; from="node2"; gas="0x16e360"; on="node2"; save="addApHash"; to="0x0000000000000000000000000000000000001004"
5. data="0xcf44550e00000000000000000000000000000000000000000000000000000000c0ffee12"; do="sendTx"; from="node1"; gas="0x16e360"; on="node1"; save="rmHash"; to="0x0000000000000000000000000000000000001004"
6. do="read"; hash="$rmHash"; save="rmPid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
7. do="read"; of=["$rmPid"]; op="abiCall"; save="rmApprove"; selector="0x98951b56"; source="derive"
8. data="$rmApprove"; do="sendTx"; from="node2"; gas="0x16e360"; on="node2"; save="rmApHash"; to="0x0000000000000000000000000000000000001004"
9. do="read"; method="eth_getTransactionReceipt"; params=["$rmApHash"]; save="rmApBlk"; select="blockNumber"; source="rpcCall"
10. expect="txStatus"; hash="$addApHash"; is="0x1"
11. expect="txStatus"; hash="$rmApHash"; is="0x1"
12. compare="Equal"; data="0xfe9fbb8000000000000000000000000000000000000000000000000000000000c0ffee12"; expect="call"; is="0x0000000000000000000000000000000000000000000000000000000000000000"; to="0x0000000000000000000000000000000000B00003"
13. address="0x0000000000000000000000000000000000001004"; compare="GreaterOrEqual"; expect="logs"; fromBlock="$rmApBlk"; is="1"; select="count"; toBlock="$rmApBlk"; topics=["0x50481a3a74ecdf57138218d3d03198c96d338a47f85893e68628c4689d715347","0x00000000000000000000000000000000000000000000000000000000c0ffee12"]

### direct-blacklist-call-rejected — tests/tc/go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/20-direct-blacklist-call-rejected.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="newAccount"; save="acct"; saveKey="acctKey"
2. do="sendTx"; from="node1"; gas="21000"; on="node1"; save="fundHash"; to="$acct"; value="10000000000000000000"
3. data="0xf9f92be400000000000000000000000000000000000000000000000000000000c0ffee40"; do="sendTx"; expect="reject"; key="$acctKey"; on="en1"; to="0x0000000000000000000000000000000000B00003"
4. expect="txStatus"; hash="$fundHash"; is="0x1"

### authorized-account-added-event — tests/tc/go-stablenet/regression/system-contracts/23-authorized-account-added-event.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/23-authorized-account-added-event.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. data="0x93a8bb9900000000000000000000000000000000000000000000000000000000c0ffee11"; do="sendTx"; from="node1"; gas="0x16e360"; on="node1"; save="propHash"; to="0x0000000000000000000000000000000000001004"
2. do="read"; hash="$propHash"; save="pid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
3. do="read"; of=["$pid"]; op="abiCall"; save="approveData"; selector="0x98951b56"; source="derive"
4. data="$approveData"; do="sendTx"; from="node2"; gas="0x16e360"; on="node2"; save="apHash"; to="0x0000000000000000000000000000000000001004"
5. do="read"; method="eth_getTransactionReceipt"; params=["$apHash"]; save="apBlk"; select="blockNumber"; source="rpcCall"
6. expect="txStatus"; hash="$propHash"; is="0x1"
7. expect="txStatus"; hash="$apHash"; is="0x1"
8. compare="Equal"; data="0xfe9fbb8000000000000000000000000000000000000000000000000000000000c0ffee11"; expect="call"; is="0x0000000000000000000000000000000000000000000000000000000000000001"; to="0x0000000000000000000000000000000000B00003"
9. address="0x0000000000000000000000000000000000001004"; compare="GreaterOrEqual"; expect="logs"; fromBlock="$apBlk"; is="1"; select="count"; toBlock="$apBlk"; topics=["0x3110d1ee06428a72c2523738b03b74d2cac172112945536944ef0d34f9175947","0x00000000000000000000000000000000000000000000000000000000c0ffee11"]

### token-metadata — tests/tc/go-stablenet/regression/system-contracts/25-token-metadata.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/25-token-metadata.json:30`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="Equal"; data="0x06fdde03"; expect="call"; is=hex payload (96 bytes; 원문 참조); to="0x0000000000000000000000000000000000001000"
2. compare="Equal"; data="0x95d89b41"; expect="call"; is=hex payload (96 bytes; 원문 참조); to="0x0000000000000000000000000000000000001000"

### minter-status-readable — tests/tc/go-stablenet/regression/system-contracts/28-minter-status-readable.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/system-contracts/28-minter-status-readable.json:30`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="NotEqual"; data="0xaa271e1a000000000000000000000000c17d493883eaa3b4cceb0f214b273392d562f9d8"; expect="call"; is="0x"; to="0x0000000000000000000000000000000000001000"
2. compare="NotEqual"; data="0x8a6db9c3000000000000000000000000c17d493883eaa3b4cceb0f214b273392d562f9d8"; expect="call"; is="0x"; to="0x0000000000000000000000000000000000001000"

### block-period-one-second — tests/tc/go-stablenet/regression/wbft/01-block-period-one-second.json

- `P2` / `adapt` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/wbft/01-block-period-one-second.json:31`
- 적용: env.chain=wemix와 gwemix 바이너리, PoA genesis/계정/수수료를 준비하고 applicableChains에 wemix를 포함한다. WBFT 1초 상수를 PoA blockCreationTime/idle 정책에 맞는 기대값으로 바꾼다. parentHash 연결로 시각을 읽는 공통 검증을 재사용한다. 실제 생산 지연을 허용하는 시각 하한·관측 범위를 정하며 모든 블록 간격의 정확한 등식을 강제하지 않는다.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`, `sources/recheck-chainbench/internal/testengine/capability.go:16`

1. do="read"; method="eth_blockNumber"; save="n"; source="rpcCall"
2. do="read"; method="eth_getBlockByNumber"; params=["$n",false]; save="t1"; select="timestamp"; source="rpcCall"
3. do="read"; method="eth_getBlockByNumber"; params=["$n",false]; save="ph1"; select="parentHash"; source="rpcCall"
4. do="read"; method="eth_getBlockByHash"; params=["$ph1",false]; save="t0"; select="timestamp"; source="rpcCall"
5. do="read"; method="eth_getBlockByHash"; params=["$ph1",false]; save="ph0"; select="parentHash"; source="rpcCall"
6. do="read"; method="eth_getBlockByHash"; params=["$ph0",false]; save="tm1"; select="timestamp"; source="rpcCall"
7. compare="Equal"; expect="derive"; is="1"; of=["$t1","$t0"]; op="diff"
8. compare="Equal"; expect="derive"; is="1"; of=["$t0","$tm1"]; op="diff"

### wbft-seals-quorum — tests/tc/go-stablenet/regression/wbft/02-wbft-seals-quorum.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/wbft/02-wbft-seals-quorum.json:33`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="read"; method="eth_blockNumber"; save="n"; source="rpcCall"
2. compare="NotEqual"; expect="rpcCall"; is="0x"; method="istanbul_getWbftExtraInfo"; params=["$n"]; select="committedSeal.signature"
3. compare="NotEqual"; expect="rpcCall"; is="0x"; method="istanbul_getWbftExtraInfo"; params=["$n"]; select="preparedSeal.signature"

### epoch-transition-carries-epoch-info — tests/tc/go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/wbft/03-epoch-transition-carries-epoch-info.json:33`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; pollInterval="2s"; target=140; timeout="240s"
2. compare="GreaterOrEqual"; expect="rpcCall"; is="1"; method="istanbul_getWbftExtraInfo"; params=["0x8c"]; select="epochInfo.validators.#"

### validator-add-member-executes — tests/tc/go-stablenet/regression/wbft/04-validator-add-member-executes.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/wbft/04-validator-add-member-executes.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="read"; of=["0x5400d8b543eaf6738c7b44799623bea88fd0f5ee","4"]; op="abiCall"; save="propData"; selector="0x5c646aa6"; source="derive"
2. data="$propData"; do="sendTx"; from="node1"; gas="0x16e360"; on="node1"; save="propHash"; to="0x0000000000000000000000000000000000001001"
3. do="read"; hash="$propHash"; save="pid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
4. do="read"; of=["$pid"]; op="abiCall"; save="approveData"; selector="0x98951b56"; source="derive"
5. data="$approveData"; do="sendTx"; from="node2"; gas="0x16e360"; on="node2"; save="apHash"; to="0x0000000000000000000000000000000000001001"
6. data="0x08ae4b0c0000000000000000000000005400d8b543eaf6738c7b44799623bea88fd0f5ee"; do="read"; save="membersRet"; source="call"; to="0x0000000000000000000000000000000000001001"
7. expect="txStatus"; hash="$propHash"; is="0x1"
8. expect="txStatus"; hash="$apHash"; is="0x1"
9. compare="Equal"; expect="derive"; index=0; is="0x0000000000000000000000000000000000000000000000000000000000000001"; of=["$membersRet"]; op="word"

### validator-add-member-epoch-activates — tests/tc/go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/wbft/04b-validator-add-member-epoch-activates.json:40`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; target=3; timeout="120s"
2. do="read"; on="en1"; source="rpcCall"; method="istanbul_nodeAddress"; params=[]; save="candidate"
3. do="read"; on="node1"; source="rpcCall"; method="istanbul_getValidators"; params=["latest"]; select="#"; save="countBefore"
4. expect="rpc"; on="en1"; method="istanbul_isValidator"; params=[]; compare="Equal"; is=false
5. do="read"; source="derive"; op="abiCall"; selector="0x5c646aa6"; of=["$candidate","4"]; save="addData"
6. do="sendTx"; on="node1"; from="node1"; to="0x0000000000000000000000000000000000001001"; gas="0x16e360"; data="$addData"; save="addHash"; expect="receipt"
7. do="read"; source="receiptLog"; hash="$addHash"; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"; topic=1; save="pid"
8. do="read"; source="derive"; op="abiCall"; selector="0x98951b56"; of=["$pid"]; save="approveData"
9. do="sendTx"; on="node2"; from="node2"; to="0x0000000000000000000000000000000000001001"; gas="0x16e360"; data="$approveData"; save="apHash"; expect="receipt"
10. do="waitBlock"; target=25; timeout="180s"
11. do="read"; source="derive"; op="sum"; of=["$countBefore","1"]; save="countExpected"
12. expect="rpc"; on="node1"; method="istanbul_getValidators"; params=["latest"]; select="#"; compare="Equal"; is="$countExpected"
13. expect="rpc"; on="en1"; method="istanbul_isValidator"; params=[]; compare="Equal"; is=true

### validator-remove-member-executes — tests/tc/go-stablenet/regression/wbft/05-validator-remove-member-executes.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/wbft/05-validator-remove-member-executes.json:31`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="read"; of=["0x5400d8b543eaf6738c7b44799623bea88fd0f5ee","4"]; op="abiCall"; save="addData"; selector="0x5c646aa6"; source="derive"
2. data="$addData"; do="sendTx"; from="node1"; gas="0x16e360"; on="node1"; save="addHash"; to="0x0000000000000000000000000000000000001001"
3. do="read"; hash="$addHash"; save="addPid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
4. do="read"; of=["$addPid"]; op="abiCall"; save="addApprove"; selector="0x98951b56"; source="derive"
5. data="$addApprove"; do="sendTx"; from="node2"; gas="0x16e360"; on="node2"; save="addApHash"; to="0x0000000000000000000000000000000000001001"
6. data="0x08ae4b0c0000000000000000000000005400d8b543eaf6738c7b44799623bea88fd0f5ee"; do="read"; save="memberAfterAdd"; source="call"; to="0x0000000000000000000000000000000000001001"
7. do="read"; of=["0x5400d8b543eaf6738c7b44799623bea88fd0f5ee","4"]; op="abiCall"; save="rmData"; selector="0xbfbd7f4c"; source="derive"
8. data="$rmData"; do="sendTx"; from="node1"; gas="0x16e360"; on="node1"; save="rmHash"; to="0x0000000000000000000000000000000000001001"
9. do="read"; hash="$rmHash"; save="rmPid"; source="receiptLog"; topic=1; topic0="0x830652010a654c24b39890c16f53e6f6179becc61702ecd9a8c88461c2ff941a"
10. do="read"; of=["$rmPid"]; op="abiCall"; save="rmApprove"; selector="0x98951b56"; source="derive"
11. data="$rmApprove"; do="sendTx"; from="node2"; gas="0x16e360"; on="node2"; save="rmApHash"; to="0x0000000000000000000000000000000000001001"
12. data="0x08ae4b0c0000000000000000000000005400d8b543eaf6738c7b44799623bea88fd0f5ee"; do="read"; save="memberAfterRemove"; source="call"; to="0x0000000000000000000000000000000000001001"
13. expect="txStatus"; hash="$addApHash"; is="0x1"
14. expect="txStatus"; hash="$rmHash"; is="0x1"
15. expect="txStatus"; hash="$rmApHash"; is="0x1"
16. compare="Equal"; expect="derive"; index=0; is="0x0000000000000000000000000000000000000000000000000000000000000001"; of=["$memberAfterAdd"]; op="word"
17. compare="Equal"; expect="derive"; index=0; is="0x0000000000000000000000000000000000000000000000000000000000000000"; of=["$memberAfterRemove"]; op="word"

### prev-seals-quorum — tests/tc/go-stablenet/regression/wbft/11-prev-seals-quorum.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/wbft/11-prev-seals-quorum.json:33`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="waitBlock"; pollInterval="1s"; target=3; timeout="60s"
2. do="read"; of=["4"]; op="quorum"; save="q"; source="derive"
3. compare="GreaterOrEqual"; expect="rpcCall"; is="$q"; method="istanbul_getWbftExtraInfo"; params=["@latest"]; select="prevCommittedSeal.sealers.#"
4. compare="GreaterOrEqual"; expect="rpcCall"; is="$q"; method="istanbul_getWbftExtraInfo"; params=["@latest"]; select="prevPreparedSeal.sealers.#"

### randao-and-mixdigest-present — tests/tc/go-stablenet/regression/wbft/13-randao-and-mixdigest-present.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/wbft/13-randao-and-mixdigest-present.json:30`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="read"; method="eth_blockNumber"; save="n"; source="rpcCall"
2. compare="NotEqual"; expect="rpcCall"; is="0x"; method="istanbul_getWbftExtraInfo"; params=["$n"]; select="randaoReveal"
3. compare="NotEqual"; expect="rpcCall"; is="0x0000000000000000000000000000000000000000000000000000000000000000"; method="eth_getBlockByNumber"; params=["$n",false]; select="mixHash"

### stablenet-gastip-field — tests/tc/go-stablenet/regression/wbft/14-stablenet-gastip-field.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-stablenet/regression/wbft/14-stablenet-gastip-field.json:30`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. do="read"; method="eth_blockNumber"; save="head"; source="rpcCall"
2. compare="NotNil"; expect="rpcCall"; is=null; method="istanbul_getWbftExtraInfo"; params=["$head"]; select="gasTip"

### secp256r1-precompile-valid — tests/tc/go-wbft/accounts/01-secp256r1-precompile-valid.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-wbft/accounts/01-secp256r1-precompile-valid.json:30`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="Equal"; data=hex payload (160 bytes; 원문 참조); expect="call"; is="0x0000000000000000000000000000000000000000000000000000000000000001"; to="0x0000000000000000000000000000000000000100"

### secp256r1-precompile-invalid — tests/tc/go-wbft/accounts/02-secp256r1-precompile-invalid.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-wbft/accounts/02-secp256r1-precompile-invalid.json:30`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="NotEqual"; data=hex payload (160 bytes; 원문 참조); expect="call"; is="0x0000000000000000000000000000000000000000000000000000000000000001"; to="0x0000000000000000000000000000000000000100"

### secp256r1-precompile-short-input — tests/tc/go-wbft/accounts/03-secp256r1-precompile-short-input.json

- `P3` / `excluded` / implemented; execution_unverified
- 원문 steps: `sources/chainbench/tests/tc/go-wbft/accounts/03-secp256r1-precompile-short-input.json:30`
- 적용: PoA 회귀 테스트로 그대로 사용 불가. 필요하면 go-wemix 계약/합의 명세로 별도 작성.
- 검증 한계: 명시된 do/expect 범위만 구현됨. PR196의 RLP 블록 크기·전파 크기 경계 assertions 없음.
- 코드 근거: `sources/pr-head/params/protocol_params.go:129`, `sources/pr-head/miner/worker.go:147`, `sources/pr-head/core/types/transaction.go:45`, `sources/chainbench/internal/chains/wemix/wemix.go:34`

1. compare="Equal"; data=hex payload (159 bytes; 원문 참조); expect="call"; is="0x"; to="0x0000000000000000000000000000000000000100"
2. compare="Equal"; data="0x"; expect="call"; is="0x"; to="0x0000000000000000000000000000000000000100"

## 테스트 외 문서

| 파일 | 역할 |
|---|---|
| [README.md](../sources/chainbench/tests/tc/README.md) | 디렉터리·실행 안내. 독립 실행 케이스 아님. |
| [SPECS.md](../sources/chainbench/tests/tc/SPECS.md) | 케이스 작성 규격. 독립 실행 케이스 아님. |
| [CHAIN-BRINGUP.md](../sources/chainbench/tests/tc/CHAIN-BRINGUP.md) | 체인 기동 환경/절차 안내. 독립 실행 케이스 아님. |
