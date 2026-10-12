package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/0xmhha/chainbench/internal/resource"
)

// ChainCLI is the chainbench command line as the Web uses it. The dashboard
// runs the CLI for every network it reads or changes, so the Web and a
// terminal see and do the same thing; the implementation that starts the
// process lives above app, in internal/webcli.
type ChainCLI interface {
	// ChainStatus is `chainbench chain status --json` for one workspace.
	ChainStatus(ctx context.Context, dir string) (State, error)
	// Run runs one chainbench command and returns what it printed on stdout.
	// A command that exits non-zero returns its stdout too, since `--json`
	// documents are printed on failure as well.
	Run(ctx context.Context, args ...string) ([]byte, error)
}

// errNoChainCLI refuses network work on a dashboard started without the CLI.
var errNoChainCLI = errors.New("the chainbench CLI is not configured for this dashboard")

// UseCLI gives the engine the chainbench command it runs.
func (e *WebChainEngine) UseCLI(c ChainCLI) { e.cli = c }

type WebNode struct {
	ID                string   `json:"id"`
	NetworkID         string   `json:"networkId"`
	Role              string   `json:"role"`
	HostIdentity      string   `json:"hostIdentity"`
	DataPath          string   `json:"dataPath"`
	PID               int      `json:"pid"`
	ObservedPID       int      `json:"observedPid,omitempty"`
	ObservationReason string   `json:"observationReason,omitempty"`
	BinaryAssetID     string   `json:"binaryAssetId,omitempty"`
	BinarySHA256      string   `json:"binarySHA256,omitempty"`
	State             string   `json:"state"`
	SupportedControls []string `json:"supportedControls"`
	// ControlLimits gives, for each node control not offered, why it is refused.
	ControlLimits map[string]string `json:"controlLimits"`
	ObservedAt    time.Time         `json:"observedAt"`
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
		if _, err := os.Stat(filepath.Join(dir, "chain-record.json")); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return nil, err
		}
		if e.cli == nil {
			return nil, errNoChainCLI
		}
		st, err := e.cli.ChainStatus(ctx, dir)
		if err != nil {
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
			n := WebNode{ID: string(ns.NodeLabel()), NetworkID: w.ID, Role: ns.Role, HostIdentity: target.HostIdentity, DataPath: ns.DataDir, PID: ns.PID, State: state, SupportedControls: controls, ObservedAt: time.Now().UTC()}
			n.ControlLimits = webControlLimits(n, network.Ownership, ns.Config != "")
			network.Nodes = append(network.Nodes, n)
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
