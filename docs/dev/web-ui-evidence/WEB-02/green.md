# WEB-02 GREEN

The final frozen-source cycle on commit `34c951a8` (source digest `470ab4bb2345f320eab9bb3e1dbeafecb130912558bc57fe4de49a910c8b8753`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `ded640d8-bdcf-45d5-8278-1fc359314823`, outcome `pass`, evidence digest `a6b61ec87b695b4fbd1ea058e2f840a6e1d8fb908771e7667c2232bbd7df6860`.
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
