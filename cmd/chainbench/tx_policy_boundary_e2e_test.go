//go:build e2e

package main

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	sdkcrypto "github.com/0xmhha/accounts/crypto"
	sdktx "github.com/0xmhha/accounts/tx"
	sdktypes "github.com/0xmhha/accounts/types"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

type policySubmission struct {
	Method, Raw, ExpectedHash, ReturnedHash, OtherError string
	ServerError                                         *rpc.ServerError
}

func classifyPolicySubmission(p policySubmission) (string, error) {
	if p.Method != "eth_sendRawTransaction" || p.OtherError != "" || !strings.HasPrefix(p.Raw, "0x") {
		return "", fmt.Errorf("not an answered raw node submission")
	}
	raw, err := hex.DecodeString(p.Raw[2:])
	if err != nil || len(raw) < 65 || (raw[0] != 0x02 && raw[0] != 0x16) || p.ExpectedHash != fmt.Sprintf("0x%x", sdkcrypto.Keccak256(raw)) {
		return "", fmt.Errorf("missing or mismatched signed envelope/hash")
	}
	if p.ServerError != nil {
		if p.ReturnedHash != "" || p.ServerError.Method != p.Method || p.ServerError.Code == 0 || p.ServerError.Message == "" {
			return "", fmt.Errorf("not an unambiguous typed node refusal")
		}
		return "rejected", nil
	}
	if p.ReturnedHash != p.ExpectedHash {
		return "", fmt.Errorf("missing or mismatched admission hash")
	}
	return "admitted", nil
}

func signedPolicyFixture(t *testing.T) policySubmission {
	t.Helper()
	key, err := sdkcrypto.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	to, err := sdktypes.HexToAddress("0x0000000000000000000000000000000000001234")
	if err != nil {
		t.Fatal(err)
	}
	tx := sdktx.DynamicFeeTx{ChainID: big.NewInt(8285), Gas: 21000, GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(1000000000), Value: big.NewInt(1), To: &to}
	if err := tx.Sign(key); err != nil {
		t.Fatal(err)
	}
	raw := tx.Encode()
	return policySubmission{Method: "eth_sendRawTransaction", Raw: "0x" + hex.EncodeToString(raw), ExpectedHash: fmt.Sprintf("0x%x", sdkcrypto.Keccak256(raw)), ServerError: &rpc.ServerError{Method: "eth_sendRawTransaction", Code: -32000, Message: "actual node reason"}}
}

func TestPolicySubmissionRejectsFalseNodeEvidence(t *testing.T) {
	p := signedPolicyFixture(t)
	if got, err := classifyPolicySubmission(p); err != nil || got != "rejected" {
		t.Fatal(got, err)
	}
	admitted := p
	admitted.ServerError = nil
	admitted.ReturnedHash = admitted.ExpectedHash
	if got, err := classifyPolicySubmission(admitted); err != nil || got != "admitted" {
		t.Errorf("actual hash must remain admission: %s %v", got, err)
	}
	for _, tc := range []struct {
		name   string
		change func(*policySubmission)
	}{
		{"preparation error", func(p *policySubmission) { p.ServerError = nil; p.OtherError = "invalid key" }},
		{"estimate refusal", func(p *policySubmission) { p.ServerError.Method = "eth_estimateGas" }},
		{"HTTP failure", func(p *policySubmission) { p.ServerError = nil; p.OtherError = "HTTP 503" }},
		{"wrong submitted hash", func(p *policySubmission) { p.ExpectedHash = "other" }},
		{"missing raw request", func(p *policySubmission) { p.Raw = "" }},
		{"wrong envelope type", func(p *policySubmission) { p.Raw = "0xff" + p.Raw[4:] }},
		{"error and hash", func(p *policySubmission) { p.ReturnedHash = p.ExpectedHash }},
		{"missing error reason", func(p *policySubmission) { p.ServerError.Message = "" }},
		{"unanswered request", func(p *policySubmission) { p.ServerError = nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := signedPolicyFixture(t)
			tc.change(&p)
			if _, err := classifyPolicySubmission(p); err == nil {
				t.Fatal("invalid submission evidence accepted")
			}
		})
	}
}

type policyAccountState struct {
	SenderBalance, SenderNonce, PayerBalance, PayerNonce, RecipientBalance string
}

