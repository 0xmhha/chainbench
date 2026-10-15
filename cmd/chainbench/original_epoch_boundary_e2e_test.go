//go:build e2e

package main

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/rpc"
)

type epochBoundaryMember struct{ Address, BLS string }
type originalEpochBoundaryProof struct {
	Fork, EpochLength, Height uint64
	Blocks                    [3]map[string]any
	Extras                    [3]map[string]any
	Expected                  []epochBoundaryMember
}

func verifyOriginalEpochBoundary(p originalEpochBoundaryProof) error {
	if p.Fork != 100 || p.EpochLength != 10 || p.Height != 110 || len(p.Expected) != 4 {
		return fmt.Errorf("not original fork100/epoch10/block110 fixture")
	}
	expected := map[string]string{}
	for _, m := range p.Expected {
		addr := strings.ToLower(m.Address)
		if !epochBoundaryBytes(addr, 20) || !epochBoundaryBytes(m.BLS, 48) || expected[addr] != "" {
			return fmt.Errorf("invalid declared member identity/key")
		}
		expected[addr] = strings.ToLower(m.BLS)
	}
	for i, h := range p.Blocks {
		if h["number"] != fmt.Sprintf("0x%x", 109+i) || !epochBoundaryBytes(h["hash"], 32) {
			return fmt.Errorf("wrong actual pinned header")
		}
		extra, ok := h["extraData"].(string)
		if !ok || !strings.HasPrefix(extra, "0x") {
			return fmt.Errorf("missing raw WBFT header")
		}
		raw, err := hex.DecodeString(extra[2:])
		if err != nil || len(raw) <= 32 {
			return fmt.Errorf("invalid raw WBFT header")
		}
		if i > 0 && (h["parentHash"] != p.Blocks[i-1]["hash"] || h["hash"] == p.Blocks[i-1]["hash"]) {
			return fmt.Errorf("wrong actual parent linkage")
		}
		epoch, present := p.Extras[i]["epochInfo"]
		if !present || (i != 1 && epoch != nil) {
			return fmt.Errorf("ordinary epoch not explicit null")
		}
	}
	ep, ok := p.Extras[1]["epochInfo"].(map[string]any)
	if !ok {
		return fmt.Errorf("boundary epoch absent")
	}
	stabilizing, ok := ep["stabilizing"].(bool)
	if !ok || !stabilizing {
		return fmt.Errorf("below-threshold stabilization flag differs")
	}
	stakers, ok := ep["stakers"].([]any)
	if !ok || len(stakers) != 4 {
		return fmt.Errorf("missing exact seeded stakers")
	}
	addresses := []string{}
	seen := map[string]bool{}
	for _, raw := range stakers {
		row, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid staker entry")
		}
		addr, ok := row["addr"].(string)
		addr = strings.ToLower(addr)
		if !ok || expected[addr] == "" || seen[addr] {
			return fmt.Errorf("wrong or duplicate staker")
		}
		diligence, ok := row["diligence"].(string)
		if !ok || !strings.HasPrefix(diligence, "0x") {
			return fmt.Errorf("missing actual diligence")
		}
		if n, ok := new(big.Int).SetString(diligence[2:], 16); !ok || n.Sign() < 0 {
			return fmt.Errorf("invalid diligence")
		}
		addresses = append(addresses, addr)
		seen[addr] = true
	}
	validators, ok := ep["validators"].([]any)
	if !ok || len(validators) != 4 {
		return fmt.Errorf("missing exact validator keys")
	}
	seen = map[string]bool{}
	for _, raw := range validators {
		row, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid validator entry")
		}
		index, ok := row["index"].(string)
		if !ok || !strings.HasPrefix(index, "0x") {
			return fmt.Errorf("missing staker index")
		}
		n, err := strconv.ParseUint(index[2:], 16, 64)
		if err != nil || n >= uint64(len(addresses)) {
			return fmt.Errorf("invalid staker index")
		}
		addr, ok := row["addr"].(string)
		addr = strings.ToLower(addr)
		if !ok || addr != addresses[n] || seen[addr] {
			return fmt.Errorf("validator not bound to indexed staker")
		}
		key, ok := row["bls"].(string)
		if !ok || !epochBoundaryBytes(key, 48) || strings.ToLower(key) != expected[addr] {
			return fmt.Errorf("validator BLS differs from declared key")
		}
		seen[addr] = true
	}
	return nil
}

