package netcompose_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/provision"
	"github.com/0xmhha/chainbench/internal/netcompose"

	_ "github.com/0xmhha/chainbench/internal/chains/all" // register chain plugins
)

func fixedClock() func() time.Time { return func() time.Time { return time.Unix(0, 0).UTC() } }

func TestWorkspace_NewPersist(t *testing.T) {
	dir := t.TempDir()

	ws, err := netcompose.Open(dir, fixedClock())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := ws.New(netcompose.NewOpts{Chain: "stablenet", KeysDir: "keys/preset"}); err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := ws.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Reopen: state must round-trip. A default local target's data root is the
	// workspace dir; the keys pointer is recorded (account inspection is the
	// `account` subcommand's job, not net's).
	ws2, err := netcompose.Open(dir, fixedClock())
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	st := ws2.State()
	if st.Chain != "stablenet" || st.KeysDir != "keys/preset" {
		t.Fatalf("state did not persist: %+v", st)
	}
	if st.Target.Kind != netcompose.TargetLocal || st.Target.DataRoot != dir {
		t.Fatalf("local target default wrong: %+v", st.Target)
	}
	if !st.Steps["new"].Done {
		t.Fatalf("new step not recorded: %+v", st.Steps)
	}
	if _, err := os.Stat(filepath.Join(dir, "workspace.json")); err != nil {
		t.Fatalf("workspace.json not written: %v", err)
	}
}

func TestWorkspace_Validation(t *testing.T) {
	ws, err := netcompose.Open(t.TempDir(), fixedClock())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := ws.New(netcompose.NewOpts{Chain: "nope"}); err == nil {
		t.Fatal("expected error for unknown chain")
	}
	if _, err := netcompose.Open("", fixedClock()); err == nil {
		t.Fatal("expected error for empty data dir")
	}
	if _, err := ws.New(netcompose.NewOpts{Chain: "stablenet", Target: netcompose.TargetSpec{Kind: netcompose.TargetRemote}}); err == nil {
		t.Fatal("expected error for incomplete remote target")
	}
}

// TestTargetResolve checks the location abstraction: a local spec yields the
// local filesystem sink + driver; a remote spec yields SSH-backed ones, reading
// creds from env (no live dial).
func TestTargetResolve(t *testing.T) {
	local, err := netcompose.TargetSpec{Kind: netcompose.TargetLocal, DataRoot: "/tmp/x"}.Resolve(nil)
	if err != nil {
		t.Fatalf("local resolve: %v", err)
	}
	if _, ok := local.Sink.(provision.LocalFileSink); !ok {
		t.Fatalf("local sink type = %T", local.Sink)
	}
	if _, ok := local.Driver.(*driver.LocalDriver); !ok {
		t.Fatalf("local driver type = %T", local.Driver)
	}

	env := map[string]string{
		"CHAINBENCH_REMOTE_PASS":           "pw",
		"CHAINBENCH_SSH_INSECURE_HOST_KEY": "1",
	}
	remoteTgt, err := netcompose.TargetSpec{
		Kind: netcompose.TargetRemote, Host: "10.0.0.1", User: "ubuntu", DataRoot: "/tmp/net",
	}.Resolve(func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("remote resolve: %v", err)
	}
	if _, ok := remoteTgt.Sink.(driver.RemoteFileSink); !ok {
		t.Fatalf("remote sink type = %T", remoteTgt.Sink)
	}
	if _, ok := remoteTgt.Driver.(*driver.RemoteDriver); !ok {
		t.Fatalf("remote driver type = %T", remoteTgt.Driver)
	}

	if _, err := (netcompose.TargetSpec{Kind: netcompose.TargetRemote, Host: "h", User: "u", DataRoot: "/d"}).Resolve(func(string) string { return "" }); err == nil {
		t.Fatal("expected error for remote target without auth")
	}
}
