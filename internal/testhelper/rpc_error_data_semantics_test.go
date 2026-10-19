package testhelper

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRPCErrorRequiresExactCustomErrorData(t *testing.T) {
	const selector = "0xe49c14cb" // keccak256("NoRefundAvailable()")[:4]
	for _, tc := range []struct {
		name          string
		data          any
		present, pass bool
	}{
		{"exact selector", selector, true, true}, {"wrong custom error", "0xdeadbeef", true, false}, {"empty bytes", "0x", true, false}, {"truncated selector", "0xe49c14", true, false}, {"missing data", nil, false, false}, {"null data", nil, true, false}, {"object data", map[string]any{"data": selector}, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var q struct{ ID any }
				if e := json.NewDecoder(r.Body).Decode(&q); e != nil {
					t.Error(e)
					return
				}
				rpcErr := map[string]any{"code": 3, "message": "execution reverted"}
				if tc.present {
					rpcErr["data"] = tc.data
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": q.ID, "error": rpcErr})
			}))
			defer srv.Close()
			got, e := checkRPCError(t, srv.URL, map[string]any{"method": "eth_call", "reason": "execution reverted", "errorData": selector})
			if e != nil || got.Pass != tc.pass {
				t.Fatalf("wrong custom-error verdict: pass=%v want=%v error=%v actual=%v", got.Pass, tc.pass, e, got.Actual)
			}
		})
	}
}

func TestRPCErrorDataAppearsInSharedEditorVocabulary(t *testing.T) {
	_, schemas, err := EditorVocabulary()
	if err != nil {
		t.Fatal(err)
	}
	schema := schemas["assertion-rpcError"]
	props := schema["properties"].(map[string]any)
	field, ok := props["errorData"].(map[string]any)
	if !ok {
		t.Fatal("exact error data cannot be authored through the shared vocabulary")
	}
	if field["type"] != "string" || field["pattern"] == nil {
		t.Fatal("error data lacks its engine-owned string/hex contract")
	}
	foundReason := false
	for _, name := range schema["required"].([]string) {
		foundReason = foundReason || name == "reason"
	}
	if !foundReason {
		t.Fatal("runtime requires reason but editor does not")
	}
}
