package chainsetup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// Lifecycle use cases for a launched network. Reading and writing its state is
// the part every surface used to repeat, so it lives here once and the CLI
// commands and MCP tools call these functions.
//
// Two stacks persist that state differently: the setup stack writes
// nodeset.json (running endpoints and PIDs) plus nodespecs.json (how to bring
// one node back), while the step stack keeps a composition workspace. The
// read and teardown paths resolve either, so a caller with a data dir need not
// know which produced it. Per-node stop/start still requires the setup stack's
// saved specs — the step stack expresses that as `net restart`.

// NetworkStatusIn identifies the launched network to read.
type NetworkStatusIn struct {
	// DataDir is the data root a setup wrote nodeset.json to.
	DataDir string
}

// NetworkStatusOut is the recorded node set.
type NetworkStatusOut struct {
	Nodes node.NodeSet
	// Composed reports which stack the directory holds: true for a step-composed
	// workspace, false for a setup-launched data root.
	Composed bool
}

// NetworkStatus reads a network's node set from whichever stack composed it —
// a step workspace or a setup's data root. Both persist different state, but
// every consumer downstream speaks NodeSet, so resolving it once here is what
// lets the two stacks be used interchangeably. Read-only.
func NetworkStatus(_ context.Context, d Deps, in NetworkStatusIn) (NetworkStatusOut, error) {
	if in.DataDir == "" {
		return NetworkStatusOut{}, ErrNoDataDir
	}
	if isComposition(in.DataDir) {
		ws, err := Open(in.DataDir, d.Clock)
		if err != nil {
			return NetworkStatusOut{}, err
		}
		return NetworkStatusOut{Nodes: ws.NodeSet(), Composed: true}, nil
	}
	ns, err := session.LoadLocalNodeSet(in.DataDir)
	if err != nil {
		return NetworkStatusOut{}, err
	}
	return NetworkStatusOut{Nodes: ns}, nil
}

// isComposition reports whether dir holds a step-composed workspace. Its state
// manifest is the marker, and session owns where that lives.
func isComposition(dir string) bool {
	_, err := os.Stat(session.CompositionFilePath(dir))
	return err == nil
}

// NetworkStopIn identifies the launched network to stop.
type NetworkStopIn struct {
	DataDir string
}

// NetworkStopOut reports what was stopped.
type NetworkStopOut struct {
	// Stopped is how many nodes were terminated.
	Stopped int
	// Failed collects the per-node failures. Stopping is best-effort — one dead
	// PID must not keep the rest of the network up — so these are reported
	// rather than returned as the call's error.
	Failed []error
}

// NetworkStop terminates every node of a launched network by its recorded PID.
// A composed workspace stops through its own step, which also clears the PIDs
// it recorded; a setup data root stops through the driver.
func NetworkStop(ctx context.Context, d Deps, in NetworkStopIn) (NetworkStopOut, error) {
	if in.DataDir == "" {
		return NetworkStopOut{}, ErrNoDataDir
	}
	if isComposition(in.DataDir) {
		// Counted before the step runs: it stops every node that still has a
		// PID, and clears them, so afterwards there is nothing left to count.
		var running int
		_, err := withWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
			running = withPID(ws.NodeSet())
			return ws.Stop(ctx)
		})
		if err != nil {
			return NetworkStopOut{}, err
		}
		return NetworkStopOut{Stopped: running}, nil
	}
	ns, err := session.LoadLocalNodeSet(in.DataDir)
	if err != nil {
		return NetworkStopOut{}, err
	}
	return stopAll(ctx, d, ns)
}

// withPID counts the nodes chainbench has a live process id for.
func withPID(ns node.NodeSet) int {
	n := 0
	for _, nd := range ns.Nodes {
		if nd.PID > 0 {
			n++
		}
	}
	return n
}

// stopAll resolves the driver and stops every launched node in ns.
func stopAll(ctx context.Context, d Deps, ns node.NodeSet) (NetworkStopOut, error) {
	dr, err := d.nodeDriver()
	if err != nil {
		return NetworkStopOut{}, err
	}
	stopped, errs := driver.StopNodeSet(ctx, dr, ns)
	return NetworkStopOut{Stopped: stopped, Failed: errs}, nil
}

// NodeStopIn selects one node of a launched network.
type NodeStopIn struct {
	DataDir string
	// Index is the 1-based node index.
	Index int
}

// NodeStop stops a single node and records it as stopped, so a sync gap can be
// created while the rest of the network keeps producing blocks. Clearing the
// PID is what makes a later status or start accurate.
func NodeStop(ctx context.Context, d Deps, in NodeStopIn) error {
	if in.DataDir == "" || in.Index <= 0 {
		return ErrNoDataDirAndIndex
	}
	ns, err := session.LoadLocalNodeSet(in.DataDir)
	if err != nil {
		return err
	}
	dr, err := d.nodeDriver()
	if err != nil {
		return err
	}
	if err := driver.StopNode(ctx, dr, ns, in.Index); err != nil {
		return err
	}
	for i := range ns.Nodes {
		if ns.Nodes[i].Index == in.Index {
			ns.Nodes[i].PID = 0
		}
	}
	return session.SaveLocalNodeSet(in.DataDir, ns)
}

