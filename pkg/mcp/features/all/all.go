// Package all registers every built-in capability project via blank imports.
// Binaries that expose the capability surface import this for side effects:
//
//	import _ "github.com/0xmhha/chainbench/pkg/mcp/features/all"
//
// Adding a project's capabilities = a new pkg/mcp/features/<project> package
// (its .jsonl catalog + handlers) plus a blank import here.
package all

import (
	_ "github.com/0xmhha/chainbench/pkg/mcp/features/common"
	_ "github.com/0xmhha/chainbench/pkg/mcp/features/stablenet"
)
