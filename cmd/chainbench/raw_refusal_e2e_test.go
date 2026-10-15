//go:build e2e

package main

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"testing"

	"golang.org/x/crypto/sha3"
)

type rawRefusalProof struct {
	Raw, Hash, ReturnedHash, Method, Message           string
	Code                                               int
	BeforeHeight, AfterHeight                          uint64
	NonceBefore, NonceAfter                            uint64
	SenderBefore, SenderAfter, PayerBefore, PayerAfter string
	Lookups                                            []rawNullLookup
}

type rawNullLookup struct{ Phase, Method, Hash, Result string }

func rawSubmissionHash(raw []byte) string {
	h := sha3.NewLegacyKeccak256()
	h.Write(raw)
	return "0x" + hex.EncodeToString(h.Sum(nil))
}

func verifyRawRefusal(p rawRefusalProof, reason string) error {
	if p.Code != -32000 || p.Method != "eth_sendRawTransaction" || reason == "" || !strings.Contains(strings.ToLower(p.Message), strings.ToLower(reason)) || p.ReturnedHash != "" {
		return fmt.Errorf("not exact node raw submission refusal")
	}
	if !strings.HasPrefix(p.Raw, "0x") {
		return fmt.Errorf("missing submitted raw bytes")
	}
	raw, err := hex.DecodeString(p.Raw[2:])
	if err != nil || len(raw) == 0 || rawSubmissionHash(raw) != p.Hash {
		return fmt.Errorf("raw hash not bound to submitted bytes")
	}
	if p.BeforeHeight < 20 || p.AfterHeight <= p.BeforeHeight || p.AfterHeight-p.BeforeHeight < 3 || p.NonceBefore != p.NonceAfter {
		return fmt.Errorf("no post-refusal progress or nonce mutated")
	}
	for _, state := range [][2]string{{p.SenderBefore, p.SenderAfter}, {p.PayerBefore, p.PayerAfter}} {
		balance, ok := new(big.Int).SetString(state[0], 10)
		if !ok || balance.Sign() <= 0 || state[0] != state[1] {
			return fmt.Errorf("funded non-validator balance changed")
		}
	}
	if len(p.Lookups) != 4 {
		return fmt.Errorf("missing actual lookup responses")
	}
	for i, row := range p.Lookups {
		phase := "initial"
		if i >= 2 {
			phase = "later"
		}
		method := "eth_getTransactionByHash"
		if i%2 == 1 {
			method = "eth_getTransactionReceipt"
		}
		if row.Phase != phase || row.Method != method || row.Hash != p.Hash || strings.TrimSpace(row.Result) != "null" {
			return fmt.Errorf("lookup not exact raw hash/null response")
		}
	}
	return nil
}

func rawRefusalFixture() rawRefusalProof {
	p := rawRefusalProof{Raw: "0x010203", Hash: rawSubmissionHash([]byte{1, 2, 3}), Method: "eth_sendRawTransaction", Message: "exceeds block gas limit", Code: -32000, BeforeHeight: 22, AfterHeight: 25, SenderBefore: "1000000000000000000", SenderAfter: "1000000000000000000", PayerBefore: "1000000000000000000", PayerAfter: "1000000000000000000"}
	for _, phase := range []string{"initial", "later"} {
		for _, method := range []string{"eth_getTransactionByHash", "eth_getTransactionReceipt"} {
			p.Lookups = append(p.Lookups, rawNullLookup{Phase: phase, Method: method, Hash: p.Hash, Result: "null"})
		}
	}
	return p
}

func TestRawRefusalRejectsUnboundErrorsAndMissingNullLookups(t *testing.T) {
	if err := verifyRawRefusal(rawRefusalFixture(), "exceeds block gas limit"); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*rawRefusalProof)
	}{
		{"transport error", func(p *rawRefusalProof) { p.Code = 0 }},
		{"wrong method", func(p *rawRefusalProof) { p.Method = "eth_estimateGas" }},
		{"protocol error", func(p *rawRefusalProof) { p.Code = -32601 }},
		{"wrong reason", func(p *rawRefusalProof) { p.Message = "insufficient funds" }},
		{"returned hash", func(p *rawRefusalProof) { p.ReturnedHash = p.Hash }},
		{"different raw hash", func(p *rawRefusalProof) { p.Hash = "0x" + strings.Repeat("ff", 32) }},
		{"empty raw", func(p *rawRefusalProof) { p.Raw = "0x" }},
		{"entered pool", func(p *rawRefusalProof) { p.Lookups[0].Result = `{ "hash":"present" }` }},
		{"later entered pool", func(p *rawRefusalProof) { p.Lookups[2].Result = `{ "hash":"present" }` }},
		{"mined receipt", func(p *rawRefusalProof) { p.Lookups[1].Result = `{ "transactionHash":"present" }` }},
		{"later mined receipt", func(p *rawRefusalProof) { p.Lookups[3].Result = `{ "transactionHash":"present" }` }},
		{"missing lookup", func(p *rawRefusalProof) { p.Lookups = p.Lookups[:3] }},
		{"lookup wrong hash", func(p *rawRefusalProof) { p.Lookups[0].Hash = "unrelated" }},
		{"lookup wrong method", func(p *rawRefusalProof) { p.Lookups[0].Method = "eth_getBlockByNumber" }},
		{"lookup wrong phase", func(p *rawRefusalProof) { p.Lookups[2].Phase = "initial" }},
		{"no block progress", func(p *rawRefusalProof) { p.AfterHeight = p.BeforeHeight }},
		{"nonce consumed", func(p *rawRefusalProof) { p.NonceAfter = 1 }},
		{"sender debited", func(p *rawRefusalProof) { p.SenderAfter = "999999999999999999" }},
		{"payer debited", func(p *rawRefusalProof) { p.PayerAfter = "999999999999999999" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := rawRefusalFixture()
			tc.change(&p)
			if verifyRawRefusal(p, "exceeds block gas limit") == nil {
				t.Fatal("an arbitrary refusal hid accepted raw bytes or state mutation")
			}
		})
	}
}
