# 추가 테스트 16건 독립 재검토

기준은 보존된 PR head 소스다. 아래 경로는 실행 산출물 루트의 `sources/pr-head/` 기준이며, 테스트 실행이나 프로젝트 소스 수정 없이 정적으로 대조했다. 제안된 테스트에서 실패했다고 해서 PR 회귀로 곧바로 판정할 수 없도록 입력 전제와 기대 결과를 검토했다.

## 항목별 판정

| ID | 판정 | 결론과 근거 |
|---|---|---|
| N-001 | 적용 범위 명확화 | 8 MiB 및 strict `>` 자체는 정확하다. known/ancestor보다 먼저라는 순서는 ValidateBody 안에서만 성립한다. InsertChain은 header 검증 오류를 먼저 반환할 수 있다. `sources/pr-head/core/block_validator.go:53`, `sources/pr-head/core/blockchain_insert.go:119` |
| N-002 | 기대 결과 수정 필요 | size로 제외된 tx도 별도 prefetch state에서 실행될 수 있다. 정식 environment의 state/nonce/receipt 및 committed 표식이 바뀌지 않는다는 조건으로 한정해야 한다. `sources/pr-head/miner/worker.go:1072`, `sources/pr-head/miner/tx_prefetch.go:65` |
| N-003 | 오류 확인 안 됨 | 최종 seal 후 크기와 별도 import 검증이 필요하다. 직접 WriteBlockAndSetHead하는 생산 경로와 수신 검증 경로를 구분한 접근이 타당하다. `sources/pr-head/miner/worker.go:1751`, `sources/pr-head/miner/worker.go:1818` |
| N-004 | 오류 확인 안 됨 | historical fixture로 경로 차이를 재현한다는 제안은 타당하다. 실제 과거 oversized 존재를 미확인으로 남긴 점도 맞다. 정책 확정 전 곧바로 취약점으로 판정하면 안 된다. `sources/pr-head/core/block_validator.go:53`, `sources/pr-head/core/blockchain.go:917` |
| N-005 | 오류 확인 안 됨 | 새 무조건 body cap의 혼합 배포 검증은 P0 유지가 타당하다. base의 100 MiB는 ETH 핸들러 상한이며 그 전체 구간이 실제 전송 가능한 것은 아니다. `sources/pr-head/core/block_validator.go:53`, `sources/pr-head/eth/protocols/eth/protocol.go:51`, `sources/pr-head/p2p/rlpx/rlpx.go:152` |
| N-006 | 측정 대상 수정 필요 | 10 MiB는 암호화·압축 전후 네트워크 wire 전체가 아닌 ETH message payload의 msg.Size 상한이다. 65/66/68은 실제 지원된다. `sources/pr-head/eth/protocols/eth/handler.go:250`, `sources/pr-head/eth/protocols/eth/protocol.go:44`, `sources/pr-head/p2p/rlpx/rlpx.go:143` |
| N-007 | 오류 확인 안 됨 | size break 이후 정상 commit/token 반환/DB write 경로 검증은 타당하다. 종료·TTL 변형은 기존 장애 정책과 비교하는 보조 조건이며 token 버그 존재를 뜻하지 않는다. `sources/pr-head/miner/worker.go:1812`, `sources/pr-head/miner/worker.go:1818`, `sources/pr-head/miner/worker.go:1829` |
| N-008 | 입력 전제 수정 필요 | pool에 무조건 남고 교체된 최종 hash만 포함된다는 보장은 없다. 용량·유효성·TTL·교체 수락 시점·새 작업 생성 시점을 통제해야 한다. `sources/pr-head/core/tx_pool.go:780`, `sources/pr-head/core/tx_pool.go:801`, `sources/pr-head/core/tx_pool.go:1271` |
| N-009 | 기대 결과 수정 필요 | size 검사가 실행 유효성 검사보다 앞선다. cap에 걸린 무효 후보 뒤의 정상 후보도 현재 break 정책상 검사하지 않는다. ApplyTransaction error 시 state rollback과 gasPool rollback을 혼동하면 안 된다. `sources/pr-head/miner/worker.go:974`, `sources/pr-head/miner/worker.go:920`, `sources/pr-head/core/state_transition.go:222`, `sources/pr-head/core/state_transition.go:350` |
| N-010 | 기대 결과 범위 명확화 | tx 추가 후 size/state 불변 검사는 타당하다. receipt 전체를 깊게 복제한다는 의미로 확장하면 틀린다. copyReceipts는 구조체 값만 복사하여 Logs 등 내부 참조를 공유한다. `sources/pr-head/miner/worker.go:111`, `sources/pr-head/miner/worker.go:1877` |
| N-011 | 오류 확인 안 됨 | typed tx의 Size와 outer RLP 길이 일치를 요구하지 않는 것은 올바르다. 활성 fork와 실제 txpool 허용 크기를 별도로 맞추어야 한다. `sources/pr-head/core/types/transaction.go:101`, `sources/pr-head/core/types/transaction.go:208`, `sources/pr-head/core/types/transaction.go:430` |
| N-012 | 프로토콜 fixture 수정 필요 | ETH65에는 request ID envelope가 없다. 65는 raw body list, 66/68은 request ID wrapper로 따로 인코딩해야 한다. 측정값은 ETH payload다. `sources/pr-head/eth/protocols/eth/peer.go:346`, `sources/pr-head/eth/protocols/eth/peer.go:351` |
| N-013 | 오류 확인 안 됨 | receipt 크기는 body cap만으로 제한되지 않고 응답 마지막 항목은 soft cap을 넘길 수 있다. gas 설정상 도달 가능 여부를 조건부로 둔 제안은 타당하다. `sources/pr-head/eth/protocols/eth/handlers.go:470` |
| N-014 | 오류 확인 안 됨 | 실행 import 경로의 거부·회복을 검증하고 snap receipt 경로와 구분해야 한다. 기존 문서가 '실행 import'로 한정한 점은 타당하다. `sources/pr-head/core/blockchain_insert.go:124`, `sources/pr-head/eth/downloader/downloader.go:1548` |
| N-015 | 오류 확인 안 됨 | 두 루프의 기존 정책 차이를 보존하며 size와 다른 종료 조건을 조합한다는 제안은 타당하다. Simple에는 MaxTxsPerBlock 검사가 없다. `sources/pr-head/miner/worker.go:978`, `sources/pr-head/miner/worker.go:1112` |
| N-016 | 해석 범위 명확화 | 'decode 전'은 ETH packet RLP decode 전이다. transport read/decompression은 이미 수행된다. RSS가 원래 값으로 반드시 복귀한다는 식으로 해석하지 말고 제한된 정상 범위에서 안정화하는지 확인해야 한다. `sources/pr-head/eth/protocols/eth/handler.go:250`, `sources/pr-head/p2p/rlpx/rlpx.go:155` |

