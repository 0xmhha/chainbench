package capability_test

import (
	"context"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/capability"
)

func TestLoadAndExpose(t *testing.T) {
	jsonl := []byte(`
{"version":"v1","chain":"common","name":"a.one","summary":"common one","params":[{"name":"x","type":"string","required":true}]}
{"version":"v1","chain":"stablenet","name":"b.two","summary":"sn two"}
{"version":"v1","chain":"stablenet","name":"c.nohandler","summary":"unbound"}
`)
	if err := capability.LoadCatalog(jsonl); err != nil {
		t.Fatal(err)
	}
	capability.RegisterHandler("v1", "common", "a.one", func(_ context.Context, args map[string]any) (string, error) {
		return "one:" + capability.ArgString(args, "x", ""), nil
	})
	capability.RegisterHandler("v1", "stablenet", "b.two", func(context.Context, map[string]any) (string, error) {
		return "two", nil
	})

	// Only catalog entries WITH a bound handler are exposed (c.nohandler is not).
	all := capability.All()
	byAddr := map[string]capability.Capability{}
	for _, c := range all {
		byAddr[c.Address()] = c
	}
	if _, ok := byAddr["v1.common.a.one"]; !ok {
		t.Error("a.one should be exposed")
	}
	if _, ok := byAddr["v1.stablenet.c.nohandler"]; ok {
		t.Error("c.nohandler has no handler and must not be exposed")
	}

	// For(chain) = common + that chain.
	sn := capability.For("stablenet")
	var haveCommon, haveSN bool
	for _, c := range sn {
		if c.Address() == "v1.common.a.one" {
			haveCommon = true
		}
		if c.Address() == "v1.stablenet.b.two" {
			haveSN = true
		}
	}
	if !haveCommon || !haveSN {
		t.Errorf("For(stablenet) should include common+stablenet: common=%v sn=%v", haveCommon, haveSN)
	}
	// A different chain must not see stablenet's capability.
	for _, c := range capability.For("wbft") {
		if c.Chain == "stablenet" {
			t.Errorf("wbft must not see stablenet capability %s", c.Address())
		}
	}

	// Lookup + invoke.
	c, ok := capability.Lookup("v1.common.a.one")
	if !ok {
		t.Fatal("lookup a.one failed")
	}
	out, err := c.Handler(context.Background(), map[string]any{"x": "hi"})
	if err != nil || out != "one:hi" {
		t.Errorf("invoke: %q %v", out, err)
	}
}

func TestLoadCatalog_BadLine(t *testing.T) {
	if err := capability.LoadCatalog([]byte(`{"version":"v1","chain":"common"}`)); err == nil {
		t.Error("expected error: missing name")
	}
}
