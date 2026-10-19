package testhelper

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

const remoteObservationWindow = 5 * time.Second

var remoteQuantityPattern = regexp.MustCompile(`^0x(?:0|[1-9a-fA-F][0-9a-fA-F]*)$`)

func remoteQuantity(value any) (*big.Int, error) {
	text, ok := value.(string)
	if !ok || !remoteQuantityPattern.MatchString(text) {
		return nil, fmt.Errorf("invalid RPC quantity %v", value)
	}
	n, _ := new(big.Int).SetString(text[2:], 16)
	return n, nil
}

// rpcHealthAssertion records optional method support. Only a typed
// method-not-found response establishes absence; failed transports, malformed
// envelopes, permissions and internal errors cannot turn a probe green.
type rpcHealthAssertion struct{}

func (rpcHealthAssertion) Check(ctx context.Context, ac *interp.AssertCtx) (session.AssertResult, error) {
	return remoteObservation(ctx, ac, "rpcHealth", func(ctx context.Context, c *rpc.Client) (map[string]any, error) {
		observed := map[string]any{}
		for _, method := range []string{"eth_blockNumber", "net_peerCount", "web3_clientVersion"} {
			var value any
			err := c.Call(ctx, method, &value)
			if err != nil {
				var reply *rpc.ServerError
				if method != "eth_blockNumber" && errors.As(err, &reply) && reply.Code == -32601 {
					observed[method] = map[string]any{"status": "unsupported", "code": reply.Code}
					continue
				}
				return observed, fmt.Errorf("%s: %w", method, err)
			}
			if method == "web3_clientVersion" {
				text, ok := value.(string)
				if !ok || strings.TrimSpace(text) == "" {
					return observed, fmt.Errorf("%s: missing client version", method)
				}
			} else if _, err = remoteQuantity(value); err != nil {
				return observed, fmt.Errorf("%s: %w", method, err)
			}
			observed[method] = map[string]any{"status": "available", "result": value}
		}
		return observed, nil
	})
}

// chainInfoAssertion retains the legacy five-second non-regression check.
// It deliberately permits an unchanged head, unlike blockAdvance. A syncing
// response must be false or an object with the standard three quantities.
type chainInfoAssertion struct{}

func (chainInfoAssertion) Check(ctx context.Context, ac *interp.AssertCtx) (session.AssertResult, error) {
	return remoteObservation(ctx, ac, "chainInfo", func(ctx context.Context, c *rpc.Client) (map[string]any, error) {
		observed := map[string]any{}
		for _, method := range []string{"eth_chainId", "eth_syncing", "eth_blockNumber"} {
			var value any
			if err := c.Call(ctx, method, &value); err != nil {
				return observed, err
			}
			observed[method] = value
			if method == "eth_syncing" {
				if err := remoteSyncStatus(value); err != nil {
					return observed, err
				}
				continue
			}
			n, err := remoteQuantity(value)
			if err != nil {
				return observed, fmt.Errorf("%s: %w", method, err)
			}
			if n.Sign() <= 0 {
				return observed, fmt.Errorf("%s must be positive", method)
			}
		}
		before, _ := remoteQuantity(observed["eth_blockNumber"])
		started := time.Now()
		observed["firstObservedAt"] = started.UTC().Format(time.RFC3339Nano)
		timer := time.NewTimer(remoteObservationWindow)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return observed, ctx.Err()
		case <-timer.C:
		}
		var after any
		if err := c.Call(ctx, "eth_blockNumber", &after); err != nil {
			return observed, err
		}
		observed["secondObservedAt"] = time.Now().UTC().Format(time.RFC3339Nano)
		observed["elapsedMillis"] = time.Since(started).Milliseconds()
		observed["secondBlockNumber"] = after
		last, err := remoteQuantity(after)
		if err != nil {
			return observed, err
		}
		if last.Cmp(before) < 0 {
			return observed, fmt.Errorf("block height regressed: %v -> %v", observed["eth_blockNumber"], after)
		}
		if err := ctx.Err(); err != nil {
			return observed, err
		}
		return observed, nil
	})
}

func remoteSyncStatus(value any) error {
	if flag, ok := value.(bool); ok && !flag {
		return nil
	}
	fields, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("eth_syncing requires false or a sync progress object")
	}
	for _, name := range []string{"startingBlock", "currentBlock", "highestBlock"} {
		if _, err := remoteQuantity(fields[name]); err != nil {
			return fmt.Errorf("eth_syncing %s: %w", name, err)
		}
	}
	return nil
}

func remoteObservation(ctx context.Context, ac *interp.AssertCtx, name string, observe func(context.Context, *rpc.Client) (map[string]any, error)) (session.AssertResult, error) {
	result := session.AssertResult{Assert: name, Provenance: ac.Spec}
	observations := map[string]any{}
	result.Actual = observations
	targets := assertTargets(ac)
	if len(targets) == 0 {
		return result, fmt.Errorf("dsl: %s: no target node RPC URL", name)
	}
	names, urls := map[string]bool{}, map[string]bool{}
	for _, target := range targets {
		if target.url == "" || names[target.name] || urls[target.url] {
			return result, fmt.Errorf("dsl: %s: missing or duplicate node endpoint", name)
		}
		names[target.name] = true
		urls[target.url] = true
	}
	var firstError error
	for _, target := range targets {
		c, err := clientFor(ac.Deps, target.url)
		observed := map[string]any{}
		if err == nil {
			observed, err = observe(ctx, c)
		}
		observations[target.name] = observed
		if err != nil {
			observed["error"] = err.Error()
			if firstError == nil {
				firstError = fmt.Errorf("dsl: %s %s: %w", name, target.name, err)
			}
		}
	}
	result.Pass = firstError == nil
	return result, firstError
}

// remoteBalanceAssertion retains the remote script's address-format check
// before IO, without changing generic balanceAt's node-authoritative policy.
type remoteBalanceAssertion struct{}

func (remoteBalanceAssertion) Check(ctx context.Context, ac *interp.AssertCtx) (session.AssertResult, error) {
	spec, err := resolveNamedArgs(ac.Deps, ac.Spec)
	address, ok := spec["address"].(string)
	if err != nil || !ok || !remoteAddressPattern.MatchString(address) {
		if err == nil {
			err = fmt.Errorf("remoteBalance requires a 0x-prefixed20-byte address")
		}
		return session.AssertResult{Assert: "remoteBalance", Provenance: ac.Spec, Actual: err.Error()}, err
	}
	return remoteObservation(ctx, ac, "remoteBalance", func(ctx context.Context, c *rpc.Client) (map[string]any, error) {
		var value any
		observed := map[string]any{"address": address}
		if err := c.Call(ctx, "eth_getBalance", &value, address, "latest"); err != nil {
			return observed, err
		}
		observed["balance"] = value
		_, err := remoteQuantity(value)
		return observed, err
	})
}

var remoteAddressPattern = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)
