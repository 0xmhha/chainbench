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

type ncpBallotStage struct {
	Raw                                               string
	Transaction, Receipt, Block                       map[string]any
	Logs                                              []accounts.Log
	Count, List, CandidateMember, Proposal, CurrentID string
}
type ncpMajorityProof struct {
	Members             []string
	Candidate, Proposer string
	ID                  uint64
	Stages              []ncpBallotStage
}

func verifyNCPMajority(p ncpMajorityProof) error {
	if len(p.Members) != 7 || len(p.Stages) != 5 || !epochBoundaryBytes(p.Candidate, 20) || strings.EqualFold(p.Candidate, "0x"+strings.Repeat("0", 40)) {
		return fmt.Errorf("not seven-member/four-vote electorate")
	}
	seen := map[string]bool{}
	for _, a := range p.Members {
		a = strings.ToLower(a)
		if !epochBoundaryBytes(a, 20) || seen[a] || strings.EqualFold(a, p.Candidate) {
			return fmt.Errorf("invalid electorate or existing candidate")
		}
		seen[a] = true
	}
	if !strings.EqualFold(p.Proposer, p.Members[0]) {
		return fmt.Errorf("incorrect initial proposer identity")
	}
	at, ok := lifecycleQuantity(p.Stages[0].Block["timestamp"])
	if !ok || !at.IsUint64() || at.Sign() <= 0 {
		return fmt.Errorf("missing mined proposal timestamp")
	}
	var lastHeight uint64
	for i, s := range p.Stages {
		raw, err := hex.DecodeString(strings.TrimPrefix(s.Raw, "0x"))
		if err != nil || len(raw) < 2 {
			return fmt.Errorf("missing submitted raw")
		}
		hash := rawSubmissionHash(raw)
		tx, r, b := s.Transaction, s.Receipt, s.Block
		input := accounts.EncodeCallArgs("newProposalToAddNCP(address)", accounts.Address(p.Candidate))
		from := p.Proposer
		if i > 0 {
			input = accounts.EncodeCallArgs("vote(uint256,bool)", accounts.Uint(new(big.Int).SetUint64(p.ID)), accounts.Word([]byte{1}))
			from = p.Members[i-1]
		}
		if tx["hash"] != hash || r["transactionHash"] != hash || r["status"] != "0x1" || !strings.EqualFold(fmt.Sprint(tx["from"]), from) || !strings.EqualFold(fmt.Sprint(tx["to"]), e2eGovNCP) || tx["input"] != input || tx["value"] != "0x0" || !epochBoundaryBytes(b["hash"], 32) || b["hash"] != r["blockHash"] || b["hash"] != tx["blockHash"] || b["number"] != r["blockNumber"] || b["number"] != tx["blockNumber"] || tx["transactionIndex"] != r["transactionIndex"] {
			return fmt.Errorf("wrong submission, mined receipt or canonical block")
		}
		height, ok := lifecycleQuantity(b["number"])
		if !ok || !height.IsUint64() || height.Uint64() <= 20 || height.Uint64() <= lastHeight {
			return fmt.Errorf("unordered postfork ballot blocks")
		}
		lastHeight = height.Uint64()
		index, ok := lifecycleQuantity(r["transactionIndex"])
		if !ok || !index.IsUint64() {
			return fmt.Errorf("missing actual inclusion index")
		}
		transactions, ok := b["transactions"].([]any)
		if !ok || index.Uint64() >= uint64(len(transactions)) || transactions[index.Uint64()] != hash {
			return fmt.Errorf("not actual receipt-indexed inclusion")
		}
		occurrences := 0
		for _, h := range transactions {
			if h == hash {
				occurrences++
			}
		}
		if occurrences != 1 {
			return fmt.Errorf("duplicate mined inclusion")
		}
		var receiptLogs []accounts.Log
		data, err := json.Marshal(r["logs"])
		if err != nil {
			return err
		}
		if err = json.Unmarshal(data, &receiptLogs); err != nil {
			return err
		}
		if !ncpBallotSameLogs(s.Logs, receiptLogs) || !ncpBallotSameLogs(s.Logs, ncpBallotLogs(p, i, at.Uint64())) {
			return fmt.Errorf("wrong exact proposal, vote, addition or finalization events")
		}
		count, member, currentID := uint64(7), uint64(0), p.ID
		members := append([]string(nil), p.Members...)
		if i == 4 {
			count = 8
			member = 1
			currentID++
			members = append(members, p.Candidate)
		}
		if s.Count != ncpBallotWord(count) || s.CandidateMember != ncpBallotWord(member) || !strings.EqualFold(s.List, ncpBallotList(members)) || s.CurrentID != ncpBallotWord(currentID) || !strings.EqualFold(s.Proposal, ncpBallotProposal(p, i, at.Uint64())) {
			return fmt.Errorf("wrong pinned membership or majority boundary state")
		}
	}
	return nil
}

