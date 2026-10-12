package process

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// launchTable answers a fixed sequence of liveness reads and one argv.
type launchTable struct {
	alive  []bool
	argv   []string
	err    error
	probes int
}

func (f *launchTable) PIDAlive(context.Context, int) (bool, error) {
	i := f.probes
	f.probes++
	if f.err != nil {
		return false, f.err
	}
	if i >= len(f.alive) {
		return false, nil
	}
	return f.alive[i], nil
}
func (f *launchTable) Cmdline(context.Context, int) ([]string, error)    { return f.argv, f.err }
func (f *launchTable) FindBinary(context.Context, string) ([]int, error) { return nil, f.err }

// TestObserveLaunchVerifiesTheRecordedProcess: only a live process whose argv
// is exactly the recorded launch is the record's; a reused pid, a different
// config, an exit during the read or an unreadable table never is.
func TestObserveLaunchVerifiesTheRecordedProcess(t *testing.T) {
	args := []string{"--config", "/owned/node1.toml"}
	for _, tc := range []struct {
		name          string
		table         launchTable
		pid           int
		state, reason string
	}{
		{name: "exact launch", table: launchTable{alive: []bool{true, true}, argv: []string{"/owned/gwbft", "--config", "/owned/node1.toml"}}, pid: 1234, state: "running"},
		{name: "missing recorded process", table: launchTable{alive: []bool{false}}, pid: 1234, state: "missing", reason: "recorded_pid_absent"},
		{name: "pid reused for a different binary", table: launchTable{alive: []bool{true}, argv: []string{"/bin/sleep", "30"}}, pid: 1234, state: "ownership_mismatch", reason: "argv_mismatch"},
		{name: "same binary, another config", table: launchTable{alive: []bool{true}, argv: []string{"/owned/gwbft", "--config", "/other/node1.toml"}}, pid: 1234, state: "ownership_mismatch", reason: "argv_mismatch"},
		{name: "same binary, incomplete args", table: launchTable{alive: []bool{true}, argv: []string{"/owned/gwbft"}}, pid: 1234, state: "ownership_mismatch", reason: "argv_mismatch"},
		{name: "process exits during the read", table: launchTable{alive: []bool{true, false}, argv: []string{"/owned/gwbft", "--config", "/owned/node1.toml"}}, pid: 1234, state: "missing", reason: "recorded_pid_absent"},
		{name: "unavailable probe", table: launchTable{err: errors.New("private-probe-error")}, pid: 1234, state: "unknown", reason: "probe_unavailable"},
		{name: "no recorded pid", pid: 0, state: "unknown", reason: "no_recorded_pid"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ObserveLaunch(context.Background(), &tc.table, tc.pid, "/owned/gwbft", args)
			if string(got.State) != tc.state || got.Reason != tc.reason {
				t.Fatal(got)
			}
			if got.State == OwnedRunning && got.PID != tc.pid {
				t.Fatal("lost the live pid")
			}
			if got.State != OwnedRunning && got.PID != 0 {
				t.Fatal("an unverified pid was presented as live")
			}
		})
	}
}

// commandTable answers ps the way a darwin machine would.
type commandTable struct {
	status, args, executable string
	err                      error
	calls                    int
}

func (f *commandTable) Run(_ context.Context, command string) (string, error) {
	f.calls++
	if f.err != nil {
		return "", f.err
	}
	if strings.Contains(command, "-o args=") {
		return f.args, nil
	}
	if strings.Contains(command, "-o comm=") {
		return f.executable, nil
	}
	return f.status, nil
}

// TestDarwinTableReadsLaunchesWithPS: argv whose first word is not the
// executable column cannot be trusted, and an invalid pid never reaches ps.
func TestDarwinTableReadsLaunchesWithPS(t *testing.T) {
	args := []string{"--config", "/owned/node.toml"}
	for _, tc := range []struct {
		name  string
		table commandTable
		state string
	}{
		{name: "actual launch columns", table: commandTable{status: "alive", args: "/owned/gwbft --config /owned/node.toml", executable: "/owned/gwbft"}, state: "running"},
		{name: "argv disagrees with the executable", table: commandTable{status: "alive", args: "/owned/gwbft --config /owned/node.toml", executable: "/bin/sleep"}, state: "unknown"},
		{name: "missing process", table: commandTable{status: "absent"}, state: "missing"},
		{name: "unavailable process table", table: commandTable{status: "unavailable"}, state: "unknown"},
		{name: "ssh cannot read", table: commandTable{err: errors.New("private-inspection-error")}, state: "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ObserveLaunch(context.Background(), darwinObserver{&tc.table}, 1234, "/owned/gwbft", args)
			if string(got.State) != tc.state || strings.Contains(got.Reason, "private") {
				t.Fatal(got)
			}
		})
	}
	table := &commandTable{}
	o := darwinObserver{table}
	if _, err := o.PIDAlive(context.Background(), 0); err == nil {
		t.Fatal("a signal-group pid was accepted")
	}
	if _, err := o.Cmdline(context.Background(), -1); err == nil {
		t.Fatal("a negative pid was accepted")
	}
	if table.calls != 0 {
		t.Fatal("an invalid pid reached the process table")
	}
}

