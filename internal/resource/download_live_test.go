package resource_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/resource"
	"github.com/0xmhha/chainbench/internal/testsupport"
)

// TestLive_ElevatedDownloadReadsARootOwnedFile verifies the sudo download path
// against the docker servers: the login user (devuser1) cannot read a
// root-owned 0600 file, but chainbench's elevated read can, because the server
// set permits sudo. Set up before running:
//
//	docker exec -u root chainbench-server1 sh -c \
//	  'echo root-only-test-content-12345 > /root/cb-sudo-test.txt && chmod 600 /root/cb-sudo-test.txt'
//	CHAINBENCH_DOCKER_SERVERS=$PWD/env/docker/build go test ./internal/resource -run TestLive_ElevatedDownload -v
func TestLive_ElevatedDownloadReadsARootOwnedFile(t *testing.T) {
	build := testsupport.ServersBuildDir(t)
	acc, err := resource.Opener{
		ServerSet: filepath.Join(build, "server-set.yaml"), Docker: true, Env: os.Getenv,
	}.Open(resource.Spec{Server: "server1", DataRoot: "/data/chainbench"})
	if err != nil {
		t.Fatalf("open server1: %v", err)
	}
	if acc.ElevatedFiles == nil {
		t.Fatal("server permits sudo but ElevatedFiles was not wired")
	}

	const remotePath = "/root/cb-sudo-test.txt"
	const want = "root-only-test-content-12345\n"

	// The ordinary login cannot read it.
	if _, err := acc.Files.Read(context.Background(), remotePath); err == nil {
		t.Skip("the login user could read the root-owned file — test fixture not set up as root-only")
	}

	// Elevated read (and download) can.
	local := filepath.Join(t.TempDir(), "downloaded")
	if err := acc.DownloadTo(context.Background(), remotePath, local); err != nil {
		t.Fatalf("elevated download: %v", err)
	}
	b, err := os.ReadFile(local)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != want {
		t.Fatalf("downloaded content = %q, want %q", b, want)
	}
	if info, _ := os.Stat(local); info.Mode().Perm() != 0o600 {
		t.Fatalf("local perm = %#o, want 0600", info.Mode().Perm())
	}
}
