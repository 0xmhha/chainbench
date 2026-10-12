# WEB-06 GREEN

The final frozen-source cycle on commit `34c951a8` (source digest `470ab4bb2345f320eab9bb3e1dbeafecb130912558bc57fe4de49a910c8b8753`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `302afbef-9a4d-4325-a6e3-dd22028b8667`, outcome `pass`, evidence digest `ba8f62095b6cfcb63d8b38f7ade6666673ec10cb15d32229b987308ba4a387db`.
- An independent reproduction from a staged source copy without `.git` or `chainbench-out` printed `WEB-06 PASS (fresh reproduction; protected source/evidence unchanged)`.
- Required scenarios executed: 6 of 6; skipped: none; failed assertions: none.

Scenarios:

- `ssh-native-deployment`
- `checksum-database-rpc-pid`
- `mid-deployment-ssh-failure`
- `mid-transfer-disconnect`
- `revocation-during-ssh`
- `no-automatic-retry`

Verify with `bash tests/webui/verify.sh --criterion WEB-06 --require-live --output chainbench-out/web-ui-acceptance`.
The summary receipts beside this file are copies of the published `result.json` and `evidence.json`. Screenshots, logs and native receipts they reference stay in the local, ignored acceptance output.
