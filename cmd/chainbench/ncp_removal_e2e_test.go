//go:build e2e

package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/accounts"
)

type ncpExclusionEpoch struct {
	Boundary            uint64
	BeforeTag, AfterTag string
	Blocks              [3]map[string]any
	Extras              [2]map[string]any
	Before, After       []string
	Expected            []epochBoundaryMember
}
type ncpRemovalProof struct {
	InitialCouncil, BeforeCouncil, Voters                                           []string
	Operator, Staker                                                                string
	Self                                                                            bool
	ID                                                                              uint64
	Stages                                                                          []ncpBallotStage
	BeforeStake, AfterStake, BeforeActive, AfterActive, BeforeMapping, AfterMapping string
	Epoch                                                                           ncpExclusionEpoch
}

func verifyNCPRemoval(p ncpRemovalProof) error {
	if len(p.InitialCouncil) != 7 || len(p.BeforeCouncil) < 3 || (!p.Self && len(p.BeforeCouncil) != 7) || !epochBoundaryBytes(p.Operator, 20) || !epochBoundaryBytes(p.Staker, 20) || strings.EqualFold(p.Operator, p.Staker) {
		return fmt.Errorf("wrong initial council or target roles")
	}
	initial := map[string]bool{}
	for _, a := range p.InitialCouncil {
		a = strings.ToLower(a)
		if !epochBoundaryBytes(a, 20) || initial[a] {
			return fmt.Errorf("invalid original council")
		}
		initial[a] = true
	}
	members := map[string]bool{}
	for _, a := range p.BeforeCouncil {
		a = strings.ToLower(a)
		if !initial[a] || members[a] {
			return fmt.Errorf("invalid active council")
		}
		members[a] = true
	}
	if !members[strings.ToLower(p.Operator)] || !members[strings.ToLower(p.InitialCouncil[0])] {
		return fmt.Errorf("target or proposer not a member")
	}
	if p.Self {
		if len(p.Voters) != 0 || len(p.Stages) != 1 {
			return fmt.Errorf("self exit must be one transaction without votes")
		}
	} else {
		if len(p.Voters) != len(p.BeforeCouncil)/2+1 || len(p.Stages) != len(p.Voters)+1 {
			return fmt.Errorf("not exact strict majority")
		}
		seen := map[string]bool{}
		for _, a := range p.Voters {
			a = strings.ToLower(a)
			if !members[a] || seen[a] || strings.EqualFold(a, p.Operator) {
				return fmt.Errorf("invalid independent removal voter")
			}
			seen[a] = true
		}
	}
	at, ok := lifecycleQuantity(p.Stages[0].Block["timestamp"])
	if !ok || !at.IsUint64() || at.Sign() <= 0 {
		return fmt.Errorf("missing proposal timestamp")
	}
	var last uint64
	for i, s := range p.Stages {
		from := p.InitialCouncil[0]
		input := accounts.EncodeCallArgs("newProposalToRemoveNCP(address)", accounts.Address(p.Operator))
		if p.Self {
			from = p.Operator
		} else if i > 0 {
			from = p.Voters[i-1]
			input = accounts.EncodeCallArgs("vote(uint256,bool)", accounts.Uint(new(big.Int).SetUint64(p.ID)), accounts.Word([]byte{1}))
		}
		height, err := verifyNCPCanonicalStage(s, from, input)
		if err != nil {
			return err
		}
		if height <= last {
			return fmt.Errorf("unordered ballot")
		}
		last = height
		if !ncpBallotSameLogs(s.Logs, ncpRemovalLogs(p, i, at.Uint64())) {
			return fmt.Errorf("wrong removal/self-exit event sequence or arguments")
		}
		final := p.Self || i == len(p.Voters)
		count := len(p.BeforeCouncil)
		member := uint64(1)
		id := p.ID
		list := p.BeforeCouncil
		if final {
			count--
			member = 0
			id++
			list = ncpWithout(list, p.Operator)
		}
		if s.Count != ncpBallotWord(uint64(count)) || s.CandidateMember != ncpBallotWord(member) || s.CurrentID != ncpBallotWord(id) || !strings.EqualFold(s.List, ncpBallotList(list)) || !strings.EqualFold(s.Proposal, ncpRemovalProposal(p, i, at.Uint64())) {
			return fmt.Errorf("incorrect pinned removal boundary state")
		}
	}
	stake, err := hex.DecodeString(strings.TrimPrefix(p.BeforeStake, "0x"))
	if err != nil || len(stake) < 6*32 || !strings.EqualFold("0x"+hex.EncodeToString(stake[12:32]), p.Operator) || new(big.Int).SetBytes(stake[160:192]).String() != governanceForkFixture().Configured[0] || p.BeforeStake != p.AfterStake || p.BeforeActive != ncpBallotWord(1) || p.AfterActive != ncpBallotWord(1) || !strings.EqualFold(p.BeforeMapping, ncpBallotABI(accounts.Address(p.Staker))) || p.AfterMapping != p.BeforeMapping {
		return fmt.Errorf("staking principal, full info, activity or operator mapping changed")
	}
	if err := verifyNCPExclusion(p.Epoch, p.Staker, last); err != nil {
		return err
	}
	return nil
}

