package serverset_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/netmap/internal/serverset"
)

// TestLive_SudoElevatesWithThePassword proves the fleet's real access model
// end to end: the login is an ordinary user authenticated by password, and
// sudo elevates by asking for that same password — which SSHSudoRunner feeds
// on stdin. Gated on the docker fleet:
//
//	CHAINBENCH_DOCKER_FLEET=<repo>/env/docker/build go test ./internal/serverset/ -run Live_Sudo -v
func TestLive_SudoElevatesWithThePassword(t *testing.T) {
	build := os.Getenv("CHAINBENCH_DOCKER_FLEET")
	if build == "" {
		t.Skip("set CHAINBENCH_DOCKER_FLEET=<repo>/env/docker/build with the fleet running")
	}

	inv := filepath.Join(build, "server-set.yaml")
	cfg, err := serverset.Load(inv)
	if err != nil {
		t.Fatal(err)
	}
	srv, err := cfg.ByName("server1")
	if err != nil {
		t.Fatal(err)
	}
	if !srv.SSH.Sudo {
		t.Fatalf("the fleet inventory should declare sudo: true")
	}
	creds, err := srv.Credentials()
	if err != nil {
		t.Fatal(err)
	}
	lm, err := serverset.LoadLocalMap(serverset.LocalMapNear(inv))
	if err != nil {
		t.Fatal(err)
	}
	creds.Host, creds.Port = lm.AddrMap(nil)(creds.Host, creds.Port)

	hostKey, err := creds.HostKey.Callback()
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	// The plain runner is the user; the sudo runner is root. Asserting both
	// proves elevation happened rather than the login secretly being root.
	plain, err := driver.SSHRunner(creds, hostKey)(ctx, "whoami")
	if err != nil {
		t.Fatalf("password login failed: %v", err)
	}
	if got := strings.TrimSpace(plain.Stdout); got != creds.User {
		t.Fatalf("login user = %q, want %q", got, creds.User)
	}

	elevated, err := driver.SSHSudoRunner(creds, hostKey)(ctx, "whoami")
	if err != nil {
		t.Fatalf("sudo exec failed: %v", err)
	}
	if got := strings.TrimSpace(elevated.Stdout); got != "root" {
		t.Fatalf("sudo whoami = %q (stderr %q), want root", got, elevated.Stderr)
	}

	// A root-only WRITE, not just an identity print: touch a file in a
	// root-owned directory, verify, and clean up — the shape a privileged
	// bring-up step will actually have.
	if _, err := driver.SSHSudoRunner(creds, hostKey)(ctx,
		"touch /etc/chainbench-sudo-probe && rm /etc/chainbench-sudo-probe"); err != nil {
		t.Fatalf("root-only write failed: %v", err)
	}
}
