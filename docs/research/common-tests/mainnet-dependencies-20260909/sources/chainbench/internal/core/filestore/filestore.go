package filestore

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Hash returns the content address of b: "sha256:" followed by the lowercase
// hex digest. It is the one checksum form the file interface and the session
// artifact references (session.ArtifactRef.Ref) share, so a reused file and its
// recorded reference speak the same language.
func Hash(b []byte) string {
	sum := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// Store is the one way chainbench touches files on a target, whether the
// target is this machine or a host reached over SSH. Callers above it do not
// branch on which.
//
// It reads as well as writes on purpose. While the contract was write-only,
// every component that needed to read a remote file grew its own SSH read
// beside it — key material got one, the wemix deploy got another — and those
// copies are where local and remote drifted apart. An abstraction that only
// goes one way gets a second one built next to it going the other.
type Store interface {
	// Exists reports whether path is present. Absence is not an error.
	Exists(ctx context.Context, path string) (bool, error)
	// Read returns the file's bytes.
	Read(ctx context.Context, path string) ([]byte, error)
	// Write creates any parent directories and writes the file with mode.
	Write(ctx context.Context, path string, content []byte, mode fs.FileMode) error
	// Checksum returns the content address ("sha256:<hex>", the Hash form) of the
	// file at path. Reuse is decided on content, not just presence: a file
	// already there with a different checksum is a different file and must be
	// rewritten, while one with the same checksum is never re-sent. Callers gate
	// on Exists first, so a missing file is an error here. A remote store hashes
	// on the host, so an unchanged file is not downloaded just to compare it.
	Checksum(ctx context.Context, path string) (string, error)
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

// Result counts what a Provision call wrote versus reused. Replaced is a file
// that existed with different content and was overwritten — the case the old
// exists-only reuse silently mistook for a reuse.
type Result struct {
	Written  int
	Skipped  int
	Replaced int
}

// Provisioner materializes node inputs through a Store.
type Provisioner struct {
	store Store
}

// New returns a Provisioner writing through store.
func New(store Store) *Provisioner { return &Provisioner{store: store} }

// Provision writes each file under in.DataDir, skipping any that already exist
// (reused). It returns how many were written and skipped.
func (p *Provisioner) Provision(ctx context.Context, in NodeInputs) (Result, error) {
	var res Result
	for _, f := range in.Files {
		full := filepath.Join(in.DataDir, f.Path)
		exists, err := p.store.Exists(ctx, full)
		if err != nil {
			return res, fmt.Errorf("filestore: stat %s: %w", full, err)
		}
		if exists {
			have, err := p.store.Checksum(ctx, full)
			if err != nil {
				return res, fmt.Errorf("filestore: checksum %s: %w", full, err)
			}
			if have == Hash(f.Content) {
				// Identical content is already there: never re-send it.
				res.Skipped++
				continue
			}
			// Same path, different content: not the file we mean. Overwrite it,
			// so a stale artifact under a reused name is never mistaken for ours.
		}
		if err := p.store.Write(ctx, full, f.Content, f.Mode); err != nil {
			return res, fmt.Errorf("filestore: write %s: %w", full, err)
		}
		if exists {
			res.Replaced++
		} else {
			res.Written++
		}
	}
	return res, nil
}

// dirPerm is the permission for directories created under a data dir.
const dirPerm fs.FileMode = 0o755

// Local reads and writes on the local filesystem.
type Local struct{}

// Exists reports whether path is present locally.
func (Local) Exists(_ context.Context, path string) (bool, error) {
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
func (Local) Read(_ context.Context, path string) ([]byte, error) {
	return os.ReadFile(path)
}

// Checksum reads the file and returns its content address ("sha256:<hex>").
func (Local) Checksum(_ context.Context, path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return Hash(b), nil
}

// Write creates any parent directories and writes the file.
func (Local) Write(_ context.Context, path string, content []byte, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return err
	}
	return os.WriteFile(path, content, mode)
}
