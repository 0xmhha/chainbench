# WEB-07 GREEN

The final frozen-source cycle on commit `91dc9336` (source digest `37b8892fd66af14cfca05a3c51782ee77f23a69935d87dd56c3f54def282a5b8`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `6f177127-cbc2-4dff-bd50-862cc05c374f`, outcome `pass`, evidence digest `f6e8eeaa6af8fa34248d2a015f3fa2b8b64d15af51db86d0c9e59ea3b809ab0d`.
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
