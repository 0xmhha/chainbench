package accounts

import (
	"context"
	"fmt"

	sdkacct "github.com/0xmhha/accounts/account"
	sdktx "github.com/0xmhha/accounts/tx"
	sdktypes "github.com/0xmhha/accounts/types"
)

func (s sdkWallet) SendFeeDelegatedTx(ctx context.Context, feePayerKey []byte, args DynamicTxArgs) (string, error) {
	if args.Value == nil || args.GasFeeCap == nil || args.GasTipCap == nil || args.Value.Sign() < 0 || args.GasFeeCap.Sign() < 0 || args.GasTipCap.Sign() < 0 {
		return "", fmt.Errorf("accounts: invalid value or fee cap")
	}
	feePayer, err := sdkacct.FromPrivateKeyBytes(feePayerKey)
	if err != nil {
		return "", fmt.Errorf("accounts: bad fee-payer key: %w", err)
	}
	var to *sdktypes.Address
	if args.ToHex != "" {
		address, err := sdktypes.HexToAddress(args.ToHex)
		if err != nil {
			return "", err
		}
		to = &address
	}
	list, err := sdkAccessList(args.AccessList)
	if err != nil {
		return "", err
	}
	nonce := uint64(0)
	if args.Nonce != nil {
		nonce = *args.Nonce
	} else {
		nonce, err = s.client.Nonce(ctx, s.account.Address())
		if err != nil {
			return "", err
		}
	}
	chainID, err := s.client.ChainID(ctx)
	if err != nil {
		return "", err
	}
	tx := sdktx.FeeDelegateTx{Sender: sdktx.DynamicFeeTx{ChainID: chainID, Nonce: nonce, Gas: args.Gas, GasTipCap: args.GasTipCap, GasFeeCap: args.GasFeeCap, Value: args.Value, Data: args.Data, To: to, AccessList: list}}
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
