package interp_test

import (
	"math/big"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/dsl/interp"
)

func TestPreparedPublicValuesCopyNestedReceiptAndExactNumbers(t *testing.T) {
	large := new(big.Int).Lsh(big.NewInt(1), 200)
	value := map[string]any{"receipt": map[string]any{"status": "0x1"}, "items": []any{true, nil, uint64(18446744073709551615), large}}
	copied, err := interp.ClonePreparedValue(value)
	if err != nil {
		t.Fatal(err)
	}
	m := copied.(map[string]any)
	m["receipt"].(map[string]any)["status"] = "0x0"
	m["items"].([]any)[3].(*big.Int).SetInt64(7)
	if value["receipt"].(map[string]any)["status"] != "0x1" || large.BitLen() != 201 || m["items"].([]any)[2] != uint64(18446744073709551615) {
		t.Fatal("public data was mutated or narrowed")
	}
	for _, secret := range []any{[]byte{1, 2, 3}, map[string]any{"privateKey": "private material"}, func() {}} {
		if _, err := interp.ClonePreparedValue(secret); err == nil {
			t.Fatal("private or opaque value became shareable", secret)
		}
	}
}

func TestPreparedExportsPublishOnlyAfterPassAndAllValidate(t *testing.T) {
	for _, pass := range []bool{true, false} {
		reg := interp.NewRegistry()
		reg.RegisterAction("produce", savingAction{value: "0xpublic"})
		reg.RegisterAssertion("check", fakeAssertion{pass: pass})
		out := map[string]any{"old": "stale"}
		spec := dsl.Spec{Prepared: &dsl.PreparedV2{Exports: map[string]string{"address": "$address"}}, PreparedOutputs: out, Steps: []map[string]any{{"produce": map[string]any{"save": "address"}}}, Assertions: []map[string]any{{"assert": "check"}}}
		status, err := interp.NewInterpreter(interp.Deps{Actions: reg}).Run(t.Context(), spec, testEnv(t), &fakeRecord{})
		if err != nil {
			t.Fatal(err)
		}
		if pass && (status != session.StatusPass || out["address"] != "0xpublic" || len(out) != 1) || !pass && (status != session.StatusFail || len(out) != 0) {
			t.Fatal("wrong publication", status, out)
		}
		spec.Prepared.Exports["missing"] = "$notSaved"
		status, err = interp.NewInterpreter(interp.Deps{Actions: reg}).Run(t.Context(), spec, testEnv(t), &fakeRecord{})
		if err != nil || status != session.StatusFail || len(out) != 0 {
			t.Fatal("partial prepared publication escaped", status, err, out)
		}
	}
}
