# PR196 영향과 검증 공백

분석 기준은 보존된 PR head `5a9355`이다. 코드 인용은 실행 산출물 루트 기준 `sources/pr-head/` 경로를 사용한다. 테스트를 새로 구현하거나 실행하지 않았으며 아래는 정적 근거와 검증 제안이다.

## 변경 범위와 비교 기준

PR base `c8321f` → PR head `5a9355`의 PR diff는 7개 파일이다. 블록 전체 RLP 8,388,608 byte 상한, 생산 중 tx packing 제한, ETH 메시지 100 MiB → 10 MiB 변경 및 테스트 3개가 핵심이다. 주석의 EIP-7934 표기는 이 코드에 실제로 존재하는 규칙을 설명하는 이름이며 외부 EIP의 전체 준수 여부를 검증한 것은 아니다.

로컬 dev HEAD `902f9f`와 PR base master `c8321f`는 분기되었다. merge-base는 `724cfa1a4e6a633955229a031228894efa55417a`, 양쪽 고유 커밋 수는 local 6 / base 15이다. 로컬→PR의 차이를 PR196 변경으로 간주하면 안 된다. 특히 wemix admin/etcdutil/sync 및 worker 주변 기존 차이는 PR196이 새로 수정한 범위가 아니다. 실제 배포 검증은 별도로 확정한 **통합 대상 SHA**에서 다시 수행해야 한다. `.new` 파일은 Go 컴파일 대상이 아니며 과거 BUILD_SOURCE_FILES 문서는 현재 빌드 구성의 증거로 사용하지 않는다.

## 실제 영향을 받는 경로

| 경로 | 코드 근거 | 의미 |
|---|---|---|
| 생산 시작 | `sources/pr-head/miner/worker.go:1627`, `sources/pr-head/miner/worker.go:1669` | WEMIX token 취득 후 `commitTransactionsEx`가 PrefetchCount에 따라 두 루프로 분기한다. |
| 크기 초기화·복사 | `sources/pr-head/miner/worker.go:111`, `sources/pr-head/miner/worker.go:864`; `sources/pr-head/core/types/block.go:187` | `size=header.Size()` 및 copy 시 값 보존. Header.Size는 메모리 근사치로 실제 RLP 바이트 수가 아니다. |
| tx 적용 | `sources/pr-head/miner/worker.go:920` | ApplyTransaction error이면 state rollback, 성공 반환이면 tx/receipt 추가와 size 증가. EVM REVERT receipt는 유효 포함 tx일 수 있으므로 실행 error와 구분해야 한다. |
| 크기 중단 | `sources/pr-head/miner/worker.go:146`, `sources/pr-head/miner/worker.go:975`, `sources/pr-head/miner/worker.go:1108` | `env.size + tx.Size() < 7,388,608`만 허용. 1,000,000은 decimal buffer이며 1 MiB가 아니다. 안 맞는 첫 tx에서 전체 loop를 break한다. |
| 최종 생산·저장 | `sources/pr-head/miner/worker.go:1743`, `sources/pr-head/miner/worker.go:1812`, `sources/pr-head/miner/worker.go:1817` | env.copy → FinalizeAndAssemble → Seal → ReleaseMiningToken → WriteBlockAndSetHead → broadcast. 최종 block.Size를 별도 검사하는 PR 변경은 없다. |
| 실행 import | `sources/pr-head/core/blockchain_insert.go:124`; `sources/pr-head/core/block_validator.go:53` | InsertChain iterator가 ValidateBody 호출. 8 MiB 초과 검사는 known block/ancestor 검사보다 앞이며 fork 활성화 조건이 없다. |
| full downloader | `sources/pr-head/eth/downloader/downloader.go:1548` | InsertChain으로 들어가 body cap 검사와 연결된다. |
| snap receipt 경로 | `sources/pr-head/eth/downloader/downloader.go:1736`, `sources/pr-head/eth/downloader/downloader.go:1748`; `sources/pr-head/core/blockchain.go:917` | pivot 이전 및 pivot block을 InsertReceiptChain으로 삽입. 이 함수에 ValidateBody/MaxBlockSize 검사가 없다. pivot 이후 full 경로와는 구분한다. |
| downloader body 검증 | `sources/pr-head/eth/downloader/queue.go:767` | txRoot/uncleRoot 비교를 확인했으며 이 함수에는 MaxBlockSize 검사가 없다. 실제 snap 수락 여부는 header/pivot/network 조건을 충족한 통합 검증이 필요하다. |
| ETH 수신 | `sources/pr-head/eth/protocols/eth/handler.go:248`, `sources/pr-head/eth/protocols/eth/handler.go:254` | 모든 해당 ETH 메시지에서 decode 전에 >10 MiB를 거부한다. =10 MiB는 크기 관문을 통과한다. |
| 응답 집계 | `sources/pr-head/eth/protocols/eth/handlers.go:354`, `sources/pr-head/eth/protocols/eth/handlers.go:470` | body/receipt 응답은 항목 추가 전에 2 MiB soft limit을 확인한다. 마지막 항목 때문에 soft limit을 넘을 수 있다. |

## 정적 분석으로 확인한 사실과 아직 확인하지 않은 위험

