# WEB-12 GREEN

The final frozen-source cycle on commit `34c951a8` (source digest `470ab4bb2345f320eab9bb3e1dbeafecb130912558bc57fe4de49a910c8b8753`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `308af749-d5bd-4444-8439-572c6b844a19`, outcome `pass`, evidence digest `9a3ca3cf27f8558c3dc11089eb8158e5e5809c1e7ded573e53f98dc042e41f93`.
- An independent reproduction from a staged source copy without `.git` or `chainbench-out` printed `WEB-12 PASS (fresh reproduction; protected source/evidence unchanged)`.
- Required scenarios executed: 5 of 5; skipped: none; failed assertions: none.

Scenarios:

- `restart-marks-running-work-interrupted`
- `actual-state-rechecked`
- `records-and-claims-preserved`
- `no-automatic-redeploy-or-reset`
- `explicit-rerun-new-plan`

Verify with `bash tests/webui/verify.sh --criterion WEB-12 --require-live --output chainbench-out/web-ui-acceptance`.
The summary receipts beside this file are copies of the published `result.json` and `evidence.json`. Screenshots, logs and native receipts they reference stay in the local, ignored acceptance output.
