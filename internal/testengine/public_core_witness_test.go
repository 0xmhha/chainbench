package testengine_test

import (
	"fmt"
	"testing"

	"github.com/0xmhha/chainbench/internal/testengine"
)

var publicCoreIDs = []string{"basic-rpc-health", "transaction-count-increments", "genesis-balance", "gas-price-positive", "contract-roundtrip", "storage-write-and-read", "view-call-leaves-state"}

func verifyPublicCoreWitness(preset string, s testengine.Summary) error {
	if preset == "" || len(s.Tests) != len(publicCoreIDs) || s.Summary.Pass != len(publicCoreIDs) || s.Summary.Fail != 0 || s.Summary.Blocked != 0 || s.Summary.Skip != 0 {
		return fmt.Errorf("not exact seven successful cases")
	}
	seen := map[string]bool{}
	for i, row := range s.Tests {
		if row.Seq != i+1 || row.ID != publicCoreIDs[i] || seen[row.ID] || row.PresetID != preset || row.Status != "pass" {
			return fmt.Errorf("case identity/status differs")
		}
		seen[row.ID] = true
	}
	return nil
}

func TestPublicCoreWitnessRejectsAggregateOnlyResults(t *testing.T) {
	for _, name := range []string{"valid", "missing case", "wrong id", "duplicate id", "wrong sequence", "wrong preset", "hidden skipped case", "hidden failed case", "extra skipped count"} {
		t.Run(name, func(t *testing.T) {
			s := testengine.Summary{}
			s.Summary.Pass = 7
			for i, id := range publicCoreIDs {
				s.Tests = append(s.Tests, testengine.TestOutcome{Seq: i + 1, ID: id, PresetID: "wbft-bp4-en1", Status: "pass"})
			}
			switch name {
			case "missing case":
				s.Tests = s.Tests[:6]
			case "wrong id":
				s.Tests[6].ID = "unrelated"
			case "duplicate id":
				s.Tests[6].ID = s.Tests[0].ID
			case "wrong sequence":
				s.Tests[6].Seq = 1
			case "wrong preset":
				s.Tests[6].PresetID = "wbft-bp4"
			case "hidden skipped case":
				s.Tests[6].Status = "skip"
			case "hidden failed case":
				s.Tests[6].Status = "fail"
			case "extra skipped count":
				s.Summary.Skip = 1
			}
			e := verifyPublicCoreWitness("wbft-bp4-en1", s)
			if name == "valid" && e != nil {
				t.Fatal(e)
			}
			if name != "valid" && e == nil {
				t.Fatal("aggregate count hid an incomplete or unrelated witness", name)
			}
		})
	}
}
