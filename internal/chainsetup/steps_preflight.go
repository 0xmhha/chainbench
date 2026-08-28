package chainsetup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/preflight"
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
	if st.GenesisPath != "" {
		if t, err := w.resolveTarget(); err == nil {
			if b, err := t.Files.Read(ctx, st.GenesisPath); err == nil {
				sum := sha256.Sum256(b)
				h.GenesisSum = hex.EncodeToString(sum[:])
			}
		}
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
	}
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
	if insp, ok := t.Driver.(driver.ProcessInspector); ok {
		alive, err := insp.PIDAlive(ctx, n.PID)
		if err != nil {
			return false, err.Error()
		}
		if !alive {
			return false, fmt.Sprintf("pid %d not running", n.PID)
		}
	}
	url := fmt.Sprintf("http://%s:%d", nodeHost(rec), n.Ports.HTTP)
	if _, err := rpc.Dial(url).BlockNumber(ctx); err != nil {
		return false, fmt.Sprintf("no RPC head at %s: %v", url, err)
	}
	return true, ""
}
