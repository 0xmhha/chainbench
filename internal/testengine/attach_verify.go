package testengine

import (
	"context"
	"fmt"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/remote"
	"net/http"
	"os"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/collector"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// VerifyAttachedNode checks an operator's chain claim through that profile's
// authentication transport. Client names match either the logical binary or its
// native build target: go-wbft builds gwemix, which chainbench calls gwbft.
// An unavailable client string falls back to comparing probe methods rather
// than detected chain names, so an operator's custom chain ID remains valid.
//
// wbft and wemix share a client name, so this identity signal alone cannot tell
// them apart. Runtime capability observations check the reached RPC separately.
// An unreachable endpoint is left to the run to diagnose as a connection failure.
func VerifyAttachedNode(ctx context.Context, declared string, endpoint node.Node, allowMismatch bool) error {
	client, err := remote.HTTPClientFromAuth(remote.Auth(endpoint.Auth), os.Getenv)
	if err != nil {
		return err
	}
	return verifyAttachedChain(ctx, declared, endpoint.RPCURL, allowMismatch, client)
}

func verifyAttachedChain(ctx context.Context, declared, rpcURL string, allowMismatch bool, client *http.Client) error {
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
	// "Gstable/v...", "Gwemix/v..." — and the manifest records both the
	// logical binary and its native build target. This is what
	// catches a stablenet declaration pointed at a wemix endpoint even when
	// every identifying namespace is shut.
	if m.Binary != "" {
		if v, err := clientVersion(ctx, rpcURL, client); err == nil && v != "" {
			name, _, _ := strings.Cut(v, "/")
			matches := strings.EqualFold(name, m.Binary) || m.Build.MakeTarget != "" && strings.EqualFold(name, m.Build.MakeTarget)
			if !matches {
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
	res, derr := collector.Detect(ctx, collector.Options{RPCURL: rpcURL, Client: client})
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
func clientVersion(ctx context.Context, rpcURL string, client *http.Client) (string, error) {
	var v string
	if err := rpc.DialWithClient(rpcURL, client).Call(ctx, "web3_clientVersion", &v); err != nil {
		return "", err
	}
	return v, nil
}
