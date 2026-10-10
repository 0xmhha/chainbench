# WEB-13 GREEN

The final frozen-source cycle on commit `c5f23d85` (source digest `a0fcd339ca22ba0f098e85df1414f704d6a721c07a066d963c4372dd1363407f`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `29011704-b343-446e-ad93-0f0b31358d4e`, outcome `pass`, evidence digest `01b1b8e967bae1d9bc677474449d2474aa6edcb8a4e66c384cd6c902cb968815`.
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
