package chainsetup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// Lifecycle use cases for a composed network, addressed by its workspace.
// Reading and writing its state is the part every surface used to repeat, so
// it lives here once and the CLI commands and MCP tools call these functions.
// There is one record of a network — the workspace — and every verb here
// reads and writes that record.

// NetworkStatusIn identifies the network to read.
type NetworkStatusIn struct {
	// DataDir is the workspace directory.
	DataDir string
}

// NetworkStatusOut is the recorded node set, with what the machines say about
// the pids in it.
type NetworkStatusOut struct {
	Nodes node.NodeSet
	// Alive maps a node index to whether its recorded pid is a live process on
	// the machine that node runs on. A node with no recorded pid is absent from
	// the map, and so is one whose machine could not be asked — "not asked" and
	// "asked and gone" are different answers and the second one is the news.
	Alive map[int]bool
}

// NetworkStatus reads a network's node set from its workspace and asks each
// node's machine whether the recorded pid is still a process.
//
// The record says a node was started; it cannot say the node is running. A pid
// outlives nothing — the process it named can be gone, or replaced by an
// unrelated one — so a status built from the record alone reports a network
// that may have died an hour ago. The check is what the ProcessInspector
// capability is for, and every other lifecycle path already asks it; status was
// the one that did not, and its signature said so: it took the context and
// discarded it.
func NetworkStatus(ctx context.Context, d Deps, in NetworkStatusIn) (NetworkStatusOut, error) {
	if in.DataDir == "" {
		return NetworkStatusOut{}, ErrNoDataDir
	}
	if !isComposition(in.DataDir) {
		return NetworkStatusOut{}, fmt.Errorf("chainsetup: %w", session.NoRecordError(in.DataDir))
	}
	ws, err := Open(in.DataDir, d.Clock)
	if err != nil {
		return NetworkStatusOut{}, err
	}
	ws.SetEnv(d.Env)
	ws.SetDriver(d.Driver)
	return NetworkStatusOut{Nodes: ws.NodeSet(), Alive: ws.livePIDs(ctx)}, nil
}

// isComposition reports whether dir holds a composed chain. Its chain record is
// the marker, and session owns where that lives.
func isComposition(dir string) bool {
	_, err := os.Stat(session.ChainRecordPath(dir))
	return err == nil
}

// NetworkStopIn identifies the network to stop.
type NetworkStopIn struct {
	DataDir string
}

// NetworkStopOut reports what was stopped.
type NetworkStopOut struct {
	// Stopped is how many nodes were terminated.
	Stopped int
}

// NetworkStop terminates every running node by its recorded PID and clears
// the PIDs, through the workspace's stop step.
func NetworkStop(ctx context.Context, d Deps, in NetworkStopIn) (NetworkStopOut, error) {
	if in.DataDir == "" {
		return NetworkStopOut{}, ErrNoDataDir
	}
	// Counted before the step runs: it stops every node that still has a
	// PID, and clears them, so afterwards there is nothing left to count.
	var running int
	_, err := WithWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		running = withPID(ws.NodeSet())
		return ws.Stop(ctx)
	})
	if err != nil {
		return NetworkStopOut{}, err
	}
	return NetworkStopOut{Stopped: running}, nil
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

// NodeStopIn selects one node of a network.
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
	_, err := WithWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		return ws.StopNode(ctx, in.Index)
	})
	return err
}

// NodeStartIn selects one stopped node of a network.
type NodeStartIn struct {
	DataDir string
	Index   int
}

// NodeStartOut is the relaunched node, with its new PID.
type NodeStartOut struct {
	Node node.Node
}

// NetRunner returns the target's remote command runner, or nil for a local
// target (which reads its own filesystem). It is what lets the run read a remote
// node's log over SSH and reconnect a dropped session (E8).
func NetRunner(d Deps, dataDir string) (process.Runner, error) {
	ws, err := Open(dataDir, d.Clock)
	if err != nil {
		return nil, err
	}
	if !ws.state.Target.IsRemote() {
		return nil, nil
	}
	t, err := ws.resolveTarget()
	if err != nil {
		return nil, err
	}
	return t.Runner, nil
}

