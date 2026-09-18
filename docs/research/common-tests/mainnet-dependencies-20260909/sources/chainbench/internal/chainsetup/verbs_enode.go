package chainsetup

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// Enode listing is stage ③ of the composition dictionary: a queryable output
// derived from the two stages before it — keys (the node's devp2p public key)
// and place (its host and p2p port). It writes nothing; the same derivation
// the config step uses to build each node's static-nodes list is exposed here
// so an operator (or a later step) can read the enodes without rendering a
// config. The enode host is the node's own recorded address — the one a peer
// dials on the network — not the harness's docker-translated dial address.

// NetEnodesIn selects the workspace and, optionally, one node.
type NetEnodesIn struct {
	// DataDir is the workspace directory.
	DataDir string
	// Node, when > 0, limits the result to that 1-based node index.
	Node int
}

// NodeEnode is one node's identity on the wire: its index, its label, and the
// enode URL a peer uses to reach it.
type NodeEnode struct {
	Index int    `json:"index"`
	Label string `json:"label"`
	Enode string `json:"enode"`
}

// NetEnodesOut is the enode list in node order.
type NetEnodesOut struct {
	Enodes []NodeEnode `json:"enodes"`
}

// NetEnodes derives each node's enode from the key set and the placement. It
// needs both stages done: the node table (place) supplies host and p2p port,
// the key set (keys) supplies the devp2p public key. It reports which one is
// missing rather than an empty list.
func NetEnodes(_ context.Context, d Deps, in NetEnodesIn) (NetEnodesOut, error) {
	if in.DataDir == "" {
		return NetEnodesOut{}, ErrNoDataDir
	}
	ws, err := Open(in.DataDir, d.Clock)
	if err != nil {
		return NetEnodesOut{}, err
	}
	st := ws.State()
	if len(st.Nodes) == 0 {
		return NetEnodesOut{}, fmt.Errorf("chainsetup: enode: no node table — run `chain place` first")
	}
	if st.KeysDir == "" {
		return NetEnodesOut{}, fmt.Errorf("chainsetup: enode: no key set — run `chain keys` first")
	}
	preset, err := store.LoadPreset(st.KeysDir)
	if err != nil {
		return NetEnodesOut{}, fmt.Errorf("chainsetup: enode: keys: %w", err)
	}
	placed, err := ws.Netmap()
	if err != nil {
		return NetEnodesOut{}, err
	}
	var out NetEnodesOut
	for _, pl := range placed.Placements() {
		if in.Node > 0 && pl.Index != in.Node {
			continue
		}
		entry, ok := preset.Node(pl.Index)
		if !ok {
			return NetEnodesOut{}, fmt.Errorf("chainsetup: enode: no key for node%d — the key set does not cover the node count", pl.Index)
		}
		out.Enodes = append(out.Enodes, NodeEnode{
			Index: pl.Index,
			Label: string(pl.Label),
			Enode: node.Enode(entry.PublicKey, pl.Host, pl.Ports.P2P),
		})
	}
	if in.Node > 0 && len(out.Enodes) == 0 {
		return NetEnodesOut{}, fmt.Errorf("chainsetup: enode: no node%d in this workspace (%d nodes)", in.Node, len(st.Nodes))
	}
	return out, nil
}
