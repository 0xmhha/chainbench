package app

import (
	"context"
	"errors"
	"github.com/0xmhha/chainbench/internal/core/node"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestChainPresetConfigContractHasEngineChoices(t *testing.T) {
	contract, err := ChainPresetContract("wbft")
	if err != nil {
		t.Fatal(err)
	}
	defs := contract["$defs"].(map[string]any)
	props := defs["envSpec"].(map[string]any)["properties"].(map[string]any)
	config := props["config"].(map[string]any)["additionalProperties"].(map[string]any)["properties"].(map[string]any)
	sync := config["syncMode"].(map[string]any)
	if !reflect.DeepEqual(sync["enum"], []string{"full", "snap", "archive"}) {
		t.Fatal("configuration editor lacks authoritative sync choices", sync)
	}
}

func TestWebConfigChangesRefuseUnreviewedChoicesAndLaunchConflicts(t *testing.T) {
	base := State{Nodes: []node.Record{{Index: 1, Label: "node1", Role: "en"}}}
	valid := webChainPayload{Input: WebPlanInput{NodeIDs: []string{"node1"}}, Arguments: webChainArguments{ConfigOverrides: map[string]string{"metricsHost": "127.0.0.1", "syncMode": "archive"}}}
	got, err := webConfigChanges(base, valid)
	if err != nil || !reflect.DeepEqual(got, []string{"metricsHost=127.0.0.1", "syncMode=archive"}) {
		t.Fatal("supported patch lost its deterministic values", got, err)
	}
	for _, kind := range []string{"empty", "unknown-key", "unknown-value", "multiple-nodes", "pinned-config", "named-binary", "command", "all", "en", "node1"} {
		t.Run(kind, func(t *testing.T) {
			state := State{Nodes: append([]node.Record{}, base.Nodes...), LaunchSet: map[string][]string{}}
			p := valid
			p.Arguments.ConfigOverrides = map[string]string{"metricsHost": "127.0.0.1"}
			switch kind {
			case "empty":
				p.Arguments.ConfigOverrides = nil
			case "unknown-key":
				p.Arguments.ConfigOverrides = map[string]string{"arbitrary": "127.0.0.1"}
			case "unknown-value":
				p.Arguments.ConfigOverrides["metricsHost"] = "outside.invalid"
			case "multiple-nodes":
				p.Input.NodeIDs = []string{"node1", "node2"}
			case "pinned-config":
				state.Nodes[0].Config = "asset"
			case "named-binary":
				state.Nodes[0].Binary = "upgrade"
			case "command":
				state.LaunchCommand = []string{"metrics.addr=0.0.0.0"}
			default:
				state.LaunchSet[kind] = []string{"metrics.addr=0.0.0.0"}
			}
			if _, err := webConfigChanges(state, p); err == nil {
				t.Fatal("unreviewed config reached execution", kind)
			}
		})
	}
	base.LaunchSet = map[string][]string{"node2": {"metrics.addr=0.0.0.0"}}
	if _, err = webConfigChanges(base, valid); err != nil {
		t.Fatal("sibling override blocks selected node", err)
	}
}

func TestWebConfigKeyBindingUsesAcceptedMaterialAndRefusesTampering(t *testing.T) {
	for _, kind := range []string{"source-change", "outside-path", "material-change", "ciphertext-change"} {
		t.Run(kind, func(t *testing.T) {
			e, source := keySnapshotEngine(t)
			snapshot, err := e.pinKeys(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			dir, err := e.materializeKeys(context.Background(), snapshot)
			if err != nil {
				t.Fatal(err)
			}
			state := State{KeysDir: dir}
			switch kind {
			case "source-change":
				err = os.WriteFile(filepath.Join(source, "nodekey"), []byte("new source input"), 0600)
			case "outside-path":
				state.KeysDir = source
			case "material-change":
				err = os.WriteFile(filepath.Join(dir, "nodekey"), []byte("changed accepted material"), 0600)
			case "ciphertext-change":
				err = os.WriteFile(filepath.Join(e.root, "key-snapshots", snapshot.SHA256+".enc"), []byte("invalid encrypted input"), 0600)
			}
			if err != nil {
				t.Fatal(err)
			}
			got, err := e.bindWebConfigKeys(context.Background(), state)
			if kind == "source-change" {
				if err != nil || got != snapshot {
					t.Fatal("source edits replaced accepted keys", got, err)
				}
			} else if !errors.Is(err, ErrDeploymentConflict) {
				t.Fatal("tampered config-rendering input accepted", err)
			}
		})
	}
}
