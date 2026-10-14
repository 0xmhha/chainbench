package wbft

import (
	"bytes"
	"fmt"

	bls "github.com/kilic/bls12-381"
	"golang.org/x/crypto/sha3"
)

// VerifyCommittedHeader verifies a raw WBFT header against validator-order BLS
// keys for its current epoch. The caller must establish that epoch context.
// It supports the WBFT nine-field and StableNet ten-field extra-data forms.
func VerifyCommittedHeader(rawHeader []byte, keys [][]byte) error {
	if len(rawHeader) > 16384 || len(keys) == 0 || len(keys) > 1024 {
		return fmt.Errorf("invalid header/validator bounds")
	}
	header, rest, err := rlpDecode(rawHeader)
	if err != nil || len(rest) != 0 || !header.isList || len(header.list) < 15 || len(header.list) > 24 || !bytes.Equal(encodeSealNode(header), rawHeader) {
		return fmt.Errorf("invalid canonical header RLP")
	}
	if header.list[12].isList {
		return fmt.Errorf("header extraData is not bytes")
	}
	extra, rest, err := rlpDecode(header.list[12].bytes)
	if err != nil || len(rest) != 0 || !extra.isList || (len(extra.list) != 9 && len(extra.list) != 10) || !bytes.Equal(encodeSealNode(extra), header.list[12].bytes) {
		return fmt.Errorf("invalid canonical WBFT extra")
	}
	if extra.list[5].isList || len(extra.list[5].bytes) > 4 {
		return fmt.Errorf("invalid round")
	}
	seal := extra.list[7]
	if !seal.isList || len(seal.list) != 2 || seal.list[0].isList || seal.list[1].isList || len(seal.list[1].bytes) != 96 || len(seal.list[0].bytes) > (len(keys)+7)/8 {
		return fmt.Errorf("invalid aggregate shape")
	}
	g1, g2 := bls.NewG1(), bls.NewG2()
	points := make([]*bls.PointG1, len(keys))
	seen := map[string]bool{}
	for i, key := range keys {
		if len(key) != 48 || seen[string(key)] {
			return fmt.Errorf("invalid/duplicate validator key %d", i)
		}
		seen[string(key)] = true
		p, err := g1.FromCompressed(key)
		if err != nil || g1.IsZero(p) {
			return fmt.Errorf("invalid validator point %d", i)
		}
		points[i] = p
	}
	aggregate := g1.Zero()
	count := 0
	for i, b := range seal.list[0].bytes {
		for bit := 0; bit < 8; bit++ {
			if b&(1<<bit) == 0 {
				continue
			}
			index := i*8 + bit
			if index >= len(keys) {
				return fmt.Errorf("signer outside validator set")
			}
			g1.Add(aggregate, aggregate, points[index])
			count++
		}
	}
	if count < 2*len(keys)/3+1 || g1.IsZero(aggregate) {
		return fmt.Errorf("insufficient/nonfinite signer aggregate")
	}
	signature, err := g2.FromCompressed(seal.list[1].bytes)
	if err != nil || g2.IsZero(signature) {
		return fmt.Errorf("invalid aggregate signature point")
	}
	extra.list[6] = rlpNode{isList: true}
	extra.list[7] = rlpNode{isList: true}
	header.list[12].bytes = encodeSealNode(extra)
	digest := sha3.NewLegacyKeccak256()
	_, _ = digest.Write(encodeSealNode(header))
	hash := digest.Sum(nil)
	digest.Reset()
	_, _ = digest.Write(append(hash, byte(1)))
	message := digest.Sum(nil)
	point, err := g2.HashToCurve(message, []byte("BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_POP_"))
	if err != nil {
		return fmt.Errorf("hash commit message: %w", err)
	}
	if !bls.NewEngine().AddPair(aggregate, point).AddPairInv(g1.One(), signature).Check() {
		return fmt.Errorf("invalid BLS committed seal")
	}
	return nil
}

func encodeSealNode(n rlpNode) []byte {
	if !n.isList {
		return rlpString(n.bytes)
	}
	var payload []byte
	for _, child := range n.list {
		payload = append(payload, encodeSealNode(child)...)
	}
	return append(rlpLen(len(payload), 0xc0), payload...)
}
