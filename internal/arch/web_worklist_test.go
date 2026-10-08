package arch

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
)

type webWorkItem struct {
	ID           string   `json:"id"`
	Status       string   `json:"status"`
	Completion   []string `json:"completion"`
	Verification string   `json:"verification"`
	Evidence     []string `json:"evidence"`
	Red          string   `json:"red"`
	Green        string   `json:"green"`
}

func validateWebWorklist(items []webWorkItem) error {
	if len(items) != 14 {
		return fmt.Errorf("all fourteen acceptance criteria must be tracked")
	}
	seen := map[string]bool{}
	for _, item := range items {
		if seen[item.ID] {
			return fmt.Errorf("duplicate criterion %s", item.ID)
		}
		seen[item.ID] = true
		if len(item.Completion) == 0 || item.Verification == "" {
			return fmt.Errorf("missing completion or verification: %s", item.ID)
		}
		switch item.Status {
		case "pending", "in_progress":
		case "complete":
			if item.Red == "" || item.Green == "" || len(item.Evidence) == 0 {
				return fmt.Errorf("completion requires RED, GREEN and live evidence: %s", item.ID)
			}
			for _, path := range append(append([]string{item.Red, item.Green}, item.Evidence...), "") {
				if path == "" {
					continue
				}
				if info, err := os.Stat("../../" + path); err != nil || !info.Mode().IsRegular() {
					return fmt.Errorf("missing recorded evidence: %s", path)
				}
			}
		default:
			return fmt.Errorf("invalid status: %s", item.Status)
		}
	}
	for n := 1; n <= 14; n++ {
		if !seen[fmt.Sprintf("WEB-%02d", n)] {
			return fmt.Errorf("criterion omitted: WEB-%02d", n)
		}
	}
	return nil
}

func TestWebWorklistTracksEveryAcceptanceCriterion(t *testing.T) {
	b, err := os.ReadFile("../../docs/dev/web-ui-worklist.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Items []webWorkItem `json:"items"`
	}
	if err = json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	if err = validateWebWorklist(doc.Items); err != nil {
		t.Fatal(err)
	}
	seed, err := os.ReadFile("../../docs/dev/architecture/chainbench-web-ui.seed.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range doc.Items {
		if !strings.Contains(string(seed), "description: '"+item.ID+":") {
			t.Errorf("%s is absent from approved Seed", item.ID)
		}
		if !strings.Contains(item.Verification, "--criterion "+item.ID+" --require-live") {
			t.Errorf("live verification missing: %s", item.ID)
		}
	}
	if !strings.Contains(openSection(t, worklist(t)), "web-ui-worklist.json") {
		t.Fatal("Web UI work is absent from the canonical open work list")
	}
}

func TestWebWorklistRejectsOmissionsAndUnsupportedCompletion(t *testing.T) {
	items := make([]webWorkItem, 14)
	for i := range items {
		items[i] = webWorkItem{ID: fmt.Sprintf("WEB-%02d", i+1), Status: "in_progress", Completion: []string{"full scenario coverage"}, Verification: "live command"}
	}
	if err := validateWebWorklist(items); err != nil {
		t.Fatal(err)
	}
	if err := validateWebWorklist(items[:13]); err == nil {
		t.Fatal("missing criterion accepted")
	}
	duplicate := append([]webWorkItem(nil), items...)
	duplicate[13].ID = duplicate[0].ID
	if err := validateWebWorklist(duplicate); err == nil {
		t.Fatal("duplicate accepted")
	}
	items[0].Status = "complete"
	if err := validateWebWorklist(items); err == nil {
		t.Fatal("completion without RED/GREEN/live evidence accepted")
	}
	items[0].Red, items[0].Green, items[0].Evidence = "missing-red.log", "missing-green.log", []string{"missing-result.json"}
	if err := validateWebWorklist(items); err == nil {
		t.Fatal("nonexistent evidence accepted")
	}
}