func ncpBallotWord(n uint64) string { return "0x" + fmt.Sprintf("%064x", n) }
func ncpBallotABI(args ...accounts.Arg) string {
	return "0x" + accounts.EncodeCallArgs("fixture()", args...)[10:]
}
func ncpBallotList(members []string) string {
	s := ncpBallotWord(32) + ncpBallotWord(uint64(len(members)))[2:]
	for _, a := range members {
		s += strings.Repeat("0", 24) + strings.TrimPrefix(strings.ToLower(a), "0x")
	}
	return s
}
func ncpBallotLogs(p ncpMajorityProof, i int, at uint64) []accounts.Log {
	log := func(sig string, data string, indexed string) accounts.Log {
		return accounts.Log{Address: e2eGovNCP, Topics: []string{accounts.EventTopic(sig), indexed}, Data: data}
	}
	if i == 0 {
		return []accounts.Log{log("NewProposal(uint256,uint256,address,address,uint256,uint256)", ncpBallotABI(accounts.Uint(big.NewInt(1)), accounts.Address(p.Candidate), accounts.Address(p.Proposer), accounts.Uint(new(big.Int).SetUint64(at)), accounts.Uint(new(big.Int).SetUint64(at+604800))), ncpBallotWord(p.ID))}
	}
	out := []accounts.Log{log("Vote(uint256,address,bool)", ncpBallotABI(accounts.Address(p.Members[i-1]), accounts.Word([]byte{1})), ncpBallotWord(p.ID))}
	if i == 4 {
		out = append(out, log("NCPAdded(address)", "0x", "0x"+strings.Repeat("0", 24)+p.Candidate[2:]), log("ProposalFinalized(uint256,bool)", ncpBallotWord(1), ncpBallotWord(p.ID)))
	}
	return out
}
func ncpBallotProposal(p ncpMajorityProof, i int, at uint64) string {
	state := int64(1)
	if i == 4 {
		state = 2
	}
	return ncpBallotABI(accounts.Uint(big.NewInt(state)), accounts.Uint(new(big.Int).SetUint64(at)), accounts.Uint(new(big.Int).SetUint64(at+604800)), accounts.Uint(big.NewInt(1)), accounts.Address(p.Candidate), accounts.Uint(big.NewInt(0)), accounts.Uint(big.NewInt(1)), accounts.Uint(big.NewInt(int64(i))), accounts.Uint(big.NewInt(0)))
}
func ncpMajorityFixture() ncpMajorityProof {
	p := ncpMajorityProof{Candidate: fmt.Sprintf("0x%040x", 8)}
	for i := 1; i <= 7; i++ {
		p.Members = append(p.Members, fmt.Sprintf("0x%040x", i))
	}
	p.Proposer = p.Members[0]
	for i := 0; i <= 4; i++ {
		raw := fmt.Sprintf("0x0200%02x", i)
		b, _ := hex.DecodeString(raw[2:])
		hash := rawSubmissionHash(b)
		input := accounts.EncodeCallArgs("newProposalToAddNCP(address)", accounts.Address(p.Candidate))
		from := p.Proposer
		if i > 0 {
			input = accounts.EncodeCallArgs("vote(uint256,bool)", accounts.Uint(new(big.Int).SetUint64(p.ID)), accounts.Word([]byte{1}))
			from = p.Members[i-1]
		}
		blockHash := fmt.Sprintf("0x%064x", 100+i)
		tag := hexUint(uint64(30 + i))
		logs := ncpBallotLogs(p, i, 1000)
		s := ncpBallotStage{Raw: raw, Transaction: map[string]any{"hash": hash, "from": from, "to": e2eGovNCP, "input": input, "value": "0x0", "blockHash": blockHash, "blockNumber": tag, "transactionIndex": "0x0"}, Receipt: map[string]any{"transactionHash": hash, "status": "0x1", "blockHash": blockHash, "blockNumber": tag, "transactionIndex": "0x0", "logs": logs}, Block: map[string]any{"hash": blockHash, "number": tag, "timestamp": hexUint(uint64(1000 + i)), "transactions": []any{hash}}, Logs: logs, Count: ncpBallotWord(7), List: ncpBallotList(p.Members), CandidateMember: ncpBallotWord(0), CurrentID: ncpBallotWord(0), Proposal: ncpBallotProposal(p, i, 1000)}
		if i == 4 {
			s.Count = ncpBallotWord(8)
			s.List = ncpBallotList(append(append([]string(nil), p.Members...), p.Candidate))
			s.CandidateMember = ncpBallotWord(1)
			s.CurrentID = ncpBallotWord(1)
		}
		p.Stages = append(p.Stages, s)
	}
	return p
}