func verifyNCPCanonicalStage(s ncpBallotStage, from, input string) (uint64, error) {
	raw, err := hex.DecodeString(strings.TrimPrefix(s.Raw, "0x"))
	if err != nil || len(raw) < 2 {
		return 0, fmt.Errorf("missing raw")
	}
	hash := rawSubmissionHash(raw)
	t, r, b := s.Transaction, s.Receipt, s.Block
	if t["hash"] != hash || r["transactionHash"] != hash || r["status"] != "0x1" || !strings.EqualFold(fmt.Sprint(t["from"]), from) || !strings.EqualFold(fmt.Sprint(t["to"]), e2eGovNCP) || t["input"] != input || t["value"] != "0x0" || !epochBoundaryBytes(b["hash"], 32) || b["hash"] != r["blockHash"] || b["hash"] != t["blockHash"] || b["number"] != r["blockNumber"] || b["number"] != t["blockNumber"] || t["transactionIndex"] != r["transactionIndex"] {
		return 0, fmt.Errorf("not exact successful canonical transaction")
	}
	h, ok := lifecycleQuantity(b["number"])
	if !ok || !h.IsUint64() || h.Uint64() <= 20 {
		return 0, fmt.Errorf("wrong postfork height")
	}
	index, ok := lifecycleQuantity(r["transactionIndex"])
	if !ok || !index.IsUint64() {
		return 0, fmt.Errorf("missing mined index")
	}
	txs, ok := b["transactions"].([]any)
	if !ok || index.Uint64() >= uint64(len(txs)) || txs[index.Uint64()] != hash {
		return 0, fmt.Errorf("missing canonical inclusion")
	}
	occurrences := 0
	for _, v := range txs {
		if v == hash {
			occurrences++
		}
	}
	if occurrences != 1 {
		return 0, fmt.Errorf("duplicate canonical inclusion")
	}
	var logs []accounts.Log
	data, err := json.Marshal(r["logs"])
	if err != nil {
		return 0, err
	}
	if err = json.Unmarshal(data, &logs); err != nil {
		return 0, err
	}
	if !ncpBallotSameLogs(logs, s.Logs) {
		return 0, fmt.Errorf("logs not from receipt")
	}
	return h.Uint64(), nil
}

