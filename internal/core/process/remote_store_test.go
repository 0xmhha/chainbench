package process_test

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/remote"
)

// RemoteFileStore must satisfy the filestore.Store boundary a launcher plugs into.
var _ filestore.Store = process.RemoteFileStore{}

func TestRemoteFileStore_Exists(t *testing.T) {
	cases := []struct {
		name    string
		exit    int
		runErr  error
		want    bool
		wantErr bool
	}{
		{"present", 0, nil, true, false},
		{"absent", 1, nil, false, false},
		{"transport error", 0, errors.New("dial failed"), false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotCmd string
			run := func(_ context.Context, cmd string) (remote.ExecResult, error) {
				gotCmd = cmd
				return remote.ExecResult{ExitCode: tc.exit}, tc.runErr
			}
			ok, err := process.NewRemoteFileStore(run).Exists(context.Background(), "/data/genesis.json")
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Exists: %v", err)
			}
			if ok != tc.want {
				t.Fatalf("exists = %v, want %v", ok, tc.want)
			}
			// test -e, not test -f: a datadir is a directory and must count as
			// present, the way the local store's os.Stat treats it.
			if !strings.Contains(gotCmd, "test -e") || !strings.Contains(gotCmd, "genesis.json") {
				t.Fatalf("unexpected probe command: %q", gotCmd)
			}
		})
	}
}

func TestRemoteFileStore_Write(t *testing.T) {
	var cmds []string
	run := func(_ context.Context, cmd string) (remote.ExecResult, error) {
		cmds = append(cmds, cmd)
		return remote.ExecResult{ExitCode: 0}, nil
	}
	if err := process.NewRemoteFileStore(run).Write(context.Background(), "/data/config.toml", []byte("hello"), 0o644); err != nil {
		t.Fatalf("Write: %v", err)
	}
	joined := strings.Join(cmds, "\n")
	// The write is base64-piped, then chmod'd, under a created parent dir.
	enc := base64.StdEncoding.EncodeToString([]byte("hello"))
	if !strings.Contains(joined, enc) {
		t.Fatalf("content not base64-shipped: %q", joined)
	}
	if !strings.Contains(joined, "mkdir -p") || !strings.Contains(joined, "chmod") {
		t.Fatalf("write command missing mkdir/chmod: %q", joined)
	}
}

func TestRemoteFileStore_Checksum(t *testing.T) {
	hex := strings.Repeat("ab", 32) // 64 hex chars
	cases := []struct {
		name    string
		stdout  string
		exit    int
		runErr  error
		want    string
		wantErr bool
	}{
		{"sha256sum output", hex + "  /data/genesis.json\n", 0, nil, "sha256:" + hex, false},
		{"transport error", "", 0, errors.New("dial failed"), "", true},
		{"nonzero exit", "sha256sum: /data/x: No such file", 1, nil, "", true},
		{"garbage output", "not-a-hash\n", 0, nil, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotCmd string
			run := func(_ context.Context, cmd string) (remote.ExecResult, error) {
				gotCmd = cmd
				return remote.ExecResult{ExitCode: tc.exit, Stdout: tc.stdout}, tc.runErr
			}
			got, err := process.NewRemoteFileStore(run).Checksum(context.Background(), "/data/genesis.json")
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Checksum: %v", err)
			}
			if got != tc.want {
				t.Fatalf("Checksum = %q, want %q", got, tc.want)
			}
			if !strings.Contains(gotCmd, "sha256sum") {
				t.Errorf("command %q should run sha256sum", gotCmd)
			}
		})
	}
}
