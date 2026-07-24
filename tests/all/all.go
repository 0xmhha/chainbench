// Package all registers every built-in test case via blank imports, so the
// chainbench binary's test runner can see them. New test-case packages under
// tests/<family>/<category>/ are added here.
package all

import (
	_ "github.com/0xmhha/chainbench/tests/wbft/accounts"
	_ "github.com/0xmhha/chainbench/tests/wbft/consensus"
)
