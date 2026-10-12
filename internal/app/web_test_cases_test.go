package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func savedWebCase(t *testing.T, engine *WebChainEngine) (DeploymentActor, DeploymentDocument) {
	t.Helper()
	actor := DeploymentActor{ID: "operator", Role: "operator"}
	document, err := engine.documents.SaveDocument(actor, "", 0, DeploymentDocumentInput{Kind: "case", Name: "Browser case", ContractVersion: "2", Content: editorCase(`[{"do":"read","source":"blockNumber","on":"node1","save":"head"},{"expect":"blockNumber","onEach":["node1","node2"],"compare":"GreaterOrEqual","is":"$head"}]`)})
	if err != nil {
		t.Fatal(err)
	}
	return actor, document
}

func TestWebTestCasesPinExecutableMeaningAndRevision(t *testing.T) {
	e, _ := keySnapshotEngine(t)
	actor, document := savedWebCase(t, e)
	refs := []DeploymentDocumentRef{{document.ID, document.Revision}}
	cases, err := e.prepareTestCases(context.Background(), "stablenet", refs)
	if err != nil || len(cases) != 1 {
		t.Fatalf("stored DSL cannot be selected for execution: %d %v", len(cases), err)
	}
	if cases[0].Document.ID != document.ID || cases[0].Document.Revision != 1 {
		t.Fatal("case identity/revision was lost")
	}
	expected, err := PrepareTestCase(TestCaseInput{Content: document.Content})
	if err != nil {
		t.Fatal(err)
	}
	actual, err := PrepareTestCase(TestCaseInput{Content: cases[0].Content})
	if err != nil || actual.SemanticFingerprint != expected.SemanticFingerprint {
		t.Fatal("selected case executable meaning changed")
	}
	updated := document.DeploymentDocumentInput
	updated.Content = json.RawMessage(strings.Replace(string(updated.Content), `"GreaterOrEqual"`, `"Equal"`, 1))
	if _, err = e.documents.SaveDocument(actor, document.ID, document.Revision, updated); err != nil {
		t.Fatal(err)
	}
	if _, err = e.prepareTestCases(context.Background(), "stablenet", refs); !errors.Is(err, ErrDeploymentConflict) {
		t.Fatalf("stale case revision accepted: %v", err)
	}
	if string(cases[0].Document.Content) != string(document.Content) {
		t.Fatal("editing the shared case changed an accepted copy")
	}
}

func TestWebTestCasesRejectMissingDuplicateWrongKindAndChain(t *testing.T) {
	e, _ := keySnapshotEngine(t)
	actor, document := savedWebCase(t, e)
	other, err := e.documents.SaveDocument(actor, "", 0, deploymentTestSet())
	if err != nil {
		t.Fatal(err)
	}
	for name, refs := range map[string][]DeploymentDocumentRef{
		"empty":           nil,
		"missing":         {{"unknown", 1}},
		"wrong-kind":      {{other.ID, other.Revision}},
		"duplicate":       {{document.ID, document.Revision}, {document.ID, document.Revision}},
		"implicit-latest": {{document.ID, 0}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := e.prepareTestCases(context.Background(), "stablenet", refs); err == nil {
				t.Fatal("invalid case selection accepted")
			}
		})
	}
	if _, err = e.prepareTestCases(context.Background(), "wbft", []DeploymentDocumentRef{{document.ID, document.Revision}}); err == nil {
		t.Fatal("case for a different chain accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err = e.prepareTestCases(ctx, "stablenet", []DeploymentDocumentRef{{document.ID, document.Revision}}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled case preparation accepted: %v", err)
	}
}
