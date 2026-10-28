Chainbench Web UI의 기존 확정 작업을 Codex에서 인계받아 이어서 구현해 줘.
아래 인계 자료를 실제로 읽고 현재 소스·Git·실행 증거와 대조한 뒤 작업을 재개해.

[작업 위치와 현재 상태]
- 프로젝트: /Users/0xtopaz/work/github/0xmhha/chainbench
- 현재 브랜치: codex/web-ui
- 마지막 검증·push 커밋: 7c8de8762e269b4a7b2156a18743d3b26968c2e5
- 기존 draft PR: https://github.com/0xmhha/chainbench/pull/443
- 이후 snapshot/replay/모니터링 구현은 미커밋 상태다. 현재 파일을 보존해.
- 부분 구현과 여러 실제 native 검증은 있으나, 전체 인수 완료는 0/14다.
- 사용자는 Codex 구현을 중단하고 이 Claude Code 세션으로 작업을 넘겼다.
  이제 여기에서 구현을 재개하되, 다른 worker와 같은 파일을 동시에 수정하지 마.

[먼저 읽을 자료]
프로젝트 루트 기준으로 다음을 읽어. 누락된 자료는 존재하지 않는다고 알리고
소스·정본·보존 로그로 복구해. 읽지 않은 것을 읽었다고 가정하지 마.
1. CLAUDE.md
2. docs/dev/web-ui-claude-handoff.md — 현재 상황·실패·코드 지도·재개 절차의 상세 인계
3. docs/dev/architecture/chainbench-web-ui.seed.yaml — 이미 확정·검증된 Seed
4. docs/dev/web-ui-spec.md, docs/dev/web-ui.openapi.yaml
5. docs/dev/web-ui-acceptance.md, docs/dev/web-ui-worklist.json,
   docs/dev/web-ui-worklist.md — 14개 전체 완료 조건과 단계별 RED/GREEN
6. docs/dev/web-ui-design.md, docs/dev/web-ui-manifests.md,
   docs/dev/dashboard-metrics-design.md, docs/dev/web-ui-implementation.md
7. docs/chain-analysis/{gstable,gwbft,gwemix}의 cli-surface.txt,
   cli-flags.txt, cli-graph.md, rpc-metrics-graph.md — 세 체인 × 네 자료
8. .ouroboros/web-ui-analysis/interview-ledger.json,
   manual-continuation.json의 최신 claudeHandoff와 누적 근거,
   recovered-snapshot/summary.json·go-graph.json·source-graph.json·native-svelte-graph.json
9. tests/webui/README.md, 현재 작업의 실제 source/test/fixture/runner와 실패 로그

확정 요구·검증 계약과 현재 코드가 기준이다. 오래된 journal 상태, 구현 문서의 과거 문단,
Seed QA 점수, receipt 파일 존재만으로 현재 구현을 완료라고 판단하지 마.
AST 그래프는 전체 코드 구조·의존·분류를 파악하는 근거이며 동작 증명이 아니다.
대상 경로·등록·interface dispatch는 실제 소스를 열어 대조하고, 변경 후 그래프를 갱신해.

[확정된 목표]
첫 출시에서 아홉 요구 모두를 브라우저에서 완료해야 한다:
체인 구성·설정 import/edit/export·서버/workspace·플러그인 매니페스트·현행 전체 DSL 작성,
실제 local/SSH 배포·소유 노드 시작/정지/설정과 바이너리 교체/재실행/허용 초기화,
상태·이벤트·노드별 metric 차트·시점 연계 로그, 히스토리 검색/상세/호환 비교/export/관리자 삭제.
개인 PC와 팀 서버 모두, 공유 revision과 암호화된 개인 SSH, 자체 계정과 세 역할을 지원해.
엔진 계약과 실제 바이너리 지원의 교집합만 선택하게 하고 현행 CLI/MCP/DSL 의미를 보존해.
attach·생산자 초기화 제한, 물리 자원 충돌, 기본 노드 보존, 철회 후 새 SSH/정리 금지,
서버 재시작 시 interrupted·명시적 재실행, 비밀 격리 정책은 확정되어 있다.
전체 DSL 전수·metric/log archive 등 미구현 범위를 생략하거나 출시에 필요한 범위를 축소하지 마.