func TestNCPMajorityRejectsMembershipOnly(t *testing.T) {
	if err := verifyNCPMajority(ncpMajorityFixture()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*ncpMajorityProof)
	}{
		{"one-member electorate", func(p *ncpMajorityProof) { p.Members = p.Members[:1] }},
		{"duplicate electorate", func(p *ncpMajorityProof) { p.Members[6] = p.Members[0] }},
		{"candidate already member", func(p *ncpMajorityProof) { p.Candidate = p.Members[6] }},
		{"nonmember proposer", func(p *ncpMajorityProof) { p.Proposer = p.Candidate }},
		{"missing third vote", func(p *ncpMajorityProof) { p.Stages = append(p.Stages[:3], p.Stages[4]) }},
		{"reused voter", func(p *ncpMajorityProof) { p.Stages[2].Transaction["from"] = p.Members[0] }},
		{"rejected vote input", func(p *ncpMajorityProof) {
			p.Stages[2].Transaction["input"] = accounts.EncodeCallArgs("vote(uint256,bool)", accounts.Uint(big.NewInt(0)), accounts.Word([]byte{0}))
		}},
		{"wrong proposal target", func(p *ncpMajorityProof) { p.Stages[0].Transaction["input"] = "0x" }},
		{"wrong raw hash", func(p *ncpMajorityProof) { p.Stages[0].Raw = "0x020001" }},
		{"failed vote", func(p *ncpMajorityProof) { p.Stages[4].Receipt["status"] = "0x0" }},
		{"wrong block", func(p *ncpMajorityProof) { p.Stages[1].Block["hash"] = "other" }},
		{"missing mined inclusion", func(p *ncpMajorityProof) { p.Stages[1].Block["transactions"] = []any{} }},
		{"wrong event emitter", func(p *ncpMajorityProof) { p.Stages[0].Logs[0].Address = p.Candidate }},
		{"wrong proposal event", func(p *ncpMajorityProof) { p.Stages[0].Logs[0].Data = "0x" }},
		{"wrong vote event", func(p *ncpMajorityProof) { p.Stages[2].Logs[0].Data = "0x" }},
		{"early added event", func(p *ncpMajorityProof) { p.Stages[3].Logs = append(p.Stages[3].Logs, p.Stages[4].Logs[1]) }},
		{"missing added event", func(p *ncpMajorityProof) { p.Stages[4].Logs = p.Stages[4].Logs[:1] }},
		{"rejected final event", func(p *ncpMajorityProof) { p.Stages[4].Logs[2].Data = ncpBallotWord(0) }},
		{"early membership", func(p *ncpMajorityProof) { p.Stages[3].CandidateMember = ncpBallotWord(1) }},
		{"early count change", func(p *ncpMajorityProof) { p.Stages[3].Count = ncpBallotWord(8) }},
		{"incorrect final list", func(p *ncpMajorityProof) { p.Stages[4].List = ncpBallotList(p.Members) }},
		{"early proposal accepted", func(p *ncpMajorityProof) { p.Stages[3].Proposal = ncpBallotProposal(*p, 4, 1000) }},
		{"missing fourth acceptance", func(p *ncpMajorityProof) { p.Stages[4].Proposal = ncpBallotProposal(*p, 3, 1000) }},
		{"no proposal ID advance", func(p *ncpMajorityProof) { p.Stages[4].CurrentID = ncpBallotWord(0) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := ncpMajorityFixture()
			tc.change(&p)
			if verifyNCPMajority(p) == nil {
				t.Fatal("membership alone hid wrong majority or events")
			}
		})
	}
}

func ncpBallotSameLogs(a, b []accounts.Log) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return strings.EqualFold(string(x), string(y))
}
