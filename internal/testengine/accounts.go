package testengine

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/dsl"
)

// keyFile is the name a key set gives an entry's private key on disk.
const keyFile = "private"

// prepareAccounts creates the test accounts a spec declared and funds the ones
// that asked for a balance.
//
// They are created after the chain is up and funded by transaction, which is
// the point of declaring them here rather than in the genesis: an account the
// genesis never mentions is one that adding, removing or renaming does not
// force a new genesis, a new set of derived files, and a re-init of every
// datadir. The cost is that funding is a transaction like any other, so it is
// only possible once the chain seals.
//
// An account declared with no balance stays at zero deliberately — that is what
// a test needs to exercise the paths that fail for want of gas.
func prepareAccounts(ctx context.Context, ring *store.KeySet, keysDir, endpoint string, declared map[string]dsl.AccountV2, funder string) error {
	if ring == nil || len(declared) == 0 {
		return nil
	}
	for _, label := range sortedLabels(declared) {
		decl := declared[label]
		src, err := declaredSource(keysDir, label, decl)
		if err != nil {
			return fmt.Errorf("account %s: %w", label, err)
		}
		entry, err := ring.Add(ctx, keyring.Label(label), src, derive.AccountOnly)
		if err != nil {
			return fmt.Errorf("account %s: %w", label, err)
		}
		amount := strings.TrimSpace(decl.Fund)
		if amount == "" {
			continue
		}
		if err := fundAccount(ctx, endpoint, funder, entry.Address, amount); err != nil {
			return fmt.Errorf("fund %s: %w", label, err)
		}
		// Wait for the balance to actually be there. Submitting is not
		// arriving: the first spec step runs the moment this returns, and a
		// transfer still in the pool leaves the account unable to pay for its
		// own first transaction — which surfaces as "insufficient funds" on a
		// step that looks unrelated to funding.
		if err := awaitBalance(ctx, endpoint, entry.Address); err != nil {
			return fmt.Errorf("fund %s: %w", label, err)
		}
	}
	return nil
}

// declaredSource picks where one declared account's key comes from.
//
// keyFile names a key somebody else holds, so the run reads it and mints
// nothing; without it the label behaves as it always has. The two cannot be
// combined: fund sends from the network's funded account, which exists because
// we composed the network, and an attached run has none — accepting both and
// skipping the transfer would leave a case believing it had a balance.
func declaredSource(keysDir, label string, d dsl.AccountV2) (keyring.Source, error) {
	// expand() here and not at the surface: the endpoint is expanded by the
	// command that reads the declaration, but accounts travel inside the spec
	// and never pass through it. Doing it where the value is used also covers
	// the compose path, so a keyFile means the same thing either way.
	path := strings.TrimSpace(expand(d.KeyFile))
	if path == "" {
		return accountSource(keysDir, label), nil
	}
	if strings.TrimSpace(d.Fund) != "" {
		return nil, fmt.Errorf("declares both keyFile and fund — keyFile is an account that already holds what it needs, fund sends from the network's funded account; keep one")
	}
	if _, err := os.Stat(path); err != nil {
		return nil, fmt.Errorf("keyFile %s: %w (an unset ${VAR} expands to nothing, which is the usual cause)", path, err)
	}
	return keyring.FileSource{Path: path}, nil
}

// accountSource keeps a declared account's identity stable across runs.
//
// A key already written under the ring is read back rather than replaced, so a
// label means the same address on the second run as on the first. Without this
// each run would mint a new key over the old file, and "dev1" would name a
// different account every time — which is exactly the drift labels exist to
// remove.
func accountSource(keysDir, label string) keyring.Source {
	path := filepath.Join(keysDir, label, keyFile)
	if _, err := os.Stat(path); err == nil {
		return keyring.FileSource{Path: path}
	}
	return keyring.RandomSource{}
}

// fundAccount sends the declared balance from the network's funded account.
func fundAccount(ctx context.Context, endpoint, from, to, amount string) error {
	wei, ok := new(big.Int).SetString(strings.TrimPrefix(amount, "0x"), base(amount))
	if !ok {
		return fmt.Errorf("%q is not an amount in wei", amount)
	}
	c := rpc.Dial(endpoint)
	_, err := c.SendTransaction(ctx, rpc.SendTxArgs{
		From: from, To: to, Value: "0x" + wei.Text(16),
	})
	return err
}

// fundWait bounds how long a declared account's balance has to appear.
const fundWait = 60 * time.Second

// awaitBalance polls until the account holds something, so a declaration is
// only reported as prepared once the chain agrees it is.
func awaitBalance(ctx context.Context, endpoint, address string) error {
	c := rpc.Dial(endpoint)
	deadline := time.Now().Add(fundWait)
	for {
		bal, err := c.BalanceAt(ctx, address)
		if err == nil && bal != nil && bal.Sign() > 0 {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("the balance never arrived within %s", fundWait)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// base reads an amount as hex when it says so, decimal otherwise.
func base(amount string) int {
	if strings.HasPrefix(amount, "0x") || strings.HasPrefix(amount, "0X") {
		return 16
	}
	return 10
}

// sortedLabels orders the declarations so a run funds them in the same order
// every time — a funding sequence that varies would give the same declaration
// different nonces from one run to the next.
func sortedLabels(m map[string]dsl.AccountV2) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