func policyState(t *testing.T, c *rpc.Client, sender, payer, recipient string) policyAccountState {
	t.Helper()
	ctx := t.Context()
	p := policyAccountState{}
	for address, dest := range map[string]*string{sender: &p.SenderBalance, payer: &p.PayerBalance, recipient: &p.RecipientBalance} {
		n, err := c.BalanceAt(ctx, address)
		if err != nil {
			t.Fatal(err)
		}
		*dest = n.String()
	}
	for address, dest := range map[string]*string{sender: &p.SenderNonce, payer: &p.PayerNonce} {
		if err := c.Call(ctx, "eth_getTransactionCount", dest, address, "latest"); err != nil {
			t.Fatal(err)
		}
	}
	return p
}

func policySubmit(t *testing.T, c *rpc.Client, raw []byte) policySubmission {
	t.Helper()
	p := policySubmission{Method: "eth_sendRawTransaction", Raw: "0x" + hex.EncodeToString(raw), ExpectedHash: fmt.Sprintf("0x%x", sdkcrypto.Keccak256(raw))}
	err := c.Call(t.Context(), p.Method, &p.ReturnedHash, p.Raw)
	if err != nil && !errors.As(err, &p.ServerError) {
		p.OtherError = err.Error()
	}
	if _, err := classifyPolicySubmission(p); err != nil {
		logGovernanceProof(t, "policy-invalid-submission", p)
		t.Fatal(err)
	}
	return p
}

func policyReceipt(t *testing.T, c *rpc.Client, hash string) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	for {
		var receipt map[string]any
		if err := c.Call(ctx, "eth_getTransactionReceipt", &receipt, hash); err != nil {
			t.Fatal(err)
		}
		if receipt != nil {
			if receipt["transactionHash"] != hash || receipt["status"] != "0x1" || receipt["blockNumber"] == nil {
				t.Fatal("positive control did not mine successfully", receipt)
			}
			return receipt
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(200 * time.Millisecond):
		}
	}
}

func policyControlReceipt(t *testing.T, c *rpc.Client, p policySubmission) map[string]any {
	t.Helper()
	classification, err := classifyPolicySubmission(p)
	if err != nil || classification != "admitted" {
		t.Fatal("positive control was not admitted", p, err)
	}
	return policyReceipt(t, c, p.ExpectedHash)
}

func policyDynamic(t *testing.T, c *rpc.Client, key *sdkcrypto.PrivateKey, to sdktypes.Address, chainID, cap, tip, value *big.Int) sdktx.DynamicFeeTx {
	t.Helper()
	var nonceHex string
	if err := c.Call(t.Context(), "eth_getTransactionCount", &nonceHex, sdkcrypto.PrivKeyToAddress(key).Hex(), "latest"); err != nil {
		t.Fatal(err)
	}
	nonce, err := strconv.ParseUint(strings.TrimPrefix(nonceHex, "0x"), 16, 64)
	if err != nil {
		t.Fatal(err)
	}
	tx := sdktx.DynamicFeeTx{ChainID: chainID, Nonce: nonce, GasTipCap: tip, GasFeeCap: cap, Gas: 21000, To: &to, Value: value}
	if err := tx.Sign(key); err != nil {
		t.Fatal(err)
	}
	return tx
}

