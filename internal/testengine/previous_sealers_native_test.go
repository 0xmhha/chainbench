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

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

func TestNativePreviousSealerMembership(t *testing.T) {
	bin := os.Getenv("GWBFT_BIN")
	if bin == "" {
		t.Skip("set GWBFT_BIN for actual previous sealers")
	}
	for _, n := range []int{5, 7} {
		t.Run(fmt.Sprintf("bp%d", n), func(t *testing.T) {
			root := failureRuntime(t)
			data, keys := filepath.Join(root, "w"), filepath.Join(root, "keyring")
			ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
			defer cancel()
			if _, err := app.KeyringNew(ctx, app.Deps{}, app.RingCreateIn{Ring: app.RingRef{Dir: keys}, Count: n, Validators: &n, WithBLS: true}); err != nil {
				t.Fatal(err)
			}
			env := map[string]any{"schemaVersion": "2", "kind": "chain-preset", "id": fmt.Sprintf("previous-sealers-bp%d", n), "chain": "wbft", "topology": map[string]any{"bp": n}, "keys": map[string]any{"nodekeys": map[string]any{"ref": keys, "source": "keyPreset"}}, "genesis": map[string]any{"set": map[string]any{"config.croissant.wBFT.targetValidators": n, "config.croissant.wBFT.stabilizingStakersThreshold": n + 1}}}
			b, err := json.Marshal(env)
			if err != nil {
				t.Fatal(err)
			}
			envPath := filepath.Join(root, "env.json")
			if err = os.WriteFile(envPath, b, 0600); err != nil {
				t.Fatal(err)
			}
			raw, err := json.Marshal(map[string]any{"schemaVersion": "2", "kind": "case", "id": "previous-sealers-ready", "chainPreset": env["id"], "requires": []string{"rpc", "consensus"}, "steps": []any{map[string]any{"do": "waitBlock", "target": 12, "timeout": "90s"}, map[string]any{"expect": "blockNumber", "compare": "GreaterOrEqual", "is": "12"}}})
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(root, "case.json")
			if err = os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			specs, err := dsl.ReadFilesWithChainPreset([]string{path}, envPath)
			if err != nil {
				t.Fatal(err)
			}
			out, runErr := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{SpecContent: specs, DataDir: data, ArtifactRoot: filepath.Join(root, "reports"), Binary: bin, KeysDir: keys, ReadOnlyKeys: true, KeepUp: true, WaitBlocks: 2})
			witnesses := []previousSealerWitness{}
			proof := map[string]any{"n": n, "quorum": n - (n-1)/3, "session": out.SessionRoot, "summary": out.Summary, "keyringDir": keys, "scope": "WBFT009 current native Croissant0 five/seven managed validators with generated BLS, no stopped nodes. Three exact parent/child pairs per profile, validators queried at parent, both previous prepared/committed signer sets unique and within parent validators with actual quorum and aggregate format. Independent raw RLP/bitmap validation follows. Not staged migration/mainnet/Linux/full87/legacy/final Web."}
			defer func() {
				proof["witnesses"] = witnesses
				b, e := json.MarshalIndent(proof, "", "  ")
				if e != nil {
					t.Error(e)
					return
				}
				if e = os.WriteFile(filepath.Join(root, "reports/previous-sealers-proof.json"), b, 0600); e != nil {
					t.Error(e)
				}
			}()
			if runErr != nil || out.Summary.Summary.Pass != 1 || out.Summary.Failed() || out.Summary.Summary.Skip != 0 {
				t.Fatal("previous-sealer fixture not ready", runErr, out.Summary)
			}
			state, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: data})
			if err != nil || len(state.Nodes.Nodes) != n {
				t.Fatal("actual validator count differs", err)
			}
			actual := map[string]bool{}
			for _, node := range state.Nodes.Nodes {
				var addr string
				if err := rpc.Dial(node.RPCURL).Call(ctx, "istanbul_nodeAddress", &addr); err != nil {
					t.Fatal(err)
				}
				key := strings.ToLower(addr)
				if !previousSealerHex(key, 20) || actual[key] {
					t.Fatal("invalid actual node identity")
				}
				actual[key] = true
			}
			proof["actualNodeAddresses"] = actual
			c := rpc.Dial(state.Nodes.Nodes[0].RPCURL)
			for {
				h, err := c.BlockNumber(ctx)
				if err != nil {
					t.Fatal(err)
				}
				if h >= 18 {
					break
				}
				select {
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				case <-time.After(200 * time.Millisecond):
				}
			}
			for parent := uint64(13); parent <= 15; parent++ {
				p := previousSealerWitness{ValidatorQueryTag: fmt.Sprintf("0x%x", parent)}
				childTag := fmt.Sprintf("0x%x", parent+1)
				if err := c.Call(ctx, "eth_getBlockByNumber", &p.Parent, p.ValidatorQueryTag, false); err != nil {
					t.Fatal(err)
				}
				if err := c.Call(ctx, "eth_getBlockByNumber", &p.Child, childTag, false); err != nil {
					t.Fatal(err)
				}
				if err := c.Call(ctx, "istanbul_getValidators", &p.ParentValidators, p.ValidatorQueryTag); err != nil {
					t.Fatal(err)
				}
				if err := c.Call(ctx, "istanbul_getWbftExtraInfo", &p.Extra, childTag); err != nil {
					t.Fatal(err)
				}
				if len(p.ParentValidators) != n {
					t.Fatal("parent set count differs")
				}
				for _, addr := range p.ParentValidators {
					if !actual[strings.ToLower(addr)] {
						t.Fatal("parent validator not one of generated live identities")
					}
				}
				for _, field := range []struct {
					name string
					out  **previousSeal
				}{{"prevPreparedSeal", &p.Prepared}, {"prevCommittedSeal", &p.Committed}} {
					b, e := json.Marshal(p.Extra[field.name])
					if e != nil {
						t.Fatal(e)
					}
					if e = json.Unmarshal(b, field.out); e != nil {
						t.Fatal(e)
					}
				}
				witnesses = append(witnesses, p)
				if err := verifyPreviousSealers(p); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
