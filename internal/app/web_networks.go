package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/0xmhha/chainbench/internal/resource"
)

type WebNode struct {
	ID                string    `json:"id"`
	NetworkID         string    `json:"networkId"`
	Role              string    `json:"role"`
	HostIdentity      string    `json:"hostIdentity"`
	DataPath          string    `json:"dataPath"`
	PID               int       `json:"pid"`
	ObservedPID       int       `json:"observedPid,omitempty"`
	ObservationReason string    `json:"observationReason,omitempty"`
	BinaryAssetID     string    `json:"binaryAssetId,omitempty"`
	BinarySHA256      string    `json:"binarySHA256,omitempty"`
	State             string    `json:"state"`
	SupportedControls []string  `json:"supportedControls"`
	ObservedAt        time.Time `json:"observedAt"`
}
type WebNetwork struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspaceId"`
	Ownership   string    `json:"ownership"`
	Nodes       []WebNode `json:"nodes"`
	Version     int       `json:"version"`
}

// Networks returns only compositions in the Web service's owned control tree.
// Recorded PIDs are not liveness proof; callers see unknown until live probing.
func (e *WebChainEngine) Networks(ctx context.Context) ([]WebNetwork, error) {
	networks := []WebNetwork{}
	for _, w := range e.documents.Workspaces() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		dir := filepath.Join(e.root, "networks", w.ID)
		b, err := os.ReadFile(filepath.Join(dir, "chain-record.json"))
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		var st State
		if err = json.Unmarshal(b, &st); err != nil {
			return nil, err
		}
		var target resource.Inspection
		if metadata, err := os.ReadFile(filepath.Join(dir, "web-target.json")); err == nil {
			if err = json.Unmarshal(metadata, &target); err != nil {
				return nil, err
			}
		}
		network := WebNetwork{ID: w.ID, WorkspaceID: w.ID, Ownership: "owned", Version: w.Revision, Nodes: []WebNode{}}
		for _, ns := range st.Nodes {
			controls := []string{"node.start", "node.stop"}
			if ns.Binary == "" {
				controls = append(controls, "node.restart")
			}
			if ns.PID > 0 && ns.Binary == "" && (ns.Role == "en" || ns.Role == "pn") {
				controls = append(controls, "node.reset")
			}
			state := "unknown"
			if ns.PID == 0 || ns.Binary != "" {
				controls = []string{}
			}
			if target.HostIdentity == "" {
				controls = []string{}
			}
			network.Nodes = append(network.Nodes, WebNode{ID: string(ns.NodeLabel()), NetworkID: w.ID, Role: ns.Role, HostIdentity: target.HostIdentity, DataPath: ns.DataDir, PID: ns.PID, State: state, SupportedControls: controls, ObservedAt: time.Now().UTC()})
		}
		networks = append(networks, network)
	}
	return networks, nil
}

func (s *WebJobs) Networks(ctx context.Context) ([]WebNetwork, error) {
	reader, ok := s.engine.(interface {
		Networks(context.Context) ([]WebNetwork, error)
	})
	if !ok {
		return []WebNetwork{}, nil
	}
	return reader.Networks(ctx)
}
