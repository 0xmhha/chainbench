//go:build e2e

package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
)

type governanceForkProof struct {
	Fork, BeforeHeight, AtHeight uint64
	CodeBefore, CodeAt           map[string]string
	Configured, Storage, Getters [4]string
}

func verifyGovernanceFork(p governanceForkProof) error {
	if p.Fork != 100 || p.BeforeHeight != 99 || p.AtHeight != 100 || len(p.CodeBefore) != 4 || len(p.CodeAt) != 4 {
		return fmt.Errorf("original fork coordinates missing")
	}
	for _, a := range []string{e2eGovConfig, e2eGovStaking, e2eGovRewardee, e2eGovNCP} {
		if p.CodeBefore[a] != "0x" || !strings.HasPrefix(p.CodeAt[a], "0x") {
			return fmt.Errorf("incorrect contract deployment transition")
		}
		code, err := hex.DecodeString(p.CodeAt[a][2:])
		if err != nil || len(code) == 0 {
			return fmt.Errorf("missing contract bytecode")
		}
	}
	expected := governanceForkFixture().Configured
	for i, v := range p.Configured {
		if v != expected[i] {
			return fmt.Errorf("original configuration differs")
		}
		n, ok := new(big.Int).SetString(v, 10)
		if !ok || n.Sign() <= 0 {
			return fmt.Errorf("invalid configured parameter")
		}
		for _, raw := range []string{p.Storage[i], p.Getters[i]} {
			if !strings.HasPrefix(raw, "0x") || len(raw) != 66 {
				return fmt.Errorf("wrong slot/getter ABI")
			}
			actual, ok := new(big.Int).SetString(raw[2:], 16)
			if !ok || actual.Cmp(n) != 0 {
				return fmt.Errorf("slot/getter differs from configured value")
			}
		}
	}
	return nil
}

func governanceForkFixture() governanceForkProof {
	p := governanceForkProof{Fork: 100, BeforeHeight: 99, AtHeight: 100, CodeBefore: map[string]string{}, CodeAt: map[string]string{}, Configured: [4]string{"10000000000000000000000000", "100000000000000000000000000", "604800", "604800"}}
	for _, a := range []string{e2eGovConfig, e2eGovStaking, e2eGovRewardee, e2eGovNCP} {
		p.CodeBefore[a] = "0x"
		p.CodeAt[a] = "0x6000"
	}
	for i, v := range p.Configured {
		n, _ := new(big.Int).SetString(v, 10)
		p.Storage[i] = "0x" + fmt.Sprintf("%064x", n)
		p.Getters[i] = p.Storage[i]
	}
	return p
}

func TestGovernanceForkRejectsPresenceAndPositiveOnly(t *testing.T) {
	if err := verifyGovernanceFork(governanceForkFixture()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*governanceForkProof)
	}{
		{"shortened fork", func(p *governanceForkProof) { p.Fork = 20 }},
		{"wrong before height", func(p *governanceForkProof) { p.BeforeHeight = 98 }},
		{"wrong deployment height", func(p *governanceForkProof) { p.AtHeight = 101 }},
		{"already deployed", func(p *governanceForkProof) { p.CodeBefore[e2eGovConfig] = "0x6000" }},
		{"missing pre state", func(p *governanceForkProof) { delete(p.CodeBefore, e2eGovStaking) }},
		{"missing config", func(p *governanceForkProof) { delete(p.CodeAt, e2eGovConfig) }},
		{"missing staking", func(p *governanceForkProof) { delete(p.CodeAt, e2eGovStaking) }},
		{"missing rewardee implementation", func(p *governanceForkProof) { delete(p.CodeAt, e2eGovRewardee) }},
		{"missing ncp", func(p *governanceForkProof) { delete(p.CodeAt, e2eGovNCP) }},
		{"non-code bytes", func(p *governanceForkProof) { p.CodeAt[e2eGovConfig] = "0xzz" }},
		{"minimum positive but wrong", func(p *governanceForkProof) { p.Storage[0] = "0x01" }},
		{"maximum positive but wrong", func(p *governanceForkProof) { p.Storage[1] = "0x01" }},
		{"staker period positive but wrong", func(p *governanceForkProof) { p.Storage[2] = "0x0f" }},
		{"delegator period positive but wrong", func(p *governanceForkProof) { p.Storage[3] = "0x05" }},
		{"getter differs", func(p *governanceForkProof) { p.Getters[0] = "0x01" }},
		{"configuration missing", func(p *governanceForkProof) { p.Configured[0] = "" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := governanceForkFixture()
			tc.change(&p)
			if verifyGovernanceFork(p) == nil {
				t.Fatal("presence and positive values hid wrong fork configuration")
			}
		})
	}
}

func writeGovernanceOriginalForkCase(t *testing.T) string {
	t.Helper()
	values := governanceForkFixture().Configured
	params := map[string]any{}
	for i, k := range []string{"minimumStaking", "maximumStaking", "unbondingPeriodStaker", "unbondingPeriodDelegator"} {
		params[k] = values[i]
	}
	path := writeHandoffCase(t, map[string]any{"config": map[string]any{"croissant": map[string]any{"govContracts": map[string]any{"govConfig": map[string]any{"params": params}}}}})
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var d map[string]any
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	preset := d["chainPreset"].(map[string]any)
	preset["upgrade"].(map[string]any)["at"] = 100
	preset["genesis"].(map[string]any)["overlay"].(map[string]any)["config"].(map[string]any)["croissantBlock"] = 100
	d["steps"] = []map[string]any{{"do": "waitBlock", "target": 102, "timeout": "300s"}, {"expect": "blockNumber", "compare": "Greater", "is": "100"}}
	b, err = json.MarshalIndent(d, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestGovernanceOriginalForkCaseCoordinates(t *testing.T) {
	b, err := os.ReadFile(writeGovernanceOriginalForkCase(t))
	if err != nil {
		t.Fatal(err)
	}
	var d map[string]any
	if err := json.Unmarshal(b, &d); err != nil {
		t.Fatal(err)
	}
	preset := d["chainPreset"].(map[string]any)
	if preset["upgrade"].(map[string]any)["at"] != float64(100) {
		t.Fatal("original GOV001 requires fork100")
	}
	cfg := preset["genesis"].(map[string]any)["overlay"].(map[string]any)["config"].(map[string]any)
	if cfg["croissantBlock"] != float64(100) {
		t.Fatal("genesis fork differs")
	}
	params := cfg["croissant"].(map[string]any)["govContracts"].(map[string]any)["govConfig"].(map[string]any)["params"].(map[string]any)
	for i, k := range []string{"minimumStaking", "maximumStaking", "unbondingPeriodStaker", "unbondingPeriodDelegator"} {
		if params[k] != governanceForkFixture().Configured[i] {
			t.Fatal("original configured parameter missing", k)
		}
	}
	steps := d["steps"].([]any)
	if steps[0].(map[string]any)["target"] != float64(102) || steps[1].(map[string]any)["is"] != "100" {
		t.Fatal("handoff wait/assertion did not follow fork100")
	}
}
