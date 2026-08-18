// Package state persists what a bringup produces — the running NodeSet and the
// armed launch specs — so a later command can act on a launched network without
// re-specifying its endpoints. It is deliberately small: two JSON files per data
// root.
//
// Its only consumer is the app layer (worklist T7.11a); nothing else should read
// or write these files. The format belongs to the pre-netcompose bringup path
// and goes away with it — the redesigned engine records its equivalent state in
// core/session instead.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// nodeSetFile is the file name under the data root.
const nodeSetFile = "nodeset.json"

// SaveNodeSet writes ns to <dir>/nodeset.json.
func SaveNodeSet(dir string, ns node.NodeSet) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("state: mkdir %s: %w", dir, err)
	}
	b, err := json.MarshalIndent(ns, "", "  ")
	if err != nil {
		return fmt.Errorf("state: marshal nodeset: %w", err)
	}
	path := filepath.Join(dir, nodeSetFile)
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		return fmt.Errorf("state: write %s: %w", path, err)
	}
	return nil
}

// LoadNodeSet reads <dir>/nodeset.json.
func LoadNodeSet(dir string) (node.NodeSet, error) {
	path := filepath.Join(dir, nodeSetFile)
	b, err := os.ReadFile(path)
	if err != nil {
		return node.NodeSet{}, fmt.Errorf("state: read %s: %w", path, err)
	}
	var ns node.NodeSet
	if err := json.Unmarshal(b, &ns); err != nil {
		return node.NodeSet{}, fmt.Errorf("state: parse %s: %w", path, err)
	}
	return ns, nil
}
