package dashboard

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	_ "github.com/0xmhha/chainbench/internal/chains/all"
	"github.com/0xmhha/chainbench/internal/core/collector"
)

func TestChainPresetHTTPValidation(t *testing.T) {
	bus := collector.NewBus()
	defer bus.Close()
	server := NewServer(bus, nil, WithChainPresets("../../presets/chain"))
	r := httptest.NewRecorder()
	server.ServeHTTP(r, httptest.NewRequest("GET", "/api/v1/chain-presets", nil))
	if r.Code != 200 {
		t.Fatalf("catalog: %d %s", r.Code, r.Body.String())
	}
	var catalog []struct {
		Chain string `json:"chain"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &catalog); err != nil {
		t.Fatal(err)
	}
	chains := map[string]bool{}
	for _, preset := range catalog {
		chains[preset.Chain] = true
	}
	for _, chain := range []string{"stablenet", "wbft", "wemix"} {
		if !chains[chain] {
			t.Errorf("missing chain %s", chain)
		}
	}
	for _, tc := range []struct {
		content string
		valid   bool
	}{
		{`{"schemaVersion":"2","kind":"chain-preset","id":"x","chain":"wbft","topology":{"bp":4}}`, true},
		{`{"schemaVersion":"2","kind":"chain-preset","id":"x","chain":"typo"}`, false},
		{`{"schemaVersion":"2","kind":"chain-preset","id":"x","chain":"wbft","launch":{"all":{"docroot":"x"}}}`, false},
	} {
		body := `{"kind":"chain-preset","name":"x","contractVersion":"2","content":` + tc.content + `}`
		r := httptest.NewRecorder()
		server.ServeHTTP(r, httptest.NewRequest("POST", "/api/v1/documents/validate", strings.NewReader(body)))
		var out struct {
			Valid bool `json:"valid"`
		}
		if err := json.Unmarshal(r.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		if r.Code != 200 || out.Valid != tc.valid {
			t.Fatalf("response: %d %s", r.Code, r.Body.String())
		}
	}
	r = httptest.NewRecorder()
	server.ServeHTTP(r, httptest.NewRequest("POST", "/api/v1/documents/validate", strings.NewReader(`{} {}`)))
	if r.Code != 400 {
		t.Fatalf("trailing body: %d", r.Code)
	}
}
