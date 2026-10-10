# WEB-02 GREEN

The final frozen-source cycle on commit `c5f23d85` (source digest `a0fcd339ca22ba0f098e85df1414f704d6a721c07a066d963c4372dd1363407f`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `9b314955-4515-4743-aa8b-40481d903c34`, outcome `pass`, evidence digest `4faec8a67017ebcc1da78ef4d99da5fa08147b774685ecde819b7aa46d47b51a`.
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
