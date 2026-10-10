# WEB-03 GREEN

The final frozen-source cycle on commit `c5f23d85` (source digest `a0fcd339ca22ba0f098e85df1414f704d6a721c07a066d963c4372dd1363407f`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `02ecab20-8471-456b-acac-913459b4737c`, outcome `pass`, evidence digest `52da7a6028cc36eb017b870da372f4f169c94b1977ae1b65b305754a83c5ff25`.
- An independent reproduction from a staged source copy without `.git` or `chainbench-out` printed `WEB-03 PASS (fresh reproduction; protected source/evidence unchanged)`.
- Required scenarios executed: 5 of 5; skipped: none; failed assertions: none.

Scenarios:

- `two-operator-shared-edit`
- `personal-ssh-binding`
- `ssh-access-permissions`
- `ports-paths-placement-validation`
- `secret-free-export`

Verify with `bash tests/webui/verify.sh --criterion WEB-03 --require-live --output chainbench-out/web-ui-acceptance`.
The summary receipts beside this file are copies of the published `result.json` and `evidence.json`. Screenshots, logs and native receipts they reference stay in the local, ignored acceptance output.
