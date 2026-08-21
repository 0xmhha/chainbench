package serverset_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/serverset"
)

// writeInventory writes an inventory file and returns its path.
func writeInventory(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "remote-server-config.yaml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestLoad_V2Pool reads the pooled schema end to end: hosts, bases, the
// file-wide credentials with sudo, and the server-side data root.
func TestLoad_V2Pool(t *testing.T) {
	cfg, err := serverset.Load(writeInventory(t, `
version: 2
pool:
  hosts: [10.0.0.1, 10.0.0.2]
  portBases: [8080, 8180]
ssh:
  user: ops
  port: 2222
  sudo: true
dataRoot: /data/chainbench
`))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	pool, err := cfg.NodePool()
	if err != nil {
		t.Fatalf("NodePool: %v", err)
	}
	if len(pool.Hosts) != 2 || len(pool.PortBases) != 2 || pool.DataRoot != "/data/chainbench" {
		t.Errorf("pool = %+v", pool)
	}
	ssh := cfg.PoolSSH()
	if ssh.User != "ops" || ssh.Port != 2222 || !ssh.Sudo {
		t.Errorf("ssh = %+v", ssh)
	}
}

// TestLoad_V2RejectsWhatCannotAssign: the pool's consistency rules run at load
// time, so a broken inventory fails when it is read, not when a node dies.
func TestLoad_V2RejectsWhatCannotAssign(t *testing.T) {
	cases := []struct {
		name, body, want string
	}{
		{
			"bases closer than a node's span",
			"version: 2\npool:\n  hosts: [a]\n  portBases: [8080, 8081]\n",
			"spans",
		},
		{
			"pool block missing",
			"version: 2\ndataRoot: /d\n",
			"no pool block",
		},
		{
			"mixing v1 servers with a v2 pool",
			"version: 2\npool:\n  hosts: [a]\n  portBases: [8080]\nservers:\n  - index: 1\n    host: b\n",
			"pick one",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := serverset.Load(writeInventory(t, tc.body))
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Errorf("Load = %v, want mention of %q", err, tc.want)
			}
		})
	}
}

// TestNodePool_V1SaysHowToMigrate: a v1 file keeps working for the older
// placement path, and asking it for a pool names the migration instead of
// inventing an empty one.
func TestNodePool_V1SaysHowToMigrate(t *testing.T) {
	cfg, err := serverset.Load(writeInventory(t,
		"version: 1\ndefaults:\n  ssh: {user: u, password: p}\nservers:\n  - index: 1\n    kind: remote\n    host: 10.0.0.1\n"))
	if err != nil {
		t.Fatalf("a v1 inventory stopped loading: %v", err)
	}
	_, err = cfg.NodePool()
	if err == nil || !strings.Contains(err.Error(), "version: 2") {
		t.Errorf("NodePool on v1 = %v, want a migration hint", err)
	}
}
