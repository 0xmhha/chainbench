package anzeon_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/anzeon" // register the cases

	"github.com/0xmhha/chainbench/pkg/accounts"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/attach"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

// minterValidatorMock answers eth_call per selector: isMinter=1,
// minterAllowance=1000, validatorList=one non-zero address, validatorToOperator=
// a non-zero operator word.
func minterValidatorMock(t *testing.T) *httptest.Server {
	t.Helper()
	sel := func(sig string) string { return strings.TrimPrefix(accounts.Selector(sig), "0x") }
	// One-validator dynamic array: offset 0x20, length 1, then the address word.
	validatorHex := "1111111111111111111111111111111111111111"
	arrayRet := "0x" + fmt.Sprintf("%064x", 0x20) + fmt.Sprintf("%064x", 1) +
		strings.Repeat("0", 24) + validatorHex
	operatorWord := "0x" + strings.Repeat("0", 24) + "2222222222222222222222222222222222222222"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.Unmarshal(body, &req)
		var result any = nil
		if req.Method == "eth_call" {
			s := string(body)
			switch {
			case strings.Contains(s, sel("isMinter(address)")):
				result = "0x" + leftpad(1)
			case strings.Contains(s, sel("minterAllowance(address)")):
				result = "0x" + leftpad(1000)
			case strings.Contains(s, sel("validatorList()")):
				result = arrayRet
			case strings.Contains(s, sel("validatorToOperator(address)")):
				result = operatorWord
			default:
				result = "0x" + leftpad(0)
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestMinterValidatorReadCases(t *testing.T) {
	ns, _ := attach.Build("stablenet", "local", []attach.Endpoint{{RPCURL: minterValidatorMock(t).URL}})
	for _, name := range []string{"minter-status-readable", "validator-metadata-readable"} {
		rep, err := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{name}})
		if err != nil {
			t.Fatal(err)
		}
		if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusPass {
			t.Errorf("%s: %+v", name, rep.Results)
		}
	}
}

func TestMinterValidatorReadCases_SkipForeignChain(t *testing.T) {
	ns, _ := attach.Build("wbft", "local", []attach.Endpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"validator-metadata-readable"}})
	if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusSkip {
		t.Fatalf("expected skip on wbft, got %+v", rep.Results)
	}
}
