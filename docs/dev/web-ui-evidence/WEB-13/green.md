# WEB-13 GREEN

The final frozen-source cycle on commit `34c951a8` (source digest `470ab4bb2345f320eab9bb3e1dbeafecb130912558bc57fe4de49a910c8b8753`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `c74f2c8a-0a60-4196-a359-dca116a2d512`, outcome `pass`, evidence digest `3d58f5126ad2ff9f6db69e518f7d9f5ef33337ae29adf64b363d73cc2873caad`.
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
