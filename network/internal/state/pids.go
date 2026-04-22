package state

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// PIDsFile mirrors the shape of state/pids.json.
type PIDsFile struct {
	ChainID   string              `json:"chain_id"`
	Profile   string              `json:"profile"`
	StartedAt string              `json:"started_at"`
	Nodes     map[string]NodeInfo `json:"nodes"`
}

// NodeInfo mirrors a single entry under state/pids.json ".nodes".
// Fields not needed by the Network mapping are intentionally omitted
// (e.g., saved_args).
type NodeInfo struct {
	PID         int    `json:"pid"`
	Type        string `json:"type"`
	P2PPort     int    `json:"p2p_port"`
	HTTPPort    int    `json:"http_port"`
	WSPort      int    `json:"ws_port"`
	AuthPort    int    `json:"auth_port"`
	MetricsPort int    `json:"metrics_port"`
	Status      string `json:"status"`
	LogFile     string `json:"log_file"`
	Binary      string `json:"binary"`
	Datadir     string `json:"datadir"`
}

// ParsePIDs decodes state/pids.json from r.
func ParsePIDs(r io.Reader) (*PIDsFile, error) {
	dec := json.NewDecoder(r)
	var p PIDsFile
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("state: parse pids.json: %w", err)
	}
	if p.Nodes == nil {
		p.Nodes = map[string]NodeInfo{}
	}
	return &p, nil
}

// ReadPIDsFile opens path and delegates to ParsePIDs.
func ReadPIDsFile(path string) (*PIDsFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("state: open pids.json: %w", err)
	}
	defer f.Close()
	return ParsePIDs(f)
}
