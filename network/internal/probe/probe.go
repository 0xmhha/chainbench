package probe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
// Caller wraps error as an APIError (INVALID_ARGS / UPSTREAM_ERROR) per context.
func Detect(ctx context.Context, opts Options) (*Result, error) {
	if opts.RPCURL == "" {
		return nil, fmt.Errorf("rpc_url required")
	}
	parsed, err := url.Parse(opts.RPCURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("rpc_url must be http(s): %q", opts.RPCURL)
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
		ChainID:    chainID,
		RPCURL:     opts.RPCURL,
		Namespaces: []string{},
		Warnings:   []string{},
	}

	if opts.Override != "" {
		if !isKnownOverride(opts.Override) {
			return nil, fmt.Errorf("unknown override %q", opts.Override)
		}
		result.ChainType = opts.Override
		result.Overridden = true
		return result, nil
	}

	for _, sig := range signatures {
		if sig.probeMethod == "" {
			continue
		}
		ok, _ := probeMethod(ctx, client, opts.RPCURL, sig.probeMethod)
		if !ok {
			continue
		}
		if sig.knownChainIDs != nil && !sig.knownChainIDs[chainID] {
			continue
		}
		result.ChainType = sig.chainType
		result.Namespaces = appendUnique(result.Namespaces, sig.namespace)
		return result, nil
	}

	// Disambiguation second pass: if a non-id-gated istanbul signature exists and we
	// saw istanbul but did not match stablenet's id gate, fall through to wbft.
	for _, sig := range signatures {
		if sig.knownChainIDs != nil || sig.probeMethod == "" {
			continue
		}
		ok, _ := probeMethod(ctx, client, opts.RPCURL, sig.probeMethod)
		if !ok {
			continue
		}
		result.ChainType = sig.chainType
		result.Namespaces = appendUnique(result.Namespaces, sig.namespace)
		return result, nil
	}

	result.ChainType = "ethereum"
	return result, nil
}

func fetchChainID(ctx context.Context, client *http.Client, url string) (int64, error) {
	resp, err := jsonRPCCall(ctx, client, url, "eth_chainId", []interface{}{})
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

func probeMethod(ctx context.Context, client *http.Client, url, method string) (bool, *jsonRPCResponse) {
	resp, err := jsonRPCCall(ctx, client, url, method, []interface{}{})
	if err != nil {
		return false, nil
	}
	if resp.Error != nil {
		return false, resp
	}
	return true, resp
}

func appendUnique(xs []string, s string) []string {
	for _, x := range xs {
		if x == s {
			return xs
		}
	}
	return append(xs, s)
}
