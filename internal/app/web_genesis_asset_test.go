package app

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWebPresetPinsRegisteredFinishedGenesis(t *testing.T) {
	e, p, a := webPresetFixture(t, `{"bp":4}`)
	var err error
	e.manifests, err = OpenManifestStore(e.root)
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte("{\n  \"config\":{\"chainId\":9311},\n  \"alloc\":{},\"extraData\":\"0x\"\n}\n")
	asset, err := e.manifests.UploadAsset(context.Background(), a, "template", "finished-genesis.json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := e.documents.Document(p.Arguments.ChainPresetRef.ID)
	if err != nil {
		t.Fatal(err)
	}
	var content map[string]any
	_ = json.Unmarshal(doc.Content, &content)
	content["genesis"] = map[string]any{"mode": "existing", "ref": "asset:" + asset.ID}
	doc.Content, _ = json.Marshal(content)
	doc.AssetRefs = []string{asset.ID}
	saved, err := e.documents.SaveDocument(a, doc.ID, doc.Revision, doc.DeploymentDocumentInput)
	if err != nil {
		t.Fatalf("registered genesis document rejected: %v", err)
	}
	p.Arguments.ChainPresetRef = &DeploymentDocumentRef{ID: saved.ID, Revision: saved.Revision}
	if _, err = e.preparePresetComposition(context.Background(), &p); err != nil {
		t.Fatalf("registered genesis plan rejected: %v", err)
	}
	path := p.Preset.Request.GenesisExisting
	if path == "" || !filepath.IsAbs(path) || !strings.HasPrefix(path, e.root+string(os.PathSeparator)) || strings.Contains(path, "asset:") {
		t.Fatal("genesis reference was not pinned privately", path)
	}
	for _, read := range []DeploymentDocument{saved, mustGenesisDocument(t, e.documents, saved.ID), e.documents.Documents("chain-preset")[0]} {
		if len(read.AssetRefs) != 1 || read.AssetRefs[0] != asset.ID {
			t.Fatal("stored dependency missing")
		}
		read.AssetRefs[0] = "mutated"
		if mustGenesisDocument(t, e.documents, saved.ID).AssetRefs[0] != asset.ID {
			t.Fatal("caller modified immutable stored dependencies")
		}
	}
	preview, err := e.documents.PreviewImport(a, DocumentImportInput{Filename: "preset.json", Format: "json", Kind: "chain-preset", Source: string(saved.Content)})
	if err != nil || !preview.Validation.Valid || len(preview.RedactedDocuments) != 1 || len(preview.RedactedDocuments[0].AssetRefs) != 1 || preview.RedactedDocuments[0].AssetRefs[0] != asset.ID {
		t.Fatal("import lost registered dependency", err, preview.Validation)
	}
	got, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(got, raw) {
		t.Fatal("finished genesis bytes changed", err)
	}
	if _, err = os.Stat(p.ControlDir); !os.IsNotExist(err) {
		t.Fatal("planning touched retained network")
	}
	if err = os.WriteFile(path, []byte(`{"config":{"chainId":1}}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = e.projectPresetComposition(context.Background(), p); err == nil {
		t.Fatal("changed pinned genesis accepted or silently repaired")
	}
}

func mustGenesisDocument(t *testing.T, s *DeploymentStore, id string) DeploymentDocument {
	t.Helper()
	doc, err := s.Document(id)
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestWebPresetRefusesUnknownOrWrongKindGenesisBeforeEffects(t *testing.T) {
	for _, kind := range []string{"unknown", "configuration"} {
		t.Run(kind, func(t *testing.T) {
			e, p, a := webPresetFixture(t, `{"bp":4}`)
			var err error
			e.manifests, err = OpenManifestStore(e.root)
			if err != nil {
				t.Fatal(err)
			}
			id := strings.Repeat("c", 32)
			if kind != "unknown" {
				asset, err := e.manifests.UploadAsset(context.Background(), a, kind, "genesis.json", strings.NewReader(`{"config":{"chainId":9311},"alloc":{}}`))
				if err != nil {
					t.Fatal(err)
				}
				id = asset.ID
			}
			doc, err := e.documents.Document(p.Arguments.ChainPresetRef.ID)
			if err != nil {
				t.Fatal(err)
			}
			var content map[string]any
			_ = json.Unmarshal(doc.Content, &content)
			content["genesis"] = map[string]any{"mode": "existing", "ref": "asset:" + id}
			doc.Content, _ = json.Marshal(content)
			doc.AssetRefs = []string{id}
			saved, err := e.documents.SaveDocument(a, doc.ID, doc.Revision, doc.DeploymentDocumentInput)
			if err != nil {
				t.Fatalf("reference cannot reach planning: %v", err)
			}
			p.Arguments.ChainPresetRef = &DeploymentDocumentRef{ID: saved.ID, Revision: saved.Revision}
			if _, err = e.preparePresetComposition(context.Background(), &p); err == nil {
				t.Fatal("unregistered or wrong-kind genesis accepted")
			}
			if _, err = os.Stat(p.ControlDir); !os.IsNotExist(err) {
				t.Fatal("rejected plan touched retained network")
			}
		})
	}
}
