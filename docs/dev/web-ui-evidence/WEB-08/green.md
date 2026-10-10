# WEB-08 GREEN

The final frozen-source cycle on commit `c5f23d85` (source digest `a0fcd339ca22ba0f098e85df1414f704d6a721c07a066d963c4372dd1363407f`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `b3bf34ea-588c-430b-a860-3a431144b453`, outcome `pass`, evidence digest `7fb800aeaa8e63714d4e7c2c0e89dd5678559bd081d640c878c9e7ad9d42b32f`.
- An independent reproduction from a staged source copy without `.git` or `chainbench-out` printed `WEB-08 PASS (fresh reproduction; protected source/evidence unchanged)`.
- Required scenarios executed: 6 of 6; skipped: none; failed assertions: none.

Scenarios:

- `native-state-metric-charts`
- `time-linked-logs`
- `browser-close-logout-continuity`
- `reconnect-snapshot-restore`
- `sse-gap-drop-stale`
- `collection-failure-reporting`

Verify with `bash tests/webui/verify.sh --criterion WEB-08 --require-live --output chainbench-out/web-ui-acceptance`.
The summary receipts beside this file are copies of the published `result.json` and `evidence.json`. Screenshots, logs and native receipts they reference stay in the local, ignored acceptance output.