// NodeSwapIn selects one node and the binary and/or config to relaunch it with.
type NodeSwapIn struct {
	DataDir string
	Index   int
	// Binary is the path to relaunch node Index on (empty keeps the current
	// one). The datadir and genesis are unchanged: this is a swap, not a rebuild.
	Binary string
	// Config is a set of key=value config overrides to apply before relaunch
	// (empty keeps the current config).
	Config []string
	// GenesisOverlay is a genesis JSON fragment deep-merged into the network
	// genesis and re-applied to this node's datadir, so one node can run a
	// genesis the rest of the network does not have.
	GenesisOverlay []byte
	// Purpose names the config fixture recorded in provenance (config-<purpose>).
	Purpose string
}

// NodeSwap stops one node and relaunches it with a different binary and/or
// config, so a network can run mixed binaries mid-test. The pre-swap pid and
// command are kept as a ledger revision; the relaunched node's new PID is
// returned.
func NodeSwap(ctx context.Context, d Deps, in NodeSwapIn) (NodeStartOut, error) {
	if in.DataDir == "" || in.Index <= 0 {
		return NodeStartOut{}, ErrNoDataDirAndIndex
	}
	var swapped node.Node
	_, err := WithWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		detail, err := ws.SwapNode(ctx, SwapNodeOpts{
			Index: in.Index, Binary: in.Binary, Config: in.Config,
			GenesisOverlay: in.GenesisOverlay, Purpose: in.Purpose,
		})
		if err != nil {
			return "", err
		}
		for _, n := range ws.NodeSet().Nodes {
			if n.Index == in.Index {
				swapped = n
			}
		}
		return detail, nil
	})
	if err != nil {
		return NodeStartOut{}, err
	}
	return NodeStartOut{Node: swapped}, nil
}

// NodeStart relaunches a single stopped node with the argv it was armed with,
// so it rejoins its peers and re-syncs the blocks it missed, and records its
// new PID.
func NodeStart(ctx context.Context, d Deps, in NodeStartIn) (NodeStartOut, error) {
	if in.DataDir == "" || in.Index <= 0 {
		return NodeStartOut{}, ErrNoDataDirAndIndex
	}
	var started node.Node
	_, err := WithWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		detail, err := ws.StartNode(ctx, in.Index)
		if err != nil {
			return "", err
		}
		for _, n := range ws.NodeSet().Nodes {
			if n.Index == in.Index {
				started = n
			}
		}
		return detail, nil
	})
	if err != nil {
		return NodeStartOut{}, err
	}
	return NodeStartOut{Node: started}, nil
}

// NetworkRemoveIn identifies the workspace to tear down.
type NetworkRemoveIn struct {
	DataDir string
}

// NetworkRemoveOut reports the teardown.
type NetworkRemoveOut struct {
	// Stopped is how many nodes the stop that precedes removal terminated.
	Stopped int
	// Removed is the directory that was deleted.
	Removed string
}

// NetworkRemove stops the network and deletes its workspace directory. It
// refuses a directory that carries no workspace, so a mistyped path cannot
// delete something unrelated.
func NetworkRemove(ctx context.Context, d Deps, in NetworkRemoveIn) (NetworkRemoveOut, error) {
	if in.DataDir == "" {
		return NetworkRemoveOut{}, ErrNoDataDir
	}
	if !isComposition(in.DataDir) {
		return NetworkRemoveOut{}, fmt.Errorf("chainsetup: refusing to remove %q: %w", in.DataDir, session.NoRecordError(in.DataDir))
	}
	stop, err := NetworkStop(ctx, d, NetworkStopIn(in))
	if err != nil {
		return NetworkRemoveOut{}, err
	}
	if err := os.RemoveAll(in.DataDir); err != nil {
		return NetworkRemoveOut{}, fmt.Errorf("chainsetup: remove %s: %w", in.DataDir, err)
	}
	return NetworkRemoveOut{Stopped: stop.Stopped, Removed: in.DataDir}, nil
}

