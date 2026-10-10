# WEB-09 GREEN

The final frozen-source cycle on commit `c5f23d85` (source digest `a0fcd339ca22ba0f098e85df1414f704d6a721c07a066d963c4372dd1363407f`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `1b87cefb-6edf-4640-9bbe-87c4a89ab72b`, outcome `pass`, evidence digest `c64fd2f2ca0dfae4ba111cefaf8743ae225232ddf468cd62620379598ee57400`.
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
