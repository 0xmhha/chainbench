package testengine_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/accounts"
	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

func TestNativeBriocheHalvingProposerLedger(t *testing.T) {
	bin := os.Getenv("GWBFT_BIN")
	if bin == "" {
		t.Skip("set GWBFT_BIN for actual halving and proposer ledger")
	}
	root := failureRuntime(t)
	workspace := filepath.Join(root, "w")
	keys := filepath.Join(repoRoot(t), "presets/keys")
	env := map[string]any{"schemaVersion": "2", "kind": "chain-preset", "id": "brioche-reward-ledger", "chain": "wbft", "topology": map[string]any{"bp": 4}, "keys": map[string]any{"nodekeys": map[string]any{"ref": keys, "source": "keyPreset"}}, "genesis": map[string]any{"set": map[string]any{
		"config.croissant.wBFT.blockReward":            "0x1bc16d674ec80000",
		"config.croissant.wBFT.blockRewardBeneficiary": nil,
		"config.brioche.blockReward":                   json.Number("2000000000000000000"),
		"config.brioche.firstHalvingBlock":             10, "config.brioche.halvingPeriod": 10, "config.brioche.finishRewardBlock": 1000, "config.brioche.halvingTimes": 4, "config.brioche.halvingRate": 50,
	}}}
	launch := map[string]map[string]any{}
	for i := 1; i <= 4; i++ {
		launch[fmt.Sprintf("node%d", i)] = map[string]any{"http": true, "http.api": "admin,eth,debug,miner,net,txpool,personal,web3,istanbul,wemix"}
	}
	env["launch"] = launch
	write := func(path string, v any) {
		b, e := json.MarshalIndent(v, "", "  ")
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(path, b, 0600); e != nil {
			t.Fatal(e)
		}
	}
	envPath := filepath.Join(root, "env.json")
	write(envPath, env)
	casePath := filepath.Join(root, "case.json")
	write(casePath, map[string]any{"schemaVersion": "2", "kind": "case", "id": "brioche-reward-ready", "chainPreset": "brioche-reward-ledger", "requires": []string{"rpc", "consensus"}, "steps": []any{map[string]any{"do": "waitBlock", "target": 16, "timeout": "90s"}, map[string]any{"expect": "blockNumber", "compare": "GreaterOrEqual", "is": "16"}}})
	specs, err := dsl.ReadFilesWithChainPreset([]string{casePath}, envPath)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	out, runErr := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{SpecContent: specs, DataDir: workspace, ArtifactRoot: filepath.Join(root, "reports"), Binary: bin, KeysDir: keys, ReadOnlyKeys: true, KeepUp: true, WaitBlocks: 2})
	p := briocheRewardProof{}
	proof := map[string]any{"workspace": workspace, "session": out.SessionRoot, "summary": out.Summary, "scope": "RPC008 exact configured2ETH reward and50percent first halving10/period10/times4/finish1000. Actual mined5/15 public-case input heights and9/10 boundary on four managed WBFT validators, direct Croissant0/Brioche0. No registered stakers or reward beneficiaries, no transactions/gas in sampled blocks, so full reward credits actual proposer fallback address. Exact parent/mined balance delta equals configured reward/API,2ETH before and1ETH after. Brioche already active from genesis: this is halving transition, not Brioche activation or staged migration/public JSON full reexecution/Linux/mainnet/full87/legacy/final Web."}
	defer func() {
		proof["witness"] = p
		writeProof, e := json.MarshalIndent(proof, "", "  ")
		if e != nil {
			t.Error(e)
			return
		}
		if e = os.WriteFile(filepath.Join(root, "reports/brioche-reward-proof.json"), writeProof, 0600); e != nil {
			t.Error(e)
		}
	}()
	if runErr != nil || out.Summary.Summary.Pass != 1 || out.Summary.Failed() || out.Summary.Summary.Skip != 0 {
		t.Fatal("halving fixture not ready", runErr, out.Summary)
	}
	genesis, err := os.ReadFile(filepath.Join(workspace, "genesis.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	dec := json.NewDecoder(bytes.NewReader(genesis))
	dec.UseNumber()
	if err := dec.Decode(&doc); err != nil {
		t.Fatal(err)
	}
	cfg := doc["config"].(map[string]any)
	wbft := cfg["croissant"].(map[string]any)["wBFT"].(map[string]any)
	brioche := cfg["brioche"].(map[string]any)
	parse := func(v any) *big.Int {
		s := fmt.Sprint(v)
		base := 10
		if strings.HasPrefix(s, "0x") {
			s = s[2:]
			base = 16
		}
		n, valid := new(big.Int).SetString(s, base)
		if !valid || n.Sign() < 0 {
			t.Fatal("invalid exact reward configuration", v)
		}
		return n
	}
	if cfg["briocheBlock"].(json.Number).String() != "0" || cfg["croissantBlock"].(json.Number).String() != "0" || wbft["blockRewardBeneficiary"] != nil {
		t.Fatal("not isolated genesis-active fixture")
	}
	p.WBFTBase = parse(wbft["blockReward"]).String()
	p.BriocheBase = parse(brioche["blockReward"]).String()
	u64 := func(v any) uint64 {
		n := parse(v)
		if !n.IsUint64() {
			t.Fatal("invalid uint64 reward configuration")
		}
		return n.Uint64()
	}
	p.First = u64(brioche["firstHalvingBlock"])
	p.Period = u64(brioche["halvingPeriod"])
	p.Finish = u64(brioche["finishRewardBlock"])
	p.Times = u64(brioche["halvingTimes"])
	p.Rate = u64(brioche["halvingRate"])
	proof["actualConfig"] = cfg
	status, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: workspace})
	if err != nil || len(status.Nodes.Nodes) != 4 {
		t.Fatal("not four recorded native validators", err)
	}
	proof["nodes"] = status.Nodes.Nodes
	for _, n := range status.Nodes.Nodes {
		if n.Role != node.RoleBP || n.Host != "127.0.0.1" || n.PID <= 0 {
			t.Fatal("not owned active validator", n)
		}
		var addr string
		if err := rpc.Dial(n.RPCURL).Call(ctx, "istanbul_nodeAddress", &addr); err != nil {
			t.Fatal(err)
		}
		p.ActualNodes = append(p.ActualNodes, strings.ToLower(addr))
	}
	c := rpc.Dial(status.Nodes.Nodes[0].RPCURL)
	for _, height := range []uint64{5, 9, 10, 15} {
		tag, parent := fmt.Sprintf("0x%x", height), fmt.Sprintf("0x%x", height-1)
		p.Rows = append(p.Rows, briocheRewardRow{Height: height, RPCQueryTag: tag, BeforeTag: parent, AfterTag: tag})
		r := &p.Rows[len(p.Rows)-1]
		if err := c.Call(ctx, "eth_getBlockByNumber", &r.Parent, parent, false); err != nil {
			t.Fatal(err)
		}
		if err := c.Call(ctx, "eth_getBlockByNumber", &r.Block, tag, false); err != nil {
			t.Fatal(err)
		}
		r.Proposer = fmt.Sprint(r.Block["miner"])
		if err := c.Call(ctx, "wemix_getBriocheBlockReward", &r.Reward, tag); err != nil {
			t.Fatal(err)
		}
		if err := c.Call(ctx, "eth_getBalance", &r.Before, r.Proposer, parent); err != nil {
			t.Fatal(err)
		}
		if err := c.Call(ctx, "eth_getBalance", &r.After, r.Proposer, tag); err != nil {
			t.Fatal(err)
		}
		if err := c.Call(ctx, "eth_call", &r.StakerCount, map[string]any{"to": "0x0000000000000000000000000000000000001001", "data": accounts.EncodeCallArgs("stakerLength()")}, tag); err != nil {
			t.Fatal(err)
		}
	}
	if err := verifyBriocheReward(p); err != nil {
		t.Fatal(err)
	}
}
