# WEB-08 GREEN

The final frozen-source cycle on commit `91dc9336` (source digest `37b8892fd66af14cfca05a3c51782ee77f23a69935d87dd56c3f54def282a5b8`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `015168b9-0f46-45bc-8826-a42cc33b0efa`, outcome `pass`, evidence digest `bce925749455faf1af16c8c2b2fc2bd1405b0cd72b34c6a9330c541f390e2ace`.
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
