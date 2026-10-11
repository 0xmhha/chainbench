# WEB-14 GREEN

The final frozen-source cycle on commit `91dc9336` (source digest `37b8892fd66af14cfca05a3c51782ee77f23a69935d87dd56c3f54def282a5b8`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `ff0150f9-97c2-47ca-8099-6ed951979b5c`, outcome `pass`, evidence digest `999d7d89a998b230319b0b75b4af7d872bb179fda62795ea31506f75640cfd52`.
- An independent reproduction from a staged source copy without `.git` or `chainbench-out` printed `WEB-14 PASS (fresh reproduction; protected source/evidence unchanged)`.
- Required scenarios executed: 6 of 6; skipped: none; failed assertions: none.

Scenarios:

- `criteria`
- `modes-transports`
- `mandatory-checks`
- `live-go-tests`
- `spa-build`
- `routes`

Verify with `bash tests/webui/verify.sh --criterion WEB-14 --require-live --output chainbench-out/web-ui-acceptance`.
The summary receipts beside this file are copies of the published `result.json` and `evidence.json`. Screenshots, logs and native receipts they reference stay in the local, ignored acceptance output.
