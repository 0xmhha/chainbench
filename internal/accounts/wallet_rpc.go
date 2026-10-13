package accounts

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	sdkacct "github.com/0xmhha/accounts/account"
	sdktransport "github.com/0xmhha/accounts/transport"
	sdktypes "github.com/0xmhha/accounts/types"
)

// RPCRequester lets the caller keep its endpoint authentication and redirect
// policy while the accounts SDK continues to own key handling and tx signing.
type RPCRequester interface {
	Call(context.Context, string, any, ...any) error
}

type walletRPC interface {
	ChainID(context.Context) (*big.Int, error)
	Nonce(context.Context, sdktypes.Address) (uint64, error)
	GasPrice(context.Context) (*big.Int, error)
	MaxPriorityFeePerGas(context.Context) (*big.Int, error)
	EstimateGas(context.Context, sdktransport.CallMsg) (uint64, error)
	SendRawTransaction(context.Context, []byte) (sdktypes.Hash, error)
	IsBlacklisted(context.Context, sdktypes.Address) (bool, error)
}

type rpcWalletClient struct{ RPCRequester }

func (c rpcWalletClient) quantity(ctx context.Context, method string, params ...any) (*big.Int, error) {
	var value string
	if err := c.Call(ctx, method, &value, params...); err != nil {
		return nil, err
	}
	value = strings.TrimPrefix(value, "0x")
	if value == "" {
		return new(big.Int), nil
	}
	n, ok := new(big.Int).SetString(value, 16)
	if !ok || n.Sign() < 0 {
		return nil, fmt.Errorf("accounts: %s returned an invalid quantity", method)
	}
	return n, nil
}

func (c rpcWalletClient) ChainID(ctx context.Context) (*big.Int, error) {
	return c.quantity(ctx, "eth_chainId")
}

func (c rpcWalletClient) Nonce(ctx context.Context, address sdktypes.Address) (uint64, error) {
	n, err := c.quantity(ctx, "eth_getTransactionCount", address.Hex(), "pending")
	if err != nil {
		return 0, err
	}
	if !n.IsUint64() {
		return 0, fmt.Errorf("accounts: nonce exceeds uint64")
	}
	return n.Uint64(), nil
}

func (c rpcWalletClient) GasPrice(ctx context.Context) (*big.Int, error) {
	return c.quantity(ctx, "eth_gasPrice")
}

func (c rpcWalletClient) MaxPriorityFeePerGas(ctx context.Context) (*big.Int, error) {
	return c.quantity(ctx, "eth_maxPriorityFeePerGas")
}

func (c rpcWalletClient) EstimateGas(ctx context.Context, message sdktransport.CallMsg) (uint64, error) {
	args := map[string]any{}
	if message.From != nil {
		args["from"] = message.From.Hex()
	}
	if message.To != nil {
		args["to"] = message.To.Hex()
	}
	if message.Gas != 0 {
		args["gas"] = fmt.Sprintf("%#x", message.Gas)
	}
	if message.Value != nil {
		args["value"] = hexQuantity(message.Value)
	}
	if len(message.Data) > 0 {
		args["data"] = "0x" + hex.EncodeToString(message.Data)
	}
	n, err := c.quantity(ctx, "eth_estimateGas", args)
	if err != nil {
		return 0, err
	}
	if !n.IsUint64() {
		return 0, fmt.Errorf("accounts: gas estimate exceeds uint64")
	}
	return n.Uint64(), nil
}

func (c rpcWalletClient) SendRawTransaction(ctx context.Context, raw []byte) (sdktypes.Hash, error) {
	var hash string
	if err := c.Call(ctx, "eth_sendRawTransaction", &hash, "0x"+hex.EncodeToString(raw)); err != nil {
		return sdktypes.Hash{}, err
	}
	bytes, err := hex.DecodeString(strings.TrimPrefix(hash, "0x"))
	if err != nil {
		return sdktypes.Hash{}, fmt.Errorf("accounts: invalid transaction hash: %w", err)
	}
	return sdktypes.BytesToHash(bytes), nil
}

func (c rpcWalletClient) IsBlacklisted(ctx context.Context, address sdktypes.Address) (bool, error) {
	var proof struct {
		Extra string `json:"extra"`
	}
	if err := c.Call(ctx, "eth_getProof", &proof, address.Hex(), []string{}, "latest"); err != nil {
		return false, err
	}
	if proof.Extra == "" {
		return false, nil // Legacy account models omit the SDK's extra flags.
	}
	n, ok := new(big.Int).SetString(strings.TrimPrefix(proof.Extra, "0x"), 16)
	if !ok || !n.IsUint64() {
		return false, fmt.Errorf("accounts: invalid account flags")
	}
	return sdkacct.Decode(n.Uint64()).Blacklisted, nil
}

func openWalletWithRPC(ctx context.Context, key []byte, caller RPCRequester) (Wallet, error) {
	if caller == nil {
		return nil, fmt.Errorf("accounts: nil wallet RPC transport")
	}
	account, err := sdkacct.FromPrivateKeyBytes(key)
	if err != nil {
		return nil, fmt.Errorf("accounts: bad private key: %w", err)
	}
	client := rpcWalletClient{caller}
	if _, err := client.ChainID(ctx); err != nil {
		return nil, fmt.Errorf("accounts: open wallet: %w", err)
	}
	return sdkWallet{client: client, account: account, addr: account.Address().Hex()}, nil
}
