# WEB-02 RED

(import, edit, export, real setup)

- RED: bundles were refused with "select document kind" and "expected one document" (`chainbench-web-bundle-unit-red.log`), and the previous build had no import screen (`chainbench-web-web02-bundle-red.log`). Fixed in `32aa8140` (feat(web): import and export workspace bundles and show redacted validation reasons). Source: WL "구성 묶음 import·export와 WEB-02 runner".
- RED: a 422 for an invalid shared document showed only a generic message, and the shared `api-error` JS module did not exist (`chainbench-web-validation-errors-red.log`, `chainbench-web-api-error-js-red.log`). Fixed in `32aa8140`. Source: WL "검증 오류의 구체적 이유 표시" (row recorded in `4a27ac72`), HO l.466/474.
- RED: case bundles carrying pinned presets failed to compile before implementation. Fixed in `8114df3f` (feat(web): export a saved case with its pinned presets and import it linked). Source: WL "구성 묶음…" ("구현 전 컴파일 RED").
- RED: the asset API was unimplemented, a keystore was shared, the probe output limit could be bypassed, and a storage error exposed paths (`chainbench-web-assets{,-secrets,-probe,-storage}-red.log`). Fixed in `50e4e52c` (feat(web): register uploaded assets for chain execution). Source: WL "업로드 바이너리 실행".
- RED: saving a registered genesis reference failed (`chainbench-web-genesis-asset-red.log`). Fixed in `5b2388f5` (feat(web): use registered genesis files in saved configurations). Source: WL "등록된 완성 genesis 적용".
- RED: test-case genesis dependencies failed in projection, tampering after transfer was not refused, the UI selection event was wrong, and PoA governance inputs were missing (`chainbench-web-test-genesis{,-projection,-recheck,-ui,-poa}-red.log`). Fixed in `173a2cd1` (feat(web): run saved tests with registered genesis files). Source: WL "테스트 케이스의 완성 genesis 자료".
- RED: key-snapshot pinning failed every predefined criterion (`chainbench-web-key-red.log`). Fixed in `6c98916c` (feat(web): pin encrypted key snapshots to approved execution plans). Source: WL "키 자료 고정" (partial WEB-02/06).
- Runner RED: the old `verify.sh` did not support WEB-02 (`chainbench-web-web02-runner-red.log`). Fixed in `4a27ac72`. Source: WL. Harness (not product): test-tool shutdown errors, fixture 409/422 expectation errors and the macOS IPC path-length failure. Source: WL "업로드 바이너리 실행", "테스트 케이스의 완성 genesis 자료".

---

Sources: `docs/dev/web-ui-worklist.md` stage table (WL, row named in Korean in quotes; the commit that added each row was confirmed with `git log -p -- docs/dev/web-ui-worklist.md`), `docs/dev/web-ui-claude-handoff.md` (HO, line numbers), PR #443 body (PR), and `git log main..HEAD` commit messages (CM). RED log files are under `/private/tmp/` as the sources name them; they were not re-opened here.

Labels:
- **RED**: product failure observed before the fix.
- **Defect (CM)**: product defect stated in a commit message without a named RED log.
- **Runner RED**: the old `verify.sh` refused the criterion. This is verification tooling, not product behaviour.
- **Harness (not product)**: test, fixture or environment failure that the sources explicitly exclude from product RED.
