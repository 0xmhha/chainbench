package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"
)

const defaultTimeout = 5 * time.Second

type Options struct {
	RPCURL   string
	Timeout  time.Duration
	Override string
	Client   *http.Client
}

type Result struct {
	ChainType  string   `json:"chain_type"`
	ChainID    int64    `json:"chain_id"`
	RPCURL     string   `json:"rpc_url"`
	Namespaces []string `json:"namespaces"`
	Overridden bool     `json:"overridden"`
	Warnings   []string `json:"warnings"`
}

// Detect probes an RPC endpoint for chain_type and chain_id.
// Returns (*Result, nil) on success; (nil, error) on unrecoverable failures.
// Input-validation failures are wrapped with ErrMissingURL / ErrInvalidURL /
// ErrUnknownOverride — callers classify via errors.Is (see IsInputError).
func Detect(ctx context.Context, opts Options) (*Result, error) {
	if opts.RPCURL == "" {
		return nil, ErrMissingURL
	}
	parsed, err := url.Parse(opts.RPCURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("%w: got %q", ErrInvalidURL, opts.RPCURL)
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	client := opts.Client
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	chainID, err := fetchChainID(ctx, client, opts.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("eth_chainId: %w", err)
	}

	result := &Result{
		ChainID: chainID,
		RPCURL:  opts.RPCURL,
		// Namespaces: accumulated as signatures match. Warnings: reserved slot
		// for non-fatal caller-visible messages (e.g., chain_id ∉ expected
		// range under override) — kept as [] to stabilize JSON output shape.
		Namespaces: []string{},
		Warnings:   []string{},
	}

	if opts.Override != "" {
		if !isKnownOverride(opts.Override) {
			return nil, fmt.Errorf("%w: %q", ErrUnknownOverride, opts.Override)
		}
		result.ChainType = opts.Override
		result.Overridden = true
		return result, nil
	}

	for _, sig := range signatures {
		if sig.probeMethod == "" {
			continue
		}
		if !probeMethod(ctx, client, opts.RPCURL, sig.probeMethod) {
			continue
		}
		if sig.knownChainIDs != nil && !sig.knownChainIDs[chainID] {
			continue
		}
		result.ChainType = sig.chainType
		result.Namespaces = appendUnique(result.Namespaces, sig.namespace)
		return result, nil
	}

	result.ChainType = "ethereum"
	return result, nil
}

func fetchChainID(ctx context.Context, client *http.Client, endpoint string) (int64, error) {
	resp, err := jsonRPCCall(ctx, client, endpoint, "eth_chainId", []any{})
	if err != nil {
		return 0, err
	}
	if resp.Error != nil {
		return 0, fmt.Errorf("rpc error %d: %s", resp.Error.Code, resp.Error.Message)
	}
	var hex string
	if err := json.Unmarshal(resp.Result, &hex); err != nil {
		return 0, fmt.Errorf("chainId not a string: %w", err)
	}
	hex = strings.TrimPrefix(hex, "0x")
	id, err := strconv.ParseInt(hex, 16, 64)
	if err != nil {
		return 0, fmt.Errorf("chainId parse %q: %w", hex, err)
	}
	return id, nil
}

func probeMethod(ctx context.Context, client *http.Client, endpoint, method string) bool {
	resp, err := jsonRPCCall(ctx, client, endpoint, method, []any{})
	if err != nil {
		return false
	}
	return resp.Error == nil
}

func appendUnique(xs []string, s string) []string {
	if slices.Contains(xs, s) {
		return xs
	}
	return append(xs, s)
}
