// Package consensus queries a node's consensus state. A family that keeps its
// validators behind a plain RPC method (wbft: istanbul_getValidators) is read
// through that method; one that keeps them elsewhere (poa: a governance
// contract) reads them its own way via RuntimeValidatorReader. RunningValidators
// picks between the two so every caller asks the one question the one way.
package registry

import (
	"context"
	"fmt"
)

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

// RunningValidators reads a chain's current validator set the way its family
// exposes it: through the family's own reader when it has one (poa's governance
// query, wbft's method call), otherwise the manifest's declared method. It
// returns a human label for how the answer was obtained alongside the set, so a
// surface can say where it came from. It is the single place the reader-or-
// method choice is made, so the verify check and the validators query cannot
// diverge.
func RunningValidators(ctx context.Context, p ChainPlugin, c Caller) (label string, vals []string, err error) {
	if r, ok := p.Family().(RuntimeValidatorReader); ok {
		vals, err = r.RuntimeValidators(ctx, c)
		return p.Family().ID() + " runtime validators", vals, err
	}
	method := p.Manifest().Consensus.ValidatorsMethod
	if method == "" {
		return "", nil, fmt.Errorf("chain %s exposes no way to read its validators", p.Manifest().ID)
	}
	vals, err = Validators(ctx, c, method)
	return method, vals, err
}
