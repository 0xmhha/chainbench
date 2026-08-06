package accounts

import (
	"context"
	"fmt"
	"math/big"

	sdkacct "github.com/0xmhha/accounts/account"
	sdktransport "github.com/0xmhha/accounts/transport"
	sdktx "github.com/0xmhha/accounts/tx"
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
	// SendLegacy sends a type 0x00 (EIP-155 legacy) value transfer of amountWei
	// to the hex recipient, returning the tx hash. SendCoin uses type 0x02; this
	// exposes the legacy path for tx-type regression coverage.
	SendLegacy(ctx context.Context, toHex string, amountWei *big.Int) (txHash string, err error)
	// SendSetCode sends a type 0x04 (EIP-7702) set-code transaction: this wallet
	// sponsors the gas, and authorityKey (a DISTINCT account) authorizes
	// delegating its account code to delegateHex. After it mines, the authority's
	// code becomes 0xef0100||delegate. Returns the tx hash.
	SendSetCode(ctx context.Context, authorityKey []byte, delegateHex string) (txHash string, err error)
	// SendDynamicFee sends a type 0x02 (EIP-1559 dynamic-fee) value transfer of
	// amountWei to the hex recipient, returning the tx hash. Fees are auto-filled
	// (GasTipCap from the node's suggestion, GasFeeCap = gasPrice + tip). This is
	// the explicit 0x02 path for tx-type regression coverage.
	SendDynamicFee(ctx context.Context, toHex string, amountWei *big.Int) (txHash string, err error)
	// SendAccessList sends a type 0x01 (EIP-2930 access-list) value transfer of
	// amountWei to the hex recipient with an empty access list, using legacy-style
	// GasPrice, returning the tx hash.
	SendAccessList(ctx context.Context, toHex string, amountWei *big.Int) (txHash string, err error)
	// SendDynamicFeeGas sends a type 0x02 transfer with an EXPLICIT GasFeeCap and
	// GasTipCap (no auto-fill), for gas-policy boundary tests. A fee below the
	// chain minimum is expected to be rejected by the node (a non-nil error).
	SendDynamicFeeGas(ctx context.Context, toHex string, amountWei, gasFeeCap, gasTipCap *big.Int) (txHash string, err error)
	// SendLegacyGas sends a type 0x00 transfer with an EXPLICIT GasPrice.
	SendLegacyGas(ctx context.Context, toHex string, amountWei, gasPrice *big.Int) (txHash string, err error)
	// SendAccessListGas sends a type 0x01 transfer (empty access list) with an
	// EXPLICIT GasPrice.
	SendAccessListGas(ctx context.Context, toHex string, amountWei, gasPrice *big.Int) (txHash string, err error)
	// SendDynamicFeeTx sends a fully-specified type-0x02 transaction: explicit
	// nonce, gas limit, fee caps, recipient, value, and calldata. It is the
	// low-level path for regression cases that need control the higher-level
	// helpers do not expose — gas-limit bounds, out-of-gas, revert receipts, nonce
	// ordering, and replacement transactions.
	SendDynamicFeeTx(ctx context.Context, args DynamicTxArgs) (txHash string, err error)
}

// DynamicTxArgs fully specifies a type-0x02 transaction. A nil Nonce uses the
// account's next nonce; an empty ToHex is a contract creation. Value, GasFeeCap,
// and GasTipCap must be non-nil; Gas must be set (no auto-estimation, so a call
// that would revert or run out of gas still mines with status 0x0).
type DynamicTxArgs struct {
	ToHex     string
	Value     *big.Int
	Data      []byte
	Gas       uint64
	GasFeeCap *big.Int
	GasTipCap *big.Int
	Nonce     *uint64
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

func (s sdkWallet) SendLegacy(ctx context.Context, toHex string, amountWei *big.Int) (string, error) {
	to, err := sdktypes.HexToAddress(toHex)
	if err != nil {
		return "", fmt.Errorf("accounts: invalid recipient %q: %w", toHex, err)
	}
	if amountWei == nil {
		return "", fmt.Errorf("accounts: nil amount")
	}
	// The SDK wallet's SendCoin is type 0x02; build a legacy tx directly using
	// the wallet's exported account and client (no SDK change needed).
	nonce, err := s.w.Client.Nonce(ctx, s.w.Account.Address())
	if err != nil {
		return "", fmt.Errorf("accounts: nonce: %w", err)
	}
	gasPrice, err := s.w.Client.GasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("accounts: gas price: %w", err)
	}
	chainID, err := s.w.Client.ChainID(ctx)
	if err != nil {
		return "", fmt.Errorf("accounts: chain id: %w", err)
	}
	t := &sdktx.LegacyTx{Nonce: nonce, GasPrice: gasPrice, Gas: 21000, To: &to, Value: amountWei}
	if err := t.Sign(chainID, s.w.Account.PrivateKey()); err != nil {
		return "", fmt.Errorf("accounts: sign legacy tx: %w", err)
	}
	h, err := s.w.Client.SendRawTransaction(ctx, t.Encode())
	if err != nil {
		return "", err
	}
	return h.Hex(), nil
}