## 반드시 교정할 내용

### N-002: prefetch 실행과 정식 블록 실행 구분

PrefetchCount=0은 가격·nonce 루프, >0은 Simple 루프라는 분기는 정확하다(`sources/pr-head/miner/worker.go:1262`, `sources/pr-head/miner/worker.go:1284`). 그러나 Simple은 크기 검사 전에 prefetch worker를 시작한다(`sources/pr-head/miner/worker.go:1072`). prefetch는 별도로 `StateAt`에서 얻은 state로 `ApplyMessage`를 수행하고 `doneTxs` 캐시에 표시한다(`sources/pr-head/miner/tx_prefetch.go:28`, `sources/pr-head/miner/tx_prefetch.go:41`, `sources/pr-head/miner/tx_prefetch.go:65`). 따라서 '실행 자체가 없어야 한다'는 oracle은 오탐을 만든다. '정식 env의 실행 결과에 포함되지 않으며 env nonce/txs/receipts/tcount/size 및 committedTxs에 반영되지 않는다. prefetch의 별도 실행·캐시 흔적은 허용한다'로 고친다.

### N-008: pool 비손실은 통제된 조건에서 검증

크기 중단 자체는 pool에서 tx를 삭제하지 않지만, pool에는 용량 초과 시 underpriced tx 삭제(`sources/pr-head/core/tx_pool.go:780`), queued TTL 삭제(`sources/pr-head/core/tx_pool.go:417`), 새 head 후 무효 tx 정리(`sources/pr-head/core/tx_pool.go:1271`), pending/queue 한도 정리(`sources/pr-head/core/tx_pool.go:1287`)가 있다. 제출된 tx가 매번 포함될 때까지 pool에 남는다는 무조건 보장은 아니다.

