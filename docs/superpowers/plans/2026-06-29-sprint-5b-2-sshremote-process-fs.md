# Sprint 5b.2 — SSHRemoteDriver (process + fs) — Plan

> 작성일: 2026-06-29 · 짝 spec: `2026-06-29-sprint-5b-2-sshremote-process-fs.md`
> 확정: process = ProviderMeta `*_cmd` 모델 · fs = tail over SSH · cap → {fs,process,rpc,ws}.
> 실행: 직접 구현 + 전 레이어 테스트. 커밋: English, no co-author, no emoji.

---

## Task 0 — spec + plan 커밋
- 브랜치 `feat/sprint-5b-2-sshremote-process-fs`.
- commit: `docs: add Sprint 5b.2 spec + plan for SSH process/fs`

## Task 1 — sshremote.Exec
- `drivers/sshremote/sshremote.go`: `ExecResult{Stdout,Stderr string; ExitCode int}` + `Exec(ctx, creds, hostKey, command) (ExecResult, error)`. ssh.Dial(5b.1 cfg 재사용 — 공통 `dialSSH` 헬퍼 추출) → NewSession → 버퍼 stdout/stderr → session.Run; `*ssh.ExitError`→ExitCode(에러 아님), 그 외 dial/세션 실패만 error. password 미노출.
- 테스트는 Task 6(서버에 exec 채널 추가 필요).
- commit: `feat(sshremote): add Exec for remote command execution`

## Task 2 — sshCredsFromNode 공통 헬퍼
- `handlers.go`: 5b.1 `dialSSHNode` 의 ssh-password 파싱(type/user/host/env/port + os.Getenv + 검증)을 `sshCredsFromNode(node *types.Node) (sshremote.Credentials, error)` 로 추출. `dialSSHNode` 가 이를 사용하도록 리팩토(동작 불변, 기존 5b.1 테스트 green 유지).
- commit: `refactor(network-net): extract sshCredsFromNode helper`

## Task 3 — tail_log over SSH (fs)
- `handlers_node_lifecycle.go newHandleNodeTailLog`: network-aware 화. local|"" → 기존 경로. 비-local → resolveNode:
  - provider==ssh-remote: log_file=ProviderMeta["log_file"](string). 없으면 NOT_SUPPORTED. creds=sshCredsFromNode, hostKey=ResolveHostKeyCallback. `Exec(..., fmt.Sprintf("tail -n %d -- %s", lines, shellQuote(logFile)))`. exit!=0→UPSTREAM. stdout 을 라인 분할해 기존 결과 shape({node_id,log_file,lines}) 로.
  - provider==remote: NOT_SUPPORTED.
- `shellQuote` 헬퍼(작은따옴표 escape) — operator-set 경로지만 방어적.
- 테스트 Task 6.
- commit: `feat(network-net): node.tail_log over SSH for ssh-remote (fs)`

## Task 4 — stop/start/restart over SSH (process)
- `handlers_node_lifecycle.go` 3 핸들러 network-aware 화. local|"" → 기존 LocalDriver 경로 무변경. 비-local → resolveNode:
  - remote → NOT_SUPPORTED.
  - ssh-remote → ProviderMeta `stop_cmd`/`start_cmd`/`restart_cmd`. 명령 미설정→NOT_SUPPORTED. Exec→exit!=0→UPSTREAM. 성공 시 기존 이벤트(node.stopped/started)+결과 shape.
  - restart: restart_cmd 있으면 단일; 없으면 stop_cmd→start_cmd 합성(이벤트 순서 불변 D4). 둘 다 없으면 NOT_SUPPORTED.
- 공통 헬퍼 `execNodeCmd(ctx, node, cmd) (ExecResult, error)`(creds+hostKey+Exec 래핑).
- 테스트 Task 6.
- commit: `feat(network-net): node stop/start/restart over SSH for ssh-remote (process)`

## Task 5 — capability 복원
- `handlers_network.go providerCaps["ssh-remote"]` → `{"fs","process","rpc","ws"}`. 주석 갱신(5b.2 구현 완료). D2 노드-레벨 미설정→런타임 NOT_SUPPORTED 한계 1줄.
- 영향: hybrid 교집합 테스트(local+ssh-remote = ssh-remote 전체) — 기존 테스트 영향 확인/갱신.
- commit: `fix(network-net): restore ssh-remote capabilities to {fs,process,rpc,ws}`

## Task 6 — 테스트 (SSH exec 서버 + 핸들러)
- `sshremote_test.go`: 5b.1 in-process 서버에 **session 채널 + exec 요청** 처리 추가(payload 의 command 파싱, fake 실행: echo + 지정 exit). `Exec` happy/비-zero/bad-password.
- `handlers_ssh_test.go`(또는 신규): ssh-remote stop/start/restart/tail_log happy(mock 서버) + 미설정 NOT_SUPPORTED + remote NOT_SUPPORTED + 비-zero UPSTREAM + password 미노출. local 무회귀(기존 테스트).
- capability 단언(ssh-remote 4-set).
- commit: `test(sshremote): SSH exec server harness + process/fs handler coverage`

## Task 7 — 문서 + 버전
- VISION 5b.2 체크박스 ✅. REMAINING_WORK §4 Priority 2: 5b 전체 완료(5b.1+5b.2), 잔여=구성 명령/키 인증/네트워크 단위(후속).
- 예제: `examples/networks/ssh-remote-example.json` 에 provider_meta(log_file/*_cmd) 추가 + README 보강.
- 버전 0.9.0 → 0.10.0.
- commit: `docs+chore(sprint-5b-2): roadmap + example + version 0.10.0`

---

## 완료 기준
- [ ] ssh-remote stop/start/restart 가 provider_meta 명령을 SSH exec (mock 서버 통합)
- [ ] ssh-remote tail_log 가 log_file 을 SSH tail
- [ ] 미설정 명령/log_file → NOT_SUPPORTED, remote → NOT_SUPPORTED, 비-zero → UPSTREAM
- [ ] password 미노출 (negative)
- [ ] providerCaps ssh-remote = {fs,process,rpc,ws}
- [ ] local lifecycle 무회귀
- [ ] 전 레이어 green: Go(전 패키지, vet/gofmt) · vitest · bash · go.mod tidy 안정
