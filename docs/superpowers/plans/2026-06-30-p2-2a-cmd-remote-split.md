# P2-2a — cmd_remote.sh split — Plan

> 짝 spec: `2026-06-30-p2-2a-cmd-remote-split.md`. test-first, behavior-preserving. 커밋: English, no co-author, no emoji.

## Task 0 — spec + plan 커밋 (branch `refactor/p2-2a-cmd-remote-split`)
- commit: `docs: add P2-2a spec + plan for cmd_remote split`

## Task 1 — characterization 테스트 (분할 전, 현재 코드 green)
- `tests/unit/tests/cmd-remote.sh`: assert.sh + CHAINBENCH_DIR=temp(state dir). `set --` 후 `source lib/cmd_remote.sh`(dispatcher 가 빈 args → usage, 무해), 이후 `cmd_remote_main add/list/info/select/remove ...` 직접 호출.
  - add(testnet, unreachable URL) → remotes.json 에 alias. list → alias. info → 메타. select → current-remote. remove → 제거. 중복 add/없는 remove/unknown subcmd → 비-0.
- 현재 코드에서 green 확인.
- commit: `test(remote): characterization tests for chainbench remote subcommands`

## Task 2 — 분할
- `lib/remote_commands.sh` 신규: 가드 + `_cb_remote_usage` + `_cb_remote_cmd_{add,list,remove,select,info}` (cmd_remote.sh 에서 그대로 이동).
- `lib/cmd_remote.sh`: 핸들러 제거, `source .../remote_commands.sh` 추가(constants 정의 후). dispatcher/entry 유지.
- 검증: cmd-remote.sh + 전체 bash suite green(동작 불변). `bash -n` 양쪽 clean.
- commit: `refactor(remote): split cmd_remote.sh handlers into remote_commands.sh`

## Task 3 — 문서 + 버전
- REFACTORING_PLAN §6.2: P2-2 → P2-2a(cmd_remote) ✅, P2-2b/c(cmd_node/cmd_test) 잔여. 파일 크기 표 갱신.
- 버전 0.13.0 → 0.13.1 (patch — behavior-preserving refactor + 테스트 추가).
- commit: `docs+chore(p2-2a): refactoring-plan + version 0.13.1`

## 완료 기준
- [ ] cmd_remote characterization 테스트 green (분할 전후 동일)
- [ ] cmd_remote.sh ~100줄(dispatcher) + remote_commands.sh ~330줄, 둘 다 <400
- [ ] 전체 bash suite + Go + vitest 무회귀
