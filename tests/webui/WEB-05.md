# WEB-05 verification

Run the acceptance check without changing workspace files:

```sh
bash tests/webui/verify.sh --criterion WEB-05 --require-live --output chainbench-out/web-ui-acceptance
```

The checker copies sources and dependencies to an isolated temporary directory
under `/private/tmp/chainbench-web-ui-e2e`. Tests, the SPA build, dashboard store,
engine network and session files run there. A fresh invocation is required;
previous workspace receipts are never used to satisfy this check.

To generate the requested workspace receipts for review, run explicitly:

```sh
python3 tests/webui/run_web05.py chainbench-out/web-ui-acceptance
```

This command replaces only WEB-05 acceptance output. Builds and tests still run
against a private source copy. The embedded frontend is archived with the receipt
so its digest does not depend on later workspace builds.

Both commands exit nonzero for incomplete coverage. The current live fixture
executes three reference/migration cases only; it does not execute every builtin
and argument. Browser field rendering is not proof of round-trip or execution.
The independent verifier derives argument coverage from the contract, including
all reader-specific alternatives, and rejects reduced or duplicate coverage.
Full WEB-05 live acceptance remains incomplete until those cases are supplied.
