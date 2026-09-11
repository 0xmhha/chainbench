package app

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/resource"
)

// These two functions decide WHERE A FILE LANDS ON SOMEBODY ELSE'S MACHINE, and
// they had no coverage at all -- picked out of internal/app's 118 uncovered
// functions by counting branches and refusals per function, which is what the
// dangerous ones have.
//
// Each is a decision table. The value of testing them is the refusals: an upload
// that resolves to the wrong path does not fail, it writes.

func testWC() resource.WorkspaceConfig {
	return resource.WorkspaceConfig{
		Version:  1,
		DataRoot: "/data/chainbench",
		Paths: resource.WorkspacePaths{
			Binaries: "bin", Genesis: "genesis", Configs: "configs",
			Keystore: "keystore", Keyrings: "keys", Nodes: "node",
			Runtime: "runtime", Logs: "logs",
		},
	}
}

func TestUploadDests_DecisionTable(t *testing.T) {
	wc := testWC()
	cases := []struct {
		name  string
		in    UploadIn
		want  []string
		errIs string
	}{
		{
			name:  "both ways of naming a destination",
			in:    UploadIn{Purpose: "bin", RemotePath: "/tmp/x", LocalPaths: []string{"a"}},
			errIs: "not both",
		},
		{
			name: "an explicit remote path takes one file",
			in:   UploadIn{RemotePath: "/data/chainbench/bin/gwemix", LocalPaths: []string{"/build/gwemix"}},
			want: []string{"/data/chainbench/bin/gwemix"},
		},
		{
			// Two files and one destination would silently overwrite: the second
			// upload lands on the first. It is refused, and the message says how
			// many were given.
			name:  "an explicit remote path refuses several files",
			in:    UploadIn{RemotePath: "/data/x", LocalPaths: []string{"a", "b"}},
			errIs: "exactly one local file, got 2",
		},
		{
			name:  "an unknown purpose lists the known ones",
			in:    UploadIn{Purpose: "binaries-ish", LocalPaths: []string{"a"}},
			errIs: "want bin, genesis, configs, keystore, or keyrings",
		},
		{
			name: "a purpose places each file under its folder",
			in:   UploadIn{Purpose: "bin", LocalPaths: []string{"/build/gwemix", "/build/gwbft"}},
			want: []string{"/data/chainbench/bin/gwemix", "/data/chainbench/bin/gwbft"},
		},
		{
			name: "the purpose alias resolves to the same folder",
			in:   UploadIn{Purpose: "binaries", LocalPaths: []string{"/build/gwemix"}},
			want: []string{"/data/chainbench/bin/gwemix"},
		},
		{
			name:  "no destination at all",
			in:    UploadIn{LocalPaths: []string{"a"}},
			errIs: "a destination is required",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := uploadDests(wc, c.in)
			if c.errIs != "" {
				if err == nil {
					t.Fatalf("wanted a refusal, got %v", got)
				}
				if !strings.Contains(err.Error(), c.errIs) {
					t.Errorf("refusal %q does not say %q", err, c.errIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected refusal: %v", err)
			}
			if len(got) != len(c.want) {
				t.Fatalf("got %v, want %v", got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Errorf("dest %d = %q, want %q", i, got[i], c.want[i])
				}
			}
		})
	}
}

// TestUploadDests_TakesTheBaseNameOnly is the property behind the purpose form.
// A destination is <dataRoot>/<purpose>/<base name>: the local directory
// structure is not recreated on the server, and a local path that walks upward
// cannot walk upward over there either, because only the last element survives.
func TestUploadDests_TakesTheBaseNameOnly(t *testing.T) {
	wc := testWC()
	got, err := uploadDests(wc, UploadIn{
		Purpose:    "genesis",
		LocalPaths: []string{"/home/me/work/deep/nest/genesis.json", "../../../etc/passwd"},
	})
	if err != nil {
		t.Fatalf("uploadDests: %v", err)
	}
	want := []string{"/data/chainbench/genesis/genesis.json", "/data/chainbench/genesis/passwd"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("dest %d = %q, want %q — only the base name may reach the server", i, got[i], want[i])
		}
	}
}

func TestTransferSource_DecisionTable(t *testing.T) {
	wc := testWC()
	cases := []struct {
		name                   string
		purpose, fname, remote string
		want, errIs            string
	}{
		{name: "both ways of naming a source", purpose: "bin", remote: "/x", errIs: "not both"},
		{name: "an explicit remote path is taken as given", remote: "/data/chainbench/bin/gwemix",
			want: "/data/chainbench/bin/gwemix"},
		{name: "an unknown purpose lists the known ones", purpose: "nope", fname: "x",
			errIs: "want bin, genesis, configs, keystore, or keyrings"},
		{name: "a purpose needs a name", purpose: "bin", errIs: "--name is required with --purpose"},
		{name: "purpose and name resolve under the folder", purpose: "keystore", fname: "UTC--x",
			want: "/data/chainbench/keystore/UTC--x"},
		{name: "no source at all", errIs: "a source is required"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := transferSource(wc, c.purpose, c.fname, c.remote)
			if c.errIs != "" {
				if err == nil {
					t.Fatalf("wanted a refusal, got %q", got)
				}
				if !strings.Contains(err.Error(), c.errIs) {
					t.Errorf("refusal %q does not say %q", err, c.errIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected refusal: %v", err)
			}
			if got != c.want {
				t.Errorf("source = %q, want %q", got, c.want)
			}
		})
	}
}

// TestTransferSource_RefusesAnEscapingName: a download names its source on the
// server by purpose plus a name, and the name must stay inside that folder.
func TestTransferSource_RefusesAnEscapingName(t *testing.T) {
	for _, name := range []string{"../../etc/passwd", "/etc/passwd"} {
		if got, err := transferSource(testWC(), "genesis", name, ""); err == nil {
			t.Errorf("name %q resolved to %q instead of being refused", name, got)
		}
	}
}

// TestOpenTransferTarget_RefusesWithoutItsInputs covers the two refusals that
// need no I/O: both directions resolve a path on a server, and neither the
// server nor the data root has a default worth guessing.
func TestOpenTransferTarget_RefusesWithoutItsInputs(t *testing.T) {
	if _, _, err := openTransferTarget(Deps{}, TransferServer{Server: "server1"}); err == nil ||
		!strings.Contains(err.Error(), "--workspace-config is required") {
		t.Errorf("a missing workspace-config should be named: %v", err)
	}
	if _, _, err := openTransferTarget(Deps{}, TransferServer{WorkspaceConfigPath: "/x.yaml"}); err == nil ||
		!strings.Contains(err.Error(), "--server is required") {
		t.Errorf("a missing server should be named: %v", err)
	}
}
