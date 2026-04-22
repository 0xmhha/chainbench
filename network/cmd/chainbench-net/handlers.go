package main

import (
	"encoding/json"
	"fmt"

	"github.com/0xmhha/chainbench/network/internal/events"
	"github.com/0xmhha/chainbench/network/internal/state"
)

// Handler is the common signature for all command handlers.
// args is the raw JSON of cmd.args from the wire envelope; bus is available
// for mid-flight progress/events. Returns (data, nil) on success — data is
// wrapped into EmitResult(true, data). Returns (_, *APIError) for typed
// failures; any other error is treated as INTERNAL by the dispatcher.
type Handler func(args json.RawMessage, bus *events.Bus) (map[string]any, error)

// allHandlers builds the command → handler dispatch table. stateDir is
// bound via closure into handlers that need it.
func allHandlers(stateDir string) map[string]Handler {
	return map[string]Handler{
		"network.load": newHandleNetworkLoad(stateDir),
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
		if req.Name != "local" {
			return nil, NewInvalidArgs(fmt.Sprintf("only 'local' supported (got %q)", req.Name))
		}

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
