package driver

import (
	"context"
	"fmt"
	"strconv"
)

// RemoteLogReader reads a node's log from another host over SSH. It satisfies
// the collector's LogReader boundary structurally, so the same tail loop follows a
// remote node's log with no change above it — which is the whole point of the
// boundary.
//
// It uses `tail -c +N`, whose offset is 1-based, so byte offset 0 is `+1`. That
// spelling matters: `tail -n` would re-read by lines and lose the exact byte
// position the collector tracks to avoid duplicating or splitting a line.
type RemoteLogReader struct {
	// Run executes a shell command on the remote host.
	Run Runner
}

// NewRemoteLogReader returns a reader that tails logs on the host run targets.
func NewRemoteLogReader(run Runner) RemoteLogReader { return RemoteLogReader{Run: run} }

// ReadFrom returns the log bytes at or after offset. A missing file is not an
// error: tailing may start before the node writes its first line, exactly as
// with a local log.
func (r RemoteLogReader) ReadFrom(ctx context.Context, path string, offset int64) ([]byte, error) {
	if r.Run == nil {
		return nil, fmt.Errorf("driver: remote log reader has no runner")
	}
	cmd := "tail -c +" + strconv.FormatInt(offset+1, 10) + " " + shq(path) + " 2>/dev/null"
	res, err := r.Run(ctx, cmd)
	if err != nil {
		return nil, fmt.Errorf("driver: remote tail %s: %w", path, err)
	}
	if res.ExitCode != 0 {
		return nil, nil // no such file yet
	}
	return []byte(res.Stdout), nil
}
