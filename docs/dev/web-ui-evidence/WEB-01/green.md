# WEB-01 GREEN

The final frozen-source cycle on commit `34c951a8` (source digest `470ab4bb2345f320eab9bb3e1dbeafecb130912558bc57fe4de49a910c8b8753`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `52224256-fa63-45a6-aa5c-c862841fd4ba`, outcome `pass`, evidence digest `0e3998e199f71d964356d81177d7d13d5aaf4b097cbfe3fb8205a298fde32097`.
- An independent reproduction from a staged source copy without `.git` or `chainbench-out` printed `WEB-01 PASS (fresh reproduction; protected source/evidence unchanged)`.
- Required scenarios executed: 7 of 7; skipped: none; failed assertions: none.

Scenarios:

- `stablenet:preset-fields`
- `wbft:preset-fields`
- `wemix:preset-fields`
- `inheritance-defaults-effective`
- `unsupported-ui-api`
- `preset-option-field-coverage`
- `binary-surface-provenance`

Verify with `bash tests/webui/verify.sh --criterion WEB-01 --require-live --output chainbench-out/web-ui-acceptance`.
The summary receipts beside this file are copies of the published `result.json` and `evidence.json`. Screenshots, logs and native receipts they reference stay in the local, ignored acceptance output.
