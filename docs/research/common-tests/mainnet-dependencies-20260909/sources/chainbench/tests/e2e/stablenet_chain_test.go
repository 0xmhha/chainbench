//go:build e2e

package e2e

import (
	"math/big"
	"testing"
	"time"
)

// TestE2E_StablenetChain is the go-stablenet basic end-to-end (former
// stablenet-chain.sh, scenario 4): boot a network, confirm block production,
// then verify a value transfer and a returns-42 contract deploy/call.
//
//	GSTABLE_BIN=/path/to/gstable go test -tags e2e -run TestE2E_StablenetChain -v ./tests/e2e/
func TestE2E_StablenetChain(t *testing.T) {
	bin := requireBinary(t, "GSTABLE_BIN", "gstable")
	cli := buildCLI(t)

	n := boot(t, cli, "stablenet", bin, 4, 1)
	url := n.rpcURL
	key := presetFundedKey(t)

	// 1. block production (poll — WBFT consensus warms up over ~10-15s)
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
