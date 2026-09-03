package collector

import (
	"context"
	"io"
	"os"
	"time"
)

// LogReader reads a node's log from a byte offset. It is the boundary that makes the
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

// DefaultReconnectAttempts and DefaultReconnectBackoff tune a reader's recovery
// from a dropped connection when the caller does not specify.
const (
	DefaultReconnectAttempts = 3
	DefaultReconnectBackoff  = time.Second
)

// ReconnectingLogReader wraps a LogReader so a failed read — an SSH session a
// remote host dropped — is retried with backoff and the tail resumes from the
// same offset, rather than the log going silent. A read that keeps failing
// through every attempt returns the last error; the tail loop then retries on
// its next tick from the unchanged offset, so no lines are lost or duplicated.
// Sleep is injected so tests do not wait; nil sleeps for real.
type ReconnectingLogReader struct {
	Reader   LogReader
	Attempts int
	Backoff  time.Duration
	Sleep    func(context.Context, time.Duration) error
}

// ReadFrom retries the wrapped reader on error, backing off between attempts and
// honouring context cancellation.
func (r ReconnectingLogReader) ReadFrom(ctx context.Context, path string, offset int64) ([]byte, error) {
	attempts := r.Attempts
	if attempts < 1 {
		attempts = DefaultReconnectAttempts
	}
	backoff := r.Backoff
	if backoff <= 0 {
		backoff = DefaultReconnectBackoff
	}
	sleep := r.Sleep
	if sleep == nil {
		sleep = func(ctx context.Context, d time.Duration) error {
			t := time.NewTimer(d)
			defer t.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-t.C:
				return nil
			}
		}
	}
	var err error
	for i := 1; i <= attempts; i++ {
		var b []byte
		b, err = r.Reader.ReadFrom(ctx, path, offset)
		if err == nil {
			return b, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if i < attempts {
			if serr := sleep(ctx, backoff); serr != nil {
				return nil, serr
			}
		}
	}
	return nil, err
}
