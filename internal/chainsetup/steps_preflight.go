package chainsetup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/preflight"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// Have reads the workspace as the chain composed on the target, in the terms
// preflight compares: the record, not a guess.
func (w *Workspace) Have(ctx context.Context) preflight.Have {
	st := w.state
	h := preflight.Have{
		Chain: st.Chain, Binary: st.Binary, KeysDir: st.KeysDir, Peering: st.Peering,
		Validators: st.Validators, Started: st.Steps["start"].Done,
	}
	// The genesis this composition was asked for, not the bytes it produced:
	// WantOf can only digest a request, so Have has to speak the same language
	// or the comparison is between two different things and never matches. The
	// request is on disk for exactly this kind of question (F1). A workspace
	// written before it was recorded digests to nothing, which skips the check
	// rather than forcing every old workspace to rebuild.
	if st.Request != nil {
		h.GenesisDeclared = GenesisDeclared(*st.Request)
	}
	for _, r := range st.Nodes {
		h.Nodes = append(h.Nodes, preflight.Node{
			Index: r.Index, Role: node.Role(r.Role), SyncMode: r.SyncMode,
			Server: r.Server, Host: r.Host, Ports: r.Endpoints, PID: r.PID,
		})
	}
	return h
}

// WantOf reads a net-up request as the chain the caller needs, in the same
// terms. It is the shape the request declares; per-node facts are pinned by
// the caller when it has them (a spec's topology), not invented here.
func WantOf(in NetUpIn) preflight.Want {
	return preflight.Want{
		Chain: in.Chain, Binary: in.Binary, KeysDir: in.KeysDir, Peering: in.Peering,
		ChainID: in.ChainID, Validators: in.Validators, Endpoints: in.Endpoints,
		GenesisDeclared: GenesisDeclared(in),
	}
}

// GenesisDeclared digests the fields of a request that decide the genesis, so
// two requests wanting different chains cannot be mistaken for one.
//
// It is exported because both sides of the preflight comparison must compute it
// identically: the request being made ([WantOf]) and the request this workspace
// was composed from ([Workspace.Have], via the recorded request). A second
// implementation is how the comparison silently stops matching.
//
// The digest covers the chain id, the overlay file, the dot-path genesis set, an
// existing genesis used verbatim, and the template and manifest that supply the
// base document. It deliberately does NOT cover the keys or the validator count:
// those are compared on their own, and folding them in here would report "genesis
// differs" for a difference the reader can already see named.
func GenesisDeclared(in NetUpIn) string {
	h := sha256.New()
	write := func(parts ...string) {
		for _, p := range parts {
			// Length-prefixed so "ab"+"c" and "a"+"bc" do not collide.
			fmt.Fprintf(h, "%d:%s\x00", len(p), p)
		}
	}
	write(fmt.Sprintf("%d", in.ChainID), in.OverlayPath, in.GenesisExisting, in.TemplatePath, in.ManifestPath)
	// Order matters: the set is applied in order and a later key wins, so two
	// different orders can produce two different genesis documents.
	write(in.GenesisSet...)
	return hex.EncodeToString(h.Sum(nil))
}

// Compare decides how much of the composed chain a request can reuse: the
// record against the request on paper, then a live probe of every node the
// paper half would keep. A node is alive when its recorded pid runs on its
// machine and it answers an RPC head.
func (w *Workspace) Compare(ctx context.Context, want preflight.Want) preflight.Decision {
	return preflight.Check(ctx, w.Have(ctx), want, w.liveness)
}

// liveness is the probe Compare injects: pid on the node's own machine, then
// an RPC head from the node's own address.
func (w *Workspace) liveness(ctx context.Context, n preflight.Node) (bool, string) {
	if n.PID <= 0 {
		return false, "stopped (no pid recorded)"
	}
	var rec node.Record
	for _, r := range w.state.Nodes {
		if r.Index == n.Index {
			rec = r
			break
		}
	}
	t, err := w.machineFor(rec)
	if err != nil {
		return false, err.Error()
	}
	if insp, ok := t.Driver.(process.ProcessInspector); ok {
		alive, err := insp.PIDAlive(ctx, n.PID)
		if err != nil {
			return false, err.Error()
		}
		if !alive {
			return false, fmt.Sprintf("pid %d not running", n.PID)
		}
	}
	// The probe dials, so it asks for the reachable address like NodeSet and
	// Health do. Probing the node's own recorded host is what made a live docker
	// network read as "no composed node is alive", and a reusable chain get
	// rebuilt.
	url, err := w.nodeHTTPURL(rec)
	if err != nil {
		return false, err.Error()
	}
	if _, err := rpc.Dial(url).BlockNumber(ctx); err != nil {
		return false, fmt.Sprintf("no RPC head at %s: %v", url, err)
	}
	return true, ""
}
