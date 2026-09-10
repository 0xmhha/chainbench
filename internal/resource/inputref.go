package resource

import (
	"fmt"
	"strings"
)

// Location is where a referenced input file lives: which server it is on and
// the absolute path there. Server is empty when the file is on the
// composition's own target (a portable reference resolved under the
// workspace-config data root). Local is true when it is on the machine running
// chainbench (a localPath). A resolved location always names the machine, so a
// reference is never dialed as a local path by mistake.
type Location struct {
	// Server is the server-set entry the file lives on; empty means the
	// composition's target.
	Server string
	// Path is the absolute path to the file on that machine.
	Path string
	// Local is true when the file is on the machine running chainbench rather
	// than on the target or a server.
	Local bool
}

// InputRef is a file reference in the portable forms the DSL and workspace
// config accept. Exactly one form is set:
//
//	Raw "srv://<name>/<abs>"  a named server's absolute path (data root not added)
//	Raw "<relative>"          the target, under dataRoot/paths[purpose]/<ref>
//	Server + Ref              a named server, under its workspace purpose path
//	ServerIndex + Ref         a server chosen by 1-based order, then as Server
//	LocalPath                 a file on the machine running chainbench
//
// Server and ServerIndex are mutually exclusive; naming a server does not move
// node placement, and resolving a reference performs no I/O.
type InputRef struct {
	Raw         string
	Server      string
	ServerIndex int
	Ref         string
	LocalPath   string
}

// forms reports how many distinct reference forms this ref sets. Exactly one is
// required; zero or more than one is a malformed reference.
func (r InputRef) forms() int {
	n := 0
	if r.Raw != "" {
		n++
	}
	if r.Server != "" {
		n++
	}
	if r.ServerIndex != 0 {
		n++
	}
	if r.LocalPath != "" {
		n++
	}
	return n
}

// Resolve turns a reference into the machine and absolute path it names.
//
// wc supplies the data root and purpose directories for the portable and
// server/index forms; purpose selects which directory a relative ref sits
// under. indexName maps a 1-based ServerIndex to a fixed server name (the
// caller passes a lookup over the resolved server set) so a later resume can
// store the name rather than re-count the list. A srv:// reference is an
// absolute server path and consults none of these.
func (r InputRef) Resolve(wc *WorkspaceConfig, purpose Purpose, indexName func(int) (string, error)) (Location, error) {
	switch {
	case r.forms() == 0:
		return Location{}, fmt.Errorf("resource: input reference is empty")
	case r.forms() > 1:
		return Location{}, fmt.Errorf("resource: input reference sets more than one form (use exactly one of srv:// or a relative ref, {server,ref}, {serverIndex,ref}, or {localPath})")
	}

	switch {
	case r.LocalPath != "":
		return Location{Local: true, Path: r.LocalPath}, nil

	case r.Server != "":
		if r.Ref == "" {
			return Location{}, fmt.Errorf("resource: {server: %q} needs a ref", r.Server)
		}
		return r.serverRefLocation(wc, purpose, r.Server)

	case r.ServerIndex != 0:
		if r.Ref == "" {
			return Location{}, fmt.Errorf("resource: {serverIndex: %d} needs a ref", r.ServerIndex)
		}
		if r.ServerIndex < 0 {
			return Location{}, fmt.Errorf("resource: serverIndex %d must be 1-based", r.ServerIndex)
		}
		if indexName == nil {
			return Location{}, fmt.Errorf("resource: serverIndex needs a server set to resolve against")
		}
		name, err := indexName(r.ServerIndex)
		if err != nil {
			return Location{}, err
		}
		return r.serverRefLocation(wc, purpose, name)

	default: // Raw
		if strings.HasPrefix(r.Raw, "srv://") {
			spec, err := Parse(r.Raw)
			if err != nil {
				return Location{}, err
			}
			// Parse puts the absolute path in DataRoot for a srv:// reference; the
			// name resolves to credentials at open time, not here.
			return Location{Server: spec.Server, Path: spec.DataRoot}, nil
		}
		if wc == nil {
			return Location{}, fmt.Errorf("resource: portable reference %q needs a workspace-config to resolve its root", r.Raw)
		}
		p, err := wc.Resolve(purpose, r.Raw)
		if err != nil {
			return Location{}, err
		}
		return Location{Path: p}, nil
	}
}

// serverRefLocation resolves a {server, ref} (or resolved {serverIndex}) to the
// named server and the workspace purpose path for ref. The path is computed
// with the workspace-config's data root and purpose directory, on that server.
func (r InputRef) serverRefLocation(wc *WorkspaceConfig, purpose Purpose, server string) (Location, error) {
	if wc == nil {
		return Location{}, fmt.Errorf("resource: a {server, ref} needs a workspace-config to resolve its purpose path")
	}
	p, err := wc.Resolve(purpose, r.Ref)
	if err != nil {
		return Location{}, err
	}
	return Location{Server: server, Path: p}, nil
}