func verifyNCPExclusion(e ncpExclusionEpoch, removed string, last uint64) error {
	if e.Boundary != (last/10+1)*10 || e.BeforeTag != hexUint(e.Boundary-1) || e.AfterTag != hexUint(e.Boundary+1) || len(e.Expected) < 2 || len(e.After) != len(e.Expected) || len(e.Before) != len(e.After)+1 {
		return fmt.Errorf("wrong next epoch or cardinality")
	}
	wanted := map[string]string{}
	for _, m := range e.Expected {
		a := strings.ToLower(m.Address)
		if !epochBoundaryBytes(a, 20) || !epochBoundaryBytes(m.BLS, 48) || wanted[a] != "" || strings.EqualFold(a, removed) {
			return fmt.Errorf("invalid retained candidate identity/key")
		}
		wanted[a] = strings.ToLower(m.BLS)
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
	before := map[string]string{}
	for k, v := range wanted {
		before[k] = v
	}
	before[strings.ToLower(removed)] = "removed"
	if !checkSet(e.Before, before) || !checkSet(e.After, wanted) {
		return fmt.Errorf("removed staker not excluded at exactly next epoch")
	}
	for i, b := range e.Blocks {
		if b["number"] != hexUint(e.Boundary-1+uint64(i)) || !epochBoundaryBytes(b["hash"], 32) || (i > 0 && b["parentHash"] != e.Blocks[i-1]["hash"]) || len(fmt.Sprint(b["extraData"])) <= 66 {
			return fmt.Errorf("wrong pinned epoch header/link")
		}
	}
	ordinary, present := e.Extras[1]["epochInfo"]
	if !present || ordinary != nil {
		return fmt.Errorf("ordinary next header must have null epoch")
	}
	ep, ok := e.Extras[0]["epochInfo"].(map[string]any)
	if !ok || ep["stabilizing"] != false {
		return fmt.Errorf("missing decided epoch")
	}
	stakers, ok := ep["stakers"].([]any)
	if !ok || len(stakers) != len(wanted) {
		return fmt.Errorf("wrong epoch candidates")
	}
	addresses := []string{}
	for _, v := range stakers {
		row, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid staker")
		}
		addresses = append(addresses, strings.ToLower(fmt.Sprint(row["addr"])))
	}
	if !checkSet(addresses, wanted) {
		return fmt.Errorf("epoch includes removed staker or omits survivor")
	}
	validators, ok := ep["validators"].([]any)
	if !ok || len(validators) != len(wanted) {
		return fmt.Errorf("incorrect decided validators")
	}
	seen := map[string]bool{}
	for _, v := range validators {
		row, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid validator")
		}
		a := strings.ToLower(fmt.Sprint(row["addr"]))
		index, valid := lifecycleQuantity(row["index"])
		if !valid || !index.IsUint64() || index.Uint64() >= uint64(len(addresses)) || addresses[index.Uint64()] != a || seen[a] || wanted[a] == "" || wanted[a] != strings.ToLower(fmt.Sprint(row["bls"])) {
			return fmt.Errorf("wrong indexed validator/BLS")
		}
		seen[a] = true
	}
	return nil
}

