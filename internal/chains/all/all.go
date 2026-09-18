// Package all registers every built-in chain plugin (and its capabilities) via
// blank imports. Binaries and tests that want the full chain set import this
// package for side effects:
//
//	import _ "github.com/0xmhha/chainbench/internal/chains/all"
//
// Each chain package registers both its chain plugin and its MCP/CLI
// capabilities; the common package registers the chain-agnostic capabilities.
//
// The consensus families are imported on their own rather than left to arrive
// with whichever chain happens to compose them. A project-supplied manifest
// names its family as a string and resolves it out of the registered set, so
// that set must be "every family this build has" and not "every family the
// linked chains needed" — otherwise dropping a chain would quietly take a
// family with it and an external manifest naming it would stop loading.
package all

import (
	_ "github.com/0xmhha/chainbench/internal/consensus/poa"
	_ "github.com/0xmhha/chainbench/internal/consensus/wbft"

	_ "github.com/0xmhha/chainbench/internal/chains/common"
	_ "github.com/0xmhha/chainbench/internal/chains/stablenet"
	_ "github.com/0xmhha/chainbench/internal/chains/wbft"
	_ "github.com/0xmhha/chainbench/internal/chains/wemix"
)
