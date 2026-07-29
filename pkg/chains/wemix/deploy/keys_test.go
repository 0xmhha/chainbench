package deploy

import (
	"strings"
	"testing"
)

func TestParseBootnodeOutput(t *testing.T) {
	out := `Some log line
address: 0x1234567890abcdef1234567890abcdef12345678
derived bls public key: 0xaaaabbbbccccdddd
bls PoP (Proof of Possession): 0x1111222233334444
`
	info, err := ParseBootnodeOutput(out)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if info.Address != "0x1234567890abcdef1234567890abcdef12345678" {
		t.Errorf("address = %q", info.Address)
	}
	if info.BLSPubKey != "0xaaaabbbbccccdddd" {
		t.Errorf("bls pubkey = %q", info.BLSPubKey)
	}
	if info.BLSPoP != "0x1111222233334444" {
		t.Errorf("bls pop = %q", info.BLSPoP)
	}
}

func TestParseBootnodeOutput_NoAddress(t *testing.T) {
	if _, err := ParseBootnodeOutput("no useful lines here\n"); err == nil {
		t.Error("expected error when no address present")
	}
}

func TestFormatAccountsFragment(t *testing.T) {
	frag := FormatAccountsFragment([]NodeKeyInfo{
		{Server: 3, Address: "0xabc", BLSPubKey: "0xbbb", BLSPoP: "0xppp"},
	})
	for _, want := range []string{"validators:", "server: 3", `addr: "0xabc"`, `bls: "0xbbb"`, `bls_pop: "0xppp"`} {
		if !strings.Contains(frag, want) {
			t.Errorf("fragment missing %q:\n%s", want, frag)
		}
	}
}

func TestShellQuote(t *testing.T) {
	if got := shellQuote("/data/go-wbft/conf/nodekey"); got != "'/data/go-wbft/conf/nodekey'" {
		t.Errorf("shellQuote = %q", got)
	}
	// an embedded single quote is escaped, not left to break the quoting.
	if got := shellQuote("a'b"); strings.Contains(got[1:len(got)-1], "'b") && !strings.Contains(got, `'\''`) {
		t.Errorf("shellQuote did not escape quote: %q", got)
	}
}

func TestPaths_Defaults(t *testing.T) {
	c := &Cluster{}
	p := c.Paths()
	if p.Nodekey != "/data/go-wbft/conf/nodekey" || p.Bootnode != "bootnode" {
		t.Errorf("default paths: %+v", p)
	}
	c2 := &Cluster{RemotePaths: RemotePaths{Nodekey: "/custom/nodekey"}}
	if p2 := c2.Paths(); p2.Nodekey != "/custom/nodekey" || p2.CoinbaseKeystore != "/data/go-wbft/conf/keystore/coinbase" {
		t.Errorf("override + default: %+v", p2)
	}
}
