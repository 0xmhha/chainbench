# web — chainbench dashboard SPA

Svelte 5 + Vite source for the dashboard (decision D5). It consumes the Go
dashboard server's `/api/v1` contract (`docs/dev/web-ui.openapi.yaml`): REST
reads through `src/api-client.mjs` with the decoders in `src/api-decode.mjs`, and
the one SSE stream at `/api/v1/events` for job changes, collector observations and
engine events.

## Build

```sh
npm --prefix web install
npm --prefix web run build
```

`vite build` writes the static bundle to `internal/dashboard/spa/` (committed),
which `internal/dashboard/spa.go` embeds with `go:embed`. The Go server serves it
at `/`, with `/chains`, `/tests`, `/monitoring`, `/history` and `/settings` routes
and the detail pages `/jobs/<id>` and `/nodes/<id>/<name>`. The server also has a
no-JS page at `/legacy` and a health check at `/healthz`. Rebuild and commit
`internal/dashboard/spa/` whenever the source changes, and build the SPA before
the Go binary that embeds it.

`npm --prefix web run check` runs svelte-check, the gate CI runs on every change.
The Node tests for the SPA live in `tests/webui/*.test.mjs`
(`node --test tests/webui/*.test.mjs`).

The normal daemon defaults to `-accounts=false`: on loopback, the Web opens
without registration, with a stable local operator, CSRF protection and audit records.
Use `-accounts=true` to enable registration, login and account roles. The first
administrator then uses the private `setup.token` in the Web data directory.
Account-free access is limited to localhost and loopback addresses, refuses to
start with team mode, `-trusted-proxy` or TLS, and rejects requests that carry
reverse-proxy forwarding headers. Personal
mode listens only on loopback; account-enabled team mode on a reachable address needs
`-tls-cert`/`-tls-key` or `-trusted-proxy`. CLI event producers use
`CHAINBENCH_PUBLISHER_TOKEN`, which authorizes event ingestion only.
`-legacy-observation` explicitly selects a loopback observer without control
APIs for producers without a token. The repository README shows each command.

The local operator retains its credential ownership across restart. Changing
account modes preserves existing users and artifacts but does not adopt the
other mode's personal credentials or job identities.

The Web does not carry its own execution engine. A job runs the `chainbench` CLI,
found beside the dashboard binary, then on PATH, or named with `-chainbench`
(`internal/webcli`).

Provision native binaries with `-manifest-assets` to enable local and SSH durable jobs.
The browser reviews a short-lived plan before submitting it. Jobs survive browser
disconnects and become interrupted on server restart. SSH uses the initiating user's
encrypted credential binding and verifies the target identity, platform and paths.
Test jobs, asset replacement, node metric charts and test/metric/log archives are
implemented; the owner of a finished job can plan it again from its recorded input
and start it after review. What remains open is listed in
[`docs/dev/web-ui-status.md`](../docs/dev/web-ui-status.md).

History supports server-side search and filters, verdict comparisons with explicit
compatibility limits, redacted JSON downloads and administrator deletion of the
server-owned capture. Original CLI artifacts and live node files are preserved.

For development proof using the three real local chain binaries:

```sh
python3 tests/webui/run_native_jobs.py
python3 tests/webui/run_ssh_jobs.py
```

This checks browser submission, disconnect, native database genesis reads and
owned node start/stop, browser history export/deletion and restart preservation.
The SSH proof uses a separately owned loopback SSH daemon, uploads a native binary,
initializes four actual databases and verifies private credentials and local/SSH aliases.
It does not replace the fourteen acceptance criteria.

## Dev

```sh
npm --prefix web run dev   # Vite dev server for the SPA alone
```

`web/vite.config.js` configures no dev proxy, so the dev server does not forward
`/api` to a running dashboard. To see the SPA against real data, build it and run
`chainbench-dashboard`.