func (s sdkWallet) SendDynamicFee(ctx context.Context, toHex string, amountWei *big.Int) (string, error) {
	to, err := sdktypes.HexToAddress(toHex)
	if err != nil {
		return "", fmt.Errorf("accounts: invalid recipient %q: %w", toHex, err)
	}
	if amountWei == nil {
		return "", fmt.Errorf("accounts: nil amount")
	}
	nonce, err := s.w.Client.Nonce(ctx, s.w.Account.Address())
	if err != nil {
		return "", fmt.Errorf("accounts: nonce: %w", err)
	}
	tip, err := s.w.Client.MaxPriorityFeePerGas(ctx)
	if err != nil {
		return "", fmt.Errorf("accounts: max priority fee: %w", err)
	}
	gasPrice, err := s.w.Client.GasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("accounts: gas price: %w", err)
	}
	chainID, err := s.w.Client.ChainID(ctx)
	if err != nil {
		return "", fmt.Errorf("accounts: chain id: %w", err)
	}
	t := &sdktx.DynamicFeeTx{
		ChainID: chainID, Nonce: nonce, GasTipCap: tip,
		GasFeeCap: new(big.Int).Add(gasPrice, tip), Gas: 21000, To: &to, Value: amountWei,
	}
	if err := t.Sign(s.w.Account.PrivateKey()); err != nil {
		return "", fmt.Errorf("accounts: sign dynamic-fee tx: %w", err)
	}
	h, err := s.w.Client.SendRawTransaction(ctx, t.Encode())
	if err != nil {
		return "", err
	}
	return h.Hex(), nil
}

func (s sdkWallet) SendAccessList(ctx context.Context, toHex string, amountWei *big.Int) (string, error) {
	to, err := sdktypes.HexToAddress(toHex)
	if err != nil {
		return "", fmt.Errorf("accounts: invalid recipient %q: %w", toHex, err)
	}
	if amountWei == nil {
		return "", fmt.Errorf("accounts: nil amount")
	}
	nonce, err := s.w.Client.Nonce(ctx, s.w.Account.Address())
	if err != nil {
		return "", fmt.Errorf("accounts: nonce: %w", err)
	}
	gasPrice, err := s.w.Client.GasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("accounts: gas price: %w", err)
	}
	chainID, err := s.w.Client.ChainID(ctx)
	if err != nil {
		return "", fmt.Errorf("accounts: chain id: %w", err)
	}
	t := &sdktx.AccessListTx{
		ChainID: chainID, Nonce: nonce, GasPrice: gasPrice, Gas: 21000,
		To: &to, Value: amountWei, AccessList: sdktx.AccessList{},
	}
	if err := t.Sign(s.w.Account.PrivateKey()); err != nil {
		return "", fmt.Errorf("accounts: sign access-list tx: %w", err)
	}
	h, err := s.w.Client.SendRawTransaction(ctx, t.Encode())
	if err != nil {
		return "", err
	}
	return h.Hex(), nil
}

