package app

import (
	"encoding/json"
	"strings"
	"testing"
)

// A case exported with the preset revisions it pins imports back as a case
// pinned to the presets that arrived with it, running the same declaration.
func TestCaseBundleCarriesItsPinnedPresets(t *testing.T) {
	e, _, a, _, in := presetCaseFixture(t)
	saved, err := e.documents.SaveDocument(a, "", 0, in)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := e.documents.ExportDocumentBundle(a, saved.ID)
	if err != nil {
		t.Fatalf("case bundle export: %v", err)
	}
	var shape struct {
		Documents []struct {
			Kind string `json:"kind"`
		} `json:"documents"`
	}
	if err = json.Unmarshal(bundle, &shape); err != nil || len(shape.Documents) != 2 || shape.Documents[0].Kind != "chain-preset" || shape.Documents[1].Kind != "case" {
		t.Fatalf("bundle does not hold the pinned preset then the case: %s", bundle)
	}
	preview, err := e.documents.PreviewImport(a, DocumentImportInput{Filename: "case.bundle.json", Format: "json", Source: string(bundle)})
	if err != nil || !preview.Validation.Valid {
		t.Fatalf("case bundle preview refused: %v %+v", err, preview.Validation)
	}
	committed, err := e.documents.CommitImport(a, DocumentImportCommit{PreviewID: preview.PreviewID})
	if err != nil {
		t.Fatalf("case bundle commit: %v", err)
	}
	preset, imported := committed[0], committed[1]
	if len(imported.PresetRefs) != 1 || imported.PresetRefs[0] != (DeploymentDocumentRef{ID: preset.ID, Revision: preset.Revision}) {
		t.Fatalf("imported case is not pinned to the preset that arrived with it: %+v", imported.PresetRefs)
	}
	before, err := e.documents.ExecutableCaseContent(saved.DeploymentDocumentInput)
	if err != nil {
		t.Fatal(err)
	}
	after, err := e.documents.ExecutableCaseContent(imported.DeploymentDocumentInput)
	if err != nil {
		t.Fatal(err)
	}
	if !sameJSON(t, before, after) {
		t.Fatalf("the imported case runs another declaration:\n%s\n%s", before, after)
	}
}

func TestCaseBundleNamesAPresetItDoesNotCarry(t *testing.T) {
	e, _, a, _, in := presetCaseFixture(t)
	var content any
	if err := json.Unmarshal(in.Content, &content); err != nil {
		t.Fatal(err)
	}
	bundle, _ := json.Marshal(map[string]any{"documents": []any{map[string]any{"kind": "case", "name": "lonely", "content": content}}})
	preview, err := e.documents.PreviewImport(a, DocumentImportInput{Filename: "case.bundle.json", Format: "json", Source: string(bundle)})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Validation.Valid || len(preview.Validation.Errors) != 1 || !strings.Contains(preview.Validation.Errors[0].Message, "stablenet-bp4-en1") {
		t.Fatalf("a case without its preset was not refused by name: %+v", preview.Validation)
	}
}

func sameJSON(t *testing.T, a, b []byte) bool {
	t.Helper()
	var x, y any
	if json.Unmarshal(a, &x) != nil || json.Unmarshal(b, &y) != nil {
		t.Fatal("not JSON")
	}
	l, _ := json.Marshal(x)
	r, _ := json.Marshal(y)
	return string(l) == string(r)
}
