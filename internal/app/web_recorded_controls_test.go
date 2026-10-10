package app

import (
	"encoding/json"
	"github.com/0xmhha/chainbench/internal/resource"
	"slices"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/node"
)

func TestWebControlRecordRefusesAnotherNodeScope(t *testing.T) {
	for _, name := range []string{"valid", "foreign server", "foreign host", "outside data", "outside config", "outside log", "unknown role", "duplicate index", "duplicate label", "negative PID", "producer count", "old format", "composition traversal"} {
		t.Run(name, func(t *testing.T) {
			state, p := webOwnedControlFixture(t)
			switch name {
			case "foreign server":
				state.Nodes[0].Server = "other"
			case "foreign host":
				state.Nodes[0].Host = "outside.invalid"
			case "outside data":
				state.Nodes[0].DataDir = "/outside"
			case "outside config":
				state.Nodes[0].ConfigPath = "/outside"
			case "outside log":
				state.Nodes[0].LogPath = "/outside"
			case "unknown role":
				state.Nodes[0].Role = "unknown"
			case "duplicate index":
				state.Nodes[1].Index = 1
			case "duplicate label":
				state.Nodes[1].Label = "node1"
			case "negative PID":
				state.Nodes[0].PID = -1
			case "producer count":
				state.BPCount = 2
				p.Arguments.Validators = 2
			case "old format":
				state.FormatVersion = 2
			case "composition traversal":
				state.CompositionID = "../outside"
			}
			err := validateWebChainRecord(state, p)
			if name == "valid" {
				if err != nil {
					t.Fatal(err)
				}
				return
			}
			if err == nil {
				t.Fatal("out-of-scope node record accepted")
			}
		})
	}
}

func TestWebControlPlacementKeepsOccupiedSlotsAndAllRoles(t *testing.T) {
	state, p := webResetFixture(t)
	state.BPCount = 1
	p.Arguments.ServerRef = "ssh"
	wc, err := deploymentWorkspace(p.Config.DeploymentDocumentInput)
	if err != nil {
		t.Fatal(err)
	}
	layout := node.Layout{Root: wc.DataRoot, CompositionID: state.CompositionID, NodesDir: wc.Paths.Nodes, RuntimeDir: wc.Paths.Runtime, LogsDir: wc.Paths.Logs}
	state.Nodes[0].Host, state.Nodes[0].LogPath = "localhost.", layout.LogPath("node1")
	second := state.Nodes[0]
	second.Index, second.Role = 2, "bp"
	second.DataDir, second.ConfigPath, second.LogPath = layout.DataDir("node2"), layout.ConfigPath("node2"), layout.LogPath("node2")
	pool := resource.Pool{Hosts: []resource.Host{{Name: "ssh", Addr: "localhost."}}, Slots: 4, Ports: resource.Bands{P2P: resource.Band{Base: 31000, Step: 10}, RPC: resource.Band{Base: 8600, Step: 10}}, Reservation: node.Reservation{P2PSpan: 3, RPCSpan: 4}}
	state.Nodes[0].Endpoints, err = resource.PlanBands(3, pool.Ports, pool.Reservation)
	if err != nil {
		t.Fatal(err)
	}
	second.Endpoints, err = resource.PlanBands(4, pool.Ports, pool.Reservation)
	if err != nil {
		t.Fatal(err)
	}
	state.Nodes = append(state.Nodes, second)
	slices.Reverse(state.Nodes)
	before := slices.Clone(state.Nodes)
	placement, err := webRecordedControlPlacement(state, &p, pool)
	if err != nil {
		t.Fatal(err)
	}
	nodes := placement.Placements()
	if p.Arguments.Validators != 1 || len(nodes) != 2 || nodes[0].Role != node.RoleEN || nodes[0].Ports != state.Nodes[1].Endpoints || nodes[1].Ports != second.Endpoints {
		t.Fatal("control allocated new slots or dropped a non-producer", nodes)
	}
	for i := range state.Nodes {
		if state.Nodes[i].Index != before[i].Index {
			t.Fatal("review mutated owned node order")
		}
	}
	for _, name := range []string{"outside port", "duplicate ports", "missing port", "explicit wrong count"} {
		t.Run(name, func(t *testing.T) {
			changed := state
			changed.Nodes = slices.Clone(state.Nodes)
			payload := p
			switch name {
			case "outside port":
				changed.Nodes[0].HTTP = 20000
			case "duplicate ports":
				changed.Nodes[0].Endpoints = changed.Nodes[1].Endpoints
			case "missing port":
				changed.Nodes[0].EtcdClient = 0
			case "explicit wrong count":
				payload.Arguments.Validators = 4
			}
			if _, err := webRecordedControlPlacement(changed, &payload, pool); err == nil {
				t.Fatal("unreviewed placement accepted")
			}
		})
	}
}

func webOwnedControlFixture(t *testing.T) (State, webChainPayload) {
	t.Helper()
	state, p := webResetFixture(t)
	state.Chain, state.Binary, state.BPCount = "wbft", "/registered/binary", 1
	p.Manifest.Manifest = json.RawMessage(`{"id":"wbft"}`)
	p.ExecutionBinary = state.Binary
	p.Arguments.Validators, p.Arguments.ServerRef = 1, "ssh"
	state.Nodes[0].Host = "localhost."
	wc, err := deploymentWorkspace(p.Config.DeploymentDocumentInput)
	if err != nil {
		t.Fatal(err)
	}
	layout := node.Layout{Root: wc.DataRoot, CompositionID: state.CompositionID, NodesDir: wc.Paths.Nodes, RuntimeDir: wc.Paths.Runtime, LogsDir: wc.Paths.Logs}
	state.Nodes[0].LogPath = layout.LogPath("node1")
	second := state.Nodes[0]
	second.Index, second.Role = 2, "bp"
	second.DataDir, second.ConfigPath, second.LogPath = layout.DataDir("node2"), layout.ConfigPath("node2"), layout.LogPath("node2")
	state.Nodes = append(state.Nodes, second)
	return state, p
}
