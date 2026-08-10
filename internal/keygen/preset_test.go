package keygen

import (
	"strings"
	"testing"
)

func TestParseBootnode(t *testing.T) {
	out := "public key: 0x8d2153cc491bdb0c1d5533c445fec3641ccbc5177ae45ce8029e8a8e33252ccbed4d08d26b6cf25ba5af75f6305615517eb53b41dc6cbd87ecf6da3191ebd16a\n" +
		"address: 0xc17d493883eaa3b4cceb0f214b273392d562f9d8\n" +
		"derived bls public key: 0xa00eb14731965f294993a2df1cf09e5b826193a41853fd9aaa7195922b8461c97b215a1181d4ddecc9f5981fdd47556f\n" +
		"bls PoP (Proof of Possession): 0xa8457c7da3280ac5c6714bcf4d65549f3644e8734fe158fcab352599290fe7671d5afbc1ce9e7e931855b9e08f21d7c40\n"
	n, err := ParseBootnode(out)
	if err != nil {
		t.Fatalf("ParseBootnode: %v", err)
	}
	if n.Address != "0xc17d493883eaa3b4cceb0f214b273392d562f9d8" {
		t.Errorf("address = %q", n.Address)
	}
	if strings.HasPrefix(n.PublicKey, "0x") || len(n.PublicKey) != 128 {
		t.Errorf("publicKey = %q (want bare 128-hex)", n.PublicKey)
	}
	if !strings.HasPrefix(n.BLSPubKey, "0xa00eb147") {
		t.Errorf("blsPubKey = %q", n.BLSPubKey)
	}
	if !strings.HasPrefix(n.BLSPoP, "0xa8457c7d") {
		t.Errorf("blsPoP = %q", n.BLSPoP)
	}
}

func TestParseBootnode_Incomplete(t *testing.T) {
	if _, err := ParseBootnode("address: 0xabc\n"); err == nil {
		t.Error("expected error for incomplete bootnode output")
	}
}
