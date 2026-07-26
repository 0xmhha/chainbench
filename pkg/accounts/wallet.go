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
	// Deploy sends a contract-creation transaction with initCode (the raw
	// creation bytecode) and returns the transaction hash and the created
	// contract address. value may be nil for a zero-value deployment.
	Deploy(ctx context.Context, initCode []byte, value *big.Int) (txHash, contract string, err error)
	// Execute sends a transaction to a contract with the given calldata (a
	// state-changing call) and returns the transaction hash. value may be nil.
	Execute(ctx context.Context, toHex string, data []byte, value *big.Int) (txHash string, err error)
	// SendFeeDelegated sends a 0x16 fee-delegation transfer: this wallet is the
	// sender (transfers amountWei) while feePayerKey (a second private key)
	// covers the gas. Returns the transaction hash. Only meaningful on chains
	// whose provider reports SupportsTxType(0x16).
	SendFeeDelegated(ctx context.Context, feePayerKey []byte, toHex string, amountWei *big.Int) (txHash string, err error)
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

func (s sdkWallet) Deploy(ctx context.Context, initCode []byte, value *big.Int) (string, string, error) {
	if len(initCode) == 0 {
		return "", "", fmt.Errorf("accounts: empty init code")
	}
	h, addr, err := s.w.Deploy(ctx, initCode, value)
	if err != nil {
		return "", "", err
	}
	return h.Hex(), addr.Hex(), nil
}

func (s sdkWallet) Execute(ctx context.Context, toHex string, data []byte, value *big.Int) (string, error) {
	to, err := sdktypes.HexToAddress(toHex)
	if err != nil {
		return "", fmt.Errorf("accounts: invalid contract address %q: %w", toHex, err)
	}
	h, err := s.w.Execute(ctx, to, data, value)
	if err != nil {
		return "", err
	}
	return h.Hex(), nil
}

func (s sdkWallet) SendFeeDelegated(ctx context.Context, feePayerKey []byte, toHex string, amountWei *big.Int) (string, error) {
	to, err := sdktypes.HexToAddress(toHex)
	if err != nil {
		return "", fmt.Errorf("accounts: invalid recipient %q: %w", toHex, err)
	}
	if amountWei == nil {
		return "", fmt.Errorf("accounts: nil amount")
	}
	feePayer, err := sdkacct.FromPrivateKeyBytes(feePayerKey)
	if err != nil {
		return "", fmt.Errorf("accounts: bad fee-payer key: %w", err)
	}
	h, err := s.w.SendFeeDelegated(ctx, feePayer, to, amountWei)
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
