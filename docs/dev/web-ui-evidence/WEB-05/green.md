# WEB-05 GREEN

The final frozen-source cycle on commit `c5f23d85` (source digest `a0fcd339ca22ba0f098e85df1414f704d6a721c07a066d963c4372dd1363407f`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `ac18fdfb-94ee-4f78-99bf-847bd461d59b`, outcome `pass`, evidence digest `a63a9c25b3039aca42428c965c52d2f039226056363d679df114a34a5476011f`.
- An independent reproduction from a staged source copy without `.git` or `chainbench-out` printed `WEB-05 PASS (fresh reproduction; protected source/evidence unchanged)`.
- Required scenarios executed: 10 of 10; skipped: none; failed assertions: none.

Scenarios:

- `grammar`
- `references`
- `v1-migration`
- `invalid-unknown`
- `live-execution`
- `attach-execution`
- `actions`
- `assertions`
- `readers`
- `arguments`

Verify with `bash tests/webui/verify.sh --criterion WEB-05 --require-live --output chainbench-out/web-ui-acceptance`.
The summary receipts beside this file are copies of the published `result.json` and `evidence.json`. Screenshots, logs and native receipts they reference stay in the local, ignored acceptance output.
