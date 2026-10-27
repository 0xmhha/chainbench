package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Every revision of a shared document can be listed with who saved it and
// when, newest first, without carrying the content.
func TestDocumentRevisionsListWhoChangedWhat(t *testing.T) {
	store, err := OpenDeploymentStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	first, second := DeploymentActor{ID: "first", Role: "operator"}, DeploymentActor{ID: "second", Role: "operator"}
	doc := dockerSet("")
	saved, err := store.SaveDocument(first, "", 0, doc)
	if err != nil {
		t.Fatal(err)
	}
	doc.Name = "renamed"
	if _, err = store.SaveDocument(second, saved.ID, 1, doc); err != nil {
		t.Fatal(err)
	}
	revisions, err := store.DocumentRevisions(saved.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(revisions) != 2 || revisions[0].Revision != 2 || revisions[0].UpdatedBy != "second" || revisions[0].Name != "renamed" || revisions[1].UpdatedBy != "first" || revisions[0].UpdatedAt.IsZero() {
		t.Fatalf("revisions %+v", revisions)
	}
	if raw, _ := json.Marshal(revisions); json.Valid(raw) && containsKey(raw, "content") {
		t.Fatalf("a revision list carried content: %s", raw)
	}
	if _, err = store.DocumentRevisions("absent"); err != ErrDeploymentNotFound {
		t.Fatalf("absent document: %v", err)
	}
}

func containsKey(raw []byte, key string) bool {
	var items []map[string]any
	_ = json.Unmarshal(raw, &items)
	for _, item := range items {
		if _, ok := item[key]; ok {
			return true
		}
	}
	return false
}

// A preset file the catalogue leaves out is named with the reason, instead of
// disappearing.
func TestChainPresetCatalogueNamesTheFilesItSkipped(t *testing.T) {
	root := t.TempDir()
	good, err := os.ReadFile("../../presets/chain/stablenet-bp4-boho.json")
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"good.json":    string(good),
		"broken.json":  `{`,
		"unknown.json": `{"schemaVersion":"2","kind":"chain-preset","id":"x","chain":"nosuchchain"}`,
		"notes.txt":    "not a preset",
	} {
		if err = os.WriteFile(filepath.Join(root, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	catalogue, err := ChainPresetCatalogue(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(catalogue.Items) != 1 || len(catalogue.Skipped) != 2 {
		t.Fatalf("catalogue %+v", catalogue)
	}
	for _, s := range catalogue.Skipped {
		if (s.File != "broken.json" && s.File != "unknown.json") || s.Reason == "" {
			t.Errorf("skipped %+v", s)
		}
	}
	listed, err := ChainPresets(root)
	if err != nil || len(listed) != 1 {
		t.Fatalf("the plain list changed: %v %v", listed, err)
	}
}

// An import preview says until when its encrypted source can be committed.
func TestImportPreviewSaysWhenItExpires(t *testing.T) {
	store, err := OpenDeploymentStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	before := time.Now()
	preview, err := store.PreviewImport(DeploymentActor{ID: "first", Role: "operator"}, DocumentImportInput{Filename: "set.json", Format: "json", Kind: "server-set", Source: string(dockerSet("").Content)})
	if err != nil {
		t.Fatal(err)
	}
	if left := preview.ExpiresAt.Sub(before); left < 14*time.Minute || left > 16*time.Minute {
		t.Fatalf("expires at %v, %v after the request", preview.ExpiresAt, left)
	}
}
