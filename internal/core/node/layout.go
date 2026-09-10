package node

import "path/filepath"

// Layout derives every path a node's files live at from one root and the node's
// label. It computes; it does not write — materializing belongs to the file
// store, and keeping the two apart is what lets the same derivation serve a
// local workspace and a remote destination without branching.
//
// It exists because these paths were built with fmt.Sprintf("node%d") at six
// call sites, each free to disagree about the shape. A node's directory is
// named after the node, so the label is the only input that varies.
type Layout struct {
	// Root is the data root on the target: this machine's workspace, or the
	// destination directory on a server.
	Root string
	// CompositionID, when set, isolates this composition's node data under
	// Root/NodesDir/<CompositionID>/<label> so two compositions sharing one data
	// root do not collide on a datadir. Empty keeps the flat Root/<label>
	// layout, which every workspace composed before this used.
	CompositionID string
	// NodesDir is the directory node datadirs sit under when CompositionID is
	// set (the workspace-config's paths.nodes). Empty defaults to "node".
	NodesDir string
	// RuntimeDir is where generated genesis and configs sit, per composition,
	// when CompositionID is set (the workspace-config's paths.runtime). Empty
	// defaults to "runtime". It keeps a composition's generated files off the
	// shared data root so two compositions do not clobber one genesis or config.
	RuntimeDir string
	// LogsDir is where node logs sit, per composition, when CompositionID is set
	// (the workspace-config's paths.logs). Empty defaults to "logs".
	LogsDir string
}

// runtimeBase is the per-composition directory generated files sit under.
func (l Layout) runtimeBase() string {
	dir := l.RuntimeDir
	if dir == "" {
		dir = "runtime"
	}
	return filepath.Join(l.Root, dir, l.CompositionID)
}

// DataDir is the node's datadir — what --datadir points at. With a
// CompositionID it is isolated per composition; without one it is the flat
// Root/<label> a pre-workspace-config layout used.
func (l Layout) DataDir(label Label) string {
	if l.CompositionID == "" {
		return filepath.Join(l.Root, string(label))
	}
	nodes := l.NodesDir
	if nodes == "" {
		nodes = "node"
	}
	return filepath.Join(l.Root, nodes, l.CompositionID, string(label))
}

// ConfigPath is the node's rendered TOML config — a generated file, so it sits
// under the composition's runtime directory when isolated, or flat otherwise.
func (l Layout) ConfigPath(label Label) string {
	if l.CompositionID == "" {
		return filepath.Join(l.Root, "config_"+string(label)+".toml")
	}
	return filepath.Join(l.runtimeBase(), "configs", string(label)+".toml")
}

// LogPath is where the node's stdout/stderr is captured. Isolated per
// composition when an id is set; otherwise one shared logs directory.
func (l Layout) LogPath(label Label) string {
	if l.CompositionID == "" {
		return filepath.Join(l.Root, "logs", string(label)+".log")
	}
	logs := l.LogsDir
	if logs == "" {
		logs = "logs"
	}
	return filepath.Join(l.Root, logs, l.CompositionID, string(label)+".log")
}

// GenesisPath is the network's genesis, shared across nodes but per composition:
// a generated genesis sits under the composition's runtime directory when
// isolated, so two compositions on one data root do not clobber one genesis.
func (l Layout) GenesisPath() string {
	if l.CompositionID == "" {
		return filepath.Join(l.Root, "genesis.json")
	}
	return filepath.Join(l.runtimeBase(), "genesis.json")
}

// NodekeyPath is the node's devp2p private key inside its datadir — where a
// geth-family binary looks for it.
func (l Layout) NodekeyPath(label Label) string {
	return filepath.Join(l.DataDir(label), "nodekey")
}

// KeystoreDir is the node's account keystore directory inside its datadir.
func (l Layout) KeystoreDir(label Label) string {
	return filepath.Join(l.DataDir(label), "keystore")
}

// StaticNodesPath is the node's static peer list inside its datadir.
func (l Layout) StaticNodesPath(label Label) string {
	return filepath.Join(l.DataDir(label), "static-nodes.json")
}

// IPCPath is the node's console socket: inside its datadir, named after the
// binary that serves it. The datadir placement is why a workspace path must
// stay short — a unix socket path is capped at 104 characters.
func (l Layout) IPCPath(label Label, binary string) string {
	return filepath.Join(l.DataDir(label), filepath.Base(binary)+".ipc")
}
