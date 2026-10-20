//go:build e2e

package main

import (
	"fmt"
	"math/big"
	"strings"
	"testing"
)

type ncpSelectionMember struct{ Staker, Operator, BLS string }
type ncpSelectionSnapshot struct {
	TriggerHeight, Boundary                            uint64
	StateTag, ValidatorTag                             string
	Blocks                                             [2]map[string]any
	Extras                                             [2]map[string]any
	Validators                                         []string
	Active, NCP, Staked                                map[string]string
	StakerCount, StakerList, CouncilCount, CouncilList string
}
type ncpSelectionProof struct {
	Fork, Target, Threshold uint64
	UseNCP                  bool
	Members                 []ncpSelectionMember
	Snapshots               []ncpSelectionSnapshot
	Addition                ncpMajorityProof
}

func verifyNCPSelection(p ncpSelectionProof) error {
	if p.Fork != 20 || p.Target != 7 || p.Threshold != 5 || !p.UseNCP || len(p.Members) != 8 || len(p.Snapshots) != 6 {
		return fmt.Errorf("wrong original target7/threshold5/NCP fixture")
	}
	stakers, operators := map[string]bool{}, map[string]bool{}
	for _, m := range p.Members {
		a, o := strings.ToLower(m.Staker), strings.ToLower(m.Operator)
		if !epochBoundaryBytes(a, 20) || !epochBoundaryBytes(o, 20) || a == o || stakers[a] || operators[o] || !epochBoundaryBytes(m.BLS, 48) {
			return fmt.Errorf("invalid eight staker/operator/BLS identities")
		}
		stakers[a] = true
		operators[o] = true
	}
	if err := verifyNCPMajority(p.Addition); err != nil {
		return err
	}
	if !strings.EqualFold(p.Addition.Candidate, p.Members[7].Operator) || !strings.EqualFold(p.Addition.Proposer, p.Members[0].Operator) {
		return fmt.Errorf("permission granted to wrong operator")
	}
	for i, a := range p.Addition.Members {
		if !strings.EqualFold(a, p.Members[i].Operator) {
			return fmt.Errorf("ballot electorate not initial operators")
		}
	}
	lastVote, valid := lifecycleQuantity(p.Addition.Stages[4].Receipt["blockNumber"])
	if !valid || !lastVote.IsUint64() || p.Snapshots[5].TriggerHeight != lastVote.Uint64() {
		return fmt.Errorf("last selection not bound to actual permission transaction")
	}
	proposalHeight, valid := lifecycleQuantity(p.Addition.Stages[0].Block["number"])
	if !valid || !proposalHeight.IsUint64() || proposalHeight.Uint64() <= p.Snapshots[4].Boundary+1 {
		return fmt.Errorf("permission predates nonNCP ranking observation")
	}
	for i, s := range p.Snapshots {
		if s.TriggerHeight <= 20 || (i > 0 && s.TriggerHeight <= p.Snapshots[i-1].Boundary+1) || s.Boundary != (s.TriggerHeight/10+1)*10 || s.StateTag != hexUint(s.Boundary) || s.ValidatorTag != hexUint(s.Boundary+1) || len(s.Active) != 8 || len(s.NCP) != 8 || len(s.Staked) != 8 {
			return fmt.Errorf("not exact following epoch and pinned state")
		}
		registered := map[int]bool{}
		registeredList := []string{}
		council := []string{}
		eligible := 0
		for _, j := range ncpSelectionRegistered(i) {
			registered[j] = true
			registeredList = append(registeredList, p.Members[j].Staker)
		}
		for j, m := range p.Members {
			active, ncp := uint64(0), uint64(0)
			amount := big.NewInt(0)
			if registered[j] {
				active = 1
				amount = ncpSelectionAmount(j, i >= 4)
			}
			if j < 7 || i == 5 {
				ncp = 1
				council = append(council, m.Operator)
			}
			if active == 1 && ncp == 1 {
				eligible++
			}
			if s.Active[m.Staker] != ncpBallotWord(active) || s.NCP[m.Operator] != ncpBallotWord(ncp) || s.Staked[m.Staker] != "0x"+fmt.Sprintf("%064x", amount) {
				return fmt.Errorf("wrong original registration, NCP eligibility or exact ranked stake")
			}
		}
		if eligible != []int{1, 4, 4, 7, 7, 8}[i] || s.StakerCount != ncpBallotWord(uint64(len(registeredList))) || !strings.EqualFold(s.StakerList, ncpBallotList(registeredList)) || s.CouncilCount != ncpBallotWord(uint64(len(council))) || !strings.EqualFold(s.CouncilList, ncpBallotList(council)) {
			return fmt.Errorf("wrong independent staker and NCP counts/lists")
		}
		for j, b := range s.Blocks {
			if b["number"] != hexUint(s.Boundary+uint64(j)) || !epochBoundaryBytes(b["hash"], 32) || len(fmt.Sprint(b["extraData"])) <= 66 || (j == 1 && b["parentHash"] != s.Blocks[0]["hash"]) {
				return fmt.Errorf("wrong exact epoch header or next linkage")
			}
		}
		ep, ok := s.Extras[0]["epochInfo"].(map[string]any)
		if !ok || ep["stabilizing"] != (i < 3) {
			return fmt.Errorf("wrong flag")
		}
		ordinary, present := s.Extras[1]["epochInfo"]
		if !present || ordinary != nil {
			return fmt.Errorf("ordinary next epoch must be explicit null")
		}
		stakerIDs, validatorIDs := ncpSelectionSets(i)
		expectedStakers, expectedValidators := map[string]string{}, map[string]string{}
		for _, j := range stakerIDs {
			m := p.Members[j]
			expectedStakers[strings.ToLower(m.Staker)] = strings.ToLower(m.BLS)
		}
		for _, j := range validatorIDs {
			m := p.Members[j]
			expectedValidators[strings.ToLower(m.Staker)] = strings.ToLower(m.BLS)
		}
		if !ncpSelectionSet(s.Validators, expectedValidators) {
			return fmt.Errorf("wrong target validators before/after NCP ranking permission")
		}
		rows, ok := ep["stakers"].([]any)
		if !ok || len(rows) != len(expectedStakers) {
			return fmt.Errorf("wrong seeded/eligible epoch stakers")
		}
		addresses := []string{}
		for _, v := range rows {
			row, ok := v.(map[string]any)
			if !ok {
				return fmt.Errorf("invalid staker")
			}
			addresses = append(addresses, strings.ToLower(fmt.Sprint(row["addr"])))
		}
		if !ncpSelectionSet(addresses, expectedStakers) {
			return fmt.Errorf("epoch stakers not correct NCP-eligible candidates")
		}
		validators, ok := ep["validators"].([]any)
		if !ok || len(validators) != len(expectedValidators) {
			return fmt.Errorf("wrong epoch validator count")
		}
		seen := map[string]bool{}
		for _, v := range validators {
			row, ok := v.(map[string]any)
			if !ok {
				return fmt.Errorf("invalid validator")
			}
			a := strings.ToLower(fmt.Sprint(row["addr"]))
			index, valid := lifecycleQuantity(row["index"])
			if !valid || !index.IsUint64() || index.Uint64() >= uint64(len(addresses)) || addresses[index.Uint64()] != a || seen[a] || expectedValidators[a] == "" || expectedValidators[a] != strings.ToLower(fmt.Sprint(row["bls"])) {
				return fmt.Errorf("wrong ranked indexed validator/BLS")
			}
			seen[a] = true
		}
	}
	return nil
}

