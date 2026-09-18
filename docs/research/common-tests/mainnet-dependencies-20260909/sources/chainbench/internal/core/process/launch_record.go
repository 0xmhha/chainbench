package process

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// ProcFor builds the ledger record for spec launched at pid. It is the single
// place the launch record's shape is decided (label from the node index, the
// binary's base name, the full command line for the operator), so every launch
// path records the same fields for the same node.
func ProcFor(spec NodeSpec, pid int) Proc {
	return Proc{
		PID:     pid,
		Label:   string(node.LabelFor(spec.Index)),
		Binary:  filepath.Base(spec.Binary),
		Command: strings.Join(append([]string{spec.Binary}, spec.Args...), " "),
		Host:    spec.Host,
		DataDir: spec.DataDir,
	}
}

// LaunchAndRecord launches spec through d and records the returned pid in l, so
// every fresh launch shares one launch-then-record step: the pid the ledger
// holds is exactly the one the driver reported. A nil ledger launches without
// recording (a caller with no persistent ledger, e.g. a workspace-less run).
// It refuses a label already recorded — that is a double launch; a deliberate
// re-launch of the same label (a binary swap) goes through the ledger's
// Supersede, which preserves the prior entry as a revision.
func LaunchAndRecord(ctx context.Context, d Driver, l *Ledger, spec NodeSpec) (Handle, error) {
	h, err := d.Launch(ctx, spec)
	if err != nil {
		return Handle{}, err
	}
	if l == nil {
		return h, nil
	}
	if err := l.Record(ProcFor(spec, h.PID)); err != nil {
		return h, err
	}
	return h, nil
}
