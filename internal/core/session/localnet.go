package session

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// The standing local network is a fourth on-disk lifetime this package owns,
// alongside the per-run session (session.New), the long-lived Composition, and
// (in core/netreg) the machine-wide named registry. It is the record a
// `chainbench setup --launch` writes into its own --data-dir so later commands
// (verify, test, node stop/start, hardfork, stop, clean) address that running
// network without re-specifying it: nodeset.json holds the running endpoints
// and PIDs, nodespecs.json the fully-armed launch specs needed to relaunch one
// node. It keyed the retired core/state package; the code is sound, only its
// legacy-tree home was (worklist T7.11), so it moves here rather than dying
// with the stack around it.
//
// It is deliberately small: one JSON file per data root, no naming or
// validation (unlike netreg, which is a shared named registry surface). "local"
// is netreg's reserved name for exactly this per-datadir nodeset.

// localNodeSetFile is the running-endpoints file under a launch data root.
const localNodeSetFile = "nodeset.json"

// localNodeSpecsFile holds the launched node specs (fully-armed launch args,
// binary, datadir, config) needed to relaunch a single node after it is
// stopped. nodeset.json records the running endpoints; the specs record how to
// bring a node back.
const localNodeSpecsFile = "nodespecs.json"

// SaveLocalNodeSet writes ns to <dir>/nodeset.json.
func SaveLocalNodeSet(dir string, ns node.NodeSet) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("session: mkdir %s: %w", dir, err)
	}
	b, err := json.MarshalIndent(ns, "", "  ")
	if err != nil {
		return fmt.Errorf("session: marshal nodeset: %w", err)
	}
	path := filepath.Join(dir, localNodeSetFile)
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		return fmt.Errorf("session: write %s: %w", path, err)
	}
	return nil
}

// LoadLocalNodeSet reads <dir>/nodeset.json.
func LoadLocalNodeSet(dir string) (node.NodeSet, error) {
	path := filepath.Join(dir, localNodeSetFile)
	b, err := os.ReadFile(path)
	if err != nil {
		return node.NodeSet{}, fmt.Errorf("session: read %s: %w", path, err)
	}
	var ns node.NodeSet
	if err := json.Unmarshal(b, &ns); err != nil {
		return node.NodeSet{}, fmt.Errorf("session: parse %s: %w", path, err)
	}
	return ns, nil
}

// SaveLocalNodeSpecs writes the launched node specs to <dir>/nodespecs.json so a
// later `node start --index` can relaunch one node from its exact spec.
func SaveLocalNodeSpecs(dir string, specs []driver.NodeSpec) error {
	b, err := json.MarshalIndent(specs, "", "  ")
	if err != nil {
		return fmt.Errorf("session: marshal node specs: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, localNodeSpecsFile), b, 0o644); err != nil {
		return fmt.Errorf("session: write node specs: %w", err)
	}
	return nil
}

// LoadLocalNodeSpecs reads the launched node specs from <dir>/nodespecs.json.
func LoadLocalNodeSpecs(dir string) ([]driver.NodeSpec, error) {
	b, err := os.ReadFile(filepath.Join(dir, localNodeSpecsFile))
	if err != nil {
		return nil, fmt.Errorf("session: read node specs: %w", err)
	}
	var specs []driver.NodeSpec
	if err := json.Unmarshal(b, &specs); err != nil {
		return nil, fmt.Errorf("session: parse node specs: %w", err)
	}
	return specs, nil
}
