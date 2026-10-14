package testengine_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

type proposerSample struct {
	Number                  uint64
	Miner, Hash, ParentHash string
}
type proposerStatus struct {
	Author     map[string]int                                     `json:"author"`
	BlockRange struct{ StartBlock, EndBlock, TotalBlocks uint64 } `json:"blockRange"`
}

func verifyRoundRobin50(validators []string, samples []proposerSample, status proposerStatus) error {
	if len(validators) != 5 || len(samples) != 50 || status.BlockRange.StartBlock != 101 || status.BlockRange.EndBlock != 150 || status.BlockRange.TotalBlocks != 50 {
		return fmt.Errorf("need five validators and exact101..150 window")
	}
	allowed := map[string]bool{}
	counts := map[string]int{}
	for _, addr := range validators {
		key := strings.ToLower(addr)
		if key == "" || allowed[key] {
			return fmt.Errorf("duplicate/empty validator")
		}
		allowed[key] = true
	}
	for i, s := range samples {
		key := strings.ToLower(s.Miner)
		if s.Number != uint64(101+i) || !allowed[key] || s.Hash == "" || i > 0 && (key == strings.ToLower(samples[i-1].Miner) || s.ParentHash != samples[i-1].Hash) {
			return fmt.Errorf("wrong proposer or linked height at %d", s.Number)
		}
		counts[key]++
	}
	actualStatus := map[string]int{}
	for key, n := range status.Author {
		key = strings.ToLower(key)
		_, duplicate := actualStatus[key]
		if !allowed[key] || duplicate {
			return fmt.Errorf("unknown/duplicate status author")
		}
		actualStatus[key] = n
	}
	if len(actualStatus) != 5 {
		return fmt.Errorf("missing status author")
	}
	for key := range allowed {
		if counts[key] != 10 || actualStatus[key] != counts[key] {
			return fmt.Errorf("unbalanced or inconsistent author count for %s", key)
		}
	}
	return nil
}

func TestRoundRobin50RejectsShortAndUnbalancedEvidence(t *testing.T) {
	for _, name := range []string{"valid", "short", "unknown proposer", "duplicate consecutive", "unbalanced", "wrong height", "wrong parent", "wrong status count", "wrong status range"} {
		t.Run(name, func(t *testing.T) {
			vals := []string{"v0", "v1", "v2", "v3", "v4"}
			samples := []proposerSample{}
			status := proposerStatus{Author: map[string]int{"v0": 10, "v1": 10, "v2": 10, "v3": 10, "v4": 10}}
			status.BlockRange.StartBlock = 101
			status.BlockRange.EndBlock = 150
			status.BlockRange.TotalBlocks = 50
			for n := uint64(101); n <= 150; n++ {
				samples = append(samples, proposerSample{Number: n, Miner: vals[(n-101)%5], Hash: fmt.Sprint(n), ParentHash: fmt.Sprint(n - 1)})
			}
			switch name {
			case "short":
				samples = samples[:16]
			case "unknown proposer":
				samples[49].Miner = "stranger"
			case "duplicate consecutive":
				samples[1].Miner = samples[0].Miner
			case "unbalanced":
				samples[49].Miner = "v0"
			case "wrong height":
				samples[1].Number++
			case "wrong parent":
				samples[1].ParentHash = "unrelated"
			case "wrong status count":
				status.Author["v0"] = 9
			case "wrong status range":
				status.BlockRange.StartBlock = 100
			}
			err := verifyRoundRobin50(vals, samples, status)
			if name == "valid" && err != nil {
				t.Fatal(err)
			}
			if name != "valid" && err == nil {
				t.Fatal("weak proposer presence accepted incomplete criterion")
			}
		})
	}
}

