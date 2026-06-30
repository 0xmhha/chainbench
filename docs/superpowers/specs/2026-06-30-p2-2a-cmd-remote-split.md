# P2-2a — cmd_remote.sh split (test-first)

> 작성일: 2026-06-30
> 상태: SPEC (검토 대기)
> 선행: `docs/REFACTORING_PLAN.md` §2.1 / §6.2 P2-2
> 짝 plan: `docs/superpowers/plans/2026-06-30-p2-2a-cmd-remote-split.md`

---

## 1. Goal & 동기 (정직하게)

`lib/cmd_remote.sh`(434줄)를 dispatcher + sub-command 핸들러로 분리한다. 조사로 드러난 두 사실이 접근을 결정한다:
- 세 대형 bash 파일(cmd_test 639 · cmd_node 602 · cmd_remote 434)은 **800 하드캡 미초과**(400 권장만 초과). 긴급도 낮음.
- `cmd_remote.sh` 는 **단위 테스트 0** (사용자-facing 명령인데 회귀 안전망 없음).

→ 본 sprint 는 **test-first**: 분할 전에 characterization 테스트로 현재 동작을 잠근다. 이로써 (1) 실제 커버리지 갭을 메우고(주 가치) (2) 분할의 behavior-preservation 을 검증한다(부 가치). 위험 가장 낮은 `cmd_remote.sh` 한 파일만 — cmd_node/cmd_test 는 후속(같은 패턴).

---

## 2. Non-goals
- cmd_node.sh / cmd_test.sh 분할 — 후속(P2-2b/c). cmd_test 는 테스트 러너라 가장 위험.
- 동작/인터페이스 변경 0. 순수 함수 재배치 + 테스트 추가.
- remote_state.sh / rpc_client.sh 변경 없음.

---

## 3. 안전망 (test-first) — `tests/unit/tests/cmd-remote.sh` (신규)

RPC 없이 동작하는 상태 CRUD 경로를 잠근다(unreachable URL → 즉시 status=unreachable, 상태는 기록됨):
- `remote add <alias> <url> --type testnet` → `state/remotes.json` 에 alias 기록(status unreachable).
- `remote list` → alias 노출.
- `remote info <alias>` → 상세(연결 불가여도 메타).
- `remote select <alias>` → `state/current-remote` 기록.
- `remote remove <alias>` → remotes.json 에서 제거.
- 에러: 중복 add → 실패, 없는 alias remove → 실패, 알 수 없는 subcommand → 실패.

현재 코드(분할 전)에서 green 확인 → 기준선.

---

## 4. 분할

- `lib/remote_commands.sh` (신규, sourced): `_cb_remote_usage` + `_cb_remote_cmd_add/list/remove/select/info`. 자체 double-source 가드.
- `lib/cmd_remote.sh` (잔여): sourcing(common/remote_state/rpc_client/**remote_commands**) + constants(`_CB_REMOTE_CURRENT_FILE`/`_CB_REMOTE_RPC_TIMEOUT`) + `cmd_remote_main` dispatcher + `cmd_remote_main "$@"`.
- constants 는 함수가 **호출 시점**에 참조 → cmd_remote.sh 가 constants 정의 후 remote_commands.sh 를 source 하면 무방.
- 결과 줄 수: cmd_remote.sh ~100 · remote_commands.sh ~330 (둘 다 400 이내).

---

## 5. Tests / 검증
- §3 characterization: 분할 전 green → 분할 후 동일 green(behavior-preserving 증명).
- 전체 bash suite + Go + vitest 무회귀.
- `bash -n` + (가능 시) shellcheck.

---

## 6. Out-of-Scope / 후속
- P2-2b: cmd_node.sh 분할(같은 test-first 패턴).
- P2-2c: cmd_test.sh 분할(테스트 러너 — 최고 위험, 가장 신중히).

---

## 7. 예상 커밋 (~4)
1. `docs: add P2-2a spec + plan for cmd_remote split`
2. `test(remote): characterization tests for chainbench remote subcommands`
3. `refactor(remote): split cmd_remote.sh handlers into remote_commands.sh`
4. `docs+chore(p2-2a): refactoring-plan + version bump`
