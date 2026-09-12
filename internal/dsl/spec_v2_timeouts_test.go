package dsl

import (
	"strings"
	"testing"
)

// TestTimeouts_AnUnknownKeyIsRefused closes the half of WA15 that was left. The
// value was checked and the NAME was not, so timeouts.step parsed as a valid
// duration and then bound nothing — a spec that looked like it set a budget and
// had none. Only case (and its alias test) is consumed.
func TestTimeouts_AnUnknownKeyIsRefused(t *testing.T) {
	for _, key := range []string{"step", "action", "Case", "assertion"} {
		t.Run(key, func(t *testing.T) {
			_, err := lowerCase(CaseV2{
				ID:       "c",
				Env:      []byte(`{"kind":"env","id":"e","chain":"stablenet"}`),
				Timeouts: map[string]string{key: "30s"},
			})
			if err == nil {
				t.Fatalf("timeouts.%s was accepted; it binds nothing, so it has to be refused", key)
			}
			if !strings.Contains(err.Error(), "not a known timeout") {
				t.Errorf("the error does not say what is wrong: %v", err)
			}
		})
	}
}

// TestTimeouts_TheConsumedKeysAreAccepted keeps the refusal from swallowing the
// vocabulary that works. Both spellings reach interp.caseTimeout.
func TestTimeouts_TheConsumedKeysAreAccepted(t *testing.T) {
	for _, key := range []string{"case", "test"} {
		spec, err := lowerCase(CaseV2{
			ID:  "c",
			Env: []byte(`{"kind":"env","id":"e","chain":"stablenet"}`),
			Steps: []map[string]any{
				{"do": "waitBlock", "target": float64(1)},
				{"expect": "blockNumber", "compare": "GreaterOrEqual", "is": "1"},
			},
			Timeouts: map[string]string{key: "10m"},
		})
		if err != nil {
			t.Fatalf("timeouts.%s was refused: %v", key, err)
		}
		if spec.Timeouts[key] != "10m" {
			t.Errorf("timeouts.%s did not survive lowering: %v", key, spec.Timeouts)
		}
	}
}

// TestTimeouts_AnUnparsableValueIsStillRefused keeps the original half working:
// the name check must not shadow the duration check.
func TestTimeouts_AnUnparsableValueIsStillRefused(t *testing.T) {
	_, err := lowerCase(CaseV2{
		ID:       "c",
		Env:      []byte(`{"kind":"env","id":"e","chain":"stablenet"}`),
		Timeouts: map[string]string{"case": "ten minutes"},
	})
	if err == nil {
		t.Fatal("an unparsable duration was accepted")
	}
	if !strings.Contains(err.Error(), "is not a duration") {
		t.Errorf("the error does not name the problem: %v", err)
	}
}
