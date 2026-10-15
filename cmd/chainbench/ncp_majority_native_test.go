//go:build e2e

package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

func TestWemixNCPSevenMemberMajorityE2E(t *testing.T) {
	from, to := os.Getenv("CHAINBENCH_E2E_FROM_BIN"), os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if from == "" || to == "" {
		t.Skip("set native handoff binaries")
	}
	p := ncpMajorityProof{}
	keys := make([][]byte, 7)
	alloc := map[string]any{}
	for i := range keys {
		var addr string
		var err error
		keys[i], addr, err = accounts.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}
		p.Members = append(p.Members, strings.ToLower(addr))
		alloc[addr] = map[string]any{"balance": handoffBalance}
	}
	_, candidate, err := accounts.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	p.Candidate, p.Proposer = strings.ToLower(candidate), p.Members[0]
	overlay := map[string]any{"alloc": alloc, "config": map[string]any{"croissant": map[string]any{"wBFT": map[string]any{"useNCP": true}, "govContracts": map[string]any{"govNCP": map[string]any{"params": map[string]any{"ncps": strings.Join(p.Members, ",")}}}}}}
	workspace, url := runGovHandoffWorkspace(t, from, to, writeHandoffCase(t, overlay))
	root := filepath.Dir(workspace)
	c := rpc.Dial(url)
	ctx := t.Context()
	observed := &observedRawRPC{client: c}
	proof := map[string]any{"workspace": workspace, "scope": "GOV006 actual seven distinct NCP operators voting after local WEMIX-to-WBFT Croissant20. Standard four managed WBFT successors; NCP electorate is independently funded operators, no staker registration or seven-node consensus claim. Exact proposal type1, canonical raw/transaction/receipt/block binding, per-vote acceptance state and exact events, unchanged7/candidatefalse through votes1..3, immediate fourth-vote8/candidatetrue. Not removal/self-exit/rank/stabilization/public JSON entire execution/mainnet/Linux/full87/legacy/final Web."}
	defer func() {
		proof["witness"], proof["submittedRaws"] = p, observed.submitted()
		b, e := json.MarshalIndent(proof, "", "  ")
		if e != nil {
			t.Error(e)
			return
		}
		if e = os.WriteFile(filepath.Join(root, "reports/ncp-majority-proof.json"), b, 0600); e != nil {
			t.Error(e)
		}
	}()
	readAt := func(sig, tag string, args ...accounts.Arg) string {
		var out string
		if err := c.Call(ctx, "eth_call", &out, map[string]any{"to": e2eGovNCP, "data": accounts.EncodeCallArgs(sig, args...)}, tag); err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(out, "0x") || len(out) < 66 {
			t.Fatal("short pinned NCP state", sig, out)
		}
		return out
	}
	head, err := c.BlockNumber(ctx)
	if err != nil || head <= 20 {
		t.Fatal("not postfork", head, err)
	}
	tag := hexUint(head)
	id, ok := lifecycleQuantity(readAt("currentProposalID()", tag))
	if !ok || !id.IsUint64() {
		t.Fatal("invalid proposal ID")
	}
	p.ID = id.Uint64()
	initial := map[string]any{"tag": tag, "count": readAt("ncpCount()", tag), "list": readAt("ncpList()", tag), "candidateMember": readAt("isNCP(address)", tag, accounts.Address(p.Candidate)), "currentID": ncpBallotWord(p.ID)}
	proof["initialState"] = initial
	if initial["count"] != ncpBallotWord(7) || !strings.EqualFold(initial["list"].(string), ncpBallotList(p.Members)) || initial["candidateMember"] != ncpBallotWord(0) {
		t.Fatal("actual initial electorate differs", initial)
	}
	provider, err := accounts.ForChainWithRPC("wbft", func(string) accounts.RPCRequester { return observed })
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i <= 4; i++ {
		fromIndex := 0
		input := accounts.EncodeCallArgs("newProposalToAddNCP(address)", accounts.Address(p.Candidate))
		if i > 0 {
			fromIndex = i - 1
			input = accounts.EncodeCallArgs("vote(uint256,bool)", accounts.Uint(new(big.Int).SetUint64(p.ID)), accounts.Word([]byte{1}))
		}
		wallet, e := provider.OpenWallet(ctx, keys[fromIndex], url)
		if e != nil {
			t.Fatal(e)
		}
		calldata, e := hex.DecodeString(input[2:])
		if e != nil {
			t.Fatal(e)
		}
		hash, e := wallet.Execute(context.Background(), e2eGovNCP, calldata, nil)
		if e != nil {
			t.Fatal(e)
		}
		raws := observed.submitted()
		if len(raws) != i+1 {
			t.Fatal("unexpected raw submission count", len(raws))
		}
		p.Stages = append(p.Stages, ncpBallotStage{Raw: raws[i]})
		s := &p.Stages[i]
		s.Receipt = policyReceipt(t, c, hash)
		if err := c.Call(ctx, "eth_getTransactionByHash", &s.Transaction, hash); err != nil {
			t.Fatal(err)
		}
		tag := s.Receipt["blockNumber"].(string)
		if err := c.Call(ctx, "eth_getBlockByNumber", &s.Block, tag, false); err != nil {
			t.Fatal(err)
		}
		b, e := json.Marshal(s.Receipt["logs"])
		if e != nil {
			t.Fatal(e)
		}
		if e = json.Unmarshal(b, &s.Logs); e != nil {
			t.Fatal(e)
		}
		s.Count = readAt("ncpCount()", tag)
		s.List = readAt("ncpList()", tag)
		s.CandidateMember = readAt("isNCP(address)", tag, accounts.Address(p.Candidate))
		s.Proposal = readAt("proposals(uint256)", tag, accounts.Uint(new(big.Int).SetUint64(p.ID)))
		s.CurrentID = readAt("currentProposalID()", tag)
		t.Logf("NCP seven-member stage%d mined%s, count%s, candidate%s", i, tag, s.Count, s.CandidateMember)
	}
	if err := verifyNCPMajority(p); err != nil {
		t.Fatal(err)
	}
}
