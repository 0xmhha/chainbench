package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/0xmhha/chainbench/network/internal/drivers/remote"
	"github.com/0xmhha/chainbench/network/internal/events"
	"github.com/0xmhha/chainbench/network/internal/probe"
	"github.com/0xmhha/chainbench/network/internal/state"
	"github.com/0xmhha/chainbench/network/internal/types"
)

// providerCaps maps a node provider type to its declared capability set.
// Sets are pre-sorted alphabetically for deterministic JSON output once
// they survive the set-intersection in inferNetworkCapabilities.
//
// Forward-compat: "ssh-remote" is declared but not yet used by any driver
// (Sprint 5b). Listing it here lets handlers infer capabilities for
// hybrid networks that include ssh-remote nodes once that provider lands,
// without requiring a follow-up edit to this map.
var providerCaps = map[string][]string{
	"local":      {"admin", "fs", "network-topology", "process", "rpc", "ws"},
	"remote":     {"rpc", "ws"},
	"ssh-remote": {"fs", "process", "rpc", "ws"},
}

// inferNetworkCapabilities returns the set intersection of provider-declared
// capabilities across all nodes. Empty input yields an empty (non-nil) slice.
// An unknown provider contributes the empty set, which collapses the
// intersection to empty — the conservative choice when we don't know what a
// node can do.
func inferNetworkCapabilities(nodes []types.Node) []string {
	if len(nodes) == 0 {
		return []string{}
	}
	common := make(map[string]struct{})
	for _, c := range providerCaps[string(nodes[0].Provider)] {
		common[c] = struct{}{}
	}
	for _, n := range nodes[1:] {
		next := providerCaps[string(n.Provider)]
		for c := range common {
			found := false
			for _, x := range next {
				if x == c {
					found = true
					break
				}
			}
			if !found {
				delete(common, c)
			}
		}
	}
	out := make([]string, 0, len(common))
	for c := range common {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

// loadNetworkNodes resolves a network name to its node list via the same
// state.LoadActive path used by network.load. Routing on Name keeps the
// "local vs remote" decision in one place (state package); the handler
// stays a thin args/result shim.
func loadNetworkNodes(stateDir, network string) ([]types.Node, error) {
	net, err := state.LoadActive(state.LoadActiveOptions{StateDir: stateDir, Name: network})
	if err != nil {
		return nil, NewUpstream("failed to load network state", err)
	}
	return net.Nodes, nil
}

// newHandleNetworkCapabilities returns the "network.capabilities" handler.
// It enumerates the network's nodes (local pids.json or
// networks/<name>.json) and returns the provider-derived capability
// intersection. No RPC dial — purely a state-file read.
//
// Args: { "network"?: "name" }  (omitted/empty defaults to "local")
//
// Result: { "network": "<name>", "capabilities": ["admin", "fs", ...] }
//
// Error mapping:
//
//	INVALID_ARGS   — malformed args JSON, or non-local name that fails the
//	                 schema pattern at the state layer.
//	UPSTREAM_ERROR — state.LoadActive failure (missing pids.json, missing
//	                 networks/<name>.json, decode error).
func newHandleNetworkCapabilities(stateDir string) Handler {
	return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
		var req struct {
			Network string `json:"network"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
			}
		}
		if req.Network == "" {
			req.Network = "local"
		}
		// Validate non-local name shape at the boundary so a structural
		// input error surfaces as INVALID_ARGS rather than UPSTREAM_ERROR
		// after a (cheap, but still) state read attempt.
		if req.Network != "local" && !state.IsValidRemoteName(req.Network) {
			return nil, NewInvalidArgs(fmt.Sprintf(
				"args.network must be 'local' or match [a-z0-9][a-z0-9_-]*: %q", req.Network,
			))
		}
		nodes, err := loadNetworkNodes(stateDir, req.Network)
		if err != nil {
			return nil, err
		}
		caps := inferNetworkCapabilities(nodes)
		return map[string]any{
			"network":      req.Network,
			"capabilities": caps,
		}, nil
	}
}

// newHandleNetworkLoad returns the "network.load" handler closing over stateDir.
func newHandleNetworkLoad(stateDir string) Handler {
	return func(args json.RawMessage, bus *events.Bus) (map[string]any, error) {
		var req struct {
			Name string `json:"name"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
			}
		}
		if req.Name == "" {
			return nil, NewInvalidArgs("args.name is required")
		}
		// Pre-check name shape at the handler boundary. "local" is always legal
		// here; remote names must match the schema pattern. Rejecting here keeps
		// structural input errors as INVALID_ARGS; state-layer errors (missing
		// file) below stay as UPSTREAM_ERROR.
		if req.Name != "local" && !state.IsValidRemoteName(req.Name) {
			return nil, NewInvalidArgs(fmt.Sprintf("args.name must be 'local' or match [a-z0-9][a-z0-9_-]*: %q", req.Name))
		}

		// Non-local names are resolved by state.LoadActive → loadRemote, which
		// returns a wrapped ErrStateNotFound when no matching attached network
		// file exists. That surfaces to callers as UPSTREAM_ERROR below.
		net, err := state.LoadActive(state.LoadActiveOptions{StateDir: stateDir, Name: req.Name})
		if err != nil {
			return nil, NewUpstream("failed to load active state", err)
		}

		// Marshal through JSON so the result is a plain map[string]any matching
		// the generated schema layout.
		raw, err := json.Marshal(net)
		if err != nil {
			return nil, NewInternal("marshal network", err)
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, NewInternal("unmarshal network", err)
		}
		return data, nil
	}
}

