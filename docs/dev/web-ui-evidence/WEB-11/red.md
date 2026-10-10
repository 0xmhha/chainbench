# WEB-11 RED

(resource conflicts, parallel work)

- RED: retained resources did not keep the lock for other workspaces (`chainbench-web-resource-holds-red.log`). Fixed in `06f5cfa6`. Source: WL "보존 자원 충돌 유지".
- RED: conflict owners were not returned to the plan owner or shown in the UI (`chainbench-web-plan-conflicts{,-ui}-red.log`). Fixed in `021068e0`. Source: WL "충돌 소유자 검토".
- RED: concurrent acceptance was unresponsive during slow target verification (`chainbench-web-job-acceptance-red.log`). Fixed in `ab85e40c`. Source: WL "응답 가능한 동시 접수".
- RED: a recomposition plan for a live residual network returned 201, cleanup deleted residual data and reported cleaned, and a PID from a different ledger was accepted (`chainbench-web-residual-{composition,cleanup-unit,ledger-unit}-red.log`). Fixed in `e8970831`. Source: WL "잔류 노드의 재구성·정리 거절".
- RED: a second WBFT workspace with different paths and ports was accepted without conflict, then failed in the engine with a PID-only error (`chainbench-web-resource-parallel-try1.log`). Fixed in `dd08cf09` (feat(web): refuse a second launch of the same node binary before execution). Source: WL "같은 장비의 노드 바이너리 충돌…" (row recorded in `15e26d2e`), PR.
- Runner RED: WEB-11 was unsupported in `verify.sh` (`chainbench-web-web11-runner-red.log`). Fixed in `15e26d2e`, with no separate product RED. Source: WL.
- Harness (not product): `browser_resource_parallel.mjs` read `nodes.length` before the composing `chain-record.json` had `nodes`. Fixed in `aa3f2e1e`. Source: HO l.507-508, PR ("test-harness defects, not product defects").

---

Sources: `docs/dev/web-ui-worklist.md` stage table (WL, row named in Korean in quotes; the commit that added each row was confirmed with `git log -p -- docs/dev/web-ui-worklist.md`), `docs/dev/web-ui-claude-handoff.md` (HO, line numbers), PR #443 body (PR), and `git log main..HEAD` commit messages (CM). RED log files are under `/private/tmp/` as the sources name them; they were not re-opened here.

Labels:
- **RED**: product failure observed before the fix.
- **Defect (CM)**: product defect stated in a commit message without a named RED log.
- **Runner RED**: the old `verify.sh` refused the criterion. This is verification tooling, not product behaviour.
- **Harness (not product)**: test, fixture or environment failure that the sources explicitly exclude from product RED.
