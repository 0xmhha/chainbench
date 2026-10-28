# Chainbench Web UI — Codex에서 Claude Code로 인계

작성: 2026-10-09 KST. 사용자가 구현을 여기서 멈추고 Claude Code로 이어받도록 요청했다.
Codex 목표는 paused다. 이 문서는 인계 기록이며 구현 완료 판정이 아니다.

## 먼저 알아야 할 현재 상태

- 작업 폴더: `/Users/0xtopaz/work/github/0xmhha/chainbench`
- 브랜치: `codex/web-ui`
- 마지막 검증·커밋·push: `7c8de8762e269b4a7b2156a18743d3b26968c2e5`
- `origin/codex/web-ui`도 같은 커밋이다. 이후 스냅샷/스트림/모니터링 변경은 **미커밋**이다.
- PR: https://github.com/0xmhha/chainbench/pull/443 — 기존 draft. 마지막 확인에서 reviewRequests는 빈 배열.
  다시 상태를 확인하며 ready/merge하거나 중복 PR을 만들지 않는다.
- **전체 인수 완료는 0/14**. 상당한 부분 구현·실제 native 개발 검증이 있지만 전체 인수와 다르다.
- 현재 단위/HTTP/JS·필수 Go 검사·관련 race·빌드는 통과했다.
  **새 native 브라우저 복원 증명은 마지막 노드 관측 단언에 실패했다. 현재 단계를 GREEN/완료로 기록하거나 커밋하지 않았다.**
- Codex의 작업을 중단한 상태에서 인계 자료만 추가했다. 이 자료가 추가되므로 이전 sourceDigest도 최신 소스 증거가 아니다.

## 읽는 순서와 정보의 우선순위

1. 이 문서와 `git status`, 마지막 커밋·working diff를 읽는다.
2. [확정 Seed](architecture/chainbench-web-ui.seed.yaml),
   [동작 명세](web-ui-spec.md), [인수 계약](web-ui-acceptance.md)를 읽는다.
3. [작업 정본 JSON](web-ui-worklist.json)의 14개 `completion`을 전부 읽고
   [단계별 기록](web-ui-worklist.md)으로 검증된 부분을 대조한다.
4. [HTTP 계약](web-ui.openapi.yaml), [UI 설계](web-ui-design.md),
   [매니페스트 계약](web-ui-manifests.md), [metric 설계](dashboard-metrics-design.md)를 읽는다.
5. [구현 기록](web-ui-implementation.md)과 실제 코드를 대조한다.
   이 구현 문서의 일부 문단과 오래된 journal 필드는 최신 커밋보다 뒤처질 수 있다.
6. `.ouroboros/web-ui-analysis/interview-ledger.json`은 실제 인터뷰 결정의 보충 자료다.
   `.ouroboros/web-ui-analysis/manual-continuation.json`은 누적 검증 기록이다.
   최신 `claudeHandoff` 필드를 먼저 읽고, 오래된 `remaining`/`status`를 현재 상태로 오해하지 않는다.
7. `docs/chain-analysis/{gstable,gwbft,gwemix}` 각각의 네 자료와 실제 선택 바이너리를 확인한다.
8. `tests/webui/README.md`, fixture/runner, 최신·과거 receipt를 읽는다. 과거 receipt는 역사적 근거다.

우선순위: 사용자의 확정 요청 → 승인된 요구·검증 계약 → 실제 소스·실제 관측 → 현재 인계 기록
→ 시간 표시가 있는 과거 기록. 충돌을 조용히 한쪽에 맞추거나 승인 범위를 줄이지 않는다.
`web-ui.openapi.yaml`에 선언되어 있다는 이유로 엔드포인트가 이미 동작한다고 판단하지 않는다.

## 사용자가 확정한 범위

원래 제안서의 단계적 읽기 전용 출시는 채택된 최종 목표가 아니다.
첫 출시에서 아홉 요구 모두 브라우저로 완료해야 한다.

1. 체인 프리셋을 바탕으로 클릭 중심 구성과 셋업 준비.
2. 설정 파일 import/edit/export 및 실제 셋업.
3. `server-set.yaml`과 `workspace-config.yaml`, 서버 배치·workspace 관리.
4. 체인 플러그인 매니페스트 관리와 기존 지원 family의 외부 매니페스트.
5. 현행 DSL 전체의 구조 편집과 import/export·실제 실행. 새 문법은 추가하지 않는다.
6. 실제 서버 배포·노드 환경 구축.
7. 소유 노드 시작/정지·설정/바이너리 교체·재실행·허용 초기화.
8. 실행·노드 상태·이벤트·노드별 metric 차트·해당 시점의 로그 연계.
9. 히스토리 검색·필터·상세·호환 비교·export·권한 있는 삭제와 metric/log archive.

개인 PC와 팀 서버를 모두 지원한다. 자체 관리자 bootstrap/login과 세 역할을 사용한다.
공유 구성·체인은 팀 자원이고 SSH 자격증명은 본인만 사용하는 암호화된 개인 자료다.
서버가 구성·작업·결과를 보관하며 import/export도 제공한다. 조회자는 공유 관측 자료만 읽는다.
작업은 브라우저 종료/로그아웃/연결 단절에도 지속된다. 서버 재시작 시 진행 중 작업은
interrupted로 저장하고 자동 실행하지 않는다. 일반 종료·취소의 기본은 보존이며 선택 정리를 지원한다.
계정/자격증명 철회는 취소·보존이 우선이고 철회 후 새 SSH 접속/후속 정리를 하지 않는다.
물리 호스트·경로·포트·프로세스의 충돌은 workspace 별칭으로 우회할 수 없다.
attach/미소유 노드의 제어와 생산자 초기화는 거절한다.

선택지는 공통 엔진 계약·실제 바이너리 지원·현재 작업 capability의 교집합이다.
`docs/chain-analysis`의 세 체인 × 네 문서를 반드시 활용한다. 오래된 capture를
선택 바이너리의 실제 지원으로 오해하지 않는다. wbft/wemix 실행 파일의 basename이 같을 수 있다.
UI는 직접 디자인한 Svelte 화면을 계속 개선한다. 별도 Prometheus/Grafana 서버를 필수로 도입하지 않는다.

## 사용자 작업·Git·PR 규칙

- 매 다음 구현 단계 전에 **14개 전체 목록**, 현재 하위 완료와 전체 미완료를 나열한다.
- 단계마다 완료 기준·테스트를 먼저 정의하고 실제 RED → 구현 → GREEN → 필수 회귀 → 커밋.
  커밋 뒤 전체 목표를 끝낼 때까지 다음 단계를 계속한다. 불완전한 것은 완료 표시하지 않는다.
- 환경 listen/디스크 오류, fixture 자체 오류, 단순 파일 존재, mock, skip, 오래된 receipt를
  실제 제품 실패/통과로 바꾸어 기록하지 않는다. 실패 원인을 모르면 모른다고 쓴다.
- 영어 Conventional Commit, 영어 PR. 수정의 실제 의미를 설명하고 stage 용어/불명확한 약어,
  emoji, co-author를 쓰지 않는다. 전체 커밋 메시지는 300자 이내.
- 사용자는 squash merge 예정. 긴 본문을 이어 붙이기보다 핵심 커밋 제목을 정리한다.
- **SAGE Core Team / `SAGE-X-project/sage-core-dev`에 어떤 경로로도 review 요청하지 않는다.**
  CODEOWNERS/workflow/자동 assignment를 확인하고 base가 해당 팀을 지정하면 draft를 유지한다.
- 사용자의 `.gitignore`가 이미 staged. 인계 시 index blob:
  `100644 54b076eab7779720ad5afae6169635d45e520759`. 변경하거나 우리 커밋에 섞지 않는다.
- `docs/dev/architecture/chainbench-refactoring-preparation*`, `baselines/`,
  `refactoring-preparation-*.md`, `stablenet-boundary-audit/`는 사용자 untracked 작업이다.
- `git add .`, `git commit -a`, reset/clean/stash로 working tree를 정리하지 않는다.
  필요한 새 파일만 명시적으로 stage하고 검증한 우리 파일만 `git commit --only -- <paths>`.
- 외부 체인 소스는 읽기 전용 근거다. 무관한 리팩터링·운영 서버 변경을 하지 않는다.

## 전체 작업 목록: 14개 모두 미완료

