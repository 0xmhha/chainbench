package keyring

import (
	"encoding/hex"
	"fmt"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"

	"github.com/0xmhha/chainbench/internal/accounts"
)

// Derivation selects how much of an identity to derive.
//
// BLS material costs real computation and only the wbft family consumes it, so
// it is opt-in. Asking for less is not a degraded identity — a wemix node has
// no BLS key, and modelling that as absence rather than as zeroes keeps the two
// cases distinguishable downstream.
type Derivation int

const (
	// AccountOnly derives the address and the devp2p public key.
	AccountOnly Derivation = iota
	// WithBLS additionally derives the BLS public key and its proof of
	// possession.
	WithBLS
)

// String names the derivation for logs and errors.
func (d Derivation) String() string {
	switch d {
	case AccountOnly:
		return "account-only"
	case WithBLS:
		return "with-bls"
	default:
		return fmt.Sprintf("Derivation(%d)", int(d))
	}
}

// BLS is a node's BLS12-381 material, used by the wbft consensus family to
// aggregate votes.
type BLS struct {
	// PublicKey is the 0x-prefixed compressed G1 point (48 bytes).
	PublicKey string
	// PoP is the 0x-prefixed proof of possession: the public key signed by its
	// own secret, as a compressed G2 point (96 bytes). It proves the holder
	// knows the secret behind PublicKey, which is what stops a rogue-key attack
	// on an aggregated signature.
	PoP string
}

// Identity is everything public that derives from one [PrivateKey]. It holds no
// secret, so it is safe to log, serialize, and pass across hosts.
type Identity struct {
	// PublicKey is the 128-hex devp2p public key, without a 0x prefix — the
	// form that appears in an enode URL.
	PublicKey string
	// Address is the 0x-prefixed account address.
	Address string
	// BLS is nil unless the identity was derived [WithBLS]. Absence means the
	// chain does not use BLS, and is deliberately distinct from a zero key.
	BLS *BLS
}

// Derive computes the public identity of k.
//
// Everything is computed in process: no chain binary is executed, so this works
// with no build of go-wbft present and with CGO disabled. The result is checked
// byte for byte against the shipped keys/preset fixture.
func Derive(k PrivateKey, d Derivation) (Identity, error) {
	raw := k.Bytes()

	address, err := accounts.AddressForKey(raw)
	if err != nil {
		return Identity{}, fmt.Errorf("keyring: derive address: %w", err)
	}

	// The devp2p public key is the uncompressed secp256k1 point with its 0x04
	// tag byte stripped — the 128-hex form that goes into an enode URL.
	priv := secp256k1.PrivKeyFromBytes(raw)
	uncompressed := priv.PubKey().SerializeUncompressed()

	id := Identity{
		PublicKey: hex.EncodeToString(uncompressed[1:]),
		Address:   address,
	}

	if d == WithBLS {
		bls, err := deriveBLS(k)
		if err != nil {
			return Identity{}, fmt.Errorf("keyring: derive bls: %w", err)
		}
		id.BLS = &bls
	}
	return id, nil
}
