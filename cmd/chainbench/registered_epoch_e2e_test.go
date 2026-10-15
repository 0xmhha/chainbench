//go:build e2e

package main

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/accounts"
)

type registeredEpochProof struct {
	Fork, EpochLength, HeldHeight uint64
	Initial                       []epochBoundaryMember
	Before, After                 []string
	Registrations                 []registrationWitness
	BeforeStaked                  []string
	AfterStaked                   []string
	Blocks                        [3]map[string]any
	Extras                        [3]map[string]any
}

func verifyRegisteredEpoch(p registeredEpochProof) error {
	if p.Fork != 100 || p.EpochLength != 10 || p.HeldHeight != 102 || len(p.Initial) != 5 || len(p.Before) != 5 || len(p.After) != 4 || len(p.Registrations) != 4 || len(p.BeforeStaked) != 4 || len(p.AfterStaked) != 4 {
		return fmt.Errorf("not original coordinates and five-to-four fixture")
	}
	initial := map[string]string{}
	for _, m := range p.Initial {
		a := strings.ToLower(m.Address)
		if !epochBoundaryBytes(a, 20) || !epochBoundaryBytes(m.BLS, 48) || initial[a] != "" {
			return fmt.Errorf("invalid initial identity/key")
		}
		initial[a] = strings.ToLower(m.BLS)
	}
	checkSet := func(got []string, want map[string]string) bool {
		if len(got) != len(want) {
			return false
		}
		seen := map[string]bool{}
		for _, a := range got {
			a = strings.ToLower(a)
			if want[a] == "" || seen[a] {
				return false
			}
			seen[a] = true
		}
		return true
	}
	if !checkSet(p.Before, initial) {
		return fmt.Errorf("initial RPC validators differ from live keys")
	}
	registered := map[string]string{}
	for i, w := range p.Registrations {
		if err := verifyRegistrationWitness(w); err != nil {
			return err
		}
		a := strings.ToLower(w.Reward.Staker)
		h, err := strconv.ParseUint(strings.TrimPrefix(fmt.Sprint(w.Receipt["blockNumber"]), "0x"), 16, 64)
		if err != nil || h < 103 || h >= 110 || (i == 0 && h != 103) || registered[a] != "" || initial[a] != strings.ToLower(w.Reward.BLSKey) || w.Reward.Minimum != governanceForkFixture().Configured[0] || p.BeforeStaked[i] != "0x"+strings.Repeat("0", 64) {
			return fmt.Errorf("registration timing, prior stake, exact amount or BLS differs")
		}
		registered[a] = strings.ToLower(w.Reward.BLSKey)
		amount, _ := new(big.Int).SetString(w.Reward.Minimum, 10)
		if p.AfterStaked[i] != "0x"+fmt.Sprintf("%064x", amount) {
			return fmt.Errorf("actual pinned total stake differs")
		}
	}
	if !checkSet(p.After, registered) {
		return fmt.Errorf("next set differs from registered identities")
	}
	for i, b := range p.Blocks {
		if b["number"] != hexUint(uint64(109+i)) || !epochBoundaryBytes(b["hash"], 32) || (i > 0 && b["parentHash"] != p.Blocks[i-1]["hash"]) {
			return fmt.Errorf("wrong pinned epoch headers/linkage")
		}
		raw, ok := b["extraData"].(string)
		if !ok || len(raw) <= 66 || !strings.HasPrefix(raw, "0x") {
			return fmt.Errorf("missing raw header")
		}
		ep, present := p.Extras[i]["epochInfo"]
		if !present || (i != 1 && ep != nil) {
			return fmt.Errorf("ordinary epoch must be explicit null")
		}
	}
	ep, ok := p.Extras[1]["epochInfo"].(map[string]any)
	if !ok || ep["stabilizing"] != false {
		return fmt.Errorf("new registered set not stabilized")
	}
	stakers, ok := ep["stakers"].([]any)
	if !ok || len(stakers) != 4 {
		return fmt.Errorf("incorrect next stakers")
	}
	addresses := []string{}
	for _, v := range stakers {
		row, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid staker row")
		}
		addresses = append(addresses, strings.ToLower(fmt.Sprint(row["addr"])))
	}
	if !checkSet(addresses, registered) {
		return fmt.Errorf("epoch stakers differ from registrations")
	}
	validators, ok := ep["validators"].([]any)
	if !ok || len(validators) != 4 {
		return fmt.Errorf("incorrect epoch validators")
	}
	seen := map[string]bool{}
	for _, v := range validators {
		row, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid validator row")
		}
		a := strings.ToLower(fmt.Sprint(row["addr"]))
		i, err := strconv.ParseUint(strings.TrimPrefix(fmt.Sprint(row["index"]), "0x"), 16, 64)
		if err != nil || i >= uint64(len(addresses)) || a != addresses[i] || seen[a] || registered[a] == "" || registered[a] != strings.ToLower(fmt.Sprint(row["bls"])) {
			return fmt.Errorf("wrong indexed validator identity/BLS")
		}
		seen[a] = true
	}
	return nil
}

func registeredEpochFixture() registeredEpochProof {
	p := registeredEpochProof{Fork: 100, EpochLength: 10, HeldHeight: 102}
	for i := 0; i < 5; i++ {
		m := epochBoundaryMember{Address: fmt.Sprintf("0x%040x", i+1), BLS: "0x" + fmt.Sprintf("%096x", i+1)}
		p.Initial = append(p.Initial, m)
		p.Before = append(p.Before, m.Address)
		if i < 4 {
			w := registeredEpochRegistrationFixture(i, m)
			p.Registrations = append(p.Registrations, w)
			p.BeforeStaked = append(p.BeforeStaked, "0x"+strings.Repeat("0", 64))
			amount, _ := new(big.Int).SetString(w.Reward.Minimum, 10)
			p.AfterStaked = append(p.AfterStaked, "0x"+fmt.Sprintf("%064x", amount))
			p.After = append(p.After, m.Address)
		}
	}
	base := originalEpochBoundaryFixture()
	p.Blocks, p.Extras = base.Blocks, base.Extras
	p.Extras[1]["epochInfo"].(map[string]any)["stabilizing"] = false
	return p
}

