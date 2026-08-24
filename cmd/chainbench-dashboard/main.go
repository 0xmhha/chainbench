// Command chainbench-dashboard is the dashboard daemon (requirement #19): it hosts the
// obs event bus and run store behind an HTTP + SSE API and serves the dashboard
// page. Pipeline runs feed it live by POSTing obs events to /api/events; with
// -artifact-root it also serves completed-run session artifacts from disk.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/0xmhha/chainbench/internal/core/obs"
	"github.com/0xmhha/chainbench/internal/dashboard"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8787", "listen address")
	artifactRoot := flag.String("artifact-root", "", "directory of session artifacts to serve under /api/sessions (optional)")
	flag.Parse()

	bus := obs.NewBus()
	defer bus.Close()
	store := obs.NewMemStore()

	var opts []dashboard.Option
	if *artifactRoot != "" {
		opts = append(opts, dashboard.WithArtifactRoot(*artifactRoot))
	}
	srv := dashboard.NewServer(bus, store, opts...)

	fmt.Fprintf(os.Stderr, "chainbench-dashboard listening on http://%s\n", *addr)
	if err := http.ListenAndServe(*addr, srv); err != nil {
		fmt.Fprintln(os.Stderr, "chainbench-dashboard:", err)
		os.Exit(1)
	}
}
