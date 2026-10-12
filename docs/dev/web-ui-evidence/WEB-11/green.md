# WEB-11 GREEN

The final frozen-source cycle on commit `34c951a8` (source digest `470ab4bb2345f320eab9bb3e1dbeafecb130912558bc57fe4de49a910c8b8753`, unchanged from start to end) passed this criterion twice.

- Published capture: invocation `30fbc746-1857-4ca2-aeb5-76d7b9acb578`, outcome `pass`, evidence digest `405e15ed1a3aac2c535fbec20604cb20e0c2887632e0c8e1c3e7e0bdb76c0746`.
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
