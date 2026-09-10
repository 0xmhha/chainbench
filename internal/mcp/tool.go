// Package mcp is the chainbench MCP surface (requirement #14) as a separate
// module: it exposes chainbench's core capabilities to an agent over the Model
// Context Protocol. Tool handlers call the same core packages the CLI uses, so
// the two surfaces stay behaviorally identical (docs/CHAINBENCH_GO_REDESIGN.md
// §B). The protocol layer here is self-contained (JSON-RPC 2.0 over stdio) to
// avoid a heavy external SDK; the transport (cmd/chainbench-mcp) is a thin loop
// over Server.Handle.
package mcp

import (
	"context"

	"github.com/0xmhha/chainbench/internal/app"
)

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
	// ReadOnly declares that calling this tool changes no file, no process and
	// no chain state, and that its result carries no secret.
	//
	// It is the same property the CLI's query projection reads, declared here
	// too because MCP has no command tree to project from
	// (surface-unification-design §4.4, rule 3). An agent asks for the tool
	// list and learns which subset it may call while exploring.
	//
	// Declared, not inferred, for the same reasons: chainbench_node_rpc takes
	// the method as an argument, so nothing about the tool says whether it
	// writes, and chainbench_keyring_show is safe while an export would not be
	// even though both only print.
	ReadOnly bool
}

// The four arg* helpers below are this surface's short spellings for the
// decoders the app layer relays from core/registry (see app/args.go). They
// delegate rather than reimplement: this file used to carry byte-identical
// copies of the string and int decoders, so a change to how an argument decodes
// had to be made twice, and the copy that was missed would have been the one an
// agent actually reached. The short names stay because the handlers read better
// with them at 183 call sites.

func argString(args map[string]any, key, def string) string {
	return app.ArgString(args, key, def)
}

func argStrings(args map[string]any, key string) []string {
	return app.ArgStrings(args, key)
}

func argInt(args map[string]any, key string, def int) int {
	return app.ArgInt(args, key, def)
}

func argBool(args map[string]any, key string, def bool) bool {
	return app.ArgBool(args, key, def)
}
