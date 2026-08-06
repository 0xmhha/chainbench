package accounts_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/wbft/accounts" // register the case

	"github.com/0xmhha/chainbench/internal/core/pipeline/attach"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

// feeDelegateSignMock answers eth_signRawFeeDelegateTransaction with a
// non-method-not-found error (as a registered method would for a throwaway
// argument): the case passes as long as it is not a -32601 response.
func feeDelegateSignMock(t *testing.T, errCode int, errMsg string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
		}
		_ = json.Unmarshal(body, &req)
		resp := map[string]any{"jsonrpc": "2.0", "id": req.ID}
		if req.Method == "eth_signRawFeeDelegateTransaction" {
			resp["error"] = map[string]any{"code": errCode, "message": errMsg}
		} else {
			resp["result"] = nil
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func runFeeDelegateSign(t *testing.T, url string) testkit.Result {
	t.Helper()
	ns, _ := attach.Build("wbft", "local", []attach.Endpoint{{RPCURL: url}})
	rep, err := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"fee-delegate-sign-rpc-present"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Results) != 1 {
		t.Fatalf("want 1 result, got %+v", rep.Results)
	}
	return rep.Results[0]
}

// A registered method that rejects the throwaway argument (any non -32601 error)
// makes the case pass.
func TestFeeDelegateSignPresent_PassesWhenRegistered(t *testing.T) {
	srv := feeDelegateSignMock(t, -32000, "invalid fee-delegate transaction argument")
	if r := runFeeDelegateSign(t, srv.URL); r.Status != testkit.StatusPass {
		t.Fatalf("expected pass for registered method, got %+v", r)
	}
}

// A -32601 (method not found) response makes the case fail.
func TestFeeDelegateSignPresent_FailsWhenAbsent(t *testing.T) {
	srv := feeDelegateSignMock(t, -32601, "the method eth_signRawFeeDelegateTransaction does not exist/is not available")
	if r := runFeeDelegateSign(t, srv.URL); r.Status != testkit.StatusFail {
		t.Fatalf("expected fail for absent method, got %+v", r)
	}
}

func TestFeeDelegateSignPresent_Registers(t *testing.T) {
	var found bool
	for _, c := range testkit.Cases() {
		if c.Name == "fee-delegate-sign-rpc-present" {
			found = true
		}
	}
	if !found {
		t.Fatal("fee-delegate-sign-rpc-present case not registered on import")
	}
}

func TestFeeDelegateSignPresent_SkipsForeignChain(t *testing.T) {
	ns, _ := attach.Build("wemix", "local", []attach.Endpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"fee-delegate-sign-rpc-present"}})
	if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusSkip {
		t.Fatalf("expected skip on wemix, got %+v", rep.Results)
	}
}
