package types

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNetwork_RoundtripLocalFixture(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "schema", "fixtures", "network-local.json"))
	if err != nil {
		t.Fatal(err)
	}
	var n Network
	if err := json.Unmarshal(data, &n); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	out, err := json.Marshal(n)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got, want map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}

	// Compare semantic equality via JSON re-marshal.
	gotJSON, _ := json.Marshal(got)
	wantJSON, _ := json.Marshal(want)
	if string(gotJSON) != string(wantJSON) {
		t.Fatalf("roundtrip lost data\n got:  %s\n want: %s", gotJSON, wantJSON)
	}
}
