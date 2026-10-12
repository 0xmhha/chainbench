# WEB-10 GREEN

The final frozen-source cycle on commit `34c951a8` (source digest `470ab4bb2345f320eab9bb3e1dbeafecb130912558bc57fe4de49a910c8b8753`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `89200345-c5b4-4376-9a9c-1afcb5e6196d`, outcome `pass`, evidence digest `bb09f62a38546a9af7a638fb27a7658612eb1a0db8b7d44b33e66d3210e22c99`.
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
