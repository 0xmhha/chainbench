package app

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/resource"
)

// Transferring files to and from a server's data plane.
//
// Both directions open the named server through the workspace-config's data
// root — the one owner of where the data plane lives — and reach files through
// the same Access the composition uses, elevating through sudo where the server
// set permits it. A destination is named either by purpose (a folder the
// workspace-config defines: bin, genesis, configs, keystore, keyrings) plus a
// file name, or by an explicit absolute path on the target.

// uploadPurposes maps the operator-facing purpose names to the workspace-config
// purpose directories. "bin" is the friendly spelling of the binaries folder.
var transferPurposes = map[string]resource.Purpose{
	"bin":      resource.PurposeBinaries,
	"binaries": resource.PurposeBinaries,
	"genesis":  resource.PurposeGenesis,
	"configs":  resource.PurposeConfigs,
	"keystore": resource.PurposeKeystore,
	"keyrings": resource.PurposeKeyrings,
}

// TransferServer names the server a transfer acts on and how to reach it.
type TransferServer struct {
	ServerSet           string
	Server              string
	Docker              bool
	WorkspaceConfigPath string
}

// UploadIn is a request to place local files on a server.
type UploadIn struct {
	Target TransferServer
	// Purpose is the destination folder by name (with a file per local input);
	// RemotePath is an explicit absolute destination (for a single local file).
	// Exactly one must be set.
	Purpose    string
	RemotePath string
	LocalPaths []string
	// Force overwrites a destination that already exists; without it, a name
	// collision is refused.
	Force bool
}

// UploadOut reports the target paths written.
type UploadOut struct {
	Uploaded []string
}

// Upload places local files on the server, refusing a name collision unless
// Force is set.
func Upload(ctx context.Context, d Deps, in UploadIn) (UploadOut, error) {
	if len(in.LocalPaths) == 0 {
		return UploadOut{}, fmt.Errorf("upload: no local file given")
	}
	wc, acc, err := openTransferTarget(d, in.Target)
	if err != nil {
		return UploadOut{}, err
	}
	dests, err := uploadDests(wc, in)
	if err != nil {
		return UploadOut{}, err
	}
	var out UploadOut
	for i, local := range in.LocalPaths {
		if err := acc.Upload(ctx, local, dests[i], in.Force); err != nil {
			return out, err
		}
		out.Uploaded = append(out.Uploaded, dests[i])
	}
	return out, nil
}

// DownloadIn is a request to copy a server file to a local path.
type DownloadIn struct {
	Target TransferServer
	// Purpose+Name names the source folder and file; RemotePath is an explicit
	// absolute source. Exactly one form must be set.
	Purpose    string
	Name       string
	RemotePath string
	// Out is the local path to write; the file lands 0600.
	Out string
}

// DownloadOut reports the local path written.
type DownloadOut struct {
	Out string
}

// Download copies a server file to a local path, elevating through sudo where
// needed and permitted.
func Download(ctx context.Context, d Deps, in DownloadIn) (DownloadOut, error) {
	if in.Out == "" {
		return DownloadOut{}, fmt.Errorf("download: --out local path is required")
	}
	wc, acc, err := openTransferTarget(d, in.Target)
	if err != nil {
		return DownloadOut{}, err
	}
	src, err := transferSource(wc, in.Purpose, in.Name, in.RemotePath)
	if err != nil {
		return DownloadOut{}, err
	}
	if err := acc.DownloadTo(ctx, src, in.Out); err != nil {
		return DownloadOut{}, err
	}
	return DownloadOut{Out: in.Out}, nil
}

// openTransferTarget loads the workspace-config (the data-root owner) and opens
// the named server through it. A remote server needs the data root to resolve,
// so the workspace-config is required for both directions.
func openTransferTarget(d Deps, t TransferServer) (resource.WorkspaceConfig, *resource.Access, error) {
	if t.WorkspaceConfigPath == "" {
		return resource.WorkspaceConfig{}, nil, fmt.Errorf("transfer: --workspace-config is required (it owns the target data root)")
	}
	if t.Server == "" {
		return resource.WorkspaceConfig{}, nil, fmt.Errorf("transfer: --server is required")
	}
	wc, err := resource.LoadWorkspaceConfig(t.WorkspaceConfigPath)
	if err != nil {
		return resource.WorkspaceConfig{}, nil, err
	}
	opener := resource.Opener{ServerSet: t.ServerSet, Docker: t.Docker, Env: d.Env, Report: d.Logf}
	acc, err := opener.Open(resource.Spec{Server: t.Server, DataRoot: wc.DataRoot})
	if err != nil {
		return resource.WorkspaceConfig{}, nil, err
	}
	return wc, acc, nil
}

// uploadDests resolves each local file's destination path. With a purpose, each
// file lands under that folder by its base name; with an explicit remote path,
// exactly one local file lands there.
func uploadDests(wc resource.WorkspaceConfig, in UploadIn) ([]string, error) {
	switch {
	case in.Purpose != "" && in.RemotePath != "":
		return nil, fmt.Errorf("upload: name a destination by --purpose or --remote, not both")
	case in.RemotePath != "":
		if len(in.LocalPaths) != 1 {
			return nil, fmt.Errorf("upload: --remote takes exactly one local file, got %d", len(in.LocalPaths))
		}
		return []string{in.RemotePath}, nil
	case in.Purpose != "":
		purpose, ok := transferPurposes[in.Purpose]
		if !ok {
			return nil, fmt.Errorf("upload: unknown purpose %q (want bin, genesis, configs, keystore, or keyrings)", in.Purpose)
		}
		dests := make([]string, len(in.LocalPaths))
		for i, local := range in.LocalPaths {
			dest, err := wc.Resolve(purpose, filepath.Base(local))
			if err != nil {
				return nil, err
			}
			dests[i] = dest
		}
		return dests, nil
	default:
		return nil, fmt.Errorf("upload: a destination is required (--purpose or --remote)")
	}
}

// transferSource resolves a download's source path, by purpose+name or by an
// explicit remote path.
func transferSource(wc resource.WorkspaceConfig, purpose, name, remotePath string) (string, error) {
	switch {
	case purpose != "" && remotePath != "":
		return "", fmt.Errorf("download: name a source by --purpose or --remote, not both")
	case remotePath != "":
		return remotePath, nil
	case purpose != "":
		p, ok := transferPurposes[purpose]
		if !ok {
			return "", fmt.Errorf("download: unknown purpose %q (want bin, genesis, configs, keystore, or keyrings)", purpose)
		}
		if name == "" {
			return "", fmt.Errorf("download: --name is required with --purpose")
		}
		return wc.Resolve(p, name)
	default:
		return "", fmt.Errorf("download: a source is required (--purpose with --name, or --remote)")
	}
}
