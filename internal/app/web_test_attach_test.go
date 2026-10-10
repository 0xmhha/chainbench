package app

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/dsl"
)

func attachCase(accounts string) json.RawMessage {
	return json.RawMessage(`{"schemaVersion":"2","kind":"case","id":"attached","requires":["rpc"],"chainPreset":{"chain":"stablenet","attach":{"rpc":["http://203.0.113.9:8545"],"keysDir":"presets/keys","provides":["rpc","consensus"]}` + accounts + `},"steps":[{"expect":"blockNumber","compare":"GreaterOrEqual","is":0}]}`)
}

func attachFixture(t *testing.T, content json.RawMessage) (*WebChainEngine, webChainPayload, State) {
	t.Helper()
	e, p := webRunPlanFixture(t, `{"bp":4}`)
	saved, err := e.documents.SaveDocument(DeploymentActor{ID: "operator", Role: "operator"}, "", 0, DeploymentDocumentInput{Kind: "case", Name: "attached", ContractVersion: "2", Content: content})
	if err != nil {
		t.Fatalf("attach case rejected: %v", err)
	}
	p.Arguments.CaseRefs = []DeploymentDocumentRef{{ID: saved.ID, Revision: saved.Revision}}
	state := State{Chain: "stablenet", Capabilities: []string{"rpc", "ws"}, Nodes: []node.Record{
		{Index: 1, Role: "bp", Endpoints: node.Endpoints{HTTP: 8601, WS: 8701}},
		{Index: 2, Role: "en", Host: "198.51.100.4", Endpoints: node.Endpoints{HTTP: 8602}},
	}}
	return e, p, state
}

// The endpoints, key set and capabilities come from the network this service
// composed and recorded; what the case wrote for some other network is not
// dialed.
func TestWebAttachTargetsTheRecordedNetwork(t *testing.T) {
	e, p, state := attachFixture(t, attachCase(""))
	if err := e.prepareTestAttach(context.Background(), DeploymentActor{ID: "operator", Role: "operator"}, &p, state); err != nil {
		t.Fatalf("attach case against a recorded network refused: %v", err)
	}
	got := p.Attach.Nodes
	if len(got.Nodes) != 2 || got.Nodes[0].RPCURL != "http://127.0.0.1:8601" || got.Nodes[0].WSURL != "ws://127.0.0.1:8701" || got.Nodes[1].RPCURL != "http://198.51.100.4:8602" {
		t.Fatalf("attach does not dial the recorded nodes: %+v", got.Nodes)
	}
	if got.Nodes[1].Role != "en" || strings.Join(got.Capabilities, ",") != "rpc,ws" {
		t.Fatalf("attach lost the recorded roles or capabilities: %+v", got)
	}
	if len(p.Attach.Cases) != 1 || len(p.Attach.Cases[0].Content) == 0 {
		t.Fatal("reviewed case content missing")
	}
}

func TestWebAttachRefusesComposedCasesAndHostKeyFiles(t *testing.T) {
	e, p, state := attachFixture(t, attachCase(""))
	composed, err := e.documents.SaveDocument(DeploymentActor{ID: "operator", Role: "operator"}, "", 0, DeploymentDocumentInput{Kind: "case", Name: "composed", ContractVersion: "2",
		Content: editorCase(`[{"expect":"blockNumber","compare":"GreaterOrEqual","is":0}]`)})
	if err != nil {
		t.Fatal(err)
	}
	p.Arguments.CaseRefs = append(p.Arguments.CaseRefs, DeploymentDocumentRef{ID: composed.ID, Revision: composed.Revision})
	if err := e.prepareTestAttach(context.Background(), DeploymentActor{ID: "operator", Role: "operator"}, &p, state); err == nil || !strings.Contains(err.Error(), "attach") {
		t.Fatalf("a composing case ran as an attach job: %v", err)
	}
	e, p, state = attachFixture(t, attachCase(`,"accounts":{"payer":{"keyFile":"/etc/secret-key"}}`))
	if err := e.prepareTestAttach(context.Background(), DeploymentActor{ID: "operator", Role: "operator"}, &p, state); err == nil || !strings.Contains(err.Error(), "credential") {
		t.Fatalf("a host key file path reached the attach run: %v", err)
	}
}

// A composed Web run mints and funds its declared accounts in memory over the
// accepted key set; a key file belongs to an attach job's private credentials.
func TestWebComposedRunAcceptsMintedAccountsOnly(t *testing.T) {
	minted, err := dsl.Parse(editorCaseWith(`,"accounts":{"dev1":{"fund":"1000"},"dev2":{}}`))
	if err != nil {
		t.Fatal(err)
	}
	if err = validateWebTestInputs(minted); err != nil {
		t.Fatalf("minted declared accounts refused: %v", err)
	}
	keyed, err := dsl.Parse(editorCaseWith(`,"accounts":{"payer":{"keyFile":"/etc/secret-key"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if err = validateWebTestInputs(keyed); err == nil || !strings.Contains(err.Error(), "attach") {
		t.Fatalf("a server key file reached a composed Web run: %v", err)
	}
}

func editorCaseWith(accounts string) []byte {
	return []byte(`{"schemaVersion":"2","kind":"case","id":"editor","chainPreset":{"chain":"stablenet","binaries":{"default":"gstable"},"topology":{"bp":4}` + accounts + `},"steps":[{"expect":"blockNumber","compare":"GreaterOrEqual","is":0}]}`)
}
