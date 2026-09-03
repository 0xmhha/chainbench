package accounts_test

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/accounts"
)

// TestCreateAddress pins the CREATE address computation against the canonical
// Ethereum vectors for deployer 0x6ac7ea…dbf0 at nonces 0..3 (keccak(rlp(addr,
// nonce))). These are stable regardless of chain, so a spec can rely on them.
func TestCreateAddress(t *testing.T) {
	const deployer = "0x6ac7ea33f8831ea9dcc53393aaa88b25a785dbf0"
	cases := []struct {
		nonce uint64
		want  string
	}{
		{0, "0xcd234a471b72ba2f1ccf0a70fcaba648a5eecd8d"},
		{1, "0x343c43a37d37dff08ae8c4a11544c718abb4fcf8"},
		{2, "0xf778b86fa74e846c4f0a1fbd1335fe81c00a0c91"},
		{3, "0xfffd933a0bc612844eaf0c6fe3e5b8e9b6c1d19c"},
	}
	for _, tc := range cases {
		got, err := accounts.CreateAddress(deployer, tc.nonce)
		if err != nil {
			t.Fatalf("nonce %d: %v", tc.nonce, err)
		}
		if !strings.EqualFold(got, tc.want) {
			t.Errorf("nonce %d = %s, want %s", tc.nonce, got, tc.want)
		}
	}
}

func TestCreateAddress_RejectsBadDeployer(t *testing.T) {
	if _, err := accounts.CreateAddress("not-an-address", 0); err == nil {
		t.Fatal("want an error for an unparseable deployer")
	}
}
