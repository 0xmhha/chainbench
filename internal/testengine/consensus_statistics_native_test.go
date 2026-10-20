package testengine_test

import (
	"context"
	"encoding/json"
	"fmt"
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

func TestNativeConsensusStatisticsFifty(t *testing.T) {
	bin := os.Getenv("GWBFT_BIN")
	if bin == "" {
		t.Skip("set GWBFT_BIN for fifty-block statistics")
	}
	root := failureRuntime(t)
	ctx, cancel := context.WithTimeout(t.Context(), 6*time.Minute)
	defer cancel()
	p := filepath.Join(repoRoot(t), "tests/tc/go-stablenet/regression/api/15-istanbul-status-fields.json")
	raws, err := dsl.ReadFilesWithChainPreset([]string{p}, filepath.Join(repoRoot(t), "presets/chain/wbft-bp4.json"))
	if err != nil {
		t.Fatal(err)
	}
	var spec map[string]any
	if err = json.Unmarshal(raws[0], &spec); err != nil {
		t.Fatal(err)
	}
	spec["steps"] = append([]any{map[string]any{"do": "waitBlock", "target": 50, "timeout": "180s"}}, spec["steps"].([]any)...)
	raws[0], err = json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(root, "case.json"), raws[0], 0600); err != nil {
		t.Fatal(err)
	}
	out, runErr := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{SpecContent: raws, DataDir: filepath.Join(root, "w"), ArtifactRoot: filepath.Join(root, "reports"), Binary: bin, KeysDir: filepath.Join(repoRoot(t), "presets/keys"), ReadOnlyKeys: true, KeepUp: true, WaitBlocks: 2})
	proof := map[string]any{"binary": bin, "session": out.SessionRoot, "summary": out.Summary, "scope": "RPC-006 current WBFT Croissant0 managed BP4. Public presence case plus exact1..50 independent header/seal/RPC counters. Not full87/Linux/mainnet/legacy/final Web."}
	defer func() {
		b, e := json.MarshalIndent(proof, "", "  ")
		if e != nil {
			t.Error(e)
			return
		}
		if e = os.WriteFile(filepath.Join(root, "reports/consensus-statistics-proof.json"), b, 0600); e != nil {
			t.Error(e)
		}
	}()
	if runErr != nil || out.Summary.Summary.Pass != 1 || out.Summary.Failed() || out.Summary.Summary.Skip != 0 {
		t.Fatalf("actual statistics setup failed: %v %+v", runErr, out.Summary)
	}
	state, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: filepath.Join(root, "w")})
	if err != nil || len(state.Nodes.Nodes) != 4 {
		t.Fatal("missing four actual nodes", err)
	}
	c := rpc.Dial(state.Nodes.Nodes[0].RPCURL)
	var vals []string
	if err = c.Call(ctx, "istanbul_getValidators", &vals, "0x1"); err != nil {
		t.Fatal(err)
	}
	var status statisticsStatus
	if err = c.Call(ctx, "istanbul_status", &status, "0x1", "0x32"); err != nil {
		t.Fatal(err)
	}
	blocks := []statisticsBlock{}
	rawExtras := []map[string]any{}
	fields := map[string]string{"prepared": "preparedSeal", "committed": "committedSeal", "prevPrepared": "prevPreparedSeal", "prevCommitted": "prevCommittedSeal"}
	for n := uint64(1); n <= 50; n++ {
		tag := fmt.Sprintf("0x%x", n)
		var b struct{ Number, Hash, ParentHash, Miner string }
		if err = c.Call(ctx, "eth_getBlockByNumber", &b, tag, false); err != nil {
			t.Fatal(err)
		}
		var extra map[string]any
		if err = c.Call(ctx, "istanbul_getWbftExtraInfo", &extra, tag); err != nil {
			t.Fatal(err)
		}
		roundText, ok := extra["round"].(string)
		if !ok {
			t.Fatal("missing actual round")
		}
		round, e := strconv.ParseUint(strings.TrimPrefix(roundText, "0x"), 16, 64)
		if e != nil || b.Number != tag {
			t.Fatal("wrong block/round", e, b, extra)
		}
		sample := statisticsBlock{Number: n, Hash: b.Hash, ParentHash: b.ParentHash, Author: b.Miner, Round: round, Seals: map[string][]string{}}
		for category, field := range fields {
			raw, exists := extra[field]
			if !exists {
				t.Fatal("missing seal field", field)
			}
			sample.Seals[category] = nil
			if raw == nil {
				continue
			}
			seal, ok := raw.(map[string]any)
			if !ok {
				t.Fatal("invalid seal", raw)
			}
			signers, ok := seal["sealers"].([]any)
			if !ok {
				t.Fatal("invalid signers", seal)
			}
			for _, s := range signers {
				v, ok := s.(string)
				if !ok {
					t.Fatal("invalid signer", s)
				}
				sample.Seals[category] = append(sample.Seals[category], v)
			}
		}
		blocks = append(blocks, sample)
		rawExtras = append(rawExtras, extra)
	}
	proof["params"] = []string{"0x1", "0x32"}
	proof["validators"] = vals
	proof["blocks"] = blocks
	proof["rawExtras"] = rawExtras
	proof["status"] = status
	proof["assertions"] = filepath.Join(out.SessionRoot, "tests/001_istanbul-status-fields/assert.json")
	proof["steps"] = filepath.Join(out.SessionRoot, "tests/001_istanbul-status-fields/steps.json")
	if err = verifyStatistics50(vals, blocks, status); err != nil {
		t.Fatal(err)
	}
}