- **확인:** 생산용 size와 실제 block RLP는 같은 개념이 아니다. typed transaction의 Size는 inner 인코딩 또는 decode 캐시를 사용한다(`sources/pr-head/core/types/transaction.go:101`, `sources/pr-head/core/types/transaction.go:208`, `sources/pr-head/core/types/transaction.go:430`). 타입 envelope, block list prefix, finalize 이후 header를 포함한 실제 RLP를 독립 oracle로 써야 한다. 1,000,000 byte buffer가 부족하다는 결론은 아직 내릴 수 없다.
- **확인:** historical block에 대한 활성화 예외가 없다. **미확인:** 실제 네트워크에 기존 8 MiB 초과 블록이 있는지. 있다면 full replay 거부 가능성이 있으므로 배포 전에 확인해야 한다.
- **확인:** full import와 receipt insertion의 검증 경로가 다르다. **미확인:** 실제 snap 모드에서 공격자가 과대블록을 canonical chain으로 강제할 수 있는지. 경로 차이만으로 공격 성공을 확정하지 않는다.
- **확인:** 정상 size break는 오류 return이 아닌 loop 종료 뒤 commitEx로 진행한다. **미확인:** 실제 etcd·seal·TTL 조건에서 다음 높이가 정체하는지. token 누수는 확인된 버그가 아니며 기존 token 코드 변경도 PR196의 변경이 아니다.
- **확인:** body 응답은 누적 <2 MiB인 상태에서 마지막 body 1개를 추가하므로 2 MiB soft limit은 넘을 수 있다. 그러나 각 block <=8 MiB이고 body는 block보다 작으므로 정상 body 원시 합계는 <10 MiB다. overflow가 확인된 것은 아니며 request ID/list wrapper를 포함한 실제 wire 여유를 확인하는 테스트로 한정한다.
- **확인:** receipt/log 크기는 8 MiB block cap으로 직접 제한되지 않는다. **미확인:** 운영 gas 한도에서 10 MiB 응답이 가능한지. 합법 LOG workload와 gas 계산으로 가능성을 먼저 확인한다.
- **확인:** 일반 tx broadcast는 100 KiB 목표 패킷(`sources/pr-head/eth/protocols/eth/broadcast.go:29`) 및 pooled tx 요청은 2 MiB soft limit을 사용한다. 정상 tx 전파가 무조건 10 MiB를 넘는다는 식의 주장은 근거가 없다.

## PR에 추가된 테스트 3개의 범위

| 기존 PR 테스트 | 확인하는 것 | 남은 공백 |
|---|---|---|
| `sources/pr-head/core/block_validator_test.go:407` TestValidateBodyBlockOversized | genesis의 오류가 oversized가 아님, 200 KiB Legacy tx 누적 oversized block 거부 | 경계 -1/= /+1, 유효 정상 block 실제 import, historical/fork/snap, malformed block과 size 오류 우선순위 |
| `sources/pr-head/miner/worker_test.go:693` TestCommitTransactionsBlockSizeLimit | 가격순 루프에서 균일 Legacy tx 크기로 packing 중단 | 실패 tx 회계, typed tx, copy/interrupt, 실제 finalize/seal RLP, pending 이월 |
| `sources/pr-head/miner/worker_test.go:703` TestCommitTransactionsSimpleBlockSizeLimit | Simple 루프에서 동일 조건 packing 중단 | prefetch 생명주기, WEMIX token/liveness, 멀티노드 전파 및 실제 txpool 상태 |

기존 TestGenerateBlockAndImportEthash/Clique, TestRegenerateMiningBlockEthash/Clique, TestEIP2718BlockEncoding, TestTransactionCoding, TestGetBlockBodies65/66/68, TestGetBlockReceipts65/66/68는 재사용 가능한 harness이다. 이름이 존재한다는 사실만으로 새 경계 조건이 커버됐다고 보지 않는다. common 문서의 full/snap/fetcher/downloader/nonce/replacement/revert 및 tc stress/fault는 관련 기능을 제공하지만 PR의 바이트 경계와 새 중단 조건이 명시되어 있지 않다. `stress/02-stress-tx-flood.json`은 fillPercent=30 및 blockAdvance를 확인한다. `fault/06-fault-txpool-leader-change.json`은 receipt를 받은 뒤 노드를 종료하므로 아직 미포함인 tx가 살아남았다는 직접 증거가 아니다.

## P0: 로컬 기존 Go 회귀 테스트 보존 — 신규 테스트 제안과 별개

로컬 `sources/local-head/miner/worker_test.go`의 `TestTimeItTimestampLowerBound`, `TestSkipMiningTokenAcquisitionWhenWorkerStopped`는 PR head에 없다. 이는 dev/master 분기 차이이며 PR196이 삭제한 테스트라고 표현하면 안 된다. 통합 대상 SHA에서 이 두 기존 테스트와 해당 수정의 보존 여부를 먼저 확인하고 실행해야 한다. `timeIt`의 parent timestamp 하한 및 정지 worker의 token 취득 방지는 크기 패치의 검증과 별도 P0 회귀 게이트로 유지한다. 이 항목은 이미 존재하는 Go 테스트이므로 additional-tests.json에 신규 N 항목으로 중복 등록하지 않았다.

## 추가 작업 순서

1. 통합 대상 SHA 확정 및 위 로컬 기존 테스트 보존 확인.
2. N-001~N-007(P0): 정확한 경계, 실제 sealed 블록, historical 정책, 혼합 배포, 메시지 cap, WEMIX token 생존성.
3. N-008~N-015(P1): 이월/회계/copy/typed RLP/응답 집계/경로 회복/기존 한도 상호작용.
4. N-016(P2): 반복 거부 자원 회복.

각 추가 테스트의 재현 입력·oracle·최저 실행 수준·기존 커버리지 공백·패치 전후 기준은 `additional-tests.md` 및 `../additional-tests.json`에 별도로 정리했다.