| ID | 전체 완료 대상 | 현재 판단 |
|---|---|---|
| WEB-01 | 세 체인의 전체 프리셋·상속·값·노드 표·chain surface | 부분 편집/실행 연결 있음, 전수 미완료 |
| WEB-02 | import/edit/export/bundle·불변 assets·실제 setup | 부분 구현/고정 입력 검증 있음, 전수 미완료 |
| WEB-03 | 두 운영자의 공유 revision·개인 SSH·실제 접근·암호화 | 과거 실제 capture/recheck 통과, 최종 인수 미부여 |
| WEB-04 | 세 내장/외부 manifest·불일치/stale 거절·실제 setup | 과거 실제 capture/recheck 통과, 최종 인수 미부여 |
| WEB-05 | 현행 DSL 22 actions/32 assertions/13 readers·전체 인자·v1·실행 의미 | 구조화/실제 test.run의 여러 부분 검증, 전수 미완료 |
| WEB-06 | native 실제 SSH 배포·checksum/DB/genesis/RPC/PID·실패/철회 | 실제 WBFT 전용 SSH/고장 주입 부분 검증 있음 |
| WEB-07 | native 소유 노드 control/config/binary/restart/reset·attach 제한 | 세 체인 controls/reset/config 및 WBFT binary 부분 검증 있음 |
| WEB-08 | 상태/metric/time-linked logs·disconnect/replay/gaps/stale | 현재 snapshot/replay WIP, metric/log 수집·차트 미완료 |
| WEB-09 | 검색/상세/호환 비교/export/admin delete·metric/log archive | 실제 session/history 부분 구현, archive 등 미완료 |
| WEB-10 | 개인/팀·세 역할·CSRF·비밀 격리·실제 브라우저 | 과거 capture/recheck 통과, 최종 인수 미부여 |
| WEB-11 | 물리 aliases/ports/paths/PIDs·독립 작업 병행 | 부분 guard/실제 fixture 검증, 전체 미완료 |
| WEB-12 | 실행 중 재시작·interrupted·잔존 상태 대조·명시적 재실행 | 부분 검증, 현재 native 복원 마지막 probe 실패 조사 필요 |
| WEB-13 | 취소/기본 보존/선택 cleanup/철회/attach | 부분 실제 local/SSH/철회 검증, 전체 미완료 |
| WEB-14 | 모든 최신 독립 live 인수·전체 race/e2e vet/build·기존 회귀 | 미완료, 전체 0/14 |

이 표는 정본 JSON의 완료 조건을 대체하지 않는다. 정확한 전수 조건을 반드시 읽는다.

## 이미 구현한 구조와 코드 지도

- `cmd/chainbench-dashboard`: 서비스 조립·계정/암호화 store·engine adapter.
- `internal/dashboard`: 기존 `/events`/read API를 보존한 인증 API·vocabulary/contracts·jobs/history.
- `internal/app/web_auth.go`: bootstrap/역할·CSRF용 세션·계정 철회.
  **로그인 세션은 의도적으로 프로세스-local이며 서버 재시작 후 로그인해야 한다.**
- `internal/app/deployment.go`·`deployment_store.go`: 공유 문서/revision/workspace·개인 credential/binding.
  import 원문과 키 자료는 private encrypted storage; 공유 응답/로그/export를 sanitize한다.
- `internal/app/web_chain_engine.go`, `web_chain_execution.go`, `web_job_execution.go`:
  같은 CLI/MCP 엔진 호출. 계획과 accepted 입력·credential ID를 고정하고 접속마다 철회 검사.
- `web_jobs.go`, `web_job_acceptance*.go`, `web_resource_holds.go`:
  durable plan/job/audit/idempotency·물리 exclusion·기본 retained 자원과 부분 실패.
- `web_test_*.go`, `web_preset_composition.go`, `web_genesis*.go`, `web_key_snapshot.go`:
  불변 case/asset/genesis/key 입력과 실제 engine session 연결. 미지원 범위는 거절.
- `web_recorded_controls.go`, `web_node_probe.go`, `web_node_observations.go`,
  `web_control_binary.go`, `web_node_binaries.go`, `web_config_change.go`:
  PID/argv/물리 경로/ledger/등록 SHA 검사·소유 control·지속 node executable binding·이력.
- `internal/resource/regular_file.go`: regular/non-symlink와 실제 계정의 `test -x`를 검사한다.
  SHA가 같아도 실행 불가 파일은 stop 전 거절하며 자동 chmod로 우회하지 않는다.
- `internal/core/remote`: SSH handshake/exec 취소·매 연결/tunnel 요청 철회·stdin file transfer.
- `web_history*.go`: 실제 session/index·숫자 정밀도·호환 비교·redacted export·tombstone/audit.
  삭제는 captured 자료를 대상으로 하고 live DB/CLI artifact/job record를 삭제하지 않는다.
- `web/src`: App/LoginPanel/ChainPresetEditor/DeploymentEditor/DSLEditor/JobPanel/
  ManifestManager/AssetManager/HistoryPanel. 현재 새 MonitorPanel은 미커밋이다.
- `tests/webui`: owned browser/실제 native/local/OpenSSH fixture·source-bound receipt 검증.
  개발 runner는 `seedAcceptanceAwarded:false`이며 전체 인수를 부여하지 않는다.

최근 검증된 커밋:

| 커밋 | 구현·검증 |
|---|---|
| `e6b3d3c1` | 검토된 native node executable 교체·지속 binding·restart 이력 |
| `3430c1e3` | 본인 SSH로 native executable 교체·후속 제어·되돌리기 부분 증명 |
| `43d86fb8` | 실제 SSH transfer 완료/응답 대기 중 단절·credential 철회와 명시적 재시도 |
| `bdd175e6` | 실제 실행 권한 없는 교체/restart를 기존 PID 변경 전 거절 |
| `7c8de876` | 관측된 실행 불가 권한을 UI 제어에 반영, 가동 PID/명시적 stop 유지 |

더 오래된 commit/단계별 RED/GREEN은 worklist MD, Git log, 누적 journal에 있다.

## 현재 미커밋 단계: snapshot + bounded replay + 모니터링 복원

완료 기준: viewer-auth read, durable job version/process cursor, 128개 bounded replay,
old/invalid/restart/gap 시 resync, private plan 제외, 범위 filter·철회·unsubscribe,
브라우저 단절 중 실제 native 작업 지속과 서버 재시작/재로그인 후 복원.
메트릭·시점 로그·전체 WEB-08 완료는 이번 하위 기준에 포함하지 않았다.

미커밋 변경 파일:

- `internal/app/web_jobs.go`: durable commit 후 public feed publish, 생성 시 feed 초기화.
- `internal/app/web_job_observation.go`, `web_job_observation_test.go`:
  snapshot/revision/cursor/journal/subscribers·replay/resync·drop·consistent retry·독립 복사.
- `internal/dashboard/jobs.go`, `snapshot.go`, `snapshot_test.go`, `snapshot_stream_test.go`:
  GET `/api/v1/snapshot`, GET `/api/v1/events`, 인증·scope·Last-Event-ID 우선·revocation.
- `web/src/job-observation.mjs`, `MonitorPanel.svelte`, `App.svelte`:
  version/generation/gap 판정·snapshot 복원·abort/close·stale/누락 표시·unknown cached nodes.
- `tests/webui/job-observation.test.mjs`, `browser_snapshot_restore.mjs`, `run_snapshot_restore.py`.
- `docs/dev/web-ui-worklist.md`, `web-ui.openapi.yaml`, `architecture/package-tree.md`.
- SPA: 기존 `index-DU6Lmz1X.js`/`index-DSLJFrmF.css` 삭제;
  새 `index-BezA55vh.js`/`index-Cruezb5v.css`와 index.html.
  새 asset 두 개는 **untracked**, commit/patch에서 빼먹지 않는다.

서버 상세:

- version = durable audit 길이. write 실패는 version/feed를 변경하지 않는다.
- cursor = 생성한 process instance 32hex + `:` + committed version.
  재시작은 버전을 유지하지만 cursor generation이 바뀐다.
- journal/subscriber buffer 128, 동시 subscription 최대 64.
  느린 구독자는 nonblocking drop, global drop counter를 별도로 표시한다.
- `job_changed`는 public detached WebJob. 계획 payload/arguments/private refs를 싣지 않는다.
- 무관한 scoped job은 payload 없이 `version_changed`로 전달하여 전체 revision 연속성을 유지한다.
- `resync_required` 이유: initial_snapshot_required/cursor_invalid/server_restarted/cursor_expired.
- snapshot은 job revision 변경 시 최대 3회 재시도. 네트워크는 owned **cached** 기록이며
  SupportedControls를 비우고 PID를 live proof로 승격하지 않는다.
  모든 구성/외부 runtime 변경을 하나의 전역 revision으로 표현한 것은 아니다.
- SSE observation heartbeat는 1초, 현재 version/cursor/drop/시각을 전달하되 **id 없음**.
  못 받은 변경까지 Last-Event-ID로 인정하지 않는다. 매 event/heartbeat에 인증 재확인.
