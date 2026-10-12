package app

import (
	"context"
	"path/filepath"
	"slices"

	"github.com/0xmhha/chainbench/internal/core/node"
)

type webProcessObserver interface {
	PIDAlive(context.Context, int) (bool, error)
	Cmdline(context.Context, int) ([]string, error)
}

type webNodeObservation struct {
	State  string `json:"state"`
	PID    int    `json:"observedPid"`
	Reason string `json:"reason,omitempty"`
}

// observeWebNodeProcess checks the recorded process without altering its record.
// Missing or unreadable argv cannot establish ownership and never permits control.
func observeWebNodeProcess(ctx context.Context, observer webProcessObserver, binary string, ns node.Record) webNodeObservation {
	unknown := webNodeObservation{State: "unknown", Reason: "probe_unavailable"}
	if ns.PID <= 0 {
		return webNodeObservation{State: "unknown", Reason: "no_recorded_pid"}
	}
	if observer == nil || ctx.Err() != nil || binary == "" || len(ns.Args) == 0 {
		return unknown
	}
	alive, err := observer.PIDAlive(ctx, ns.PID)
	if err != nil {
		return unknown
	}
	missing := webNodeObservation{State: "missing", Reason: "recorded_pid_absent"}
	if !alive {
		return missing
	}
	argv, err := observer.Cmdline(ctx, ns.PID)
	if err != nil || ctx.Err() != nil {
		return unknown
	}
	if len(argv) == 0 || filepath.Clean(argv[0]) != filepath.Clean(binary) || !slices.Equal(argv[1:], ns.Args) {
		return webNodeObservation{State: "ownership_mismatch", Reason: "argv_mismatch"}
	}
	alive, err = observer.PIDAlive(ctx, ns.PID)
	if err != nil || ctx.Err() != nil {
		return unknown
	}
	if !alive {
		return missing
	}
	return webNodeObservation{State: "running", PID: ns.PID}
}
