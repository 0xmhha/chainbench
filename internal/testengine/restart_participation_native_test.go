package testengine_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

func TestNativeValidatorRestartParticipation(t *testing.T) {
	bin := os.Getenv("GWBFT_BIN")
	if bin == "" {
		t.Skip("set GWBFT_BIN for actual restart participation")
	}
	root := failureRuntime(t)
	data := filepath.Join(root, "w")
	ctx, cancel := context.WithTimeout(t.Context(), 6*time.Minute)
	defer cancel()
	path := filepath.Join(repoRoot(t), "tests/tc/common/fault/CT-FAULT-001-producer-crash-and-restart.json")
	raws, err := dsl.ReadFilesWithChainPreset([]string{path}, "wbft-bp4")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "case.json"), raws[0], 0o600); err != nil {
		t.Fatal(err)
	}
	out, runErr := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{SpecContent: raws, DataDir: data, ArtifactRoot: filepath.Join(root, "reports"), Binary: bin, KeysDir: filepath.Join(repoRoot(t), "presets/keys"), ReadOnlyKeys: true, KeepUp: true})
	p := restartWindowProof{}
	observed := []restartWindowSample{}
	proof := map[string]any{"scope": "NODE005 managed WBFT Croissant0 four validators. Unmodified public crash/restart case checks surviving progress, catch-up, common hash/stateRoot; retained old/new process records and same datadir, future six-header healthy one-second window and original node commit participation. Not staged migration/Linux/mainnet/legacy/full87/final Web.", "binary": bin, "session": out.SessionRoot, "summary": out.Summary, "publicCase": path}
	defer func() {
		proof["window"], proof["observedCandidates"] = p, observed
		body, err := json.MarshalIndent(proof, "", "  ")
		if err != nil {
			t.Error(err)
			return
		}
		if err := os.WriteFile(filepath.Join(root, "reports/restart-participation-proof.json"), body, 0o600); err != nil {
			t.Error(err)
		}
	}()
	if runErr != nil || out.Summary.Summary.Pass != 1 || out.Summary.Failed() || out.Summary.Summary.Skip != 0 {
		t.Fatal("public restart case did not pass", runErr, out.Summary)
	}
	genesis := filepath.Join(data, "genesis.json")
	body, err := os.ReadFile(genesis)
	if err != nil {
		t.Fatal(err)
	}
	var gen struct {
		Config struct {
			CroissantBlock uint64
			Croissant      struct {
				WBFT struct {
					BlockPeriod uint64 `json:"blockPeriodSeconds"`
				} `json:"wBFT"`
			}
		}
	}
	if err := json.Unmarshal(body, &gen); err != nil {
		t.Fatal(err)
	}
	if gen.Config.CroissantBlock != 0 || gen.Config.Croissant.WBFT.BlockPeriod != 1 {
		t.Fatal("actual genesis is not Croissant0/one-second", gen)
	}
	proof["genesisPath"] = genesis
	state, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: data})
	if err != nil || len(state.Nodes.Nodes) != 4 {
		t.Fatal("not four managed validators", err)
	}
	var primary, target *rpc.Client
	for _, n := range state.Nodes.Nodes {
		if n.Index == 1 {
			primary = rpc.Dial(n.RPCURL)
		}
		if n.Index == 3 {
			target = rpc.Dial(n.RPCURL)
		}
	}
	if primary == nil || target == nil {
		t.Fatal("public case nodes missing")
	}
	ledger, err := process.OpenLedger(data)
	if err != nil {
		t.Fatal(err)
	}
	history := ledger.History("node3")
	current, ok := ledger.Get("node3")
	if !ok || len(history) == 0 {
		t.Fatal("restart process history missing")
	}
	old := history[len(history)-1]
	if old.PID <= 0 || current.PID <= 0 || old.PID == current.PID || old.DataDir == "" || old.DataDir != current.DataDir || old.Binary != current.Binary || old.Command != current.Command || current.Revision <= old.Revision {
		t.Fatal("restart did not preserve actual node command/datadir and replace pid", old, current)
	}
	proof["previousProcess"], proof["currentProcess"] = old, current
	meta, err := os.ReadFile(filepath.Join(repoRoot(t), "presets/keys/metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	var keys struct {
		Nodes []struct {
			Index   int
			Address string
		}
	}
	if err := json.Unmarshal(meta, &keys); err != nil {
		t.Fatal(err)
	}
	expected := ""
	for _, n := range keys.Nodes {
		if n.Index == 3 {
			expected = n.Address
		}
	}
	if expected == "" {
		t.Fatal("original node3 key identity missing")
	}
	p.Restarted, err = waitRestartIdentity(ctx, target, expected)
	if err != nil {
		t.Fatal(err)
	}
	anchor, err := primary.BlockNumber(ctx)
	if err != nil {
		t.Fatal(err)
	}
	wait := func(c *rpc.Client, height uint64) {
		t.Helper()
		for {
			head, err := c.BlockNumber(ctx)
			if err == nil && head >= height {
				return
			}
			select {
			case <-ctx.Done():
				t.Fatal("head not reached", height, ctx.Err(), err)
			case <-time.After(100 * time.Millisecond):
			}
		}
	}
	wait(target, anchor)
	tag := fmt.Sprintf("0x%x", anchor)
	var reference, restarted map[string]any
	if err := primary.Call(ctx, "eth_getBlockByNumber", &reference, tag, false); err != nil {
		t.Fatal(err)
	}
	if err := target.Call(ctx, "eth_getBlockByNumber", &restarted, tag, false); err != nil {
		t.Fatal(err)
	}
	if reference["number"] != tag || restarted["number"] != tag || reference["hash"] == nil || reference["stateRoot"] == nil || reference["hash"] != restarted["hash"] || reference["stateRoot"] != restarted["stateRoot"] {
		t.Fatal("restarted fixed-height state differs", reference, restarted)
	}
	proof["syncAgreement"] = map[string]any{"height": anchor, "primary": reference, "restarted": restarted}
	p.SyncHeight, err = primary.BlockNumber(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := primary.Call(ctx, "istanbul_getValidators", &p.Validators, fmt.Sprintf("0x%x", p.SyncHeight)); err != nil {
		t.Fatal(err)
	}
	for n := p.SyncHeight + 1; n <= p.SyncHeight+40; n++ {
		wait(primary, n)
		h, err := readViewHeader(ctx, primary, n)
		if err != nil {
			t.Fatal(err)
		}
		sample := restartWindowSample{Header: h}
		if err := primary.Call(ctx, "istanbul_getCommitSignersFromBlock", &sample.Signers, fmt.Sprintf("0x%x", n)); err != nil {
			t.Fatal(err)
		}
		observed = append(observed, sample)
		if len(observed) < 6 {
			continue
		}
		p.Samples = append([]restartWindowSample(nil), observed[len(observed)-6:]...)
		if verifyRestartWindow(p) == nil {
			break
		}
	}
	if err := verifyRestartWindow(p); err != nil {
		t.Fatal("no bounded recovered one-second participation window", err)
	}
	last := p.Samples[len(p.Samples)-1].Header
	wait(target, last.Number)
	var final map[string]any
	if err := target.Call(ctx, "eth_getBlockByNumber", &final, fmt.Sprintf("0x%x", last.Number), false); err != nil {
		t.Fatal(err)
	}
	if final["hash"] != last.Hash || !strings.HasPrefix(fmt.Sprint(final["stateRoot"]), "0x") {
		t.Fatal("restarted node did not reach the actual recovery window", final)
	}
	proof["finalRestartedHeader"] = final
}
