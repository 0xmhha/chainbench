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
}

// DataDir is the node's datadir — what --datadir points at.
func (l Layout) DataDir(label Label) string {
	return filepath.Join(l.Root, string(label))
}

// ConfigPath is the node's rendered TOML config.
func (l Layout) ConfigPath(label Label) string {
	return filepath.Join(l.Root, "config_"+string(label)+".toml")
}

// LogPath is where the node's stdout/stderr is captured. Logs share one
// directory so a run can be read as a whole.
func (l Layout) LogPath(label Label) string {
	return filepath.Join(l.Root, "logs", string(label)+".log")
}

// GenesisPath is the network's genesis, which is shared rather than per-node.
func (l Layout) GenesisPath() string {
	return filepath.Join(l.Root, "genesis.json")
}
