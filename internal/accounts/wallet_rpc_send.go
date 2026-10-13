package accounts

import (
	"context"
	"fmt"
	"math/big"

	sdkacct "github.com/0xmhha/accounts/account"
	sdktransport "github.com/0xmhha/accounts/transport"
	sdktx "github.com/0xmhha/accounts/tx"
	sdktypes "github.com/0xmhha/accounts/types"
)

// These SDK-backed operations use the caller-owned RPC transport. Public SDK
// wallets retain their existing high-level behavior; no global client changes.
func (s sdkWallet) guardTransfer(ctx context.Context, to *sdktypes.Address, value *big.Int) error {
	if err := sdktx.GuardValueTransfer(to, value); err != nil {
		return err
	}
	for index, address := range []*sdktypes.Address{addressPointer(s.account.Address()), to} {
		if address == nil {
			continue
		}
		label := "sender"
		if index == 1 {
			label = "recipient"
		}
		blocked, err := s.client.IsBlacklisted(ctx, *address)
		if err != nil {
			return fmt.Errorf("blacklist check (%s): %w", label, err)
		}
		if blocked {
			return fmt.Errorf("%s %s is blacklisted", label, address.Hex())
		}
	}
	return nil
}

func addressPointer(value sdktypes.Address) *sdktypes.Address { return &value }

func (s sdkWallet) autoDynamic(ctx context.Context, to *sdktypes.Address, value *big.Int, data []byte, gas uint64) (string, uint64, error) {
	if value == nil {
		value = new(big.Int)
	}
	nonce, err := s.client.Nonce(ctx, s.account.Address())
	if err != nil {
		return "", 0, err
	}
	tip, err := s.client.MaxPriorityFeePerGas(ctx)
	if err != nil {
		return "", 0, err
	}
	price, err := s.client.GasPrice(ctx)
	if err != nil {
		return "", 0, err
	}
	args := DynamicTxArgs{Value: value, Data: data, Gas: gas, Nonce: &nonce, GasTipCap: tip, GasFeeCap: new(big.Int).Add(price, tip)}
	if to != nil {
		args.ToHex = to.Hex()
	}
	hash, err := s.SendDynamicFeeTx(ctx, args)
	return hash, nonce, err
}

func (s sdkWallet) estimatedGas(ctx context.Context, to *sdktypes.Address, value *big.Int, data []byte, fallback uint64) uint64 {
	from := s.account.Address()
	gas, err := s.client.EstimateGas(ctx, sdktransport.CallMsg{From: &from, To: to, Value: value, Data: data})
	if err != nil {
		return fallback // Preserve the SDK high-level deploy/execute fallback.
	}
	return gas * 12 / 10
}

func (s sdkWallet) rpcFeeDelegated(ctx context.Context, feePayer *sdkacct.Account, to sdktypes.Address, value *big.Int) (string, error) {
	if err := s.guardTransfer(ctx, &to, value); err != nil {
		return "", err
	}
	nonce, err := s.client.Nonce(ctx, s.account.Address())
	if err != nil {
		return "", err
	}
	chainID, err := s.client.ChainID(ctx)
	if err != nil {
		return "", err
	}
	tip, err := s.client.MaxPriorityFeePerGas(ctx)
	if err != nil {
		return "", err
	}
	price, err := s.client.GasPrice(ctx)
	if err != nil {
		return "", err
	}
	tx := sdktx.FeeDelegateTx{Sender: sdktx.DynamicFeeTx{ChainID: chainID, Nonce: nonce, Gas: 21000, To: &to, Value: value, GasTipCap: tip, GasFeeCap: new(big.Int).Add(price, tip)}}
	if err := tx.Sign(s.account.PrivateKey(), feePayer.PrivateKey()); err != nil {
		return "", err
	}
	raw, err := tx.Encode()
	if err != nil {
		return "", err
	}
	hash, err := s.client.SendRawTransaction(ctx, raw)
	if err != nil {
		return "", err
	}
	return hash.Hex(), nil
}
