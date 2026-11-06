# Web UI 현황과 남은 작업 (2026-11-06)

이 문서는 Web UI의 지금 상태와 남은 일만 적는다. 끝난 항목은 지운다. 지난 경과는 git 이력에 있다. 화면 규칙은 [web/DESIGN.md](../../web/DESIGN.md), 사용자와 사용 장면은 [web/PRODUCT.md](../../web/PRODUCT.md), API 계약은 [web-ui.openapi.yaml](web-ui.openapi.yaml)이 정본이다.

## 1. 지금 상태

WEB-01~13은 2026-11-06 `22ac4c35`에서 모두 통과했다. 그 뒤 화면과 문서가 바뀌었으므로 다시 돌려야 한다. WEB-14는 마지막 작업으로 두었고, 14개 전체 통과의 마지막 기록은 `658e8632`다.

stablenet, wbft, wemix를 모두 Web에서 실제 클릭으로 끝까지 돌렸다.

`/impeccable critique`는 25/40, `audit`은 15/20이다. 이 점수는 `/chains` 순서를 바꾸기 전에 쟀다.

## 2. 남은 일

끝나면 이 목록에서 지운다.

1. `generate` 키를 쓰는 케이스 220건 중 15건을 Web에서 돌릴지 정한다. 지금은 Web이 검토한 키 스냅숏만 읽으므로 CLI로만 돌릴 수 있다. 넣으려면 Web의 키 처리 구조를 바꿔야 한다.
2. 낮은 우선순위 화면 항목을 처리한다. 작업별 페이지 제목, 정적 자산 압축과 캐시, 숨은 탭의 폴링, 전역 머리줄의 체인 안내, 서버 풀과 서버 세트 용어 통일이다.
3. `tests/repro/attach-external.sh`를 고친다. 지금 없는 `chainbench test --rpc`를 부른다. 지금 동작하는 방법은 `chainbench run --rpc <url> --chain <id> <spec>`이다.
4. `scripts/inventory/list-adapter-functions.sh`와 `scan-binary-hardcoding.sh`를 고치거나 지운다. 지금 없는 옛 셸 파일을 읽어 오류로 끝난다.
5. 위가 끝나면 WEB-01~14를 같은 소스로 돌린다. 지문이 하나여야 하므로 소스가 바뀌었으면 WEB-01~13도 다시 돈다.

## 3. 확인하지 못한 것

- 노드에 실제 이상이 있어 헤드라인에 원이 그려지는 경우는 단위 테스트로만 확인했다. 녹화 표본에 그런 실행이 없다.
- 오류 단축키는 표본에 ERROR 로그가 없어, 확인용 서버가 넣은 ERROR 두 줄로만 확인했다.
- 다시 실행 화면은 개발 증명의 단언으로만 확인했다.
- 계획 검토의 노드별 서버 배치는 아직 보여 주지 않는다. 대상 서버 수와 케이스, 끝난 뒤 처리만 보여 준다.
- 실행 중인 작업 화면, 파일 가져오기 미리 보기, 자원 충돌 상태는 재채점에서 보지 못했다. VoiceOver, Safari, Firefox, 실제 휴대폰도 확인하지 않았다.
- `tests/webui/browser_history_web09.mjs`는 기록 목록에서 실행 이름으로 찾은 첫 행을 누른다. 작업 기록 행과 세션 기록 행이 같은 이름을 쓰므로 작업 기록 행이 먼저 잡힐 수 있다.
- WEB-14 로그의 `Unknown config environment variable envvar=GWEMIX_BIN` 경고는 geth의 환경변수 검사가 내는 것으로 보이지만 확인하지 않았다.

## 4. 시험할 때 알아 둘 것

- `tests/webui/*.test.mjs`만 CI가 돌린다. `tests/webui/browser_*.mjs`, `run_*.py`, `test_evidence_*.py`는 CI도 `make check`도 돌리지 않는다. Web을 고치면 그 화면을 쓰는 개발 증명을 머지 전에 돌린다.
- 화면 작업은 실제 클릭으로 끝까지 가야 완료로 친다. API 호출 결과는 화면 검증이 아니다.
- WEB-14는 전체 Go 테스트를 race로 돌려 약 3시간 걸린다. 고정 커밋 바이너리(StableNet `4d17eea`, WBFT `d535dd3`, WEMIX `4538a9f`)와 `CHAINBENCH_W4_PROOF=1`, docker 서버 경로를 넘긴다.
- capture 전에 `selected-binaries.json`의 checksum을 확인한다. macOS는 `/private/tmp`에서 오래 쓰지 않은 파일을 지운다.
- 서버 풀은 다른 작업과 함께 쓴다. docker를 쓰는 시험 전에 서버에 노드가 없는지 본다.
- `docs/dev/web-ui-evidence/`의 게시 증거는 capture와 독립 재현을 함께 담은 2026-10-13 기록이다. 바꾸려면 같은 소스로 독립 재현까지 돌려야 한다.

## 5. 정해 둔 것

wemix 노드 종료는 wemix팀 몫이다. wemix 노드는 블록을 만들 때 네 노드가 함께 띄운 etcd에 기록하므로, 네 노드를 한꺼번에 멈추면 노드 하나가 15초 유예를 거의 다 쓸 수 있다. go-wemix 자체에서 생기는 문제는 엔진에서 우회하지 않는다. 원인이 go-wemix임을 로그로 보였다면 인수 확인에서 통과로 처리하고, 근거와 테스트 이름을 이 문서에 적는다. 유예를 넘겨 SIGKILL이 나면 그때 다시 본다.

## 6. 지금도 지키는 원칙

1. **계약의 원본은 Go 응답 구조체다.** OpenAPI는 손으로 고치되, Go 구조체와 OpenAPI 스키마가 다르거나 서버가 연 경로가 OpenAPI에 없으면 테스트(`TestWebOpenAPIMatchesGoResponses`, `TestWebOpenAPIListsEveryServedRoute`)가 실패한다.
2. **화면은 API 경계에서 응답 모양을 확인한다.** `web/src/api-decode.mjs`의 해독 함수가 그 일을 한다.
3. **실시간 전달은 `/api/v1/events` SSE 하나로 모은다.** 끊기면 스냅숏으로 다시 맞춘다.
4. **Web에서 여는 CLI 인자는 구현이 실제로 읽는 것만이다.** 노드 구성은 체인 preset 계약으로 표현한다.
5. **실패 증거는 비밀 검사를 거친 뒤 Web 기록에 넣는다.**
6. **새 진입점은 기존 기능을 덧붙이는 방식으로 만든다.** 마우스로 설정하고 실행하는 기존 흐름은 그대로 두고, 새 API는 기존 계획·실행 함수를 부른다.
7. **WEB-14는 큰 변경을 마친 뒤 마지막에 한 번만 돌린다.** 그 전에는 바꾼 기능의 개발 증명만 골라 돌린다.
