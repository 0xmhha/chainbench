//go:build e2e

package main

import (
	"fmt"
	"math/big"
	"testing"
)

type belowBaseSample struct {
	Height   uint64
	BaseFee  string
	Included bool
	Receipt  map[string]any
}
type belowBaseProof struct {
	Cap                  string
	Submission           policySubmission
	Samples              []belowBaseSample
	Transaction, Receipt map[string]any
	Before, After        policyAccountState
}

func verifyBelowBaseProof(p belowBaseProof) error {
	classification, err := classifyPolicySubmission(p.Submission)
	if err != nil {
		return err
	}
	cap, ok := new(big.Int).SetString(p.Cap, 10)
	if !ok || cap.Sign() <= 0 || len(p.Samples) < 5 {
		return fmt.Errorf("missing positive cap or target block window")
	}
	if p.Before != p.After || p.Receipt != nil {
		return fmt.Errorf("transaction changed state")
	}
	for i, sample := range p.Samples {
		base, ok := new(big.Int).SetString(sample.BaseFee, 10)
		if !ok || base.Cmp(cap) <= 0 || sample.Included || sample.Receipt != nil || i > 0 && sample.Height != p.Samples[i-1].Height+1 {
			return fmt.Errorf("target block does not prove insufficient cap and absence")
		}
	}
	if classification == "rejected" && p.Transaction != nil {
		return fmt.Errorf("rejected transaction remains")
	}
	if classification == "admitted" && (p.Transaction == nil || p.Transaction["hash"] != p.Submission.ExpectedHash || p.Transaction["blockNumber"] != nil || p.Transaction["blockHash"] != nil) {
		return fmt.Errorf("not the actual unmined pending hash")
	}
	return nil
}

func TestBelowBaseProofRejectsSnapshotOnlyEvidence(t *testing.T) {
	valid := func() belowBaseProof {
		p := belowBaseProof{Cap: "1", Submission: signedPolicyFixture(t)}
		for i := uint64(1); i <= 5; i++ {
			p.Samples = append(p.Samples, belowBaseSample{Height: i, BaseFee: "100"})
		}
		return p
	}
	if err := verifyBelowBaseProof(valid()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*belowBaseProof)
	}{
		{"missing target blocks", func(p *belowBaseProof) { p.Samples = nil }},
		{"target fee no longer exceeds cap", func(p *belowBaseProof) { p.Samples[4].BaseFee = "1" }},
		{"invalid target fee", func(p *belowBaseProof) { p.Samples[4].BaseFee = "bogus" }},
		{"wrong height sequence", func(p *belowBaseProof) { p.Samples[4].Height = 99 }},
		{"included without receipt", func(p *belowBaseProof) { p.Samples[4].Included = true }},
		{"sample has receipt", func(p *belowBaseProof) { p.Samples[4].Receipt = map[string]any{"status": "0x1"} }},
		{"rejected hash exists", func(p *belowBaseProof) { p.Transaction = map[string]any{"hash": p.Submission.ExpectedHash} }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := valid()
			tc.change(&p)
			if err := verifyBelowBaseProof(p); err == nil {
				t.Fatal("snapshot-only evidence accepted")
			}
		})
	}
	pending := valid()
	pending.Submission.ServerError = nil
	pending.Submission.ReturnedHash = pending.Submission.ExpectedHash
	pending.Transaction = map[string]any{"hash": pending.Submission.ExpectedHash, "blockNumber": nil, "blockHash": nil}
	if err := verifyBelowBaseProof(pending); err != nil {
		t.Fatal(err)
	}
	pending.Transaction["blockNumber"] = "0x1"
	if err := verifyBelowBaseProof(pending); err == nil {
		t.Fatal("mined transaction accepted")
	}
}

func verifyUnfundedPayer(p policySubmission, before, after policyAccountState, transaction, receipt map[string]any) error {
	class, err := classifyPolicySubmission(p)
	if err != nil || class != "rejected" {
		return fmt.Errorf("not an actual node refusal: %v", err)
	}
	if p.ServerError.Code != -32000 || p.ServerError.Message != "fee delegation: insufficient feePayer's funds for gas * price" {
		return fmt.Errorf("not exact fee payer insufficiency")
	}
	if before.PayerBalance != "0" || before != after || transaction != nil || receipt != nil {
		return fmt.Errorf("unfunded rejection state differs")
	}
	return nil
}
func TestUnfundedPayerRejectsGenericNodeErrors(t *testing.T) {
	p := signedPolicyFixture(t)
	p.ServerError.Message = "fee delegation: insufficient feePayer's funds for gas * price"
	state := policyAccountState{PayerBalance: "0"}
	if err := verifyUnfundedPayer(p, state, state, nil, nil); err != nil {
		t.Fatal(err)
	}
	for _, reason := range []string{"insufficient funds", "invalid signature", "unknown transaction type"} {
		t.Run(reason, func(t *testing.T) {
			p := signedPolicyFixture(t)
			p.ServerError.Message = reason
			if err := verifyUnfundedPayer(p, state, state, nil, nil); err == nil {
				t.Fatal("wrong actual refusal accepted")
			}
		})
	}
	p.ServerError.Code = -32602
	if err := verifyUnfundedPayer(p, state, state, nil, nil); err == nil {
		t.Fatal("wrong RPC code accepted")
	}
}
