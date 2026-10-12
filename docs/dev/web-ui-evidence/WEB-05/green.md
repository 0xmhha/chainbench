# WEB-05 GREEN

The final frozen-source cycle on commit `34c951a8` (source digest `470ab4bb2345f320eab9bb3e1dbeafecb130912558bc57fe4de49a910c8b8753`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `341e8ab6-4162-49ab-8e17-03e1dedb7b09`, outcome `pass`, evidence digest `136eab04486442416b9513f3f9dfd2bca98f0c49b8d465372c553cbe8aa23779`.
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
