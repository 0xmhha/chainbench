package accounts

import (
	"math/big"
	"strings"
	"testing"
)

func TestEncodeFeeDelegatedTampered(t *testing.T) {
	sk, _, err := GenerateKey()
	if err != nil {
		t.Fatalf("gen sender key: %v", err)
	}
	fk, _, err := GenerateKey()
	if err != nil {
		t.Fatalf("gen fee-payer key: %v", err)
	}
	const to = "0x0000000000000000000000000000000000000001"
	one := big.NewInt(1)

	// invalid which
	if _, err := EncodeFeeDelegatedTampered(sk, fk, to, one, 1, 0, one, one, "bad"); err == nil {
		t.Error("bad which should error")
	}
	// nil fee cap
	if _, err := EncodeFeeDelegatedTampered(sk, fk, to, one, 1, 0, nil, one, "sender"); err == nil {
		t.Error("nil fee cap should error")
	}
	// valid sender/feepayer tampering produces a 0x16-typed envelope.
	for _, which := range []string{"sender", "feepayer"} {
		raw, err := EncodeFeeDelegatedTampered(sk, fk, to, one, 1, 0, one, one, which)
		if err != nil || len(raw) == 0 || raw[0] != 0x16 {
			t.Errorf("%s: raw=%x err=%v, want non-empty 0x16 envelope", which, raw, err)
		}
	}
}

// TestFeeDelegatedTampered_SenderAlwaysRecovers: a tampered sender signature
// must still recover — to some other address — on every key, so the node's
// refusal is always the fee-payer check and never a per-run toss between that
// and a failed sender recovery.
func TestFeeDelegatedTampered_SenderAlwaysRecovers(t *testing.T) {
	const to = "0x0000000000000000000000000000000000000001"
	one := big.NewInt(1)
	for i := 0; i < 64; i++ {
		sk, senderAddr, err := GenerateKey()
		if err != nil {
			t.Fatalf("gen sender key: %v", err)
		}
		fk, _, err := GenerateKey()
		if err != nil {
			t.Fatalf("gen fee-payer key: %v", err)
		}
		tx, err := feeDelegatedTampered(sk, fk, to, one, 1, 0, one, one, "sender")
		if err != nil {
			t.Fatalf("build: %v", err)
		}
		got, err := tx.SenderAddress()
		if err != nil {
			t.Fatalf("key %d: tampered sender signature did not recover: %v", i, err)
		}
		if strings.EqualFold(got.Hex(), senderAddr) {
			t.Fatalf("key %d: tampered sender signature still recovers the sender", i)
		}
		if _, err := tx.RecoverFeePayer(); err == nil {
			t.Fatalf("key %d: fee-payer signature still verifies over a tampered sender signature", i)
		}
	}
}
