package deploy

import (
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/keyring/derive"
)

func TestFormatAccountsFragment(t *testing.T) {
	frag := FormatAccountsFragment([]ServerIdentity{{
		Server: 3,
		Identity: derive.Identity{
			Address: "0xabc",
			BLS:     &derive.BLS{PublicKey: "0xbbb", PoP: "0xppp"},
		},
	}})
	for _, want := range []string{"validators:", "server: 3", `addr: "0xabc"`, `bls: "0xbbb"`, `bls_pop: "0xppp"`} {
		if !strings.Contains(frag, want) {
			t.Errorf("fragment missing %q:\n%s", want, frag)
		}
	}
}

// TestFormatAccountsFragment_NoBLS covers a poa server, whose identity has no
// BLS material: the fragment must omit the keys rather than emit empty ones a
// reader would take for real values.
func TestFormatAccountsFragment_NoBLS(t *testing.T) {
	frag := FormatAccountsFragment([]ServerIdentity{{
		Server:   1,
		Identity: derive.Identity{Address: "0xabc"},
	}})
	if strings.Contains(frag, "bls:") || strings.Contains(frag, "bls_pop:") {
		t.Errorf("fragment emitted empty BLS fields:\n%s", frag)
	}
	if !strings.Contains(frag, `addr: "0xabc"`) {
		t.Errorf("fragment lost the address:\n%s", frag)
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
	if p.Nodekey != "/data/go-wbft/conf/nodekey" {
		t.Errorf("default paths: %+v", p)
	}
	c2 := &Cluster{RemotePaths: RemotePaths{Nodekey: "/custom/nodekey"}}
	if p2 := c2.Paths(); p2.Nodekey != "/custom/nodekey" || p2.CoinbaseKeystore != "/data/go-wbft/conf/keystore/coinbase" {
		t.Errorf("override + default: %+v", p2)
	}
}