func (s sdkWallet) SendDynamicFeeGas(ctx context.Context, toHex string, amountWei, gasFeeCap, gasTipCap *big.Int) (string, error) {
	to, err := sdktypes.HexToAddress(toHex)
	if err != nil {
		return "", fmt.Errorf("accounts: invalid recipient %q: %w", toHex, err)
	}
	if amountWei == nil || gasFeeCap == nil || gasTipCap == nil {
		return "", fmt.Errorf("accounts: nil amount or fee")
	}
	nonce, err := s.w.Client.Nonce(ctx, s.w.Account.Address())
	if err != nil {
		return "", fmt.Errorf("accounts: nonce: %w", err)
	}
	chainID, err := s.w.Client.ChainID(ctx)
	if err != nil {
		return "", fmt.Errorf("accounts: chain id: %w", err)
	}
	t := &sdktx.DynamicFeeTx{
		ChainID: chainID, Nonce: nonce, GasTipCap: gasTipCap, GasFeeCap: gasFeeCap,
		Gas: 21000, To: &to, Value: amountWei,
	}
	if err := t.Sign(s.w.Account.PrivateKey()); err != nil {
		return "", fmt.Errorf("accounts: sign dynamic-fee tx: %w", err)
	}
	h, err := s.w.Client.SendRawTransaction(ctx, t.Encode())
	if err != nil {
		return "", err
	}
	return h.Hex(), nil
}

func (s sdkWallet) SendDynamicFeeTx(ctx context.Context, args DynamicTxArgs) (string, error) {
	if args.Value == nil || args.GasFeeCap == nil || args.GasTipCap == nil {
		return "", fmt.Errorf("accounts: nil value or fee cap")
	}
	var toPtr *sdktypes.Address
	if args.ToHex != "" {
		to, err := sdktypes.HexToAddress(args.ToHex)
		if err != nil {
			return "", fmt.Errorf("accounts: invalid recipient %q: %w", args.ToHex, err)
		}
		toPtr = &to
	}
	nonce := uint64(0)
	if args.Nonce != nil {
		nonce = *args.Nonce
	} else {
		n, err := s.w.Client.Nonce(ctx, s.w.Account.Address())
		if err != nil {
			return "", fmt.Errorf("accounts: nonce: %w", err)
		}
		nonce = n
	}
	chainID, err := s.w.Client.ChainID(ctx)
	if err != nil {
		return "", fmt.Errorf("accounts: chain id: %w", err)
	}
	t := &sdktx.DynamicFeeTx{
		ChainID: chainID, Nonce: nonce, GasTipCap: args.GasTipCap, GasFeeCap: args.GasFeeCap,
		Gas: args.Gas, To: toPtr, Value: args.Value, Data: args.Data,
	}
	if err := t.Sign(s.w.Account.PrivateKey()); err != nil {
		return "", fmt.Errorf("accounts: sign dynamic-fee tx: %w", err)
	}
	h, err := s.w.Client.SendRawTransaction(ctx, t.Encode())
	if err != nil {
		return "", err
	}
	return h.Hex(), nil
}

func (s sdkWallet) SendLegacyGas(ctx context.Context, toHex string, amountWei, gasPrice *big.Int) (string, error) {
	to, err := sdktypes.HexToAddress(toHex)
	if err != nil {
		return "", fmt.Errorf("accounts: invalid recipient %q: %w", toHex, err)
	}
	if amountWei == nil || gasPrice == nil {
		return "", fmt.Errorf("accounts: nil amount or gas price")
	}
	nonce, err := s.w.Client.Nonce(ctx, s.w.Account.Address())
	if err != nil {
		return "", fmt.Errorf("accounts: nonce: %w", err)
	}
	chainID, err := s.w.Client.ChainID(ctx)
	if err != nil {
		return "", fmt.Errorf("accounts: chain id: %w", err)
	}
	t := &sdktx.LegacyTx{Nonce: nonce, GasPrice: gasPrice, Gas: 21000, To: &to, Value: amountWei}
	if err := t.Sign(chainID, s.w.Account.PrivateKey()); err != nil {
		return "", fmt.Errorf("accounts: sign legacy tx: %w", err)
	}
	h, err := s.w.Client.SendRawTransaction(ctx, t.Encode())
	if err != nil {
		return "", err
	}
	return h.Hex(), nil
}