[첫 재개 지점: 아직 실패 중인 snapshot 복원 단계]
- 새로운 /api/v1/snapshot, /api/v1/events, 128개 bounded replay,
  viewer 인증·scope·revocation·generation/version/gap 판정·프론트 복원 코드가 미커밋이다.
- make check, app/dashboard race, Go/SPA build, 실제 TCP HTTP SSE와 JS 13개 검사는 통과했다.
- 그러나 tests/webui/run_snapshot_restore.py의 전체 native 브라우저 검증은 실패했다.
- 로그: /private/tmp/chainbench-web-snapshot-browser-green.log
  파일 이름의 green은 통과 판정이 아니다.
- 마지막 실패: live.nodes.every(n => n.state === 'running').
  정확한 node state/reason을 저장하지 않아 원인은 미확정이다.
- 조사 자료: chainbench-out/web-ui-development/snapshot-restore,
  /private/tmp/cbui/b99b880a-c444-4f0b-b8cd-90f6e82bdbd1.
- 기존 계정 세션은 의도적으로 서버 재시작 시 만료된다.
  fixture는 실제 로그인 UI로 재인증하도록 이미 수정했다. 이 인증 정책을 완화하지 마.
- 실패 전에 observation JSON·PID/argv·ledger·target·권한·관련 로그를 보존하는 진단부터 추가해.
  disk/권한/프로세스 종료라고 추측하거나 단언을 삭제하여 통과시키지 마.
- 기록 PID/cached snapshot은 live proof가 아니며 조회가 control 실행/자동 reconcile을 유발하면 안 된다.
  실제 native 효과와 연결 단절/재시작/재로그인 후 복원을 끝까지 검증한 뒤에만 하위 완료로 처리해.

[ooo 활용: 설치된 실제 기능과 현재 작업을 연결]
Ouroboros를 계속 활용해. 먼저 이 Claude Code 런타임의 ooo help/스킬·MCP 등록 상태를 읽고
현재 제공되는 명령과 인자를 확인해. Codex의 tool 이름·세션·권한을 그대로 사용할 수 있다고 가정하지 마.

1. ooo resume-session, ooo status로 관련 실행·worker·cwd/worktree를 읽기 전용 확인해.
   관련 기존 session: orch_0dce4e2ea49d; execution: exec_ecefc15077b4.
   원 실행 job_9cbf20bc218b은 verify_workspace_mutated,
   평가 job_9ee04cd84339은 전체 AC 비준수,
   Ralph job_efbac5e0f3a6은 iteration timeout으로 끝난 과거 기록이다.
   오래된 running journal만 믿고 실패한 세션을 자동 resume하거나 중복 worker를 시작하지 마.
2. 확정 인터뷰·기존 validated Seed를 재사용해. 처음부터 ooo interview/seed로 다시 기획하지 마.
   인터뷰는 실제로 새 결정이 필요한 경우에만 사용하고 승인된 목표·AC·제약 변경은 사용자 확인을 받아.
3. 우선 현재 working tree에서 실패 중인 snapshot 단계를 조사·수정·검증해.
   ooo qa를 하위 결과와 실제 관측·해당 완료 기준에 적용하고,
   막히면 ooo unstuck 또는 지원되는 ooo lateral researcher로 원인 조사와 접근법을 검토해.
   범위 축소를 제안받아도 확정 계약과 분리하고 임의로 채택하지 마.
4. 실행 입력이 보존·고정되고 해당 단계가 검증·커밋된 뒤에는 기존 Seed 경로를 명시해
   ooo run과 그 런타임이 지원하는 Ralph 반복 실행을 활용해 남은 구현을 계속해.
   기존 활성 lineage의 명시적 continuation이면 정확한 lineage를 확인하고,
   새 Ralph 실행이면 기존 validated Seed와 별개의 새 lineage_id를 사용해.
   lineage/session/Seed ID를 섞거나 raw 요청을 Seed YAML로 전달하지 마.
5. 실행 전 실제 project_dir/cwd/worktree·기준 커밋·미커밋 포함 여부를 확인해.
   새 worker가 HEAD만 복사하여 현재 변경을 잃거나 옛 source를 검증하게 하지 마.
   한 workspace에는 한 구현 owner만 두고, 원본 수정과 source-bound 검증을 동시에 하지 마.
