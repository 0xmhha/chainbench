package wbft

import (
	"math/big"
	"testing"

	bls "github.com/kilic/bls12-381"
	"golang.org/x/crypto/sha3"
)

func sealHash(raw []byte) []byte {
	h := sha3.NewLegacyKeccak256()
	_, _ = h.Write(raw)
	return h.Sum(nil)
}

func signedHeaderFixture(t *testing.T, stable ...bool) ([]byte, [][]byte) {
	t.Helper()
	g1, g2 := bls.NewG1(), bls.NewG2()
	keys := [][]byte{}
	for i := int64(1); i <= 4; i++ {
		p := g1.New()
		g1.MulScalarBig(p, g1.One(), big.NewInt(i))
		keys = append(keys, g1.ToCompressed(p))
	}
	extra := []any{[]byte{}, make([]byte, 65), uint64(0), []any{}, []any{}, uint64(0), []any{}, []any{}, []any{}}
	if len(stable) > 0 && stable[0] {
		extra[8] = uint64(initialGasTip)
		extra = append(extra, []any{})
	}
	header := []any{make([]byte, 32), make([]byte, 32), make([]byte, 20), make([]byte, 32), make([]byte, 32), make([]byte, 32), make([]byte, 256), uint64(1), uint64(25), uint64(30000000), uint64(0), uint64(1025), rlpEncode(extra), make([]byte, 32), make([]byte, 8), uint64(100)}
	msg := sealHash(append(sealHash(rlpEncode(header)), byte(1)))
	point, err := g2.HashToCurve(msg, []byte("BLS_SIG_BLS12381G2_XMD:SHA-256_SSWU_RO_POP_"))
	if err != nil {
		t.Fatal(err)
	}
	sig := g2.New()
	g2.MulScalarBig(sig, point, big.NewInt(6))
	extra[7] = []any{[]byte{7}, g2.ToCompressed(sig)}
	header[12] = rlpEncode(extra)
	return rlpEncode(header), keys
}

func TestCommittedSealRejectsPresenceOnlyEvidence(t *testing.T) {
	for _, name := range []string{"valid", "changed signature", "wrong bitmap", "insufficient quorum", "outside bitmap", "wrong header", "wrong round", "wrong key", "infinity signature", "duplicate keys"} {
		t.Run(name, func(t *testing.T) {
			raw, keys := signedHeaderFixture(t)
			h, _, _ := rlpDecode(raw)
			e, _, _ := rlpDecode(h.list[12].bytes)
			switch name {
			case "changed signature":
				e.list[7].list[1].bytes[10] ^= 1
			case "wrong bitmap":
				e.list[7].list[0].bytes = []byte{14}
			case "insufficient quorum":
				e.list[7].list[0].bytes = []byte{3}
			case "outside bitmap":
				e.list[7].list[0].bytes = []byte{135}
			case "wrong header":
				h.list[11].bytes = []byte{4}
			case "wrong round":
				e.list[5].bytes = []byte{1}
			case "wrong key":
				keys[0] = keys[3]
			case "infinity signature":
				e.list[7].list[1].bytes = make([]byte, 96)
				e.list[7].list[1].bytes[0] = 0xc0
			case "duplicate keys":
				keys[1] = keys[0]
			}
			h.list[12].bytes = encodeSealNode(e)
			err := VerifyCommittedHeader(encodeSealNode(h), keys)
			if name == "valid" && err != nil {
				t.Fatal(err)
			}
			if name != "valid" && err == nil {
				t.Fatal("presence-only aggregate accepted invalid cryptographic context")
			}
		})
	}
}

func TestCommittedSealStableExtraAndMalformedInputs(t *testing.T) {
	raw, keys := signedHeaderFixture(t, true)
	if err := VerifyCommittedHeader(raw, keys); err != nil {
		t.Fatal("ten-field stable extra rejected", err)
	}
	for _, raw := range [][]byte{nil, []byte{0xc0}, append(append([]byte(nil), raw...), 0), make([]byte, 16385)} {
		if VerifyCommittedHeader(raw, keys) == nil {
			t.Fatal("invalid raw header accepted")
		}
	}
	_, keys = signedHeaderFixture(t)
	keys[0] = make([]byte, 48)
	keys[0][0] = 0xc0
	if VerifyCommittedHeader(raw, keys) == nil {
		t.Fatal("infinite public key accepted")
	}
}
