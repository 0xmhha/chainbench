package chainsetup

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/remote"
	"github.com/0xmhha/chainbench/internal/resource"
)

// Compositions a server still holds that nothing on this machine refers to.
//
// A composition isolated by id writes under node/<id>, runtime/<id> and
// logs/<id> on every server it is placed on, and the id is a hash of the
// workspace directory — so every new --workspace-dir is a new set of
// directories, and `chain rm` is the only thing that removes them. A workspace
// deleted without an rm, or a run that failed before anyone ran one, leaves
// them behind for good. On the 15-container docker set a few hundred of those
// filled the disk.
//
// This finds them. It does not decide on its own: the default is to list, and
// only Apply removes. A composition is kept when a workspace this machine can
// see names its id, or when a process on the server runs out of one of its
// directories — a workspace kept somewhere this was not told to look is still
// safe while its nodes run.

// StaleCompositionsIn is where to look and whether to remove what is found.
type StaleCompositionsIn struct {
	// ServerSet is the server-set file whose servers are searched.
	ServerSet string
	// Docker dials the servers through the localmap next to the set.
	Docker bool
	// WorkspaceConfigPath gives the data root and the node, runtime and logs
	// directories under it.
	WorkspaceConfigPath string
	// KeepUnder are directories searched, recursively, for workspaces whose
	// compositions are kept. ~/.chainbench is always searched.
	KeepUnder []string
	// Apply removes what is found. Without it nothing is changed.
	Apply bool
}

// StaleComposition is one composition's directories on one server.
type StaleComposition struct {
	Server string
	ID     string
	Dirs   []string
}

// StaleCompositionsOut is what was found, and what was done about it.
type StaleCompositionsOut struct {
	// Stale are the compositions nothing refers to.
	Stale []StaleComposition
	// Running are compositions no known workspace names, kept because a
	// process on the server runs out of them.
	Running []StaleComposition
	// Known is how many composition ids the searched workspaces name.
	Known int
	// Removed is how many directories Apply removed.
	Removed int
}

// workspaceRecordFile is the record every workspace keeps, which is what makes a
// directory a workspace.
const workspaceRecordFile = "chain-record.json"

// compositionIDPattern is the shape compositionID produces. Anything else under
// the purpose directories — a flat layout's node1, a file somebody put there —
// is not a composition and is never touched.
var compositionIDPattern = regexp.MustCompile(`^[0-9a-f]{12}$`)

// StaleCompositions lists, and with Apply removes, the compositions the set's
// servers hold that no known workspace refers to and no process runs from.
func StaleCompositions(ctx context.Context, d Deps, in StaleCompositionsIn) (StaleCompositionsOut, error) {
	var out StaleCompositionsOut
	if in.ServerSet == "" || in.WorkspaceConfigPath == "" {
		return out, fmt.Errorf("chainsetup: stale compositions: a server set and a workspace-config are both required")
	}
	wc, err := resource.LoadWorkspaceConfig(in.WorkspaceConfigPath)
	if err != nil {
		return out, err
	}
	set, err := resource.LoadSet(in.ServerSet)
	if err != nil {
		return out, err
	}
	known, err := knownCompositions(in.KeepUnder)
	if err != nil {
		return out, err
	}
	out.Known = len(known)

	root := wc.DataRoot
	purposes := []string{wc.Paths.Nodes, wc.Paths.Runtime, wc.Paths.Logs}
	opener := resource.Opener{ServerSet: in.ServerSet, Docker: in.Docker, Env: d.Env, Report: d.Logf}
	for _, s := range set.Servers {
		acc, err := opener.Open(resource.TargetOf(s, root))
		if err != nil {
			return out, fmt.Errorf("chainsetup: stale compositions: %s: %w", s.Name, err)
		}
		cmd, ok := acc.Driver.(process.Commander)
		if !ok {
			return out, fmt.Errorf("chainsetup: stale compositions: %s cannot run a command", s.Name)
		}
		found, err := compositionsOn(ctx, cmd, root, purposes)
		if err != nil {
			return out, fmt.Errorf("chainsetup: stale compositions: %s: %w", s.Name, err)
		}
		procs, err := cmd.Run(ctx, "ps -eo args")
		if err != nil {
			return out, fmt.Errorf("chainsetup: stale compositions: %s: list processes: %w", s.Name, err)
		}
		for _, id := range sortedKeys(found) {
			if known[id] {
				continue
			}
			c := StaleComposition{Server: s.Name, ID: id, Dirs: found[id]}
			if strings.Contains(procs, "/"+id+"/") {
				out.Running = append(out.Running, c)
				continue
			}
			out.Stale = append(out.Stale, c)
			if !in.Apply {
				continue
			}
			for _, dir := range c.Dirs {
				if err := filestore.CheckWithin(root, dir); err != nil {
					return out, err
				}
				if err := acc.Files.Remove(ctx, dir); err != nil {
					return out, fmt.Errorf("chainsetup: stale compositions: %s: %w", s.Name, err)
				}
				out.Removed++
			}
		}
	}
	return out, nil
}

// compositionsOn is every composition id under the purpose directories on one
// machine, with the directories that hold it.
func compositionsOn(ctx context.Context, cmd process.Commander, root string, purposes []string) (map[string][]string, error) {
	var script strings.Builder
	for _, p := range purposes {
		dir := path.Join(root, p)
		// One line per entry, prefixed by the directory it is in; a purpose
		// directory that does not exist yet lists nothing.
		fmt.Fprintf(&script, "for e in %s/*; do [ -d \"$e\" ] && echo \"$e\"; done; ", remote.ShellQuote(dir))
	}
	listing, err := cmd.Run(ctx, "sh -c "+remote.ShellQuote(script.String()+"true"))
	if err != nil {
		return nil, fmt.Errorf("list compositions: %w", err)
	}
	found := map[string][]string{}
	for _, line := range strings.Split(listing, "\n") {
		line = strings.TrimSpace(line)
		if id := path.Base(line); compositionIDPattern.MatchString(id) {
			found[id] = append(found[id], line)
		}
	}
	return found, nil
}

// knownCompositions is the id of every workspace under ~/.chainbench and under
// each of dirs.
func knownCompositions(dirs []string) (map[string]bool, error) {
	known := map[string]bool{}
	var workspaces []string
	if root, err := DefaultRoot(); err == nil {
		found, err := Discover(root)
		if err != nil {
			return nil, err
		}
		workspaces = append(workspaces, found...)
	}
	for _, dir := range dirs {
		err := filepath.WalkDir(dir, func(p string, e fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if !e.IsDir() && e.Name() == workspaceRecordFile {
				workspaces = append(workspaces, filepath.Dir(p))
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("chainsetup: stale compositions: search %s: %w", dir, err)
		}
	}
	for _, ws := range workspaces {
		if _, err := os.Stat(filepath.Join(ws, workspaceRecordFile)); err != nil {
			continue
		}
		known[compositionID(ws)] = true
	}
	return known, nil
}

func sortedKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
