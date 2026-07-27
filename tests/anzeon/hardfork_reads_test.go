package anzeon_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/anzeon" // register the cases

	"github.com/0xmhha/chainbench/pkg/core/pipeline/attach"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

// validVector is the valid P-256 test vector (hash||r||s||x||y, no 0x prefix) the
// hardfork mock recognizes as the one input that verifies.
const validVector = "bb5a52f42f9c9261ed4361f59422a1e30036e7c32b270c8807a419feca605023" +
	"2927b10512bae3eddcfe467828128bad2903269919f7086069c8c4df6c732838" +
	"c7787964eaac00e5921fb1498a60f4606766b3d9685001558d1a974e7341513e" +
	"04e04e18e1ff7b70e7b5e14d1b70e0bdb8ece3acf34ffee3e8e5a2e4266bfbb0" +
	"f6afd7ebfa4dfddd60ab0272c226d19c1f6aed1cdee3a51a35e415f4dcc33d70"

// hardforkMock answers the h-hardfork post-fork state reads with values that make
// the ported cases pass: the P-256 precompile returns the success word ONLY for
// the exact valid vector (corrupted/short inputs get a zero word), GovMinter has
// substantial code, and the chain reports a positive block number, chain id, and
// baseFeePerGas.
func hardforkMock(t *testing.T) *httptest.Server {
	t.Helper()
	zeroWord := "0x" + strings.Repeat("0", 64)
	successWord := "0x" + strings.Repeat("0", 63) + "1"
	// A non-trivial (> 100 hex char) fake bytecode stands in for the v2 contract.
	govCode := "0x60806040" + strings.Repeat("ab", 200)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		s := string(body)
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.Unmarshal(body, &req)
		var result any
		switch req.Method {
		case "eth_call":
			if strings.Contains(s, validVector) {
				result = successWord
			} else {
				result = zeroWord
			}
		case "eth_getCode":
			result = govCode
		case "eth_blockNumber":
			result = "0x2a" // 42 > 0
		case "eth_chainId":
			result = "0x2328" // 9000 > 0
		case "eth_getBlockByNumber":
			result = map[string]any{"number": "0x2a", "baseFeePerGas": "0x1234"}
		default:
			result = nil
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestHardforkReadCases(t *testing.T) {
	ns, _ := attach.Build("stablenet", "local", []attach.Endpoint{{RPCURL: hardforkMock(t).URL}})
	for _, name := range []string{
		"p256-precompile-active",
		"p256-rejects-invalid",
		"govminter-v2-code",
		"boho-chain-config-active",
	} {
		rep, err := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{name}})
		if err != nil {
			t.Fatal(err)
		}
		if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusPass {
			t.Errorf("%s: %+v", name, rep.Results)
		}
	}
}

func TestHardforkReadCases_SkipForeignChain(t *testing.T) {
	ns, _ := attach.Build("wbft", "local", []attach.Endpoint{{RPCURL: "http://x"}})
	names := []string{"p256-precompile-active", "govminter-v2-code", "boho-chain-config-active"}
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: names})
	if len(rep.Results) != len(names) {
		t.Fatalf("ran %d cases, want %d", len(rep.Results), len(names))
	}
	for _, res := range rep.Results {
		if res.Status != testkit.StatusSkip {
			t.Errorf("%s: status %s, want skip on wbft", res.Name, res.Status)
		}
	}
}
