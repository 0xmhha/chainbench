package chainsetup

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/0xmhha/chainbench/internal/core/node"
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

// NetworkStatusOut is the recorded node set.
type NetworkStatusOut struct {
	Nodes node.NodeSet
}

// NetworkStatus reads a network's node set from its workspace. Read-only.
func NetworkStatus(_ context.Context, d Deps, in NetworkStatusIn) (NetworkStatusOut, error) {
	if in.DataDir == "" {
		return NetworkStatusOut{}, ErrNoDataDir
	}
	if !isComposition(in.DataDir) {
		return NetworkStatusOut{}, fmt.Errorf("chainsetup: %s holds no workspace (no %s)", in.DataDir, session.CompositionFilePath(in.DataDir))
	}
	ws, err := Open(in.DataDir, d.Clock)
	if err != nil {
		return NetworkStatusOut{}, err
	}
	return NetworkStatusOut{Nodes: ws.NodeSet()}, nil
}

// isComposition reports whether dir holds a workspace. Its state manifest is
// the marker, and session owns where that lives.
func isComposition(dir string) bool {
	_, err := os.Stat(session.CompositionFilePath(dir))
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
	_, err := withWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
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
	_, err := withWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
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

// NodeSwapIn selects one node and the binary to relaunch it on.
type NodeSwapIn struct {
	DataDir string
	Index   int
	// Binary is the path to relaunch node Index on. The datadir and genesis are
	// unchanged: this is a per-node binary swap, not a rebuild.
	Binary string
}

// NodeSwap stops one node and relaunches it on a different binary, so a network
// can run mixed binaries mid-test. The pre-swap pid and command are kept as a
// ledger revision; the relaunched node's new PID is returned.
func NodeSwap(ctx context.Context, d Deps, in NodeSwapIn) (NodeStartOut, error) {
	if in.DataDir == "" || in.Index <= 0 {
		return NodeStartOut{}, ErrNoDataDirAndIndex
	}
	var swapped node.Node
	_, err := withWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
		detail, err := ws.SwapNode(ctx, in.Index, in.Binary)
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
	_, err := withWorkspace(d, in.DataDir, func(ws *Workspace) (string, error) {
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
		return NetworkRemoveOut{}, fmt.Errorf(
			"chainsetup: %q does not look like a chainbench workspace (no %s); refusing to remove", in.DataDir, session.CompositionFilePath(in.DataDir))
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
