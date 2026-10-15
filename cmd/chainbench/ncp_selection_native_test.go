//go:build e2e

package main

import (
	"bytes"
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
	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

type selectionRing struct {
	Nodes []struct {
		Index                         int
		Address, BLSPublicKey, BLSPoP string
	}
}

func selectionHandoff(t *testing.T, from, to string, operators []string) (string, string, selectionRing) {
	t.Helper()
	t.Setenv("GWEMIX_BIN", from)
	t.Setenv("GWBFT_BIN", to)
	var last string
	for attempt := 1; attempt <= govHandoffAttempts; attempt++ {
		root := createHandoffRuntime(t, writeHandoffCase(t, nil))
		keys := filepath.Join(root, "keyring")
		validators := 8
		if _, err := app.KeyringNew(t.Context(), app.Deps{}, app.RingCreateIn{Ring: app.RingRef{Dir: keys}, Count: 9, Validators: &validators, WithBLS: true, Password: "1"}); err != nil {
			t.Fatal(err)
		}
		metadata := readPostforkPublicJSON[selectionRing](t, filepath.Join(keys, "metadata.json"))
		if len(metadata.Nodes) != 9 {
			t.Fatal("not nine generated identities")
		}
		doc := readPostforkPublicJSON[map[string]any](t, filepath.Join(root, "case.json"))
		preset := doc["chainPreset"].(map[string]any)
		preset["keys"].(map[string]any)["nodekeys"].(map[string]any)["ref"] = keys
		topology := []any{}
		for i := 1; i <= 9; i++ {
			n := map[string]any{"index": i, "role": "en", "binary": "next"}
			if i == 9 {
				n = map[string]any{"index": i, "role": "bp"}
			}
			topology = append(topology, n)
		}
		preset["topology"] = map[string]any{"nodes": topology}
		genesis := preset["genesis"].(map[string]any)["overlay"].(map[string]any)
		croissant := genesis["config"].(map[string]any)["croissant"].(map[string]any)
		croissant["wBFT"] = map[string]any{"useNCP": true, "targetValidators": 7, "stabilizingStakersThreshold": 5, "epochLength": 10}
		croissant["govContracts"].(map[string]any)["govNCP"].(map[string]any)["params"].(map[string]any)["ncps"] = strings.Join(operators[:7], ",")
		alloc := genesis["alloc"].(map[string]any)
		for _, m := range metadata.Nodes {
			alloc[m.Address] = map[string]any{"balance": handoffBalance}
		}
		for _, a := range operators {
			alloc[a] = map[string]any{"balance": handoffBalance}
		}
		spec := filepath.Join(root, "selection-case.json")
		b, err := json.MarshalIndent(doc, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(spec, b, 0600); err != nil {
			t.Fatal(err)
		}
		cmd := newRootCmd()
		workspace := filepath.Join(root, "w")
		cmd.SetArgs([]string{"run", spec, "--workspace-dir", workspace, "--artifact-root", filepath.Join(root, "reports"), "--keys", keys, "--read-only-keys", "--no-skips", "--keep-up", "--node-monitor-timeout", "5m"})
		var output bytes.Buffer
		cmd.SetOut(&output)
		cmd.SetErr(&output)
		err = cmd.Execute()
		preserveHandoffRun(t, root, output.String(), err)
		if err == nil && strings.Contains(output.String(), "fail=0") {
			return root, successorRPC(t, workspace), metadata
		}
		last = fmt.Sprintf("attempt%d: %v\n%s", attempt, err, output.String())
		if err := closeHandoffRuntime(root); err != nil {
			t.Fatalf("cannot reclaim failed selection fixture: %v", err)
		}
		t.Logf("selection handoff attempt%d failed; retained and stopped before retry", attempt)
	}
	t.Fatal(last)
	return "", "", selectionRing{}
}

func TestWemixNCPStabilizationAndRankingE2E(t *testing.T) {
	from, to := os.Getenv("CHAINBENCH_E2E_FROM_BIN"), os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if from == "" || to == "" {
		t.Skip("set native handoff binaries")
	}
	keys := make([][]byte, 8)
	operators := make([]string, 8)
	for i := range keys {
		var err error
		keys[i], operators[i], err = accounts.GenerateKey()
		if err != nil {
			t.Fatal(err)
		}
		operators[i] = strings.ToLower(operators[i])
	}
	root, url, metadata := selectionHandoff(t, from, to, operators)
	c := rpc.Dial(url)
	ctx, cancel := context.WithTimeout(t.Context(), 8*time.Minute)
	defer cancel()
	p := ncpSelectionProof{Fork: 20, Target: 7, Threshold: 5, UseNCP: true}
	for i := 0; i < 8; i++ {
		m := metadata.Nodes[i]
		if m.Index != i+1 {
			t.Fatal("generated node order differs")
		}
		p.Members = append(p.Members, ncpSelectionMember{Staker: strings.ToLower(m.Address), Operator: operators[i], BLS: strings.ToLower(m.BLSPublicKey)})
	}
	observed := &observedRawRPC{client: c}
	provider, err := accounts.ForChainWithRPC("wbft", func(string) accounts.RPCRequester { return observed })
	if err != nil {
		t.Fatal(err)
	}
	wallets := make([]accounts.Wallet, 8)
	for i := range wallets {
		wallets[i], err = provider.OpenWallet(ctx, keys[i], url)
		if err != nil {
			t.Fatal(err)
		}
	}
	inputs := []map[string]any{}
	proof := map[string]any{"workspace": filepath.Join(root, "w"), "keyring": filepath.Join(root, "keyring"), "scope": "GOV009/010 actual generated eight WBFT successors plus one WEMIX producer, Croissant20/epoch10/UseNCPtrue/target7/threshold5. NCP-active stakers1,4,4(with nonNCP H registered),7 verify stabilization flags true,true,true,false; NCP council initially seven distinct operators throughout. Registered A..G power40/35/30/25/20/15/10 million WMX, H10M then45M. Stable top7 excludes highest nonNCP H; exact strict-majority NCP addition then first epoch includes H and excludes lowest G. Initial bootstrap seed8 is separate from decided target7. Exact pinned registration/eligibility/power/council lists, indexed epoch BLS and validators. Not public JSON entire execution, other thresholds/churn, Linux/mainnet/full87/legacy/final Web."}
	defer func() {
		proof["witness"], proof["inputs"], proof["submittedRaws"] = p, inputs, observed.submitted()
		b, e := json.MarshalIndent(proof, "", "  ")
		if e != nil {
			t.Error(e)
			return
		}
		if e = os.WriteFile(filepath.Join(root, "reports/ncp-selection-proof.json"), b, 0600); e != nil {
			t.Error(e)
		}
	}()
	readAt := func(contract, sig, tag string, args ...accounts.Arg) string {
		var out string
		if err := c.Call(ctx, "eth_call", &out, map[string]any{"to": contract, "data": accounts.EncodeCallArgs(sig, args...)}, tag); err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(out, "0x") || len(out) < 66 {
			t.Fatal("short pinned selection state", sig, out)
		}
		return out
	}
	state, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: filepath.Join(root, "w")})
	if err != nil {
		t.Fatal(err)
	}
	successors := []node.Node{}
	for _, n := range state.Nodes.Nodes {
		if n.Chain != "wbft" {
			continue
		}
		if n.Index < 1 || n.Index > 8 || n.Host != "127.0.0.1" || n.PID <= 0 || n.Role != node.RoleBP {
			t.Fatal("not owned live successor", n)
		}
		var address string
		if err := rpc.Dial(n.RPCURL).Call(ctx, "istanbul_nodeAddress", &address); err != nil || !strings.EqualFold(address, p.Members[n.Index-1].Staker) {
			t.Fatal("actual validator identity differs", err, address)
		}
		successors = append(successors, n)
	}
	proof["successors"] = successors
	if len(successors) != 8 || len(state.Nodes.Nodes) != 9 {
		t.Fatal("not actual eight successors plus one producer")
	}
	recordInput := func(kind string, index int, amount *big.Int, hash string) uint64 {
		rc := policyReceipt(t, c, hash)
		var tx map[string]any
		if err := c.Call(ctx, "eth_getTransactionByHash", &tx, hash); err != nil {
			t.Fatal(err)
		}
		raws := observed.submitted()
		if len(raws) == 0 {
			t.Fatal("missing signed selection input")
		}
		inputs = append(inputs, map[string]any{"kind": kind, "index": index, "amount": amount.String(), "hash": hash, "raw": raws[len(raws)-1], "transaction": tx, "receipt": rc})
		h, ok := lifecycleQuantity(rc["blockNumber"])
		if !ok || !h.IsUint64() {
			t.Fatal("input receipt height absent")
		}
		return h.Uint64()
	}
	register := func(index int) uint64 {
		m := metadata.Nodes[index]
		pk, e := hex.DecodeString(strings.TrimPrefix(m.BLSPublicKey, "0x"))
		if e != nil {
			t.Fatal(e)
		}
		pop, e := hex.DecodeString(strings.TrimPrefix(m.BLSPoP, "0x"))
		if e != nil {
			t.Fatal(e)
		}
		amount := ncpSelectionAmount(index, false)
		hash := stakingRegister(t, c, wallets[index], p.Members[index].Staker, pk, pop, amount)
		return recordInput("register", index, amount, hash)
	}
	snapshot := func(trigger uint64) {
		boundary := (trigger/10 + 1) * 10
		waitForHead(t, c, boundary+2, 90*time.Second)
		p.Snapshots = append(p.Snapshots, ncpSelectionSnapshot{TriggerHeight: trigger, Boundary: boundary, StateTag: hexUint(boundary), ValidatorTag: hexUint(boundary + 1), Active: map[string]string{}, NCP: map[string]string{}, Staked: map[string]string{}})
		s := &p.Snapshots[len(p.Snapshots)-1]
		for i := 0; i < 2; i++ {
			tag := hexUint(boundary + uint64(i))
			if err := c.Call(ctx, "eth_getBlockByNumber", &s.Blocks[i], tag, false); err != nil {
				t.Fatal(err)
			}
			if err := c.Call(ctx, "istanbul_getWbftExtraInfo", &s.Extras[i], tag); err != nil {
				t.Fatal(err)
			}
		}
		if err := c.Call(ctx, "istanbul_getValidators", &s.Validators, s.ValidatorTag); err != nil {
			t.Fatal(err)
		}
		for _, m := range p.Members {
			s.Active[m.Staker] = readAt(e2eGovStaking, "isStaker(address)", s.StateTag, accounts.Address(m.Staker))
			s.NCP[m.Operator] = readAt(e2eGovNCP, "isNCP(address)", s.StateTag, accounts.Address(m.Operator))
			info := readAt(e2eGovStaking, "stakerInfo(address)", s.StateTag, accounts.Address(m.Staker))
			if len(info) < 2+6*64 {
				t.Fatal("short staker info")
			}
			s.Staked[m.Staker] = "0x" + info[2+5*64:2+6*64]
		}
		s.StakerCount = readAt(e2eGovStaking, "stakerLength()", s.StateTag)
		s.StakerList = readAt(e2eGovStaking, "stakers()", s.StateTag)
		s.CouncilCount = readAt(e2eGovNCP, "ncpCount()", s.StateTag)
		s.CouncilList = readAt(e2eGovNCP, "ncpList()", s.StateTag)
		t.Logf("selection frame%d trigger%d first epoch%d validators%d", len(p.Snapshots)-1, trigger, boundary, len(s.Validators))
	}
	last := register(0)
	snapshot(last)
	for _, i := range []int{1, 2, 3} {
		last = register(i)
	}
	snapshot(last)
	last = register(7)
	snapshot(last)
	for _, i := range []int{4, 5, 6} {
		last = register(i)
	}
	snapshot(last)
	// H moves from10M to45M while still nonNCP. A remains40M, G10M.
	topup := new(big.Int).Sub(ncpSelectionAmount(7, true), ncpSelectionAmount(7, false))
	data := accounts.EncodeCallArgs("stake(uint256)", accounts.Uint(topup))
	raw, e := hex.DecodeString(data[2:])
	if e != nil {
		t.Fatal(e)
	}
	hash, e := wallets[7].Execute(ctx, e2eGovStaking, raw, topup)
	if e != nil {
		t.Fatal(e)
	}
	last = recordInput("stake", 7, topup, hash)
	snapshot(last)
	p.Addition = ncpMajorityProof{Members: append([]string(nil), operators[:7]...), Candidate: operators[7], Proposer: operators[0]}
	head, e := c.BlockNumber(ctx)
	if e != nil {
		t.Fatal(e)
	}
	id, ok := lifecycleQuantity(readAt(e2eGovNCP, "currentProposalID()", hexUint(head)))
	if !ok || !id.IsUint64() {
		t.Fatal("invalid addition ID")
	}
	p.Addition.ID = id.Uint64()
	for i := 0; i <= 4; i++ {
		wallet := wallets[0]
		input := accounts.EncodeCallArgs("newProposalToAddNCP(address)", accounts.Address(operators[7]))
		if i > 0 {
			wallet = wallets[i-1]
			input = accounts.EncodeCallArgs("vote(uint256,bool)", accounts.Uint(new(big.Int).SetUint64(p.Addition.ID)), accounts.Word([]byte{1}))
		}
		calldata, e := hex.DecodeString(input[2:])
		if e != nil {
			t.Fatal(e)
		}
		hash, e := wallet.Execute(ctx, e2eGovNCP, calldata, nil)
		if e != nil {
			t.Fatal(e)
		}
		raws := observed.submitted()
		p.Addition.Stages = append(p.Addition.Stages, ncpBallotStage{Raw: raws[len(raws)-1]})
		s := &p.Addition.Stages[i]
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
		s.Count = readAt(e2eGovNCP, "ncpCount()", tag)
		s.List = readAt(e2eGovNCP, "ncpList()", tag)
		s.CandidateMember = readAt(e2eGovNCP, "isNCP(address)", tag, accounts.Address(operators[7]))
		s.Proposal = readAt(e2eGovNCP, "proposals(uint256)", tag, accounts.Uint(new(big.Int).SetUint64(p.Addition.ID)))
		s.CurrentID = readAt(e2eGovNCP, "currentProposalID()", tag)
		h, ok := lifecycleQuantity(tag)
		if !ok || !h.IsUint64() {
			t.Fatal("missing vote receipt height")
		}
		last = h.Uint64()
	}
	if err := verifyNCPMajority(p.Addition); err != nil {
		t.Fatal(err)
	}
	snapshot(last)
	if err := verifyNCPSelection(p); err != nil {
		t.Fatal(err)
	}
}
