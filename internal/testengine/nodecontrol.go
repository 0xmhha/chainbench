package testengine

import (
	"context"
	"errors"
	"fmt"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"io"
	"io/fs"
	"os"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

// The control a spec's steps get over the composed nodes.
//
// A step that stops a node, swaps its binary or reads its log goes through this
// adapter rather than touching the workspace, so the engine holds one narrow
// surface and the steps cannot reach past it into the composition.

// workspaceNodes adapts the workspace's node verbs to the interpreter's
// NodeControl, so fault steps (stopNode/startNode/restartNode) act on a
// suite-composed network through the same record every other verb uses.
type workspaceNodes struct {
	sd      chainsetup.Deps
	dataDir string
}

// Stop stops one node through the workspace and returns it with its pid
// cleared, which the interpreter writes back to the environment's node table.
func (w workspaceNodes) Stop(ctx context.Context, n node.Node) (node.Node, error) {
	if err := verb.NodeStop(ctx, w.sd, verb.NodeStopIn{DataDir: w.dataDir, Index: n.Index}); err != nil {
		return n, err
	}
	n.PID = 0
	return n, nil
}

// Start relaunches one previously stopped node through the workspace and
// returns it with its new pid.
func (w workspaceNodes) Start(ctx context.Context, n node.Node) (node.Node, error) {
	out, err := verb.NodeStart(ctx, w.sd, verb.NodeStartIn{DataDir: w.dataDir, Index: n.Index})
	if err != nil {
		return n, err
	}
	return out.Node, nil
}

// Swap relaunches one node with a different binary and/or config through the
// workspace, satisfying interp.NodeSwapper so the swapNode action reaches it.
func (w workspaceNodes) Swap(ctx context.Context, n node.Node, change interp.NodeChange) (node.Node, error) {
	out, err := verb.NodeSwap(ctx, w.sd, verb.NodeSwapIn{
		DataDir: w.dataDir, Index: n.Index,
		Binary: change.Binary, Config: change.Config,
		GenesisOverlay: change.GenesisOverlay, Purpose: change.Purpose,
	})
	if err != nil {
		return n, err
	}
	return out.Node, nil
}

// CrossFork waits for the network to reach the block before its declared
// hardfork and hands production to the build that seals after it, satisfying
// interp.ForkCrosser so the crossFork action reaches it.
//
// The whole table comes back: crossing gives every successor a new role and a
// new pid, and a caller holding the old ones would stop the wrong process.
func (w workspaceNodes) CrossFork(ctx context.Context, timeout time.Duration) ([]node.Node, error) {
	out, err := verb.ChainCrossFork(ctx, w.sd, verb.ChainCrossForkIn{
		DataDir: w.dataDir, Timeout: timeout,
	})
	if err != nil {
		return nil, err
	}
	return out.Nodes.Nodes, nil
}

// Log returns the tail of one node's captured stdout/stderr, satisfying
// interp.NodeLogReader. It is what lets a spec say WHY a node is not up: a node
// that refuses its genesis prints the reason and exits, and the process manager
// sees only an exit.
//
// The log lives in the workspace this suite composed, under the conventional
// per-node label. A node that has never been launched has no log file, which is
// not an error — it reads as an empty log.
func (w workspaceNodes) Log(_ context.Context, n node.Node, maxBytes int) (string, error) {
	path := node.Layout{Root: w.dataDir}.LogPath(node.LabelFor(n.Index))
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return "", nil
		}
		return "", fmt.Errorf("engine: open node%d log %s: %w", n.Index, path, err)
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("engine: stat node%d log: %w", n.Index, err)
	}
	size := info.Size()
	if maxBytes <= 0 || int64(maxBytes) > size {
		maxBytes = int(size)
	}
	if maxBytes == 0 {
		return "", nil
	}
	if _, err := f.Seek(size-int64(maxBytes), io.SeekStart); err != nil {
		return "", fmt.Errorf("engine: seek node%d log: %w", n.Index, err)
	}
	buf := make([]byte, maxBytes)
	read, err := io.ReadFull(f, buf)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", fmt.Errorf("engine: read node%d log: %w", n.Index, err)
	}
	return string(buf[:read]), nil
}

// verifyAgainstPlan holds the launched network to the plan and refuses to test
// one that is not it.
//
// Every other check in this package asks whether the declaration is coherent.
// This is the only one that asks whether the network that came up is the one
// described, which matters because the last word belongs to the command line:
// an override naming no layer beats every document, so the merge can be right
// and the nodes still run something else. A test against the wrong network
// does not fail, it answers a question nobody asked.
//
// Reading the record rather than the in-memory state is deliberate: the record
// is what a later reader sees, so a fact that never reached it is a fact the
// run cannot show afterwards either.
