package app

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/state"
)

// Lifecycle use cases for a network launched by the setup stack. Their state is
// the data root: nodeset.json records the running endpoints and PIDs,
// nodespecs.json records how to bring one node back. Reading and writing that
// pair is the part every surface used to repeat, so it lives here once and the
// CLI commands and MCP tools call these functions (worklist T7.11).

// NetworkStatusIn identifies the launched network to read.
type NetworkStatusIn struct {
	// DataDir is the data root a setup wrote nodeset.json to.
	DataDir string
}

// NetworkStatusOut is the recorded node set.
type NetworkStatusOut struct {
	Nodes node.NodeSet
}

// NetworkStatus reads the launched network's node set. Read-only.
func NetworkStatus(_ context.Context, _ Deps, in NetworkStatusIn) (NetworkStatusOut, error) {
	if in.DataDir == "" {
		return NetworkStatusOut{}, errNoDataDir
	}
	ns, err := state.LoadNodeSet(in.DataDir)
	if err != nil {
		return NetworkStatusOut{}, err
	}
	return NetworkStatusOut{Nodes: ns}, nil
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

// NetworkStop terminates every node the setup launched, by the PIDs its node
// set records.
func NetworkStop(ctx context.Context, d Deps, in NetworkStopIn) (NetworkStopOut, error) {
	if in.DataDir == "" {
		return NetworkStopOut{}, errNoDataDir
	}
	ns, err := state.LoadNodeSet(in.DataDir)
	if err != nil {
		return NetworkStopOut{}, err
	}
	return stopAll(ctx, d, ns)
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
		return errNoDataDirAndIndex
	}
	ns, err := state.LoadNodeSet(in.DataDir)
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
	return state.SaveNodeSet(in.DataDir, ns)
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
		return NodeStartOut{}, errNoDataDirAndIndex
	}
	ns, err := state.LoadNodeSet(in.DataDir)
	if err != nil {
		return NodeStartOut{}, err
	}
	specs, err := state.LoadNodeSpecs(in.DataDir)
	if err != nil {
		return NodeStartOut{}, err
	}
	spec, ok := specFor(specs, in.Index)
	if !ok {
		return NodeStartOut{}, fmt.Errorf("app: no saved spec for node%d in %s", in.Index, in.DataDir)
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
	if err := state.SaveNodeSet(in.DataDir, ns); err != nil {
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
		return NetworkRemoveOut{}, errNoDataDir
	}
	if !isDataRoot(in.DataDir) {
		return NetworkRemoveOut{}, fmt.Errorf(
			"app: %q does not look like a chainbench data dir (no nodeset.json/genesis.json); refusing to remove", in.DataDir)
	}
	out := NetworkRemoveOut{Removed: in.DataDir}
	// Best-effort: a data root that was provisioned but never launched has no
	// node set, and removing it is still the right thing to do.
	if ns, err := state.LoadNodeSet(in.DataDir); err == nil {
		stop, serr := stopAll(ctx, d, ns)
		if serr != nil {
			return NetworkRemoveOut{}, serr
		}
		out.Stopped, out.Failed = stop.Stopped, stop.Failed
	}
	if err := os.RemoveAll(in.DataDir); err != nil {
		return NetworkRemoveOut{}, fmt.Errorf("app: remove %s: %w", in.DataDir, err)
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
	errNoDataDir         = errors.New("app: a data dir with the setup's nodeset.json is required")
	errNoDataDirAndIndex = errors.New("app: a data dir and a 1-based node index are required")
)
