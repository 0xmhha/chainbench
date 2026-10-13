package accounts

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	sdktx "github.com/0xmhha/accounts/tx"
	sdktypes "github.com/0xmhha/accounts/types"
)

func (s sdkWallet) SendLegacyTx(ctx context.Context, args LegacyTxArgs) (string, error) {
	if args.Value == nil || args.GasPrice == nil || args.Value.Sign() < 0 || args.GasPrice.Sign() < 0 {
		return "", fmt.Errorf("accounts: invalid value or gas price")
	}
	var to *sdktypes.Address
	if args.ToHex != "" {
		address, err := sdktypes.HexToAddress(args.ToHex)
		if err != nil {
			return "", fmt.Errorf("accounts: recipient: %w", err)
		}
		to = &address
	}
	list := make(sdktx.AccessList, len(args.AccessList))
	for i, tuple := range args.AccessList {
		address, err := sdktypes.HexToAddress(tuple.Address)
		if err != nil {
			return "", fmt.Errorf("accounts: accessList address: %w", err)
		}
		list[i].Address = address
		for _, key := range tuple.StorageKeys {
			bytes, err := hex.DecodeString(strings.TrimPrefix(key, "0x"))
			if err != nil || len(bytes) != 32 || !strings.HasPrefix(key, "0x") {
				return "", fmt.Errorf("accounts: accessList storage key must be 32-byte hex")
			}
			list[i].StorageKeys = append(list[i].StorageKeys, sdktypes.BytesToHash(bytes))
		}
	}
	nonce := uint64(0)
	if args.Nonce != nil {
		nonce = *args.Nonce
	} else {
		n, err := s.client.Nonce(ctx, s.account.Address())
		if err != nil {
			return "", fmt.Errorf("accounts: nonce: %w", err)
		}
		nonce = n
	}
	chainID, err := s.client.ChainID(ctx)
	if err != nil {
		return "", fmt.Errorf("accounts: chain id: %w", err)
	}
	var encoded []byte
	if args.AccessList == nil {
		tx := sdktx.LegacyTx{Nonce: nonce, GasPrice: args.GasPrice, Gas: args.Gas, To: to, Value: args.Value, Data: args.Data}
		if err := tx.Sign(chainID, s.account.PrivateKey()); err != nil {
			return "", err
		}
		encoded = tx.Encode()
	} else {
		tx := sdktx.AccessListTx{ChainID: chainID, Nonce: nonce, GasPrice: args.GasPrice, Gas: args.Gas, To: to, Value: args.Value, Data: args.Data, AccessList: list}
		if err := tx.Sign(s.account.PrivateKey()); err != nil {
			return "", err
		}
		encoded = tx.Encode()
	}
	hash, err := s.client.SendRawTransaction(ctx, encoded)
	if err != nil {
		return "", err
	}
	return hash.Hex(), nil
}
