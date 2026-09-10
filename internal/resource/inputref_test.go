package resource

import (
	"fmt"
	"strings"
	"testing"
)

func wcForRefs(t *testing.T) *WorkspaceConfig {
	t.Helper()
	c, err := ParseWorkspaceConfig([]byte(validConfig))
	if err != nil {
		t.Fatal(err)
	}
	return &c
}

// TestInputRef_Portable resolves a bare relative ref under the workspace-config
// purpose directory, on the target (no server).
func TestInputRef_Portable(t *testing.T) {
	wc := wcForRefs(t)
	loc, err := InputRef{Raw: "genesis-wemix.json"}.Resolve(wc, PurposeGenesis, nil)
	if err != nil {
		t.Fatal(err)
	}
	if loc.Server != "" || loc.Local || loc.Path != "/data/genesis/genesis-wemix.json" {
		t.Fatalf("portable ref = %+v, want target path under /data/genesis", loc)
	}
}

// TestInputRef_Srv keeps the existing srv:// contract: the named server and the
// absolute path verbatim, with no data root prepended.
func TestInputRef_Srv(t *testing.T) {
	loc, err := InputRef{Raw: "srv://server-01/data/genesis/base.json"}.Resolve(nil, PurposeGenesis, nil)
	if err != nil {
		t.Fatal(err)
	}
	if loc.Server != "server-01" || loc.Path != "/data/genesis/base.json" {
		t.Fatalf("srv ref = %+v, want server-01 and the verbatim absolute path", loc)
	}
	// srv://1/... keeps meaning the name "1", not an index.
	loc, err = InputRef{Raw: "srv://1/data/x"}.Resolve(nil, PurposeGenesis, nil)
	if err != nil || loc.Server != "1" {
		t.Fatalf("srv://1 = %+v (%v), want server name \"1\"", loc, err)
	}
}

// TestInputRef_ServerRef resolves {server, ref} to that server and the workspace
// purpose path — the root form that survives a data-root change.
func TestInputRef_ServerRef(t *testing.T) {
	wc := wcForRefs(t)
	loc, err := InputRef{Server: "server-01", Ref: "base.json"}.Resolve(wc, PurposeGenesis, nil)
	if err != nil {
		t.Fatal(err)
	}
	if loc.Server != "server-01" || loc.Path != "/data/genesis/base.json" {
		t.Fatalf("{server,ref} = %+v, want server-01 under /data/genesis", loc)
	}
}

// TestInputRef_ServerIndex resolves a 1-based index to a fixed name, then as a
// server ref. The name is what a resume would store, so the list order can
// change without re-pointing.
func TestInputRef_ServerIndex(t *testing.T) {
	wc := wcForRefs(t)
	indexName := func(i int) (string, error) {
		if i == 2 {
			return "server-02", nil
		}
		return "", fmt.Errorf("no server %d", i)
	}
	loc, err := InputRef{ServerIndex: 2, Ref: "base.json"}.Resolve(wc, PurposeGenesis, indexName)
	if err != nil {
		t.Fatal(err)
	}
	if loc.Server != "server-02" || loc.Path != "/data/genesis/base.json" {
		t.Fatalf("{serverIndex,ref} = %+v, want the resolved name server-02", loc)
	}
	// A bad index surfaces the lookup's error.
	if _, err := (InputRef{ServerIndex: 9, Ref: "x"}).Resolve(wc, PurposeGenesis, indexName); err == nil {
		t.Fatal("an out-of-range serverIndex was accepted")
	}
}

// TestInputRef_LocalPath names a file on the machine running chainbench.
func TestInputRef_LocalPath(t *testing.T) {
	loc, err := InputRef{LocalPath: "/private/fixtures/base.json"}.Resolve(nil, PurposeGenesis, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !loc.Local || loc.Server != "" || loc.Path != "/private/fixtures/base.json" {
		t.Fatalf("localPath = %+v, want a local file", loc)
	}
}

// TestInputRef_Rejects: empty, more than one form, a portable ref with no
// workspace-config, and a server ref with no ref are all refused.
func TestInputRef_Rejects(t *testing.T) {
	wc := wcForRefs(t)
	cases := map[string]struct {
		ref InputRef
		wc  *WorkspaceConfig
	}{
		"empty":               {InputRef{}, wc},
		"two forms":           {InputRef{Raw: "x", LocalPath: "/y"}, wc},
		"server and index":    {InputRef{Server: "a", ServerIndex: 1, Ref: "r"}, wc},
		"server without ref":  {InputRef{Server: "a"}, wc},
		"index without ref":   {InputRef{ServerIndex: 1}, wc},
		"portable without wc": {InputRef{Raw: "rel.json"}, nil},
		"negative index":      {InputRef{ServerIndex: -1, Ref: "r"}, wc},
		"traversing portable": {InputRef{Raw: "../escape"}, wc},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := tc.ref.Resolve(tc.wc, PurposeGenesis, func(int) (string, error) { return "n", nil }); err == nil {
				t.Fatalf("%s was accepted, want rejected", name)
			}
		})
	}
}

// TestInputRef_ServerRefSurvivesRootChange: the same {server, ref} resolves under
// whatever data root the workspace-config names — the reason to prefer it over
// an absolute srv:// path.
func TestInputRef_ServerRefSurvivesRootChange(t *testing.T) {
	other := strings.Replace(validConfig, "dataRoot: /data", "dataRoot: /srv/x", 1)
	c, err := ParseWorkspaceConfig([]byte(other))
	if err != nil {
		t.Fatal(err)
	}
	loc, err := InputRef{Server: "s1", Ref: "g.json"}.Resolve(&c, PurposeGenesis, nil)
	if err != nil {
		t.Fatal(err)
	}
	if loc.Path != "/srv/x/genesis/g.json" {
		t.Fatalf("server ref did not follow the data root: %q", loc.Path)
	}
}
