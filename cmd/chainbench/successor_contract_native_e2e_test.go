//go:build e2e

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/testengine"
)

func TestNativeSuccessorContractCapabilityE2E(t *testing.T) {
	from, to := os.Getenv("CHAINBENCH_E2E_FROM_BIN"), os.Getenv("CHAINBENCH_E2E_TO_BIN")
	if from == "" || to == "" {
		t.Skip("set native handoff binaries")
	}
	workspace, url := runGovHandoffWorkspace(t, from, to, writeHandoffCase(t, nil))
	root := filepath.Dir(workspace)
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	state, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: workspace})
	if err != nil {
		t.Fatal(err)
	}
	successors := node.NodeSet{Chain: "wemix", Network: "recorded-contract-successors"}
	for _, n := range state.Nodes.Nodes {
		if n.Chain == "wbft" {
			successors.Nodes = append(successors.Nodes, n)
		}
	}
	if len(state.Nodes.Nodes) != 5 || len(successors.Nodes) != 4 {
		t.Fatal("missing recorded successors", state.Nodes)
	}
	code, err := rpc.Dial(url).CodeAt(ctx, e2eGovStaking)
	if err != nil || code == "0x" {
		t.Fatal("actual successor contract absent", err)
	}
	decoded, err := hex.DecodeString(code[2:])
	if err != nil {
		t.Fatal(err)
	}
	codeHash := sha256.Sum256(decoded)
	proof := map[string]any{"scope": "Contract gate binds each recorded WBFT successor despite predecessor WEMIX run dialect; unknown recorded chain blocks, legacy missing chain uses explicit WBFT run fallback. Four actual successors on pinned handoff. Not mixed wbft-stablenet, contract ABI/version/permission, full R3/P4/L1/87/mainnet/legacy/final Web acceptance.", "allNodes": state.Nodes, "contract": e2eGovStaking, "code": code, "codeSHA256": hex.EncodeToString(codeHash[:])}
	profiles := []map[string]any{}
	defer func() {
		proof["profiles"] = profiles
		b, e := json.MarshalIndent(proof, "", "  ")
		if e != nil {
			t.Error(e)
			return
		}
		if e = os.WriteFile(filepath.Join(root, "reports/successor-contract-proof.json"), b, 0600); e != nil {
			t.Error(e)
		}
	}()
	for _, profile := range []string{"recorded-successor", "unknown-recorded-chain", "legacy-fallback"} {
		outer := "wemix"
		ns := successors
		ns.Nodes = append([]node.Node(nil), successors.Nodes...)
		if profile == "unknown-recorded-chain" {
			outer = "stablenet"
			ns.Nodes[0].Chain = "unknown-chain"
		}
		if profile == "legacy-fallback" {
			outer = "wbft"
			for i := range ns.Nodes {
				ns.Nodes[i].Chain = ""
			}
		}
		raw, err := json.Marshal(map[string]any{"schemaVersion": "2", "kind": "case", "id": profile, "chainPreset": map[string]any{"chain": outer}, "requires": []string{"rpc", "contract:govStaking"}, "steps": []any{map[string]any{"expect": "rpc", "method": "eth_getCode", "params": []any{e2eGovStaking, "latest"}, "is": code}}})
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(root, profile+".json"), raw, 0600); err != nil {
			t.Fatal(err)
		}
		engine, err := testengine.NewAttachEngine(testengine.AttachConfig{Chain: outer, Nodes: ns, Caps: []string{"contract:govStaking"}, KeysDir: presetKeysDir(t), ReadOnlyKeys: true, ArtifactRoot: filepath.Join(root, "reports", profile)})
		if err != nil {
			t.Fatal(err)
		}
		dir, err := engine.Run(ctx, [][]byte{raw})
		if err != nil {
			t.Fatal(err)
		}
		summary, err := testengine.ReadSessionSummary(dir)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, "tests", "001_"+profile, "observations/network-capability-probes.json")
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var observation struct {
			Probes []struct {
				Capability, Status string
				Nodes              []struct{ Chain, Address, Status, Reason, CodeSHA256 string }
			}
		}
		if err = json.Unmarshal(b, &observation); err != nil {
			t.Fatal(err)
		}
		found := false
		for _, p := range observation.Probes {
			if p.Capability != "contract:govStaking" {
				continue
			}
			found = true
			if len(p.Nodes) != 4 {
				t.Fatal(string(b))
			}
			for i, n := range p.Nodes {
				if profile == "unknown-recorded-chain" && i == 0 {
					if n.Chain != "unknown-chain" || n.Status != "unknown" || n.Reason != "recorded_chain_unavailable" || n.Address != "" {
						t.Fatal(string(b))
					}
					continue
				}
				if n.Chain != "wbft" || n.Status != "supported" || n.Address != e2eGovStaking || n.CodeSHA256 != hex.EncodeToString(codeHash[:]) {
					t.Fatal(string(b))
				}
			}
		}
		if !found {
			t.Fatal("contract observation missing")
		}
		if summary.Summary.Fail != 0 || summary.Summary.Skip != 0 {
			t.Fatal(summary)
		}
		if profile == "unknown-recorded-chain" {
			if summary.Summary.Blocked != 1 || summary.Summary.Pass != 0 {
				t.Fatal(summary)
			}
		} else if summary.Summary.Pass != 1 || summary.Summary.Blocked != 0 {
			t.Fatal(summary)
		}
		profiles = append(profiles, map[string]any{"profile": profile, "outerChain": outer, "session": dir, "summary": summary, "observation": json.RawMessage(b)})
	}
	after, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: workspace})
	if err != nil || len(after.Nodes.Nodes) != len(state.Nodes.Nodes) {
		t.Fatal("attach changed node table", err)
	}
	for i, n := range state.Nodes.Nodes {
		got := after.Nodes.Nodes[i]
		if n.PID != got.PID || n.Chain != got.Chain || n.RPCURL != got.RPCURL {
			t.Fatal("attach changed recorded node identity", n, got)
		}
	}
	proof["afterNodes"] = after.Nodes
}