// NodeStartIn selects one stopped node of a launched network.
type NodeStartIn struct {
	DataDir string
	Index   int
}

// NodeStartOut is the relaunched node, with its new PID.
type NodeStartOut struct {
	Node node.Node
}

// NodeStart relaunches a single stopped node from the spec the launch saved, so
// it rejoins its peers and re-syncs the blocks it missed, and records its new
// PID.
func NodeStart(ctx context.Context, d Deps, in NodeStartIn) (NodeStartOut, error) {
	if in.DataDir == "" || in.Index <= 0 {
		return NodeStartOut{}, ErrNoDataDirAndIndex
	}
	ns, err := session.LoadLocalNodeSet(in.DataDir)
	if err != nil {
		return NodeStartOut{}, err
	}
	specs, err := session.LoadLocalNodeSpecs(in.DataDir)
	if err != nil {
		return NodeStartOut{}, err
	}
	spec, ok := specFor(specs, in.Index)
	if !ok {
		return NodeStartOut{}, fmt.Errorf("chainsetup: no saved spec for node%d in %s", in.Index, in.DataDir)
	}
	dr, err := d.nodeDriver()
	if err != nil {
		return NodeStartOut{}, err
	}
	refreshed, err := driver.RelaunchNode(ctx, dr, spec)
	if err != nil {
		return NodeStartOut{}, err
	}
	ns.Nodes = replaceNode(ns.Nodes, refreshed)
	if err := session.SaveLocalNodeSet(in.DataDir, ns); err != nil {
		return NodeStartOut{}, err
	}
	return NodeStartOut{Node: refreshed}, nil
}

// NetworkRemoveIn identifies the data root to tear down.
type NetworkRemoveIn struct {
	DataDir string
}

// NetworkRemoveOut reports the teardown.
type NetworkRemoveOut struct {
	// Stopped and Failed describe the stop that precedes removal. A data root
	// with no readable node set removes nothing and reports zero — there is no
	// record of anything to stop.
	Stopped int
	Failed  []error
	// Removed is the directory that was deleted.
	Removed string
}

// NetworkRemove stops the network and deletes its data root. It refuses a
// directory that carries no setup artifact, so a mistyped path cannot delete
// something unrelated.
func NetworkRemove(ctx context.Context, d Deps, in NetworkRemoveIn) (NetworkRemoveOut, error) {
	if in.DataDir == "" {
		return NetworkRemoveOut{}, ErrNoDataDir
	}
	if !isDataRoot(in.DataDir) {
		return NetworkRemoveOut{}, fmt.Errorf(
			"chainsetup: %q does not look like a chainbench data dir (no nodeset.json/genesis.json); refusing to remove", in.DataDir)
	}
	out := NetworkRemoveOut{Removed: in.DataDir}
	// Best-effort: a data root that was provisioned but never launched has no
	// node set, and removing it is still the right thing to do.
	if ns, err := session.LoadLocalNodeSet(in.DataDir); err == nil {
		stop, serr := stopAll(ctx, d, ns)
		if serr != nil {
			return NetworkRemoveOut{}, serr
		}
		out.Stopped, out.Failed = stop.Stopped, stop.Failed
	}
	if err := os.RemoveAll(in.DataDir); err != nil {
		return NetworkRemoveOut{}, fmt.Errorf("chainsetup: remove %s: %w", in.DataDir, err)
	}
	return out, nil
}

// dataRootMarkers are the files that identify a directory as a chainbench data
// root. Either one is enough: a provisioned-but-never-launched root has only
// the genesis.
var dataRootMarkers = []string{"nodeset.json", "genesis.json"}

// isDataRoot reports whether dir holds setup artifacts.
func isDataRoot(dir string) bool {
	for _, f := range dataRootMarkers {
		if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
			return true
		}
	}
	return false
}

// specFor returns the saved launch spec for a node index.
func specFor(specs []driver.NodeSpec, index int) (driver.NodeSpec, bool) {
	for _, s := range specs {
		if s.Index == index {
			return s, true
		}
	}
	return driver.NodeSpec{}, false
}

// replaceNode substitutes n for the node with its index, appending it when the
// set does not carry that index yet.
func replaceNode(nodes []node.Node, n node.Node) []node.Node {
	for i := range nodes {
		if nodes[i].Index == n.Index {
			nodes[i] = n
			return nodes
		}
	}
	return append(nodes, n)
}

var (
	// ErrNoDataDir refuses a verb that needs a workspace but was given none.
	ErrNoDataDir         = errors.New("chainsetup: a data dir with the setup's nodeset.json is required")
	ErrNoDataDirAndIndex = errors.New("chainsetup: a data dir and a 1-based node index are required")
)
