// Package consensus queries a node's consensus state through the RPC method a
// chain's manifest declares (istanbul_getValidators for the wbft family,
// wemix_getValidators for poa). Driving the method name from the manifest keeps
// this chain-agnostic — the hardcoded "istanbul_*" in the old TS consensus tool
// is gone (docs/CHAINBENCH_GO_REDESIGN.md §C.5).
package registry

import "context"

// Caller is the RPC surface consensus queries need. *rpc.Client satisfies it.
type Caller interface {
	Call(ctx context.Context, method string, out any, params ...any) error
}

// Validators returns the validator/producer set via the given RPC method (from
// the chain manifest's consensus.validators_method).
func Validators(ctx context.Context, c Caller, method string) ([]string, error) {
	var vals []string
	if err := c.Call(ctx, method, &vals); err != nil {
		return nil, err
	}
	return vals, nil
}
