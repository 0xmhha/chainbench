//go:build e2e

package main

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"
)

type ncpWhitespaceProof struct {
	RawCSV, BlockTag, Count, List, ZeroMember string
	Fork, Height, Later                       uint64
	UseNCP                                    bool
	Members                                   map[string]string
}

func verifyNCPWhitespace(p ncpWhitespaceProof) error {
	if !p.UseNCP || p.Height == 0 || p.Height <= p.Fork || p.Later <= p.Height || p.Later-p.Height < 2 || p.BlockTag != fmt.Sprintf("0x%x", p.Height) {
		return fmt.Errorf("NCP network not active/producing or query not pinned")
	}
	tokens := strings.Split(p.RawCSV, ",")
	if len(tokens) != 7 || len(p.Members) != 7 {
		return fmt.Errorf("missing seven-member input/queries")
	}
	wanted := map[string]bool{}
	padded := false
	for _, raw := range tokens {
		trimmed := strings.TrimSpace(raw)
		padded = padded || raw != trimmed
		addr := strings.ToLower(trimmed)
		if !strings.HasPrefix(addr, "0x") {
			return fmt.Errorf("invalid NCP address")
		}
		b, err := hex.DecodeString(addr[2:])
		if err != nil || len(b) != 20 || new(big.Int).SetBytes(b).Sign() == 0 || wanted[addr] {
			return fmt.Errorf("invalid/duplicate NCP input")
		}
		wanted[addr] = true
		if p.Members[addr] != "0x"+fmt.Sprintf("%064x", 1) {
			return fmt.Errorf("trimmed NCP not an actual member")
		}
	}
	if !padded || p.Count != "0x"+fmt.Sprintf("%064x", 7) || p.ZeroMember != "0x"+fmt.Sprintf("%064x", 0) {
		return fmt.Errorf("wrong count/zero-address control or unpadded fixture")
	}
	if !strings.HasPrefix(p.List, "0x") {
		return fmt.Errorf("missing member list")
	}
	raw, err := hex.DecodeString(p.List[2:])
	if err != nil || len(raw) != 9*32 || new(big.Int).SetBytes(raw[:32]).Cmp(big.NewInt(32)) != 0 || new(big.Int).SetBytes(raw[32:64]).Cmp(big.NewInt(7)) != 0 {
		return fmt.Errorf("invalid NCP list ABI")
	}
	seen := map[string]bool{}
	for i := 0; i < 7; i++ {
		word := raw[64+i*32 : 96+i*32]
		if new(big.Int).SetBytes(word[:12]).Sign() != 0 {
			return fmt.Errorf("invalid address padding")
		}
		addr := "0x" + hex.EncodeToString(word[12:])
		if !wanted[addr] || seen[addr] {
			return fmt.Errorf("actual NCP list differs")
		}
		seen[addr] = true
	}
	return nil
}

func whitespaceNCPCSV(a [7]string) string {
	return strings.Join([]string{a[0], " " + a[1], a[2] + " ", "  " + a[3] + "  ", "   " + a[4], a[5] + "   ", "    " + a[6] + "    "}, ",")
}

func ncpWhitespaceFixture() ncpWhitespaceProof {
	a := [7]string{}
	p := ncpWhitespaceProof{Fork: 20, Height: 22, Later: 24, UseNCP: true, BlockTag: "0x16", Count: "0x" + fmt.Sprintf("%064x", 7), ZeroMember: "0x" + fmt.Sprintf("%064x", 0), Members: map[string]string{}, List: "0x" + fmt.Sprintf("%064x%064x", 32, 7)}
	for i := range a {
		a[i] = fmt.Sprintf("0x%040x", i+1)
		p.Members[a[i]] = "0x" + fmt.Sprintf("%064x", 1)
		p.List += strings.Repeat("0", 24) + a[i][2:]
	}
	p.RawCSV = whitespaceNCPCSV(a)
	return p
}

