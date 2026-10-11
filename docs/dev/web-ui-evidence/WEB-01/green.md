# WEB-01 GREEN

The final frozen-source cycle on commit `91dc9336` (source digest `37b8892fd66af14cfca05a3c51782ee77f23a69935d87dd56c3f54def282a5b8`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `a06860aa-3f8c-4cd9-b476-6aeb1fc776d6`, outcome `pass`, evidence digest `54a04b6c0cc07e160e4f13cf5d5bd475b2893f290ac177ca3b6ab19655d14a82`.
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
