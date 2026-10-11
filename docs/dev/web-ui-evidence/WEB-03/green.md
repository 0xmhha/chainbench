# WEB-03 GREEN

The final frozen-source cycle on commit `91dc9336` (source digest `37b8892fd66af14cfca05a3c51782ee77f23a69935d87dd56c3f54def282a5b8`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `65285c98-3eb6-470c-987c-a1b5ed449c76`, outcome `pass`, evidence digest `ef94e094fd7f373d2a0e5c483fcef25dfc06ec0370c685a3e67db41385574918`.
- An independent reproduction from a staged source copy without `.git` or `chainbench-out` printed `WEB-03 PASS (fresh reproduction; protected source/evidence unchanged)`.
- Required scenarios executed: 5 of 5; skipped: none; failed assertions: none.

Scenarios:

- `two-operator-shared-edit`
- `personal-ssh-binding`
- `ssh-access-permissions`
- `ports-paths-placement-validation`
- `secret-free-export`

Verify with `bash tests/webui/verify.sh --criterion WEB-03 --require-live --output chainbench-out/web-ui-acceptance`.
The summary receipts beside this file are copies of the published `result.json` and `evidence.json`. Screenshots, logs and native receipts they reference stay in the local, ignored acceptance output.
