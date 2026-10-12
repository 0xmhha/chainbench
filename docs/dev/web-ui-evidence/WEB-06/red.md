# WEB-06 RED

(SSH deployment)

- RED: key-snapshot pinning failed every predefined criterion. Fixed in `6c98916c`. Source: WL "키 자료 고정" (partial WEB-02/06).
- RED: a retained pre-replacement dashboard returned HTTP 422 on a replacement plan after real SSH setup and two node starts (`chainbench-web-ssh-binary-replacement-browser-red.log`). Fixed in `3430c1e3` (test(web): verify native executable replacement over private SSH). Source: WL "개인 SSH를 통한 등록 바이너리 교체", PR.
- RED: a nonexecutable native target replacement over real SSH was reported as successful and changed the existing node (`chainbench-web-nonexecutable-ssh-browser-red.log`). Fixed in `bdd175e6` (fix(web): reject nonexecutable files before replacing or restarting nodes). Source: WL "교체·재실행 전 실제 실행 권한 확인", PR.
- Harness (not product): the SSH fault forced-command gate failed for a missing helper (`chainbench-web-ssh-fault-gate-harness-red.log`). The product's existing disconnect/revoke handling passed on its first run. Source: WL "실제 SSH 복사 응답 중 단절·철회" (`43d86fb8`), PR.
- Runner RED: the old `verify.sh` did not support WEB-06 (`chainbench-web-web06-runner-red.log`). Fixed in `15e26d2e`. The new deploy-fault proof found no separate product RED. Source: WL "WEB-06·11·12 인수 runner".
- Unexplained (not counted as RED): an earlier SSH run refused a node review with HTTP 409, a fresh full run passed, and the cause remains unresolved. Source: PR.

---

Sources: `docs/dev/web-ui-worklist.md` stage table (WL, row named in Korean in quotes; the commit that added each row was confirmed with `git log -p -- docs/dev/web-ui-worklist.md`), `docs/dev/web-ui-claude-handoff.md` (HO, line numbers), PR #443 body (PR), and `git log main..HEAD` commit messages (CM). RED log files are under `/private/tmp/` as the sources name them; they were not re-opened here.

Labels:
- **RED**: product failure observed before the fix.
- **Defect (CM)**: product defect stated in a commit message without a named RED log.
- **Runner RED**: the old `verify.sh` refused the criterion. This is verification tooling, not product behaviour.
- **Harness (not product)**: test, fixture or environment failure that the sources explicitly exclude from product RED.
