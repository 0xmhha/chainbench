# 검증 계획과 기준선의 한계

이 문서는 후속 구현의 비교 절차다. 여기 적힌 명령을 이번 문서 PR에서 새로 실행했다는 뜻은 아니다. 전체 호환성 판정은 **WITHHELD**다. [목표 계획의 AC2-V1–V8](target-and-plan.md)에 기존 테스트 경로, 추가 시험 및 단계 매핑이 있다.

## 보존할 계약

| 표면 | 비교 대상 | 보강할 증거 |
|---|---|---|
| CLI | 명령·flag·기본값·exit·stdout/stderr | 루트/하위 help 및 정상·오류 golden |
| DSL | v1/v2 문법·action 이름·인자·환경 정규화 | valid/invalid fixture, Registry 등록 어휘 |
| MCP | tool 이름·입력 schema·결과·오류 | initialize/tools/list 및 부정 입력 |
| 설정 | 생략/zero/default·우선순위·경로·직렬화 | 기존 파일 roundtrip, unknown 키, override |
| 결과·세션 | 상태·파일 형식·참조·재개·잠금·scrub | 기존 파일 읽기, 실패 자료, 복구 fixture |
| 체인 연동 | capability·구성·attach/reuse·handoff | 세 체인별 적용성, 실제 바이너리 hash, owned/borrowed 자원 |

## 수집된 부분 기준선

기준선 ID `20260913T101856Z`, 코드 revision `7f39c5627e0bdaccaa8e207a67767d71579283f5`, darwin/arm64 Go 1.25.13의 과거 기록이다.

- 기본 58 패키지 PASS, 테스트/하위 테스트 1,944 PASS·27 SKIP, 테스트 없는 12 패키지. SKIP은 성공에 합산하지 않는다.
- CLI build/help/chains/capabilities, WBFT 기본, Wemix→WBFT handoff는 해당 입력에서 PASS.
- Stablenet 최초 실행은 receipt 뒤 잔액 0→0으로 FAIL, contract 단계 미도달.
- 복원 관측은 node1/3/4에서 block 1·1 ETH, node2/5에서 genesis. 새 instrumented 네트워크의 150개 잔액 관측과 contract 42는 PASS, unmodified 재실행 1회도 PASS(42.47초).
- 복원 상태와 후속 단발 PASS는 최초 실패를 해결한 증거가 아니다. 실패 순간의 receipt/canonical/state·노드별 동시 관측이 부족해 직접 원인은 미확정이다.
- hardfork·장애·동기화·거버넌스, Wemix 독립 전체, live SKIP 항목, race, 성능·장시간은 미검증이다.

이 PR은 해당 수집 결과의 요약과 선택 근거를 제공한다. 원시 로그 전체를 포함하지 않으므로 새 호환성 승인에 사용하려면 원 기록을 확보하거나 별도 입력으로 재수집해야 한다.

## 구현 단계의 실행과 비교

1. 원본/후보 revision과 dirty/untracked 입력, 도구·OS/arch·CGO/tags, 비밀을 제외한 환경 조건, fixture·바이너리 hash를 새 실행 디렉터리에 기록한다. 과거 바이너리의 빌드 revision을 추정하지 않는다.
2. 저장소 루트에서 `make test`, `make lint`, `make build`, `make vet`, `make cover`를 실제 실행한다. 도구 버전과 실행된 명령을 확인한다. 구성 누락으로 SKIP된 검사는 통과로 세지 않는다.
3. 변경 경계의 테스트는 `go test -count=1 -json -timeout=6m ./해당/패키지 -run '대상테스트'` 형태로 명시한다. 실제 패키지·테스트는 AC2-V*의 기존 시험과 추가 fixture에서 선택해 실행 기록에 확정한다. 기본 테스트 wall 한도는 600초, 계약별 명령은 420초/정리 유예 5초다. lint/build/vet/cover도 실행 전 유한한 한도를 명시한다.
4. before/after에 같은 fixture·시나리오를 적용한다. exit 0, 필요한 테스트의 실제 PASS, 계약 관찰값 동일, 정리 완료를 모두 요구한다. SKIP/NOT_RUN/환경 부족/TIMEOUT/INTERRUPTED는 미완료다.
5. E2E는 `-tags e2e`, 별도 CLI 빌드와 CHAINBENCH, WBFT_BIN/GSTABLE_BIN/WEMIX_BIN, handoff 입력 등 해당 테스트의 전제조건을 준비한다. 기본 테스트 PASS를 E2E 증거로 대체하지 않는다.
6. 시간·PID·임시 경로·포트만 의미를 확인해 정규화한다. 거래/receipt/canonical 간 연결과 status·exit·schema·잔액·결과를 비교에서 제거하지 않는다.
7. 실패 자료를 보존한 후 해당 실행이 소유한 프로세스만 정리한다. attach한 사용자 노드를 종료하지 않는다. primary failure와 cleanup failure를 별도로 기록한다.

## Stablenet B1–B6 검증 절차

다음은 추가 관측의 목표다. 기존 테스트가 이미 모두 구현한다는 뜻은 아니다. 실제 판정 강화는 P4의 별도 결정에 따른다.

1. 노드 identity, chain ID, genesis hash를 기대 입력과 대조한다. 성공한 두 높이 응답 사이에서만 진행을 판정한다. RPC 오류 -1→genesis 0은 진행이 아니다.
2. 요청 to/value/chain identity와 반환 txHash, receipt status/blockNumber/blockHash를 기록한다. 제출 결과 불명 상태를 자동 재송금으로 해결하지 않는다.
3. receipt block과 같은 canonical block의 잔액을 비교한다. blockHash 지정이 미지원이면 blockNumber 조회 전후 hash 불변을 확인한다. RPC 오류를 잔액 0으로 치환하지 않는다.
4. 노드별 같은 높이 blockHash/stateRoot를 비교한다. 높이가 다른 상태는 지연 여부를 따로 판단한다. latest 관측과 receipt block 관측을 섞지 않는다.
5. readiness/progress, receipt, latest 수렴은 각각 총 60초, 요청당 2초, 관측 간격 1초를 후속 예산으로 사용한다. 변경하면 새 입력 조건으로 기록한다. 실패 직전 자료는 최대 10초 best-effort로 수집하며 전체 시나리오 420초를 넘기지 않는다.
6. reorg·hash/state 불일치·RPC 오류·timeout을 보존한다. original/restored/instrumented/unmodified 구분, 노드별 관측 이력과 정리 결과를 함께 남긴다.

## 재확인 조건

소스·문서·계약 등록·도구·빌드 조건이 달라지면 새 inventory와 snapshot을 만들고 영향받는 진단·목표·검증 연결을 갱신한다. 모든 산출물의 ID 및 evidence_id 참조 통합 검증, 현재 입력 재열거와 보존 snapshot 내부 무결성 검증은 별도로 통과해야 한다. 선택 근거 발췌의 무결성만으로 이 조건을 충족했다고 판정하지 않는다.
