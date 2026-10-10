# WEB-03 RED

(shared server-set, private SSH credentials)

- RED: planning with the workspace's pinned server/path revisions returned HTTP 409 after a newer shared document existed (`chainbench-web-pinned-inputs-red.log`). Fixed in `7ddce354` (fix(web): execute pinned workspace configuration revisions). Source: WL "워크스페이스의 고정 입력 실행".
- RED: after saving server r2, the workspace's r1 server choice disappeared (`chainbench-web-catalog-pins-red.log`). Fixed in `4c11982f`. Source: WL "저장 후 실행 자료 갱신".
- Harness (not product): during pinned-input validation the browser-close wait hung after WEB-03's assertions, so that run was not counted. Source: WL "기존 증거의 한계".
- Harness (not product): WEB-03/04/10 Python receipt checks failed because the SPA rebuild made the old receipts stale. Source: HO l.437.
- Otherwise no product RED is recorded specific to WEB-03.

---

Sources: `docs/dev/web-ui-worklist.md` stage table (WL, row named in Korean in quotes; the commit that added each row was confirmed with `git log -p -- docs/dev/web-ui-worklist.md`), `docs/dev/web-ui-claude-handoff.md` (HO, line numbers), PR #443 body (PR), and `git log main..HEAD` commit messages (CM). RED log files are under `/private/tmp/` as the sources name them; they were not re-opened here.

Labels:
- **RED**: product failure observed before the fix.
- **Defect (CM)**: product defect stated in a commit message without a named RED log.
- **Runner RED**: the old `verify.sh` refused the criterion. This is verification tooling, not product behaviour.
- **Harness (not product)**: test, fixture or environment failure that the sources explicitly exclude from product RED.
