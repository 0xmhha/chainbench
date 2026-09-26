package verb

import (
	"context"

	"github.com/0xmhha/chainbench/internal/chainsetup"
)

// Composing against a target that may already hold what the request wants.
//
// This is the comparison a suite makes before it builds anything: what is on
// the target against what this run declares. It was a switch over four verdicts
// written outside this package, and the four things it did — leave the network
// alone, restart the nodes that differ, stop and recompose, compose — are four
// states with four moves out of them.
//
// What the switch could not say is where a run WAS. A run that recomposed and
// then failed in the genesis step reported a genesis failure with no trace of
// the comparison that sent it there; now the walk passes through
// CompareChainNetworkDiffers and the state says so.

// CompareIn asks for a composition that reuses what the target already has when
// the comparison says it can.
type CompareIn struct {
	// Up is the composition to reach, and what the comparison compares against.
	Up chainsetup.ChainUpIn
}

// CompareOut is what the comparison and the composition did.
type CompareOut struct {
	ChainUpOut
	// Decision is the comparison as a report prints it, verdict and reasons.
	Decision string
	// At is where the walk ended, as a path.
	At string
	// Failed is the failure state the failing stage named, when the walk ended
	// in CHAIN_FAILED; empty otherwise.
	Failed string
}

// ChainUpComparing composes the network the request declares, reusing what is
// on the target when the comparison says it can.
//
// Four ways through, and they are four states rather than four arms of a
// switch: the network is the one wanted (nothing to do), some nodes differ
// (bring those back), a network-wide fact differs (stop, then compose), or
// nothing is composed (compose).
//
// It stops before the readiness gate. Whether the network that resulted is
// producing is a question this package cannot answer — the gate belongs to
// whoever owns the monitor — so the walk hands it over rather than guessing.
//
// It is held to what `chain up` is held to — the same request checks, the
// workspace-config's execution.chain, and the workspace locked for the whole
// walk. It used to skip all three: a run's request was never checked for inline
// key material before it was recorded, execution.chain was read by `chain up`
// and ignored by a run, and another run could compose over this one between
// two of its stages.
func ChainUpComparing(ctx context.Context, d chainsetup.Deps, in CompareIn) (CompareOut, error) {
	var out CompareOut
	up, err := planUp(in.Up)
	if err != nil {
		return out, err
	}
	ws, release, err := holdWorkspace(d, in.Up.DataDir)
	if err != nil {
		return out, err
	}
	defer release()
	mgr := newManagerFor(ctx, d, ws, up.stage, up.mode)
	cerr := mgr.ComposeComparing(ctx, in.Up)
	out.Steps = mgr.Steps()
	out.Decision = mgr.Decision()
	out.At = mgr.At()
	if st := mgr.FailStatus(); st != 0 {
		out.Failed = st.String()
	}
	return out, cerr
}
