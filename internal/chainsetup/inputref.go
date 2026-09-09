package chainsetup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

// wc returns the parsed workspace-config this composition was set up with, or
// nil when none was named. It is loaded once per command; a portable reference
// needs it to resolve its root, while an absolute or srv:// reference does not.
func (w *Workspace) wc() (*resource.WorkspaceConfig, error) {
	if w.wcLoaded {
		return w.wcCache, nil
	}
	w.wcLoaded = true
	if w.state.WorkspaceConfig == "" {
		return nil, nil
	}
	c, err := resource.LoadWorkspaceConfig(w.state.WorkspaceConfig)
	if err != nil {
		return nil, fmt.Errorf("chainsetup: workspace-config: %w", err)
	}
	w.wcCache = &c
	return w.wcCache, nil
}

// indexName maps a 1-based serverIndex to a fixed server name from the
// workspace's server set, so a reference chosen by number resolves to a name
// that a resume records rather than re-counting the list.
func (w *Workspace) indexName(i int) (string, error) {
	if w.state.ServerSet == "" {
		return "", fmt.Errorf("chainsetup: serverIndex needs a server set, but none is recorded")
	}
	set, err := resource.LoadSet(w.state.ServerSet)
	if err != nil {
		return "", err
	}
	s, err := set.Server(i)
	if err != nil {
		return "", err
	}
	return s.Name, nil
}

// readInputRef reads a referenced input file from the machine it names, so a
// reference is never dialed as a local path by mistake. An absolute path is
// read locally (the pre-workspace-config behaviour for a pinned file); every
// other form is resolved through the reference resolver and read on its server
// (or the node's own target when the reference names none). purpose selects the
// directory a portable reference sits under.
//
// It reads a source that already exists on its machine; it does not move a file
// between machines (that transfer is a separate, explicit contract).
func (w *Workspace) readInputRef(ctx context.Context, ns node.Record, ref string, purpose resource.Purpose) ([]byte, error) {
	// An absolute, non-srv path is a local file named directly — the behaviour a
	// pinned config or key relied on before workspace-config existed.
	if filepath.IsAbs(ref) && !strings.HasPrefix(ref, "srv://") {
		return os.ReadFile(ref)
	}
	wc, err := w.wc()
	if err != nil {
		return nil, err
	}
	loc, err := resource.InputRef{Raw: ref}.Resolve(wc, purpose, w.indexName)
	if err != nil {
		return nil, err
	}
	if loc.Local {
		return os.ReadFile(loc.Path)
	}
	// A reference naming no server resolves on the node's own target; a srv://
	// reference resolves on the server it names, whose host and credentials come
	// from the server set through the opener (not the node's host).
	var t *resource.Access
	if loc.Server == "" {
		t, err = w.machineFor(ns)
	} else {
		t, err = w.opener().Open(resource.Spec{Server: loc.Server})
	}
	if err != nil {
		return nil, err
	}
	b, err := t.Files.Read(ctx, loc.Path)
	if err != nil {
		return nil, fmt.Errorf("chainsetup: read %s reference %q: %w", purpose, ref, err)
	}
	return b, nil
}
