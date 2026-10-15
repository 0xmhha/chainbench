//go:build e2e

package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

func TestWemixNCPRemovalAndSelfExitE2E(t *testing.T) {
	from, to := os.Getenv("CHAINBENCH_E2E_FROM_BIN"), os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if from == "" || to == "" {
		t.Skip("set native handoff binaries")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 7*time.Minute)
	defer cancel()
	keys := make([][]byte, 7)
	operators := make([]string, 7)
	alloc := map[string]any{}
	for i := range keys {
		var err error
		keys[i], operators[i], err = accounts.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}
		operators[i] = strings.ToLower(operators[i])
		alloc[operators[i]] = map[string]any{"balance": handoffBalance}
	}
	overlay := map[string]any{"alloc": alloc, "config": map[string]any{"croissant": map[string]any{"wBFT": map[string]any{"useNCP": true, "targetValidators": 4, "stabilizingStakersThreshold": 4}, "govContracts": map[string]any{"govNCP": map[string]any{"params": map[string]any{"ncps": strings.Join(operators, ",")}}}}}}
	workspace, url := runGovHandoffWorkspace(t, from, to, writeHandoffCase(t, overlay))
	root := filepath.Dir(workspace)
	c := rpc.Dial(url)
	observed := &observedRawRPC{client: c}
	provider, err := accounts.ForChainWithRPC("wbft", func(string) accounts.RPCRequester { return observed })
	if err != nil {
		t.Fatal(err)
	}
	wallets := make([]accounts.Wallet, 7)
	for i := range wallets {
		wallets[i], err = provider.OpenWallet(ctx, keys[i], url)
		if err != nil {
			t.Fatal(err)
		}
	}
	changes := []ncpRemovalProof{}
	proof := map[string]any{"workspace": workspace, "operators": operators, "scope": "GOV007/008 local WEMIX-to-WBFT Croissant20, UseNCPtrue, NCP7 independent operators and four managed initial seed stakers registered at original minimum1e25. Target4/threshold4 exits stabilization before council changes. Other-member removal needs four distinct votes of seven; self-exit begins with six members>=3 and is one no-vote transaction. Exact proposal/removal/finalized events, pinned full staking info/activity/operator mapping preserved, then first following epoch excludes only removed operator staker and live validators progress4->3->2. Not seven-node consensus, public JSON whole execution/ranking/default-threshold5/Linux/mainnet/full87/legacy/final Web."}
	defer func() {
		proof["changes"], proof["submittedRaws"] = changes, observed.submitted()
		b, e := json.MarshalIndent(proof, "", "  ")
		if e != nil {
			t.Error(e)
			return
		}
		if e = os.WriteFile(filepath.Join(root, "reports/ncp-removal-proof.json"), b, 0600); e != nil {
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
	stakeAt := func(sig, tag string, args ...accounts.Arg) string {
		var out string
		if err := c.Call(ctx, "eth_call", &out, map[string]any{"to": e2eGovStaking, "data": accounts.EncodeCallArgs(sig, args...)}, tag); err != nil {
			t.Fatal(err)
		}
		return out
	}
	live := []epochBoundaryMember{}
	registrations := []map[string]any{}
	amount := govConfigUint(t, c, "minimumStaking()")
	if amount.String() != governanceForkFixture().Configured[0] {
		t.Fatal("wrong actual minimum", amount)
	}
	for i := 0; i < 4; i++ {
		staker := presetNodeAddr(t, i+1)
		pk, pop := presetNodeBLS(t, i+1)
		hash := stakingRegister(t, c, wallets[i], staker, pk, pop, amount)
		rc := policyReceipt(t, c, hash)
		registrations = append(registrations, map[string]any{"staker": staker, "operator": operators[i], "receipt": rc})
		proof["registrations"] = registrations
		live = append(live, epochBoundaryMember{Address: staker, BLS: "0x" + hex.EncodeToString(pk)})
	}
	last, ok := lifecycleQuantity(registrations[3]["receipt"].(map[string]any)["blockNumber"])
	if !ok {
		t.Fatal("missing registration height")
	}
	boundary := (last.Uint64()/10 + 1) * 10
	waitForHead(t, c, boundary+1, 90*time.Second)
	var initialExtra map[string]any
	if err := c.Call(ctx, "istanbul_getWbftExtraInfo", &initialExtra, hexUint(boundary)); err != nil {
		t.Fatal(err)
	}
	proof["initialDecidedEpoch"] = map[string]any{"height": boundary, "extra": initialExtra}
	ep, ok := initialExtra["epochInfo"].(map[string]any)
	if !ok || ep["stabilizing"] != false {
		t.Fatal("registered set not stabilized")
	}
	if vs, ok := ep["validators"].([]any); !ok || len(vs) != 4 {
		t.Fatal("not four initial registered validators")
	}
	council := append([]string(nil), operators...)
	for _, self := range []bool{false, true} {
		target := 3
		if self {
			target = 2
		}
		p := ncpRemovalProof{InitialCouncil: append([]string(nil), operators...), BeforeCouncil: append([]string(nil), council...), Operator: operators[target], Staker: presetNodeAddr(t, target+1), Self: self}
		if !self {
			p.Voters = []string{operators[0], operators[1], operators[2], operators[4]}
		}
		head, err := c.BlockNumber(ctx)
		if err != nil {
			t.Fatal(err)
		}
		id, valid := lifecycleQuantity(readAt("currentProposalID()", hexUint(head)))
		if !valid || !id.IsUint64() {
			t.Fatal("invalid proposal ID")
		}
		p.ID = id.Uint64()
		changes = append(changes, p)
		change := &changes[len(changes)-1]
		for i := 0; i <= len(change.Voters); i++ {
			wallet := wallets[0]
			input := accounts.EncodeCallArgs("newProposalToRemoveNCP(address)", accounts.Address(change.Operator))
			if self {
				wallet = wallets[target]
			} else if i > 0 {
				for j, a := range operators {
					if a == change.Voters[i-1] {
						wallet = wallets[j]
					}
				}
				input = accounts.EncodeCallArgs("vote(uint256,bool)", accounts.Uint(new(big.Int).SetUint64(change.ID)), accounts.Word([]byte{1}))
			}
			data, e := hex.DecodeString(input[2:])
			if e != nil {
				t.Fatal(e)
			}
			hash, e := wallet.Execute(ctx, e2eGovNCP, data, nil)
			if e != nil {
				t.Fatal(e)
			}
			raws := observed.submitted()
			if len(raws) == 0 {
				t.Fatal("missing submitted raw")
			}
			change.Stages = append(change.Stages, ncpBallotStage{Raw: raws[len(raws)-1]})
			s := &change.Stages[i]
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
			s.CandidateMember = readAt("isNCP(address)", tag, accounts.Address(change.Operator))
			s.Proposal = readAt("proposals(uint256)", tag, accounts.Uint(new(big.Int).SetUint64(change.ID)))
			s.CurrentID = readAt("currentProposalID()", tag)
			if self || i == len(change.Voters) {
				h, valid := lifecycleQuantity(tag)
				if !valid || h.Uint64() == 0 {
					t.Fatal("missing final removal height")
				}
				parent := hexUint(h.Uint64() - 1)
				change.BeforeStake = stakeAt("stakerInfo(address)", parent, accounts.Address(change.Staker))
				change.AfterStake = stakeAt("stakerInfo(address)", tag, accounts.Address(change.Staker))
				change.BeforeActive = stakeAt("isStaker(address)", parent, accounts.Address(change.Staker))
				change.AfterActive = stakeAt("isStaker(address)", tag, accounts.Address(change.Staker))
				change.BeforeMapping = stakeAt("stakerByOperator(address)", parent, accounts.Address(change.Operator))
				change.AfterMapping = stakeAt("stakerByOperator(address)", tag, accounts.Address(change.Operator))
				change.Epoch.Boundary = (h.Uint64()/10 + 1) * 10
			}
			t.Logf("NCP removal self%v stage%d mined%s count%s", self, i, tag, s.Count)
		}
		for _, m := range live {
			if !strings.EqualFold(m.Address, change.Staker) {
				change.Epoch.Expected = append(change.Epoch.Expected, m)
			}
		}
		e := &change.Epoch
		e.BeforeTag = hexUint(e.Boundary - 1)
		e.AfterTag = hexUint(e.Boundary + 1)
		waitForHead(t, c, e.Boundary+2, 90*time.Second)
		for i := 0; i < 3; i++ {
			if err := c.Call(ctx, "eth_getBlockByNumber", &e.Blocks[i], hexUint(e.Boundary-1+uint64(i)), false); err != nil {
				t.Fatal(err)
			}
		}
		for i := 0; i < 2; i++ {
			if err := c.Call(ctx, "istanbul_getWbftExtraInfo", &e.Extras[i], hexUint(e.Boundary+uint64(i))); err != nil {
				t.Fatal(err)
			}
		}
		if err := c.Call(ctx, "istanbul_getValidators", &e.Before, e.BeforeTag); err != nil {
			t.Fatal(err)
		}
		if err := c.Call(ctx, "istanbul_getValidators", &e.After, e.AfterTag); err != nil {
			t.Fatal(err)
		}
		if err := verifyNCPRemoval(*change); err != nil {
			t.Fatal(fmt.Sprintf("self%v", self), err)
		}
		council = ncpWithout(council, change.Operator)
		live = append([]epochBoundaryMember(nil), e.Expected...)
	}
}
