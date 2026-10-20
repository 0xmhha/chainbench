package testhelper

import (
	"encoding/json"
	"testing"
)

func TestAccessListResolvesContractReferenceWithoutMutatingDeclaration(t *testing.T) {
	d := deps()
	d.Contracts = map[string]string{"contract": testTo}
	raw := []any{map[string]any{"address": "contract", "storageKeys": []any{}}}
	before, _ := json.Marshal(raw)
	list, err := localAccessList(&d, raw)
	if err != nil || len(list) != 1 || list[0].Address != testTo || list[0].StorageKeys == nil {
		t.Fatalf("reference or explicit empty keys lost: %+v %v", list, err)
	}
	after, _ := json.Marshal(raw)
	if string(before) != string(after) {
		t.Fatal("address resolution changed imported declaration")
	}
	for _, invalid := range []any{
		[]any{map[string]any{"address": "unknown", "storageKeys": []any{}}},
		[]any{map[string]any{"address": testTo}},
		[]any{map[string]any{"address": testTo, "storageKeys": nil}},
	} {
		if _, err := localAccessList(&d, invalid); err == nil {
			t.Fatalf("invalid tuple accepted: %+v", invalid)
		}
	}
}
