# web — chainbench dashboard SPA

Svelte 5 + Vite source for the dashboard (decision D5). It consumes the Go
dashboard server's contract: the SSE stream at `/events` (obs.Event JSON) and the
run list at `/api/runs` (obs.RunRecord JSON).

## Build

```sh
npm --prefix web install
npm --prefix web run build
```

`vite build` writes the static bundle to `internal/dashboard/spa/` (committed),
which `internal/dashboard/spa.go` embeds with `go:embed`. The Go server serves it
at `/`, with `/chains`, `/tests`, `/monitoring`, `/history` and `/settings` routes.
Rebuild and commit `internal/dashboard/spa/` whenever the source
changes.

The normal daemon requires an account session in personal and team modes. The
first administrator uses the private `setup.token` in the Web data directory.
CLI event producers use `CHAINBENCH_PUBLISHER_TOKEN`, which authorizes event
ingestion only. `-legacy-observation` explicitly selects a loopback observer.

Provision native binaries with `-manifest-assets` to enable local and SSH durable jobs.
The browser reviews a short-lived plan before submitting it. Jobs survive browser
disconnects and become interrupted on server restart. SSH uses the initiating user's
encrypted credential binding and verifies the target identity, platform and paths.
Test jobs, asset replacement, metric charts and complete test/metric/log archives remain under implementation; the
approved Web specification retains all nine requirements.

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
npm --prefix web run dev   # Vite dev server; proxy /events + /api to a running dashboard
```
