//go:build e2e

package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/testengine"
)

var postforkPublicIDs = []string{"fee-delegated-transfer", "secp256r1-precompile-valid", "insufficient-funds-rejected", "fd-feepayer-sig-invalid-rejected", "secp256r1-precompile-short-input", "secp256r1-precompile-invalid"}

func verifyPostforkPublicSummary(s testengine.Summary) error {
	if len(s.Tests) != len(postforkPublicIDs) || s.Summary.Pass != len(postforkPublicIDs) || s.Summary.Fail != 0 || s.Summary.Blocked != 0 || s.Summary.Skip != 0 {
		return fmt.Errorf("not all six successful cases")
	}
	for i, row := range s.Tests {
		if row.Seq != i+1 || row.ID != postforkPublicIDs[i] || row.PresetID != "wbft-bp4" || row.Status != "pass" {
			return fmt.Errorf("case identity/status differs")
		}
	}
	return nil
}

func TestPostforkPublicSummaryRejectsAggregateOnly(t *testing.T) {
	for _, name := range []string{"valid", "missing", "wrong id", "duplicate", "wrong sequence", "wrong preset", "hidden skip", "hidden failure", "blocked count", "extra case"} {
		t.Run(name, func(t *testing.T) {
			s := testengine.Summary{}
			s.Summary.Pass = len(postforkPublicIDs)
			for i, id := range postforkPublicIDs {
				s.Tests = append(s.Tests, testengine.TestOutcome{Seq: i + 1, ID: id, PresetID: "wbft-bp4", Status: "pass"})
			}
			switch name {
			case "missing":
				s.Tests = s.Tests[:4]
			case "wrong id":
				s.Tests[4].ID = "unrelated"
			case "duplicate":
				s.Tests[4].ID = s.Tests[0].ID
			case "wrong sequence":
				s.Tests[4].Seq = 1
			case "wrong preset":
				s.Tests[4].PresetID = "wemix-poa4"
			case "hidden skip":
				s.Tests[4].Status = "skip"
			case "hidden failure":
				s.Tests[4].Status = "fail"
			case "blocked count":
				s.Summary.Blocked = 1
			case "extra case":
				s.Tests = append(s.Tests, s.Tests[0])
			}
			err := verifyPostforkPublicSummary(s)
			if name == "valid" && err != nil {
				t.Fatal(err)
			}
			if name != "valid" && err == nil {
				t.Fatal("aggregate count hid missing or unrelated post-fork witness")
			}
		})
	}
}

func verifyPostforkRecordedSpec(original, resolved, recorded []byte) error {
	var source, input, stored map[string]any
	for _, pair := range []struct {
		raw []byte
		out *map[string]any
	}{{original, &source}, {resolved, &input}, {recorded, &stored}} {
		if err := json.Unmarshal(pair.raw, pair.out); err != nil {
			return err
		}
	}
	if source["id"] != input["id"] || !reflect.DeepEqual(source["steps"], input["steps"]) {
		return fmt.Errorf("executed public input changed")
	}
	var expected map[string]any
	if err := json.Unmarshal(session.Scrub(resolved), &expected); err != nil {
		return err
	}
	if !reflect.DeepEqual(expected, stored) {
		return fmt.Errorf("stored public input changed")
	}
	return nil
}

func TestPostforkRecordedSpecPreservesScrubbingAndRejectsChangedInputs(t *testing.T) {
	original := []byte(`{"id":"public-case","steps":[{"do":"sendTx","key":"$senderKey","saveKey":"senderKey","to":"node2"}]}`)
	resolved := []byte(`{"id":"public-case","chainPreset":{"id":"wbft-bp4"},"steps":[{"do":"sendTx","key":"$senderKey","saveKey":"senderKey","to":"node2"}]}`)
	recorded := session.Scrub(resolved)
	if err := verifyPostforkRecordedSpec(original, resolved, recorded); err != nil {
		t.Fatal("expected secret scrubbing rejected", err)
	}
	for _, name := range []string{"input key binding", "input target", "stored target", "stored missing step", "stored identity"} {
		t.Run(name, func(t *testing.T) {
			var input, stored map[string]any
			if err := json.Unmarshal(resolved, &input); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(recorded, &stored); err != nil {
				t.Fatal(err)
			}
			switch name {
			case "input key binding":
				input["steps"].([]any)[0].(map[string]any)["key"] = "$otherKey"
			case "input target":
				input["steps"].([]any)[0].(map[string]any)["to"] = "node3"
			case "stored target":
				stored["steps"].([]any)[0].(map[string]any)["to"] = "node3"
			case "stored missing step":
				stored["steps"] = []any{}
			case "stored identity":
				stored["id"] = "unrelated"
			}
			a, _ := json.Marshal(input)
			b, _ := json.Marshal(stored)
			if verifyPostforkRecordedSpec(original, a, b) == nil {
				t.Fatal("changed public input or stored action accepted")
			}
		})
	}
}

// A complete post-fork report must include the approved invalid-signature
// result, alongside the genuine valid-signature control, not just old totals.
func TestPostforkPublicSummaryRequiresApprovedInvalidSignatureWitness(t *testing.T) {
	s := testengine.Summary{}
	ids := []string{"fee-delegated-transfer", "secp256r1-precompile-valid", "insufficient-funds-rejected", "fd-feepayer-sig-invalid-rejected", "secp256r1-precompile-short-input"}
	for i, id := range ids {
		s.Tests = append(s.Tests, testengine.TestOutcome{Seq: i + 1, ID: id, PresetID: "wbft-bp4", Status: "pass"})
	}
	s.Summary.Pass = 5
	if verifyPostforkPublicSummary(s) == nil {
		t.Error("old five-case report omits approved TX019 witness")
	}
	s.Tests = append(s.Tests, testengine.TestOutcome{Seq: 6, ID: "secp256r1-precompile-invalid", PresetID: "wbft-bp4", Status: "pass"})
	s.Summary.Pass = 6
	if err := verifyPostforkPublicSummary(s); err != nil {
		t.Fatal("complete six-case report rejected", err)
	}
}
