package accounts

import (
	"bytes"
	"math/big"
	"testing"

	sdkcrypto "github.com/0xmhha/accounts/crypto"
	sdktx "github.com/0xmhha/accounts/tx"
	sdktypes "github.com/0xmhha/accounts/types"
)

func TestTamperRawChangesOnlySelectedSignatureComponent(t *testing.T) {
	s, sender, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	p, _, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	one := big.NewInt(1)
	tx, err := feeDelegatedTampered(s, p, "0x0000000000000000000000000000000000000001", one, 8283, 7, one, one, "sender")
	if err != nil {
		t.Fatal(err)
	}
	// Nonempty data/access list also lengthen the nested RLP headers. A raw
	// implementation that rebuilds a plain transfer must fail this control.
	tx.Sender.Data = []byte{0x60, 0x00, 0x60, 0x01, 0x55}
	var target sdktypes.Address
	target[19] = 0x42
	var slot sdktypes.Hash
	slot[31] = 7
	tx.Sender.AccessList = sdktx.AccessList{{Address: target, StorageKeys: []sdktypes.Hash{slot}}}
	sPriv, err := sdkcrypto.PrivKeyFromBytes(s)
	if err != nil {
		t.Fatal(err)
	}
	pPriv, err := sdkcrypto.PrivKeyFromBytes(p)
	if err != nil {
		t.Fatal(err)
	}
	if err = tx.Sign(sPriv, pPriv); err != nil {
		t.Fatal(err)
	}
	got, err := tx.SenderAddress()
	if err != nil || got.Hex() != sender {
		t.Fatal("invalid sender control", err)
	}
	if _, err = tx.RecoverFeePayer(); err != nil {
		t.Fatal("invalid payer control", err)
	}
	raw, err := tx.Encode()
	if err != nil {
		t.Fatal(err)
	}
	original := append([]byte(nil), raw...)
	for _, which := range []string{"sender", "feepayer"} {
		t.Run(which, func(t *testing.T) {
			out, err := TamperFeeDelegatedRaw(raw, which)
			if err != nil {
				t.Fatal(err)
			}
			copyTx := *tx
			if which == "sender" {
				copyTx.Sender.S = new(big.Int).Xor(tx.Sender.S, one)
			} else {
				copyTx.FR = new(big.Int).Xor(tx.FR, one)
			}
			expected, err := copyTx.Encode()
			if err != nil || !bytes.Equal(out, expected) || bytes.Equal(out, original) || !bytes.Equal(raw, original) {
				t.Fatal("mutation changed other fields or its source buffer", err)
			}
			if _, err := copyTx.RecoverFeePayer(); err == nil {
				t.Fatal("payer still verifies the modified envelope")
			}
		})
	}
	for end := range len(raw) {
		if _, err := TamperFeeDelegatedRaw(raw[:end], "sender"); err == nil {
			t.Fatalf("truncation at %d passed", end)
		}
	}
	if _, err := TamperFeeDelegatedRaw(append(append([]byte(nil), raw...), 0), "sender"); err == nil {
		t.Fatal("trailing data passed")
	}
	if _, err := TamperFeeDelegatedRaw(raw, "unknown"); err == nil {
		t.Fatal("unknown signature passed")
	}
	tx.FR = new(big.Int)
	unsigned, _ := tx.Encode()
	if _, err := TamperFeeDelegatedRaw(unsigned, "sender"); err == nil {
		t.Fatal("missing payer signature passed")
	}
}

func FuzzTamperRawNeverPanics(f *testing.F) {
	sender, payer := make([]byte, 32), make([]byte, 32)
	sender[31], payer[31] = 1, 2 // Disposable wire-format fixture keys.
	one := big.NewInt(1)
	raw, err := EncodeFeeDelegatedTampered(sender, payer, "0x0000000000000000000000000000000000000001", one, 8283, 0, one, one, "sender")
	if err != nil {
		f.Fatal(err)
	}
	f.Add(raw)
	for _, seed := range [][]byte{nil, {0x16, 0xc0}, {0x16, 0xff, 0xff, 0xff}, {0x16, 0xf8, 0x01, 0x80}} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		for _, which := range []string{"sender", "feepayer"} {
			original := append([]byte(nil), raw...)
			out, err := TamperFeeDelegatedRaw(raw, which)
			if !bytes.Equal(original, raw) {
				t.Fatal("caller bytes mutated")
			}
			if err == nil {
				if len(out) != len(raw) || out[0] != 0x16 {
					t.Fatal("wire size/type changed")
				}
				differences := 0
				for i := range raw {
					if raw[i] != out[i] {
						differences++
					}
				}
				if differences != 1 {
					t.Fatal("not one byte changed", differences)
				}
			}
		}
	})
}
