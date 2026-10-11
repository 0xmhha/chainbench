# WEB-13 GREEN

The final frozen-source cycle on commit `91dc9336` (source digest `37b8892fd66af14cfca05a3c51782ee77f23a69935d87dd56c3f54def282a5b8`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `fabec228-31b6-41ea-a9e6-a71ede87d92d`, outcome `pass`, evidence digest `99e05f94efeae45e0bad279b2448928c223022c255480f690b9830e8da77b7c2`.
- An independent reproduction from a staged source copy without `.git` or `chainbench-out` printed `WEB-13 PASS (fresh reproduction; protected source/evidence unchanged)`.
- Required scenarios executed: 6 of 6; skipped: none; failed assertions: none.

Scenarios:

- `normal-end-retain-and-cleanup`
- `user-cancel-retain-and-cleanup`
- `executor-and-admin-cancel`
- `cleanup-failure-reported`
- `revocation-retains-and-blocks-access`
- `attach-protection`

Verify with `bash tests/webui/verify.sh --criterion WEB-13 --require-live --output chainbench-out/web-ui-acceptance`.
The summary receipts beside this file are copies of the published `result.json` and `evidence.json`. Screenshots, logs and native receipts they reference stay in the local, ignored acceptance output.
