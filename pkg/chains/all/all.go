// Package all registers every built-in chain plugin via blank imports. Binaries
// and tests that want the full chain set import this package for side effects:
//
//	import _ "github.com/0xmhha/chainbench/pkg/chains/all"
package all

import (
	_ "github.com/0xmhha/chainbench/pkg/chains/stablenet"
	_ "github.com/0xmhha/chainbench/pkg/chains/wbft"
	_ "github.com/0xmhha/chainbench/pkg/chains/wemix"
)
