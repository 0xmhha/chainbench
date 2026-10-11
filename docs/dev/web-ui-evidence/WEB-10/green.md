# WEB-10 GREEN

The final frozen-source cycle on commit `91dc9336` (source digest `37b8892fd66af14cfca05a3c51782ee77f23a69935d87dd56c3f54def282a5b8`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `27c5bc1e-7d64-4063-81f1-e5233894e6d8`, outcome `pass`, evidence digest `7cc4cc7b580449a5e0decf3d4965c64805dea90d3cb9096d7138916131a8eae7`.
- An independent reproduction from a staged source copy without `.git` or `chainbench-out` printed `WEB-10 PASS (fresh reproduction; protected source/evidence unchanged)`.
- Required scenarios executed: 7 of 7; skipped: none; failed assertions: none.

Scenarios:

- `personal-team-login`
- `role-ui-api-matrix`
- `csrf-ownership`
- `foreign-credential-refusal`
- `secret-free-output`
- `bootstrap-restrictions`
- `encrypted-storage-session`

Verify with `bash tests/webui/verify.sh --criterion WEB-10 --require-live --output chainbench-out/web-ui-acceptance`.
The summary receipts beside this file are copies of the published `result.json` and `evidence.json`. Screenshots, logs and native receipts they reference stay in the local, ignored acceptance output.
