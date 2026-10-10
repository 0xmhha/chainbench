# WEB-01 RED

(presets, options, chain surface)

- RED: a saved-preset plan returned HTTP 422 (`chainbench-web-preset-red.log`). Fixed in `22862e12` (feat(web): apply saved chain configurations to execution jobs). Source: WL "저장 프리셋의 실제 구성".
- RED: the archive node configuration was generated wrong (`chainbench-web-preset-archive-{unit-red,red}.log`). Fixed in `22862e12`. Source: WL same row and CM "generate valid archive node settings".
- RED: the engine config-change contract had no enum choices, the change overwrote the existing metrics launch argument, and the live plan returned HTTP 422 (`chainbench-web-config-change-{contract,argv,live}-red.log`). Fixed in `e5566fa8` (feat(web): apply reviewed node configuration during relaunch). Source: WL "구조화된 생성 설정 변경·재실행" (marked partial for WEB-01/07).
- RED: after saving, the saved-case list stayed at r1, and saving server r2 removed the workspace's r1 server choice (`chainbench-web-catalog-{refresh,pins}-red.log`). Fixed in `4c11982f` (fix(web): refresh saved execution choices without reloading). Source: WL "저장 후 실행 자료 갱신".
- Harness (not product): an address-constraint fixture failure and an initial unsupported-verbosity fixture. Source: WL "저장 프리셋의 실제 구성".
- Harness (not product): in a full frozen cycle the WEB-01 reproduction failed because its fixture read an excluded `.ouroboros` manifest. The recheck was changed to pass the selected binary manifest in `4a27ac72`. Source: HO l.473/477, PR.

---

Sources: `docs/dev/web-ui-worklist.md` stage table (WL, row named in Korean in quotes; the commit that added each row was confirmed with `git log -p -- docs/dev/web-ui-worklist.md`), `docs/dev/web-ui-claude-handoff.md` (HO, line numbers), PR #443 body (PR), and `git log main..HEAD` commit messages (CM). RED log files are under `/private/tmp/` as the sources name them; they were not re-opened here.

Labels:
- **RED**: product failure observed before the fix.
- **Defect (CM)**: product defect stated in a commit message without a named RED log.
- **Runner RED**: the old `verify.sh` refused the criterion. This is verification tooling, not product behaviour.
- **Harness (not product)**: test, fixture or environment failure that the sources explicitly exclude from product RED.
