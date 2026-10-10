# WEB-12 RED

(server restart)

- RED: the target record was not persisted before execution, and an interrupted test lost node observations (`chainbench-web-test-target-red.log`, `chainbench-web-interrupted-test-red.log`). Fixed in `034f56fd` (fix(web): preserve node observations after interrupted tests). Source: WL "실행 중 재시작 후 기록된 노드 관측".
- RED: a live PID-zero residual start was planned with 201. Fixed in `4b819add`. Source: WL "PID 없는 잔류 노드 발견·제어 차단".
- RED: the snapshot/replay endpoint returned 404 and the UI had no status heading, which blocked restart restoration. Fixed in `cb74142e`. Source: HO l.194-199.
- Runner RED: WEB-12 was unsupported in `verify.sh` (`chainbench-web-web12-runner-red.log`). Fixed in `15e26d2e`, with no separate product RED for restart during deployment. Source: WL.
- Harness (not product): the interrupted-test fixture had initial DSL, readiness-wait and omitempty errors. Source: WL.
- Harness (not product): the first native restore failed on a fixture session assumption and then on the low-disk node shutdown. Source: HO l.211-216/433-434.

---

Sources: `docs/dev/web-ui-worklist.md` stage table (WL, row named in Korean in quotes; the commit that added each row was confirmed with `git log -p -- docs/dev/web-ui-worklist.md`), `docs/dev/web-ui-claude-handoff.md` (HO, line numbers), PR #443 body (PR), and `git log main..HEAD` commit messages (CM). RED log files are under `/private/tmp/` as the sources name them; they were not re-opened here.

Labels:
- **RED**: product failure observed before the fix.
- **Defect (CM)**: product defect stated in a commit message without a named RED log.
- **Runner RED**: the old `verify.sh` refused the criterion. This is verification tooling, not product behaviour.
- **Harness (not product)**: test, fixture or environment failure that the sources explicitly exclude from product RED.