- legacy `/events`는 별도 observational bus timeline. 신규 job stream과 의미/손실 범위가 다르다.
- 프론트는 sequence gap/generation 변경 시 fresh snapshot을 받아 복원한다.
  연결 단절 시 마지막 상태/경고를 유지하며 session 변경·unmount 때 fetch/stream을 정리한다.

## 이번 단계의 실제 검증과 실패

| 검증 | 마지막 결과·로그 |
|---|---|
| 최초 snapshot HTTP | RED: `/private/tmp/chainbench-web-job-snapshot-http-red.log` — endpoint 404 |
| jobId network scope | RED: `/private/tmp/chainbench-web-job-snapshot-scope-red.log` — 무관 network 포함 |
| cached slice 분리 | RED: `/private/tmp/chainbench-web-job-snapshot-copy-red.log` — reader 원본 변경 |
| app/실제 TCP HTTP SSE | GREEN: `/private/tmp/chainbench-web-job-snapshot-stream-green.log` |
| JS 복원 로직 | missing module RED 후 GREEN; 현재 전체 JS 13 pass: `chainbench-web-snapshot-js.log` |
| 실제 UI 초기 RED | `/private/tmp/chainbench-web-snapshot-browser-red.log` — 실행 상태 heading 없음 |
| `make check` | PASS: `/private/tmp/chainbench-web-snapshot-check.log` |
| app/dashboard race | PASS: `/private/tmp/chainbench-web-snapshot-race.log` |
| 세 Go build | PASS: `/private/tmp/chainbench-web-snapshot-build.log` |
| SPA build | PASS: `/private/tmp/chainbench-web-snapshot-spa.log` |
| native browser 전체 | **FAIL**: `/private/tmp/chainbench-web-snapshot-browser-green.log` — 이름의 green은 통과 의미 아님 |
| Python·새 full acceptance captures | 이번 snapshot 단계에서는 아직 수행하지 않음 |

Go 단위 controlled metadata executor는 native 블록체인 증거가 아니다.
초기 HTTP stream fixture는 같은 physical port claim 충돌·sandbox listener 거절도 있었다.
이를 제품 RED로 기록하지 않는다. 다른 fixture port와 loopback listener 권한으로 정확한 테스트가 통과했다.

native 첫 복원 시도는 서버 재시작 후 session 자동 유지라는 잘못된 fixture 가정 때문에 실패했다.
`web_auth.go`는 의도적으로 restart에서 세션을 만료한다. 실제 로그인 UI로 재인증하도록
fixture를 수정했으며 인증 정책 자체는 변경하지 않았다.
그 다음 실행은 실제 test.run, 브라우저 offline 중 완료, snapshot 복원, 네 native PID/argv,
실제 genesis RPC, 서버 재시작/재로그인, 동일 job/version·새 cursor, 기록 bytes 보존,
모바일 넘침 단언까지 진행했다. **마지막 `live.nodes.every(state==='running')`가 실패했다.**
정확한 node state/reason을 파일로 저장하지 않아 원인이 아직 확정되지 않았다.
디스크·권한·PID 종료·ledger 문제라고 단정하지 않는다. 재현해서 원인 자료를 먼저 보존한다.
단언을 지우거나 느슨하게 만들어 GREEN으로 바꾸지 않는다.

실패 fixture는 아직 남아 있다:
`/private/tmp/cbui/b99b880a-c444-4f0b-b8cd-90f6e82bdbd1`.
`chainbench-out/web-ui-development/snapshot-restore`의 server/browser/build 로그와
` snapshot-desktop.png`, `snapshot-mobile.png`는 실패 실행의 자료다.
이번 runner의 성공 receipt/browser.json은 생성되지 않았다.
기존 fixture finally는 dashboard/browser를 종료하고 정확한 runtime/argv 소유 확인 뒤
그 fixture의 native 프로세스만 종료하도록 설계되어 있다. 현재 프로세스 존재 여부는 별도 확인한다.

## 재개할 첫 행동

1. 읽기 전용으로 Git/index/작업 정본·현재 diff와 위 실패 로그를 확인한다.
2. 14개 목록을 나열하고 현재 snapshot 하위 기준을 다시 명시한다.
3. native fixture의 마지막 observation JSON을 실패 전에 보관하고
   node state/reason·실제 ps/argv·ledger/target·권한·로그를 확인하도록 **진단을 추가한다**.
   이 진단 추가는 아직 수행하지 않았다.
4. 디스크 공간·fixture 생존/소유 여부를 확인하고 동일 실패를 재현한다.
   자동 삭제 거절 이력을 준수한다. 전체 root 삭제로 우회하지 않는다.
5. 원인이 실제 제품이면 의미 있는 RED → 원인 수정 → GREEN.
   fixture 가정이면 정확한 실제 계약을 검증하도록 고치되 기존 단언 범위를 줄이지 않는다.
6. 소스/문서를 고정하고 SPA build → Go build/native proof.
   `run_snapshot_restore.py` 전체 성공 후에만 이 하위 단계 완료로 계산한다.
7. 필요한 필수 검사·Python·AST·fresh WEB-03/04/10 capture와 독립 재현을 수행한다.
   성공한 검증 중 source 수정이 있으면 다시 해당 증거를 생성한다.
8. 사용자 파일을 제외한 명시 경로로 영어 커밋 → push → 기존 draft PR 본문에
   검증된 범위와 전체 0/14를 반영한다. PR reviewer 정책을 재확인한다.
9. 그 다음 전체 목록을 다시 나열하고 metric/time-scoped log 수집·차트·archive,
   나머지 preset/DSL/assets/control/reconciliation 인수 빈틈을 계속 구현한다.

## 검증 실행 환경과 명령

macOS arm64, repo `go.mod`의 Go 1.26.8 계약, Svelte 5/Vite 6.
이 머신에서는 항상 짧은 runtime root를 지정한다. default의 긴 path는 IPC 소켓 제한을 넘을 수 있다.

```sh
export WEBUI_RUNTIME_ROOT=/private/tmp/cbui
export GOCACHE=/private/tmp/chainbench-go-cache

# repo root에서
make check
make build
node --test tests/webui/*.test.mjs
PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tests/webui -p 'test_*.py'
go test -race ./internal/app ./internal/dashboard -count=1
python3 tests/webui/run_snapshot_restore.py
```

SPA는 `web/`에서 `npm run build` 또는 repo root에서 `npm --prefix web run build`.
설치된 `web/node_modules`, `tests/webui/node_modules`가 있다. 필요 시 각 lockfile로 `npm ci`.
브라우저는 전용 profile의 installed Chrome, `WEBUI_BROWSER_CHANNEL=chromium`이면
별도로 설치된 Playwright Chromium. `owned-browser.mjs`와 `browser_process.py`를 사용한다.
단순 브라우저 종료가 멈춘 실행은 통과로 계산하지 않는다.

최종 인수는 Seed의 전체 race/e2e vet와 14개 verify_command가 필요하다.
WEB-03/04/10의 기존 fresh capture/recheck는 과거 source-bound 통과 기록이며 현재 소스로 재사용하지 않는다.
원본 capture는 SPA를 재빌드하므로 serialize한다. 독립 recheck는 source copy에서 실행한다.

```sh
bash tests/webui/verify.sh --criterion WEB-03 --require-live --capture
bash tests/webui/verify.sh --criterion WEB-03 --require-live
# WEB-04, WEB-10 및 나머지도 각 계약에 따라 수행
```

Go cache/loopback sockets/native/SSH/ps/네트워크 작업은 Codex sandbox에서 별도 권한이 필요했다.
Claude Code의 권한 설정은 그 런타임에서 확인하며 Codex의 grant를 자동 승인으로 가정하지 않는다.

## native 바이너리와 실제 개발 증거

인접 원본 소스/바이너리:
`/Users/0xtopaz/work/github/0xmhha/chain/go-{stablenet,wbft,wemix}`.
fixture `tests/webui/fixtures/prepare_web04.py`가 commit/version/help/platform을 실제 검증한다.
원본 repo를 변경하거나 basename만 보고 family를 판정하지 않는다.

별도 WBFT 교체 candidate:

- `/private/tmp/chainbench-web-replacement-native/gwbft`
- 63,299,026 bytes, SHA-256 `f073315d1ae9b035c526475bbba91a831e166b5f4b43a5dbf32b5674dcbaca78`
- source commit `7af50e45db4a48a2a77e565c7ff7bde0e8f95505` (원본과 같은 commit, 다른 executable bytes)
- `provenance.json`, `asset-inspection.json`, `plan-red.json`이 같은 parent에 있다.
- 외부 source를 변경하지 않고 Go 1.25.2 compatibility build를 사용했다:
  `go build -o /private/tmp/chainbench-web-replacement-native/gwbft -ldflags '-buildid=chainbench-web-replacement -checklinkname=0' ./cmd/gwemix`
