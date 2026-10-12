# WEB-05 RED

(DSL editing and execution)

- RED: saved-case test jobs failed before implementation on history, binary, default binary, layout and native-name checks (`chainbench-web-test-{history,binary,binary-default,layout,native-name}-red.log`). Fixed in `b1ad1c4f` (feat(web): run saved test cases and preserve their engine results). Source: WL "저장 케이스 실행과 세션 연결".
- RED: a DSL node table was refused (`chainbench-web-test-table-red.log`). Fixed in `f263072f` (feat(web): run tests with reviewed per-node placement). Source: WL "DSL 노드 표의 실제 실행".
- RED: a generated-genesis saved case was refused on re-check because of an empty-array vs omitted-field difference (`chainbench-web-generated-test-inputs-red.log`). Fixed in `0eed60eb`. Source: WL "일반 genesis 테스트의 승인 snapshot".
- Defect (CM): a node that was not yet listening passed `expect fail`, its exit reason was read before it was written, and `swapNode binary default` reached exec as a path. Fixed in `9b0cfcda` (fix(dsl): judge expect fail over the whole probe window and swap onto default by name).
- Defect (CM): a failed launch in a workspace with its own data root gave no reason because the log was looked up under the control directory. Fixed in `4140d74a` (fix(engine): read node logs from their recorded path and name unregistered builtins).
- RED: `TestWebAttachTargetsTheRecordedNetwork`, `TestWebAttachRefusesComposedCasesAndHostKeyFiles` and the `caseAttaches` JS test failed before implementation, and `TestWebAttachBindsKeyFileAccountsToPrivateCredentials` and the `keyFileAccounts` JS test also failed before implementation. Fixed in `5b584327` and `a9bf05ed`. Source: WL "Web attach 실행".
- RED: `TestSuite_Live_DeclaredAccountsWithReadOnlyKeys` failed on real gstable ("a read-only key set gained the declared accounts"), and `TestWebComposedRunAcceptsMintedAccountsOnly` failed with the old refusal text. Fixed in `a9bf05ed`. A related real defect was that a composed run funded declared accounts a second time from an empty sender. Fixed in `2c0fed6d` (fix(engine): stop refunding declared accounts a composed run already funded). Source: WL "Web composed run의 선언 계정", HO l.490-491, PR.
- Harness (not product): the first attach development run was blocked by a normal executable-conflict refusal. The value-transfer case failed because the payer node1 was a validator earning block rewards. The probes case queried `callError` before en1 had the deploy block. `test_web05_isolation.py` cut the roughly 9 min 43 s recheck at 600 s; this was fixed in `aa3f2e1e`. Source: WL "Web attach 실행", "Web composed run의 선언 계정", HO l.509-511.
- RED: after merging main, the semantic contract tests failed because main's `swapNode args` was missing from the precise argument contract and editor, and main's mixed cases set a swapNode `purpose` that a binary or args swap never reads. Fixed in the merge `857286ec`; `91dc9336` added an args relaunch to the node control coverage case (423 of 423 argument paths in a WEB-05 development run). Source: CM, PR.
- RED: on the frozen run at `3f3d3831` the wemix-to-wbft fork job ended `cleanup_failed` with "selected placement differs from the owned network": the record kept BPCount 1 while the fork had promoted endpoints to producers. `TestWebControlRecordAcceptsForkPromotedProducers` failed first. Fixed in `a053fd7a`.
- RED: after a page reload the job form draft restored the previous job's node cleanup into the fork job. The WEB-05 browser proof now checks the reloaded form shows retain and that the fork network is cleaned only when chosen. Fixed in `f24b7dfc`.
- Harness (not product): `11-arguments-transfers` read a reverted transaction's receipt on en1 before en1 held the block ("no receipt"). The case now waits for en1 first. Fixed in `34677851`.

---

Sources: `docs/dev/web-ui-worklist.md` stage table (WL, row named in Korean in quotes; the commit that added each row was confirmed with `git log -p -- docs/dev/web-ui-worklist.md`), `docs/dev/web-ui-claude-handoff.md` (HO, line numbers), PR #443 body (PR), and `git log main..HEAD` commit messages (CM). RED log files are under `/private/tmp/` as the sources name them; they were not re-opened here.

Labels:
- **RED**: product failure observed before the fix.
- **Defect (CM)**: product defect stated in a commit message without a named RED log.
- **Runner RED**: the old `verify.sh` refused the criterion. This is verification tooling, not product behaviour.
- **Harness (not product)**: test, fixture or environment failure that the sources explicitly exclude from product RED.
