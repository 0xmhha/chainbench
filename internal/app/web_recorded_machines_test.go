package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

func TestRecordedMachineOfANodeIsItsOwnServer(t *testing.T) {
	dir := t.TempDir()
	raw, err := json.Marshal([]webRecordedMachine{
		{Server: "server1", Target: resource.Inspection{HostIdentity: "machine-1"}},
		{Server: "server2", Target: resource.Inspection{HostIdentity: "machine-2"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "web-targets.json"), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	machines, err := readWebRecordedMachines(dir)
	if err != nil {
		t.Fatal(err)
	}
	first := resource.Inspection{HostIdentity: "machine-1"}
	if got := webMachineOf(machines, first, node.Record{Server: "server2"}); got.HostIdentity != "machine-2" {
		t.Fatalf("node on server2 reads %q", got.HostIdentity)
	}
	if got := webMachineOf(machines, first, node.Record{Server: "elsewhere"}); got.HostIdentity != "machine-1" {
		t.Fatalf("unrecorded server reads %q, want the network's target", got.HostIdentity)
	}
}

func TestRecordedMachinesOfASingleServerNetworkAreEmpty(t *testing.T) {
	machines, err := readWebRecordedMachines(t.TempDir())
	if err != nil || len(machines) != 0 {
		t.Fatalf("machines = %v, err = %v", machines, err)
	}
}
