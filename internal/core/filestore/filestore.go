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
	// Remove deletes the file or directory tree at path. Absence is not an
	// error: clearing a composition twice must not fail the second time.
	//
	// It is the one operation here that destroys something, and it is on the
	// interface for the same reason Read is. While the contract could only
	// create, the one caller that had to delete reached past it to os.RemoveAll
	// and could therefore only ever work locally — `chain rm` refused on a
	// remote target because the abstraction had no way to say it. An interface
	// that only builds gets a second one built next to it that tears down.
	//
	// Every implementation applies [CheckRemovable] first. A caller that knows
	// the target's data root should apply [CheckWithin] as well.
	Remove(ctx context.Context, path string) error
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

// Remove deletes the file or directory tree at path. Absence is not an error.
func (Local) Remove(_ context.Context, p string) error {
	if err := CheckRemovable(p); err != nil {
		return err
	}
	return os.RemoveAll(p)
}

// Write creates any parent directories and writes the file with mode.
//
// The file ends up with the mode the caller asked for whether or not something
// was already at path. That is the contract the remote store has always had (it
// runs an explicit chmod), and the local one used to break it: os.WriteFile
// applies its perm only when it CREATES the file, so writing a key onto an
// existing 0644 file left the key world-readable while the caller believed it
// had asked for 0600.
//
// The bytes go to a fresh file that is renamed into place, rather than into
// path directly. A fresh O_EXCL file cannot be a symlink to somewhere else,
// carries its mode from before the content lands rather than after, and the
// rename replaces the name in one step — so a reader never sees a half-written
// file, and a secret is never briefly readable under the old file's mode.
func (Local) Write(_ context.Context, path string, content []byte, mode fs.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, dirPerm); err != nil {
		return err
	}
	// CreateTemp opens with O_EXCL at 0600 — safe by default for the secret
	// case; the requested mode is applied below, before the file gets its name.
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	// Removes the temp file on every failure path; a no-op once renamed.
	defer func() { _ = os.Remove(tmpName) }()

	if _, err := tmp.Write(content); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	// Chmod rather than the open mode: umask applies to the latter, and a
	// caller asking for 0600 must get 0600.
	if err := os.Chmod(tmpName, mode.Perm()); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}
