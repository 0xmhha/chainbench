// Package all registers every built-in chain plugin (and its capabilities) via
// blank imports. Binaries and tests that want the full chain set import this
// package for side effects:
//
//	import _ "github.com/0xmhha/chainbench/pkg/chains/all"
//
// Each chain package registers both its chain plugin and its MCP/CLI
// capabilities; the common package registers the chain-agnostic capabilities.
package all

import (
	_ "github.com/0xmhha/chainbench/pkg/chains/common"
	_ "github.com/0xmhha/chainbench/pkg/chains/stablenet"
	_ "github.com/0xmhha/chainbench/pkg/chains/wbft"
	_ "github.com/0xmhha/chainbench/pkg/chains/wemix"
)