func ncpWithout(xs []string, target string) []string {
	out := append([]string(nil), xs...)
	for i, a := range out {
		if strings.EqualFold(a, target) {
			out[i] = out[len(out)-1]
			return out[:len(out)-1]
		}
	}
	return out
}
func ncpRemovalLogs(p ncpRemovalProof, i int, at uint64) []accounts.Log {
	proposer := p.InitialCouncil[0]
	if p.Self {
		proposer = p.Operator
	}
	log := func(sig string, indexed, data string) accounts.Log {
		return accounts.Log{Address: e2eGovNCP, Topics: []string{accounts.EventTopic(sig), indexed}, Data: data}
	}
	out := []accounts.Log{}
	if i == 0 {
		data := ncpBallotABI(accounts.Uint(big.NewInt(2)), accounts.Address(p.Operator), accounts.Address(proposer), accounts.Uint(new(big.Int).SetUint64(at)), accounts.Uint(new(big.Int).SetUint64(at+604800)))
		out = append(out, log("NewProposal(uint256,uint256,address,address,uint256,uint256)", ncpBallotWord(p.ID), data))
	} else {
		out = append(out, log("Vote(uint256,address,bool)", ncpBallotWord(p.ID), ncpBallotABI(accounts.Address(p.Voters[i-1]), accounts.Word([]byte{1}))))
	}
	if p.Self || i == len(p.Voters) {
		out = append(out, log("NCPRemoved(address)", "0x"+strings.Repeat("0", 24)+p.Operator[2:], "0x"), log("ProposalFinalized(uint256,bool)", ncpBallotWord(p.ID), ncpBallotWord(1)))
	}
	return out
}
func ncpRemovalProposal(p ncpRemovalProof, i int, at uint64) string {
	proposer := p.InitialCouncil[0]
	if p.Self {
		proposer = p.Operator
	}
	ids := map[string]int64{}
	for j, a := range p.InitialCouncil {
		ids[strings.ToLower(a)] = int64(j + 1)
	}
	state := int64(1)
	if p.Self || i == len(p.Voters) {
		state = 2
	}
	return ncpBallotABI(accounts.Uint(big.NewInt(state)), accounts.Uint(new(big.Int).SetUint64(at)), accounts.Uint(new(big.Int).SetUint64(at+604800)), accounts.Uint(big.NewInt(ids[strings.ToLower(proposer)])), accounts.Address("0x"+strings.Repeat("0", 40)), accounts.Uint(big.NewInt(ids[strings.ToLower(p.Operator)])), accounts.Uint(big.NewInt(2)), accounts.Uint(big.NewInt(int64(i))), accounts.Uint(big.NewInt(0)))
}
func ncpRemovalFixture(self bool) ncpRemovalProof {
	p := ncpRemovalProof{Self: self, Staker: fmt.Sprintf("0x%040x", 4), BeforeActive: ncpBallotWord(1), AfterActive: ncpBallotWord(1)}
	for i := 1; i <= 7; i++ {
		p.InitialCouncil = append(p.InitialCouncil, fmt.Sprintf("0x%040x", 10+i))
	}
	p.BeforeCouncil = append([]string(nil), p.InitialCouncil...)
	p.Operator = p.InitialCouncil[3]
	if self {
		p.ID = 1
		p.Staker = fmt.Sprintf("0x%040x", 3)
		p.Operator = p.InitialCouncil[2]
		p.BeforeCouncil = ncpWithout(p.BeforeCouncil, p.InitialCouncil[3])
	} else {
		p.Voters = []string{p.InitialCouncil[0], p.InitialCouncil[1], p.InitialCouncil[2], p.InitialCouncil[4]}
	}
	p.BeforeMapping = ncpBallotABI(accounts.Address(p.Staker))
	p.AfterMapping = p.BeforeMapping
	amount, _ := new(big.Int).SetString(governanceForkFixture().Configured[0], 10)
	p.BeforeStake = ncpBallotABI(accounts.Address(p.Operator), accounts.Address(fmt.Sprintf("0x%040x", 99)), accounts.Address(p.Staker), accounts.Uint(big.NewInt(0)), accounts.Uint(big.NewInt(224)), accounts.Uint(amount), accounts.Uint(big.NewInt(0)), accounts.Uint(big.NewInt(0)))
	p.AfterStake = p.BeforeStake
	count := len(p.Voters) + 1
	for i := 0; i < count; i++ {
		raw := fmt.Sprintf("0x0201%02x", i)
		b, _ := hex.DecodeString(raw[2:])
		hash := rawSubmissionHash(b)
		bh := fmt.Sprintf("0x%064x", 300+i)
		tag := hexUint(uint64(33 + i))
		from := p.InitialCouncil[0]
		input := accounts.EncodeCallArgs("newProposalToRemoveNCP(address)", accounts.Address(p.Operator))
		if self {
			from = p.Operator
		} else if i > 0 {
			from = p.Voters[i-1]
			input = accounts.EncodeCallArgs("vote(uint256,bool)", accounts.Uint(new(big.Int).SetUint64(p.ID)), accounts.Word([]byte{1}))
		}
		logs := ncpRemovalLogs(p, i, 2000)
		s := ncpBallotStage{Raw: raw, Transaction: map[string]any{"hash": hash, "from": from, "to": e2eGovNCP, "input": input, "value": "0x0", "blockHash": bh, "blockNumber": tag, "transactionIndex": "0x0"}, Receipt: map[string]any{"transactionHash": hash, "status": "0x1", "blockHash": bh, "blockNumber": tag, "transactionIndex": "0x0", "logs": logs}, Block: map[string]any{"hash": bh, "number": tag, "timestamp": hexUint(uint64(2000 + i)), "transactions": []any{hash}}, Logs: logs, Count: ncpBallotWord(uint64(len(p.BeforeCouncil))), List: ncpBallotList(p.BeforeCouncil), CandidateMember: ncpBallotWord(1), CurrentID: ncpBallotWord(p.ID), Proposal: ncpRemovalProposal(p, i, 2000)}
		if self || i == len(p.Voters) {
			s.Count = ncpBallotWord(uint64(len(p.BeforeCouncil) - 1))
			s.List = ncpBallotList(ncpWithout(p.BeforeCouncil, p.Operator))
			s.CandidateMember = ncpBallotWord(0)
			s.CurrentID = ncpBallotWord(p.ID + 1)
		}
		p.Stages = append(p.Stages, s)
	}
	e := ncpExclusionEpoch{Boundary: 40, BeforeTag: "0x27", AfterTag: "0x29"}
	stakers, validators := []any{}, []any{}
	last := 4
	if self {
		last = 3
	}
	for i := 1; i <= last; i++ {
		a := fmt.Sprintf("0x%040x", i)
		e.Before = append(e.Before, a)
		if a == p.Staker {
			continue
		}
		m := epochBoundaryMember{Address: a, BLS: "0x" + fmt.Sprintf("%096x", i)}
		e.Expected = append(e.Expected, m)
		e.After = append(e.After, a)
		stakers = append(stakers, map[string]any{"addr": a, "diligence": "0x1"})
		validators = append(validators, map[string]any{"addr": a, "index": hexUint(uint64(len(stakers) - 1)), "bls": m.BLS})
	}
	for i := 0; i < 3; i++ {
		e.Blocks[i] = map[string]any{"number": hexUint(uint64(39 + i)), "hash": fmt.Sprintf("0x%064x", 500+i), "extraData": "0x" + strings.Repeat("ab", 40)}
		if i > 0 {
			e.Blocks[i]["parentHash"] = e.Blocks[i-1]["hash"]
		}
	}
	e.Extras[0] = map[string]any{"epochInfo": map[string]any{"stabilizing": false, "stakers": stakers, "validators": validators}}
	e.Extras[1] = map[string]any{"epochInfo": nil}
	p.Epoch = e
	return p
}

