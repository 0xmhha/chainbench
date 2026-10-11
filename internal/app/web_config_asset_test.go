package app

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

func TestRegisteredConfigFilesBecomeReviewedConfigChanges(t *testing.T) {
	got, err := webConfigAssetOverrides("node-config.json", []byte(`{"syncMode":"snap","httpHost":"10.1.2.3"}`))
	if err != nil || got["syncMode"] != "snap" || got["httpHost"] != "10.1.2.3" || len(got) != 2 {
		t.Fatalf("%v %v", got, err)
	}
	if got, err := webConfigAssetOverrides("node-config.yaml", []byte("metricsHost: 127.0.0.1\n")); err != nil || got["metricsHost"] != "127.0.0.1" {
		t.Fatalf("yaml: %v %v", got, err)
	}
	for name, content := range map[string]string{
		"a.json": `{"syncMode":"light"}`,
		"b.json": `{"httpHost":"not-an-address"}`,
		"c.json": `{"maxPeers":"50"}`,
		"d.json": `{}`,
		"e.json": `["syncMode"]`,
		"f.json": `{"syncMode":4}`,
	} {
		if _, err := webConfigAssetOverrides(name, []byte(content)); err == nil {
			t.Errorf("%s accepted %s", name, content)
		}
	}
}

func TestConfigChangesComeFromThePinnedRegisteredFile(t *testing.T) {
	state := State{Nodes: nil}
	p := webChainPayload{ConfigAsset: &webConfigAsset{Overrides: map[string]string{"httpHost": "10.1.2.3"}}}
	p.Input.NodeIDs = []string{"node1"}
	state.Nodes = append(state.Nodes, webConfigTestNode())
	changes, err := webConfigChanges(state, p)
	if err != nil || strings.Join(changes, ",") != "httpHost=10.1.2.3" {
		t.Fatalf("%v %v", changes, err)
	}
	p.Arguments.ConfigOverrides = map[string]string{"syncMode": "snap"}
	if _, err := webConfigChanges(state, p); err == nil {
		t.Fatal("a registered file and typed choices were combined")
	}
}

func webConfigTestNode() node.Record {
	return node.Record{Index: 1, Label: "node1", Role: "en"}
}
