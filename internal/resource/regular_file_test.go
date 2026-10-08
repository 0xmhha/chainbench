package resource

import (
	"context"
	"errors"
	"github.com/0xmhha/chainbench/internal/core/remote"
	"os"
	"path/filepath"
	"testing"
)

func TestRegularTargetFileRejectsSymlinkDirectoryAndCancellation(t *testing.T) {
	root := t.TempDir()
	regular := filepath.Join(root, "executable")
	if err := os.WriteFile(regular, []byte("owned executable bytes"), 0700); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(regular, alias); err != nil {
		t.Fatal(err)
	}
	access := &Access{}
	if err := access.VerifyRegularFile(context.Background(), regular); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{alias, root, filepath.Join(root, "missing")} {
		if err := access.VerifyRegularFile(context.Background(), path); err == nil {
			t.Fatal("nonregular target accepted", path)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := access.VerifyRegularFile(ctx, regular); !errors.Is(err, context.Canceled) {
		t.Fatal("cancelled file check continued", err)
	}
}

func TestRegularRemoteFileCheckPreservesShellQuotingAndFailure(t *testing.T) {
	path := "/owned/a' $(touch unapproved) file"
	expected := "/bin/sh -c " + remote.ShellQuote("test -f "+remote.ShellQuote(path)+" && test ! -L "+remote.ShellQuote(path))
	for _, exit := range []int{0, 1} {
		access := &Access{Runner: func(ctx context.Context, command string) (remote.ExecResult, error) {
			if command != expected {
				t.Fatal("path escaped into command", command)
			}
			return remote.ExecResult{ExitCode: exit}, nil
		}}
		err := access.VerifyRegularFile(context.Background(), path)
		if (err == nil) != (exit == 0) {
			t.Fatal("remote nonregular file accepted", err)
		}
	}
}
