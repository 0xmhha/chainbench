package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

type darwinProcessTable struct {
	status, args, executable string
	err                      error
	calls                    int
}

func (f *darwinProcessTable) Run(_ context.Context, command string) (string, error) {
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
func TestWebDarwinProcessObservation(t *testing.T) {
	ns := node.Record{PID: 1234, Args: []string{"--config", "/owned/node.toml"}}
	for _, tc := range []struct {
		name  string
		table darwinProcessTable
		state string
	}{
		{name: "actual launch columns", table: darwinProcessTable{status: "alive", args: "/owned/gwbft --config /owned/node.toml", executable: "/owned/gwbft"}, state: "running"},
		{name: "spoofed argv differs from executable", table: darwinProcessTable{status: "alive", args: "/owned/gwbft --config /owned/node.toml", executable: "/bin/sleep"}, state: "unknown"},
		{name: "missing process", table: darwinProcessTable{status: "absent"}, state: "missing"},
		{name: "unavailable process table", table: darwinProcessTable{status: "unavailable"}, state: "unknown"},
		{name: "SSH cannot read", table: darwinProcessTable{err: errors.New("private-inspection-error")}, state: "unknown"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := observeWebNodeProcess(context.Background(), webDarwinObserver{&tc.table}, "/owned/gwbft", ns)
			if got.State != tc.state {
				t.Fatal(got)
			}
			if strings.Contains(got.Reason, "private") {
				t.Fatal("raw probe failure disclosed")
			}
		})
	}
	table := &darwinProcessTable{}
	observer := webDarwinObserver{table}
	if _, err := observer.PIDAlive(context.Background(), 0); err == nil {
		t.Fatal("signal-group PID accepted")
	}
	if _, err := observer.Cmdline(context.Background(), -1); err == nil {
		t.Fatal("negative PID accepted")
	}
	if table.calls != 0 {
		t.Fatal("invalid PID reached the process table")
	}
}
