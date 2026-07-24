// Command chainbench is the user-facing CLI (requirement #15) that drives the
// three-phase pipeline: setup (plan/launch a local network), verify (confirm
// block production over RPC), test (run cases), plus faucet and chain listing.
// It is built on cobra and imports all chain plugins for registration.
package main

import (
	"fmt"
	"os"

	_ "github.com/0xmhha/chainbench/pkg/chains/all"
	_ "github.com/0xmhha/chainbench/tests/all"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
