# WEB-05 GREEN

The final frozen-source cycle on commit `91dc9336` (source digest `37b8892fd66af14cfca05a3c51782ee77f23a69935d87dd56c3f54def282a5b8`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `eb41f01d-91c4-4527-95c7-bd69bfd2ed88`, outcome `pass`, evidence digest `2132ed01eababab957caecb5cb581fbe2afab59826819e55435b6202aa0f2baf`.
- An independent reproduction from a staged source copy without `.git` or `chainbench-out` printed `WEB-05 PASS (fresh reproduction; protected source/evidence unchanged)`.
- Required scenarios executed: 10 of 10; skipped: none; failed assertions: none.

Scenarios:

- `grammar`
- `references`
- `v1-migration`
- `invalid-unknown`
- `live-execution`
- `attach-execution`
- `actions`
- `assertions`
- `readers`
- `arguments`

Verify with `bash tests/webui/verify.sh --criterion WEB-05 --require-live --output chainbench-out/web-ui-acceptance`.
The summary receipts beside this file are copies of the published `result.json` and `evidence.json`. Screenshots, logs and native receipts they reference stay in the local, ignored acceptance output.