- proof: `WEBUI_REPLACEMENT_BINARY=<위 파일> python3 tests/webui/run_ssh_binary_replacement.py`.
  다른 machine에서는 실제 호환 source/build를 새로 준비해야 한다.

최신 커밋까지의 중요한 부분 receipt (모두 `chainbench-out/web-ui-development/` 아래):

| receipt 디렉터리 | 검증된 부분/런타임 |
|---|---|
| `ssh-binary-replacement` | 실행 권한 UI/stop/복원·native 교체, runtime `4b6c7378-2397-4516-89be-5e9265fb9356` |
| `recorded-controls` | 세 체인 18노드 control, runtime `502a667e-c902-4bef-9ccb-1f20a97189fa` |
| `ssh-transfer-disconnect` | native 복사 완료 후 소유 SSH 연결만 단절, runtime `2580f483-abb6-4adf-8980-78a135d22cae` |
| `ssh-transfer-credential-revoke` | 복사 중 개인 credential 철회·부분 자료/노드 보존, runtime `af57c0c9-cf83-40f6-941d-b72e33135bc3` |
| `ssh-jobs` | 원래 실제 SSH 배포/control 회귀, runtime `b7a5f50e-f80d-4fa8-bb0e-d83027702089` |

첫 두 receipt sourceDigest:
`a8e7d10d7e68cfe862514c9e3fb4c4a1374d5ec2b5009973ab3adfa76e6026f6`.
transfer 두 receipt sourceDigest:
`37fa53fc32d2d5a311d6f2c1542eb7a28e0e1e95dc585c4c2e053c82aa0aa19d`.
현재 source와 다르므로 역사적 부분 근거로만 사용한다. 실제 receipt의 값/파일을 다시 확인한다.

native fixed ports를 공유하는 control/reset/pinned proof는 serialize한다.
SSH proof는 RPC 10800/P2P 35500을 공유하므로 serialize한다.
현재 snapshot proof는 RPC 11700/P2P 39100, 새 독립 root/dashboard port를 쓴다.

## AST 그래프

`.ouroboros/web-ui-analysis/recovered-snapshot`에 현재 단계의 구문 그래프가 있다.
마지막 snapshot: source 1,188 files, Go 1,016, Svelte 12, symbols 9,590,
edges 76,984, generic grammar diagnostics 22.
Native Svelte compiler parse는 12 files, diagnostics 0.
이후 fixture의 재로그인 코드와 인계 문서 추가는 해당 graph 시점보다 뒤다.
기존 최초 그래프/인터뷰 그래프도 parent 아래에 보존되어 있다.

```sh
go run ./scripts/inventory/code-graph . > .ouroboros/web-ui-analysis/recovered-snapshot/go-graph.json
PYTHONPATH=/private/tmp/chainbench-ast-libs PYTHONDONTWRITEBYTECODE=1 python3 .ouroboros/web-ui-analysis/parse_sources.py --root . --output .ouroboros/web-ui-analysis/recovered-snapshot --go-graph .ouroboros/web-ui-analysis/recovered-snapshot/go-graph.json
node .ouroboros/web-ui-analysis/parse_native_svelte.mjs /Users/0xtopaz/work/github/0xmhha/chainbench /Users/0xtopaz/work/github/0xmhha/chainbench/.ouroboros/web-ui-analysis/recovered-snapshot
```

AST는 syntax/import/symbol 근거다. interface dispatch/receiver 타입/runtime registry와
build constraints를 완전히 해석한 call graph가 아니며 동작/인수 증명이 아니다.
YAML/JSON은 별도 configuration inventory. generic Svelte grammar 한계를 native compiler로 대조한다.

## 디스크와 삭제 거절 — 우회 금지

마지막 확인은 free 약 1.7 GiB, task-specific Go cache 약 3.9 GiB였다.
기존 genesis preflight는 2 GiB 미만이면 거절한다. production threshold를 낮추지 않는다.
이전에는 정지·소유가 확인된 superseded fixture의 공개 기록을 보존하고 정리한 이력이 있다.
다만 마지막 **다섯 root 전체 삭제** 요청은 자동 승인 검토에서 거절됐다:
일부만 archive하고 나머지 data를 돌이킬 수 없이 삭제하며 정확한 root 삭제의 명시 승인이 없다는 이유다.
해당 삭제와 뒤따르는 진단 수정/재실행은 수행되지 않았다.

거절된 root:

- `/private/tmp/cbui/38c452f4-1060-48ae-aace-68dfb016258a`
- `/private/tmp/cbui/cbea6dcb-300f-4d1a-9044-701963097a6b`
- `/private/tmp/cbui/b99b880a-c444-4f0b-b8cd-90f6e82bdbd1`
- `/private/tmp/cbui/ea0710ec-253a-44a0-a9ed-478f2ec4c0ec`
- `/private/tmp/cbui/b9c1f2ec-f17b-4ac1-bcae-f1a4a91dc242`

다른 shell/agent/도구로 같은 삭제를 우회하지 않는다. 사용자 데이터/cache는 건드리지 않는다.
전체 자료 보존 같은 안전한 대안·명확한 소유/폐기 승인·현재 프로세스 확인부터 검토한다.
광범위한 kill/rmtree 또는 저장된 raw PID만으로 signal을 보내지 않는다.
특히 `run_ssh_jobs.py` finally의 raw-PID cleanup은 별도 소유 확인을 강화할 여지가 남아 있다.

## 이전 Ouroboros 실행 기록

초기 인터뷰/Seed 생성·실행·평가를 사용했으나, 해당 실행은 전체 인수를 달성하지 못했다.
원 실행 `job_9cbf20bc218b`은 verify_workspace_mutated 실패,
평가 `job_9ee04cd84339`은 14개 AC 비준수,
Ralph `job_efbac5e0f3a6`은 1,800초 iteration timeout으로 실패했다.
이후 여기 checkout에서 수동 구현·검증·커밋을 계속했다.
run_observer child는 완료 상태다. `.ouroboros/.../last-progress.json`의 running 표시가 오래된 것일 수 있다.
옛 worker worktree를 새 현재 작업으로 오해하거나 실패 세션을 자동 resume하지 않는다.
이 인계 시점에는 새 worker를 만들지 않았으며 Codex 목표는 paused다.
옛 외부 runtime process 종료 여부는 당시 관찰자도 모두 확인하지 못했다. 필요하면 읽기 전용으로 확인한다.

## 인계 사용 방법

같은 머신/working tree에서 Claude Code를 열고 다음을 요청한다:

> CLAUDE.md와 docs/dev/web-ui-claude-handoff.md를 먼저 읽고 연결된 확정 Seed/명세/인수 계약과
> 14개 작업 정본을 모두 확인해. 현재 Git/index·미커밋 변경·실패 로그를 대조하여
> 확정 범위, 검증된 부분, 미완료 14개, 마지막 native 실패, 첫 재개 행동을 보고해.
> 그 뒤 snapshot 복원 단계의 실패 조사부터 구현을 이어가고, 의미 있는 RED→GREEN과
> 필수 회귀를 통과한 단계만 규칙대로 커밋하며 전체 요구가 완료될 때까지 계속해.
> 사용자 변경과 삭제 거절 이력을 보존하고 범위를 줄이거나 부분 완료를 전체 완료로 표시하지 마.

이 checkout에는 모든 working files와 private 분석 경로가 남아 있어 같은 머신 인계가 가장 간단하다.
다른 머신은 origin의 현재 branch만 clone하면 미커밋 코드를 받지 못한다.
별도 local handoff bundle의 명시 WIP patch와 context를 받아 baseline commit에 **충돌 확인 후** 적용해야 한다.
원본 checkout에 이미 있는 patch를 다시 적용하지 않는다. private credential/key/runtime store를 통째로 옮기지 않는다.
전용 fixture·호환 native 바이너리·브라우저·패키지·권한은 새로운 환경에서 다시 준비하고 검증한다.

이 문서/CLAUDE.md와 별도 인계 bundle은 아직 커밋하지 않았다.
실패 중인 snapshot 변경을 완료 커밋으로 포장하지 않고 현재 그대로 넘긴다.


## 실행 중인 Claude Code에 붙여 넣을 프롬프트와 ooo 활용

[재개 프롬프트](web-ui-claude-resume-prompt.md)의 전체 내용을 실행 중인 Claude Code에 전달한다.
이 프롬프트는 별도 인터뷰를 다시 시작하는 대신 기존 확정 Seed·현재 WIP·실패를 읽고
바로 구현을 이어가도록 한다. Codex에서는 구현을 재개하거나 Ouroboros worker를 시작하지 않았다.

