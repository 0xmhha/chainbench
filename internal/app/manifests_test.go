package app_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
	_ "github.com/0xmhha/chainbench/internal/chains/all"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

func manifestInput(t *testing.T, chain string) app.ManifestInput {
	t.Helper()
	p, err := registry.Get(chain)
	if err != nil {
		t.Fatal(err)
	}
	m := p.Manifest()
	m.ID = "external-" + chain
	m.Protocol = chain
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return app.ManifestInput{Manifest: raw, Template: string(p.GenesisTemplate())}
}

func TestManagedManifestRoundTripAndEngineApplication(t *testing.T) {
	root := t.TempDir()
	s, err := app.OpenManifestStore(root)
	if err != nil {
		t.Fatal(err)
	}
	actor := app.DeploymentActor{ID: "operator", Role: "operator"}
	for _, chain := range registry.Names() {
		t.Run(chain, func(t *testing.T) {
			in := manifestInput(t, chain)
			saved, err := s.Save(actor, in)
			if err != nil {
				t.Fatal(err)
			}
			reopened, err := app.OpenManifestStore(root)
			if err != nil {
				t.Fatal(err)
			}
			got, err := reopened.Get(saved.ID)
			if err != nil {
				t.Fatal(err)
			}
			if string(got.Manifest) != string(in.Manifest) || string(got.Template) != string(in.Template) {
				t.Fatal("declaration changed")
			}
			dir := filepath.Join(root, "compose-"+chain)
			if _, err = s.ApplyManifest(context.Background(), app.Deps{}, actor, saved.ID, dir, "../../../presets/keys", ""); err != nil {
				t.Fatal(err)
			}
			state, err := app.ChainStatus(context.Background(), app.Deps{}, app.ChainStatusIn{DataDir: dir})
			if err != nil {
				t.Fatal(err)
			}
			if state.State.Chain != "external-"+chain || state.State.ManifestPath == "" {
				t.Fatal("engine selection was not applied")
			}
		})
	}
	list, err := s.List()
	if err != nil || len(list) != 6 {
		t.Fatalf("list: %d %v", len(list), err)
	}
}

func TestManagedManifestRejectsUnsupportedDeclarations(t *testing.T) {
	in := manifestInput(t, "wbft")
	cases := map[string]string{"family": strings.Replace(string(in.Manifest), `"consensus_family":"wbft"`, `"consensus_family":"raft"`, 1), "dialect": strings.Replace(string(in.Manifest), `"dialect":"geth114"`, `"dialect":"unknown"`, 1), "foreign dialect": strings.Replace(string(in.Manifest), `"dialect":"geth114"`, `"dialect":"geth110-wemix"`, 1), "unknown": strings.Replace(string(in.Manifest), `"id":`, `"raw_argv":[],"id":`, 1)}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			bad := in
			bad.Manifest = json.RawMessage(raw)
			if _, err := app.ValidateManifest(bad); err == nil {
				t.Fatal("unsupported declaration accepted")
			}
		})
	}
	s, err := app.OpenManifestStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = s.Save(app.DeploymentActor{ID: "viewer", Role: "viewer"}, in); err == nil {
		t.Fatal("viewer wrote manifest")
	}
	// Traversal and tampering cannot resolve an executable plugin.
	if _, err = s.Get("../manifest"); err == nil {
		t.Fatal("traversal accepted")
	}
}

func TestManifestIntegrityAndBuiltinProtection(t *testing.T) {
	root := t.TempDir()
	s, err := app.OpenManifestStore(root)
	if err != nil {
		t.Fatal(err)
	}
	actor := app.DeploymentActor{ID: "admin", Role: "admin"}
	in := manifestInput(t, "wbft")
	saved, err := s.Save(actor, in)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "manifests", saved.ID+".json"), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Get(saved.ID); err == nil {
		t.Fatal("tampered manifest accepted")
	}
	p, _ := registry.Get("wbft")
	raw, _ := json.Marshal(p.Manifest())
	if _, err = s.Save(actor, app.ManifestInput{Manifest: raw, Template: string(p.GenesisTemplate())}); err == nil {
		t.Fatal("builtin replaced")
	}
}
