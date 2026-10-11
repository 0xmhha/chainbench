# WEB-09 GREEN

The final frozen-source cycle on commit `91dc9336` (source digest `37b8892fd66af14cfca05a3c51782ee77f23a69935d87dd56c3f54def282a5b8`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `bc41449b-d5c5-409a-bcff-327240e88a6a`, outcome `pass`, evidence digest `286c17214607597bcc015fd305932d8bbe474d904f5097f688395368cd213ff6`.
- An independent reproduction from a staged source copy without `.git` or `chainbench-out` printed `WEB-09 PASS (fresh reproduction; protected source/evidence unchanged)`.
- Required scenarios executed: 8 of 8; skipped: none; failed assertions: none.

Scenarios:

- `search-filter-detail`
- `compatible-comparison-and-reasons`
- `safe-export`
- `retention-across-restart`
- `admin-only-deletion`
- `protected-active-shared-live`
- `partial-deletion-failure`
- `run-metric-log-archive`

Verify with `bash tests/webui/verify.sh --criterion WEB-09 --require-live --output chainbench-out/web-ui-acceptance`.
The summary receipts beside this file are copies of the published `result.json` and `evidence.json`. Screenshots, logs and native receipts they reference stay in the local, ignored acceptance output.
