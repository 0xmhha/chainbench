# WEB-09 RED

(history)

- RED: saved-case history and session linkage failed before implementation (`chainbench-web-test-history-red.log`). Fixed in `b1ad1c4f`. Source: WL "저장 케이스 실행과 세션 연결".
- RED: a real test.run Run on the previous dashboard had no archive window (`chainbench-web-history-archive-red.log`). A session record of the same execution blocked deleting the job record (`chainbench-web-history-samejob-red.log`). Fixed in `e3449e3f` (feat(web): link history runs to archived metrics and delete only run-owned data). Further review fixes were tested after the fix and have no RED log. Source: WL "실행 단위 지표·로그 archive와 관리자 삭제".
- RED: a real test.run with cleanup selected ended `cleanup_failed` with `rm: node1 is running` (`chainbench-web-history-web09-try1.log`). Fixed in `23c143e2` (fix(web): stop owned nodes before removing them in selected cleanup). Source: WL "노드를 남기는 작업의 선택 정리", HO l.458, PR.
- RED: a failed replacement left PID 1001 and reset the revision history to 0. Fixed in `372ef55a` (WL marks it partial WEB-07/09).
- Runner RED: WEB-09 was unsupported in `verify.sh` (`chainbench-web-web09-runner-red.log`). Fixed in `25d4527c`. Source: WL "WEB-09 인수 runner".

---

Sources: `docs/dev/web-ui-worklist.md` stage table (WL, row named in Korean in quotes; the commit that added each row was confirmed with `git log -p -- docs/dev/web-ui-worklist.md`), `docs/dev/web-ui-claude-handoff.md` (HO, line numbers), PR #443 body (PR), and `git log main..HEAD` commit messages (CM). RED log files are under `/private/tmp/` as the sources name them; they were not re-opened here.

Labels:
- **RED**: product failure observed before the fix.
- **Defect (CM)**: product defect stated in a commit message without a named RED log.
- **Runner RED**: the old `verify.sh` refused the criterion. This is verification tooling, not product behaviour.
- **Harness (not product)**: test, fixture or environment failure that the sources explicitly exclude from product RED.
