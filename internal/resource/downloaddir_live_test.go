package resource_test

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/resource"
	"github.com/0xmhha/chainbench/internal/testsupport"
)

// TestLive_DownloadDirRecoversARootOwnedTree verifies recursive directory
// download against the docker servers, elevating through sudo. Set up first:
//
//	docker exec -u root chainbench-server1 sh -c \
//	  'mkdir -p /root/cb-kr-test/node1 && echo pw > /root/cb-kr-test/password && \
//	   echo meta > /root/cb-kr-test/metadata.json && echo k1 > /root/cb-kr-test/node1/keystore.json && \
//	   chmod -R 600 /root/cb-kr-test && find /root/cb-kr-test -type d -exec chmod 700 {} \;'
//	CHAINBENCH_DOCKER_SERVERS=$PWD/env/docker/build go test ./internal/resource -run TestLive_DownloadDir -v
func TestLive_DownloadDirRecoversARootOwnedTree(t *testing.T) {
	build := testsupport.ServersBuildDir(t)
	acc, err := resource.Opener{
		ServerSet: filepath.Join(build, "server-set.yaml"), Docker: true, Env: os.Getenv,
	}.Open(resource.Spec{Server: "server1", DataRoot: "/data/chainbench"})
	if err != nil {
		t.Fatalf("open server1: %v", err)
	}
	if acc.ElevatedRunner == nil {
		t.Fatal("server permits sudo but ElevatedRunner was not wired")
	}

	const remoteDir = "/root/cb-kr-test"
	local := t.TempDir()
	if err := acc.DownloadDir(context.Background(), remoteDir, local); err != nil {
		t.Fatalf("DownloadDir: %v", err)
	}

	var got []string
	_ = filepath.Walk(local, func(p string, info os.FileInfo, _ error) error {
		if info != nil && !info.IsDir() {
			got = append(got, strings.TrimPrefix(p, local+"/"))
		}
		return nil
	})
	sort.Strings(got)
	want := []string{"metadata.json", "node1/keystore.json", "password"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("downloaded tree = %v, want %v", got, want)
	}
	// A file's content and 0600 mode survive the trip.
	b, err := os.ReadFile(filepath.Join(local, "node1/keystore.json"))
	if err != nil || strings.TrimSpace(string(b)) != "k1" {
		t.Fatalf("node1/keystore.json = %q err=%v", b, err)
	}
	if info, _ := os.Stat(filepath.Join(local, "password")); info.Mode().Perm() != 0o600 {
		t.Fatalf("local perm = %#o, want 0600", info.Mode().Perm())
	}
}
