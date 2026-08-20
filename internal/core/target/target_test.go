package target_test

import (
	"testing"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/provision"
	"github.com/0xmhha/chainbench/internal/core/target"
)

// TestTargetResolve checks the location abstraction: a local spec yields the
// local filesystem sink + driver; a remote spec yields SSH-backed ones, reading
// creds from env (no live dial).
func TestTargetResolve(t *testing.T) {
	local, err := target.TargetSpec{Kind: target.TargetLocal, DataRoot: "/tmp/x"}.Resolve(nil)
	if err != nil {
		t.Fatalf("local resolve: %v", err)
	}
	if _, ok := local.Files.(provision.LocalFileStore); !ok {
		t.Fatalf("local sink type = %T", local.Files)
	}
	if _, ok := local.Driver.(*driver.LocalDriver); !ok {
		t.Fatalf("local driver type = %T", local.Driver)
	}

	env := map[string]string{
		"CHAINBENCH_REMOTE_PASS":           "pw",
		"CHAINBENCH_SSH_INSECURE_HOST_KEY": "1",
	}
	remoteTgt, err := target.TargetSpec{
		Kind: target.TargetRemote, Host: "10.0.0.1", User: "ubuntu", DataRoot: "/tmp/net",
	}.Resolve(func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("remote resolve: %v", err)
	}
	if _, ok := remoteTgt.Files.(driver.RemoteFileStore); !ok {
		t.Fatalf("remote file store type = %T", remoteTgt.Files)
	}
	if _, ok := remoteTgt.Driver.(*driver.RemoteDriver); !ok {
		t.Fatalf("remote driver type = %T", remoteTgt.Driver)
	}

	if _, err := (target.TargetSpec{Kind: target.TargetRemote, Host: "h", User: "u", DataRoot: "/d"}).Resolve(func(string) string { return "" }); err == nil {
		t.Fatal("expected error for remote target without auth")
	}
}

func TestParseTarget(t *testing.T) {
	cases := []struct {
		in   string
		want target.TargetSpec
	}{
		{"/data/net1", target.TargetSpec{Kind: target.TargetLocal, DataRoot: "/data/net1"}},
		{"rel/dir", target.TargetSpec{Kind: target.TargetLocal, DataRoot: "rel/dir"}},
		{"alice@10.0.0.5:/data/net1", target.TargetSpec{
			Kind: target.TargetRemote, User: "alice", Host: "10.0.0.5", DataRoot: "/data/net1"}},
		{"ssh://bob@host9:2222/data/n", target.TargetSpec{
			Kind: target.TargetRemote, User: "bob", Host: "host9", Port: 2222, DataRoot: "/data/n"}},
		{"ssh://carol@host9/data/n", target.TargetSpec{
			Kind: target.TargetRemote, User: "carol", Host: "host9", DataRoot: "/data/n"}},
	}
	for _, tc := range cases {
		got, err := target.ParseTarget(tc.in)
		if err != nil {
			t.Errorf("%q: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%q = %+v, want %+v", tc.in, got, tc.want)
		}
	}

	for _, bad := range []string{"", "user@host", "user@:/path", "ssh://host-only"} {
		if _, err := target.ParseTarget(bad); err == nil {
			t.Errorf("%q must fail", bad)
		}
	}
}
