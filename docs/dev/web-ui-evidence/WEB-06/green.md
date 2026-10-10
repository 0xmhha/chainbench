# WEB-06 GREEN

The final frozen-source cycle on commit `c5f23d85` (source digest `a0fcd339ca22ba0f098e85df1414f704d6a721c07a066d963c4372dd1363407f`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `7d5d7d66-cc07-4943-8033-3847d0f990d5`, outcome `pass`, evidence digest `77baab859467680e85614afeb6ee07207d30af686940e88ad6309f9c1afa308c`.
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
