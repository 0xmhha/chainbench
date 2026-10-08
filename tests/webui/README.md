# WEB-01 verification

Install dependencies with `npm --prefix web ci` and `npm --prefix tests/webui ci`.
The browser fixture uses installed Chrome with a separate temporary profile.
Set `WEBUI_BROWSER_CHANNEL=chromium` to use a Playwright-installed Chromium.

Run from the repository root:

```sh
bash tests/webui/verify.sh --criterion WEB-01 --require-live --output chainbench-out/web-ui-acceptance
```

Each invocation builds the embedded SPA and Go server, validates the engine
usecase, starts an isolated server under `/private/tmp/chainbench-web-ui-e2e`,
and exercises the three chain forms in a real browser. The server process is
terminated in a finally block. Screenshots, observations and build digests
are retained in the workspace result directory.

The present implementation edits count topology and sync mode. Other preset
fields and node tables are preserved, but their complete structured editing,
shared parameter schemas and binary-backed ChainSurface coverage are unfinished.
Consequently this runner records **incomplete** and exits nonzero even when the
implemented form checks pass. It never reports partial coverage as WEB-01 PASS.
The independent `evidence.py` verifier also rejects incomplete observations,
reduced scenario denominators, stale invocation IDs and digest mismatches.

The editor is enabled explicitly with the dashboard's `-chain-presets
presets/chain` flag. It validates without saving or executing declarations.

WEB-03 uses the real embedded SPA, two provisioned operator accounts, and an
invocation-specific OpenSSH daemon under `/private/tmp/chainbench-web-ui-e2e`:

```sh
bash tests/webui/verify.sh --criterion WEB-03 --require-live --output chainbench-out/web-ui-acceptance
```

The runner creates and removes disposable private fixture keys, checks host-key
identity, observes accepted and rejected SSH authentication and target-directory
permissions, restarts the dashboard, and verifies encrypted overlays still work.
It fails when a browser, SSH target, required scenario, or assertion is absent.
`WEBUI_BROWSER_CHANNEL` selects the installed Playwright browser (default `chrome`).

The dashboard's `-deployment-accounts` adapter accepts a private (0600) JSON array
of `{id, username, role, passwordHash}` records, with bcrypt password hashes. The
`DeploymentAuthenticator` boundary supplies trusted identities to these APIs; the
fixture provisions its own accounts. This adapter is scoped to the deployment
editor. `-deployment-root` defaults to `chainbench-out/web-ui` and contains the
atomic shared revision snapshot, encrypted credentials, private bindings, audit
metadata, and a private encryption key.

Shared documents use `POST /api/v1/documents`, `PATCH /api/v1/documents/{id}` with
`If-Match`, and `GET /api/v1/documents/{id}?revision=N`. Workspace references pin
immutable revisions. Validation uses `/api/v1/documents/validate`; the structured
field contract is `/api/v1/contracts/deployment`. Credentials return metadata only.
The editor's private overlay extension is
`GET/PUT /api/v1/workspaces/{id}/credential-bindings` (PUT body:
`{serverRef, credentialId}`). It never changes the shared workspace revision or
exports a private binding. `/api/v1/credentials/{id}/check` performs real SSH
host-key verification, authentication, and a read-only data-root permission probe.

## Immutable verification

Generate receipts from frozen source with `verify.sh --criterion WEB-NN --require-live --capture --output chainbench-out/web-ui-acceptance`. The same command without `--capture` validates source-bound published receipts and runs a fresh independent live reproduction in a staged source tree. Missing, stale or incomplete evidence fails. Rechecks leave source, embedded assets and published evidence unchanged. Set `WEBUI_RUNTIME_ROOT` to an explicitly allowed scratch directory in isolated environments; its default is `/private/tmp/chainbench-web-ui-e2e`. No criterion denominator or live requirement is reduced.

## Resource retention development proof

`WEBUI_RUNTIME_ROOT=/private/tmp/cb-holds python3 tests/webui/run_resource_holds.py`
uses native WBFT setup and an exclusively owned dashboard/browser fixture. It
checks that a different workspace alias receives 409 after retained completion,
that owner controls remain allowed, and that verified removal of all four actual
node directories permits the alias to execute. The engine retains its audit
record with an empty node table. Results go to
`chainbench-out/web-ui-development/resource-holds/receipt.json`.
This is partial development evidence, not full WEB-11/12 acceptance or a live
restart/reconciliation proof.
