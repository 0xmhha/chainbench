package provision

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// FileSink writes provisioned files to a target (local FS or remote host) and
// reports whether a path already exists, so existing files are reused rather
// than overwritten (upload-if-absent).
type FileSink interface {
	Exists(ctx context.Context, path string) (bool, error)
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

// Provisioner materializes node inputs through a FileSink.
type Provisioner struct {
	sink FileSink
}

// New returns a Provisioner writing through sink.
func New(sink FileSink) *Provisioner { return &Provisioner{sink: sink} }

// Provision writes each file under in.DataDir, skipping any that already exist
// (reused). It returns how many were written and skipped.
func (p *Provisioner) Provision(ctx context.Context, in NodeInputs) (Result, error) {
	var res Result
	for _, f := range in.Files {
		full := filepath.Join(in.DataDir, f.Path)
		exists, err := p.sink.Exists(ctx, full)
		if err != nil {
			return res, fmt.Errorf("provision: stat %s: %w", full, err)
		}
		if exists {
			res.Skipped++
			continue
		}
		if err := p.sink.Write(ctx, full, f.Content, f.Mode); err != nil {
			return res, fmt.Errorf("provision: write %s: %w", full, err)
		}
		res.Written++
	}
	return res, nil
}

// dirPerm is the permission for directories created under a data dir.
const dirPerm fs.FileMode = 0o755

// LocalFileSink writes to the local filesystem.
type LocalFileSink struct{}

// Exists reports whether path is present locally.
func (LocalFileSink) Exists(_ context.Context, path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// Write creates any parent directories and writes the file.
func (LocalFileSink) Write(_ context.Context, path string, content []byte, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), dirPerm); err != nil {
		return err
	}
	return os.WriteFile(path, content, mode)
}
