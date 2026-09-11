package app

import (
	"context"
	"io/fs"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/resource"
)

// A bare binary name is a name on the TARGET. Before this, resolveBinaryOn
// refused it outright, so every remote run had to spell out a path the
// workspace-config already described, and binaryAliases -- which exists to map a
// logical name onto a file under the binaries directory -- had no consumer
// anywhere in production.

// probeStore answers Exists from a set of paths and records what it was asked,
// so a test can tell WHICH path the resolution decided on.
type probeStore struct {
	present map[string]bool
	asked   []string
}

func (s *probeStore) Exists(_ context.Context, p string) (bool, error) {
	s.asked = append(s.asked, p)
	return s.present[p], nil
}
func (s *probeStore) Read(context.Context, string) ([]byte, error) { return nil, nil }
func (s *probeStore) Write(context.Context, string, []byte, fs.FileMode) error {
	return nil
}
func (s *probeStore) Checksum(context.Context, string) (string, error) { return "", nil }
func (s *probeStore) Remove(context.Context, string) error             { return nil }

func targetWith(present ...string) (*resource.Access, *probeStore) {
	st := &probeStore{present: map[string]bool{}}
	for _, p := range present {
		st.present[p] = true
	}
	return &resource.Access{
		Spec:     resource.Spec{Server: "server1"},
		DataRoot: "/data/chainbench",
		Files:    st,
	}, st
}

func wcFor(t *testing.T, aliases map[string]string) *resource.WorkspaceConfig {
	t.Helper()
	return &resource.WorkspaceConfig{
		Version:       1,
		DataRoot:      "/data/chainbench",
		Paths:         resource.WorkspacePaths{Binaries: "bin"},
		BinaryAliases: aliases,
	}
}

// TestResolveBinaryOn_BareNameComesFromTheWorkspaceConfig: the config says where
// names live on the target, so "gwemix" is /data/chainbench/bin/gwemix.
func TestResolveBinaryOn_BareNameComesFromTheWorkspaceConfig(t *testing.T) {
	acc, st := targetWith("/data/chainbench/bin/gwemix")
	got, err := resolveBinaryOn(context.Background(), acc, wcFor(t, nil), "", "gwemix")
	if err != nil {
		t.Fatalf("resolveBinaryOn: %v", err)
	}
	if got != "/data/chainbench/bin/gwemix" {
		t.Errorf("resolved %q, want the path under the target's binaries directory", got)
	}
	if len(st.asked) != 1 || st.asked[0] != got {
		t.Errorf("probed %v, want exactly the resolved path", st.asked)
	}
}

// TestResolveBinaryOn_AppliesABinaryAlias is the assertion binaryAliases never
// had: a declared alias changes which file the name resolves to.
func TestResolveBinaryOn_AppliesABinaryAlias(t *testing.T) {
	acc, _ := targetWith("/data/chainbench/bin/linux-arm64/gwemix")
	wc := wcFor(t, map[string]string{"gwemix": "linux-arm64/gwemix"})
	got, err := resolveBinaryOn(context.Background(), acc, wc, "", "gwemix")
	if err != nil {
		t.Fatalf("resolveBinaryOn: %v", err)
	}
	if got != "/data/chainbench/bin/linux-arm64/gwemix" {
		t.Errorf("resolved %q, want the alias applied", got)
	}
}

// TestResolveBinaryOn_AbsolutePathIsTakenAsGiven: an operator naming a path on
// the target means that path, alias or no alias.
func TestResolveBinaryOn_AbsolutePathIsTakenAsGiven(t *testing.T) {
	acc, _ := targetWith("/opt/build/gwemix")
	wc := wcFor(t, map[string]string{"gwemix": "linux-arm64/gwemix"})
	got, err := resolveBinaryOn(context.Background(), acc, wc, "/opt/build/gwemix", "gwemix")
	if err != nil {
		t.Fatalf("resolveBinaryOn: %v", err)
	}
	if got != "/opt/build/gwemix" {
		t.Errorf("resolved %q, want the explicit path untouched", got)
	}
}

// TestResolveBinaryOn_AbsentBinaryNamesTheFileAndTheServer keeps the refusal
// that made the first remote run diagnosable: an absent binary fails here, not
// as a node that never came up.
func TestResolveBinaryOn_AbsentBinaryNamesTheFileAndTheServer(t *testing.T) {
	acc, _ := targetWith() // nothing present
	_, err := resolveBinaryOn(context.Background(), acc, wcFor(t, nil), "", "gwemix")
	if err == nil {
		t.Fatal("a binary that is not on the target was accepted")
	}
	for _, want := range []string{"/data/chainbench/bin/gwemix", "server1"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal should name %q: %v", want, err)
		}
	}
}

// TestResolveBinaryOn_BareNameWithoutAConfigIsRefused: without a workspace-config
// nothing says where a name lives on the target, and looking it up in the
// operator's own PATH is how the first remote run reported a binary "not found"
// that was sitting on the server all along.
func TestResolveBinaryOn_BareNameWithoutAConfigIsRefused(t *testing.T) {
	acc, _ := targetWith("/data/chainbench/bin/gwemix")
	_, err := resolveBinaryOn(context.Background(), acc, nil, "", "gwemix")
	if err == nil {
		t.Fatal("a bare name with no workspace-config was accepted")
	}
	if !strings.Contains(err.Error(), "workspace-config") {
		t.Errorf("the refusal should say what is missing: %v", err)
	}
}