func TestNCPRemovalRejectsCountAndMembershipOnly(t *testing.T) {
	for _, self := range []bool{false, true} {
		t.Run(fmt.Sprintf("self%v", self), func(t *testing.T) {
			if err := verifyNCPRemoval(ncpRemovalFixture(self)); err != nil {
				t.Fatal(err)
			}
			for _, tc := range []struct {
				name   string
				change func(*ncpRemovalProof)
			}{
				{"too small council", func(p *ncpRemovalProof) {
					p.BeforeCouncil = p.BeforeCouncil[:2]
					p.Stages[len(p.Stages)-1].Count = ncpBallotWord(1)
				}},
				{"duplicate council", func(p *ncpRemovalProof) { p.BeforeCouncil[1] = p.BeforeCouncil[0] }},
				{"wrong raw", func(p *ncpRemovalProof) { p.Stages[0].Raw = "0x02010001" }},
				{"wrong sender", func(p *ncpRemovalProof) { p.Stages[0].Transaction["from"] = p.InitialCouncil[5] }},
				{"wrong input", func(p *ncpRemovalProof) { p.Stages[0].Transaction["input"] = "0x" }},
				{"wrong canonical block", func(p *ncpRemovalProof) { p.Stages[0].Block["hash"] = "other" }},
				{"missing transaction", func(p *ncpRemovalProof) { p.Stages[0].Block["transactions"] = []any{} }},
				{"duplicate transaction", func(p *ncpRemovalProof) {
					hash := p.Stages[0].Transaction["hash"]
					p.Stages[0].Block["transactions"] = []any{hash, hash}
				}},
				{"invalid voters", func(p *ncpRemovalProof) {
					if p.Self {
						p.Voters = []string{p.InitialCouncil[0]}
					} else {
						p.Voters[1] = p.Voters[0]
					}
				}},
				{"extra ballot transaction", func(p *ncpRemovalProof) { p.Stages = append(p.Stages, p.Stages[len(p.Stages)-1]) }},
				{"wrong event emitter", func(p *ncpRemovalProof) { p.Stages[0].Logs[0].Address = p.Operator }},
				{"wrong proposal type", func(p *ncpRemovalProof) { p.Stages[0].Logs[0].Data = "0x" }},
				{"missing removal event", func(p *ncpRemovalProof) { p.Stages[len(p.Stages)-1].Logs = p.Stages[len(p.Stages)-1].Logs[:1] }},
				{"rejected final event", func(p *ncpRemovalProof) {
					p.Stages[len(p.Stages)-1].Logs[len(p.Stages[len(p.Stages)-1].Logs)-1].Data = ncpBallotWord(0)
				}},
				{"incorrect final list", func(p *ncpRemovalProof) { p.Stages[len(p.Stages)-1].List = ncpBallotList(p.BeforeCouncil) }},
				{"no proposal ID advance", func(p *ncpRemovalProof) { p.Stages[len(p.Stages)-1].CurrentID = ncpBallotWord(p.ID) }},
				{"wrong proposal state", func(p *ncpRemovalProof) { p.Stages[len(p.Stages)-1].Proposal = "0x" }},
				{"staking amount changed", func(p *ncpRemovalProof) { p.AfterStake = "0x" }},
				{"staking removed", func(p *ncpRemovalProof) { p.AfterActive = ncpBallotWord(0) }},
				{"operator mapping removed", func(p *ncpRemovalProof) { p.AfterMapping = ncpBallotWord(0) }},
				{"epoch not reached", func(p *ncpRemovalProof) { p.Epoch.Boundary = 30 }},
				{"later epoch substitutes for first", func(p *ncpRemovalProof) {
					p.Epoch.Boundary = 50
					p.Epoch.BeforeTag = "0x31"
					p.Epoch.AfterTag = "0x33"
					for i := range p.Epoch.Blocks {
						p.Epoch.Blocks[i]["number"] = hexUint(uint64(49 + i))
					}
				}},
				{"wrong epoch tags", func(p *ncpRemovalProof) { p.Epoch.AfterTag = p.Epoch.BeforeTag }},
				{"wrong epoch parent", func(p *ncpRemovalProof) { p.Epoch.Blocks[2]["parentHash"] = "other" }},
				{"unchanged validators", func(p *ncpRemovalProof) { p.Epoch.After = append([]string(nil), p.Epoch.Before...) }},
				{"wrong remaining BLS", func(p *ncpRemovalProof) { p.Epoch.Expected[0].BLS = "0x" + strings.Repeat("ff", 48) }},
				{"ordinary epoch not null", func(p *ncpRemovalProof) { p.Epoch.Extras[1]["epochInfo"] = map[string]any{} }},
			} {
				t.Run(tc.name, func(t *testing.T) {
					p := ncpRemovalFixture(self)
					tc.change(&p)
					if verifyNCPRemoval(p) == nil {
						t.Fatal("removal count hid wrong events, stake or next epoch")
					}
				})
			}
		})
	}
}
