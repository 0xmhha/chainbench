package target_test

import (
	"strings"
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

// TestParseTarget_Syntaxes pins the whole single-path grammar in one table, so
// that adding a form cannot quietly change how an existing one parses.
func TestParseTarget_Syntaxes(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    target.TargetSpec
		wantErr bool
	}{
		{
			name: "bare path is local",
			in:   "/data/net1",
			want: target.TargetSpec{Kind: target.TargetLocal, DataRoot: "/data/net1"},
		},
		{
			name: "relative path is local",
			in:   "keys/preset",
			want: target.TargetSpec{Kind: target.TargetLocal, DataRoot: "keys/preset"},
		},
		{
			// The point of srv://: the address is not here.
			name: "inventory entry",
			in:   "srv://bp1/data/go-wbft/conf/nodekey",
			want: target.TargetSpec{
				Kind: target.TargetServer, Server: "bp1",
				DataRoot: "/data/go-wbft/conf/nodekey",
			},
		},
		{
			name: "host and path, no user",
			in:   "10.0.0.1:/keys/node1",
			want: target.TargetSpec{Kind: target.TargetRemote, Host: "10.0.0.1", DataRoot: "/keys/node1"},
		},
		{
			name: "user, host and path",
			in:   "ubuntu@host:/k",
			want: target.TargetSpec{Kind: target.TargetRemote, Host: "host", User: "ubuntu", DataRoot: "/k"},
		},
		{
			name: "ssh url with a port",
			in:   "ssh://ubuntu@host:2222/data/net1",
			want: target.TargetSpec{
				Kind: target.TargetRemote, Host: "host", User: "ubuntu",
				Port: 2222, DataRoot: "/data/net1",
			},
		},
		{
			// A colon in a local path must not be read as a host separator.
			name: "local path containing a colon",
			in:   "./notes:draft/key",
			want: target.TargetSpec{Kind: target.TargetLocal, DataRoot: "./notes:draft/key"},
		},
		{name: "empty", in: "", wantErr: true},
		{name: "srv with no path", in: "srv://bp1", wantErr: true},
		{name: "srv with no entry", in: "srv:///k", wantErr: true},
		{name: "user with no host", in: "user@:/k", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := target.ParseTarget(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("accepted %q as %+v", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseTarget(%q): %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("got %+v\nwant %+v", got, tc.want)
			}
		})
	}
}

// TestResolve_ServerNeedsAnInventory keeps an srv:// target from silently
// degrading into something else when no inventory was supplied.
func TestResolve_ServerNeedsAnInventory(t *testing.T) {
	spec, err := target.ParseTarget("srv://bp1/k")
	if err != nil {
		t.Fatalf("ParseTarget: %v", err)
	}
	_, err = spec.Resolve(func(string) string { return "" })
	if err == nil {
		t.Fatal("resolved an srv:// target with no inventory")
	}
	if !strings.Contains(err.Error(), "bp1") {
		t.Errorf("error should name the entry, got: %v", err)
	}
}
