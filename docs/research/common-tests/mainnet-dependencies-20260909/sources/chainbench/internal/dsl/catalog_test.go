package dsl

import (
	"os"
	"path/filepath"
	"testing"
)

// TestListSpecs pins WA2: the catalog lists runnable cases (with id, chain, and
// description), skips env declarations, and reads the chain from either the v1
// or the inline-v2 form. A case whose env is a reference lists with no chain.
func TestListSpecs(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("a/case-v2.json", `{"schemaVersion":"2","kind":"case","id":"c2","description":"a v2 case",
	  "env":{"chain":"wbft","binaries":{"default":"gwbft"}},"steps":[{"expect":"blockNumber","is":1}]}`)
	write("a/env.json", `{"schemaVersion":"2","kind":"env","id":"e1","chain":"wbft"}`)
	write("b/case-ref.json", `{"schemaVersion":"2","kind":"case","id":"cref","env":"e1","steps":[{"expect":"blockNumber","is":1}]}`)
	write("notjson.txt", `ignore me`)

	got, err := ListSpecs(root)
	if err != nil {
		t.Fatalf("ListSpecs: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("listed %d cases, want 2 (the env is skipped): %+v", len(got), got)
	}
	// Sorted by path: a/case-v2 then b/case-ref.
	if got[0].ID != "c2" || got[0].Chain != "wbft" || got[0].Description != "a v2 case" {
		t.Errorf("v2 inline case = %+v", got[0])
	}
	if got[1].ID != "cref" || got[1].Chain != "" {
		t.Errorf("env-ref case must list with no chain: %+v", got[1])
	}
}
