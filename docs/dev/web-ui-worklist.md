# Web UI 전체 작업 목록과 완료 기준

상태 정본은 [14개 작업 목록](web-ui-worklist.json)이다. 저장소 전체의
[열린 작업 목록](chainbench-worklist.md)에 연결한다. 승인된 Seed·동작 명세·
인수 시나리오를 줄이지 않는다. 현재 **전체 완료 0/14**다.

각 항목의 `completion`을 모두 만족하고, 테스트의 실제 실패(RED)·통과(GREEN),
전체 필수 시나리오의 최신 독립 live 증거를 기록한 경우에만 `complete`로 바꾼다.
부분 기능의 단위 테스트 통과는 해당 구현 단계의 완료이며 인수 항목 완료와 구분한다.
환경/권한 오류는 기능의 RED 증거로 사용하지 않는다. 과거 실패 로그를 만들거나
기존 구현의 RED를 소급해 주장하지 않는다. `internal/arch/web_worklist_test.go`는
항목 누락·중복·Seed 불일치·live 명령 누락·증거 없는 완료 표시를 거절한다.

매 단계 전 전체 목록을 읽고 완료/미완료를 나열한다. 해당 단계의 완료 기준과
테스트를 먼저 정하고 RED → 구현 → GREEN → 관련 회귀 → 영어 커밋으로 진행한다.
커밋 후 다음 단계를 이어가며 전체 인수 종료 전에는 목표를 완료 처리하지 않는다.
커밋과 PR에는 검증된 범위와 남은 한계를 명시한다.

## 단계 기록

| 단계 | 완료 기준 | 상태 및 근거 |
|---|---|---|
| 현재 구현 보존 | 기존 Web UI 변경만 커밋하고 사용자 스테이징 보존 | `0ee43cfa`: secure configuration editors and durable local and SSH jobs. 기존 make check·관련 race·JS/Python·실제 local/SSH 개발 검증 통과. 과거 RED가 없어 인수 완료 아님 |
| 전체 목록 보호 | 14개 모두 추적하고 누락·근거 없는 완료를 테스트에서 거절 | RED: 목록 파일 부재로 `TestWebWorklistTracksEveryAcceptanceCriterion` 실패. GREEN: `go test ./internal/arch -run 'Test(WebWorklist|WorklistOpenWork)' -count=1` 통과. RED/GREEN 로그는 `/private/tmp/chainbench-web-worklist-{red,green}.log` |
| 키 자료 고정 | 준비한 키 snapshot을 해시로 고정하고 원본 변경이 승인된 실행을 바꾸지 않음; 링크·특수 파일·과대 자료 거절; 비밀은 계획 응답에 없음 | RED: `/private/tmp/chainbench-web-key-red.log`에서 모든 사전 기준 실패. GREEN: `/private/tmp/chainbench-web-key-green.log`; make check·관련 race·실제 세 체인 로컬/SSH 개발 검증 통과. 전용 브라우저 종료 지연을 보강해 실제 재검증 통과. WEB-02/06의 부분 작업이며 업로드 미완료 |
| 저장 케이스 실행과 세션 연결 | 리비전 고정·변경 거절, 기존 엔진 실행, 실제 성공/실패 판정·세션 참조·기본 히스토리 보관 | 부분 구현 검증: `/private/tmp/chainbench-web-test-{history,binary,binary-default,layout,native-name}-red.log` → `chainbench-web-test-jobs-unit-green.log`. 실제 세 체인 브라우저 판정·리비전 고정·세션/히스토리 증거는 `chainbench-out/web-ui-development/test-jobs/receipt.json`. 전체 DSL 어휘/인자·자료·attach 등 WEB-05 완료 아님 |
| 전체 구성과 자료 연결 | 프리셋·업로드·매니페스트·노드 배치를 실제 계획/실행에 적용 | 미완료: WEB-01/02/04/06 |
| 노드 교체와 초기화 | 실제 설정/바이너리 교체·재실행, 비생산자 초기화, attach/생산자 거절 | 미완료: WEB-07 |
| DSL 실행 작업 | 모든 현행 DSL 구성·내장 어휘·인자 전수 편집 및 실제 실행, job/session 연결 | 미완료: WEB-05 |
| 자원 대조·취소·철회 | 잔존 노드 실제 확인·충돌·재시작·정리·실행 중 SSH 실패/철회 | 미완료: WEB-06/11/12/13 |
| metric·로그·히스토리 | 실제 수집·차트·시점 연계·archive·호환 비교·보호된 삭제 | 미완료: WEB-08/09 |
| 전체 인수·회귀·PR | 14개 전체 최신 live 검증·필수 회귀·PR 검토 | 미완료: WEB-03/04/10도 최종 소스에서 재검증 |

## 기존 증거의 한계

WEB-03·04·10의 독립 live 검증은 이전 소스에서 통과했다. 새 소스의 완료 증거로
재사용하지 않는다. `run_native_jobs.py`·`run_ssh_jobs.py`의 실제 바이너리/브라우저
개발 검증은 유효한 관측이지만 전체 인수 시나리오를 포함하지 않는다.
AST 그래프는 구문/의존 근거이며 실행 동작과 인수 통과를 증명하지 않는다.
