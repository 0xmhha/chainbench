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

	url, err := w.nodeHTTPURL(w.state.Nodes[0])
	if err != nil {
		return ValidatorCheck{}, err
	}
	caller := rpc.Dial(url)

	// How the running set is read — a getValidators RPC or a governance query —
	// is the family's choice, made once in registry.RunningValidators so this
	// check and the `validators` query cannot ask it differently.
	method, actual, err := registry.RunningValidators(ctx, p, caller)
	if err != nil {
		return ValidatorCheck{}, fmt.Errorf("chainsetup: verify validators: %w", err)
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

// nodeHTTPURL is the URL to dial for a node's HTTP RPC.
//
// This layer names the node's own address — its recorded host, or the target's
// when a node carries none — and the resource layer decides how that address is
// reached (a docker container's is translated through the localmap, a remote
// server's is not). Holding a translation map here is what let one dial site
// forget to apply it.
func (w *Workspace) nodeHTTPURL(ns node.Record) (string, error) {
	host := ns.Host
	if host == "" {
		host = w.RPCHost()
	}
	return w.opener().HTTPEndpoint(host, ns.HTTP)
}

// rpcURLOf is nodeHTTPURL for a caller that cannot return an error — NodeSet,
// which renders recorded state and must stay total. A resolution failure (docker
// mode with no localmap) yields an empty URL rather than a wrong one: a reader
// sees "no endpoint", and the steps that actually dial (health, endpoints,
// preflight) still refuse loudly and name the fix.
func rpcURLOf(w *Workspace, ns node.Record) string {
	url, err := w.nodeHTTPURL(ns)
	if err != nil {
		return ""
	}
	return url
}

// metricsURLOf is rpcURLOf for the metrics endpoint. It goes through the same
// opener, so a docker node's metrics are scraped at the published port rather
// than the container-internal one — the translation the RPC dial has always had
// and this one did not. A node with no metrics port has no endpoint to give.
func metricsURLOf(w *Workspace, ns node.Record) string {
	if ns.Metrics == 0 {
		return ""
	}
	host := ns.Host
	if host == "" {
		host = w.RPCHost()
	}
	url, err := w.opener().HTTPEndpoint(host, ns.Metrics)
	if err != nil {
		return ""
	}
	return url
}
