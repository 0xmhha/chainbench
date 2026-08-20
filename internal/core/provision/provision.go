package provision

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// FileStore is the one way chainbench touches files on a target, whether the
// target is this machine or a host reached over SSH. Callers above it do not
// branch on which.
//
// It reads as well as writes on purpose. While the contract was write-only,
// every component that needed to read a remote file grew its own SSH read
// beside it — key material got one, the wemix deploy got another — and those
// copies are where local and remote drifted apart. An abstraction that only
// goes one way gets a second one built next to it going the other.
type FileStore interface {
	// Exists reports whether path is present. Absence is not an error.
	Exists(ctx context.Context, path string) (bool, error)
	// Read returns the file's bytes.
	Read(ctx context.Context, path string) ([]byte, error)
	// Write creates any parent directories and writes the file with mode.
	Write(ctx context.Context, path string, content []byte, mode fs.FileMode) error
}

// File is one file to place, at a path relative to the node's data dir.
type File struct {
	Path    string
	Content []byte
	Mode    fs.FileMode
}

// NodeInputs is one node's on-disk materials (config, genesis, keys, ...).
type NodeInputs struct {
	DataDir string
	Files   []File
}

// Result counts what a Provision call wrote versus reused.
type Result struct {
	Written int
	Skipped int
}

// Provisioner materializes node inputs through a FileStore.
type Provisioner struct {
	store FileStore
}

// New returns a Provisioner writing through store.
func New(store FileStore) *Provisioner { return &Provisioner{store: store} }

// Provision writes each file under in.DataDir, skipping any that already exist
// (reused). It returns how many were written and skipped.
func (p *Provisioner) Provision(ctx context.Context, in NodeInputs) (Result, error) {
	var res Result
	for _, f := range in.Files {
		full := filepath.Join(in.DataDir, f.Path)
		exists, err := p.store.Exists(ctx, full)
		if err != nil {
			return res, fmt.Errorf("provision: stat %s: %w", full, err)
		}
		if exists {
			res.Skipped++
			continue
		}
		if err := p.store.Write(ctx, full, f.Content, f.Mode); err != nil {
			return res, fmt.Errorf("provision: write %s: %w", full, err)
		}
		res.Written++
	}
	return res, nil
}

// dirPerm is the permission for directories created under a data dir.
const dirPerm fs.FileMode = 0o755

// LocalFileStore reads and writes on the local filesystem.
type LocalFileStore struct{}

// Exists reports whether path is present locally.
func (LocalFileStore) Exists(_ context.Context, path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// Read returns the file's bytes.
func (LocalFileStore) Read(_ context.Context, path string) ([]byte, error) {
	return os.ReadFile(path)
}

// Write creates any parent directories and writes the file.
func (LocalFileStore) Write(_ context.Context, path string, content []byte, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return err
	}
	return os.WriteFile(path, content, mode)
}
