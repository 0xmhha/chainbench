package accounts

import (
	"fmt"
	"math/big"

	sdkcrypto "github.com/0xmhha/accounts/crypto"
	sdktx "github.com/0xmhha/accounts/tx"
	sdktypes "github.com/0xmhha/accounts/types"
)

// EncodeFeeDelegatedTampered builds and signs a 0x16 fee-delegated transfer, then
// corrupts one signature and returns the raw encoded transaction. It is for
// regression cases that assert the node rejects an invalid sender or fee-payer
// signature; which is "sender" or "feepayer". The caller submits the bytes with
// eth_sendRawTransaction and expects an error.
func EncodeFeeDelegatedTampered(senderKey, feePayerKey []byte, toHex string, amountWei *big.Int, chainID int64, nonce uint64, gasFeeCap, gasTipCap *big.Int, which string) ([]byte, error) {
	tx, err := feeDelegatedTampered(senderKey, feePayerKey, toHex, amountWei, chainID, nonce, gasFeeCap, gasTipCap, which)
	if err != nil {
		return nil, err
	}
	return tx.Encode()
}

// feeDelegatedTampered is EncodeFeeDelegatedTampered before encoding, so a test
// can recover the signers from what it built.
func feeDelegatedTampered(senderKey, feePayerKey []byte, toHex string, amountWei *big.Int, chainID int64, nonce uint64, gasFeeCap, gasTipCap *big.Int, which string) (*sdktx.FeeDelegateTx, error) {
	if amountWei == nil || gasFeeCap == nil || gasTipCap == nil {
		return nil, fmt.Errorf("accounts: nil amount or fee cap")
	}
	sPriv, err := sdkcrypto.PrivKeyFromBytes(senderKey)
	if err != nil {
		return nil, fmt.Errorf("accounts: sender key: %w", err)
	}
	fPriv, err := sdkcrypto.PrivKeyFromBytes(feePayerKey)
	if err != nil {
		return nil, fmt.Errorf("accounts: fee-payer key: %w", err)
	}
	to, err := sdktypes.HexToAddress(toHex)
	if err != nil {
		return nil, fmt.Errorf("accounts: recipient %q: %w", toHex, err)
	}
	feePayerAddr := sdkcrypto.PrivKeyToAddress(fPriv)

	tx := &sdktx.FeeDelegateTx{
		Sender: sdktx.DynamicFeeTx{
			ChainID: big.NewInt(chainID), Nonce: nonce, GasTipCap: gasTipCap, GasFeeCap: gasFeeCap,
			Gas: 21000, To: &to, Value: amountWei,
		},
		FeePayer: &feePayerAddr,
	}
	if err := tx.Sign(sPriv, fPriv); err != nil {
		return nil, fmt.Errorf("accounts: sign fee-delegated tx: %w", err)
	}

	switch which {
	case "sender":
		// S, not R. A changed R is a valid x-coordinate about half the time, so
		// the node refused either at sender recovery ("invalid sender") or, when
		// recovery found some other address, at the fee-payer signature that
		// covers the sender's ("invalid feePayer") — a coin toss per run. A
		// changed S always recovers, to another address, so the refusal is
		// always the fee-payer check: the one check that can tell an altered
		// sender signature from a genuine one.
		tx.Sender.S = corruptSig(tx.Sender.S)
	case "feepayer":
		tx.FR = corruptSig(tx.FR)
	default:
		return nil, fmt.Errorf("accounts: which must be \"sender\" or \"feepayer\", got %q", which)
	}
	return tx, nil
}

// AddressOf returns the 0x-prefixed hex address of a raw secp256k1 private key.
// It is the address the key signs as, used to read that account's nonce before
// building a raw transaction for it.
func AddressOf(key []byte) (string, error) {
	priv, err := sdkcrypto.PrivKeyFromBytes(key)
	if err != nil {
		return "", fmt.Errorf("accounts: private key: %w", err)
	}
	return sdkcrypto.PrivKeyToAddress(priv).Hex(), nil
}

// corruptSig flips the low byte of a signature component so it no longer recovers
// the original signer.
func corruptSig(v *big.Int) *big.Int {
	if v == nil {
		return big.NewInt(1)
	}
	return new(big.Int).Xor(v, big.NewInt(0xff))
}