const (
	minProbeTimeoutMs = 100
	maxProbeTimeoutMs = 60000
)

// newHandleNetworkProbe returns the "network.probe" handler. It accepts
//
//	{ "rpc_url": "...", "timeout_ms"?: 100..60000, "override"?: "stablenet|wbft|wemix|ethereum" }
//
// and returns the probe.Result marshalled as a plain map. Error mapping:
//
//	INVALID_ARGS    — malformed args, missing rpc_url, out-of-range timeout_ms,
//	                  or probe.IsInputError(err) == true (sentinel
//	                  ErrMissingURL / ErrInvalidURL / ErrUnknownOverride).
//	UPSTREAM_ERROR  — any other probe.Detect failure (RPC HTTP/transport/parse).
//	INTERNAL        — JSON round-trip failure (should not happen in practice).
//
// Stateless: no stateDir / chainbenchDir dependencies, so constructed with no
// args and registered in allHandlers with a zero-arg closure.
func newHandleNetworkProbe() Handler {
	return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
		var req struct {
			RPCURL    string `json:"rpc_url"`
			TimeoutMs *int   `json:"timeout_ms"`
			Override  string `json:"override"`
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
			}
		}
		if req.RPCURL == "" {
			return nil, NewInvalidArgs("args.rpc_url is required")
		}
		opts := probe.Options{
			RPCURL:   req.RPCURL,
			Override: req.Override,
		}
		if req.TimeoutMs != nil {
			if *req.TimeoutMs < minProbeTimeoutMs || *req.TimeoutMs > maxProbeTimeoutMs {
				return nil, NewInvalidArgs(fmt.Sprintf(
					"args.timeout_ms must be %d..%d, got %d",
					minProbeTimeoutMs, maxProbeTimeoutMs, *req.TimeoutMs,
				))
			}
			opts.Timeout = time.Duration(*req.TimeoutMs) * time.Millisecond
		}

		result, err := probe.Detect(context.Background(), opts)
		if err != nil {
			// Sentinel-based classification (errors.Is via probe.IsInputError)
			// avoids fragile substring matching on error messages.
			if probe.IsInputError(err) {
				return nil, NewInvalidArgs(err.Error())
			}
			return nil, NewUpstream("probe failed", err)
		}

		// Round-trip through JSON so the returned map matches the wire schema
		// layout exactly (same approach used by newHandleNetworkLoad).
		raw, err := json.Marshal(result)
		if err != nil {
			return nil, NewInternal("marshal probe result", err)
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, NewInternal("unmarshal probe result", err)
		}
		return data, nil
	}
}

