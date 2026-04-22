package state

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

// Profile is the subset of profile YAML fields the state package needs.
// Extend as more handlers require more data.
type Profile struct {
	Name  string     `yaml:"name"`
	Chain ChainBlock `yaml:"chain"`
	Nodes NodesBlock `yaml:"nodes"`
	Ports PortsBlock `yaml:"ports"`
}

// ChainBlock mirrors the "chain:" section of a profile.
type ChainBlock struct {
	Binary     string `yaml:"binary"`
	BinaryPath string `yaml:"binary_path"`
	ChainID    int64  `yaml:"chain_id"`
	NetworkID  int64  `yaml:"network_id"`
	Type       string `yaml:"type"` // optional; default applied at consumption time
}

// NodesBlock mirrors the "nodes:" section.
type NodesBlock struct {
	Validators int `yaml:"validators"`
	Endpoints  int `yaml:"endpoints"`
}

// PortsBlock mirrors the "ports:" section.
type PortsBlock struct {
	BaseP2P  int `yaml:"base_p2p"`
	BaseHTTP int `yaml:"base_http"`
	BaseWS   int `yaml:"base_ws"`
}

// ParseProfile decodes a profile YAML from r into a Profile. Missing fields
// get zero values; consumers apply defaults.
func ParseProfile(r io.Reader) (*Profile, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("state: read profile: %w", err)
	}
	var p Profile
	if len(data) == 0 {
		return &p, nil
	}
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("state: parse profile: %w", err)
	}
	return &p, nil
}

// ReadProfileFile opens path and delegates to ParseProfile.
func ReadProfileFile(path string) (*Profile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("state: open profile: %w", err)
	}
	defer f.Close()
	return ParseProfile(f)
}
