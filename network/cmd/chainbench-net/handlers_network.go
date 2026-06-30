package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"time"

	"github.com/0xmhha/chainbench/network/internal/drivers/remote"
	"github.com/0xmhha/chainbench/network/internal/drivers/sshremote"
	"github.com/0xmhha/chainbench/network/internal/events"
	"github.com/0xmhha/chainbench/network/internal/probe"
	"github.com/0xmhha/chainbench/network/internal/state"
	"github.com/0xmhha/chainbench/network/internal/types"
)

// providerCaps maps a node provider type to its declared capability set.
// Sets are pre-sorted alphabetically for deterministic JSON output once
// they survive the set-intersection in inferNetworkCapabilities.
//
// "ssh-remote" advertises read-only RPC over an SSH tunnel (Sprint 5b.1) plus
// fs (log tail) and process (node lifecycle) via SSH shell exec (Sprint 5b.2).
// It lacks only admin and network-topology, which are local-host concepts.
//
// Note: this is the provider-level upper bound. A capability set is per-provider,
// so it cannot express that an individual ssh-remote node omitted, say, its
// provider_meta.stop_cmd — such a node reports "process" here but the handler
// returns NOT_SUPPORTED at call time. Capability gating is a coarse pre-filter;
// the handler is the authority for a specific node.
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
			if !slices.Contains(next, c) {
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
			RPCURL       string          `json:"rpc_url"`
			Name         string          `json:"name"`
			Override     string          `json:"override"`
			Auth         json.RawMessage `json:"auth"`          // optional; raw passthrough to types.Auth
			Provider     string          `json:"provider"`      // "" / "remote" (default) | "ssh-remote"
			ProviderMeta json.RawMessage `json:"provider_meta"` // ssh-remote: log_file / *_cmd
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

		// Build the single node, probing for chain_type/chain_id. The "remote"
		// path probes the RPC URL directly over HTTP; the "ssh-remote" path
		// probes through an SSH tunnel (the RPC port lives on the remote host).
		node := types.Node{Id: "node1", Http: req.RPCURL}
		var result *probe.Result
		var err error

		switch req.Provider {
		case "", "remote":
			node.Provider = types.NodeProviderRemote
			result, err = probe.Detect(context.Background(), probe.Options{
				RPCURL:   req.RPCURL,
				Override: req.Override,
			})
			if err != nil {
				if probe.IsInputError(err) {
					return nil, NewInvalidArgs(err.Error())
				}
				return nil, NewUpstream("probe failed", err)
			}
			// Optional auth config. The raw JSON is decoded into types.Auth (a
			// loose map) and attached to Node.Auth — only the env-var NAME is
			// persisted; the credential stays in env vars, resolved at dial time.
			// ValidateAuth runs before assignment so a malformed payload is an
			// INVALID_ARGS at the boundary, not a dial-time UPSTREAM_ERROR.
			if len(req.Auth) > 0 {
				var auth types.Auth
				if err := json.Unmarshal(req.Auth, &auth); err != nil {
					return nil, NewInvalidArgs(fmt.Sprintf("args.auth: %v", err))
				}
				if err := remote.ValidateAuth(auth); err != nil {
					return nil, NewInvalidArgs(err.Error())
				}
				node.Auth = auth
			}
		case "ssh-remote":
			node.Provider = types.NodeProviderSshRemote
			if len(req.Auth) == 0 {
				return nil, NewInvalidArgs("ssh-remote attach requires ssh-password auth")
			}
			var auth types.Auth
			if err := json.Unmarshal(req.Auth, &auth); err != nil {
				return nil, NewInvalidArgs(fmt.Sprintf("args.auth: %v", err))
			}
			// Reuse the dispatch-side parser so creds validation, env lookup, and
			// error classification match node.* handlers exactly.
			creds, cerr := sshCredsFromNode(&types.Node{Auth: auth})
			if cerr != nil {
				return nil, cerr
			}
			hostKey, herr := sshremote.ResolveHostKeyCallback(os.Getenv)
			if herr != nil {
				return nil, NewUpstream("ssh host key", herr)
			}
			client, closer, derr := sshremote.DialTunnelClient(creds, hostKey)
			if derr != nil {
				return nil, NewUpstream("ssh tunnel", derr)
			}
			defer closer.Close()
			result, err = probe.Detect(context.Background(), probe.Options{
				RPCURL:   req.RPCURL,
				Override: req.Override,
				Client:   client,
			})
			if err != nil {
				if probe.IsInputError(err) {
					return nil, NewInvalidArgs(err.Error())
				}
				return nil, NewUpstream("probe failed (over ssh tunnel)", err)
			}
			node.Auth = auth
			if len(req.ProviderMeta) > 0 {
				var meta types.NodeProviderMeta
				if err := json.Unmarshal(req.ProviderMeta, &meta); err != nil {
					return nil, NewInvalidArgs(fmt.Sprintf("args.provider_meta: %v", err))
				}
				node.ProviderMeta = meta
			}
		default:
			return nil, NewInvalidArgs(fmt.Sprintf(
				"args.provider must be 'remote' or 'ssh-remote', got %q", req.Provider))
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
			Nodes:     []types.Node{node},
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

// stopAllTimeout caps the chainbench.sh stop wall-clock at 30s. Read handlers
// (e.g. network.probe) use 10s, but `chainbench stop` does a graceful SIGTERM
// + wait per node before SIGKILL, which can legitimately take longer than a
// single RPC round-trip. 30s leaves headroom without making a hung shutdown
// invisible.
const stopAllTimeout = 30 * time.Second

// newHandleNetworkStopAll returns the "network.stop_all" handler. It is the
// first lifecycle reroute (Sprint 5c.4.1) and follows the VISION §5.12 M2 thin
// wrapper pattern: spawn the existing bash `chainbench.sh stop --quiet` via
// os/exec rather than reimplementing the per-PID SIGTERM/wait/SIGKILL logic
// natively in Go.
//
// Args: { "network"?: "name" }  (omitted/empty defaults to "local")
//
// Result: { "network": "local", "stdout": "<bash output>" }
//
// Error mapping:
//
//	INVALID_ARGS    — malformed args JSON.
//	NOT_SUPPORTED   — non-local network (remote networks have no node lifecycle
//	                  to stop; the caller's name is structurally fine, the
//	                  capability simply doesn't apply).
//	UPSTREAM_ERROR  — chainbench.sh non-zero exit or spawn failure. The wrapped
//	                  error includes the script's combined output for diagnosis.
func newHandleNetworkStopAll(stateDir, chainbenchDir string) Handler {
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
		if req.Network != "local" {
			return nil, NewNotSupported(fmt.Sprintf(
				"network.stop_all only operates on the local network; got %q", req.Network,
			))
		}

		// stateDir is bound by closure for symmetry with the rest of the
		// dispatch table; the bash CLI reads pids.json from CHAINBENCH_DIR's
		// state subdir directly, so we do not pass stateDir into the spawn.
		_ = stateDir

		ctx, cancel := context.WithTimeout(context.Background(), stopAllTimeout)
		defer cancel()
		cmd := exec.CommandContext(ctx,
			filepath.Join(chainbenchDir, "chainbench.sh"),
			"stop", "--quiet",
		)
		// Inherit the parent env so caller-supplied CHAINBENCH_STATE_DIR and
		// related test/profile overrides flow through. Append CHAINBENCH_DIR
		// last so the spawned bash always knows where its repo root lives even
		// if the parent did not export it.
		cmd.Env = append(os.Environ(), "CHAINBENCH_DIR="+chainbenchDir)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, NewUpstream("chainbench stop", fmt.Errorf("%w: %s", err, string(out)))
		}
		return map[string]any{
			"network": "local",
			"stdout":  string(out),
		}, nil
	}
}

