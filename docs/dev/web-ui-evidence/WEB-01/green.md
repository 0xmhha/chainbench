# WEB-01 GREEN

The final frozen-source cycle on commit `c5f23d85` (source digest `a0fcd339ca22ba0f098e85df1414f704d6a721c07a066d963c4372dd1363407f`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `27347674-7711-46a4-93f2-efbc6dde5300`, outcome `pass`, evidence digest `1d9f056b483e620ad8570c8d6ea0b5a414e210a10dff7ef616eaca040a385fd4`.
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
