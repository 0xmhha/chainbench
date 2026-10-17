package wbft

import (
	"context"
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/registry"
)

type plugin struct{ registry.StaticPlugin }

// PreForkValidators reads WEMIX PoA before Croissant using WBFT's Ethereum
// info method. Member decoding stays in the PoA reader; only this binary's
// supported method name changes. Other methods and errors pass through.
func (plugin) PreForkValidators(ctx context.Context, fork string, predecessor registry.ChainPlugin, c registry.Caller) (string, []string, error) {
	if fork != "croissant" || predecessor.Family().ID() != "poa" {
		return "", nil, fmt.Errorf("wbft: unsupported pre-fork consensus reader")
	}
	label, validators, err := registry.RunningValidators(ctx, predecessor, predecessorRPC{c})
	return label + " via eth_wemixInfo", validators, err
}

type predecessorRPC struct{ registry.Caller }

func (c predecessorRPC) Call(ctx context.Context, method string, out any, params ...any) error {
	if method == "admin_wemixInfo" {
		method = "eth_wemixInfo"
	}
	return c.Caller.Call(ctx, method, out, params...)
}
