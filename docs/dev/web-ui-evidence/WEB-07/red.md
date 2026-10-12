# WEB-07 RED

(owned node control, replacement, reset, attach refusal)

- RED: a stop succeeded through an unverified PID path (`chainbench-web-node-probe-red.log`, `chainbench-web-node-control-red.log`). Fixed in `1a55d4ed` (feat(web): verify recorded processes before node control). Also RED: real test-network control (`chainbench-web-test-control-red.log`), fixed in `0a986b41`. Source: WL "기록된 프로세스 관측·제어 검증", "보존된 테스트 네트워크 제어".
- RED: a recorded-control plan returned HTTP 422 for five producers, and an out-of-range record was accepted (`chainbench-web-recorded-controls{,-unit}-red.log`). Fixed in `d3a27520` (fix(web): control recorded node layouts and preserve key inputs). Source: WL "기록 배치 기반 노드 제어".
- RED: a start plan for a live PID-zero node returned HTTP 201 (`chainbench-web-unrecorded-controls-red.log`). Fixed in `4b819add`. RED: a stop succeeded with an unverified ledger PID, and a live WBFT control plan returned 201 (`chainbench-web-node-ledger-{unit,live}-red.log`). Fixed in `777dc5ea`. Source: WL "PID 없는 잔류 노드…", "개별 제어의 프로세스 ledger 대조".
- RED: the reset execution adapter was missing, the reset plan returned 422, and the capability registry was missing (`chainbench-web-node-reset-{unit,live,registry}-red.log`). Fixed in `0eed60eb`. RED: a stopped node was refused and a repeated reset plan returned 409 (`chainbench-web-stopped-reset-{unit,live}-red.log`). Fixed in `6a8996bb`. Source: WL "소유한 비생산자의 명시적 초기화", "실제 정지가 확인된 비생산자 초기화".
- RED: the restart plan for real Stablenet returned 422 (`chainbench-web-restart-live-red.log`). Fixed in `78267837`. RED: PID 1001 remained after two failures, the revision reset to 0, and a stopped PID remained natively (`chainbench-web-swap-{failure,history,native}-red.log`). Fixed in `372ef55a`. Source: WL "소유 노드의 명시적 재실행", "교체 실패의 정지 상태·실행 이력".
- RED: config change had no enum choices, overwrote argv, and returned live 422. Fixed in `e5566fa8`. RED: an unregistered per-node binary name was planned with 201, and a real stop occurred on a baseline overlay (`chainbench-web-node-binary-binding{,-live,-process}-red.log`). Fixed in `4287b409`. Source: WL rows "구조화된 생성 설정 변경·재실행", "개별 실행 파일의 등록 근거 대조".
- RED: a registered native replacement plan returned 422, a linked binding was accepted, restart history was lost, and a different protocol mapping was accepted (`chainbench-web-binary-replacement-{plan,history-live,history-unit,mapping}-red.log`, `chainbench-web-binary-history-symlink-red.log`). Fixed in `e6b3d3c1`. The SSH replacement returned 422 before `3430c1e3`. Source: WL, PR.
- RED: replacement of a nonexecutable file succeeded over SSH, and four launch actions were allowed (`chainbench-web-nonexecutable-control-unit-red.log`). Fixed in `bdd175e6`. RED: launch actions were still offered after the permission was removed (`chainbench-web-observed-permission-unit-red.log`). Fixed in `7c8de876`. Runner RED: WEB-07 was unsupported in `verify.sh` before `be04ad9a`. Harness (not product): the initial key shortage, CLI option fixture errors, 409/422 expectation errors, an initial unit fixture contract error, and the copied macOS system-binary fixture exiting. Source: WL.
- Defect (CM): the process observer was in a `_darwin.go` file, so `internal/app` did not compile on Linux CI although the observer judges the target's OS at runtime. Fixed in `8b6f67f4` (fix(web): build the process observer on every OS so Linux CI compiles internal/app). Source: CM, CI.
- RED (regression introduced in `a053fd7a`): a wrong validator argument came back as 409 instead of 422, failing the recorded-controls proof. `TestWebControlRecordReportsWrongValidatorsAsInput` failed first. Fixed in `0200bf33`.
- RED: SSH node observation took 8011 ms against an 8 s budget, so a node was reported unknown and the SSH binary replacement proof failed. The proof now fails when any node goes unobserved; it reproduced every time before the fix and observed all four nodes in 10.6 s after. Fixed in `39b341ca`.

---

Sources: `docs/dev/web-ui-worklist.md` stage table (WL, row named in Korean in quotes; the commit that added each row was confirmed with `git log -p -- docs/dev/web-ui-worklist.md`), `docs/dev/web-ui-claude-handoff.md` (HO, line numbers), PR #443 body (PR), `git log main..HEAD` commit messages (CM), and GitHub Actions runs on PR #443 (CI). RED log files are under `/private/tmp/` as the sources name them; they were not re-opened here.

Labels:
- **RED**: product failure observed before the fix.
- **Defect (CM)**: product defect stated in a commit message without a named RED log.
- **Runner RED**: the old `verify.sh` refused the criterion. This is verification tooling, not product behaviour.
- **Harness (not product)**: test, fixture or environment failure that the sources explicitly exclude from product RED.
