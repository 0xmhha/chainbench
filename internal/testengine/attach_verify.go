package testengine

import (
	"context"
	"fmt"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// VerifyAttachedChain refuses an attach whose endpoint speaks a different RPC
// surface than the chain the declaration claims.
//
// Nothing composed an attached network, so its chain-preset is an assertion by
// whoever set it up: "chain": "stablenet" is a claim, not a fact, and nothing
// compared it with the endpoint. Pointing the endpoint variable at a wemix node
// and running with stablenet-attached was accepted, and skipsOn and requires
// then gated on a chain the run was not talking to — a green light for a
// question nobody asked.
//
// It compares PROBE METHODS, not detected chain names, and that is the whole
// design. The manifests detect stablenet and wbft with the same method
// (istanbul_getValidators), separated only by a chain-id gate on stablenet's
// 8283; an endpoint that answers that method with any other id detects as wbft.
// A network whose operator chose a different chain id is the normal case for
// anything we did not compose — a testnet, a mock — so refusing on the detected
// NAME fires on exactly the networks this check exists to let people reach.
// Refusing on the method instead fires only when the endpoint answers a
// different RPC surface, which is the confusion worth catching: wemix
// (wemix_getReward) against a declaration of stablenet or wbft, or the reverse.
//
// What it cannot do: wbft and wemix are not told apart by either signal.
// go-wbft forked from go-wemix and its binary still reports "gwemix", and the
// manifests give them different probe methods only when those namespaces are
// open. Within that pair this check is silent, and says so rather than
// pretending otherwise.
//
// A probe that cannot reach the endpoint is not a mismatch. The run will fail
// on its own with a clearer message, and refusing here would turn "the node is
// down" into "the chain is wrong".
func VerifyAttachedChain(ctx context.Context, declared, rpcURL string, allowMismatch bool) error {
	if declared == "" || rpcURL == "" || allowMismatch {
		return nil
	}
	want, err := registry.Get(declared)
	if err != nil {
		return nil // an unregistered chain is NewAttachEngine's to reject
	}
	m := want.Manifest()

	// First signal: the client string, because it is the one an operator does
	// not close. A node answers web3_clientVersion with its binary's name —
	// "Gstable/v...", "Gwemix/v..." — and the manifest already records which
	// binary a chain runs, so no new declaration is needed. This is what
	// catches a stablenet declaration pointed at a wemix endpoint even when
	// every identifying namespace is shut.
	if m.Binary != "" {
		if v, err := clientVersion(ctx, rpcURL); err == nil && v != "" {
			if !strings.Contains(strings.ToLower(v), strings.ToLower(m.Binary)) {
				return fmt.Errorf(
					"engine: attach declares chain %q, which runs %s, but %s answers web3_clientVersion %q — "+
						"point the endpoint at a %s node, name the chain the endpoint really is, or pass --allow-chain-mismatch",
					declared, m.Binary, rpcURL, v, declared)
			}
			return nil
		}
	}

	// Second signal, when the client string is unavailable: the probe method.
	// Compared by METHOD and not by detected name on purpose. The manifests
	// detect stablenet and wbft with the same method (istanbul_getValidators),
	// separated only by a chain-id gate on 8283, so an endpoint that answers it
	// with any other id detects as wbft — and a network whose operator chose a
	// different chain id is the normal case for anything we did not compose.
	// Refusing on the name would fire on exactly the networks this check exists
	// to let people reach.
	if m.Probe.Method == "" {
		return nil
	}
	res, derr := collector.Detect(ctx, collector.Options{RPCURL: rpcURL})
	if derr != nil || res == nil || res.ChainType == "" || res.ChainType == declared {
		return nil
	}
	got, gerr := registry.Get(res.ChainType)
	if gerr != nil || got.Manifest().Probe.Method == m.Probe.Method {
		return nil
	}
	return fmt.Errorf(
		"engine: attach declares chain %q, whose probe is %s, but %s answers %s and detects as %q (chain id %d) — "+
			"point the endpoint at a %s node, name the chain the endpoint really is, or pass --allow-chain-mismatch",
		declared, m.Probe.Method, rpcURL, got.Manifest().Probe.Method, res.ChainType, res.ChainID, declared)
}

// clientVersion reads web3_clientVersion from one endpoint.
func clientVersion(ctx context.Context, rpcURL string) (string, error) {
	var v string
	if err := rpc.Dial(rpcURL).Call(ctx, "web3_clientVersion", &v); err != nil {
		return "", err
	}
	return v, nil
}
