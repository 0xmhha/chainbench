# WEB-10 GREEN

The final frozen-source cycle on commit `c5f23d85` (source digest `a0fcd339ca22ba0f098e85df1414f704d6a721c07a066d963c4372dd1363407f`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `b943821e-7078-4e96-b898-c676069bedf6`, outcome `pass`, evidence digest `7ca8bc450d3cae52153867a598070e500c457b9df1435350bb7a2af09ff1a473`.
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
