package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/core/collector"
)

func TestContractsCarryADigestThatSavedDraftsMustMatch(t *testing.T) {
	root := t.TempDir()
	auth, err := app.OpenWebAuth(root, "team")
	if err != nil {
		t.Fatal(err)
	}
	store, err := app.OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	bus := collector.NewBus()
	defer bus.Close()
	s := NewServer(bus, nil, WithChainPresets("../../presets/chain"), WithDeployments(store, WebAuthenticator(auth)), WithWebSecurity(auth, store))
	setup, _ := os.ReadFile(filepath.Join(root, "setup.token"))
	_, token, err := auth.Bootstrap("admin", "long-admin-password", string(setup))
	if err != nil {
		t.Fatal(err)
	}
	session, _ := auth.Session(token)
	call := func(method, path, body string) (int, map[string]any) {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.AddCookie(&http.Cookie{Name: webCookie, Value: token})
		r.Header.Set("X-CSRF-Token", session.CSRFToken)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		var out map[string]any
		_ = json.Unmarshal(w.Body.Bytes(), &out)
		return w.Code, out
	}
	code, contract := call("GET", "/api/v1/contracts/chain-preset?chain=stablenet", "")
	digest, _ := contract["x-contractDigest"].(string)
	if code != 200 || len(digest) != 64 || contract["x-contractVersion"] == nil {
		t.Fatalf("chain-preset contract %d digest %q version %v", code, digest, contract["x-contractVersion"])
	}
	if _, deployment := call("GET", "/api/v1/contracts/deployment", ""); deployment["x-contractDigest"] == digest || deployment["x-contractDigest"] == nil {
		t.Errorf("the deployment contract has no digest of its own: %v", deployment["x-contractDigest"])
	}
	preset := `{"schemaVersion":"2","kind":"chain-preset","id":"p","chain":"stablenet"}`
	var refusal map[string]any
	save := func(d string) int {
		code, body := call("POST", "/api/v1/documents", `{"kind":"chain-preset","name":"p","contractVersion":"2","content":`+preset+`,"assetRefs":[]`+d+`}`)
		refusal = body
		return code
	}
	if c := save(`,"contractDigest":"` + strings.Repeat("0", 64) + `"`); c != 409 {
		t.Errorf("a draft opened against another contract was saved: %d", c)
	}
	// The editor tells this refusal from a revision conflict by a reason it
	// can read, not by the wording of the message.
	if details, _ := refusal["details"].(map[string]any); details["reason"] != "contract_changed" {
		t.Errorf("contract change refusal: %v", refusal)
	}
	if c := save(`,"contractDigest":"` + digest + `"`); c != 201 {
		t.Errorf("a draft opened against the current contract was refused: %d", c)
	}
	if c := save(``); c != 201 {
		t.Errorf("a client that sends no digest was refused: %d", c)
	}
	_, stored := call("GET", "/api/v1/documents?kind=chain-preset", "")
	if strings.Contains(func() string { b, _ := json.Marshal(stored); return string(b) }(), "contractDigest") {
		t.Error("the digest was stored with the document")
	}
}
