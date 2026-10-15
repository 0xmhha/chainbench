package testengine_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/preset"
	"github.com/0xmhha/chainbench/internal/testengine"
)

func TestNativeIdentityEpochGenesisQueries(t *testing.T) {
	bin := os.Getenv("GWBFT_BIN")
	if bin == "" {
		t.Skip("set GWBFT_BIN for actual BP/EN identity and epoch queries")
	}
	root := failureRuntime(t)
	ctx, cancel := context.WithTimeout(t.Context(), 6*time.Minute)
	defer cancel()
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "presets/chain/wbft-bp4-en1.json"))
	if err != nil {
		t.Fatal(err)
	}
	var env map[string]any
	if err = json.Unmarshal(b, &env); err != nil {
		t.Fatal(err)
	}
	env["capabilities"] = []string{"rpc", "consensus"}
	env["genesis"] = map[string]any{"set": map[string]any{"config.croissant.wBFT.epochLength": 10, "config.croissant.wBFT.targetValidators": 4}}
	b, err = json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	envPath := filepath.Join(root, "env.json")
	if err = os.WriteFile(envPath, b, 0600); err != nil {
		t.Fatal(err)
	}
	cases := []string{"tests/tc/go-stablenet/regression/api/11-node-address-returned.json", "tests/tc/go-stablenet/regression/api/13-commit-signers-quorum.json", "tests/tc/go-stablenet/regression/api/16-is-validator-flags.json", "tests/tc/common/node/CT-NODE-003-genesis-init.json"}
	raws := [][]byte{}
	for i, p := range cases {
		raw, e := dsl.ReadFilesWithChainPreset([]string{filepath.Join(repoRoot(t), p)}, envPath)
		if e != nil {
			t.Fatal(e)
		}
		if i == 0 {
			var spec map[string]any
			if e = json.Unmarshal(raw[0], &spec); e != nil {
				t.Fatal(e)
			}
			spec["steps"] = append([]any{map[string]any{"do": "waitBlock", "target": 21, "timeout": "180s"}}, spec["steps"].([]any)...)
			raw[0], e = json.Marshal(spec)
			if e != nil {
				t.Fatal(e)
			}
		}
		if e = os.WriteFile(filepath.Join(root, fmt.Sprintf("case%d.json", i+1)), raw[0], 0600); e != nil {
			t.Fatal(e)
		}
		raws = append(raws, raw...)
	}
	out, runErr := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{SpecContent: raws, DataDir: filepath.Join(root, "w"), ArtifactRoot: filepath.Join(root, "reports"), Binary: bin, KeysDir: filepath.Join(repoRoot(t), "presets/keys"), ReadOnlyKeys: true, KeepUp: true, WaitBlocks: 2})
	observed := map[string]any{"binary": bin, "session": out.SessionRoot, "summary": out.Summary, "scope": "RPC003/004/010/011/013 current WBFT Croissant0 managed BP4 EN1, epoch10. Four public cases plus tagged epoch/quorum/genesis/identity oracle. Not full87/Linux/mainnet/legacy/staged handoff/final Web."}
	defer func() {
		b, e := json.MarshalIndent(observed, "", "  ")
		if e != nil {
			t.Error(e)
			return
		}
		if e = os.WriteFile(filepath.Join(root, "reports/identity-epoch-proof.json"), b, 0600); e != nil {
			t.Error(e)
		}
	}()
	if runErr != nil || out.Summary.Summary.Pass != 4 || out.Summary.Failed() || out.Summary.Summary.Skip != 0 {
		t.Fatalf("public identity queries failed: %v %+v", runErr, out.Summary)
	}
	state, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: filepath.Join(root, "w")})
	if err != nil || len(state.Nodes.Nodes) != 5 {
		t.Fatal("not actual BP4 EN1", err)
	}
	p := identityEpochProof{}
	var genesisBytes []byte
	err = filepath.WalkDir(filepath.Join(root, "w"), func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() && (d.Name() == "chaindata" || d.Name() == "lightchaindata" || d.Name() == "etcd") {
			return filepath.SkipDir
		}
		if d.IsDir() || d.Name() != "genesis.json" {
			return nil
		}
		raw, e := os.ReadFile(path)
		if e != nil {
			return e
		}
		if genesisBytes != nil && !bytes.Equal(genesisBytes, raw) {
			return fmt.Errorf("generated genesis copies differ")
		}
		genesisBytes = raw
		p.GenesisPath = path
		return nil
	})
	if err != nil || genesisBytes == nil {
		t.Fatal("generated genesis missing/different", err)
	}
	var g struct {
		Config struct {
			ChainID        json.Number `json:"chainId"`
			CroissantBlock uint64      `json:"croissantBlock"`
			Croissant      struct {
				WBFT struct {
					EpochLength uint64 `json:"epochLength"`
					Target      int    `json:"targetValidators"`
				} `json:"wBFT"`
			} `json:"croissant"`
		} `json:"config"`
	}
	if err = json.Unmarshal(genesisBytes, &g); err != nil {
		t.Fatal(err)
	}
	p.GenesisChainID = g.Config.ChainID.String()
	p.Fork = g.Config.CroissantBlock
	p.EpochLength = g.Config.Croissant.WBFT.EpochLength
	p.Target = g.Config.Croissant.WBFT.Target
	keys, err := preset.LoadKeyPreset(filepath.Join(repoRoot(t), "presets/keys"))
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range state.Nodes.Nodes {
		key, ok := keys.Node(n.Index)
		if !ok {
			t.Fatal("missing expected public identity", n.Index)
		}
		row := identityNode{Index: n.Index, Role: string(n.Role), ExpectedAddress: key.Address, PublicKey: key.PublicKey}
		c := rpc.Dial(n.RPCURL)
		ready, cancelReady := context.WithTimeout(ctx, 90*time.Second)
		for {
			head, e := c.BlockNumber(ready)
			if e == nil && head >= 21 {
				break
			}
			select {
			case <-ready.Done():
				cancelReady()
				t.Fatal("node did not reach queried epoch", n.Index, e)
			case <-time.After(200 * time.Millisecond):
			}
		}
		cancelReady()
		if err = c.Call(ctx, "istanbul_nodeAddress", &row.Address); err != nil {
			t.Fatal(err)
		}
		if err = c.Call(ctx, "eth_chainId", &row.ChainID); err != nil {
			t.Fatal(err)
		}
		if err = c.Call(ctx, "istanbul_isValidator", &row.Validator, "0x15"); err != nil {
			t.Fatal(err)
		}
		var info struct{ ID, Enode string }
		if err = c.Call(ctx, "admin_nodeInfo", &info); err != nil {
			t.Fatal(err)
		}
		u, e := url.Parse(info.Enode)
		if e != nil || u.User == nil {
			t.Fatal("missing actual node enode", e)
		}
		row.AdminID = info.ID
		row.AdminPublicKey = u.User.Username()
		p.Nodes = append(p.Nodes, row)
	}
	c := rpc.Dial(state.Nodes.Nodes[0].RPCURL)
	for _, n := range []uint64{9, 10, 11, 19, 20, 21} {
		tag := fmt.Sprintf("0x%x", n)
		q := epochQuery{At: n}
		if err = c.Call(ctx, "istanbul_getValidators", &q.Validators, tag); err != nil {
			t.Fatal(err)
		}
		var signers struct {
			Number       uint64
			Hash, Author string
			Committers   []string
		}
		if err = c.Call(ctx, "istanbul_getCommitSignersFromBlock", &signers, tag); err != nil {
			t.Fatal(err)
		}
		q.SignerNumber = signers.Number
		q.SignerHash = signers.Hash
		q.Author = signers.Author
		q.Signers = signers.Committers
		var header struct{ Number, Hash, ParentHash, Miner string }
		if err = c.Call(ctx, "eth_getBlockByNumber", &header, tag, false); err != nil || header.Number != tag {
			t.Fatal("wrong tagged header", err, header)
		}
		q.Hash = header.Hash
		q.HeaderAuthor = header.Miner
		q.ParentHash = header.ParentHash
		p.Queries = append(p.Queries, q)
	}
	observed["proof"] = p
	observed["publicCases"] = cases
	observed["keyMetadata"] = filepath.Join(repoRoot(t), "presets/keys/metadata.json")
	if err = verifyIdentityEpoch(p); err != nil {
		t.Fatal(err)
	}
}
