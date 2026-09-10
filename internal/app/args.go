package app

import "github.com/0xmhha/chainbench/internal/core/registry"

// Argument decoding, relayed for the surfaces that cannot reach core.
//
// A capability or MCP tool call arrives as a JSON object decoded into
// map[string]any, and the rules for reading one value out of it belong to one
// implementation: registry.Arg* owns them, because a capability handler is
// where they were needed first.
//
// internal/mcp may not import core directly (architecture-v2 §2, enforced by
// arch.TestMCPGoesThroughApp), so before this file it carried byte-identical
// private copies of the string and int decoders. That is the shape the rule
// produces if nothing relays: the surface reimplements what it is not allowed
// to import, and a change to how an argument decodes then has to be made twice.
// These four one-line pass-throughs are that relay — the app layer doing for
// argument decoding exactly what it does for every other core capability MCP
// needs. The decoding itself still lives in exactly one place.
func ArgString(args map[string]any, key, def string) string {
	return registry.ArgString(args, key, def)
}

// ArgStrings returns a []string argument (a JSON array of strings), or nil.
func ArgStrings(args map[string]any, key string) []string {
	return registry.ArgStrings(args, key)
}

// ArgInt returns an integer argument, or def. JSON numbers decode as float64; a
// numeric string is also accepted.
func ArgInt(args map[string]any, key string, def int) int {
	return registry.ArgInt(args, key, def)
}

// ArgBool returns a boolean argument, or def if absent/not a boolean.
func ArgBool(args map[string]any, key string, def bool) bool {
	return registry.ArgBool(args, key, def)
}
