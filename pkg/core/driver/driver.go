// Package driver runs chain nodes on behalf of the setup pipeline. A Driver
// abstracts where/how a node runs so the pipeline treats local, remote, and
// attached nodes through one surface (requirements #5–7,
// docs/CHAINBENCH_GO_REDESIGN.md §7). This package ships the local driver; the
// remote (ssh key / id·pw) and attach drivers, ported from
// network/internal/drivers, land with G3.
package driver

import (
	"context"

	"github.com/0xmhha/chainbench/pkg/core/node"
)

// NodeSpec is the fully-resolved launch description for one node, produced by
// the setup planner. It is self-contained: a driver needs nothing else to
// provision and launch the node.
type NodeSpec struct {
	Index         int
	Role          node.Role
	Host          string
	Binary        string
	DataDir       string
	ConfigPath    string // where the node config file is written
	ConfigContent []byte // node config bytes (empty = nothing to write)
	LogPath       string
	Args          []string // full launch args (excluding the binary itself)
	Ports         node.Endpoints
}

// Handle identifies a launched node so it can later be stopped.
type Handle struct {
	Index int
	PID   int
}

// Driver provisions, launches, and stops nodes.
type Driver interface {
	// Provision prepares the node's on-disk environment (data dir, config
	// file). It is idempotent.
	Provision(ctx context.Context, spec NodeSpec) error
	// Launch starts the node process and returns a Handle. It does not block
	// on the node's lifetime.
	Launch(ctx context.Context, spec NodeSpec) (Handle, error)
	// Stop terminates a previously launched node.
	Stop(ctx context.Context, h Handle) error
}