// TestWemixTransactionPolicyBoundaryE2E separates snapshot and target-block fees
// and verifies the user-approved current-node fee-payer rejection after handoff.
func TestWemixTransactionPolicyBoundaryE2E(t *testing.T) {
	fromBin, toBin := os.Getenv("CHAINBENCH_E2E_FROM_BIN"), os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if fromBin == "" || toBin == "" {
		t.Skip("policy diagnostic requires WEMIX and WBFT binaries")
	}
	actors := make([]*sdkcrypto.PrivateKey, 4)
	addresses := make([]sdktypes.Address, 4)
	for i := range actors {
		key, err := sdkcrypto.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}
		actors[i] = key
		addresses[i] = sdkcrypto.PrivKeyToAddress(key)
	}
	alloc := map[string]any{}
	for _, address := range addresses[:2] {
		alloc[address.Hex()] = map[string]any{"balance": "0x56bc75e2d63100000"}
	}
	url := runGovHandoffOverlay(t, fromBin, toBin, map[string]any{"alloc": alloc})
	c := rpc.Dial(url)
	chainNumber, err := c.ChainID(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	chainID := new(big.Int).SetUint64(chainNumber)
	latest, err := c.BlockByNumber(t.Context(), "latest")
	if err != nil || latest.BaseFeePerGas == nil || latest.BaseFeePerGas.Cmp(big.NewInt(1)) <= 0 {
		t.Fatal("missing positive postfork baseFee", err)
	}
	head, err := c.BlockNumber(t.Context())
	if err != nil || head < 20 {
		t.Fatal("not a post-Croissant head", head, err)
	}
	cap := new(big.Int).Sub(latest.BaseFeePerGas, big.NewInt(1))
	var suggestedTipHex string
	if err := c.Call(t.Context(), "eth_maxPriorityFeePerGas", &suggestedTipHex); err != nil {
		t.Fatal(err)
	}
	lowTip, ok := new(big.Int).SetString(strings.TrimPrefix(suggestedTipHex, "0x"), 16)
	if !ok || lowTip.Sign() < 0 {
		t.Fatal("invalid suggested tip", suggestedTipHex)
	}
	if lowTip.Sign() == 0 {
		lowTip.SetInt64(1)
	}
	if lowTip.Cmp(cap) > 0 {
		lowTip.Set(cap)
	}
	tx := policyDynamic(t, c, actors[0], addresses[3], chainID, cap, lowTip, big.NewInt(1))
	before := policyState(t, c, addresses[0].Hex(), addresses[2].Hex(), addresses[3].Hex())
	submission := policySubmit(t, c, tx.Encode())
	classification, _ := classifyPolicySubmission(submission)
	samples := []map[string]any{}
	for i := uint64(1); i <= 5; i++ {
		waitForHead(t, c, head+i, 30*time.Second)
		block, err := c.BlockByNumber(t.Context(), fmt.Sprintf("0x%x", head+i))
		if err != nil || block.BaseFeePerGas == nil {
			t.Fatal(err)
		}
		var receipt map[string]any
		if err := c.Call(t.Context(), "eth_getTransactionReceipt", &receipt, submission.ExpectedHash); err != nil {
			t.Fatal(err)
		}
		samples = append(samples, map[string]any{"height": head + i, "baseFee": block.BaseFeePerGas.String(), "receipt": receipt})
	}
	var actualTx, actualReceipt map[string]any
	if err := c.Call(t.Context(), "eth_getTransactionByHash", &actualTx, submission.ExpectedHash); err != nil {
		t.Fatal(err)
	}
	if err := c.Call(t.Context(), "eth_getTransactionReceipt", &actualReceipt, submission.ExpectedHash); err != nil {
		t.Fatal(err)
	}
	after := policyState(t, c, addresses[0].Hex(), addresses[2].Hex(), addresses[3].Hex())
	if classification == "rejected" && (actualTx != nil || actualReceipt != nil || before != after) {
		t.Fatal("rejected low-cap transaction had an effect")
	}
	if classification == "admitted" && actualReceipt == nil && (actualTx == nil || actualTx["hash"] != submission.ExpectedHash || before != after) {
		t.Fatal("admitted pending low-cap observation mismatched")
	}
	low := map[string]any{"criterion": "TX-002", "input": tx, "initialBlock": head, "baseFee": latest.BaseFeePerGas.String(), "submission": submission, "classification": classification, "samples": samples, "transaction": actualTx, "receipt": actualReceipt, "before": before, "after": after, "originalExpectationObserved": classification == "rejected"}
	var tipHex, priceHex string
	if err := c.Call(t.Context(), "eth_maxPriorityFeePerGas", &tipHex); err != nil {
		t.Fatal(err)
	}
	if err := c.Call(t.Context(), "eth_gasPrice", &priceHex); err != nil {
		t.Fatal(err)
	}
	tip, ok := new(big.Int).SetString(strings.TrimPrefix(tipHex, "0x"), 16)
	if !ok {
		t.Fatal(tipHex)
	}
	price, ok := new(big.Int).SetString(strings.TrimPrefix(priceHex, "0x"), 16)
	if !ok {
		t.Fatal(priceHex)
	}
	controlTip := new(big.Int).Set(lowTip)
	if classification == "admitted" {
		controlTip.Mul(controlTip, big.NewInt(2))
	}
	highCap := new(big.Int).Add(new(big.Int).Mul(price, big.NewInt(3)), controlTip)
	replacementFloor := new(big.Int).Mul(cap, big.NewInt(2))
	if highCap.Cmp(replacementFloor) < 0 {
		highCap.Set(replacementFloor)
	}
	control := policyDynamic(t, c, actors[0], addresses[3], chainID, highCap, controlTip, big.NewInt(1))
	positive := policySubmit(t, c, control.Encode())
	low["positiveControl"] = map[string]any{"submission": positive, "receipt": policyControlReceipt(t, c, positive)}
	logGovernanceProof(t, "tx002-policy", low)

	// A separate low cap must remain below every actual target block, rather than
	// assuming the earlier latest quote equals a later block's base fee.
	bounded := policyDynamic(t, c, actors[0], addresses[3], chainID, big.NewInt(1), big.NewInt(1), big.NewInt(1))
	boundedBefore := policyState(t, c, addresses[0].Hex(), addresses[2].Hex(), addresses[3].Hex())
	boundedHead, err := c.BlockNumber(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	boundedSubmit := policySubmit(t, c, bounded.Encode())
	boundedProof := belowBaseProof{Cap: "1", Submission: boundedSubmit, Before: boundedBefore}
	for i := uint64(1); i <= 5; i++ {
		waitForHead(t, c, boundedHead+i, 30*time.Second)
		tag := fmt.Sprintf("0x%x", boundedHead+i)
		block, err := c.BlockByNumber(t.Context(), tag)
		if err != nil || block.BaseFeePerGas == nil {
			t.Fatal("target block fee missing", err)
		}
		var body struct{ Transactions []string }
		if err = c.Call(t.Context(), "eth_getBlockByNumber", &body, tag, false); err != nil {
			t.Fatal(err)
		}
		sample := belowBaseSample{Height: boundedHead + i, BaseFee: block.BaseFeePerGas.String()}
		for _, hash := range body.Transactions {
			if hash == boundedSubmit.ExpectedHash {
				sample.Included = true
			}
		}
		if err = c.Call(t.Context(), "eth_getTransactionReceipt", &sample.Receipt, boundedSubmit.ExpectedHash); err != nil {
			t.Fatal(err)
		}
		boundedProof.Samples = append(boundedProof.Samples, sample)
	}
	if err = c.Call(t.Context(), "eth_getTransactionByHash", &boundedProof.Transaction, boundedSubmit.ExpectedHash); err != nil {
		t.Fatal(err)
	}
	if err = c.Call(t.Context(), "eth_getTransactionReceipt", &boundedProof.Receipt, boundedSubmit.ExpectedHash); err != nil {
		t.Fatal(err)
	}
	boundedProof.After = policyState(t, c, addresses[0].Hex(), addresses[2].Hex(), addresses[3].Hex())
	logGovernanceProof(t, "tx002-target-basefee", boundedProof)
	if err = verifyBelowBaseProof(boundedProof); err != nil {
		t.Fatal(err)
	}

	// Same valid dual-signed bytes are resubmitted after funding only the fee payer.
	inner := policyDynamic(t, c, actors[1], addresses[3], chainID, highCap, tip, big.NewInt(1))
	delegated := sdktx.FeeDelegateTx{Sender: inner}
	if err := delegated.Sign(actors[1], actors[2]); err != nil {
		t.Fatal(err)
	}
	sender, err := delegated.SenderAddress()
	if err != nil || sender != addresses[1] {
		t.Fatal("invalid sender signature", err)
	}
	payer, err := delegated.RecoverFeePayer()
	if err != nil || payer != addresses[2] {
		t.Fatal("invalid fee payer signature", err)
	}
	raw, err := delegated.Encode()
	if err != nil {
		t.Fatal(err)
	}
	before = policyState(t, c, addresses[1].Hex(), addresses[2].Hex(), addresses[3].Hex())
	if before.PayerBalance != "0" {
		t.Fatal("fee payer not unfunded", before)
	}
	refusal := policySubmit(t, c, raw)
	classification, _ = classifyPolicySubmission(refusal)
	head, err = c.BlockNumber(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	waitForHead(t, c, head+5, 30*time.Second)
	actualTx = nil
	actualReceipt = nil
	if err := c.Call(t.Context(), "eth_getTransactionByHash", &actualTx, refusal.ExpectedHash); err != nil {
		t.Fatal(err)
	}
	if err := c.Call(t.Context(), "eth_getTransactionReceipt", &actualReceipt, refusal.ExpectedHash); err != nil {
		t.Fatal(err)
	}
	after = policyState(t, c, addresses[1].Hex(), addresses[2].Hex(), addresses[3].Hex())
	if before != after || actualReceipt != nil {
		logGovernanceProof(t, "tx016-unexpected-effect", map[string]any{"submission": refusal, "receipt": actualReceipt, "before": before, "after": after})
		t.Fatal("unfunded fee-payer diagnostic needs reassessment")
	}
	if classification == "rejected" && actualTx != nil {
		t.Fatal("rejected dual signed hash remains")
	}
	if err = verifyUnfundedPayer(refusal, before, after, actualTx, actualReceipt); err != nil {
		t.Fatal(err)
	}
	original := actualReceipt != nil && actualReceipt["status"] == "0x0"
	fund := policyDynamic(t, c, actors[0], addresses[2], chainID, highCap, tip, big.NewInt(1000000000000000000))
	funded := policySubmit(t, c, fund.Encode())
	fundReceipt := policyControlReceipt(t, c, funded)
	// Funding may immediately make an admitted pending transaction mine. Derive
	// the before-state from the observed rejection state plus this exact funding
	// value, rather than racing a post-funding RPC read against that execution.
	controlBefore := after
	fundedBalance, ok := new(big.Int).SetString(after.PayerBalance, 10)
	if !ok {
		t.Fatal("invalid observed fee-payer balance")
	}
	controlBefore.PayerBalance = new(big.Int).Add(fundedBalance, big.NewInt(1000000000000000000)).String()
	resubmission := refusal
	if classification == "rejected" {
		resubmission = policySubmit(t, c, raw)
	}
	receipt := policyControlReceipt(t, c, resubmission)
	if receipt["type"] != "0x16" {
		t.Fatal("control is not fee delegated", receipt)
	}
	controlAfter := policyState(t, c, addresses[1].Hex(), addresses[2].Hex(), addresses[3].Hex())
	used, ok := new(big.Int).SetString(strings.TrimPrefix(fmt.Sprint(receipt["gasUsed"]), "0x"), 16)
	if !ok {
		t.Fatal("missing control gas used")
	}
	actualPrice, ok := new(big.Int).SetString(strings.TrimPrefix(fmt.Sprint(receipt["effectiveGasPrice"]), "0x"), 16)
	if !ok {
		t.Fatal("missing control gas price")
	}
	fee := new(big.Int).Mul(used, actualPrice)
	senderBefore, _ := new(big.Int).SetString(controlBefore.SenderBalance, 10)
	payerBefore, _ := new(big.Int).SetString(controlBefore.PayerBalance, 10)
	recipientBefore, _ := new(big.Int).SetString(controlBefore.RecipientBalance, 10)
	if controlAfter.SenderBalance != new(big.Int).Sub(senderBefore, big.NewInt(1)).String() || controlAfter.PayerBalance != new(big.Int).Sub(payerBefore, fee).String() || controlAfter.RecipientBalance != new(big.Int).Add(recipientBefore, big.NewInt(1)).String() || controlAfter.SenderNonce != "0x1" || controlAfter.PayerNonce != controlBefore.PayerNonce {
		t.Fatal("same-bytes positive control did not separate sender value and payer gas", controlBefore, controlAfter)
	}
	logGovernanceProof(t, "tx016-policy", map[string]any{"criterion": "TX-016", "input": delegated, "senderSignatureValid": true, "payerSignatureValid": true, "submission": refusal, "classification": classification, "initialBlock": head, "observedThroughBlock": head + 5, "transaction": actualTx, "receipt": actualReceipt, "before": before, "after": after, "originalExpectationObserved": original, "fundingControl": map[string]any{"submission": funded, "receipt": fundReceipt}, "sameBytesControl": map[string]any{"submission": resubmission, "receipt": receipt, "before": controlBefore, "after": controlAfter, "exactFee": fee.String()}, "userApprovedCurrentNodeCriteria": true})
}
