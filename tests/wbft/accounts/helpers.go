// helpers.go gathers what the remaining Go-func cases in this package share:
// the faucet key and wallet, and the balance read. Collected here when the
// ported cases (now DSL specs under tests/specs) were deleted; it goes away
// with the cases that stay.
package accounts

import (
	"math/big"
	"os"
	"path/filepath"
	"strings"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/testkit"
)

// balanceOf reads an account's latest balance via the primary node; a failed
// or unparsable read yields zero so WaitFor keeps polling.
func balanceOf(t *testkit.T, addr string) *big.Int {
	var hexBal string
	if err := t.Primary().Call(t.Ctx(), "eth_getBalance", &hexBal, addr, "latest"); err != nil {
		return big.NewInt(0)
	}
	v, ok := new(big.Int).SetString(strings.TrimPrefix(hexBal, "0x"), 16)
	if !ok {
		return big.NewInt(0)
	}
	return v
}

// openFaucetWallet opens a wallet for the shared genesis-funded key against the
// primary node — the common prologue of the three metadata cases.
func openFaucetWallet(t *testkit.T) (accounts.Wallet, *rpc.Client) {
	primary, ok := t.NodeSet().Primary()
	t.Truef(ok, "node set has no primary node")

	ap, err := accounts.ForChain(t.NodeSet().Chain)
	t.NoErr(err, "accounts.ForChain")

	key := fundedKey(t)

	w, err := ap.OpenWallet(t.Ctx(), key, primary.RPCURL)
	t.NoErr(err, "open wallet")
	return w, rpc.Dial(primary.RPCURL)
}

// fundedKey is the private key of a genesis-funded account the cases spend
// from. An operator-supplied CHAINBENCH_FUNDED_KEY wins (that is how a case
// acts on a chain chainbench did not compose); otherwise the first preset
// node's key is used — every preset node account is in the genesis alloc, so
// there is no separate faucet key to keep anywhere.
func fundedKey(t *testkit.T) []byte {
	if k, ok := t.FundedKey(); ok {
		return k
	}
	dir, ok := presetDir()
	if !ok {
		t.Skip("no funded key: set CHAINBENCH_FUNDED_KEY or run from the repository (keys/preset)")
	}
	p, err := store.LoadPreset(dir)
	t.NoErr(err, "load preset")
	nk, ok := p.Node(1)
	t.Truef(ok, "preset has no node1")
	return nk.Nodekey.Bytes()
}

// presetDir finds the repository's preset by walking up from the working
// directory, so the cases run from any directory inside the checkout.
func presetDir() (string, bool) {
	dir, err := os.Getwd()
	if err != nil {
		return "", false
	}
	for {
		cand := filepath.Join(dir, "keys", "preset")
		if _, err := os.Stat(filepath.Join(cand, "metadata.json")); err == nil {
			return cand, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}