현재 설치된 Ouroboros 스킬 안내를 확인하여 다음 운영 기준을 추가했다:

- `ooo help`로 Claude 런타임의 실제 기능/인자를 먼저 확인한다.
- `ooo resume-session`/`ooo status`는 기존 상태·worker·worktree를 확인하는 데 사용한다.
  옛 실패 세션을 자동 resume하지 않는다. Claude 자체 `/resume`와 Ouroboros의
  `resume-session`은 다른 기능이다.
- 기존 validated Seed를 재사용한다. 새 기획·인터뷰·Seed 생성으로 승인 범위를 재설정하지 않는다.
- 미커밋 snapshot 하위 단계는 먼저 현재 working tree에서 원인을 조사·검증한다.
  `ooo qa`는 실제 하위 동작 증거를 보조 평가하며 `ooo unstuck` 또는 설치 버전의
  `ooo lateral researcher`로 미확정 원인을 조사한다. 범위 축소는 임의 채택하지 않는다.
- 단계가 검증·커밋되어 입력이 고정된 후 `ooo run`과 지원되는 Ralph 반복을 활용한다.
  실제 cwd/project_dir/worktree와 WIP 포함을 확인하고 같은 파일의 이중 구현을 막는다.
- 현재 Ralph 스킬은 새 실행에 validated `seed_content`와 별개 `lineage_id`를 요구한다.
  기존 lineage continuation은 정확한 live identity와 명시적 재개 의도를 확인한다.
  단순 `ooo ralph <요청문>`을 Seed 또는 lineage로 보내지 않는다.
- `ooo qa`는 전체 정식 평가를 대체하지 않는다. 마지막에는 현재 실행과 최신 독립 live 증거로
  `ooo evaluate`를 수행한다. QA pass/generation 종료/timeout으로 전체 완료를 표시하지 않는다.

관련 안내는 이 환경의 `/Users/0xtopaz/.codex/skills/ouroboros-help/SKILL.md`,
`ouroboros-run/SKILL.md`, `ouroboros-ralph/SKILL.md`, `ouroboros-qa/SKILL.md`,
`ouroboros-resume-session/SKILL.md`, `ouroboros-status/SKILL.md`에서 확인했다.
Claude Code에서는 자신의 설치 스킬과 실제 MCP schema를 확인한다. Codex 전용 tool 이름이나
경로가 그대로 동작한다고 가정하지 않는다. 아직 Claude Code에 직접 입력하거나 실행하지 않았다.

## Claude Code 재개 기록 (2026-10-09)

- Ouroboros 상태: `orch_0dce4e2ea49d` failed, Ralph 세대 실행 cancelled, `job_efbac5e0f3a6` failed.
  실행 중인 worker가 없어 자동 재개나 이중 구현 위험이 없음을 확인했다.
- snapshot native 실패 원인: 실패 fixture의 `d/logs/22e68f682c78/node*.log`에
  `Low disk space. Gracefully shutting down Geth … available=428.13MiB`가 네 노드 모두 남아 있다.
  노드가 스스로 종료했고 관측 API는 running으로 보고하지 않았다. 제품 결함이 아니므로 단언을 유지했다.
- `browser_snapshot_restore.mjs`가 재시작 전후 `observations-*.json`과 기록 PID의 `ps` 행을 남긴다.
- 여유 공간 11 GiB에서 같은 소스로 두 번 통과했다. 실패 fixture root는 삭제하지 않고 그대로 보존했다.
- WEB-03/04/10 Python receipt 검사는 SPA 재빌드로 옛 receipt가 stale이 되어 실패한다.
  현재 소스로 새 capture를 만든 뒤 다시 확인한다.
- 커밋 `cb74142e`: snapshot·bounded replay·재연결/재시작 복원 하위 단계. 고정 소스에서 native 증명 2회와 필수 회귀를 통과했다.
- 커밋 `558014af`: 노드 metric 수집·SVG 차트·시점 연계 로그·만료 없는 archive(`internal/app/web_monitor.go`·`web_monitor_read.go`·`web_monitor_log.go`,
  `internal/core/session/observationstore.go`, `web/src/NodeMetrics.svelte`). `run_monitoring.py` native 증명 2회와
  snapshot 회귀, WEB-03/04/10 새 capture, make check, race, build, JS 18개, Python 30개가 digest
  `5569e6941e70ae0c51934c5d3c3150fb708a2d01ce2e73143b5ed8f9c29e8981`에서 통과했다.
  `cb74142e`가 깨뜨린 OpenAPI YAML도 `TestWebOpenAPIParses`를 추가해 고쳤다.
- 남은 WEB-08/09 범위: SSH 노드 로그 수집(현재 `remote_log_collection_unavailable` gap), Run 단위 archive 삭제,
  겹치는 노드 선의 시각 구분, collector context 취소. 전체 인수는 여전히 0/14다.
- 사용자 결정(2026-10-09): ooo run/Ralph는 시작하지 않고 이 세션이 단일 구현자로 진행한다. SSH 노드 로그는
  운영자가 시작한 `network.monitor` job이 본인 binding으로만 수집한다(명세 반영).
- 커밋 `40c8adc2`: SSH 노드 로그 수집 job. `run_ssh_log_collection.py` 2회와 회귀 전체가 digest
  `e7b59f6f5782df3e79b9d12b51042acb7d0d1d2fe6c198fa3e431ae00bdac3ce`에서 통과했다.
  다음 후보: Prepare 부정 사례 테스트, 멈춘 원격 명령 시험, Run 단위 archive 삭제(WEB-09),
  그리고 9개 기준(WEB-02·06·07·08·09·11·12·13·14)의 `verify.sh` runner.
- 커밋 `e3449e3f`: 히스토리 Run의 archive 구간 표시·`runId` 조회·관리자 삭제(실행 소유 구간만, 부분 실패 시 Run 보존).
  `run_history_archive.py` 2회와 회귀 전체가 digest `4bf6c1df47ec89e0bf9657ac7c656d32dccba478367caecf795cbd2da4238421`에서 통과했다.
- 커밋 `2a175cd6`: WEB-08 인수 runner(`run_web08.py`, `evidence_web08.py`)와 로그아웃 지속 증명. capture와 독립 재현이
  digest `04d580d7da4e9b29e8cc2543e17fca6f6f15d88dddaa8ae3ee88e08bb584ae19`에서 통과했다. 판정은 WEB-14 종합 때 한다.
- `/private/tmp/cbui`에 fixture runtime이 많이 쌓였다(약 14 GiB 이상). 삭제는 사용자 확인 전까지 하지 않는다.
- 커밋 `23c143e2`: 선택 정리가 실행 중 노드를 먼저 정지(이전에는 항상 cleanup_failed). `25d4527c`: WEB-09 runner와 공통
  `proof_acceptance.py`. WEB-08·09 capture와 독립 재현이 digest `6a2c356e882f12da34552ae892d09351612cf33c0201fadf17fb691b6922c466`에서 통과했다.
- 남은 runner: WEB-02·06·07·11·12·13·14. 기존 증명 공백: attach 거절, 독립 자원 병행, 실행 중 테스트와 외부 제어 충돌,
  배포·초기화 중 재시작, 사용자/관리자 취소×유지/정리, 정리 실패, 계정 철회, SSH 정리. WEB-13 공백 증명 초안은
  scratchpad의 `browser_cancel_retention.mjs`(아직 저장소 밖)에서 시작했다.
- 커밋 `f23bd071`(attach 화면 설명), `4b736df1`(WEB-13 runner). WEB-13·08·09 capture와 독립 재현이 digest
  `be1ba94de213414eb63f5b9e6bc9b1b35e5bb579030e7cc7f570dee7f32d80ff`에서 통과했다. WEB-13 검증에는
  `WEBUI_REPLACEMENT_BINARY=/private/tmp/chainbench-web-replacement-native/gwbft`가 필요하다.
- 결정 대기: 보안 계층이 모든 API 오류 본문을 일반 문구로 바꾸는 정책과 WEB-02의 구체적 필드 오류 요구.
- 2026-10-09 사용자 승인으로 이 세션이 만든 `/private/tmp/cbui` runtime 중 게시 증거·receipt·독립 재현이 참조하지 않는
  85개(약 13 GiB)를 삭제했다. 이전 세션의 23개(삭제 거절된 Codex root 5개 포함)와 참조된 32개는 남겼다.
  삭제 뒤 WEB-03/04/08/09/10/13 검증기가 모두 다시 통과했다.
