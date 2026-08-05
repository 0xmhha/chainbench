// Command chainbenchd is the dashboard daemon (requirement #19): it hosts the
// obs event bus and run store behind an HTTP + SSE API and serves the dashboard
// page. Pipeline runs feed it by POSTing obs events to /api/events (or, in a
// future in-process integration, by publishing to the shared bus).
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
	flag.Parse()

	bus := obs.NewBus()
	defer bus.Close()
	store := obs.NewMemStore()
	srv := dashboard.NewServer(bus, store)

	fmt.Fprintf(os.Stderr, "chainbenchd listening on http://%s\n", *addr)
	if err := http.ListenAndServe(*addr, srv); err != nil {
		fmt.Fprintln(os.Stderr, "chainbenchd:", err)
		os.Exit(1)
	}
}
