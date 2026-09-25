package chainsetup_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/resource"
)

// spaceDriver is the stub driver on a machine with a fixed amount of free space.
type spaceDriver struct {
	stubDriver
	free uint64
}

func (d *spaceDriver) FreeBytes(context.Context, string) (uint64, error) { return d.free, nil }

// placedWorkspace is a two-node composition placed on a local data root whose
// workspace-config sets limits.minFreeDisk to floor.
func placedWorkspace(t *testing.T, floor string, free uint64) *chainsetup.Workspace {
	t.Helper()
	dir, dataRoot := t.TempDir(), t.TempDir()
	wcPath := filepath.Join(dir, "workspace-config.yaml")
	limits := ""
	if floor != "" {
		limits = "limits: {minFreeDisk: " + floor + "}\n"
	}
	must(t, os.WriteFile(wcPath, []byte(`version: 1
dataRoot: `+dataRoot+`
paths: {binaries: bin, configs: configs, genesis: genesis, keystore: keystore, keyrings: keys, nodes: node, runtime: runtime, logs: logs}
control: {artifactRoot: ~/.chainbench}
inputs: {mode: generated}
execution: {chain: fresh}
`+limits), 0o644))
	ws, err := chainsetup.Open(dir, fixedClock())
	must(t, err)
	d := &spaceDriver{free: free}
	ws.SetDriver(func() (process.Driver, error) { return d, nil })
	_, err = ws.New(chainsetup.NewOpts{
		Chain: "stablenet", KeysDir: filepath.Join("..", "..", "presets", "keys"),
		Target: resource.Spec{DataRoot: dataRoot}, WorkspaceConfigPath: wcPath,
	})
	must(t, err)
	_, err = ws.Allocate(chainsetup.AllocateOpts{BPCount: 2})
	must(t, err)
	return ws
}

// TestCheckFreeSpace_RefusesAShortMachine: a data root with less free space
// than the floor is refused by name before anything is written — not found out
// by the first write that does not fit, halfway through a launch.
func TestCheckFreeSpace_RefusesAShortMachine(t *testing.T) {
	ws := placedWorkspace(t, "10GiB", 1<<30)
	_, err := ws.CheckFreeSpace(context.Background())
	if err == nil {
		t.Fatal("1GiB free under a 10GiB floor was accepted")
	}
	for _, want := range []string{"less than 10.0GiB free", "1.0GiB free"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not say %q: %v", want, err)
		}
	}
}

// TestCheckFreeSpace_TheFloorAndItsSwitch: the default floor is 2GiB, and "0"
// turns the check off.
func TestCheckFreeSpace_TheFloorAndItsSwitch(t *testing.T) {
	for _, c := range []struct {
		floor string
		free  uint64
		ok    bool
	}{
		{"", 1 << 30, false}, // default 2GiB, 1GiB free
		{"", 3 << 30, true},  // default 2GiB, 3GiB free
		{"0", 1 << 20, true}, // check off
		{"500MiB", 1 << 30, true},
	} {
		ws := placedWorkspace(t, c.floor, c.free)
		_, err := ws.CheckFreeSpace(context.Background())
		if (err == nil) != c.ok {
			t.Errorf("floor %q with %d bytes free: err = %v, want ok=%v", c.floor, c.free, err, c.ok)
		}
	}
}
