package app

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
)

type webProcessDiscovery interface {
	FindBinary(context.Context, string) ([]int, error)
}

type webDiscoveringObserver struct {
	webProcessObserver
	commander process.Commander
}

func (o webDiscoveringObserver) FindBinary(ctx context.Context, name string) ([]int, error) {
	return webFindProcessCandidates(ctx, o.commander, name)
}

func (o webDarwinObserver) FindBinary(ctx context.Context, name string) ([]int, error) {
	return webFindProcessCandidates(ctx, o.commander, name)
}

// webFindProcessCandidates requires a readable process table; command errors
// must not become an empty process set that could falsely establish vacancy.
func webFindProcessCandidates(ctx context.Context, commander process.Commander, name string) ([]int, error) {
	if commander == nil || name == "" || ctx.Err() != nil {
		return nil, errors.New("process discovery unavailable")
	}
	raw, err := commander.Run(ctx, "ps -ax -o pid= -o comm=")
	if err != nil || strings.TrimSpace(raw) == "" {
		return nil, errors.New("process discovery unavailable")
	}
	pids := []int{}
	for _, line := range strings.Split(strings.TrimSpace(raw), "\n") {
		line = strings.TrimSpace(line)
		split := strings.IndexFunc(line, unicode.IsSpace)
		if split < 1 {
			return nil, errors.New("process discovery unavailable")
		}
		pid, err := strconv.Atoi(line[:split])
		command := strings.TrimSpace(line[split:])
		if err != nil || pid <= 0 || command == "" {
			return nil, errors.New("process discovery unavailable")
		}
		if filepath.Base(command) == filepath.Base(name) {
			pids = append(pids, pid)
		}
	}
	slices.Sort(pids)
	return slices.Compact(pids), nil
}

// observeWebNodeVacancy never adopts a PID. An exact unrecorded launch or a
// conflicting launch referring to the same node paths prevents all controls.
func observeWebNodeVacancy(ctx context.Context, observer webProcessObserver, binary string, ns node.Record) webNodeObservation {
	unknown := webNodeObservation{State: "unknown", Reason: "discovery_unavailable"}
	discovery, ok := observer.(webProcessDiscovery)
	if !ok || observer == nil || ns.PID != 0 || binary == "" || len(ns.Args) == 0 || ctx.Err() != nil {
		return unknown
	}
	pids, err := discovery.FindBinary(ctx, filepath.Base(binary))
	if err != nil || len(pids) > 4096 {
		return unknown
	}
	for _, pid := range pids {
		if pid <= 0 {
			return unknown
		}
		alive, err := observer.PIDAlive(ctx, pid)
		if err != nil {
			return unknown
		}
		if !alive {
			continue
		}
		argv, err := observer.Cmdline(ctx, pid)
		if err != nil || len(argv) == 0 {
			return unknown
		}
		if filepath.Clean(argv[0]) == filepath.Clean(binary) && slices.Equal(argv[1:], ns.Args) {
			candidate := ns
			candidate.PID = pid
			observed := observeWebNodeProcess(ctx, observer, binary, candidate)
			if observed.State != "running" {
				return unknown
			}
			return webNodeObservation{State: "unrecorded_running", PID: pid, Reason: "unrecorded_pid_found"}
		}
		for _, arg := range argv[1:] {
			value := arg
			if _, after, found := strings.Cut(arg, "="); found {
				value = after
			}
			if value != "" && (value == ns.DataDir || value == ns.ConfigPath) {
				return webNodeObservation{State: "ownership_mismatch", Reason: "unrecorded_argv_mismatch"}
			}
		}
	}
	after, err := discovery.FindBinary(ctx, filepath.Base(binary))
	if err != nil || ctx.Err() != nil || !slices.Equal(pids, after) {
		return unknown
	}
	return webNodeObservation{State: "stopped", Reason: "no_matching_process"}
}