func TestNCPWhitespaceRejectsInitOnlyAndPartialMembership(t *testing.T) {
	if err := verifyNCPWhitespace(ncpWhitespaceFixture()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*ncpWhitespaceProof)
	}{
		{"NCP disabled", func(p *ncpWhitespaceProof) { p.UseNCP = false }},
		{"no live block", func(p *ncpWhitespaceProof) { p.Height = 0 }},
		{"before activation", func(p *ncpWhitespaceProof) { p.Fork = 23 }},
		{"no production", func(p *ncpWhitespaceProof) { p.Later = p.Height }},
		{"query not pinned", func(p *ncpWhitespaceProof) { p.BlockTag = "latest" }},
		{"positive count but wrong", func(p *ncpWhitespaceProof) { p.Count = "0x" + fmt.Sprintf("%064x", 1) }},
		{"missing membership", func(p *ncpWhitespaceProof) { delete(p.Members, fmt.Sprintf("0x%040x", 2)) }},
		{"trimmed address not member", func(p *ncpWhitespaceProof) { p.Members[fmt.Sprintf("0x%040x", 2)] = "0x" + fmt.Sprintf("%064x", 0) }},
		{"invalid boolean", func(p *ncpWhitespaceProof) { p.Members[fmt.Sprintf("0x%040x", 2)] = "0x" + fmt.Sprintf("%064x", 2) }},
		{"zero address included", func(p *ncpWhitespaceProof) { p.ZeroMember = "0x" + fmt.Sprintf("%064x", 1) }},
		{"list differs", func(p *ncpWhitespaceProof) { p.List = p.List[:len(p.List)-40] + strings.Repeat("f", 40) }},
		{"no whitespace input", func(p *ncpWhitespaceProof) { p.RawCSV = strings.ReplaceAll(p.RawCSV, " ", "") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := ncpWhitespaceFixture()
			tc.change(&p)
			if verifyNCPWhitespace(p) == nil {
				t.Fatal("init success hid missing live NCP semantics")
			}
		})
	}
}

func TestWemixNCPWhitespaceE2E(t *testing.T) {
	from, to := os.Getenv("CHAINBENCH_E2E_FROM_BIN"), os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if from == "" || to == "" {
		t.Skip("set native handoff binaries")
	}
	a := [7]string{}
	for i := 0; i < 5; i++ {
		a[i] = strings.ToLower(presetNodeAddr(t, i+1))
	}
	a[5], a[6] = "0x000000000000000000000000000000000000aa06", "0x000000000000000000000000000000000000aa07"
	raw := whitespaceNCPCSV(a)
	overlay := map[string]any{"config": map[string]any{"croissant": map[string]any{"wBFT": map[string]any{"useNCP": true}, "govContracts": map[string]any{"govNCP": map[string]any{"params": map[string]any{"ncps": raw}}}}}}
	c := rpc.Dial(runGovHandoffOverlay(t, from, to, overlay))
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	p := ncpWhitespaceProof{RawCSV: raw, Fork: 20, UseNCP: true, Members: map[string]string{}}
	var err error
	p.Height, err = c.BlockNumber(ctx)
	if err != nil {
		t.Fatal(err)
	}
	p.BlockTag = fmt.Sprintf("0x%x", p.Height)
	for _, q := range []struct {
		sig string
		out *string
	}{{"ncpCount()", &p.Count}, {"ncpList()", &p.List}} {
		if err := c.Call(ctx, "eth_call", q.out, map[string]string{"to": e2eGovNCP, "data": accounts.Selector(q.sig)}, p.BlockTag); err != nil {
			t.Fatal(err)
		}
	}
	for _, addr := range a {
		var result string
		if err := c.Call(ctx, "eth_call", &result, map[string]string{"to": e2eGovNCP, "data": accounts.EncodeCallArgs("isNCP(address)", accounts.Address(addr))}, p.BlockTag); err != nil {
			t.Fatal(err)
		}
		p.Members[addr] = result
	}
	if err := c.Call(ctx, "eth_call", &p.ZeroMember, map[string]string{"to": e2eGovNCP, "data": accounts.EncodeCallArgs("isNCP(address)", accounts.Address("0x0000000000000000000000000000000000000000"))}, p.BlockTag); err != nil {
		t.Fatal(err)
	}
	waitForHead(t, c, p.Height+2, 90*time.Second)
	p.Later, err = c.BlockNumber(ctx)
	if err != nil {
		t.Fatal(err)
	}
	logGovernanceProof(t, "NODE-006-live-NCP-whitespace", p)
	if err := verifyNCPWhitespace(p); err != nil {
		t.Fatal(err)
	}
}

func TestNCPWhitespaceMemberListOrderIsNotConstrained(t *testing.T) {
	p := ncpWhitespaceFixture()
	prefix := p.List[:130]
	words := p.List[130:]
	reversed := ""
	for i := 6; i >= 0; i-- {
		reversed += words[i*64 : (i+1)*64]
	}
	p.List = prefix + reversed
	if err := verifyNCPWhitespace(p); err != nil {
		t.Fatal("membership order is not a requirement", err)
	}
}
