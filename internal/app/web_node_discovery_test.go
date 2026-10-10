package app

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

type webDiscoveryFixture struct {
	pids, after []int
	argv        []string
	err         error
	calls       int
}

func (f *webDiscoveryFixture) FindBinary(context.Context, string) ([]int, error) {
	f.calls++
	if f.calls > 1 && f.after != nil {
		return f.after, f.err
	}
	return f.pids, f.err
}
func (f *webDiscoveryFixture) PIDAlive(context.Context, int) (bool, error) {
	return true, f.err
}
func (f *webDiscoveryFixture) Cmdline(context.Context, int) ([]string, error) {
	return f.argv, f.err
}

func TestWebVacancyRequiresReadableStableProcessDiscovery(t *testing.T) {
	ns := node.Record{Index: 1, DataDir: "/owned/node1", ConfigPath: "/owned/node1.toml", Args: []string{"--config", "/owned/node1.toml"}}
	for _, tc := range []struct {
		name, state string
		fixture     webDiscoveryFixture
	}{
		{name: "verified empty", state: "stopped"},
		{name: "unrecorded owned launch", state: "unrecorded_running", fixture: webDiscoveryFixture{pids: []int{101}, argv: []string{"/owned/gwbft", "--config", "/owned/node1.toml"}}},
		{name: "different node", state: "stopped", fixture: webDiscoveryFixture{pids: []int{101}, argv: []string{"/owned/gwbft", "--config", "/owned/node2.toml"}}},
		{name: "changed node argv", state: "ownership_mismatch", fixture: webDiscoveryFixture{pids: []int{101}, argv: []string{"/owned/gwbft", "--config=/owned/node1.toml", "--other"}}},
		{name: "another executable uses owned path", state: "ownership_mismatch", fixture: webDiscoveryFixture{pids: []int{101}, argv: []string{"/other/gwbft", "--config", "/owned/node1.toml"}}},
		{name: "table changed", state: "unknown", fixture: webDiscoveryFixture{after: []int{102}}},
		{name: "invalid PID", state: "unknown", fixture: webDiscoveryFixture{pids: []int{0}}},
		{name: "unreadable argv", state: "unknown", fixture: webDiscoveryFixture{pids: []int{101}}},
		{name: "failed SSH or table read", state: "unknown", fixture: webDiscoveryFixture{err: errors.New("private-failure")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := ns
			got := observeWebNodeVacancy(context.Background(), &tc.fixture, "/owned/gwbft", ns)
			if got.State != tc.state {
				t.Fatal(got)
			}
			if got.State == "unrecorded_running" && got.PID != 101 {
				t.Fatal("discovered PID missing")
			}
			if !reflect.DeepEqual(before, ns) {
				t.Fatal("discovery mutated the record")
			}
		})
	}
	if got := observeWebNodeVacancy(context.Background(), nil, "/owned/gwbft", ns); got.State != "unknown" {
		t.Fatal("absent observer established vacancy")
	}
}

type webProcessTableFixture struct{ output string }

func (f webProcessTableFixture) Run(context.Context, string) (string, error) { return f.output, nil }

func TestWebProcessTableRejectsUnavailableAndMalformedRows(t *testing.T) {
	for _, raw := range []string{"", "unavailable", "0 /owned/gwbft", "101", "pid /owned/gwbft"} {
		if _, err := webFindProcessCandidates(context.Background(), webProcessTableFixture{raw}, "gwbft"); err == nil {
			t.Fatal("unreadable process table established vacancy")
		}
	}
	pids, err := webFindProcessCandidates(context.Background(), webProcessTableFixture{"202 /other/gwbft\n101 /path with space/gwbft\n303 /bin/ps\n"}, "gwbft")
	if err != nil || !reflect.DeepEqual(pids, []int{101, 202}) {
		t.Fatal(pids, err)
	}
}
