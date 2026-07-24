// Package mcp is the chainbench MCP surface (requirement #14) as a separate
// module: it exposes chainbench's core capabilities to an agent over the Model
// Context Protocol. Tool handlers call the same core packages the CLI uses, so
// the two surfaces stay behaviorally identical (docs/CHAINBENCH_GO_REDESIGN.md
// §B). The protocol layer here is self-contained (JSON-RPC 2.0 over stdio) to
// avoid a heavy external SDK; the transport (cmd/chainbench-mcp) is a thin loop
// over Server.Handle.
package mcp

import "context"

// Handler runs a tool with decoded arguments and returns human/agent-readable
// text (mirroring the TS server's text results).
type Handler func(ctx context.Context, args map[string]any) (string, error)

// Tool is one MCP tool: its name, description, JSON-schema for inputs, and
// handler.
type Tool struct {
	Name        string
	Description string
	InputSchema map[string]any
	Handler     Handler
}

// argString returns a string argument or def if absent/not a string.
func argString(args map[string]any, key, def string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return def
}

// argStrings returns a []string argument (JSON array of strings), or nil.
func argStrings(args map[string]any, key string) []string {
	v, ok := args[key]
	if !ok {
		return nil
	}
	switch t := v.(type) {
	case []string:
		return t
	case []any:
		out := make([]string, 0, len(t))
		for _, e := range t {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}
