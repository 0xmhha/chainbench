package resource

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/remote"
)

// Exercise the remote snapshot's POSIX command without needing an SSH server;
// the actual SSH transport is covered by the native Web acceptance proof.
func TestRemoteSnapshotQuotesPathsAndReportsRotationAndAbsence(t *testing.T) {
	a := &Access{Runner: func(ctx context.Context, command string) (remote.ExecResult, error) {
		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		var out, stderr strings.Builder
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		err := cmd.Run()
		res := remote.ExecResult{Stdout: out.String(), Stderr: stderr.String()}
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			res.ExitCode = exit.ExitCode()
			err = nil
		}
		return res, err
	}}
	path := filepath.Join(t.TempDir(), "node's $HOME.log")
	if err := os.WriteFile(path, []byte("first bytes"), 0600); err != nil {
		t.Fatal(err)
	}
	r := LogRead{Path: path, Offset: 6, Limit: 3}
	s, err := readLogSnapshot(context.Background(), a, r)
	if err != nil || string(s.Data) != "byt" || string(s.Head) != "first bytes" || s.Size != 11 {
		t.Fatal(s, err)
	}
	if err = os.Rename(path, path+".old"); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, []byte("next"), 0600); err != nil {
		t.Fatal(err)
	}
	next, err := readLogSnapshot(context.Background(), a, r)
	if err != nil || next.ID == s.ID || string(next.Head) != "next" {
		t.Fatal("rotation not observed", next, err)
	}
	if err = os.Remove(path); err != nil {
		t.Fatal(err)
	}
	if _, err = readLogSnapshot(context.Background(), a, r); !errors.Is(err, ErrLogMissing) {
		t.Fatal("missing file not reported", err)
	}
}
