# WEB-04 GREEN

The final frozen-source cycle on commit `c5f23d85` (source digest `a0fcd339ca22ba0f098e85df1414f704d6a721c07a066d963c4372dd1363407f`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `e80f1b5a-97df-4b07-95fa-2c2ee54290c8`, outcome `pass`, evidence digest `9b5dccd07745c4a5f5d299b144b763f56e055f0a7c84ef5e978b2ba321c579e6`.
- An independent reproduction from a staged source copy without `.git` or `chainbench-out` printed `WEB-04 PASS (fresh reproduction; protected source/evidence unchanged)`.
- Required scenarios executed: 10 of 10; skipped: none; failed assertions: none.

Scenarios:

- `builtin-stablenet`
- `external-stablenet`
- `builtin-wbft`
- `external-wbft`
- `builtin-wemix`
- `external-wemix`
- `unsupported-family`
- `unsupported-dialect`
- `wrong-binary-identity`
- `manifest-management`

Verify with `bash tests/webui/verify.sh --criterion WEB-04 --require-live --output chainbench-out/web-ui-acceptance`.
The summary receipts beside this file are copies of the published `result.json` and `evidence.json`. Screenshots, logs and native receipts they reference stay in the local, ignored acceptance output.
