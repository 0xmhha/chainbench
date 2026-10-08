# WEB-01 verification

For saved configuration refresh regressions, run
`WEBUI_RUNTIME_ROOT=/private/tmp/cbui python3 tests/webui/run_catalog_refresh.py`
and `WEBUI_RUNTIME_ROOT=/private/tmp/cbui python3 tests/webui/run_genesis_test_jobs.py`.
The first checks chain/server/path/workspace saves and pinned revisions without
node effects; the second also checks delayed case responses and native sessions.
These development receipts do not award complete Seed criteria.

`WEBUI_RUNTIME_ROOT=/private/tmp/cbui python3 tests/webui/run_node_reset.py` checks
the actual non-producer reset UI and native DB head at genesis across three chains,
producer preservation, stopped state and explicit relaunch. It also refuses
producer and PID-zero reset plans. File/physical-path and foreign-process refusal
have application unit coverage. This remains partial WEB-07 verification.

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
The security runner seeds legacy node-key declarations only into its stopped,
disposable fixture store through `fixtures/legacykeys`. Restart must quarantine
their full histories and saved previews before authenticated reads or exports.
It checks old import refusal, encrypted storage and actual SSE masking. This
fixture never modifies a user's dashboard storage or bypasses a live API guard.
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
that the plan UI displays the owning job/workspace/actor and disables execution,
that owner controls remain allowed, and that verified removal of all four actual
node directories permits the alias to refresh its conflict review and execute. The engine retains its audit
record with an empty node table. Results go to
`chainbench-out/web-ui-development/resource-holds/receipt.json`.
This is partial development evidence, not full WEB-11/12 acceptance or a live
restart/reconciliation proof.

## Recorded process checks

The resource retention fixture also starts a native WBFT node, observes its
recorded process in the UI and substitutes an exclusively owned sleep process
into a fixture record. Planning stop must return 409, expose no controls for the
mismatch, leave the foreign process running, and preserve the original record.
A separate application regression verifies rejection immediately before the stop
verb and confirms the process and record remain untouched.

Run the SSH development fixture with short paths:
`WEBUI_RUNTIME_ROOT=/private/tmp/s python3 tests/webui/run_ssh_jobs.py`.
Its native node must serve the expected chain RPC and pass live process observation
before the stop check. This avoids mistaking a recorded PID for a surviving node
when Unix IPC path limits make startup fail. All resources belong to isolated
UUID fixtures; these checks do not award full restart/reconciliation acceptance.

## Restart during a native test

`WEBUI_RUNTIME_ROOT=/private/tmp/cb-restart python3 tests/webui/run_interrupted_test.py`
runs a native four-node WBFT test and kills only the disposable dashboard while
the test waits for a deliberately distant block. The runner restarts the same
service and data store; the browser logs in again and checks interrupted state,
idempotent replay without execution, unchanged launch record and node directory
identity, live recorded PIDs, alias exclusion and explicit new control/test jobs.
It retains the original interrupted result. The fixture's restart request file
is not a product endpoint and contains no credentials. All launched processes
and data belong to its private UUID tree. Results are in
`chainbench-out/web-ui-development/interrupted-test/receipt.json`.
This does not award WEB-12 or cover PID-free residual discovery/reconciliation.

## Saved preset execution

`WEBUI_RUNTIME_ROOT=/private/tmp/cb-preset python3 tests/webui/run_preset_jobs.py`
selects an immutable saved chain-preset revision for actual composition jobs.
The browser checks review and submission, revision conflicts and accepted content
preservation. All three native chains initialize five databases whose dumped
genesis carries the reviewed chain ID. Recorded launch arguments contain scoped
`maxpeers` and `cache` options from the engine mapping backed by
`docs/chain-analysis/`; the endpoint-first table preserves its order. Each archive
endpoint must actually start, serve its chain ID over RPC, pass process observation
and stop explicitly. Bounded requests use an available local address. Results are
in `chainbench-out/web-ui-development/preset-jobs/receipt.json` and do not award
full WEB-01/02 acceptance or every option/asset scenario.


The owned browser helper disposes request contexts and disconnects the client,
terminates its exact Chrome process, and closes its dedicated WebSocket listener.
After its exact Chrome process exits, the helper also destroys its own stdio
pipe ends: detached crashpad/updater descendants may inherit stderr and delay
ChildProcess.close. Existing updater/browser processes are never terminated.
The listener cleanup uses the testing hook in the pinned Playwright 1.58.2
implementation; it is confined to test tooling. Run `browser_shutdown.mjs` through
`browser_process.run_browser(..., timeout=30)` to verify actual SSE/request socket
teardown and process exit without affecting an existing browser. The
`--stalled-close` variant kills only its owned Chrome process group and withholds
the client close acknowledgement, proving that teardown remains bounded. Native fixture
shutdown or database-lock mistakes are not product RED evidence.
