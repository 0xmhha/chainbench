# WEB-11 GREEN

The final frozen-source cycle on commit `c5f23d85` (source digest `a0fcd339ca22ba0f098e85df1414f704d6a721c07a066d963c4372dd1363407f`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `ee689e43-cd7f-4a0c-acbf-8d0d5d6f19e0`, outcome `pass`, evidence digest `6c3f9fc2e8639139487936b4fa33f22362ceb907f93106fb7de2ecf4fb3d215e`.
- An independent reproduction from a staged source copy without `.git` or `chainbench-out` printed `WEB-11 PASS (fresh reproduction; protected source/evidence unchanged)`.
- Required scenarios executed: 6 of 6; skipped: none; failed assertions: none.

Scenarios:

- `alias-path-port-conflict-with-owner`
- `local-ssh-alias-same-physical`
- `process-and-binary-conflict`
- `independent-resources-parallel`
- `test-internal-vs-external-control`
- `cleanup-releases-claims`

Verify with `bash tests/webui/verify.sh --criterion WEB-11 --require-live --output chainbench-out/web-ui-acceptance`.
The summary receipts beside this file are copies of the published `result.json` and `evidence.json`. Screenshots, logs and native receipts they reference stay in the local, ignored acceptance output.
