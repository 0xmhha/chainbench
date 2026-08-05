package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/driver"
)

// nodeSpecsFile is the file name under the data root that holds the launched
// node specs (the fully-armed launch args, binary, datadir, config) needed to
// relaunch a single node after it is stopped. The NodeSet (nodeset.json) records
// the running endpoints; the specs record how to bring a node back.
const nodeSpecsFile = "nodespecs.json"

// SaveNodeSpecs writes the launched node specs to <dir>/nodespecs.json so a
// later `node start --index` can relaunch one node from its exact spec.
func SaveNodeSpecs(dir string, specs []driver.NodeSpec) error {
	b, err := json.MarshalIndent(specs, "", "  ")
	if err != nil {
		return fmt.Errorf("state: marshal node specs: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, nodeSpecsFile), b, 0o644); err != nil {
		return fmt.Errorf("state: write node specs: %w", err)
	}
	return nil
}

// LoadNodeSpecs reads the launched node specs from <dir>/nodespecs.json.
func LoadNodeSpecs(dir string) ([]driver.NodeSpec, error) {
	b, err := os.ReadFile(filepath.Join(dir, nodeSpecsFile))
	if err != nil {
		return nil, fmt.Errorf("state: read node specs: %w", err)
	}
	var specs []driver.NodeSpec
	if err := json.Unmarshal(b, &specs); err != nil {
		return nil, fmt.Errorf("state: parse node specs: %w", err)
	}
	return specs, nil
}
