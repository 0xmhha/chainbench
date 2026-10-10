package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl"
)

func webCaseGenesisFixture(t *testing.T, legacy bool) (*WebChainEngine, webChainPayload, WebAsset, DeploymentActor, []byte) {
	t.Helper()
	e, p := webRunPlanFixture(t, `{"bp":4}`)
	a := DeploymentActor{ID: "operator", Role: "operator"}
	var err error
	e.manifests, err = OpenManifestStore(e.root)
	if err != nil {
		t.Fatal(err)
	}
	raw := []byte("{\n\"config\":{\"chainId\":9311},\"alloc\":{},\"extraData\":\"0x\"\n}\n")
	asset, err := e.manifests.UploadAsset(context.Background(), a, "template", "finished.json", bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := e.documents.Document(p.Arguments.CaseRefs[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if legacy {
		doc.Content = json.RawMessage(`{"schemaVersion":"1","id":"legacy-genesis","chain":{"name":"stablenet","binary":"gstable","genesisExisting":"asset:` + asset.ID + `"},"topology":{"bp":4},"assertions":[{"assert":"blockNumber","expected":0}]}`)
	} else {
		var content map[string]any
		if err = json.Unmarshal(doc.Content, &content); err != nil {
			t.Fatal(err)
		}
		content["chainPreset"].(map[string]any)["genesis"] = map[string]any{"mode": "existing", "ref": "asset:" + asset.ID}
		doc.Content, _ = json.Marshal(content)
	}
	doc.AssetRefs = []string{asset.ID}
	saved, err := e.documents.SaveDocument(a, doc.ID, doc.Revision, doc.DeploymentDocumentInput)
	if err != nil {
		t.Fatalf("registered case genesis dependency rejected: %v", err)
	}
	p.Arguments.CaseRefs = []DeploymentDocumentRef{{ID: saved.ID, Revision: saved.Revision}}
	return e, p, asset, a, raw
}

func TestWebTestRunPinsRegisteredFinishedGenesisWithoutChangingDeclaration(t *testing.T) {
	for _, legacy := range []bool{false, true} {
		t.Run(map[bool]string{false: "v2", true: "v1"}[legacy], func(t *testing.T) {
			e, p, asset, _, raw := webCaseGenesisFixture(t, legacy)
			if _, err := e.prepareTestRun(context.Background(), &p); err != nil {
				t.Fatalf("registered case genesis cannot be planned: %v", err)
			}
			spec, err := dsl.Parse(p.TestRun.Content[0])
			if err != nil || !filepath.IsAbs(spec.Chain.GenesisExisting) || !strings.HasPrefix(spec.Chain.GenesisExisting, e.root+string(os.PathSeparator)) {
				t.Fatal("genesis was not projected to a private immutable input", err)
			}
			got, err := os.ReadFile(spec.Chain.GenesisExisting)
			if err != nil || !bytes.Equal(got, raw) {
				t.Fatal("registered finished genesis bytes changed", err)
			}
			if !bytes.Contains(p.TestRun.Cases[0].Document.Content, []byte("asset:"+asset.ID)) {
				t.Fatal("execution replaced the public declaration")
			}
			if _, err = os.Stat(p.ControlDir); !os.IsNotExist(err) {
				t.Fatal("genesis planning touched retained nodes", err)
			}
		})
	}
}

func TestWebCaseGenesisDependencySurvivesImportAndExport(t *testing.T) {
	e, p, asset, a, _ := webCaseGenesisFixture(t, false)
	doc, err := e.documents.Document(p.Arguments.CaseRefs[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	preview, err := e.documents.PreviewImport(a, DocumentImportInput{Filename: "case.json", Format: "json", Kind: "case", Source: string(doc.Content)})
	if err != nil || !preview.Validation.Valid || len(preview.RedactedDocuments) != 1 {
		t.Fatal("case genesis import failed", err)
	}
	if len(preview.RedactedDocuments[0].AssetRefs) != 1 || preview.RedactedDocuments[0].AssetRefs[0] != asset.ID {
		t.Fatal("import dropped genesis dependency")
	}
	exported, err := ExportDeploymentDocument(doc.DeploymentDocumentInput, "json")
	if err != nil || !bytes.Equal(exported, doc.Content) {
		t.Fatal("export changed the registered genesis reference", err)
	}
}

func TestWebTestGenesisRefusesUnknownWrongKindAndMissingDependency(t *testing.T) {
	for _, kind := range []string{"unknown", "configuration", "missing-dependency"} {
		t.Run(kind, func(t *testing.T) {
			e, p, asset, a, raw := webCaseGenesisFixture(t, false)
			doc := mustGenesisDocument(t, e.documents, p.Arguments.CaseRefs[0].ID)
			id := strings.Repeat("f", 32)
			switch kind {
			case "configuration":
				other, err := e.manifests.UploadAsset(context.Background(), a, kind, "genesis.json", bytes.NewReader(raw))
				if err != nil {
					t.Fatal(err)
				}
				id = other.ID
			case "missing-dependency":
				doc.AssetRefs = nil
				if _, err := e.documents.SaveDocument(a, doc.ID, doc.Revision, doc.DeploymentDocumentInput); err == nil {
					t.Fatal("genesis dependency can be omitted")
				}
				return
			}
			doc.Content = bytes.ReplaceAll(doc.Content, []byte(asset.ID), []byte(id))
			doc.AssetRefs = []string{id}
			saved, err := e.documents.SaveDocument(a, doc.ID, doc.Revision, doc.DeploymentDocumentInput)
			if err != nil {
				t.Fatal(err)
			}
			p.Arguments.CaseRefs = []DeploymentDocumentRef{{saved.ID, saved.Revision}}
			if _, err = e.prepareTestRun(context.Background(), &p); err == nil {
				t.Fatal("unknown or wrong-kind genesis was executable")
			}
			if _, err = os.Stat(p.ControlDir); !os.IsNotExist(err) {
				t.Fatal("rejected genesis touched retained network", err)
			}
		})
	}
}

func TestWebTestGenesisChangesRefusedBeforeNodeEffects(t *testing.T) {
	for _, where := range []string{"source", "snapshot"} {
		t.Run(where, func(t *testing.T) {
			e, p, asset, a, _ := webCaseGenesisFixture(t, false)
			if _, err := e.prepareTestRun(context.Background(), &p); err != nil {
				t.Fatal(err)
			}
			path := p.TestRun.Genesis[0].Path
			if where == "source" {
				path = filepath.Join(e.root, "assets", asset.ID, "payload")
			}
			if err := os.WriteFile(path, []byte(`{"config":{"chainId":1}}`), 0600); err != nil {
				t.Fatal(err)
			}
			_, err := e.executeTestRun(context.Background(), a, p, func(WebJobPhase) error { return nil })
			if err == nil {
				t.Fatal("changed genesis reached native execution")
			}
			if _, err = os.Stat(p.ControlDir); !os.IsNotExist(err) {
				t.Fatal("changed genesis touched retained network", err)
			}
		})
	}
}

func TestWebTestGenesisRecheckedAfterBinaryTransfer(t *testing.T) {
	e, p, _, a, _ := webCaseGenesisFixture(t, false)
	// This fixture transfers inert bytes; no native binary can be launched.
	data := []byte("inert-owned-test-binary")
	path := filepath.Join(e.root, "source-binary")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	p.Arguments.AssetID = "fixture"
	p.Binary.SHA256 = manifestHash(data)
	e.assets["fixture"] = ManifestBinary{ID: "fixture", Path: path, SHA256: p.Binary.SHA256}
	if _, err := e.prepareTestRun(context.Background(), &p); err != nil {
		t.Fatal(err)
	}
	seen := false
	_, err := e.executeTestRun(context.Background(), a, p, func(phase WebJobPhase) error {
		if phase.Name == "test.run" && phase.State == "running" {
			seen = true
			return os.WriteFile(p.TestRun.Genesis[0].Path, []byte(`{"config":{"chainId":1}}`), 0600)
		}
		return nil
	})
	if !seen || !errors.Is(err, ErrDeploymentConflict) {
		t.Fatal("genesis mutation after transfer was not caught at the native boundary", seen, err)
	}
	if _, err = os.Stat(filepath.Join(p.ControlDir, "chain-record.json")); !os.IsNotExist(err) {
		t.Fatal("changed genesis created a native record", err)
	}
}

func TestWebCaseGenesisProjectionPreservesLargeIntegers(t *testing.T) {
	for _, raw := range []string{
		`{"schemaVersion":"2","chainPreset":{"genesis":{"mode":"existing","ref":"asset:test"}},"steps":[{"expect":"blockNumber","is":9007199254740993}]}`,
		`{"schemaVersion":"1","chain":{"genesisExisting":"asset:test"},"assertions":[{"assert":"blockNumber","expected":9007199254740993}]}`,
	} {
		got, err := projectWebCaseGenesisPath(json.RawMessage(raw), "/owned/genesis.json")
		if err != nil || !bytes.Contains(got, []byte("9007199254740993")) || bytes.Contains(got, []byte("asset:test")) {
			t.Fatal("private path projection rounded another declaration field", string(got), err)
		}
	}
}

func TestWebTestCasesShareOneRegisteredGenesisSnapshot(t *testing.T) {
	e, p, asset, a, _ := webCaseGenesisFixture(t, false)
	doc := mustGenesisDocument(t, e.documents, p.Arguments.CaseRefs[0].ID)
	var content map[string]json.RawMessage
	if err := json.Unmarshal(doc.Content, &content); err != nil {
		t.Fatal(err)
	}
	content["id"] = json.RawMessage(`"second-genesis-case"`)
	doc.Content, _ = json.Marshal(content)
	other, err := e.documents.SaveDocument(a, "", 0, doc.DeploymentDocumentInput)
	if err != nil {
		t.Fatal(err)
	}
	p.Arguments.CaseRefs = append(p.Arguments.CaseRefs, DeploymentDocumentRef{other.ID, other.Revision})
	if _, err = e.prepareTestRun(context.Background(), &p); err != nil {
		t.Fatal("identical registered genesis split a compatible suite", err)
	}
	if len(p.TestRun.Genesis) != 1 || p.TestRun.Genesis[0].Asset.ID != asset.ID {
		t.Fatal("shared dependency is not pinned once")
	}
	for _, raw := range p.TestRun.Content {
		spec, err := dsl.Parse(raw)
		if err != nil || spec.Chain.GenesisExisting != p.TestRun.Genesis[0].Path {
			t.Fatal("compatible cases have distinct environment paths", err)
		}
	}
}