// discoveryTable lists candidates, optionally a different list the second time.
type discoveryTable struct {
	pids, after []int
	argv        []string
	err         error
	calls       int
}

func (f *discoveryTable) FindBinary(context.Context, string) ([]int, error) {
	f.calls++
	if f.calls > 1 && f.after != nil {
		return f.after, f.err
	}
	return f.pids, f.err
}
func (f *discoveryTable) PIDAlive(context.Context, int) (bool, error)    { return true, f.err }
func (f *discoveryTable) Cmdline(context.Context, int) ([]string, error) { return f.argv, f.err }

// TestObserveVacancyRequiresAReadableStableTable: a node with no pid is
// stopped only when the table was read twice and agreed; its exact launch is
// found if the record lost it, and another launch using its paths is a
// mismatch.
func TestObserveVacancyRequiresAReadableStableTable(t *testing.T) {
	args := []string{"--config", "/owned/node1.toml"}
	for _, tc := range []struct {
		name, state string
		table       discoveryTable
	}{
		{name: "verified empty", state: "stopped"},
		{name: "unrecorded owned launch", state: "unrecorded_running", table: discoveryTable{pids: []int{101}, argv: []string{"/owned/gwbft", "--config", "/owned/node1.toml"}}},
		{name: "a different node", state: "stopped", table: discoveryTable{pids: []int{101}, argv: []string{"/owned/gwbft", "--config", "/owned/node2.toml"}}},
		{name: "changed node argv", state: "ownership_mismatch", table: discoveryTable{pids: []int{101}, argv: []string{"/owned/gwbft", "--config=/owned/node1.toml", "--other"}}},
		{name: "another executable uses the owned path", state: "ownership_mismatch", table: discoveryTable{pids: []int{101}, argv: []string{"/other/gwbft", "--config", "/owned/node1.toml"}}},
		{name: "table changed", state: "unknown", table: discoveryTable{after: []int{102}}},
		{name: "invalid pid", state: "unknown", table: discoveryTable{pids: []int{0}}},
		{name: "unreadable argv", state: "unknown", table: discoveryTable{pids: []int{101}}},
		{name: "failed read", state: "unknown", table: discoveryTable{err: errors.New("private-failure")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := ObserveVacancy(context.Background(), &tc.table, "/owned/gwbft", args, "/owned/node1", "/owned/node1.toml")
			if string(got.State) != tc.state {
				t.Fatal(got)
			}
			if got.State == OwnedUnrecorded && got.PID != 101 {
				t.Fatal("the discovered pid is missing")
			}
		})
	}
	if got := ObserveVacancy(context.Background(), nil, "/owned/gwbft", args, "", ""); got.State != OwnedUnknown {
		t.Fatal("no table established a vacancy")
	}
}

// fixedCommand answers every command with one output.
type fixedCommand struct{ output string }

func (f fixedCommand) Run(context.Context, string) (string, error) { return f.output, nil }

// TestDarwinTableRefusesUnreadableRows: a process table that cannot be read
// row by row establishes nothing; a readable one lists matching names, paths
// with spaces included.
func TestDarwinTableRefusesUnreadableRows(t *testing.T) {
	for _, raw := range []string{"", "unavailable", "0 /owned/gwbft", "101", "pid /owned/gwbft"} {
		if _, err := (darwinObserver{fixedCommand{raw}}).FindBinary(context.Background(), "gwbft"); err == nil {
			t.Fatalf("%q established a candidate list", raw)
		}
	}
	pids, err := (darwinObserver{fixedCommand{"202 /other/gwbft\n101 /path with space/gwbft\n303 /bin/ps\n"}}).FindBinary(context.Background(), "gwbft")
	if err != nil || !reflect.DeepEqual(pids, []int{101, 202}) || !slices.IsSorted(pids) {
		t.Fatal(pids, err)
	}
}