func TestNativeRoundRobinFiveValidators(t *testing.T) {
	bin := os.Getenv("GWBFT_BIN")
	if bin == "" {
		t.Skip("set GWBFT_BIN for actual five-validator proof")
	}
	root := failureRuntime(t)
	ctx, cancel := context.WithTimeout(t.Context(), 6*time.Minute)
	defer cancel()
	specs := roundRobinSpecs(t, root)
	out, runErr := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{SpecContent: specs, DataDir: filepath.Join(root, "w"), ArtifactRoot: filepath.Join(root, "reports"), Binary: bin, KeysDir: filepath.Join(repoRoot(t), "presets/keys"), ReadOnlyKeys: true, KeepUp: true, WaitBlocks: 2})
	proof := map[string]any{"binary": bin, "session": out.SessionRoot, "summary": out.Summary, "scope": "WBFT-006 managed current WBFT Croissant0, five validators, exact blocks101..150. Not original migrated/mainnet/Linux/complete87/final Web acceptance."}
	defer func() {
		b, err := json.MarshalIndent(proof, "", "  ")
		if err != nil {
			t.Error(err)
			return
		}
		if err = os.WriteFile(filepath.Join(root, "reports/round-robin-proof.json"), b, 0600); err != nil {
			t.Error(err)
		}
	}()
	if runErr != nil || out.Summary.Summary.Pass != 1 || out.Summary.Failed() || out.Summary.Summary.Skip != 0 {
		t.Fatalf("real fifty blocks failed: %v %+v", runErr, out.Summary)
	}
	state, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: filepath.Join(root, "w")})
	if err != nil || len(state.Nodes.Nodes) != 5 {
		t.Fatal("not five actual nodes", err)
	}
	c := rpc.Dial(state.Nodes.Nodes[0].RPCURL)
	var validators []string
	if err = c.Call(ctx, "istanbul_getValidators", &validators, "0x65"); err != nil {
		t.Fatal(err)
	}
	var status proposerStatus
	if err = c.Call(ctx, "istanbul_status", &status, "0x65", "0x96"); err != nil {
		t.Fatal(err)
	}
	samples := []proposerSample{}
	for n := uint64(101); n <= 150; n++ {
		var b struct{ Number, Miner, Hash, ParentHash string }
		if err = c.Call(ctx, "eth_getBlockByNumber", &b, fmt.Sprintf("0x%x", n), false); err != nil {
			t.Fatal(err)
		}
		height, e := strconv.ParseUint(strings.TrimPrefix(b.Number, "0x"), 16, 64)
		if e != nil || height != n {
			t.Fatal("wrong actual header height", b.Number)
		}
		samples = append(samples, proposerSample{Number: height, Miner: b.Miner, Hash: b.Hash, ParentHash: b.ParentHash})
	}
	var policy map[string]any
	var genesisPath string
	err = filepath.WalkDir(filepath.Join(root, "w"), func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() && (d.Name() == "chaindata" || d.Name() == "lightchaindata" || d.Name() == "etcd") {
			return filepath.SkipDir
		}
		if d.IsDir() || d.Name() != "genesis.json" {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		var g struct {
			Config struct {
				Croissant struct {
					WBFT map[string]any `json:"wBFT"`
				} `json:"croissant"`
			}
		}
		if err = json.Unmarshal(b, &g); err != nil {
			return err
		}
		policy = g.Config.Croissant.WBFT
		genesisPath = p
		return fs.SkipAll
	})
	if err != nil || policy == nil || policy["proposerPolicy"] != float64(0) || policy["targetValidators"] != float64(5) {
		t.Fatal("actual genesis not roundrobin five", policy, err)
	}
	proof["genesisPath"] = genesisPath
	proof["wbftConfig"] = policy
	proof["validators"] = validators
	proof["samples"] = samples
	proof["status"] = status
	if err = verifyRoundRobin50(validators, samples, status); err != nil {
		t.Fatal(err)
	}
}

func roundRobinSpecs(t *testing.T, root string) [][]byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "presets/chain/wbft-bp4.json"))
	if err != nil {
		t.Fatal(err)
	}
	var env map[string]any
	if err = json.Unmarshal(b, &env); err != nil {
		t.Fatal(err)
	}
	env["id"] = "wbft-bp5-round-robin"
	env["topology"] = map[string]any{"bp": 5}
	b, err = json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	envPath := filepath.Join(root, "env.json")
	if err = os.WriteFile(envPath, b, 0600); err != nil {
		t.Fatal(err)
	}
	casePath := filepath.Join(root, "case.json")
	raw := []byte(`{"schemaVersion":"2","kind":"case","id":"round-robin-fifty","chainPreset":"wbft-bp5-round-robin","requires":["rpc"],"steps":[{"do":"waitBlock","target":150,"timeout":"240s"},{"expect":"blockNumber","compare":"GreaterOrEqual","is":"150"}]}`)
	if err = os.WriteFile(casePath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	specs, err := dsl.ReadFilesWithChainPreset([]string{casePath}, envPath)
	if err != nil {
		t.Fatal(err)
	}
	return specs
}

func TestRoundRobinSpecsBindFiveValidatorPreset(t *testing.T) {
	raws := roundRobinSpecs(t, t.TempDir())
	if len(raws) != 1 {
		t.Fatal("wrong case count")
	}
	s, err := dsl.ParseV2(raws[0])
	if err != nil {
		t.Fatal(err)
	}
	if s.Chain.Name != "wbft" || fmt.Sprint(s.Topology["bp"]) != "5" {
		t.Fatalf("wrong real fixture: chain=%s topology=%v", s.Chain.Name, s.Topology)
	}
}
