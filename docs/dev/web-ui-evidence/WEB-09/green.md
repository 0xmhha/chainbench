# WEB-09 GREEN

The final frozen-source cycle on commit `34c951a8` (source digest `470ab4bb2345f320eab9bb3e1dbeafecb130912558bc57fe4de49a910c8b8753`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `d42aad09-fe51-42c1-b3c9-6a49cbb06ddf`, outcome `pass`, evidence digest `1ba68d166536ea04f3d1783afe67ec97d80c6aaf5a006ad50573c8a1451639a2`.
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