- 커밋 `be04ad9a`: WEB-07 runner. capture와 독립 재현이 digest `2f63980063a304638c2e77c4d718af02e2394258a5112f636ffdf7b0726a7121`에서 통과했다.
- 커밋 `dd08cf09`(같은 장비의 같은 노드 바이너리를 계획 단계에서 차단; 메시지가 304자로 규칙 300자를 넘음, push 뒤라 그대로 둠),
  `15e26d2e`(WEB-06·11·12 runner). 전체 고정 주기(digest `aa9de4ef5624bd9a419cce0de336eefb12a7cbdfbcd46e5d6baf5f88394b6c61`)에서
  WEB-03/04/06/07/08/09/10/11/12/13 capture·독립 재현 통과, WEB-01 재현 실패(제외된 `.ouroboros` manifest 참조), WEB-05 runner 미완성.
- 사용자 결정(2026-10-09): 400/409/422 검증 응답은 비밀 제거 후 엔진의 구체 메시지를 보여 준다. 401/403/500은 일반 문구 유지.
- 2026-10-09 승인 규칙으로 미참조 세션 runtime 21개를 추가 삭제했다(목록 /private/tmp/chainbench-web-cleanup-2.log). 삭제 뒤 WEB-03/04/06~13 검증기가 모두 통과했다.
- 2026-10-09 승인 규칙(더 새 재현이 있는 사본 포함)으로 recheck 사본 11개와 runtime 34개를 삭제했다(목록 /private/tmp/chainbench-web-cleanup-3.log). 고정 주기 중 정리 목록은 /private/tmp/chainbench-web-full2-cleanup.log.
- 커밋 `32aa8140`(bundle 가져오기·내보내기와 비밀 제거한 검증 사유), `4a27ac72`(WEB-02 runner, WEB-01 재현의 선택 바이너리 manifest 전달).
  전체 고정 주기 full2(digest `4aea5930ed13d2dd31a21773ee0021343ffb20327e76a2cfea95ee6b5a9c2eda`)에서 WEB-01~04·06~13 capture와 독립 재현,
  make check·race·build·JS·Python이 통과했다. WEB-05만 실패한다(실행 coverage 미완성).
- 사용자 결정(2026-10-10): DSL 편집 계약은 각 명령의 구현과 해석기가 실제로 읽는 인자만 노출한다. 읽지 않는 인자는 Web 편집기와
  `chainbench validate`가 이름을 대며 거절하고, 그런 인자를 쓰는 corpus 케이스는 고친다. WEB-05 분모는 그렇게 좁힌 인자 집합이다.
- 커밋 `964d311e`·`d09b5bb2`·`ff8badf3`: 명령이 읽지 않는 인자와, 문장이 타지 않는 경로에서만 읽히는 인자를 `validate`·실행·편집기가
  이름을 대며 거절한다. `9b0cfcda`·`54b6600b`·`4140d74a`: 실패 기대를 probe 구간 전체로 판정하고, 모든 builtin 인자 경로를 실행하는
  stablenet 케이스를 추가하고, 노드 로그를 기록된 경로에서 읽는다. `8839cd1e`·`c400486b`: 저장한 케이스가 chain preset revision을
  고정하고, WEB-05가 브라우저 편집, corpus 왕복, native Web job 두 개로 실행된다.
- 전체 고정 주기 full3(`c400486b`, digest `3f7281e22e2109b5845d10455b49db03a69500befc75658e38ae23d934f27437, digest 유지`)에서 WEB-01~13 capture와
  독립 재현, make check·race·build·JS·Python이 모두 통과했다. 이때 WEB-05의 attach 케이스 6개와 선언 계정 케이스 1개는 아직 거절 대상이었다.
- 외부 바이너리 관찰: gstable은 살아 있는 peer와 handshake하는 도중 시작에 실패하면 `Fatal`을 찍지 않고 멈춰 있다. peer가 없으면 사유를
  찍고 바로 끝난다. 체인 소스는 고치지 않았고, 노드 제어 케이스가 실패할 시작 전에 다른 노드를 먼저 멈추도록 했다. 상위 저장소에 보고할 만하다.
- 커밋 `2c0fed6d`: race 실행이 건너뛰던 live 테스트 26개를 실제 바이너리로 돌려 보니, composed 실행이 이미 충전한 선언 계정을 빈 송신자로
  다시 충전해 실패했다. 실패하는 테스트를 먼저 쓰고 고쳤다.
- 커밋 `f16ec75c`: WEB-14 runner. 같은 source digest에서 WEB-01~13을 각자의 검증기로 다시 판정하고, live 관문을 모두 연 race,
  e2e vet, SPA 재빌드, JS·Python을 돌린다. Go 테스트가 하나라도 skip되면 미완료로 판정한다. 재빌드한 SPA가 내장 SPA와 바이트 단위로 같고
  Go 서버가 그 파일을 그대로 내보내는지, 여섯 경로를 탐색·새로고침·직접 열기로 확인한다.
- Docker 시험 서버: `SERVERS=1 CHAIN_SRC_ROOT=/Users/0xtopaz/work/github/0xmhha/chain ./setup.sh check|generate|image|up|binaries go-stablenet`
  (`env/docker`)로 띄웠다. 비밀번호는 무작위이고 gitignore된 `env/docker/accounts.env`(mode 600)에만 있다. WEB-14는
  `CHAINBENCH_DOCKER_SERVERS=$PWD/env/docker/build`를 받는다. 컨테이너 `chainbench-server1`이 계속 떠 있다. live 환경 변수는 WEB-14의
  직렬 race 실행에만 넘긴다. 병렬 make check에 넘기면 기본 포트의 live 네트워크가 chainsetup 단위 테스트의 포트 검사와 부딪힌다.
- 사용자 결정(2026-10-10): Web의 attach 케이스는 Web이 직접 배포하고 기록한 네트워크에만 붙는다. RPC는 기록된 배포에서 가져오고,
  읽기 전용으로 실행하며 노드를 제어하지 않는다. `keyFile` 계정은 사용자 본인의 암호화 자격증명(`account-key`)에서 키를 읽는다.
  외부 testnet을 가리키는 corpus 케이스는 Web에서 계속 거절한다.
- 커밋 `5b584327`·`a9bf05ed`: 그 결정대로 `test.attach` job을 구현했다. 실행 때 키를 `account-keys/run-*` 아래 0600 파일로 쓰고 끝나면
  지운다. 자격증명을 철회하면 job이 취소된다. composed Web 실행은 선언 계정을 메모리 key ring에서 만들고 충전하며, attach 엔진에 그 ring을
  넘겨 다시 충전하지 않는다. `8114df3f`: 저장한 케이스를 고정한 preset revision과 함께 bundle로 내보내고, 가져올 때 같은 bundle의 preset에
  연결한다. bundle에 없는 preset은 이름을 대며 거절한다.
- 전체 고정 주기 full4(`8114df3f`, digest `d2ca800bed96a2dcf1e3e8e71018478a7814af6803078ea60500c06716a9f9c9`, digest 유지)에서
  WEB-01~10·12·13 capture와 독립 재현, make check·race·build·JS가 통과했다. WEB-11은 시험 스크립트의 경쟁 조건으로 실패했다.
  구성 중인 `chain-record.json`에는 아직 `nodes`가 없는데(`omitempty`) `browser_resource_parallel.mjs`가 바로 `nodes.length`를 읽었다.
  WEB-14는 WEB-11을 다시 판정하다 멈춰 필수 검사를 돌리지 못했다. Python은 WEB-11 receipt 검사와, attach 단계가 늘어 약 9분 43초가 된
  WEB-05 재현을 600초로 끊던 `test_web05_isolation.py` 때문에 실패했다.
- 커밋 `aa3f2e1e`: 그 두 시험을 고쳤다. 전체 고정 주기 full5(digest `f4f466c6d223faca56e86f21da77d4218818bbe6083f311f6131c237c555fdb1`, digest 유지)에서
  WEB-01~13 capture와 독립 재현, WEB-14 capture, make check·race·build·JS·Python이 통과했다. WEB-14 capture는 live 관문을 모두 연 race와
  e2e vet, SPA 바이트 일치, 여섯 경로 확인까지 통과했다. WEB-14 독립 재현만 실패했다. 재현은 `chainbench-out`이 빠진 staged 사본에서 돌아가는데,
  각 기준의 검증기가 작업 공간 기준 상대 경로를 현재 디렉터리에서 찾아 WEB-01 artifact를 모두 잘못된 경로로 판정했다.
- 커밋 `c5f23d85`: WEB-14가 각 기준을 그것을 게시한 작업 공간에서 판정한다. 다른 디렉터리에서 실행하면 실패하는 단위 테스트를 먼저 썼고(RED),
  고친 뒤 게시된 열세 기준이 무관한 디렉터리에서도 모두 통과함을 확인했다.
