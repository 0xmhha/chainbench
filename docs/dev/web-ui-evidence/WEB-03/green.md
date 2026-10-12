# WEB-03 GREEN

The final frozen-source cycle on commit `34c951a8` (source digest `470ab4bb2345f320eab9bb3e1dbeafecb130912558bc57fe4de49a910c8b8753`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `14f42b1d-d119-4920-9926-5ec5e23b7dcc`, outcome `pass`, evidence digest `5d57b1e3acee665f5708186e1ebba951227e16d2236ee36878e1e5406f07ad04`.
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
