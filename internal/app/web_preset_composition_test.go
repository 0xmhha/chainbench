package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

func webPresetFixture(t *testing.T, topology string) (*WebChainEngine, webChainPayload, DeploymentActor) {
	t.Helper()
	e, p := webRunPlanFixture(t, `{"bp":4}`)
	actor := DeploymentActor{ID: "operator", Role: "operator"}
	raw := json.RawMessage(`{"schemaVersion":"2","kind":"chain-preset","id":"saved-preset","chain":"stablenet","topology":` + topology + `,"genesis":{"overlay":{"config":{"chainId":9123}}},"launch":{"all":{"maxpeers":"40","cache":"64"},"node1":{"maxpeers":"50"}},"config":{"all":{"httpHost":"127.0.0.1","metricsHost":"127.0.0.1"}}}`)
	doc, err := e.documents.SaveDocument(actor, "", 0, DeploymentDocumentInput{Kind: "chain-preset", Name: "saved", ContractVersion: "2", Content: raw})
	if err != nil {
		t.Fatal(err)
	}
	p.Arguments.CaseRefs = nil
	p.Arguments.ChainPresetRef = &DeploymentDocumentRef{ID: doc.ID, Revision: doc.Revision}
	return e, p, actor
}

func TestWebPresetCompositionPreservesCountsOptionsAndSnapshot(t *testing.T) {
	e, p, a := webPresetFixture(t, `{"bp":4,"en":1,"syncMode":"archive"}`)
	requests, err := e.preparePresetComposition(context.Background(), &p)
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 5 || requests[4].Role != node.RoleEN || p.Arguments.Validators != 4 {
		t.Fatal("preset counts lost", requests)
	}
	in := p.Preset.Request
	if in.BPCount != 4 || in.ENCount != 1 || in.EndpointSyncMode != "archive" || in.Binary != p.ExecutionBinary {
		t.Fatal("execution projection differs from declaration", in)
	}
	if in.KeysDir != webAcceptedKeyPath(e.root, p.Keys.SHA256) || in.DataDir != p.ControlDir {
		t.Fatal("unpinned keys or control root")
	}
	if strings.Join(in.LaunchScoped["all"], ",") != "cache=64,maxpeers=40" || strings.Join(in.ConfigSet["all"], ",") != "httpHost=127.0.0.1,metricsHost=127.0.0.1" {
		t.Fatal("knob order is not deterministic", in)
	}
	overlay, err := os.ReadFile(in.OverlayPath)
	if err != nil || !strings.Contains(string(overlay), "9123") {
		t.Fatal("accepted genesis lost", err)
	}
	if _, err = os.Stat(p.ControlDir); !os.IsNotExist(err) {
		t.Fatal("planning touched retained network")
	}
	doc := p.Preset.Document
	changed := doc.DeploymentDocumentInput
	changed.Content = json.RawMessage(strings.Replace(string(doc.Content), "9123", "9999", 1))
	if _, err = e.documents.SaveDocument(a, doc.ID, doc.Revision, changed); err != nil {
		t.Fatal(err)
	}
	if _, err = e.preparePresetComposition(context.Background(), &p); !errors.Is(err, ErrDeploymentConflict) {
		t.Fatal("stale selection accepted", err)
	}
	before, _ := json.Marshal(p.Preset)
	for range 40 {
		current, err := e.projectPresetComposition(context.Background(), p)
		if err != nil {
			t.Fatal(err)
		}
		after, _ := json.Marshal(current)
		if string(before) != string(after) {
			t.Fatal("accepted projection changed after shared edit or map iteration")
		}
	}
}

func TestWebPresetCompositionKeepsNodeTableClaimOrder(t *testing.T) {
	e, p, _ := webPresetFixture(t, `{"nodes":[{"index":1,"role":"en","sync":"archive"},{"index":2,"role":"bp"},{"index":3,"role":"bp"},{"index":4,"role":"bp"},{"index":5,"role":"bp"}]}`)
	requests, err := e.preparePresetComposition(context.Background(), &p)
	if err != nil {
		t.Fatal(err)
	}
	if len(requests) != 5 || requests[0].Role != node.RoleEN || requests[0].Label != "node1" || requests[1].Role != node.RoleBP {
		t.Fatal("table claims do not match execution order", requests)
	}
	if !p.Preset.Plan.Nodes.Declared || p.Preset.Request.Topology.Sorted()[0].EffectiveSyncMode() != "archive" {
		t.Fatal("table options lost")
	}
}

func TestWebPresetCompositionRejectsUnregisteredInputs(t *testing.T) {
	e, p, _ := webPresetFixture(t, `{"bp":4}`)
	if _, err := e.preparePresetComposition(context.Background(), &p); err != nil {
		t.Fatal(err)
	}
	for name, edit := range map[string]func(map[string]any){
		"other-chain":      func(d map[string]any) { d["chain"] = "wemix" },
		"managed-port":     func(d map[string]any) { d["launch"] = map[string]any{"node1": map[string]any{"http.port": "9000"}} },
		"data-path":        func(d map[string]any) { d["launch"] = map[string]any{"all": map[string]any{"datadir": "/other"}} },
		"generate-keys":    func(d map[string]any) { d["keys"] = map[string]any{"nodekeys": map[string]any{"source": "generate"}} },
		"named-binary":     func(d map[string]any) { d["binaries"] = map[string]any{"alternate": "/unregistered"} },
		"existing-genesis": func(d map[string]any) { d["genesis"] = map[string]any{"mode": "existing", "ref": "/unregistered"} },
		"node-file": func(d map[string]any) {
			d["topology"] = map[string]any{"nodes": []any{map[string]any{"role": "bp", "config": "/unregistered"}}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			var d map[string]any
			if err := json.Unmarshal(p.Preset.Document.Content, &d); err != nil {
				t.Fatal(err)
			}
			edit(d)
			raw, _ := json.Marshal(d)
			copyP := p
			copyPreset := *p.Preset
			copyPreset.Document.Content = raw
			copyP.Preset = &copyPreset
			if _, err := e.projectPresetComposition(context.Background(), copyP); err == nil {
				t.Fatal("unregistered input accepted")
			}
		})
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(p.Preset.Request.Server.SetPath), "server-set.yaml"), []byte("redirected"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := e.projectPresetComposition(context.Background(), p); !errors.Is(err, ErrDeploymentConflict) {
		t.Fatal("changed cached target declaration accepted", err)
	}
}
