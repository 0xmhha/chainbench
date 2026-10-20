package testengine_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
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
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
	"golang.org/x/crypto/sha3"
)

func TestNativeTamperSubmissionRejections(t *testing.T) {
	families := []struct{ chain, variable string }{{"stablenet", "GSTABLE_BIN"}, {"wbft", "GWBFT_BIN"}, {"wemix", "GWEMIX_BIN"}}
	for _, f := range families {
		if os.Getenv(f.variable) == "" {
			t.Skip("native tamper proof requires " + f.variable)
		}
	}
	for _, f := range families {
		if !t.Run(f.chain, func(t *testing.T) {
			root := failureRuntime(t)
			ctx, cancel := context.WithTimeout(t.Context(), 8*time.Minute)
			defer cancel()
			var specs [][]byte
			for _, name := range []string{"CT-TX-006-fd-sender-sig-tampered-rejected.json", "CT-TX-007-fd-feepayer-sig-tampered-rejected.json"} {
				path := filepath.Join(repoRoot(t), "tests/tc/common/tx", name)
				b, e := os.ReadFile(path)
				if e != nil {
					t.Fatal(e)
				}
				if e = os.WriteFile(filepath.Join(root, name), b, 0600); e != nil {
					t.Fatal(e)
				}
				raw, e := dsl.ReadFilesWithChainPreset([]string{path}, f.chain+"-bp4")
				if e != nil {
					t.Fatal(e)
				}
				specs = append(specs, raw...)
			}
			out, runErr := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{SpecContent: specs, DataDir: filepath.Join(root, "w"), ArtifactRoot: filepath.Join(root, "reports"), Binary: os.Getenv(f.variable), KeysDir: filepath.Join(repoRoot(t), "presets/keys"), ReadOnlyKeys: true, KeepUp: true, WaitBlocks: 2, LaunchOpts: []string{"cache=64"}})
			proof := map[string]any{"chain": f.chain, "binary": os.Getenv(f.variable), "session": out.SessionRoot, "summary": out.Summary, "scope": "Two unchanged public locally-signed tamper cases on funded accounts, plus independent raw node rejection witnesses on fresh unfunded accounts. No personal signing API, whole R5/L1/legacy/87 or final Web acceptance."}
			defer func() {
				b, e := json.MarshalIndent(proof, "", "  ")
				if e != nil {
					t.Error(e)
					return
				}
				if e = os.WriteFile(filepath.Join(root, "reports/tamper-rejection-proof.json"), b, 0600); e != nil {
					t.Error(e)
				}
			}()
			if runErr != nil {
				proof["runError"] = runErr.Error()
				t.Fatal(runErr)
			}
			if out.Summary.Summary.Pass != 2 || out.Summary.Failed() || out.Summary.Summary.Skip != 0 || len(out.Summary.Tests) != 2 {
				t.Fatal("expected exactly two public passes", out.Summary)
			}
			state, e := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: filepath.Join(root, "w")})
			if e != nil || len(state.Nodes.Nodes) != 4 {
				t.Fatal("missing actual BP4", e)
			}
			client := rpc.Dial(state.Nodes.Nodes[1].RPCURL)
			chainID, e := client.ChainID(ctx)
			if e != nil {
				t.Fatal(e)
			}
			head, e := client.BlockByNumber(ctx, "latest")
			if e != nil || head.BaseFeePerGas == nil {
				t.Fatal("missing actual base fee", e)
			}
			var tipHex string
			if e = client.Call(ctx, "eth_maxPriorityFeePerGas", &tipHex); e != nil {
				t.Fatal(e)
			}
			tip, ok := new(big.Int).SetString(strings.TrimPrefix(tipHex, "0x"), 16)
			if !ok || tip.Sign() < 0 {
				t.Fatal("invalid actual tip", tipHex)
			}
			cap := new(big.Int).Add(new(big.Int).Mul(head.BaseFeePerGas, big.NewInt(2)), tip)
			var witnesses []map[string]any
			for _, which := range []string{"sender", "feepayer"} {
				sender, _, e := accounts.GenerateKey()
				if e != nil {
					t.Fatal(e)
				}
				payer, _, e := accounts.GenerateKey()
				if e != nil {
					t.Fatal(e)
				}
				raw, e := accounts.EncodeFeeDelegatedTampered(sender, payer, "0x00000000000000000000000000000000C0FFEE20", big.NewInt(1), int64(chainID), 0, cap, tip, which)
				if e != nil {
					t.Fatal(e)
				}
				h := sha3.NewLegacyKeccak256()
				_, _ = h.Write(raw)
				hash := fmt.Sprintf("0x%x", h.Sum(nil))
				encoded := "0x" + hex.EncodeToString(raw)
				var submitted string
				submitErr := client.Call(ctx, "eth_sendRawTransaction", &submitted, encoded)
				var rejection *rpc.ServerError
				if submitted != "" || !errors.As(submitErr, &rejection) || rejection == nil || rejection.Method != "eth_sendRawTransaction" || rejection.Code != -32000 || rejection.Message != "fee delegation: invalid feePayer" {
					t.Fatal("expected exact actual signature rejection", submitted, submitErr)
				}
				var tx, receipt map[string]any
				if e = client.Call(ctx, "eth_getTransactionByHash", &tx, hash); e != nil {
					t.Fatal(e)
				}
				if e = client.Call(ctx, "eth_getTransactionReceipt", &receipt, hash); e != nil {
					t.Fatal(e)
				}
				if tx != nil || receipt != nil {
					t.Fatal("rejected raw bytes have a transaction or receipt", hash)
				}
				witnesses = append(witnesses, map[string]any{"which": which, "raw": encoded, "hash": hash, "returnedHash": submitted, "method": rejection.Method, "code": rejection.Code, "message": rejection.Message, "transaction": tx, "receipt": receipt})
			}
			proof["witnesses"] = witnesses
		}) {
			t.Fatal("stop after preceding proof or cleanup failed")
		}
	}
}