var (
	// ErrNoDataDir refuses a verb that needs a workspace but was given none.
	ErrNoDataDir = errors.New("chainsetup: a workspace directory is required")
	// ErrNoDataDirAndIndex refuses a per-node verb missing its workspace or index.
	ErrNoDataDirAndIndex = errors.New("chainsetup: a workspace directory and a 1-based node index are required")
)

// NetCrossForkIn names the composition whose declared hardfork is to be crossed.
type NetCrossForkIn struct {
	DataDir string
	// Timeout bounds the wait for the network to reach the block before the
	// fork; zero takes the step's own default.
	Timeout time.Duration
}

// NetCrossForkOut is the network as it stands once the successors produce.
type NetCrossForkOut struct {
	// Detail is what the step recorded.
	Detail string `json:"detail"`
	// Nodes is the whole node table, because crossing changes more than one
	// node: every successor has a new role and a new pid.
	Nodes node.NodeSet `json:"nodes"`
}

// NetCrossFork waits for the network to reach the block before its declared
// hardfork and hands production to the build that seals after it.
//
// The whole table comes back rather than the nodes that changed. A caller holds
// a table and has to write the result into it, and returning only the changed
// ones makes every caller re-derive which those were — from the same fact the
// step already knows.
func NetCrossFork(ctx context.Context, d Deps, in NetCrossForkIn) (NetCrossForkOut, error) {
	if in.DataDir == "" {
		return NetCrossForkOut{}, ErrNoDataDir
	}
	var out NetCrossForkOut
	_, err := WithWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		detail, err := ws.CrossFork(ctx, CrossForkOpts{Timeout: in.Timeout})
		if err != nil {
			return "", err
		}
		out.Detail, out.Nodes = detail, ws.NodeSet()
		return detail, nil
	})
	if err != nil {
		return NetCrossForkOut{}, err
	}
	return out, nil
}

// NetForkIn identifies the composition to read the declared hardfork from.
type NetForkIn struct {
	DataDir string
}

// NetForkOut is the hardfork a composition is built to cross and which of its
// nodes stand on each side of it.
type NetForkOut struct {
	// Fork is the declared hardfork, nil when the network crosses none.
	Fork *GenesisFork `json:"fork,omitempty"`
	// PreFork are the indices of the nodes running the build that seals up to
	// the fork block and stops there. After the handover they stay where they
	// stopped: they cannot validate what the successors produce.
	PreFork []int `json:"preFork,omitempty"`
	// HaltsAt is the block this network's genesis makes it stop one short of,
	// and 0 when it keeps producing. A network can halt without crossing a
	// fork — a genesis naming a system-contract version the build does not have
	// is one — so it is answered even when Fork is nil.
	HaltsAt int64 `json:"haltsAt,omitempty"`
}

// NetFork reads the hardfork a composition is built to cross.
//
// It is read from the record rather than carried from the request because the
// two callers are not the same run: a network is composed once and attached to
// afterwards, and the second one has no request to read.
func NetFork(_ context.Context, d Deps, in NetForkIn) (NetForkOut, error) {
	if in.DataDir == "" {
		return NetForkOut{}, ErrNoDataDir
	}
	var out NetForkOut
	_, err := WithWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		out.HaltsAt = ws.state.HaltsAt
		out.Fork = ws.state.Fork
		if out.Fork == nil {
			return "", nil
		}
		for _, ns := range ws.state.Nodes {
			if ns.Binary != out.Fork.Binary {
				out.PreFork = append(out.PreFork, ns.Index)
			}
		}
		return "", nil
	})
	if err != nil {
		return NetForkOut{}, err
	}
	return out, nil
}
