# WEB-08 GREEN

The final frozen-source cycle on commit `34c951a8` (source digest `470ab4bb2345f320eab9bb3e1dbeafecb130912558bc57fe4de49a910c8b8753`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `47b0a287-ffbc-42ba-8811-b12238dc741d`, outcome `pass`, evidence digest `19d0797faf02b0aac0134715d3063fa35bd90d81a93aa90b9527469652c55d28`.
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
