# WEB-06 GREEN

The final frozen-source cycle on commit `91dc9336` (source digest `37b8892fd66af14cfca05a3c51782ee77f23a69935d87dd56c3f54def282a5b8`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `d3697c7d-5be8-4c7d-9b6a-a5707c456d2d`, outcome `pass`, evidence digest `1733012ed9f356596f5ee448a552b0865fe75f7f3d2923fe854cb5db9da0f35b`.
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
