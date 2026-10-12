# WEB-07 GREEN

The final frozen-source cycle on commit `34c951a8` (source digest `470ab4bb2345f320eab9bb3e1dbeafecb130912558bc57fe4de49a910c8b8753`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `299fede7-4a77-462f-b777-3a0e69af1a4a`, outcome `pass`, evidence digest `4c981e0714f4942fc2e400bb7dc9f651882a2dafeb8d94c733fc1057d4720334`.
- An independent reproduction from a staged source copy without `.git` or `chainbench-out` printed `WEB-07 PASS (fresh reproduction; protected source/evidence unchanged)`.
- Required scenarios executed: 6 of 6; skipped: none; failed assertions: none.

Scenarios:

- `owned-start-stop`
- `config-replacement-restart`
- `binary-replacement-restart`
- `non-producer-reset-stopped`
- `producer-and-attach-refused`
- `cli-semantics-preserved`

Verify with `bash tests/webui/verify.sh --criterion WEB-07 --require-live --output chainbench-out/web-ui-acceptance`.
The summary receipts beside this file are copies of the published `result.json` and `evidence.json`. Screenshots, logs and native receipts they reference stay in the local, ignored acceptance output.
