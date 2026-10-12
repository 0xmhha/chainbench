package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/resource"
)

// verifyWebNetworkProcesses checks every node before a network-wide operation.
// Cleanup may stop verified launches; recomposition requires confirmed vacancy.
func verifyWebNetworkProcesses(ctx context.Context, state State, p webChainPayload, lookup resource.Lookup, vacant bool) error {
	if err := verifyWebRecordLedger(state, p.ControlDir); err != nil {
		return err
	}
	actual, err := (resource.Opener{Lookup: lookup}).Inspect(ctx, state.Target)
	if err != nil || actual != p.Target {
		return webConflict("the recorded network target is not the reviewed target")
	}
	if len(state.Nodes) == 0 {
		return nil
	}
	wc, err := deploymentWorkspace(p.Config.DeploymentDocumentInput)
	if err != nil {
		return err
	}
	// Recomposition can request a new count. Validate the old layout against
	// its own recorded count before deciding whether it can be replaced.
	scope := p
	scope.Arguments.Validators = 0
	if err = validateWebControlLayout(state, scope, wc); err != nil {
		return err
	}
	for _, ns := range state.Nodes {
		if vacant && ns.PID != 0 {
			return webConflict("a recorded node still runs where the network must be vacant")
		}
		if err = verifyWebNodeProcesses(ctx, state, []string{string(ns.NodeLabel())}, lookup, p.Targets...); err != nil {
			return err
		}
	}
	return nil
}

// recheckWebNetworkRecord verifies the current record and live processes without
// opening the mutable core workspace or adopting ledger identities.
func recheckWebNetworkRecord(ctx context.Context, p webChainPayload, lookup resource.Lookup, vacant bool) error {
	raw, err := os.ReadFile(filepath.Join(p.ControlDir, "chain-record.json"))
	if os.IsNotExist(err) && (!vacant || p.RecordDigest == "") {
		return nil
	}
	if err != nil {
		return webConflict("the network record cannot be read")
	}
	if vacant && manifestHash(raw) != p.RecordDigest {
		return webConflict("the network record changed since review")
	}
	var state State
	if err = json.Unmarshal(raw, &state); err != nil {
		return webConflict("the network record is malformed")
	}
	return verifyWebNetworkProcesses(ctx, state, p, lookup, vacant)
}

// verifyWebRecordLedger refuses identities that core Open would substitute for
// the checked record. Reading this file does not seed or repair the ledger.
func verifyWebRecordLedger(state State, dir string) error {
	raw, err := os.ReadFile(filepath.Join(dir, process.LedgerFile))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return webConflict("the node process ledger cannot be read")
	}
	var ledger struct {
		Procs []process.Proc `json:"procs"`
	}
	if err = json.Unmarshal(raw, &ledger); err != nil {
		return webConflict("the node process ledger is malformed")
	}
	seen := map[string]bool{}
	for _, proc := range ledger.Procs {
		if proc.PID <= 0 || seen[proc.Label] {
			return webConflict("the node process ledger repeats a node or lacks its PID")
		}
		seen[proc.Label] = true
		found := false
		for _, ns := range state.Nodes {
			if string(ns.NodeLabel()) == proc.Label {
				found = proc.PID == ns.PID && proc.Host == ns.Host && proc.DataDir == ns.DataDir
				break
			}
		}
		if !found {
			return webConflict("a ledger entry differs from the recorded node")
		}
	}
	return nil
}
