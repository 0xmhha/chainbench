# WEB-14 GREEN

The final frozen-source cycle on commit `c5f23d85` (source digest `a0fcd339ca22ba0f098e85df1414f704d6a721c07a066d963c4372dd1363407f`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `676748b7-5e39-4da9-8152-53ee3783224a`, outcome `pass`, evidence digest `a6079657b687a2daa3c1fd13f8a3afa2b6eb01e7487e7214f29958cd13f7af09`.
- An independent reproduction from a staged source copy without `.git` or `chainbench-out` printed `WEB-14 PASS (fresh reproduction; protected source/evidence unchanged)`.
- Required scenarios executed: 6 of 6; skipped: none; failed assertions: none.

Scenarios:

- `criteria`
- `modes-transports`
- `mandatory-checks`
- `live-go-tests`
- `spa-build`
- `routes`

Verify with `bash tests/webui/verify.sh --criterion WEB-14 --require-live --output chainbench-out/web-ui-acceptance`.
The summary receipts beside this file are copies of the published `result.json` and `evidence.json`. Screenshots, logs and native receipts they reference stay in the local, ignored acceptance output.
