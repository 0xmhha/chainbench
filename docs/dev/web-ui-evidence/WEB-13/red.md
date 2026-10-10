# WEB-13 RED

(cancel, retention, revocation)

- RED: selected cleanup always failed with `cleanup_failed` because nodes were still running. Fixed in `23c143e2`. Source: WL, HO l.458, PR.
- RED: the job form showed only a reasonless 422 for an attach workspace (`chainbench-web-cancel-retention-try4.log`). Fixed in `f23bd071` (feat(web): explain attached-network refusal in the job form). Source: WL "attach 네트워크의 화면 거절 설명" (row recorded in `4b736df1`).
- RED: residual cleanup deleted data and reported cleaned, and retained resources were not held until verified cleanup. Fixed in `e8970831` and `06f5cfa6`. Source: WL.
- RED: before implementation, tests failed for a credential revocation that cancels attach jobs. Fixed in `a9bf05ed`. Source: WL "Web attach 실행".
- RED: the previous dashboard lacked a remote log job with revocation behaviour. Fixed in `40c8adc2`. Source: WL "SSH 노드 로그 수집 작업".
- Runner RED: WEB-13 was unsupported in `verify.sh` (`chainbench-web-web13-runner-red.log`). Fixed in `4b736df1`. Source: WL.
- Harness (not product): the SSH fault gate helper was missing (`43d86fb8`). The product disconnect/revocation handling passed on its first run. Source: WL, PR.

---

Sources: `docs/dev/web-ui-worklist.md` stage table (WL, row named in Korean in quotes; the commit that added each row was confirmed with `git log -p -- docs/dev/web-ui-worklist.md`), `docs/dev/web-ui-claude-handoff.md` (HO, line numbers), PR #443 body (PR), and `git log main..HEAD` commit messages (CM). RED log files are under `/private/tmp/` as the sources name them; they were not re-opened here.

Labels:
- **RED**: product failure observed before the fix.
- **Defect (CM)**: product defect stated in a commit message without a named RED log.
- **Runner RED**: the old `verify.sh` refused the criterion. This is verification tooling, not product behaviour.
- **Harness (not product)**: test, fixture or environment failure that the sources explicitly exclude from product RED.
