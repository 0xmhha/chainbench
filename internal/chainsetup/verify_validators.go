package chainsetup

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// ValidatorCheck is the outcome of asking a running chain which validators it
// recognizes and comparing them to the keys this composition runs.
type ValidatorCheck struct {
	// Method is the RPC that returned the set (per consensus family).
	Method string `json:"method"`
	// Expected is the validator addresses the composed keys derive.
	Expected []string `json:"expected"`
	// Actual is the validator addresses the running chain reports.
	Actual []string `json:"actual"`
	// Match is true when the two are the same set.
	Match bool `json:"match"`
	// Mismatch names the offending addresses when Match is false.
	Mismatch string `json:"mismatch,omitempty"`
}

// VerifyValidators asks the running chain for its validator set and checks it is
// exactly the set of addresses the composed keys derive.
//
// It is the runtime counterpart of the genesis-time check. For wbft the genesis
// carries the validator set, so a mismatch is caught before launch; for poa the
// set lives in the governance contract the first producer deploys, so the only
// way to know what the chain actually recognizes is to ask it once it is up
// (the family's <ns>_getValidators). A node whose key is not in the set the
// chain reports cannot produce or sign — the chain stalls with the cause far
// from the symptom — so a mismatch is a failure the operator needs named.
func (w *Workspace) VerifyValidators(ctx context.Context) (ValidatorCheck, error) {
	p, err := w.plugin()
	if err != nil {
		return ValidatorCheck{}, err
	}
	if len(w.state.Nodes) == 0 {
		return ValidatorCheck{}, fmt.Errorf("chainsetup: verify validators: no node table")
	}
	preset, err := store.LoadPreset(w.state.KeysDir)
	if err != nil {
		return ValidatorCheck{}, fmt.Errorf("chainsetup: verify validators: load keys: %w", err)
	}
	expected := preset.NetworkFor(w.state.Validators).Validators

	m, err := w.opener().AddrMap()
	if err != nil {
		return ValidatorCheck{}, err
	}
	method := p.Manifest().Consensus.ValidatorsMethod
	if method == "" {
		return ValidatorCheck{}, fmt.Errorf("chainsetup: verify validators: chain %s declares no validators RPC method", p.Manifest().ID)
	}
	actual, err := registry.Validators(ctx, rpc.Dial(w.nodeHTTPURL(w.state.Nodes[0], m)), method)
	if err != nil {
		return ValidatorCheck{}, fmt.Errorf("chainsetup: verify validators: %s: %w", method, err)
	}

	check := ValidatorCheck{Method: method, Expected: expected, Actual: actual}
	if err := sameValidatorSet(actual, expected, "validators the chain reports", "the composed keys"); err != nil {
		check.Mismatch = err.Error()
	} else {
		check.Match = true
	}
	return check, nil
}

// NetVerifyValidatorsIn names the composed workspace to check.
type NetVerifyValidatorsIn struct {
	DataDir string `json:"dataDir,omitempty"`
}

// NetVerifyValidatorsOut carries the check result.
type NetVerifyValidatorsOut struct {
	Check ValidatorCheck `json:"check"`
}

// NetVerifyValidators opens a workspace and runs the runtime validator check,
// the step-verb behind the surface's `verify --validators`.
func NetVerifyValidators(ctx context.Context, d Deps, in NetVerifyValidatorsIn) (NetVerifyValidatorsOut, error) {
	ws, err := Open(in.DataDir, d.Clock)
	if err != nil {
		return NetVerifyValidatorsOut{}, err
	}
	ws.SetEnv(d.Env)
	ws.SetDriver(d.Driver)
	check, err := ws.VerifyValidators(ctx)
	return NetVerifyValidatorsOut{Check: check}, err
}

// nodeHTTPURL is a node's http RPC URL, translated through the dial-time address
// map (docker/ssh) the same way Health reaches a node.
func (w *Workspace) nodeHTTPURL(ns node.Record, m func(string, int) (string, int)) string {
	host, port := ns.Host, ns.HTTP
	if host == "" {
		host = w.RPCHost()
	}
	if m != nil {
		host, port = m(host, port)
	}
	return fmt.Sprintf("http://%s:%d", host, port)
}
