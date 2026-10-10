# WEB-12 GREEN

The final frozen-source cycle on commit `c5f23d85` (source digest `a0fcd339ca22ba0f098e85df1414f704d6a721c07a066d963c4372dd1363407f`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `8a968c73-7167-4c5a-8ff7-8a245bd8551b`, outcome `pass`, evidence digest `c7971f8d77250eca448b3fa23f5b2bcb5561c0e42a118fcd3f6a48ed65f7bef5`.
- An independent reproduction from a staged source copy without `.git` or `chainbench-out` printed `WEB-12 PASS (fresh reproduction; protected source/evidence unchanged)`.
- Required scenarios executed: 5 of 5; skipped: none; failed assertions: none.

Scenarios:

- `restart-marks-running-work-interrupted`
- `actual-state-rechecked`
- `records-and-claims-preserved`
- `no-automatic-redeploy-or-reset`
- `explicit-rerun-new-plan`

Verify with `bash tests/webui/verify.sh --criterion WEB-12 --require-live --output chainbench-out/web-ui-acceptance`.
The summary receipts beside this file are copies of the published `result.json` and `evidence.json`. Screenshots, logs and native receipts they reference stay in the local, ignored acceptance output.
