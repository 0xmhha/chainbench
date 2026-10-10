# WEB-14 RED

(embedded SPA, full regression)

- RED: the worklist file did not exist, so `TestWebWorklistTracksEveryAcceptanceCriterion` failed (`chainbench-web-worklist-red.log`). Fixed in `9ec1449f`. Source: WL "전체 목록 보호".
- Runner RED: WEB-14 was unsupported in `verify.sh` (`chainbench-web-web14-runner-red.log`). Fixed in `f16ec75c`. Source: WL "WEB-14 인수 runner".
- RED: the first full race run skipped 26 live tests. Running them on real gstable exposed a product defect: declared accounts were funded again from an empty sender. Fixed in `2c0fed6d`, with a failing test first (`TestDeclaredAccountsAlreadyPreparedAreNotFundedAgain`). Source: WL, HO l.490-491, PR.
- RED: the OpenAPI YAML did not parse after `cb74142e` (`chainbench-web-openapi-parse-red.log`). Fixed in `558014af`, which added `TestWebOpenAPIParses`. Source: WL, HO l.444.
- Harness (not product): the WEB-14 independent reproduction judged artifacts from the staged copy without `chainbench-out`, so WEB-01 artifacts were reported invalid. A failing unit test was written first and the fix landed in `c5f23d85`. Source: HO l.513-516, PR ("test-harness defects, not product defects").
- Harness (not product): in cycle full4, WEB-14 stalled while re-judging WEB-11 because of the WEB-11 harness race. Fixed in `aa3f2e1e`. Source: HO l.509.

---

Sources: `docs/dev/web-ui-worklist.md` stage table (WL, row named in Korean in quotes; the commit that added each row was confirmed with `git log -p -- docs/dev/web-ui-worklist.md`), `docs/dev/web-ui-claude-handoff.md` (HO, line numbers), PR #443 body (PR), and `git log main..HEAD` commit messages (CM). RED log files are under `/private/tmp/` as the sources name them; they were not re-opened here.

Labels:
- **RED**: product failure observed before the fix.
- **Defect (CM)**: product defect stated in a commit message without a named RED log.
- **Runner RED**: the old `verify.sh` refused the criterion. This is verification tooling, not product behaviour.
- **Harness (not product)**: test, fixture or environment failure that the sources explicitly exclude from product RED.
