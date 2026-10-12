# WEB-04 GREEN

The final frozen-source cycle on commit `34c951a8` (source digest `470ab4bb2345f320eab9bb3e1dbeafecb130912558bc57fe4de49a910c8b8753`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `532f3650-42c1-4e44-b099-e8e9ab634771`, outcome `pass`, evidence digest `68ac1101971e2f189b4e0d892594eb06ea5f62bee94bb334f90a26c39caf3f7a`.
- An independent reproduction from a staged source copy without `.git` or `chainbench-out` printed `WEB-04 PASS (fresh reproduction; protected source/evidence unchanged)`.
- Required scenarios executed: 10 of 10; skipped: none; failed assertions: none.

Scenarios:

- `builtin-stablenet`
- `external-stablenet`
- `builtin-wbft`
- `external-wbft`
- `builtin-wemix`
- `external-wemix`
- `unsupported-family`
- `unsupported-dialect`
- `wrong-binary-identity`
- `manifest-management`

Verify with `bash tests/webui/verify.sh --criterion WEB-04 --require-live --output chainbench-out/web-ui-acceptance`.
The summary receipts beside this file are copies of the published `result.json` and `evidence.json`. Screenshots, logs and native receipts they reference stay in the local, ignored acceptance output.
