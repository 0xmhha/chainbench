//go:build e2e

package main

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"testing"
)

type stakingReadWitness struct {
	Registration                                        registrationWitness
	ConfiguredMinimum, Minimum, BeforeTotal, AfterTotal string
	ParentTag, MinedTag                                 string
}

func verifyStakingRead(w stakingReadWitness) error {
	if err := verifyRegistrationWitness(w.Registration); err != nil {
		return err
	}
	parent, e1 := strconv.ParseUint(strings.TrimPrefix(w.ParentTag, "0x"), 16, 64)
	mined, e2 := strconv.ParseUint(strings.TrimPrefix(w.MinedTag, "0x"), 16, 64)
	if !strings.HasPrefix(w.ParentTag, "0x") || !strings.HasPrefix(w.MinedTag, "0x") || e1 != nil || e2 != nil || mined != parent+1 || w.Registration.Transaction["blockNumber"] != w.MinedTag || w.Registration.Receipt["blockNumber"] != w.MinedTag {
		return fmt.Errorf("read not fixed to registration parent/block")
	}
	configured, ok := new(big.Int).SetString(w.ConfiguredMinimum, 10)
	if !ok || configured.Sign() <= 0 || w.Registration.Reward.Minimum != w.ConfiguredMinimum {
		return fmt.Errorf("registered amount differs from configured minimum")
	}
	values := []*big.Int{}
	for _, raw := range []string{w.Minimum, w.BeforeTotal, w.AfterTotal} {
		if !strings.HasPrefix(raw, "0x") || len(raw) != 66 {
			return fmt.Errorf("invalid uint256 read ABI")
		}
		n, ok := new(big.Int).SetString(raw[2:], 16)
		if !ok {
			return fmt.Errorf("invalid read value")
		}
		values = append(values, n)
	}
	if values[0].Cmp(configured) != 0 || new(big.Int).Sub(values[2], values[1]).Cmp(configured) != 0 {
		return fmt.Errorf("minimum or total staking delta differs")
	}
	return nil
}

func stakingReadFixture() stakingReadWitness {
	r := registrationFixture()
	return stakingReadWitness{Registration: r, ConfiguredMinimum: r.Reward.Minimum, Minimum: "0x" + fmt.Sprintf("%064x", 100), BeforeTotal: "0x" + fmt.Sprintf("%064x", 0), AfterTotal: "0x" + fmt.Sprintf("%064x", 100), ParentTag: "0x4", MinedTag: "0x5"}
}

func TestStakingReadRejectsPositiveOnlyAndUnpinnedState(t *testing.T) {
	if err := verifyStakingRead(stakingReadFixture()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*stakingReadWitness)
	}{
		{"minimum differs from config", func(p *stakingReadWitness) { p.ConfiguredMinimum = "101" }},
		{"minimum positive but wrong", func(p *stakingReadWitness) { p.Minimum = "0x" + fmt.Sprintf("%064x", 1) }},
		{"minimum invalid ABI", func(p *stakingReadWitness) { p.Minimum = "0x64" }},
		{"total positive but wrong", func(p *stakingReadWitness) { p.AfterTotal = "0x" + fmt.Sprintf("%064x", 99) }},
		{"total before missing", func(p *stakingReadWitness) { p.BeforeTotal = "" }},
		{"total latest query", func(p *stakingReadWitness) { p.MinedTag = "latest" }},
		{"parent not adjacent", func(p *stakingReadWitness) { p.ParentTag = "0x3" }},
		{"query block differs from receipt", func(p *stakingReadWitness) { p.Registration.Receipt["blockNumber"] = "0x6" }},
		{"receipt not successful", func(p *stakingReadWitness) { p.Registration.Receipt["status"] = "0x0" }},
		{"registration input differs", func(p *stakingReadWitness) { p.Registration.Transaction["input"] = "0x" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := stakingReadFixture()
			tc.change(&p)
			if verifyStakingRead(p) == nil {
				t.Fatal("positive-only read hid incorrect registration delta")
			}
		})
	}
}