- 전체 고정 주기 full6(`c5f23d85`, digest `a0fcd339ca22ba0f098e85df1414f704d6a721c07a066d963c4372dd1363407f`, digest 유지)에서 WEB-01~14의
  capture 14개와 독립 재현 14개가 모두 통과했고, 이어서 make check·race·build·JS(28개)·Python(52개)도 통과했다. WEB-14 증거에 따르면 열세 기준을
  각자의 검증기로 다시 판정해 모두 통과했고, live 관문을 모두 연 `-race`에서 Go 테스트 2858개가 skip과 실패 없이 통과했다. SPA 파일 3개는
  바이트 단위로 같게 재빌드됐고 여섯 경로가 모두 열렸으며, personal·team 방식과 local·SSH 전송을 모두 관측했다. 결과는 `chainbench-out/web-ui-acceptance/WEB-*`에 있다.
- 최종 `ooo evaluate`(job `job_47b683dafbe4`, session `orch_0dce4e2ea49d`, Seed `seed_2a20eeab7a55`, auto_evolve 끔): **APPROVED**.
  Stage 1에서 lint·build·test·vet·cover가 모두 통과했고, Stage 2는 점수 0.85, AC 준수 YES, 목표 정합 0.84, drift 0.12였다. Stage 3 합의는 돌지 않았다.
- `docs/dev/web-ui-worklist.json`의 열네 항목은 아직 `in_progress`다. arch 테스트는 `complete`인 항목의 RED·GREEN·증거 파일이 저장소에 있기를
  요구하는데, live 증거는 gitignore된 `chainbench-out/`(약 29 MB)에만 있다. 증거를 커밋할지, 상태 표기 방식을 바꿀지는 사용자 결정이다.
- 사용자 결정(2026-10-10): 요약 증거를 커밋해 열네 항목을 완료로 표시한다. 커밋 `c765a36b`: 기준별 `result.json`·`evidence.json` 사본과
  RED·GREEN 기록을 `docs/dev/web-ui-evidence/WEB-*`에 두고 worklist를 `complete`로 바꿨으며, 저장소 열린 작업 수를 11에서 10으로 낮췄다.
  비밀 패턴, 고정 키, Docker 계정 값이 증거에 없음을 확인했다. WEB-04에는 기록된 제품 RED가 없어 그렇게 적었다.
  이 커밋 메시지는 318자로 규칙 300자를 넘는다. push 뒤에 알아서 그대로 두었다.
- 2026-10-10 PR #443이 main(#444~#447)과 충돌했다. main을 merge해 풀었다(`857286ec`, rebase나 force-push 없음). `package-tree.md`의 줄 수를
  다시 계산했고, `swapNode args`를 인자 계약과 편집기에 넣었으며, 새 mixed 케이스 세 개에서 엔진이 읽지 않는 `swapNode purpose` 11개를 지웠다.
- Linux CI가 Mac에서 보이지 않던 결함 둘을 드러냈다. 이 브랜치의 CI는 오래전부터 vet 단계에서 실패하고 있었다. 커밋 `8b6f67f4`: 프로세스 관측기의 `_darwin`
  파일 이름 때문에 Linux에서 `internal/app`이 컴파일되지 않았다. 커밋 `5f9bb2c4`: Linux가 지운 파일의 inode 번호를 바로 다시 쓰면 로그 교체를 놓쳤는데,
  이제 파일 앞 256바이트를 함께 비교한다. 그 뒤 CI의 lint·test·secrets·shell이 모두 통과했고 PR은 MERGEABLE CLEAN이다.
- 인수 증거는 `c5f23d85` 기준이다. 병합과 수정 뒤 소스로 다시 인수하려면 전체 고정 주기를 다시 돌려야 한다(사용자 결정 대기).
- 2026-10-11 병합한 소스 `91dc9336`에서 전체 고정 주기(full7)를 다시 돌렸다. 소스 digest `37b8892f…`가 처음부터 끝까지 같았고, 열네 기준의
  capture와 독립 재현 28회가 모두 통과했다. 주기 뒤 make check, race, build, JS 28개, Python 53개도 통과했고, WEB-14는 race Go 테스트 2865개를 skip 없이 돌렸다.
- 요약 receipt와 green.md를 full7 기준으로 바꾸고 WEB-05·07·08 red.md에 병합 뒤 RED를 더해 커밋했다(`ecd01797`). 작업 목록 두 곳도 `91dc9336`을 가리킨다.
- 최종 `ooo evaluate` 첫 시도(job_32d37b8c55c8)는 평가기의 Claude 하위 프로세스가 끝나며 SessionEnd hook이 취소되어 판정 없이 멈췄다. 같은 입력으로 다시
  돌린 job_6be7ef1cf944는 mechanical 검사를 통과하고 열네 기준을 모두 통과로 판정했다. PR #443 설명을 이 결과로 고쳤고 draft와 리뷰어 없음을 유지했다.
- 2026-10-11 사용자 결정: 인수가 끝나 Docker 컨테이너 `chainbench-server1`을 멈췄다(삭제하지 않음, `docker start chainbench-server1`로 다시 띄운다).
  300자를 넘긴 `c765a36b` 커밋 메시지는 이미 push했으므로 고치지 않고 둔다. 새 push의 CI lint·test·secrets·shell은 모두 통과했다.

## 2026-10-11 계획 대비 누락 검토와 재개

사용자 요청으로 문서와 코드를 대조했더니 인수 통과와 별개로 Seed·명세·OpenAPI·화면 디자인의 누락 42개가 나왔다.
`249fd11c`에서 `docs/dev/web-ui-gaps.json`(상태 정본)과 `docs/dev/web-ui-gap-review.md`를 만들고 열네 기준을 모두
`in_progress`로 되돌렸다. `internal/arch/web_gaps_test.go`가 열린 누락이 가리키는 기준의 complete 표시를 거절한다.
PR #443 본문도 다시 연 상태로 고쳤다(draft 유지, 리뷰어 없음). 사용자 지시: 1~4 정리 후 제안 순서(Seed → 명세 → 화면,
결함은 같은 파일 묶음에서)로 구현한다. 지시의 6번은 내용이 비어 있었다.
- 2026-10-11 `4b524de7`·`6b8b2a42`: ChainSurface 카탈로그(`internal/core/chainsurface`, `GET /api/v1/chains/{chainId}/surface`), 계획의 surface 고정과 실행 전 재확인, 옵션 값 범위 검사, 격리된 바이너리 확인. G-A01·A03·A04·D07·D08 닫음, G-A02는 실제 계획 응답 확인 전까지 열림.
  확인 중 숨김 플래그 질의가 `~/Library/Ethereum/gstable`·`gwemix`를 만들었다(15:13~15:14). 지우지 않고 `/private/tmp/chainbench-probe-leftovers-20261011`로 옮겼다. verify-docs.sh도 같은 위험이 있어 고쳤다.
- 2026-10-11 누락 진행: 닫음 19개(A01·A03·A04, B01·B02·B03·B05·B06·B07·B09·B10·B19, D01·D02·D03·D05·D06·D07·D08). 여러 서버 배치(A05)와 노드별 제어(A06)는 코드와 단위 테스트까지 커밋했고(`3152e8ba`, `808d243e`), 실제 두 서버 확인은 인수 재실행에서 한다. 그 확인에는 Docker server2와 같은 네트워크 안의 제어 서버 컨테이너가 필요하다(macOS는 컨테이너 주소로 라우팅하지 못한다).
- 2026-10-11 주의: `tests/webui` Python 테스트 중 "latest live receipt" 11개는 저장된 인수 결과물의 소스 digest를 검사하므로, 이번 누락 작업으로 소스가 바뀐 뒤 "artifact changed"로 실패한다. 마지막 고정 소스 인수 재실행으로 새 결과물을 만들어야 다시 통과한다. 나머지 42개는 통과한다.

## 2026-10-11 진행 기록 (D04 코드, C05 설정 화면)

- 0285ba2a: `-deployment-accounts` 플래그와 Basic 계정 경로, 편집기 세 곳의 Basic 로그인 폼을 없앴다. 설정 화면에 본인 자격증명(연결 서버, 진행 중 작업 수, 접근 확인, 폐기), 관리자용 사용자 관리와 감사 기록을 두었다. 역할과 자격증명 종류는 `GET /api/v1/contracts/access`에서 받는다.
- D04 RED 로그 `/private/tmp/chainbench-web-d04-red.log`(옛 바이너리가 CSRF 없는 Basic 쓰기에 201), GREEN 로그 `/private/tmp/chainbench-web-d04-green.log`(새 바이너리는 플래그를 거절하고 기본 모드의 Basic 쓰기에 401).
- G-D04와 G-C05는 아직 열려 있다. run_web03.py, browser_web03.mjs, run_web04.py, browser_web04.mjs가 여전히 `-deployment-accounts`와 영어 라벨을 쓴다. 이후 UI 단계에서 라벨이 더 바뀌므로 증명 갱신 단계에서 세션 로그인으로 함께 옮긴다. C05는 브라우저 증명(폐기 시 작업 철회, 운영자에게 관리 화면 없음)이 남았다.

