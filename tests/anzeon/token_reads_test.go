package anzeon_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/pkg/core/pipeline/attach"
	"github.com/0xmhha/chainbench/pkg/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/pkg/testkit"
)

// selectorMock answers eth_call per function selector: balanceOf=5,
// totalSupply=10 (so totalSupply >= balance), isAuthorized=1.
func selectorMock(t *testing.T) *httptest.Server {
	t.Helper()
	word := func(n int64) string { return "0x" + leftpad(n) }
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
			case strings.Contains(s, "0x70a08231"): // balanceOf
				result = word(5)
			case strings.Contains(s, "0x18160ddd"): // totalSupply
				result = word(10)
			case strings.Contains(s, "0xfe9fbb80"): // isAuthorized
				result = word(1)
			case strings.Contains(s, "0xfe575a87"): // isBlacklisted
				result = word(0)
			default:
				result = word(0)
			}
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func leftpad(n int64) string {
	h := ""
	for n > 0 {
		d := n % 16
		if d < 10 {
			h = string(rune('0'+d)) + h
		} else {
			h = string(rune('a'+d-10)) + h
		}
		n /= 16
	}
	if h == "" {
		h = "0"
	}
	return strings.Repeat("0", 64-len(h)) + h
}

func TestTokenReadCases(t *testing.T) {
	ns, _ := attach.Build("stablenet", "local", []attach.Endpoint{{RPCURL: selectorMock(t).URL}})
	for _, name := range []string{"token-balance-readable", "account-authorization-readable", "account-blacklist-readable"} {
		rep, err := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{name}})
		if err != nil {
			t.Fatal(err)
		}
		if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusPass {
			t.Errorf("%s: %+v", name, rep.Results)
		}
	}
}