func originalEpochBoundaryFixture() originalEpochBoundaryProof {
	p := originalEpochBoundaryProof{Fork: 100, EpochLength: 10, Height: 110}
	stakers, validators := []any{}, []any{}
	for i := 0; i < 4; i++ {
		m := epochBoundaryMember{Address: fmt.Sprintf("0x%040x", i+1), BLS: "0x" + fmt.Sprintf("%096x", i+1)}
		p.Expected = append(p.Expected, m)
		stakers = append(stakers, map[string]any{"addr": m.Address, "diligence": "0x1"})
		validators = append(validators, map[string]any{"addr": m.Address, "bls": m.BLS, "index": fmt.Sprintf("0x%x", i)})
	}
	for i := 0; i < 3; i++ {
		p.Blocks[i] = map[string]any{"number": fmt.Sprintf("0x%x", 109+i), "hash": "0x" + strings.Repeat(fmt.Sprintf("%02x", i+1), 32), "extraData": "0x" + strings.Repeat("ab", 40)}
		p.Extras[i] = map[string]any{"epochInfo": nil}
		if i > 0 {
			p.Blocks[i]["parentHash"] = p.Blocks[i-1]["hash"]
		}
	}
	p.Extras[1]["epochInfo"] = map[string]any{"stakers": stakers, "validators": validators, "stabilizing": true}
	return p
}

func TestOriginalEpochBoundaryRejectsPresenceOnly(t *testing.T) {
	if err := verifyOriginalEpochBoundary(originalEpochBoundaryFixture()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*originalEpochBoundaryProof)
	}{
		{"wrong fork", func(p *originalEpochBoundaryProof) { p.Fork = 20 }},
		{"wrong epoch", func(p *originalEpochBoundaryProof) { p.EpochLength = 20 }},
		{"wrong boundary", func(p *originalEpochBoundaryProof) { p.Height = 120 }},
		{"wrong pinned header", func(p *originalEpochBoundaryProof) { p.Blocks[1]["number"] = "0x78" }},
		{"wrong parent", func(p *originalEpochBoundaryProof) { p.Blocks[2]["parentHash"] = p.Blocks[0]["hash"] }},
		{"missing header bytes", func(p *originalEpochBoundaryProof) { p.Blocks[1]["extraData"] = "0x" }},
		{"before carries epoch", func(p *originalEpochBoundaryProof) { p.Extras[0]["epochInfo"] = map[string]any{} }},
		{"after omits epoch field", func(p *originalEpochBoundaryProof) { delete(p.Extras[2], "epochInfo") }},
		{"empty stakers", func(p *originalEpochBoundaryProof) { p.Extras[1]["epochInfo"].(map[string]any)["stakers"] = []any{} }},
		{"wrong staker", func(p *originalEpochBoundaryProof) {
			p.Extras[1]["epochInfo"].(map[string]any)["stakers"].([]any)[0].(map[string]any)["addr"] = "0x" + strings.Repeat("f", 40)
		}},
		{"duplicate staker", func(p *originalEpochBoundaryProof) {
			s := p.Extras[1]["epochInfo"].(map[string]any)["stakers"].([]any)
			s[1] = s[0]
		}},
		{"empty validators", func(p *originalEpochBoundaryProof) { p.Extras[1]["epochInfo"].(map[string]any)["validators"] = []any{} }},
		{"wrong BLS", func(p *originalEpochBoundaryProof) {
			p.Extras[1]["epochInfo"].(map[string]any)["validators"].([]any)[0].(map[string]any)["bls"] = "0x" + strings.Repeat("ff", 48)
		}},
		{"invalid staker index", func(p *originalEpochBoundaryProof) {
			p.Extras[1]["epochInfo"].(map[string]any)["validators"].([]any)[0].(map[string]any)["index"] = "0x4"
		}},
		{"missing stabilizing flag", func(p *originalEpochBoundaryProof) { delete(p.Extras[1]["epochInfo"].(map[string]any), "stabilizing") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := originalEpochBoundaryFixture()
			tc.change(&p)
			if verifyOriginalEpochBoundary(p) == nil {
				t.Fatal("epoch presence hid wrong origin or members")
			}
		})
	}
}

// epochBoundaryBytes checks only the fixed-width public encodings. Actual RLP
// epoch contents are also independently decoded from retained native headers.
func epochBoundaryBytes(v any, n int) bool {
	s, ok := v.(string)
	if !ok || !strings.HasPrefix(s, "0x") {
		return false
	}
	b, err := hex.DecodeString(s[2:])
	return err == nil && len(b) == n
}

func captureOriginalEpochBoundary(t *testing.T, c *rpc.Client) {
	t.Helper()
	waitForHead(t, c, 111, 90*time.Second)
	p := originalEpochBoundaryProof{Fork: 100, EpochLength: 10, Height: 110}
	for i := 0; i < 3; i++ {
		tag := hexUint(uint64(109 + i))
		if err := c.Call(t.Context(), "eth_getBlockByNumber", &p.Blocks[i], tag, false); err != nil {
			t.Fatal(err)
		}
		if err := c.Call(t.Context(), "istanbul_getWbftExtraInfo", &p.Extras[i], tag); err != nil {
			t.Fatal(err)
		}
	}
	for i := 1; i <= 4; i++ {
		key, _ := presetNodeBLS(t, i)
		p.Expected = append(p.Expected, epochBoundaryMember{Address: presetNodeAddr(t, i), BLS: "0x" + hex.EncodeToString(key)})
	}
	logGovernanceProof(t, "RPC-022-original-epoch", p)
	if err := verifyOriginalEpochBoundary(p); err != nil {
		t.Fatal(err)
	}
}
