package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAccountList_KeystoreAndRaw(t *testing.T) {
	dir := t.TempDir()
	ks := keyJSON(t, "account", "new", "--out", dir, "--name", "a1", "--store", "keystore", "--password", "pw", "--json")
	raw := keyJSON(t, "account", "new", "--out", dir, "--name", "a2", "--store", "file", "--json")

	out, err := run(t, "account", "list", "--dir", dir, "--json")
	if err != nil {
		t.Fatalf("account list: %v\n%s", err, out)
	}
	var list []struct{ Name, Type, Address string }
	if err := json.Unmarshal([]byte(out), &list); err != nil {
		t.Fatalf("list not JSON: %v\n%s", err, out)
	}
	if len(list) != 2 {
		t.Fatalf("want 2 accounts, got %d: %v", len(list), list)
	}
	byName := map[string]struct{ Type, Address string }{}
	for _, a := range list {
		byName[a.Name] = struct{ Type, Address string }{a.Type, a.Address}
	}
	if byName["a1.json"].Type != "keystore" || !strings.EqualFold(byName["a1.json"].Address, ks["address"]) {
		t.Fatalf("keystore entry wrong: %+v vs %s", byName["a1.json"], ks["address"])
	}
	if byName["a2.key"].Type != "raw" || !strings.EqualFold(byName["a2.key"].Address, raw["address"]) {
		t.Fatalf("raw entry wrong: %+v vs %s", byName["a2.key"], raw["address"])
	}
}

func TestAccountList_RequiresDir(t *testing.T) {
	if _, err := run(t, "account", "list"); err == nil {
		t.Fatal("expected error without --dir")
	}
}

func TestAccountFund_Validation(t *testing.T) {
	// bad amount
	if _, err := run(t, "account", "fund", "--rpc", "http://127.0.0.1:1", "--to", "0xabc", "--amount", "xx", "--private-key", "0x01"); err == nil {
		t.Fatal("expected bad-amount error")
	}
	// no funding source
	if _, err := run(t, "account", "fund", "--rpc", "http://127.0.0.1:1", "--to", "0xabc", "--amount", "1"); err == nil {
		t.Fatal("expected missing-source error")
	}
}
