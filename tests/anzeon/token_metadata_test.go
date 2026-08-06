package anzeon_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	_ "github.com/0xmhha/chainbench/tests/anzeon" // register the case

	"github.com/0xmhha/chainbench/internal/core/pipeline/attach"
	"github.com/0xmhha/chainbench/internal/core/pipeline/testrun"
	"github.com/0xmhha/chainbench/internal/testkit"
)

func TestTokenMetadataCase_Passes(t *testing.T) {
	// ABI encoding of "WKRC": offset 0x20, length 4, "WKRC" padded.
	wkrc := "0x" + strings.Repeat("0", 62) + "20" + strings.Repeat("0", 63) + "4" +
		"574b524300000000000000000000000000000000000000000000000000000000"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req struct {
			ID int `json:"id"`
		}
		_ = json.Unmarshal(body, &req)
		var result any
		if strings.Contains(string(body), "eth_call") {
			result = wkrc
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
	}))
	defer srv.Close()

	ns, _ := attach.Build("stablenet", "local", []attach.Endpoint{{RPCURL: srv.URL}})
	rep, err := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"token-metadata"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusPass {
		t.Fatalf("token-metadata: %+v", rep.Results)
	}
}

func TestTokenMetadataCase_SkipsForeignChain(t *testing.T) {
	ns, _ := attach.Build("wbft", "local", []attach.Endpoint{{RPCURL: "http://x"}})
	rep, _ := testrun.Run(context.Background(), ns, testrun.Options{Names: []string{"token-metadata"}})
	if len(rep.Results) != 1 || rep.Results[0].Status != testkit.StatusSkip {
		t.Fatalf("expected skip on wbft, got %+v", rep.Results)
	}
}
