# web — chainbench dashboard SPA

Svelte 5 + Vite source for the dashboard (decision D5). It consumes the Go
dashboard server's contract: the SSE stream at `/events` (obs.Event JSON) and the
run list at `/api/runs` (obs.RunRecord JSON).

## Build

```sh
npm --prefix web install
npm --prefix web run build
```

`vite build` writes the static bundle to `pkg/dashboard/spa/` (committed), which
`pkg/dashboard/spa.go` embeds with `go:embed`. The Go server serves it under
`/app/`. Rebuild and commit `pkg/dashboard/spa/` whenever the source changes.

## Dev

```sh
npm --prefix web run dev   # Vite dev server; proxy /events + /api to a running dashboard
```
