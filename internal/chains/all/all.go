// Package all registers every built-in chain plugin (and its capabilities) via
// blank imports. Binaries and tests that want the full chain set import this
// package for side effects:
//
//	import _ "github.com/0xmhha/chainbench/internal/chains/all"
//
// Each chain package registers both its chain plugin and its MCP/CLI
// capabilities; the common package registers the chain-agnostic capabilities.
package all

import (
	_ "github.com/0xmhha/chainbench/internal/chains/common"
	_ "github.com/0xmhha/chainbench/internal/chains/stablenet"
	_ "github.com/0xmhha/chainbench/internal/chains/wbft"
	_ "github.com/0xmhha/chainbench/internal/chains/wemix"
)
