package arch

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
)

// webGapsDoc lists what the approved Seed, specification, OpenAPI and screen
// design ask for but the code did not do when every acceptance proof passed.
// A passing proof covers only its own scenarios, so a criterion stays open
// while any gap that names it is open.
const webGapsDoc = "../../docs/dev/web-ui-gaps.json"

type webGap struct {
	ID           string   `json:"id"`
	Group        string   `json:"group"`
	Title        string   `json:"title"`
	Criteria     []string `json:"criteria"`
	Spec         []string `json:"spec"`
	Code         []string `json:"code"`
	Completion   string   `json:"completion"`
	Status       string   `json:"status"`
	Verification string   `json:"verification"`
	Commit       string   `json:"commit"`
}

var (
	webGapID     = regexp.MustCompile(`^G-(A|B|C|D)\d{2}$`)
	webCriterion = regexp.MustCompile(`^WEB-(0[1-9]|1[0-4])$`)
	webGapGroups = map[string]string{"A": "seed", "B": "spec", "C": "design", "D": "defect"}
)

func validateWebGaps(gaps []webGap, items []webWorkItem) error {
	if len(gaps) == 0 {
		return fmt.Errorf("the gap list is empty")
	}
	seen := map[string]bool{}
	open := map[string][]string{}
	for _, g := range gaps {
		m := webGapID.FindStringSubmatch(g.ID)
		if m == nil {
			return fmt.Errorf("malformed gap id %q", g.ID)
		}
		if seen[g.ID] {
			return fmt.Errorf("duplicate gap %s", g.ID)
		}
		seen[g.ID] = true
		if webGapGroups[m[1]] != g.Group {
			return fmt.Errorf("%s belongs to group %s, not %q", g.ID, webGapGroups[m[1]], g.Group)
		}
		if g.Title == "" || g.Completion == "" || len(g.Spec) == 0 || len(g.Code) == 0 || len(g.Criteria) == 0 {
			return fmt.Errorf("%s needs a title, completion test, specification and code reference and criteria", g.ID)
		}
		for _, c := range g.Criteria {
			if !webCriterion.MatchString(c) {
				return fmt.Errorf("%s names unknown criterion %q", g.ID, c)
			}
		}
		switch g.Status {
		case "open":
			for _, c := range g.Criteria {
				open[c] = append(open[c], g.ID)
			}
		case "closed":
			if g.Verification == "" || g.Commit == "" {
				return fmt.Errorf("closing %s requires the verifying command and the commit", g.ID)
			}
		default:
			return fmt.Errorf("%s has invalid status %q", g.ID, g.Status)
		}
	}
	for _, item := range items {
		if item.Status == "complete" && len(open[item.ID]) > 0 {
			return fmt.Errorf("%s is marked complete while gaps %s are open", item.ID, strings.Join(open[item.ID], ", "))
		}
	}
	return nil
}

func TestWebGapsKeepCriteriaOpenUntilClosed(t *testing.T) {
	b, err := os.ReadFile(webGapsDoc)
	if err != nil {
		t.Fatal(err)
	}
	var gaps struct {
		Gaps []webGap `json:"gaps"`
	}
	if err = json.Unmarshal(b, &gaps); err != nil {
		t.Fatal(err)
	}
	w, err := os.ReadFile("../../docs/dev/web-ui-worklist.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Items []webWorkItem `json:"items"`
	}
	if err = json.Unmarshal(w, &doc); err != nil {
		t.Fatal(err)
	}
	if err = validateWebGaps(gaps.Gaps, doc.Items); err != nil {
		t.Fatal(err)
	}
	review, err := os.ReadFile("../../docs/dev/web-ui-gap-review.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range gaps.Gaps {
		if !strings.Contains(string(review), g.ID) {
			t.Errorf("%s is absent from the gap review", g.ID)
		}
	}
	md, err := os.ReadFile("../../docs/dev/web-ui-worklist.md")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(md), "web-ui-gaps.json") {
		t.Error("the Web UI work list does not link the gap list")
	}
}

func TestWebGapsRejectCompletionOverOpenGaps(t *testing.T) {
	items := []webWorkItem{{ID: "WEB-01", Status: "complete"}, {ID: "WEB-02", Status: "in_progress"}}
	gap := webGap{ID: "G-A01", Group: "seed", Title: "t", Criteria: []string{"WEB-01"}, Spec: []string{"s"}, Code: []string{"c"}, Completion: "test", Status: "open"}
	if err := validateWebGaps([]webGap{gap}, items); err == nil {
		t.Fatal("a complete criterion with an open gap was accepted")
	}
	gap.Criteria = []string{"WEB-02"}
	if err := validateWebGaps([]webGap{gap}, items); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []webGap{
		{ID: "G-A01", Group: "spec", Title: "t", Criteria: []string{"WEB-02"}, Spec: []string{"s"}, Code: []string{"c"}, Completion: "x", Status: "open"},
		{ID: "G-B01", Group: "spec", Title: "t", Criteria: []string{"WEB-15"}, Spec: []string{"s"}, Code: []string{"c"}, Completion: "x", Status: "open"},
		{ID: "G-B01", Group: "spec", Title: "t", Criteria: []string{"WEB-02"}, Spec: []string{"s"}, Code: []string{"c"}, Completion: "x", Status: "closed"},
		{ID: "G-B01", Group: "spec", Criteria: []string{"WEB-02"}, Spec: []string{"s"}, Code: []string{"c"}, Completion: "x", Status: "open"},
	} {
		if err := validateWebGaps([]webGap{bad}, items); err == nil {
			t.Fatalf("accepted %+v", bad)
		}
	}
	if err := validateWebGaps([]webGap{gap, gap}, items); err == nil {
		t.Fatal("duplicate gap accepted")
	}
}