func ncpSelectionSet(got []string, want map[string]string) bool {
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

func ncpSelectionAmount(index int, topup bool) *big.Int {
	minimum, _ := new(big.Int).SetString(governanceForkFixture().Configured[0], 10)
	units := []int64{8, 7, 6, 5, 4, 3, 2, 2}
	if index == 7 && topup {
		units[7] = 9
	}
	return new(big.Int).Mul(new(big.Int).Div(minimum, big.NewInt(2)), big.NewInt(units[index]))
}
func ncpSelectionRegistered(stage int) []int {
	order := []int{0, 1, 2, 3, 7, 4, 5, 6}
	counts := []int{1, 4, 5, 8, 8, 8}
	return order[:counts[stage]]
}
func ncpSelectionSets(stage int) ([]int, []int) {
	all := []int{0, 1, 2, 3, 4, 5, 6, 7}
	seven := all[:7]
	if stage < 3 {
		return all, all
	}
	if stage < 5 {
		return seven, seven
	}
	return all, []int{7, 0, 1, 2, 3, 4, 5}
}
func ncpSelectionFixture() ncpSelectionProof {
	p := ncpSelectionProof{Fork: 20, Target: 7, Threshold: 5, UseNCP: true, Addition: ncpMajorityFixture()}
	for i := 0; i < 8; i++ {
		p.Members = append(p.Members, ncpSelectionMember{Staker: fmt.Sprintf("0x%040x", 101+i), Operator: fmt.Sprintf("0x%040x", 1+i), BLS: "0x" + fmt.Sprintf("%096x", i+1)})
	}
	for i := range p.Addition.Stages {
		tag := hexUint(uint64(83 + i))
		s := &p.Addition.Stages[i]
		s.Transaction["blockNumber"], s.Receipt["blockNumber"], s.Block["number"] = tag, tag, tag
	}
	for stage := 0; stage < 6; stage++ {
		boundary := uint64(40 + stage*10)
		trigger := boundary - 5
		if stage == 5 {
			trigger = 87
		}
		s := ncpSelectionSnapshot{TriggerHeight: trigger, Boundary: boundary, StateTag: hexUint(boundary), ValidatorTag: hexUint(boundary + 1), Active: map[string]string{}, NCP: map[string]string{}, Staked: map[string]string{}}
		registered := map[int]bool{}
		registeredList := []string{}
		council := []string{}
		for _, i := range ncpSelectionRegistered(stage) {
			registered[i] = true
			registeredList = append(registeredList, p.Members[i].Staker)
		}
		for i, m := range p.Members {
			s.Active[m.Staker] = ncpBallotWord(0)
			s.Staked[m.Staker] = ncpBallotWord(0)
			if registered[i] {
				s.Active[m.Staker] = ncpBallotWord(1)
				s.Staked[m.Staker] = "0x" + fmt.Sprintf("%064x", ncpSelectionAmount(i, stage >= 4))
			}
			s.NCP[m.Operator] = ncpBallotWord(0)
			if i < 7 || stage == 5 {
				s.NCP[m.Operator] = ncpBallotWord(1)
				council = append(council, m.Operator)
			}
		}
		s.StakerCount = ncpBallotWord(uint64(len(registeredList)))
		s.StakerList = ncpBallotList(registeredList)
		s.CouncilCount = ncpBallotWord(uint64(len(council)))
		s.CouncilList = ncpBallotList(council)
		for i := 0; i < 2; i++ {
			s.Blocks[i] = map[string]any{"number": hexUint(boundary + uint64(i)), "hash": fmt.Sprintf("0x%064x", 1000+stage*2+i), "extraData": "0x" + strings.Repeat("ab", 40)}
			if i == 1 {
				s.Blocks[i]["parentHash"] = s.Blocks[0]["hash"]
			}
		}
		stakerIDs, validatorIDs := ncpSelectionSets(stage)
		rows, validators := []any{}, []any{}
		positions := map[int]int{}
		for j, i := range stakerIDs {
			rows = append(rows, map[string]any{"addr": p.Members[i].Staker, "diligence": "0x1"})
			positions[i] = j
		}
		for _, i := range validatorIDs {
			m := p.Members[i]
			validators = append(validators, map[string]any{"addr": m.Staker, "bls": m.BLS, "index": hexUint(uint64(positions[i]))})
			s.Validators = append(s.Validators, m.Staker)
		}
		s.Extras[0] = map[string]any{"epochInfo": map[string]any{"stabilizing": stage < 3, "stakers": rows, "validators": validators}}
		s.Extras[1] = map[string]any{"epochInfo": nil}
		p.Snapshots = append(p.Snapshots, s)
	}
	return p
}

func TestNCPSelectionRejectsFlagsWithoutEligibility(t *testing.T) {
	if err := verifyNCPSelection(ncpSelectionFixture()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*ncpSelectionProof)
	}{
		{"wrong target", func(p *ncpSelectionProof) { p.Target = 4 }},
		{"wrong threshold", func(p *ncpSelectionProof) { p.Threshold = 4 }},
		{"NCP disabled", func(p *ncpSelectionProof) { p.UseNCP = false }},
		{"only four seeds", func(p *ncpSelectionProof) { p.Members = p.Members[:4] }},
		{"duplicate staker", func(p *ncpSelectionProof) { p.Members[7].Staker = p.Members[0].Staker }},
		{"duplicate operator", func(p *ncpSelectionProof) { p.Members[7].Operator = p.Members[0].Operator }},
		{"wrong BLS identity", func(p *ncpSelectionProof) { p.Members[0].BLS = "0x" + strings.Repeat("ff", 48) }},
		{"state latest", func(p *ncpSelectionProof) { p.Snapshots[0].StateTag = "latest" }},
		{"wrong validator origin", func(p *ncpSelectionProof) { p.Snapshots[5].ValidatorTag = p.Snapshots[4].ValidatorTag }},
		{"later epoch substitute", func(p *ncpSelectionProof) {
			s := &p.Snapshots[5]
			s.Boundary += 10
			s.StateTag = hexUint(s.Boundary)
			s.ValidatorTag = hexUint(s.Boundary + 1)
			for i := range s.Blocks {
				s.Blocks[i]["number"] = hexUint(s.Boundary + uint64(i))
			}
		}},
		{"wrong next parent", func(p *ncpSelectionProof) { p.Snapshots[0].Blocks[1]["parentHash"] = "other" }},
		{"first staker absent", func(p *ncpSelectionProof) { p.Snapshots[0].Active[p.Members[0].Staker] = ncpBallotWord(0) }},
		{"nonNCP counted early", func(p *ncpSelectionProof) { p.Snapshots[2].NCP[p.Members[7].Operator] = ncpBallotWord(1) }},
		{"nonNCP registration missing", func(p *ncpSelectionProof) { p.Snapshots[2].Active[p.Members[7].Staker] = ncpBallotWord(0) }},
		{"wrong registered count", func(p *ncpSelectionProof) { p.Snapshots[2].StakerCount = ncpBallotWord(4) }},
		{"wrong registered list", func(p *ncpSelectionProof) { p.Snapshots[2].StakerList = p.Snapshots[1].StakerList }},
		{"four NCP count was five", func(p *ncpSelectionProof) {
			p.Snapshots[1].NCP[p.Members[4].Operator] = ncpBallotWord(0)
			p.Snapshots[1].Active[p.Members[4].Staker] = ncpBallotWord(1)
		}},
		{"A below original40M", func(p *ncpSelectionProof) { p.Snapshots[3].Staked[p.Members[0].Staker] = ncpBallotWord(1) }},
		{"H not45M", func(p *ncpSelectionProof) {
			p.Snapshots[4].Staked[p.Members[7].Staker] = "0x" + fmt.Sprintf("%064x", ncpSelectionAmount(7, false))
		}},
		{"bootstrap set truncated", func(p *ncpSelectionProof) { p.Snapshots[2].Validators = p.Snapshots[2].Validators[:7] }},
		{"nonNCP highest included", func(p *ncpSelectionProof) { p.Snapshots[4].Validators[6] = p.Members[7].Staker }},
		{"highest excluded after permission", func(p *ncpSelectionProof) { p.Snapshots[5].Validators[0] = p.Members[6].Staker }},
		{"target count wrong", func(p *ncpSelectionProof) {
			p.Snapshots[5].Validators = append(p.Snapshots[5].Validators, p.Members[6].Staker)
		}},
		{"wrong epoch staker", func(p *ncpSelectionProof) {
			p.Snapshots[4].Extras[0]["epochInfo"].(map[string]any)["stakers"].([]any)[6].(map[string]any)["addr"] = p.Members[7].Staker
		}},
		{"wrong epoch BLS", func(p *ncpSelectionProof) {
			p.Snapshots[5].Extras[0]["epochInfo"].(map[string]any)["validators"].([]any)[0].(map[string]any)["bls"] = "0x" + strings.Repeat("ff", 48)
		}},
		{"ordinary epoch not null", func(p *ncpSelectionProof) { p.Snapshots[5].Extras[1]["epochInfo"] = map[string]any{} }},
		{"H not council member", func(p *ncpSelectionProof) { p.Snapshots[5].NCP[p.Members[7].Operator] = ncpBallotWord(0) }},
		{"council list not extended", func(p *ncpSelectionProof) { p.Snapshots[5].CouncilList = p.Snapshots[4].CouncilList }},
		{"ballot lacked majority", func(p *ncpSelectionProof) { p.Addition.Stages = append(p.Addition.Stages[:3], p.Addition.Stages[4]) }},
		{"permission predates ranking", func(p *ncpSelectionProof) {
			s := &p.Addition.Stages[0]
			s.Block["number"] = "0x1a"
			s.Transaction["blockNumber"] = "0x1a"
			s.Receipt["blockNumber"] = "0x1a"
		}},
		{"wrong promoted candidate", func(p *ncpSelectionProof) { p.Addition.Candidate = p.Members[6].Operator }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := ncpSelectionFixture()
			tc.change(&p)
			if verifyNCPSelection(p) == nil {
				t.Fatal("flags hid wrong original NCP counts, rank or eligibility")
			}
		})
	}
}