## 2026-10-11 진행 기록 (C10, C02, C06, C07, D09, D10)

- a37327c4/0315fa7f: 선택지(작업 종류, 역할, 자격증명 종류, 노드 역할, syncMode)를 계약에서 받는다. G-C10 닫음.
- f248fef8: help 검사 출력의 임시 HOME을 $HOME으로 바꿔 surface 지문이 반복 가능해졌다(G-D09). 4b524de7 이후 화면에서 검토한 계획을 시작하면 409가 났다.
- 2df3278e: 정리에 성공한 작업이 지운 자원을 더 이상 미해결로 보이지 않는다(G-D10).
- 45b2a68e: 작업 상세(/jobs/<id>), 실시간 대시보드, 상단 바, 화면 간 workspace 선택. 증명 `tests/webui/run_navigation.py`(실제 wbft). RED/GREEN 로그 `/private/tmp/chainbench-web-nav-{red,green}.log`. G-C02, G-C06, G-C07 닫음.
- 남은 UI 단계: C01·C03·C13, C04·C12, C08·C09, C11. 그 뒤 B04, 다중 호스트(A02·A05·A06), 증명 갱신(D04 포함), 고정 소스 인수 재실행.

## 2026-10-11 진행 기록 (UI 여섯 단계 완료)

- 닫음: C01–C04, C06–C13, A02, D09, D10 (각 단계 커밋과 증명은 web-ui-gaps.json의 evidence). 새 개발 증명: run_navigation.py, run_node_detail.py, run_monitoring_range.py, run_dsl_editor.py, run_forms.py.
- 남은 gap: A05, A06(Docker 다중 호스트 live), B04(OpenAPI·라우트 동기화 테스트), C05(설정 화면 브라우저 증명), D04(WEB-03/04 증명을 세션 로그인으로 이전).
- 기존 증명 다수가 영어 라벨과 옛 화면 구조를 쓴다. 고정 소스 인수 재실행 전에 갱신해야 한다.
- 주의: 명령을 `&&`로 잇다가 중간에 `;`를 섞으면 실패 뒤에도 commit이 돈다. 065161c4가 그렇게 잘못 만들어져 로컬에서 amend로 바로잡았다(ddc70fe7).

## 2026-10-12 누락 48개 종료 뒤 최종 고정 주기

- 사용자 결정(2026-10-11): 지금 구조로 결함을 고쳐 PR을 merge한 뒤, Web 실행 경로를 chainbench CLI(또는 MCP) 호출 구조로 바꾸는 리팩토링을 따로 설계한다.
  Web이 `internal/app`을 프로세스 안에서 직접 불러 Web 전용 조율 코드(약 17,600줄)가 생긴 점을 사용자가 구조 결함으로 판단했다.
- 고정 주기에서 나온 결함과 수정. 각 수정은 실패하는 테스트를 먼저 확인했다.
  - `a053fd7a`: fork로 bp가 늘어난 네트워크의 노드 제어·정리 거절.
  - `34677851`: corpus case가 en1이 블록을 받기 전에 영수증을 읽던 경합(확인 도구).
  - `f24b7dfc`: 작업 양식 초안이 지난 작업의 "정리"를 되살림.
  - `0200bf33`: 앞 수정이 검증자 수 입력 오류를 422에서 409로 바꾼 회귀.
  - `39b341ca`: SSH 노드 관찰이 8초 제한을 넘김(명령마다 새 SSH 연결). 30초로 늘림.
  - `107bf225`: 390px 화면에서 모니터링 선택 상자 넘침. 첫 수정은 우연히 통과했고 독립 재현에서 다시 실패했다.
  - `34c951a8`: fixture 재시작 응답 파일을 다 쓰기 전에 읽던 경합(확인 도구).
- 실행 도구 실수: WEB-14 실행에 `GWBFT_BIN`을 빠뜨려 race 테스트 하나가 skip되었고 WEB-14가 정상적으로 거절했다.
  독립 재현 중 디스크가 229 MiB까지 줄어 genesis 사전 검사가 거절했다. 2 GiB 검사는 유지했고, 승인된 규칙대로 미참조 fixture runtime 155개(35.5 GiB)를 지웠다.
  재부팅으로 사라진 scratchpad의 `cbui_cleanup.py`를 다시 썼다.
- 최종 결과: 커밋 `34c951a8`, 소스 digest `470ab4bb…`가 처음부터 끝까지 같았다. 열네 기준의 capture 14회와 독립 재현 14회가 모두 통과했다.
  그 뒤 make check·build·JS·Python(54개)이 통과했다. 영수증과 RED·GREEN 기록, 작업 목록 완료 표시는 `0ec93b47`에 커밋했다.
- 재부팅으로 `/private/tmp`의 이전 RED 로그가 지워졌다. 누락 목록의 `evidence`가 가리키는 그 로그는 이제 커밋과 테스트로만 다시 확인할 수 있다.
- 공식 `ooo evaluate`를 job `job_4db7d69a3c43`(session `orch_0dce4e2ea49d`, auto_evolve 끔)으로 시작했다. 결과는 아래에 덧붙인다.
- push와 PR 본문 갱신은 아직 하지 않았다. Docker 컨테이너는 모두 멈춰 있다.
- 공식 평가 결과(job `job_4db7d69a3c43`): **APPROVED**. Stage 1의 lint·build·test·static·coverage가 모두 통과했다. Stage 2는 점수 0.86, AC 준수 YES,
  목표 정합 0.85, drift 0.10, 불확실성 0.27이었고 Stage 3은 돌지 않았다. 평가기가 지적한 `presets/keys`의 평문 개인키는 main의 #422에서 온
  테스트 전용 키이고(README 394행), 이 브랜치가 추가하지 않았다.

- 2026-10-12: 사용자 지시로 `.github/CODEOWNERS`(`* @0xmhha @onlyhyde`)를 `0394a44f`로 커밋하고 브랜치를 push했다. GitHub 검증은 onlyhyde에 쓰기 권한이 없어 "Unknown owner"를 보고한다. PR #443 본문을 최종 주기 결과로 갱신했고, draft와 리뷰 요청 없음 상태를 유지했다. CI(lint, shell, test, secrets)는 `0394a44f`에서 통과했다.
- 2026-10-12: 사용자 지시로 onlyhyde는 초대하지 않고 CODEOWNERS를 `* @0xmhha`로 바꿔 `46f9afb0`로 커밋, push했다. GitHub 검증 오류는 없다.
- 2026-10-12: 사용자가 PR #443을 squash merge했다(main `b9e5be39`). 로컬을 main으로 전환해 fast-forward했고, main 트리는 검증한 `46f9afb0`과 같으며 main에서 make check와 make build가 통과했다. 다음 작업은 Web 실행 경로를 CLI 또는 MCP 호출로 바꾸는 리팩토링의 결정 문서이며, 승인 전에는 구현하지 않는다.
- 2026-10-12 리팩토링 진행: 브랜치 `refactor/web-cli-adapter`. P0, P1, P2 완료(결정 문서에 커밋별 기록). 다음은 P3(체인 구성, 매니페스트 setup, 테스트 계획과 바이너리 업로드를 CLI로).
- 2026-10-12: P3 대부분 완료(체인 구성, 매니페스트 setup이 chain up 경유). 남은 P3은 테스트 계획과 테스트 바이너리 업로드, 다음은 P4 노드 제어.
- 2026-10-12: P4 완료(노드 제어, 정리, 교체 바이너리 배치가 CLI 경유, CLI 코어 신원 확인). 남은 일은 읽기 전용 경로(노드 관찰 등)의 CLI 이동과 P5 최종 인수.

## 2026-10-12 이후: Web CLI 전환 리팩토링

PR #443은 main에 squash merge되었다(`b9e5be39`). 이어지는 작업은 브랜치 `refactor/web-cli-adapter`의 Web 실행 경로 CLI 전환이며, 인수인계는 `docs/dev/web-cli-refactor-handoff.md`, 결정과 단계 기록은 `docs/dev/architecture/web-execution-adapter-decision.md`를 먼저 읽는다. 이 문서의 이전 내용(codex/web-ui, PR #443 진행)은 이력이다.
