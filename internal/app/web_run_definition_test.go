package app

import (
	"encoding/json"
	"strings"
	"testing"
)

func runDefinitionCase(id string) DeploymentDocumentInput {
	return DeploymentDocumentInput{Kind: "case", Name: id, ContractVersion: "2", AssetRefs: []string{}, Content: json.RawMessage(`{"schemaVersion":"2","kind":"case","id":"` + id + `","chainPreset":{"chain":"stablenet","topology":{"bp":1}},"steps":[{"expect":"blockNumber","is":0}]}`)}
}

func TestRunDefinitionsPinTheirCasesInOrder(t *testing.T) {
	store, err := OpenDeploymentStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "operator", Role: "operator"}
	first, err := store.SaveDocument(a, "", 0, runDefinitionCase("first"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.SaveDocument(a, "", 0, runDefinitionCase("second"))
	if err != nil {
		t.Fatal(err)
	}
	def := func(content string, refs ...DeploymentDocumentRef) DeploymentDocumentInput {
		return DeploymentDocumentInput{Kind: "run-definition", Name: "suite", ContractVersion: "2", AssetRefs: []string{}, Content: json.RawMessage(content), CaseRefs: refs}
	}
	saved, err := store.SaveDocument(a, "", 0, def(`{"cases":["second","first"]}`, DeploymentDocumentRef{second.ID, 1}, DeploymentDocumentRef{first.ID, 1}))
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.CaseRefs) != 2 || saved.CaseRefs[0].ID != second.ID {
		t.Fatalf("pinned %+v", saved.CaseRefs)
	}
	for content, refs := range map[string][]DeploymentDocumentRef{
		`{"cases":["first","second"]}`: {{second.ID, 1}, {first.ID, 1}},
		`{"cases":["first"]}`:          {{first.ID, 1}, {second.ID, 1}},
		`{"cases":[]}`:                 {},
		`{"cases":["first","first"]}`:  {{first.ID, 1}, {first.ID, 1}},
		`{"cases":["first"],"x":1}`:    {{first.ID, 1}},
	} {
		if _, err := store.SaveDocument(a, "", 0, def(content, refs...)); err == nil {
			t.Errorf("accepted %s with %v", content, refs)
		}
	}
	set, _ := store.SaveDocument(a, "", 0, deploymentTestSet())
	if _, err := store.SaveDocument(a, "", 0, def(`{"cases":["first"]}`, DeploymentDocumentRef{set.ID, 1})); err == nil {
		t.Error("a server set was accepted as a case")
	}
	if _, err := store.SaveDocument(a, "", 0, DeploymentDocumentInput{Kind: "case", Name: "x", ContractVersion: "2", Content: runDefinitionCase("x").Content, CaseRefs: []DeploymentDocumentRef{{first.ID, 1}}}); err == nil {
		t.Error("a case carried case references")
	}
}

func TestRunDefinitionBundlesCarryTheirCases(t *testing.T) {
	store, err := OpenDeploymentStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "operator", Role: "operator"}
	first, _ := store.SaveDocument(a, "", 0, runDefinitionCase("first"))
	second, _ := store.SaveDocument(a, "", 0, runDefinitionCase("second"))
	saved, err := store.SaveDocument(a, "", 0, DeploymentDocumentInput{Kind: "run-definition", Name: "suite", ContractVersion: "2", AssetRefs: []string{}, Content: json.RawMessage(`{"cases":["first","second"]}`), CaseRefs: []DeploymentDocumentRef{{first.ID, 1}, {second.ID, 1}}})
	if err != nil {
		t.Fatal(err)
	}
	exported, err := store.ExportDocumentBundle(a, saved.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(exported), `"id": "first"`) || !strings.Contains(string(exported), `"id": "second"`) {
		t.Fatalf("bundle lacks its cases: %s", exported)
	}
	other, err := OpenDeploymentStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	preview, err := other.PreviewImport(a, DocumentImportInput{Filename: "suite.json", Format: "json", Source: string(exported)})
	if err != nil || !preview.Validation.Valid {
		t.Fatalf("%+v %v", preview, err)
	}
	committed, err := other.CommitImport(a, DocumentImportCommit{PreviewID: preview.PreviewID})
	if err != nil {
		t.Fatal(err)
	}
	var def DeploymentDocument
	ids := map[string]string{}
	for _, d := range committed {
		if d.Kind == "run-definition" {
			def = d
		}
		var head struct{ ID string }
		_ = json.Unmarshal(d.Content, &head)
		ids[d.ID] = head.ID
	}
	if len(def.CaseRefs) != 2 || ids[def.CaseRefs[0].ID] != "first" || ids[def.CaseRefs[1].ID] != "second" {
		t.Fatalf("imported run definition does not point at the imported cases: %+v", def.CaseRefs)
	}
}

func TestTestJobsRunARunDefinitionsCasesInOrder(t *testing.T) {
	store, err := OpenDeploymentStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a := DeploymentActor{ID: "operator", Role: "operator"}
	first, _ := store.SaveDocument(a, "", 0, runDefinitionCase("first"))
	second, _ := store.SaveDocument(a, "", 0, runDefinitionCase("second"))
	def, err := store.SaveDocument(a, "", 0, DeploymentDocumentInput{Kind: "run-definition", Name: "suite", ContractVersion: "2", AssetRefs: []string{}, Content: json.RawMessage(`{"cases":["second","first"]}`), CaseRefs: []DeploymentDocumentRef{{second.ID, 1}, {first.ID, 1}}})
	if err != nil {
		t.Fatal(err)
	}
	e := &WebChainEngine{documents: store}
	args := webChainArguments{RunDefinitionRef: &DeploymentDocumentRef{def.ID, def.Revision}}
	if err = e.resolveRunDefinition(&args, "test.run"); err != nil {
		t.Fatal(err)
	}
	if len(args.CaseRefs) != 2 || args.CaseRefs[0].ID != second.ID || args.CaseRefs[1].ID != first.ID {
		t.Fatalf("case order %+v", args.CaseRefs)
	}
	for _, bad := range []struct {
		args webChainArguments
		op   string
	}{
		{webChainArguments{RunDefinitionRef: &DeploymentDocumentRef{def.ID, def.Revision}}, "chain.deploy"},
		{webChainArguments{RunDefinitionRef: &DeploymentDocumentRef{def.ID, def.Revision}, CaseRefs: []DeploymentDocumentRef{{first.ID, 1}}}, "test.run"},
		{webChainArguments{RunDefinitionRef: &DeploymentDocumentRef{first.ID, 1}}, "test.run"},
	} {
		if err := e.resolveRunDefinition(&bad.args, bad.op); err == nil {
			t.Errorf("accepted %+v for %s", bad.args, bad.op)
		}
	}
}
