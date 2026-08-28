// Package driver runs chain nodes on behalf of the setup pipeline. A Driver
// abstracts where/how a node runs so the pipeline treats local, remote, and
// attached nodes through one surface (requirements #5–7,
// docs/CHAINBENCH_GO_REDESIGN.md §7). This package ships the local driver; the
// remote (ssh key / id·pw) and attach drivers, ported from
// network/internal/drivers, land with G3.
package driver

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/0xmhha/chainbench/internal/core/node"
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
	// SyncMode is the node's sync mode (full|snap|archive); empty means the
	// pipeline picks a role-based default. Set per-node from a topology config.
	SyncMode string
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

// Initializer is an optional Driver capability: place the genesis on the node's
// host and run the datadir `init`, so the genesis-and-init step goes through the
// same driver as provision/launch (local writes+execs; remote ships+execs over
// SSH) instead of assuming a local filesystem. A caller type-asserts a Driver to
// Initializer and falls back to the local InitDatadir when it is not supported.
type Initializer interface {
	InitDatadir(ctx context.Context, spec NodeSpec, genesis []byte) error
}

// FileProvisioner is an optional Driver capability: place an arbitrary file on
// the node's host. It is how the setup pipeline ships per-node identity files
// (devp2p nodekey, validator keystore, password) to a remote host — the local
// driver has the files in place already and does not implement it, so the caller
// type-asserts and only ships when the driver requires it (i.e. is remote).
type FileProvisioner interface {
	ProvisionFile(ctx context.Context, remotePath string, content []byte, mode fs.FileMode) error
}

// NodeOf is the runtime node a launched spec becomes: the identity and ports
// the spec carried, the pid the launch returned, and the RPC URL derived from
// them. It is the one place a Node is assembled from a launch — three copies
// of these six lines used to exist, and the etcd port was lost in one.
func NodeOf(spec NodeSpec, pid int) node.Node {
	return node.Node{
		Index:  spec.Index,
		Role:   spec.Role,
		Host:   spec.Host,
		RPCURL: fmt.Sprintf("http://%s:%d", spec.Host, spec.Ports.HTTP),
		Ports:  spec.Ports,
		PID:    pid,
	}
}

// localHost is where a node with no recorded host is reachable: this machine.
const localHost = "127.0.0.1"

// SpecOf is the launch description a recorded node becomes: what the record
// says about identity, paths, ports and argv, with this machine as the host
// when the record names none. The binary is not on the record — the caller
// resolves it — so it is left for the caller to set.
func SpecOf(r node.Record) NodeSpec {
	host := r.Host
	if host == "" {
		host = localHost
	}
	return NodeSpec{
		Index:      r.Index,
		Role:       node.Role(r.Role),
		SyncMode:   r.SyncMode,
		Host:       host,
		DataDir:    r.DataDir,
		ConfigPath: r.ConfigPath,
		LogPath:    r.LogPath,
		Args:       r.Args,
		Ports:      r.Endpoints,
	}
}
