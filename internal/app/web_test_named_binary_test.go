package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// namedBinaryFixture saves a case that runs two builds: the network's default
// and a successor named "next" of another chain, crossing a fork between them.
func namedBinaryFixture(t *testing.T) (*WebChainEngine, webChainPayload) {
	t.Helper()
	e, p := webRunPlanFixture(t, `{"bp":4}`)
	a := DeploymentActor{ID: "operator", Role: "operator"}
	content := json.RawMessage(`{"schemaVersion":"2","kind":"case","id":"crossing","chainPreset":{"chain":"wemix",
		"binaries":{"default":"gwemix","next":{"binary":"gwbft","chain":"wbft"}},
		"upgrade":{"fork":"croissant","at":20,"from":"default","to":"next","style":"concurrent"},
		"topology":{"nodes":[{"index":1,"role":"en","binary":"next"},{"index":2,"role":"en","binary":"next"},{"index":3,"role":"en","binary":"next"},{"index":4,"role":"en","binary":"next"},{"index":5,"role":"bp"}]}},
		"steps":[{"do":"crossFork","timeout":"300s"},{"expect":"blockNumber","compare":"Greater","is":"20"}]}`)
	saved, err := e.documents.SaveDocument(a, "", 0, DeploymentDocumentInput{Kind: "case", Name: "crossing", ContractVersion: "2", Content: content})
	if err != nil {
		t.Fatalf("crossing case rejected: %v", err)
	}
	p.Arguments.CaseRefs = []DeploymentDocumentRef{{ID: saved.ID, Revision: saved.Revision}}
	p.Binary.Chain = "wemix"
	for id, chain := range map[string]string{"wbft-build": "wbft", "stable-build": "stablenet"} {
		path := filepath.Join(e.root, id)
		if err := os.WriteFile(path, []byte(id), 0o755); err != nil {
			t.Fatal(err)
		}
		e.assets[id] = ManifestBinary{ID: id, Chain: chain, Path: path, SHA256: manifestHash([]byte(id))}
	}
	e.verifyAsset = func(_ context.Context, asset ManifestBinary, chain string) (ManifestBinaryEvidence, error) {
		if asset.Chain != chain {
			return ManifestBinaryEvidence{}, errIncompatibleBinary
		}
		return ManifestBinaryEvidence{ID: asset.ID, Chain: chain, SHA256: asset.SHA256, OS: p.Binary.OS, Architecture: p.Binary.Architecture}, nil
	}
	return e, p
}

func TestWebTestRunMapsEachNamedBinaryToAVerifiedAsset(t *testing.T) {
	e, p := namedBinaryFixture(t)
	p.Arguments.BinaryAssets = map[string]string{"next": "wbft-build"}
	if _, err := e.prepareTestRun(context.Background(), &p); err != nil {
		t.Fatalf("a case crossing to a registered successor cannot be planned: %v", err)
	}
	if len(p.NamedBinaries) != 1 || p.NamedBinaries[0].Name != "next" || p.NamedBinaries[0].Binary.Chain != "wbft" {
		t.Fatalf("named binary not pinned: %+v", p.NamedBinaries)
	}
	in := e.webSuiteInput(p)
	if in.BinaryOverrides["next"] != p.NamedBinaries[0].ExecutionBinary || !strings.Contains(in.BinaryOverrides["next"], p.NamedBinaries[0].Binary.SHA256) {
		t.Fatalf("the engine would not run the verified successor: %v", in.BinaryOverrides)
	}
}

func TestWebTestRunRefusesMissingWrongChainAndUndeclaredNamedBinaries(t *testing.T) {
	for name, assets := range map[string]map[string]string{
		"missing":    nil,
		"wrongChain": {"next": "stable-build"},
		"undeclared": {"next": "wbft-build", "other": "wbft-build"},
		"unknown":    {"next": "no-such-asset"},
	} {
		e, p := namedBinaryFixture(t)
		p.Arguments.BinaryAssets = assets
		if _, err := e.prepareTestRun(context.Background(), &p); err == nil {
			t.Errorf("%s named binary selection accepted", name)
		}
	}
}
