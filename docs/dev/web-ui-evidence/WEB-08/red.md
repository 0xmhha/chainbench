# WEB-08 RED

(monitoring, metrics, logs, continuity)

- RED: `/api/v1/snapshot` returned 404, the jobId scope included unrelated networks, a reader mutated the cached slice, and the UI had no run-status heading (`chainbench-web-job-snapshot-{http,scope,copy}-red.log`, `chainbench-web-snapshot-browser-red.log`). Fixed in `cb74142e` (feat(web): restore job monitoring from snapshots after reconnects and restarts). Source: HO l.194-199, WL "실행 스냅샷·변경 스트림·재연결 복원".
- RED: after a real native test.run with recorded metrics ports, the metrics query returned 404, and the collector and chart modules were missing (`chainbench-web-monitoring-{browser,unit,js}-red.log`). Fixed in `558014af` (feat(web): chart archived node metrics and link chart times to node logs). Source: WL "노드 지표 수집·차트·시점 로그".
- RED: review-found defects were multi-line PEM key storage, outage shrink during long rounds, and unreported failing siblings (`chainbench-web-monitoring-review-red.log`). The OpenAPI YAML broken by `cb74142e` was reproduced in `chainbench-web-openapi-parse-red.log`. Fixed in `558014af`. Source: WL, HO l.444.
- RED: the previous dashboard could not start remote log collection after a real SSH deployment (`chainbench-web-ssh-log-collection-red.log`). Fixed in `40c8adc2` (feat(web): collect SSH node logs through an operator-started job). Five review fixes have tests written after the fix and no RED log. Source: WL "SSH 노드 로그 수집 작업".
- Runner RED: WEB-08 was unsupported in `verify.sh` (`chainbench-web-web08-runner-red.log`). Fixed in `2a175cd6`. Sign-out continuity was a new observation of existing behaviour with no product RED. Source: WL "WEB-08 인수 runner".
- Harness (not product): the first native snapshot-restore failure was the gwbft nodes' own low-disk shutdown at 428 MiB free, and the assertion was kept. Source: HO l.433-434, WL, PR.
- Harness (not product): a fixture wrongly assumed the session survives a restart, a physical port claim conflicted, a sandbox listener was refused, the JS module was missing, and the metadata executor was a unit-level stand-in. Source: HO l.207-213, WL.

---

Sources: `docs/dev/web-ui-worklist.md` stage table (WL, row named in Korean in quotes; the commit that added each row was confirmed with `git log -p -- docs/dev/web-ui-worklist.md`), `docs/dev/web-ui-claude-handoff.md` (HO, line numbers), PR #443 body (PR), and `git log main..HEAD` commit messages (CM). RED log files are under `/private/tmp/` as the sources name them; they were not re-opened here.

Labels:
- **RED**: product failure observed before the fix.
- **Defect (CM)**: product defect stated in a commit message without a named RED log.
- **Runner RED**: the old `verify.sh` refused the criterion. This is verification tooling, not product behaviour.
- **Harness (not product)**: test, fixture or environment failure that the sources explicitly exclude from product RED.
