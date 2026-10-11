# WEB-02 GREEN

The final frozen-source cycle on commit `91dc9336` (source digest `37b8892fd66af14cfca05a3c51782ee77f23a69935d87dd56c3f54def282a5b8`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `bf8a43b0-7dcc-45b1-8226-03e0cc9191ab`, outcome `pass`, evidence digest `de98832dfeb55fb02376ad86d5afec8a56b9aa4c8556730283693fc884a987bb`.
- An independent reproduction from a staged source copy without `.git` or `chainbench-out` printed `WEB-02 PASS (fresh reproduction; protected source/evidence unchanged)`.
- Required scenarios executed: 6 of 6; skipped: none; failed assertions: none.

Scenarios:

- `bundle-import-preview-commit`
- `concrete-field-and-extension-errors`
- `structured-edit-revision-conflict`
- `export-round-trip-semantics`
- `setup-from-imported-configuration`
- `pinned-assets-and-revisions`

Verify with `bash tests/webui/verify.sh --criterion WEB-02 --require-live --output chainbench-out/web-ui-acceptance`.
The summary receipts beside this file are copies of the published `result.json` and `evidence.json`. Screenshots, logs and native receipts they reference stay in the local, ignored acceptance output.
