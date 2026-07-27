package accounts

import (
	"math/big"
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