func TestRegisteredEpochRejectsPresenceOnly(t *testing.T) {
	if err := verifyRegisteredEpoch(registeredEpochFixture()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*registeredEpochProof)
	}{
		{"shortened fork", func(p *registeredEpochProof) { p.Fork = 20 }},
		{"wrong epoch length", func(p *registeredEpochProof) { p.EpochLength = 20 }},
		{"missed registration boundary", func(p *registeredEpochProof) { p.HeldHeight = 103 }},
		{"wrong pinned epoch", func(p *registeredEpochProof) { p.Blocks[1]["number"] = "0x78" }},
		{"wrong parent", func(p *registeredEpochProof) { p.Blocks[2]["parentHash"] = "other" }},
		{"ordinary block has epoch", func(p *registeredEpochProof) { p.Extras[2]["epochInfo"] = map[string]any{} }},
		{"missing null field", func(p *registeredEpochProof) { delete(p.Extras[0], "epochInfo") }},
		{"still stabilizing", func(p *registeredEpochProof) { p.Extras[1]["epochInfo"].(map[string]any)["stabilizing"] = true }},
		{"unchanged validator set", func(p *registeredEpochProof) { p.After = append([]string(nil), p.Before...) }},
		{"duplicate initial member", func(p *registeredEpochProof) { p.Initial[4] = p.Initial[0] }},
		{"wrong previous set", func(p *registeredEpochProof) { p.Before[4] = p.After[0] }},
		{"wrong new member", func(p *registeredEpochProof) { p.After[3] = p.Initial[4].Address }},
		{"already staked", func(p *registeredEpochProof) { p.BeforeStaked[0] = "0x01" }},
		{"total stake absent", func(p *registeredEpochProof) { p.AfterStaked[0] = "0x" + strings.Repeat("0", 64) }},
		{"first registration at104", func(p *registeredEpochProof) {
			p.Registrations[0].Receipt["blockNumber"] = "0x68"
			p.Registrations[0].Transaction["blockNumber"] = "0x68"
		}},
		{"registration after epoch", func(p *registeredEpochProof) {
			p.Registrations[3].Receipt["blockNumber"] = "0x6f"
			p.Registrations[3].Transaction["blockNumber"] = "0x6f"
		}},
		{"wrong registered stake", func(p *registeredEpochProof) { p.Registrations[0].Transaction["value"] = "0x1" }},
		{"wrong registration event", func(p *registeredEpochProof) { p.Registrations[0].Reward.Logs = nil }},
		{"wrong epoch staker", func(p *registeredEpochProof) {
			p.Extras[1]["epochInfo"].(map[string]any)["stakers"].([]any)[0].(map[string]any)["addr"] = p.Initial[4].Address
		}},
		{"wrong epoch BLS", func(p *registeredEpochProof) {
			p.Extras[1]["epochInfo"].(map[string]any)["validators"].([]any)[0].(map[string]any)["bls"] = "0x" + strings.Repeat("ff", 48)
		}},
		{"wrong epoch index", func(p *registeredEpochProof) {
			p.Extras[1]["epochInfo"].(map[string]any)["validators"].([]any)[0].(map[string]any)["index"] = "0x4"
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := registeredEpochFixture()
			tc.change(&p)
			if verifyRegisteredEpoch(p) == nil {
				t.Fatal("epoch presence hid wrong registration or set transition")
			}
		})
	}
}

func registeredEpochRegistrationFixture(i int, m epochBoundaryMember) registrationWitness {
	w := registrationFixture()
	w.Reward.Staker, w.Mapped = m.Address, m.Address
	w.Reward.BLSKey = m.BLS
	w.Reward.Minimum = governanceForkFixture().Configured[0]
	amount, _ := new(big.Int).SetString(w.Reward.Minimum, 10)
	pk, _ := hex.DecodeString(m.BLS[2:])
	pop, _ := hex.DecodeString(w.PoP[2:])
	w.Hash = fmt.Sprintf("0x%064x", i+1)
	w.Transaction["hash"], w.Receipt["transactionHash"] = w.Hash, w.Hash
	w.Transaction["blockNumber"], w.Receipt["blockNumber"] = hexUint(uint64(103+i)), hexUint(uint64(103+i))
	w.Transaction["value"] = "0x" + amount.Text(16)
	w.Transaction["input"] = accounts.EncodeCallArgs("registerStaker(uint256,address,address,uint256,bytes,bytes)", accounts.Uint(amount), accounts.Address(m.Address), accounts.Address(m.Address), accounts.Uint(big.NewInt(0)), accounts.Bytes(pk), accounts.Bytes(pop))
	w.Reward.Logs[0].Topics[1] = "0x" + strings.Repeat("0", 24) + m.Address[2:]
	w.Reward.Logs[0].Data = "0x" + strings.Repeat("0", 24) + w.Reward.Operator[2:] + strings.Repeat("0", 24) + w.Reward.Rewardee[2:] + strings.Repeat("0", 24) + m.Address[2:] + fmt.Sprintf("%064x%064x%064x%064x", 0, amount, 192, 48) + hex.EncodeToString(pk) + strings.Repeat("0", 32)
	return w
}