6. QA 점수·실행 종료·generation 한도·timeout은 전체 완료가 아니다.
   실패·누락은 기록하고 원인을 해결하며 승인된 전체 작업을 계속해.
   status/drift로 기존 Seed 정합성을 확인하고, 실제 최신 독립 live 증거로 전체를 검증해.
7. 마지막에 실제 최신 결과·14개 인수 증거를 대상으로 ooo evaluate의 정식 평가를 수행해.
   ooo qa는 정식 전체 인수의 대체물이 아니다. 과거 실패 세션을 새 성공으로 평가하지 마.
   필요한 ooo 기능이 없다면 정확히 알리고 가능한 수동 구현·검증을 계속하며,
   사용하지 않은 명령을 사용했다고 기록하지 마.

[지속 구현·완료·Git 규칙]
- 매 다음 구현 단계 전 WEB-01~14 전체를 나열하고 검증된 하위 범위·전체 완료·남은 일을 구분해.
- 단계마다 완료 기준·의미 있는 테스트를 먼저 정해 RED → 구현 → GREEN → 필수 회귀 → 커밋.
- 모든 승인 작업과 전체 인수가 완료될 때까지 단계마다 멈추거나 다시 진행할지 묻지 말고 계속해.
- 검증하지 않은 것, skip/mock, 환경 오류, stale digest, 실패한 native proof를 완료로 표기하지 마.
- 사용자 .gitignore가 이미 staged이고 index blob은
  100644 54b076eab7779720ad5afae6169635d45e520759이다. 그대로 보존해.
  refactoring-preparation 문서·baselines·stablenet-boundary-audit 등 사용자 작업도 보존해.
- git add . / commit -a / reset / clean으로 정리하지 마. 우리 명시 파일만 stage/commit해.
  새 SPA JS/CSS 두 파일 등 untracked 구현을 빠뜨리지 마.
- 커밋/PR은 영어 Conventional Commit, 구체적인 수정 의미, 커밋 전체 300자 이내,
  stage jargon·불명확한 약어·emoji·co-author 금지. squash 설명은 핵심 제목으로 정리해.
- 기존 PR #443을 사용해. SAGE Core Team 및 SAGE-X-project/sage-core-dev에 직접/자동 review 요청 금지.
  CODEOWNERS/workflow/자동 assignment를 확인하고 해당 팀을 지정하면 draft를 유지해.
- 전체 snapshot 단계를 검증하기 전에 완료 커밋으로 포장하지 마.
- 진척·증거·실패·다음 행동을 정본과 인계 기록에 지속 기록해 문맥 압축 후에도 이어갈 수 있게 해.

[환경·자료 보존]
- 이 Mac에서는 WEBUI_RUNTIME_ROOT=/private/tmp/cbui,
  GOCACHE=/private/tmp/chainbench-go-cache를 사용해. 짧은 IPC 경로가 필요하다.
- SPA build는 web/에서 완료한 다음 Go embed 빌드·native proof를 실행해.
- fixed RPC/P2P/SSH 포트가 겹치는 native proof와 원본 capture는 순차 실행해.
- 실제 호환 바이너리·본인 전용 fixture만 사용해. 운영 서버나 외부 체인 source는 수정하지 마.
- 최근 free disk는 약 1.7 GiB이고 기존 genesis preflight 최소치는 2 GiB다.
  production threshold를 낮추거나 이를 제품 RED로 계산하지 마.
- 임시 root 다섯 개 전체 삭제는 자동 승인 검토에서 자료 손실 위험으로 거절됐다.
  정확한 경로와 이유는 인계 문서에 있다. 다른 도구/agent로 같은 삭제를 우회하지 마.
  보존 가능한 대안·현재 소유/생존 확인·필요한 승인을 먼저 검토해.
- private credential/key/runtime store를 공유 문서·로그·export·인계 묶음에 넣지 마.
  원본 working tree에는 WIP가 이미 있으므로 인계 wip.patch를 다시 적용하지 마.

먼저 읽은 자료와 실제 상태를 대조하여 확정 목표, 검증된 부분, 미완료 14개,
현재 실패와 첫 행동을 간단히 보고한 뒤, snapshot 실패 조사부터 곧바로 이어서 구현해.
