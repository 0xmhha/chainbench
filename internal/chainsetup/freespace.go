package chainsetup

import (
	"context"
	"fmt"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/resource"
)

// CheckFreeSpace refuses a composition whose data root is short of space on any
// machine it was placed on.
//
// A full disk used to surface as whatever the first write that did not fit
// happened to be — on the 15-node docker set, "mkdir: cannot create directory"
// from node13's launch, after twelve nodes were already running. Asked here,
// right after placement, it is one message naming every machine that is short,
// before anything is written to any of them.
//
// The floor is the workspace-config's limits.minFreeDisk, or
// resource.DefaultMinFreeDisk; "0" turns the check off. A machine whose driver
// cannot say how much space it has is not guessed at.
func (w *Workspace) CheckFreeSpace(ctx context.Context) (string, error) {
	floor := resource.DefaultMinFreeDisk
	wc, err := w.wc()
	if err != nil {
		return "", err
	}
	if wc != nil {
		if floor, err = wc.Limits.MinFreeDiskBytes(); err != nil {
			return "", fmt.Errorf("chainsetup: limits.minFreeDisk: %w", err)
		}
	}
	root := w.state.Target.DataRoot
	if floor == 0 || root == "" {
		return "free space not checked", nil
	}
	var short []string
	checked := 0
	err = w.eachMachine(func(t *resource.Access, nodes []node.Record) error {
		sr, ok := t.Driver.(process.SpaceReporter)
		if !ok {
			return nil
		}
		free, ferr := sr.FreeBytes(ctx, root)
		if ferr != nil {
			return fmt.Errorf("chainsetup: free space on %s: %w", machineName(nodes[0]), ferr)
		}
		checked++
		if free < floor {
			short = append(short, fmt.Sprintf("  %s: %s free under %s", machineName(nodes[0]), sizeOf(free), root))
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if len(short) > 0 {
		return "", fmt.Errorf("chainsetup: %d machine(s) have less than %s free:\n%s\nfree space on them (`chain rm` a composition you no longer need, or `clean --stale-compositions`), or lower limits.minFreeDisk in the workspace-config",
			len(short), sizeOf(floor), strings.Join(short, "\n"))
	}
	return fmt.Sprintf("%d machine(s) have at least %s free", checked, sizeOf(floor)), nil
}

// machineName is how a message names the machine a node runs on.
func machineName(ns node.Record) string {
	if ns.Server != "" {
		return ns.Server
	}
	if ns.Host != "" {
		return ns.Host
	}
	return "this machine"
}

// sizeOf prints a byte count in the largest binary unit it fills.
func sizeOf(b uint64) string {
	for _, u := range []struct {
		name string
		size uint64
	}{{"GiB", 1 << 30}, {"MiB", 1 << 20}, {"KiB", 1 << 10}} {
		if b >= u.size {
			return fmt.Sprintf("%.1f%s", float64(b)/float64(u.size), u.name)
		}
	}
	return fmt.Sprintf("%dB", b)
}
