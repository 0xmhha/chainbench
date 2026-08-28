package chainsetup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// teardownGrace is how long Down waits for a graceful stop before escalating.
const teardownGrace = 5 * time.Second

// stateFile records a brought-up network under its data root, so status and
// teardown work from a later process.
const stateFile = "chain-network.json"

// writeNodeSet persists a node set through the file interface.
func writeNodeSet(ctx context.Context, files filestore.Store, path string, ns node.NodeSet) error {
	b, err := json.MarshalIndent(ns, "", "  ")
	if err != nil {
		return err
	}
	return files.Write(ctx, path, b, 0o644)
}

// LoadNetwork reads the node set a bring-up left under dataDir.
func LoadNetwork(dataDir string) (node.NodeSet, error) {
	b, err := os.ReadFile(filepath.Join(dataDir, stateFile))
	if err != nil {
		return node.NodeSet{}, fmt.Errorf("chainsetup: no network under %s (was it brought up there?): %w", dataDir, err)
	}
	var ns node.NodeSet
	if err := json.Unmarshal(b, &ns); err != nil {
		return node.NodeSet{}, fmt.Errorf("chainsetup: %s: %w", stateFile, err)
	}
	return ns, nil
}

// Probe is one node's observed state at a moment — an answer to `net status`,
// not the node itself (the node's fact record is node.Record).
type Probe struct {
	Index   int
	RPCURL  string
	PID     int
	Alive   bool
	Head    uint64
	Peers   uint64
	ChainID uint64
	Err     string
}

// Status probes every node of a recorded network. A node that does not answer is
// reported with its error rather than dropped, because "which node is down" is
// usually the question being asked.
func Status(ctx context.Context, dataDir string) ([]Probe, error) {
	ns, err := LoadNetwork(dataDir)
	if err != nil {
		return nil, err
	}
	out := make([]Probe, 0, len(ns.Nodes))
	for _, n := range ns.Nodes {
		st := Probe{Index: n.Index, RPCURL: n.RPCURL, PID: n.PID, Alive: process.Alive(n.PID)}
		c := rpc.Dial(n.RPCURL)
		if h, err := c.BlockNumber(ctx); err == nil {
			st.Head = h
		} else {
			st.Err = err.Error()
		}
		if p, err := c.PeerCount(ctx); err == nil {
			st.Peers = p
		}
		if id, err := c.ChainID(ctx); err == nil {
			st.ChainID = id
		}
		out = append(out, st)
	}
	return out, nil
}

// Down stops every node of a recorded network and verifies it is gone, returning
// any leaked pids. RemoveData additionally deletes the data root, which is a
// separate operation from stopping (design S2).
func Down(dataDir string, removeData bool) ([]int, error) {
	ns, err := LoadNetwork(dataDir)
	if err != nil {
		return nil, err
	}
	m := process.New()
	for _, n := range ns.Nodes {
		m.TrackProc(process.Proc{PID: n.PID, Label: string(node.LabelFor(n.Index)), DataDir: dataDir, Host: n.Host})
	}
	leaks := m.StopAll(teardownGrace)
	if removeData {
		if err := os.RemoveAll(dataDir); err != nil {
			return leaks, err
		}
	}
	return leaks, nil
}
