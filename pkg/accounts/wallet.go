package accounts

import (
	"context"
	"fmt"
	"math/big"

	sdkacct "github.com/0xmhha/accounts/account"
	sdktransport "github.com/0xmhha/accounts/transport"
	sdktypes "github.com/0xmhha/accounts/types"
	sdkwallet "github.com/0xmhha/accounts/wallet"
)

// Wallet is chainbench's account-operation handle for one funded key against
// one node. It uses primitive types (hex strings, wei big.Int) so callers stay
// decoupled from the accounts SDK's concrete types; the implementation is
// backed by github.com/0xmhha/accounts/wallet.
type Wallet interface {
	// Address returns the wallet's 0x-prefixed hex address.
	Address() string
	// SendCoin transfers amountWei to the hex recipient and returns the
	// 0x-prefixed transaction hash. This is the faucet primitive
	// (requirement #3): a genesis-funded key sends value to another account.
	SendCoin(ctx context.Context, toHex string, amountWei *big.Int) (string, error)
}

// sdkWallet adapts *sdkwallet.Wallet to the Wallet interface.
type sdkWallet struct {
	w    *sdkwallet.Wallet
	addr string
}

func (s sdkWallet) Address() string { return s.addr }

func (s sdkWallet) SendCoin(ctx context.Context, toHex string, amountWei *big.Int) (string, error) {
	to, err := sdktypes.HexToAddress(toHex)
	if err != nil {
		return "", fmt.Errorf("accounts: invalid recipient %q: %w", toHex, err)
	}
	if amountWei == nil {
		return "", fmt.Errorf("accounts: nil amount")
	}
	h, err := s.w.SendCoin(ctx, to, amountWei)
	if err != nil {
		return "", err
	}
	return h.Hex(), nil
}

// openWallet builds an SDK-backed wallet for privKey against rpcURL. wallet.New
// fetches the chain id over RPC, so rpcURL must be reachable.
func openWallet(ctx context.Context, privKey []byte, rpcURL string) (Wallet, error) {
	acct, err := sdkacct.FromPrivateKeyBytes(privKey)
	if err != nil {
		return nil, fmt.Errorf("accounts: bad private key: %w", err)
	}
	client := sdktransport.Dial(rpcURL)
	w, err := sdkwallet.New(ctx, acct, client)
	if err != nil {
		return nil, fmt.Errorf("accounts: open wallet: %w", err)
	}
	return sdkWallet{w: w, addr: acct.Address().Hex()}, nil
}

// addressForKey derives the 0x-prefixed hex address for a private key, offline.
func addressForKey(privKey []byte) (string, error) {
	acct, err := sdkacct.FromPrivateKeyBytes(privKey)
	if err != nil {
		return "", fmt.Errorf("accounts: bad private key: %w", err)
	}
	return acct.Address().Hex(), nil
}