func (s sdkWallet) SendAccessListGas(ctx context.Context, toHex string, amountWei, gasPrice *big.Int) (string, error) {
	to, err := sdktypes.HexToAddress(toHex)
	if err != nil {
		return "", fmt.Errorf("accounts: invalid recipient %q: %w", toHex, err)
	}
	if amountWei == nil || gasPrice == nil {
		return "", fmt.Errorf("accounts: nil amount or gas price")
	}
	nonce, err := s.w.Client.Nonce(ctx, s.w.Account.Address())
	if err != nil {
		return "", fmt.Errorf("accounts: nonce: %w", err)
	}
	chainID, err := s.w.Client.ChainID(ctx)
	if err != nil {
		return "", fmt.Errorf("accounts: chain id: %w", err)
	}
	t := &sdktx.AccessListTx{
		ChainID: chainID, Nonce: nonce, GasPrice: gasPrice, Gas: 21000,
		To: &to, Value: amountWei, AccessList: sdktx.AccessList{},
	}
	if err := t.Sign(s.w.Account.PrivateKey()); err != nil {
		return "", fmt.Errorf("accounts: sign access-list tx: %w", err)
	}
	h, err := s.w.Client.SendRawTransaction(ctx, t.Encode())
	if err != nil {
		return "", err
	}
	return h.Hex(), nil
}

func (s sdkWallet) SendSetCode(ctx context.Context, authorityKey []byte, delegateHex string) (string, error) {
	delegate, err := sdktypes.HexToAddress(delegateHex)
	if err != nil {
		return "", fmt.Errorf("accounts: invalid delegate %q: %w", delegateHex, err)
	}
	authority, err := sdkacct.FromPrivateKeyBytes(authorityKey)
	if err != nil {
		return "", fmt.Errorf("accounts: bad authority key: %w", err)
	}
	chainID, err := s.w.Client.ChainID(ctx)
	if err != nil {
		return "", fmt.Errorf("accounts: chain id: %w", err)
	}
	// The authority delegates its own code to `delegate`, signed by itself, at
	// its current nonce.
	authNonce, err := s.w.Client.Nonce(ctx, authority.Address())
	if err != nil {
		return "", fmt.Errorf("accounts: authority nonce: %w", err)
	}
	auth := sdktx.SetCodeAuthorization{ChainID: chainID, Address: delegate, Nonce: authNonce}
	if err := auth.Sign(authority.PrivateKey()); err != nil {
		return "", fmt.Errorf("accounts: sign authorization: %w", err)
	}
	nonce, err := s.w.Client.Nonce(ctx, s.w.Account.Address())
	if err != nil {
		return "", fmt.Errorf("accounts: nonce: %w", err)
	}
	tip, err := s.w.Client.MaxPriorityFeePerGas(ctx)
	if err != nil {
		return "", fmt.Errorf("accounts: priority fee: %w", err)
	}
	gp, err := s.w.Client.GasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("accounts: gas price: %w", err)
	}
	t := &sdktx.SetCodeTx{
		ChainID: chainID, Nonce: nonce, GasTipCap: tip, GasFeeCap: new(big.Int).Add(gp, tip),
		Gas: 200000, To: s.w.Account.Address(), Value: big.NewInt(0),
		AuthorizationList: []sdktx.SetCodeAuthorization{auth},
	}
	if err := t.Sign(s.w.Account.PrivateKey()); err != nil {
		return "", fmt.Errorf("accounts: sign set-code tx: %w", err)
	}
	h, err := s.w.Client.SendRawTransaction(ctx, t.Encode())
	if err != nil {
		return "", err
	}
	return h.Hex(), nil
}

// GenerateKey creates a fresh random account, returning its private key bytes and
// 0x-prefixed address. Useful for tests needing a distinct account (e.g. a
// set-code authority). TEST/UTILITY use — the key is not persisted.
func GenerateKey() (privKey []byte, addressHex string, err error) {
	acct, err := sdkacct.Generate()
	if err != nil {
		return nil, "", fmt.Errorf("accounts: generate key: %w", err)
	}
	return acct.PrivateKeyBytes(), acct.Address().Hex(), nil
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
