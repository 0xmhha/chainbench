package machine_test

import (
	"github.com/0xmhha/chainbench/internal/core/remote"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/machine"
)

// TestTargetResolve checks the location abstraction: a local spec yields the
// local filesystem sink + driver; a remote spec yields SSH-backed ones, reading
// creds from env (no live dial).
func TestTargetResolve(t *testing.T) {
	local, err := machine.Spec{DataRoot: "/tmp/x"}.Resolve(nil)
	if err != nil {
		t.Fatalf("local resolve: %v", err)
	}
	if _, ok := local.Files.(filestore.Local); !ok {
		t.Fatalf("local sink type = %T", local.Files)
	}
	if _, ok := local.Driver.(*driver.LocalDriver); !ok {
		t.Fatalf("local driver type = %T", local.Driver)
	}

	env := map[string]string{remote.EnvPass: "pw"}
	// The host-key policy is the caller's (the server set's) — resolving
	// never consults the environment for it.
	remoteTgt, err := machine.Spec{
		Host: "10.0.0.1", User: "ubuntu", DataRoot: "/tmp/net",
	}.ResolveWithPolicy(func(k string) string { return env[k] }, nil, nil,
		remote.HostKeyPolicy{InsecureHostKey: true})
	if err != nil {
		t.Fatalf("remote resolve: %v", err)
	}
	if _, ok := remoteTgt.Files.(driver.RemoteFileStore); !ok {
		t.Fatalf("remote file store type = %T", remoteTgt.Files)
	}
	if _, ok := remoteTgt.Driver.(*driver.RemoteDriver); !ok {
		t.Fatalf("remote driver type = %T", remoteTgt.Driver)
	}

	if _, err := (machine.Spec{Host: "h", User: "u", DataRoot: "/d"}).Resolve(func(string) string { return "" }); err == nil {
		t.Fatal("expected error for remote target without auth")
	}
}

func TestParse(t *testing.T) {
	cases := []struct {
		in   string
		want machine.Spec
	}{
		{"/data/net1", machine.Spec{DataRoot: "/data/net1"}},
		{"rel/dir", machine.Spec{DataRoot: "rel/dir"}},
		{"alice@10.0.0.5:/data/net1", machine.Spec{
			User: "alice", Host: "10.0.0.5", DataRoot: "/data/net1"}},
		{"ssh://bob@host9:2222/data/n", machine.Spec{
			User: "bob", Host: "host9", Port: 2222, DataRoot: "/data/n"}},
		{"ssh://carol@host9/data/n", machine.Spec{
			User: "carol", Host: "host9", DataRoot: "/data/n"}},
	}
	for _, tc := range cases {
		got, err := machine.Parse(tc.in)
		if err != nil {
			t.Errorf("%q: %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("%q = %+v, want %+v", tc.in, got, tc.want)
		}
	}

	for _, bad := range []string{"", "user@host", "user@:/path", "ssh://host-only"} {
		if _, err := machine.Parse(bad); err == nil {
			t.Errorf("%q must fail", bad)
		}
	}
}

// TestParse_Syntaxes pins the whole single-path grammar in one table, so
// that adding a form cannot quietly change how an existing one parses.
func TestParse_Syntaxes(t *testing.T) {
	cases := []struct {
		name    string
		in      string
		want    machine.Spec
		wantErr bool
	}{
		{
			name: "bare path is local",
			in:   "/data/net1",
			want: machine.Spec{DataRoot: "/data/net1"},
		},
		{
			name: "relative path is local",
			in:   "keys/preset",
			want: machine.Spec{DataRoot: "keys/preset"},
		},
		{
			// The point of srv://: the address is not here.
			name: "inventory entry",
			in:   "srv://bp1/data/go-wbft/conf/nodekey",
			want: machine.Spec{
				Server:   "bp1",
				DataRoot: "/data/go-wbft/conf/nodekey",
			},
		},
		{
			name: "host and path, no user",
			in:   "10.0.0.1:/keys/node1",
			want: machine.Spec{Host: "10.0.0.1", DataRoot: "/keys/node1"},
		},
		{
			name: "user, host and path",
			in:   "ubuntu@host:/k",
			want: machine.Spec{Host: "host", User: "ubuntu", DataRoot: "/k"},
		},
		{
			name: "ssh url with a port",
			in:   "ssh://ubuntu@host:2222/data/net1",
			want: machine.Spec{
				Host: "host", User: "ubuntu",
				Port: 2222, DataRoot: "/data/net1",
			},
		},
		{
			// A colon in a local path must not be read as a host separator.
			name: "local path containing a colon",
			in:   "./notes:draft/key",
			want: machine.Spec{DataRoot: "./notes:draft/key"},
		},
		{name: "empty", in: "", wantErr: true},
		{name: "srv with no path", in: "srv://bp1", wantErr: true},
		{name: "srv with no entry", in: "srv:///k", wantErr: true},
		{name: "user with no host", in: "user@:/k", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := machine.Parse(tc.in)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("accepted %q as %+v", tc.in, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("Parse(%q): %v", tc.in, err)
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
	spec, err := machine.Parse("srv://bp1/k")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, err = spec.Resolve(func(string) string { return "" })
	if err == nil {
		t.Fatal("resolved an srv:// target with no inventory")
	}
	if !strings.Contains(err.Error(), "bp1") {
		t.Errorf("error should name the entry, got: %v", err)
	}
}
