package derive

import (
	"crypto/hkdf"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"

	bls12381 "github.com/kilic/bls12-381"
)

// BLS derivation constants, all fixed by the specs the chains implement.
const (
	// blsKeyGenSalt is the initial salt of KeyGen in
	// draft-irtf-cfrg-bls-signature (EIP-2333).
	blsKeyGenSalt = "BLS-SIG-KEYGEN-SALT-"

	// blsPoPDST is the domain separation tag for proof-of-possession
	// signatures. Signing without it produces a proof of the right *shape*
	// that fails verification on chain, which is why the golden test pins it.
	blsPoPDST = "BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_POP_"

	// blsCurveOrder is r, the order of the BLS12-381 scalar field.
	blsCurveOrder = "73eda753299d7d483339d80809a1d80553bda402fffe5bfeffffffff00000001"

	// blsKeyGenOKMLen is L from the spec: the HKDF output length whose value
	// also seeds the expansion info.
	blsKeyGenOKMLen = 48

	// blsKeyGenAttempts bounds the KeyGen retry loop. The spec retries when the
	// derived scalar is zero, which has never been observed; the bound keeps a
	// broken hash from spinning forever.
	blsKeyGenAttempts = 8
)

// deriveBLS computes the BLS public key and proof of possession for k.
//
// This mirrors blst's blst_keygen so that output matches the go-wbft bootnode
// tool byte for byte. The subtle part is version 4 of that implementation: the
// salt is hashed once *before* the first iteration, not only between retries.
// Getting that wrong yields a well-formed key that no wbft node will accept.
func deriveBLS(k PrivateKey) (BLS, error) {
	secret, err := blsKeyGen(k.Bytes())
	if err != nil {
		return BLS{}, err
	}

	// Public key: secret * G1 generator, compressed to 48 bytes.
	g1 := bls12381.NewG1()
	public := g1.New()
	g1.MulScalarBig(public, g1.One(), secret)
	publicBytes := g1.ToCompressed(public)

	// Proof of possession: sign the public key with the secret behind it. The
	// message is hashed onto G2 under the POP tag, so a signature made for any
	// other purpose cannot be replayed as a proof.
	g2 := bls12381.NewG2()
	message, err := g2.HashToCurve(publicBytes, []byte(blsPoPDST))
	if err != nil {
		return BLS{}, fmt.Errorf("hash public key to G2: %w", err)
	}
	signature := g2.New()
	g2.MulScalarBig(signature, message, secret)

	return BLS{
		PublicKey: "0x" + hex.EncodeToString(publicBytes),
		PoP:       "0x" + hex.EncodeToString(g2.ToCompressed(signature)),
	}, nil
}

// blsKeyGen is KeyGen from draft-irtf-cfrg-bls-signature:
//
//	SK = OS2IP(HKDF-Expand(HKDF-Extract(salt, IKM || I2OSP(0,1)),
//	                       "" || I2OSP(L,2), L)) mod r
//
// with the salt rehashed each round, matching blst's version 4.
func blsKeyGen(ikm []byte) (*big.Int, error) {
	order, ok := new(big.Int).SetString(blsCurveOrder, 16)
	if !ok {
		return nil, fmt.Errorf("keyring: bad curve order constant")
	}

	info := make([]byte, 2)
	binary.BigEndian.PutUint16(info, uint16(blsKeyGenOKMLen))

	// IKM is the key material with a single zero byte appended, per the spec.
	keyMaterial := make([]byte, 0, len(ikm)+1)
	keyMaterial = append(keyMaterial, ikm...)
	keyMaterial = append(keyMaterial, 0x00)

	// Version 4 hashes the salt before the first round, not only on retry.
	digest := sha256.Sum256([]byte(blsKeyGenSalt))
	salt := digest[:]

	for range blsKeyGenAttempts {
		prk, err := hkdf.Extract(sha256.New, keyMaterial, salt)
		if err != nil {
			return nil, fmt.Errorf("hkdf extract: %w", err)
		}
		okm, err := hkdf.Expand(sha256.New, prk, string(info), blsKeyGenOKMLen)
		if err != nil {
			return nil, fmt.Errorf("hkdf expand: %w", err)
		}
		secret := new(big.Int).Mod(new(big.Int).SetBytes(okm), order)
		if secret.Sign() != 0 {
			return secret, nil
		}
		digest = sha256.Sum256(salt)
		salt = digest[:]
	}
	return nil, fmt.Errorf("keyring: bls keygen produced only zero secrets")
}
