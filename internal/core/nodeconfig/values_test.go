package nodeconfig

import "testing"

func TestResolve_Precedence(t *testing.T) {
	file := Values{"nodes.validators": "2", "ports.base_http": "9000"}
	override := Values{"ports.base_http": "9999"}
	got := Resolve(file, override)

	// override wins over file
	if got.Int("ports.base_http", 0) != 9999 {
		t.Errorf("base_http: got %d, want 9999 (override wins)", got.Int("ports.base_http", 0))
	}
	// file wins over default
	if got.Int("nodes.validators", 0) != 2 {
		t.Errorf("validators: got %d, want 2 (file wins over default)", got.Int("nodes.validators", 0))
	}
	// default when neither file nor override set
	if got.String("chain", "") != "stablenet" {
		t.Errorf("chain: got %q, want default stablenet", got.String("chain", ""))
	}
	if got.Int("ports.base_p2p", 0) != 30301 {
		t.Errorf("base_p2p: got %d, want default 30301", got.Int("ports.base_p2p", 0))
	}
}

func TestMerge_DoesNotMutateInputs(t *testing.T) {
	a := Values{"k": "1"}
	b := Values{"k": "2"}
	_ = Merge(a, b)
	if a["k"] != "1" || b["k"] != "2" {
		t.Error("Merge mutated an input layer")
	}
}

func TestFlatten(t *testing.T) {
	nested := map[string]any{
		"chain": "wbft",
		"ports": map[string]any{
			"base_http": float64(8501), // JSON number
			"base_p2p":  30301,         // int
		},
		"logging": map[string]any{"rotation": true},
		"nodes":   map[string]any{"extra_flags": []any{"--x"}}, // slice skipped
		"empty":   nil,                                         // skipped
	}
	v := Flatten(nested)

	if v.String("chain", "") != "wbft" {
		t.Errorf("chain: got %q", v.String("chain", ""))
	}
	if v.String("ports.base_http", "") != "8501" {
		t.Errorf("base_http: got %q, want 8501 (no .0)", v.String("ports.base_http", ""))
	}
	if v.Int("ports.base_p2p", 0) != 30301 {
		t.Errorf("base_p2p: got %d", v.Int("ports.base_p2p", 0))
	}
	if !v.Bool("logging.rotation", false) {
		t.Error("logging.rotation: want true")
	}
	if _, ok := v["nodes.extra_flags"]; ok {
		t.Error("slice value should be skipped")
	}
	if _, ok := v["empty"]; ok {
		t.Error("nil value should be skipped")
	}
}

func TestTypedGetters_Fallback(t *testing.T) {
	v := Values{"n": "notanint", "b": "maybe"}
	if v.Int("n", 7) != 7 {
		t.Error("Int should fall back on unparseable value")
	}
	if v.Int("missing", 7) != 7 {
		t.Error("Int should fall back on missing key")
	}
	if v.Bool("b", true) != true {
		t.Error("Bool should fall back on unrecognized value")
	}
}
