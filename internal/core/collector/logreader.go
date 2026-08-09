package collector

import (
	"context"
	"io"
	"os"
)

// LogReader reads a node's log from a byte offset. It is the seam that makes the
// tail loop indifferent to where the log lives: locally it is a file read,
// remotely an SSH command. Returning fewer bytes than exist is fine — the tail
// polls, and the offset only advances past complete lines.
type LogReader interface {
	ReadFrom(ctx context.Context, path string, offset int64) ([]byte, error)
}

// LocalLogReader reads from the local filesystem. A missing file is not an
// error: tailing may start before the node writes its first line.
type LocalLogReader struct{}

func (LocalLogReader) ReadFrom(_ context.Context, path string, offset int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil
	}
	defer func() { _ = f.Close() }()
	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, nil
	}
	return io.ReadAll(f)
}
