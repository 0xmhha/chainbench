package driver

import (
	"context"
	"io/fs"
)

// Transport is the unified local/remote seam (DDD context C7) over which upper
// layers reach and control a node without knowing whether it is local or
// remote. Local uses exec/os/signals on the loopback address; remote uses
// SSH/SFTP. It is stateless: no remote agent is deployed.
//
// Status: interface freeze only (T0.1). It formalizes the existing Driver plus
// its Initializer/FileProvisioner capabilities; implementations land in T2.1.
type Transport interface {
	// Exec runs cmd (nohup for launches) and returns its stdout and the PID.
	Exec(ctx context.Context, cmd string) (stdout string, pid int, err error)
	// PutFile writes content to path (local write, or remote SFTP ship).
	PutFile(ctx context.Context, path string, content []byte, mode fs.FileMode) error
	// GetFile reads an existing file (local read, or remote download).
	GetFile(ctx context.Context, path string) ([]byte, error)
	// Kill terminates the process by PID (local signal, or remote ssh kill).
	Kill(ctx context.Context, pid int) error
	// TailLog streams appended log lines until ctx is done.
	TailLog(ctx context.Context, path string) (<-chan string, error)
	// Endpoint returns host:port for a node port: loopback locally, the server
	// IP remotely.
	Endpoint(port int) string
}
