package app

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl"
)

// presetCaseFixture saves the corpus chain preset as a shared document and a
// case that extends it by id, the way most of tests/tc is written.
func presetCaseFixture(t *testing.T) (*WebChainEngine, webChainPayload, DeploymentActor, DeploymentDocument, DeploymentDocumentInput) {
	t.Helper()
	e, p := webRunPlanFixture(t, `{"bp":4}`)
	a := DeploymentActor{ID: "operator", Role: "operator"}
	raw, err := os.ReadFile("../../presets/chain/stablenet-bp4-en1.json")
	if err != nil {
		t.Fatal(err)
	}
	preset, err := e.documents.SaveDocument(a, "", 0, DeploymentDocumentInput{Kind: "chain-preset", Name: "stablenet-bp4-en1", ContractVersion: "2", Content: raw})
	if err != nil {
		t.Fatalf("corpus chain preset rejected: %v", err)
	}
	content := json.RawMessage(`{"schemaVersion":"2","kind":"case","id":"extends-preset","chainPreset":{"extends":"stablenet-bp4-en1","binaries":{"default":"gstable"}},"steps":[{"expect":"blockNumber","is":0}]}`)
	in := DeploymentDocumentInput{Kind: "case", Name: "extends-preset", ContractVersion: "2", Content: content,
		PresetRefs: []DeploymentDocumentRef{{ID: preset.ID, Revision: preset.Revision}}}
	return e, p, a, preset, in
}

func TestWebCaseSavesWithPinnedPresetRevision(t *testing.T) {
	e, _, a, preset, in := presetCaseFixture(t)
	saved, err := e.documents.SaveDocument(a, "", 0, in)
	if err != nil {
		t.Fatalf("case extending a shared preset revision rejected: %v", err)
	}
	if len(saved.PresetRefs) != 1 || saved.PresetRefs[0] != (DeploymentDocumentRef{ID: preset.ID, Revision: preset.Revision}) {
		t.Fatalf("saved case lost its pinned preset: %+v", saved.PresetRefs)
	}
	if string(saved.Content) != string(in.Content) {
		t.Fatal("saving rewrote the case declaration")
	}
}

func TestWebCaseRefusesMissingWrongAndUnusedPresetRefs(t *testing.T) {
	e, _, a, preset, in := presetCaseFixture(t)
	other, err := e.documents.SaveDocument(a, "", 0, DeploymentDocumentInput{Kind: "chain-preset", Name: "other", ContractVersion: "2",
		Content: json.RawMessage(`{"schemaVersion":"2","kind":"chain-preset","id":"unused-preset","chain":"stablenet","topology":{"bp":4}}`)})
	if err != nil {
		t.Fatal(err)
	}
	serverSet, err := e.documents.SaveDocument(a, "", 0, deploymentTestSet())
	if err != nil {
		t.Fatal(err)
	}
	for name, refs := range map[string][]DeploymentDocumentRef{
		"missing":   nil,
		"wrongKind": {{ID: serverSet.ID, Revision: serverSet.Revision}},
		"unknown":   {{ID: preset.ID, Revision: preset.Revision + 5}},
		"unused":    {{ID: preset.ID, Revision: preset.Revision}, {ID: other.ID, Revision: other.Revision}},
	} {
		next := in
		next.PresetRefs = refs
		if _, err := e.documents.SaveDocument(a, "", 0, next); err == nil {
			t.Errorf("%s preset references accepted", name)
		}
	}
	plain := DeploymentDocumentInput{Kind: "server-set", Name: "x", ContractVersion: "2", Content: serverSet.Content,
		PresetRefs: []DeploymentDocumentRef{{ID: preset.ID, Revision: preset.Revision}}}
	if _, err := e.documents.SaveDocument(a, "", 0, plain); err == nil || !strings.Contains(err.Error(), "preset") {
		t.Errorf("preset references on a non-case document accepted: %v", err)
	}
}

// The job executes the preset revision the case pinned, even after the shared
// preset is edited.
func TestWebTestRunExecutesThePinnedPresetRevision(t *testing.T) {
	e, p, a, preset, in := presetCaseFixture(t)
	saved, err := e.documents.SaveDocument(a, "", 0, in)
	if err != nil {
		t.Fatal(err)
	}
	var edited map[string]any
	if err = json.Unmarshal(preset.Content, &edited); err != nil {
		t.Fatal(err)
	}
	edited["topology"] = map[string]any{"bp": 4, "en": 2}
	raw, _ := json.Marshal(edited)
	if _, err = e.documents.SaveDocument(a, preset.ID, preset.Revision, DeploymentDocumentInput{Kind: "chain-preset", Name: preset.Name, ContractVersion: "2", Content: raw}); err != nil {
		t.Fatal(err)
	}
	p.Arguments.CaseRefs = []DeploymentDocumentRef{{ID: saved.ID, Revision: saved.Revision}}
	cases, err := e.prepareTestCases(context.Background(), "stablenet", p.Arguments.CaseRefs)
	if err != nil {
		t.Fatalf("case with a pinned preset cannot be prepared: %v", err)
	}
	spec, err := dsl.Parse(cases[0].Content)
	if err != nil {
		t.Fatalf("executable content still needs its preset: %v", err)
	}
	if spec.Topology["en"] != float64(1) && spec.Topology["en"] != 1 {
		t.Fatalf("job ran a preset revision the case did not pin: topology %v", spec.Topology)
	}
}
