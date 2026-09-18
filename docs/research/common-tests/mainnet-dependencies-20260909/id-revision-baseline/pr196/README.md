# go-wemix PR #196 회귀 테스트 검토

**원본 263개 항목을 모두 목록화했고, go-wemix 수행 후보 144개를 중요도 순으로 추렸다. 기존 목록에 없는 필수 추가 검증은 16개로 따로 정리했다.** 원본 항목은 문서의 시나리오 행 또는 JSON 파일 단위다. 서로 겹치는 기능이 있으므로 독립된 테스트 실행 263회 또는 144회라는 뜻은 아니다.

코드와 테스트는 변경하거나 실행하지 않았다. AST 분석 도구와 문서 검증만 실행했다. 이 보고서의 `direct`와 `implemented`는 실행 성공을 의미하지 않는다.

재검토에서 후보 누락·이식 조건·기대값을 교정했다. [변경 내역과 검증 범위](recheck-summary.md)를 함께 확인한다.

## 먼저 읽을 문서

| 문서 | 내용 |
|---|---|
| [전체 테스트 목록](01-all-tests.md) | Common 45행, Dual 29행, JSON 189개 전량 |
| [go-wemix 수행 후보와 우선순위](02-wemix-priority.md) | P0 → P1 → P2 → P3, 각 항목의 준비·변경 조건 |
| [필수 추가 테스트](additional-tests.md) | 신규 제안 16개. 입력, 기대 결과, 구현 수준, 기존 공백 |
| [PR 영향 분석](pr-impact.md) | 실제 호출 경로, PR 기존 테스트 3개, 로컬 기존 테스트 보존 |
| [AST 그래프](03-code-graph.md) / [다이어그램](04-diagrams.md) | 두 코드 기준의 그래프와 변경 경로 |
| [패치 후 실행 순서](05-execution-plan.md) | 통합 SHA 확인부터 실행·판정·증거 기록까지 |
| [문서와 JSON 교차참조](source-crosswalk.md) | 같은 이름의 테스트가 실제로 같은 조건을 검증하는지 |
| [검증 결과](verification.txt) / [재검토 절차](../REPLAY.md) | 누락·인용·모델·그림 검사와 최신 코드 재분석 방법 |

## 전체 목록에서 추린 결과

| 입력 | 전체 | 수행 후보 | 제외 |
|---|---:|---:|---:|
| common_test_scenarios.md | 45행 | 45행 | 0 |
| dual_chain_test_scenarios.md | 29행 | 2행 | 27행 |
| tests/tc JSON | 189개 | 97개 | 92개 |
| 합계 | 263개 항목 | 144개 항목 | 119개 항목 |

수행 후보 144개는 **그대로 실행 후보 JSON 5개, JSON 수정 필요 89개, 문서 명세 이식 43행, 조건부 7개 항목**으로 나뉜다. 조건부는 JSON 3개와 문서 4행이다. 모든 조건은 상세 목록에 있다. `tests/tc`의 README, SPECS, CHAIN-BRINGUP 문서 3개는 테스트 수에서 분리했다.

제외한 항목의 대부분은 WBFT/Istanbul, EIP-7702, P256, StableNet의 Anzeon·Boho·시스템 계약 조건이다. go-wemix의 Fee Delegation 타입 0x16은 지원되므로 이 관련 시나리오는 무조건 제외하지 않았다. 원래 fork·수수료·계정 조건은 조정해야 한다.

## 가장 중요한 결론

1. **현재 JSON 189개에 블록 8 MiB나 메시지 10 MiB의 실제 직렬화 크기를 검사하는 테스트가 없다.** `stress-tx-flood`도 작은 gas 부하를 주는 구현이므로 이번 장애의 재현·해소 증거가 되지 않는다.
2. **추가 P0 7개를 배포 판단 전에 확보해야 한다.** 실제 블록 RLP 경계, 두 packing 경로, 최종 sealed block, historical/full/snap 처리, 혼합 배포, 메시지 경계, token과 다음 블록 생산을 검증한다.
3. **로컬 dev와 PR master는 분기되어 있다.** 로컬의 블록 시각과 중지 worker token 관련 기존 회귀 테스트 2개를 최종 통합 SHA에서 보존한다. PR196이 이 테스트를 삭제했다는 뜻은 아니다.
4. **기존 테스트 이름보다 assertion을 기준으로 판단해야 한다.** nonce 최종값만으로 포함 순서를 증명하지 못하고, receipt status=0만으로 상태 롤백·환불을 모두 증명하지 못한다. generic sync도 downloader/fetcher/snap 중 어느 경로로 동기화했는지 증명하지 못한다.

위 결론의 근거와 반론은 [PR 영향](pr-impact.md), [TC 범위 분석](tc-applicability-notes.md), [교차참조](source-crosswalk.md), [반대 검토](counter-review.md)에 있다. 크기 추정 오차가 존재한다는 이유만으로 실제 잘못된 블록 생성이 입증됐다고 판단하지 않았다. 정상 body가 무제한 집계되어 10 MiB를 넘는다는 주장도 채택하지 않았다.

## 코드와 PR 기준

[PR #196](https://github.com/wemixarchive/go-wemix/pull/196)은 RLP 블록 상한 8,388,608바이트와 eth 메시지 상한 10,485,760바이트를 도입한다. 생산 중 판단은 추정 크기에 1,000,000바이트 여유를 둔 `<7,388,608` 조건이다. 실제 블록 RLP, 생산 추정치, 네트워크 메시지 크기를 혼동하지 않는다.

- 로컬 dev: `902f9fce85c108cf24cdeb77bc70a622049748ce`
- PR base master: `c8321fc693bef619ec8abd80c82f5a224e89802a`
- 검토한 PR head: `5a93553cb25d41770c7419cd325229ac81bb3042`

기준 소스와 입력 문서는 `../sources/`, 해시는 `../source-manifest.json`에 보존했다. PR이 추가 수정되거나 실제 적용할 SHA가 달라지면 새 기준으로 다시 판정해야 한다.

기계 처리용으로 [전체 JSON](all-tests.json), [선별 JSON](wemix-prioritized.json), [추가 테스트 JSON](../additional-tests.json)을 제공한다. 세부 원문·기대 결과·스크립트 ID는 document-catalog.json과 tc-catalog.json에 보존했다.
