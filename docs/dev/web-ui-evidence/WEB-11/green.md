# WEB-11 GREEN

The final frozen-source cycle on commit `91dc9336` (source digest `37b8892fd66af14cfca05a3c51782ee77f23a69935d87dd56c3f54def282a5b8`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `95114dbc-df73-49d7-8765-d27e2090cce8`, outcome `pass`, evidence digest `c2d87720a718ecf3f8557c3457290ac1cffe8b2183080a57866b39bf24d4475d`.
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
