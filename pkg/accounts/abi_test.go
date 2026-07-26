package accounts_test

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/pkg/accounts"
)

func TestSelector(t *testing.T) {
	// Known ERC-20 selectors.
	cases := map[string]string{
		"totalSupply()":             "0x18160ddd",
		"balanceOf(address)":        "0x70a08231",
		"transfer(address,uint256)": "0xa9059cbb",
	}
	for sig, want := range cases {
		if got := accounts.Selector(sig); got != want {
			t.Errorf("Selector(%q) = %s, want %s", sig, got, want)
		}
	}
}

func TestEncodeCall(t *testing.T) {
	// no-arg call is just the selector.
	if got := accounts.EncodeCall("totalSupply()"); got != "0x18160ddd" {
		t.Errorf("no-arg encode = %s", got)
	}
	// balanceOf(address): selector + 32-byte left-padded address.
	addr := accounts.AddressArg("0x00000000000000000000000000000000000000ff")
	got := accounts.EncodeCall("balanceOf(address)", addr)
	want := "0x70a08231" + strings.Repeat("0", 62) + "ff"
	if got != want {
		t.Errorf("balanceOf encode:\n got=%s\nwant=%s", got, want)
	}
}
