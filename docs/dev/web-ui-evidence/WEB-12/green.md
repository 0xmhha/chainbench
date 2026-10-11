# WEB-12 GREEN

The final frozen-source cycle on commit `91dc9336` (source digest `37b8892fd66af14cfca05a3c51782ee77f23a69935d87dd56c3f54def282a5b8`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `a45695df-a746-4e0d-9c80-3a2edea6eb4a`, outcome `pass`, evidence digest `eca6ad13381f97bdc79bba717046d241719eab2c5aac0a4b09b84f434268e7b3`.
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