// newHandleNetworkAttach returns the "network.attach" handler. It probes a
// remote RPC endpoint via probe.Detect, builds a types.Network from the
// result, and persists it as <stateDir>/networks/<name>.json. Subsequent
// network.load calls can resolve the network by name.
//
// Args: { "rpc_url": "...", "name": "...", "override"?: "stablenet|wbft|wemix|ethereum" }
//
// Returns: {name, chain_type, chain_id, rpc_url, nodes, created}
//
//	created=true  — no prior state file existed
//	created=false — an existing file was overwritten
//
// Error mapping:
//
//	INVALID_ARGS   — malformed args, missing/invalid name, reserved "local" name,
//	                 missing rpc_url, or probe.IsInputError(err) sentinel.
//	UPSTREAM_ERROR — probe.Detect failure on the RPC endpoint, or SaveRemote
//	                 I/O failure.
//	INTERNAL       — JSON round-trip failure.
//
// Probe runs BEFORE any filesystem write so a failed classification never
// leaves a partial state file behind.
func newHandleNetworkAttach(stateDir string) Handler {
	return func(args json.RawMessage, _ *events.Bus) (map[string]any, error) {
		var req struct {
			RPCURL   string          `json:"rpc_url"`
			Name     string          `json:"name"`
			Override string          `json:"override"`
			Auth     json.RawMessage `json:"auth"` // optional; raw passthrough to types.Auth
		}
		if len(args) > 0 {
			if err := json.Unmarshal(args, &req); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args: %v", err))
			}
		}
		if req.RPCURL == "" {
			return nil, NewInvalidArgs("args.rpc_url is required")
		}
		if req.Name == "" {
			return nil, NewInvalidArgs("args.name is required")
		}
		if req.Name == "local" {
			return nil, NewInvalidArgs("args.name 'local' is reserved")
		}
		// Reject structurally-invalid names at the boundary so we don't waste a
		// probe round-trip on a name we can't persist anyway.
		if !state.IsValidRemoteName(req.Name) {
			return nil, NewInvalidArgs(fmt.Sprintf(
				"args.name must match [a-z0-9][a-z0-9_-]*: %q", req.Name,
			))
		}

		result, err := probe.Detect(context.Background(), probe.Options{
			RPCURL:   req.RPCURL,
			Override: req.Override,
		})
		if err != nil {
			if probe.IsInputError(err) {
				return nil, NewInvalidArgs(err.Error())
			}
			return nil, NewUpstream("probe failed", err)
		}

		// Detect prior existence to set created=true on first save, false on
		// overwrite. Stat after probe succeeds so we never report a pseudo-
		// create on a failed attach.
		path := filepath.Join(stateDir, "networks", req.Name+".json")
		_, statErr := os.Stat(path)
		created := os.IsNotExist(statErr)

		net := &types.Network{
			Name:      req.Name,
			ChainType: types.NetworkChainType(result.ChainType),
			ChainId:   int(result.ChainID),
			Nodes: []types.Node{{
				Id:       "node1",
				Provider: types.NodeProvider("remote"),
				Http:     req.RPCURL,
			}},
		}
		// Optional auth config. The raw JSON is decoded into types.Auth (a loose
		// map[string]interface{}) and attached to Node.Auth. Only the env-var
		// NAME is persisted; the actual credential stays in env vars and is
		// resolved at dial time via remote.AuthFromNode.
		//
		// ValidateAuth runs BEFORE the Node.Auth assignment and BEFORE
		// SaveRemote so a structurally-invalid payload (unknown type, missing
		// required env field) is rejected with INVALID_ARGS at the input
		// boundary rather than deferring to dial time as UPSTREAM_ERROR. The
		// check is pure structural — no round trip — so it can run after the
		// probe without penalty.
		if len(req.Auth) > 0 {
			var auth types.Auth
			if err := json.Unmarshal(req.Auth, &auth); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args.auth: %v", err))
			}
			if err := remote.ValidateAuth(auth); err != nil {
				return nil, NewInvalidArgs(err.Error())
			}
			net.Nodes[0].Auth = auth
		}
		if err := state.SaveRemote(stateDir, net); err != nil {
			// The handler pre-checks reserved and invalid names, so the sentinel
			// branches should be unreachable. Classify defensively anyway — a
			// future refactor that skips the pre-check must not surface a
			// structural-input error as an endpoint/UPSTREAM failure.
			if errors.Is(err, state.ErrReservedName) || errors.Is(err, state.ErrInvalidName) {
				return nil, NewInvalidArgs(err.Error())
			}
			return nil, NewUpstream("save remote state", err)
		}

		raw, err := json.Marshal(net)
		if err != nil {
			return nil, NewInternal("marshal network", err)
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return nil, NewInternal("unmarshal network", err)
		}
		data["rpc_url"] = req.RPCURL
		data["created"] = created
		return data, nil
	}
}
