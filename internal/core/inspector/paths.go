package inspector

import (
	"context"
	"fmt"
	"sort"

	"github.com/0xmhha/chainbench/internal/core/filestore"
)

// Path is one file or directory a plan expects on the target, named by what
// it is for so a report can say "node2 config" rather than a bare path.
type Path struct {
	Path string
	// Node is the 1-based node index the path belongs to; 0 for a network-wide
	// file such as the genesis or the binary.
	Node int
	// Purpose is the path's role ("binary", "genesis", "datadir", "config",
	// "nodekey", "keystore", "log").
	Purpose string
}

// String renders a path the way an operator reads it.
func (p Path) String() string {
	if p.Node == 0 {
		return fmt.Sprintf("%s (%s)", p.Path, p.Purpose)
	}
	return fmt.Sprintf("%s (node%d %s)", p.Path, p.Node, p.Purpose)
}

// Paths reports which of the given paths are absent on the target, through the
// store that reaches it — this machine's filesystem, or a remote host's over
// the file seam — so the answer comes from where the files would be.
//
// Absence is the only thing it reports. A path that exists but is unreadable
// or the wrong shape is found by the step that reads it, which can say what
// shape it wanted.
func Paths(ctx context.Context, store filestore.Store, paths []Path) ([]Path, error) {
	var missing []Path
	for _, p := range paths {
		if p.Path == "" {
			continue
		}
		ok, err := store.Exists(ctx, p.Path)
		if err != nil {
			return nil, fmt.Errorf("inspector: %s: %w", p, err)
		}
		if !ok {
			missing = append(missing, p)
		}
	}
	sort.Slice(missing, func(i, j int) bool {
		if missing[i].Node != missing[j].Node {
			return missing[i].Node < missing[j].Node
		}
		return missing[i].Path < missing[j].Path
	})
	return missing, nil
}