// statusTimeout caps the chainbench.sh status wall-clock at 30s. Status walks
// every node in pids.json and issues per-node RPC queries (block height, peer
// count, etc.) — under load this can take longer than a single read RPC, so
// the bound matches stop_all's 30s rather than the 10s used for atomic reads.
const statusTimeout = 30 * time.Second

// newHandleNetworkStatus returns the "network.status" handler. Second
// lifecycle reroute (Sprint 5c.4.1) — same VISION §5.12 M2 thin wrapper
// pattern as network.stop_all: spawn the existing bash `chainbench.sh status
// --json` via os/exec rather than reimplementing the 387-line per-node RPC
// composite formatter natively in Go.
//
// Args: { "network"?: "name" }  (omitted/empty defaults to "local")
//
// Result: the parsed JSON payload from `chainbench.sh status --json` —
// returned directly as the wire result data (NOT wrapped in {network, stdout}
// the way stop_all forwards bash text). The bash CLI is contracted to emit a
// JSON object in --json mode, so the handler's job is to parse + pass it
// through unchanged so the LLM sees the same per-node block height / peer
// count / health shape it has always seen.
//
// Error mapping:
//
//	INVALID_ARGS    — malformed args JSON.
//	NOT_SUPPORTED   — non-local network (remote networks are queried via
//	                  node.rpc / node.block_number etc., not this composite).
//	UPSTREAM_ERROR  — chainbench.sh non-zero exit or spawn failure. The wrapped
//	                  error includes the script's stderr (captured separately
//	                  from stdout via *exec.ExitError so the JSON we parse
//	                  below is not contaminated).
//	INTERNAL        — bash stdout is not valid JSON. Invariant violation —
//	                  bash is contracted to emit JSON in --json mode, so a
//	                  parse failure indicates a regression in the bash side,
//	                  not a caller bug.
func newHandleNetworkStatus(stateDir, chainbenchDir string) Handler {
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
		if req.Network != "local" {
			return nil, NewNotSupported(fmt.Sprintf(
				"network.status only operates on the local network; got %q", req.Network,
			))
		}

		// stateDir is bound by closure for symmetry with the rest of the
		// dispatch table; the bash CLI reads pids.json from CHAINBENCH_DIR's
		// state subdir directly. A future native port would read pids.json
		// from stateDir here.
		_ = stateDir

		ctx, cancel := context.WithTimeout(context.Background(), statusTimeout)
		defer cancel()
		cmd := exec.CommandContext(ctx,
			filepath.Join(chainbenchDir, "chainbench.sh"),
			"status", "--json",
		)
		cmd.Env = append(os.Environ(), "CHAINBENCH_DIR="+chainbenchDir)
		// Use Output() (not CombinedOutput()) so bash stderr does not
		// contaminate the JSON payload we need to parse below. On non-zero
		// exit, *exec.ExitError carries the captured stderr separately.
		out, err := cmd.Output()
		if err != nil {
			var stderr []byte
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				stderr = exitErr.Stderr
			}
			return nil, NewUpstream("chainbench status", fmt.Errorf("%w: %s", err, string(stderr)))
		}
		var statusJson map[string]any
		if err := json.Unmarshal(out, &statusJson); err != nil {
			return nil, NewInternal("status output is not valid JSON", err)
		}
		return statusJson, nil
	}
}
