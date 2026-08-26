package driver

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/0xmhha/chainbench/internal/core/remote"
)

// RemoteFileStore reads and writes files on a remote host over SSH. It
// satisfies filestore.Store structurally, so the file interface ships genesis,
// config and key material to another host with no code change on the callers'
// side.
//
// Reads use the same base64 wire format as remote.ReadFile — the format lives
// in one place so a remote read means the same thing wherever it is issued.
type RemoteFileStore struct {
	// Run executes a shell command on the remote host.
	Run Runner
	// Writer ships file content to the host. It only writes; Run covers the
	// read and probe halves of the store.
	Writer FileProvisioner
}

// NewRemoteFileStore returns a RemoteFileStore backed by run for the existence
// probe and reads, and (via a RemoteDriver) for the file writes.
func NewRemoteFileStore(run Runner) RemoteFileStore {
	return RemoteFileStore{Run: run, Writer: NewRemoteDriver(run)}
}

// Exists reports whether remotePath is present on the host. A `test -f` exit of
// 0 means present, non-zero means absent; only a transport failure is an error.
func (s RemoteFileStore) Exists(ctx context.Context, remotePath string) (bool, error) {
	res, err := s.Run(ctx, "test -f "+shq(remotePath))
	if err != nil {
		return false, fmt.Errorf("driver: remote exists %s: %w", remotePath, err)
	}
	return res.ExitCode == 0, nil
}

// Read returns the remote file's bytes.
func (s RemoteFileStore) Read(ctx context.Context, remotePath string) ([]byte, error) {
	res, err := s.Run(ctx, remote.ReadFileCommand(remotePath))
	if err != nil {
		return nil, fmt.Errorf("driver: remote read %s: %w", remotePath, err)
	}
	return remote.DecodeReadFile(remotePath, res)
}

// Write ships content to remotePath with the given mode.
func (s RemoteFileStore) Write(ctx context.Context, remotePath string, content []byte, mode fs.FileMode) error {
	return s.Writer.ProvisionFile(ctx, remotePath, content, mode)
}
