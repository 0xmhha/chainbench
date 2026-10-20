package testengine_test

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"testing"
)

type previousSeal struct {
	Sealers   []string `json:"sealers"`
	Signature string   `json:"signature"`
}
type previousSealerWitness struct {
	Parent, Child       map[string]any
	ValidatorQueryTag   string
	ParentValidators    []string
	Prepared, Committed *previousSeal
	Extra               map[string]any
}

func verifyPreviousSealers(p previousSealerWitness) error {
	parent, err := previousSealerNumber(p.Parent["number"])
	if err != nil {
		return err
	}
	child, err := previousSealerNumber(p.Child["number"])
	if err != nil || parent == 0 || child != parent+1 || p.ValidatorQueryTag != p.Parent["number"] || !previousSealerHex(p.Parent["hash"], 32) || !previousSealerHex(p.Child["hash"], 32) || p.Child["parentHash"] != p.Parent["hash"] || p.Child["hash"] == p.Parent["hash"] {
		return fmt.Errorf("not the exact previous block/query origin")
	}
	n := len(p.ParentValidators)
	if n < 5 {
		return fmt.Errorf("need at least five actual previous validators")
	}
	allowed := map[string]bool{}
	for _, addr := range p.ParentValidators {
		key := strings.ToLower(addr)
		if !previousSealerHex(key, 20) || allowed[key] {
			return fmt.Errorf("invalid previous validator set")
		}
		allowed[key] = true
	}
	quorum := n - (n-1)/3
	for _, seal := range []*previousSeal{p.Prepared, p.Committed} {
		if seal == nil || len(seal.Sealers) < quorum || len(seal.Sealers) > n || !previousSealerHex(seal.Signature, 96) {
			return fmt.Errorf("missing actual aggregate/quorum")
		}
		seen := map[string]bool{}
		for _, addr := range seal.Sealers {
			key := strings.ToLower(addr)
			if !allowed[key] || seen[key] {
				return fmt.Errorf("duplicate or non-parent sealer")
			}
			seen[key] = true
		}
	}
	return nil
}

func previousSealerFixture() previousSealerWitness {
	p := previousSealerWitness{Parent: map[string]any{"number": "0xd", "hash": "0x" + strings.Repeat("aa", 32)}, Child: map[string]any{"number": "0xe", "hash": "0x" + strings.Repeat("bb", 32), "parentHash": "0x" + strings.Repeat("aa", 32)}, ValidatorQueryTag: "0xd", Prepared: &previousSeal{Signature: "0x" + strings.Repeat("ab", 96)}, Committed: &previousSeal{Signature: "0x" + strings.Repeat("cd", 96)}}
	for i := 1; i <= 5; i++ {
		addr := fmt.Sprintf("0x%040x", i)
		p.ParentValidators = append(p.ParentValidators, addr)
		if i <= 4 {
			p.Prepared.Sealers = append(p.Prepared.Sealers, addr)
			p.Committed.Sealers = append(p.Committed.Sealers, addr)
		}
	}
	return p
}

func TestPreviousSealersRejectCountOnlyEvidence(t *testing.T) {
	if err := verifyPreviousSealers(previousSealerFixture()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*previousSealerWitness)
	}{
		{"current-block query", func(p *previousSealerWitness) { p.ValidatorQueryTag = p.Child["number"].(string) }},
		{"wrong parent number", func(p *previousSealerWitness) { p.Parent["number"] = "0xc" }},
		{"wrong parent hash", func(p *previousSealerWitness) { p.Child["parentHash"] = "other" }},
		{"same child hash", func(p *previousSealerWitness) { p.Child["hash"] = p.Parent["hash"] }},
		{"fewer than five validators", func(p *previousSealerWitness) { p.ParentValidators = p.ParentValidators[:4] }},
		{"duplicate validator", func(p *previousSealerWitness) { p.ParentValidators[4] = p.ParentValidators[0] }},
		{"prepared below actual quorum", func(p *previousSealerWitness) { p.Prepared.Sealers = p.Prepared.Sealers[:3] }},
		{"committed below actual quorum", func(p *previousSealerWitness) { p.Committed.Sealers = p.Committed.Sealers[:3] }},
		{"prepared outsider", func(p *previousSealerWitness) { p.Prepared.Sealers[0] = "0x" + strings.Repeat("ff", 20) }},
		{"committed outsider", func(p *previousSealerWitness) { p.Committed.Sealers[0] = "0x" + strings.Repeat("ff", 20) }},
		{"duplicate prepared signer", func(p *previousSealerWitness) { p.Prepared.Sealers[1] = p.Prepared.Sealers[0] }},
		{"duplicate committed signer", func(p *previousSealerWitness) { p.Committed.Sealers[1] = p.Committed.Sealers[0] }},
		{"missing prepared aggregate", func(p *previousSealerWitness) { p.Prepared.Signature = "0x" }},
		{"short committed aggregate", func(p *previousSealerWitness) { p.Committed.Signature = "0xab" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := previousSealerFixture()
			tc.change(&p)
			if verifyPreviousSealers(p) == nil {
				t.Fatal("counts hid wrong previous-block signer context")
			}
		})
	}
}

func previousSealerHex(v any, size int) bool {
	s, ok := v.(string)
	if !ok || !strings.HasPrefix(s, "0x") {
		return false
	}
	b, err := hex.DecodeString(s[2:])
	return err == nil && len(b) == size
}
func previousSealerNumber(v any) (uint64, error) {
	s, ok := v.(string)
	if !ok || !strings.HasPrefix(s, "0x") {
		return 0, fmt.Errorf("invalid header number")
	}
	return strconv.ParseUint(s[2:], 16, 64)
}
