package app

import (
	"encoding/json"
	"testing"
)

func TestReadRevisionDoesNotShareItsPresetReferences(t *testing.T) {
	store, err := OpenDeploymentStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "alice", Role: "operator"}
	preset, err := store.SaveDocument(a, "", 0, DeploymentDocumentInput{Kind: "chain-preset", Name: "p", ContractVersion: "2", Content: json.RawMessage(`{"schemaVersion":"2","kind":"chain-preset","id":"p","chain":"stablenet"}`), AssetRefs: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	c, err := store.SaveDocument(a, "", 0, DeploymentDocumentInput{Kind: "case", Name: "c", ContractVersion: "2", Content: json.RawMessage(`{"schemaVersion":"2","kind":"case","id":"c","chainPreset":"p","steps":[{"expect":"blockNumber","is":0}]}`), AssetRefs: []string{}, PresetRefs: []DeploymentDocumentRef{{ID: preset.ID, Revision: preset.Revision}}})
	if err != nil {
		t.Fatal(err)
	}
	read, err := store.DocumentRevision(c.ID, c.Revision)
	if err != nil || len(read.PresetRefs) != 1 {
		t.Fatalf("%+v %v", read, err)
	}
	read.PresetRefs[0].Revision = 99
	again, _ := store.DocumentRevision(c.ID, c.Revision)
	if again.PresetRefs[0].Revision != preset.Revision {
		t.Fatalf("changing a read revision changed the stored one: %+v", again.PresetRefs)
	}
}
