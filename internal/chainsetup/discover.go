package chainsetup

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/resource"
)

// Workspaces are found, not registered. There is no index of them — a
// workspace is a directory with a workspace.json in it — so the default root
// is where compositions land when no --workspace-dir is given, and Discover
// is how a later command sees the ones that exist. A workspace composed
// elsewhere is counted only when it is named.

// defaultRootDir is the directory under the operator's home that holds
// timestamped compositions.
const defaultRootDir = ".chainbench"

// setupDir is the segment under a composition that holds its setup: the
// workspace.json and everything the steps generate before launch.
const setupDir = "chainsetup"

// timestampLayout names a composition by when it began, to the second — two
// compositions a second apart do not collide, and an operator can find the
// last one by sorting.
const timestampLayout = "20060102-150405"

// DefaultRoot is where compositions default to: ~/.chainbench.
func DefaultRoot() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("chainsetup: no home directory to default the workspace under: %w", err)
	}
	return filepath.Join(home, defaultRootDir), nil
}

// DefaultWorkspaceDir is the workspace a composition gets when none is named:
// ~/.chainbench/<YYYYMMDD-HHMMSS>/chainsetup. A <test-name> segment joins the
// scheme when the test-running surface exists to supply one.
//
// The path is deliberately short — node IPC sockets live under it and a unix
// socket path is capped at 104 characters.
func DefaultWorkspaceDir(now func() time.Time) (string, error) {
	root, err := DefaultRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, now().Format(timestampLayout), setupDir), nil
}

// Discover lists the workspaces under root, oldest first. A missing root is
// simply no workspaces, not an error: the first composition has nothing to
// find.
func Discover(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("chainsetup: discover workspaces under %s: %w", root, err)
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(root, e.Name(), setupDir)
		if _, err := os.Stat(session.CompositionFilePath(dir)); err == nil {
			out = append(out, dir)
		}
	}
	return out, nil
}

// Allocations are the claims a workspace's node records make on a resource,
// in the shape the inventory adopts. The workspace is the record; this is a
// reading of it, never a second copy.
func Allocations(w *Workspace) []resource.Allocation {
	st := w.State()
	out := make([]resource.Allocation, 0, len(st.Nodes))
	for _, r := range st.Nodes {
		out = append(out, resource.Allocation{
			Network: w.Dir(),
			Node:    string(node.Record(r).NodeLabel()),
			Host:    hostOf(r),
			P2P:     r.P2P,
		})
	}
	return out
}

// hostOf is the address a record claims: its server-set entry when it names
// one, else the address it recorded.
func hostOf(r node.Record) string {
	if r.Server != "" {
		return r.Server
	}
	return r.Host
}