충분한 pool capacity·잔액·gas·fee cap, 유효 fork, eviction 없는 관측 시간, 외부 경쟁 tx 차단을 전제로 한다. 새 head 후 pool reset이 완료된 뒤 미포함 tx를 확인한다. 교체는 PriceBump 조건을 충족해 수락됐는지 확인하고, 기존 후보를 이미 읽은 miner 작업을 종료하거나 새 작업 생성 이전에 교체를 완료해야 '교체 hash만 포함'을 요구할 수 있다(`sources/pr-head/core/tx_pool.go:801`, `sources/pr-head/miner/worker.go:1256`). miner 변경 시 수신 노드가 tx를 보유하거나 재전송을 완료했다는 전제도 필요하다.

### N-009: size-first 중단과 기존 gas 회계 보존

size gate는 tx 검증보다 먼저다(`sources/pr-head/miner/worker.go:974`, `sources/pr-head/miner/worker.go:1107`). cap을 넘는 무효 tx도 즉시 break하며 이후 정상 tx를 보지 않는다. 이를 회귀 실패로 단정하면 현재 의도된 strict packing 정책을 버그로 검출한다. 두 fixture를 분리한다.

1. 크기 gate를 통과하는 무효 tx: size/txs/receipts/tcount는 증가하지 않으며 기존 Pop/Shift 정책대로 진행한다.
2. 크기 gate에 걸리는 무효 tx: 실제 유효성 검사 전에 중단되는 현 정책과 base/head 차이를 기록한다. 더 작은 후속 tx를 찾아야 한다는 요구는 별도 정책 변경이다.

StateDB는 오류 시 snapshot으로 되돌리지만(`sources/pr-head/miner/worker.go:925`) 별도 gasPool을 일반적으로 되돌리는 코드는 없다. `buyGas` 이후 intrinsic-gas 오류가 나면 gasPool을 이미 차감했을 수 있다(`sources/pr-head/core/state_transition.go:222`, `sources/pr-head/core/state_transition.go:345`). 'state rollback'을 gasPool·header·모든 객체 rollback으로 확장하지 않고 오류 종류별 base/head의 기존 결과를 비교한다. 유효 EVM REVERT/OOG의 실패 receipt·nonce 반영 제안은 유지한다.

### N-006 / N-012 / N-016: ETH payload와 전송 계층 분리

상한은 `msg.Size`에 적용된다. RLPx는 frame 처리와 Snappy 압축 해제를 먼저 수행하며, 그 과정에는 별도의 24-bit 상한이 있다(`sources/pr-head/p2p/rlpx/rlpx.go:135`, `sources/pr-head/p2p/rlpx/rlpx.go:152`). 따라서 '실제 wire 길이 10 MiB'를 시험 기준으로 삼으면 잘못된 경계 시험이 된다. ETH packet RLP payload 길이를 10 MiB±1로 맞추고 transport overhead 및 압축 길이는 참고 측정으로 분리한다.

N-006의 10~100 MiB 비교는 핸들러에 직접 주입할 때의 size gate 비교다. 실제 peer 통합 시험은 전송 계층이 허용하는 구간에서 한다. N-012는 ETH65 raw list와 ETH66/68 request ID wrapper를 구분한다. N-016은 ETH packet decode 전 차단이지, 수신 버퍼 할당이나 압축 해제 이전 방어가 아님을 명시한다.

## 우선순위와 중복

P0/P1/P2의 대규모 재분류가 필요한 근거는 찾지 못했다. N-003/006, N-004/014는 일부 경로를 공유하지만 각각 생산 후 최종 크기/프로토콜 경계, historical 호환성/오류 후 진행 회복이라는 별도 기대 결과가 있다. 같은 fixture를 재사용할 수는 있으나 동일 테스트로 삭제할 근거는 부족하다. N-010의 깊은 복사 범위 확장이나 N-009의 기존 gas 회계 변경은 PR196 필수 회귀 테스트의 요구에 추가하지 않는다.
