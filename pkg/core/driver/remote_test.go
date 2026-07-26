package driver_test

import (
	"context"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/pkg/core/driver"
	"github.com/0xmhha/chainbench/pkg/core/node"
	"github.com/0xmhha/chainbench/pkg/core/remote"
)

// fakeRunner records the commands it is asked to run and returns canned output.
type fakeRunner struct {
	cmds []string
	out  map[string]remote.ExecResult // substring -> result
}

func (f *fakeRunner) run(_ context.Context, command string) (remote.ExecResult, error) {
	f.cmds = append(f.cmds, command)
	for sub, res := range f.out {
		if strings.Contains(command, sub) {
			return res, nil
		}
	}
	return remote.ExecResult{ExitCode: 0}, nil
}

func (f *fakeRunner) last() string { return f.cmds[len(f.cmds)-1] }

func TestRemoteDriver_ProvisionLaunchStop(t *testing.T) {
	f := &fakeRunner{out: map[string]remote.ExecResult{
		"echo $!": {Stdout: "12345\n", ExitCode: 0}, // launch returns a pid
	}}
	d := driver.NewRemoteDriver(f.run)
	ctx := context.Background()

	spec := driver.NodeSpec{
		Index: 1, Binary: "/opt/gwbft", DataDir: "/data/node1",
		ConfigPath: "/data/node1/config.toml", ConfigContent: []byte("[Node]\n"),
		LogPath: "/data/logs/node1.log",
		Args:    []string{"--datadir", "/data/node1", "--mine"},
	}

	// Provision: mkdir datadir + write config (base64-piped).
	if err := d.Provision(ctx, spec); err != nil {
		t.Fatal(err)
	}
	all := strings.Join(f.cmds, "\n")
	if !strings.Contains(all, "mkdir -p '/data/node1'") || !strings.Contains(all, "base64 -d > '/data/node1/config.toml'") {
		t.Errorf("provision commands missing:\n%s", all)
	}

	// Launch: nohup in the background, quoted args, returns the pid.
	h, err := d.Launch(ctx, spec)
	if err != nil {
		t.Fatal(err)
	}
	if h.PID != 12345 || h.Index != 1 {
		t.Errorf("launch handle = %+v, want pid 12345 index 1", h)
	}
	launch := f.last()
	for _, want := range []string{"nohup '/opt/gwbft'", "'--datadir' '/data/node1' '--mine'", "> '/data/logs/node1.log' 2>&1 &", "echo $!"} {
		if !strings.Contains(launch, want) {
			t.Errorf("launch command missing %q:\n%s", want, launch)
		}
	}

	// Stop: kill the pid, tolerating an already-gone process.
	if err := d.Stop(ctx, h); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.last(), "kill 12345") {
		t.Errorf("stop command = %s", f.last())
	}
}

func TestRemoteDriver_LaunchNoPID(t *testing.T) {
	f := &fakeRunner{out: map[string]remote.ExecResult{"echo $!": {Stdout: "not-a-pid\n"}}}
	d := driver.NewRemoteDriver(f.run)
	if _, err := d.Launch(context.Background(), driver.NodeSpec{Index: 1, LogPath: "/l/n.log", Ports: node.Endpoints{}}); err == nil {
		t.Error("launch with no pid should error")
	}
}

func TestRemoteDriver_NonZeroExitIsError(t *testing.T) {
	f := &fakeRunner{out: map[string]remote.ExecResult{"mkdir": {ExitCode: 1, Stderr: "permission denied"}}}
	d := driver.NewRemoteDriver(f.run)
	err := d.Provision(context.Background(), driver.NodeSpec{DataDir: "/data"})
	if err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("non-zero exit should surface stderr: %v", err)
	}
}
