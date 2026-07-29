//go:build e2e

package e2e

import (
	"math/big"
	"testing"
	"time"
)

// TestE2E_WbftChain is the fresh go-wbft chain scenario (former wbft-chain.sh,
// scenario 3): boot a wbft network from genesis (static bootstrap, WBFT consensus
// family), confirm block production, then verify a value transfer and a
// returns-42 contract deploy/call. Unlike the wemix→wbft handoff, this starts on
// go-wbft at block 0.
//
// The go-wbft node binary is built as `gwemix`; point WBFT_BIN at it:
//
//	WBFT_BIN=/path/to/go-wbft/build/bin/gwemix go test -tags e2e -run TestE2E_WbftChain -v ./tests/e2e/
func TestE2E_WbftChain(t *testing.T) {
	bin := requireBinary(t, "WBFT_BIN", "gwbft")
	cli := buildCLI(t)

	n := boot(t, cli, "wbft", bin, 4, 1)
	url := n.rpcURL
	key := presetFundedKey(t)

	// 1. block production
	n.waitAdvancing(url, 45*time.Second)

	// 2. value transfer: receipt success + recipient credited
	oneEth := new(big.Int).SetUint64(1_000_000_000_000_000_000)
	before := balance(t, url, deadAddr)
	n.sendValue(key, url, deadAddr, oneEth)
	after := balance(t, url, deadAddr)
	if new(big.Int).Sub(after, before).Cmp(oneEth) < 0 {
		t.Fatalf("recipient not credited: %s -> %s", before, after)
	}

	// 3. contract deploy + call returns 42
	addr := n.deployReturns42(key, url)
	n.waitCallReturns42(url, addr, 90*time.Second)
}
