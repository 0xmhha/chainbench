# WEB-07 GREEN

The final frozen-source cycle on commit `c5f23d85` (source digest `a0fcd339ca22ba0f098e85df1414f704d6a721c07a066d963c4372dd1363407f`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `9230a0f6-9d91-4f6d-a5de-3de3c788f686`, outcome `pass`, evidence digest `3b3d90208d5608c7ea2cc49e883420e2577f4fb3f39b87c2ec2f9c80fa344b49`.
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
