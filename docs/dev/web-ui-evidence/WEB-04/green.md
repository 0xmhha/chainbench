# WEB-04 GREEN

The final frozen-source cycle on commit `91dc9336` (source digest `37b8892fd66af14cfca05a3c51782ee77f23a69935d87dd56c3f54def282a5b8`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `90e1ffeb-279f-4f1b-8f0d-74047d302a29`, outcome `pass`, evidence digest `14812a20946b4af3c537826bc08099573bc53047dbbf810660a5c049505f4953`.
- An independent reproduction from a staged source copy without `.git` or `chainbench-out` printed `WEB-04 PASS (fresh reproduction; protected source/evidence unchanged)`.
- Required scenarios executed: 10 of 10; skipped: none; failed assertions: none.

Scenarios:

- `builtin-stablenet`
- `external-stablenet`
- `builtin-wbft`
- `external-wbft`
- `builtin-wemix`
- `external-wemix`
- `unsupported-family`
- `unsupported-dialect`
- `wrong-binary-identity`
- `manifest-management`

Verify with `bash tests/webui/verify.sh --criterion WEB-04 --require-live --output chainbench-out/web-ui-acceptance`.
The summary receipts beside this file are copies of the published `result.json` and `evidence.json`. Screenshots, logs and native receipts they reference stay in the local, ignored acceptance output.
