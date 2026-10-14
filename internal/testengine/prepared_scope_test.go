package testengine_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

func TestPreparedScopeCopiesValuesAndBlocksOtherEnvironmentsBeforeBuild(t *testing.T) {
	h := &harness{fpByChain: map[string]session.Fingerprint{"wbft": "same-environment", "wemix": "other-environment"}}
	deps := h.deps(t)
	runs := []string{}
	deps.RunSpec = func(_ context.Context, s dsl.Spec, _ session.Environment, r session.TestRecord) (session.TestStatus, error) {
		runs = append(runs, s.ID)
		if s.ID == "prepare" {
			s.PreparedOutputs["receipt"] = map[string]any{"status": "0x1"}
		} else {
			value := s.PreparedInputs["receipt"].(map[string]any)
			if value["status"] != "0x1" {
				t.Fatal("consumer modified the shared source", value)
			}
			value["status"] = "0x0"
		}
		r.Status(session.StatusPass)
		return session.StatusPass, nil
	}
	raw := func(id, chain string) []byte {
		var s map[string]any
		_ = json.Unmarshal(specJSON(id, chain), &s)
		if id == "prepare" {
			s["prepared"] = map[string]any{"exports": map[string]string{"receipt": "$receipt"}}
		} else {
			s["prepared"] = map[string]any{"imports": map[string]any{"receipt": map[string]string{"case": "prepare", "name": "receipt"}}}
		}
		b, _ := json.Marshal(s)
		return b
	}
	_, err := testengine.New(deps).Run(t.Context(), [][]byte{raw("prepare", "wbft"), raw("first", "wbft"), raw("second", "wbft"), raw("wrong-environment", "wemix")})
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 3 || h.buildCount != 1 {
		t.Fatal("foreign preparation reached environment build or execution", runs, h.buildCount)
	}
}
