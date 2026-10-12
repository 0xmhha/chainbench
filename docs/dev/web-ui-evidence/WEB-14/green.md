# WEB-14 GREEN

The final frozen-source cycle on commit `34c951a8` (source digest `470ab4bb2345f320eab9bb3e1dbeafecb130912558bc57fe4de49a910c8b8753`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `da63a50c-ef13-4df6-a603-92fce2cea48c`, outcome `pass`, evidence digest `5cf3c22ae67e6e011f5931b6de74ef0b71d1bfc970c5205a52b3ab060e23d88d`.
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
