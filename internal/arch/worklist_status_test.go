package arch

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

func TestWebSummaryMatchesRecordedAcceptance(t *testing.T) {
	b, err := os.ReadFile("../../docs/dev/web-ui-worklist.json")
	if err != nil {
		t.Fatal(err)
	}
	var list struct {
		Items []webWorkItem `json:"items"`
	}
	if err := json.Unmarshal(b, &list); err != nil {
		t.Fatal(err)
	}
	complete := 0
	for _, item := range list.Items {
		if item.Status == "complete" {
			complete++
		}
	}
	b, err = os.ReadFile("../../docs/dev/web-ui-worklist.md")
	if err != nil {
		t.Fatal(err)
	}
	summary := strings.SplitN(string(b), "\n## 단계 기록", 2)[0]
	if !strings.Contains(summary, fmt.Sprintf("완료 %d/%d", complete, len(list.Items))) {
		t.Fatalf("Markdown summary disagrees with recorded acceptance: %d/%d complete", complete, len(list.Items))
	}
	if !strings.Contains(summary, "후속 변경") || !strings.Contains(summary, "3887afccc004727a051e913ce61d937e37a7274cd1036bc3b0372bb3d15a70bc") {
		t.Fatal("summary must distinguish the accepted source from subsequent changes")
	}
}

func TestFeatureWorklistMatchesCoverageCeiling(t *testing.T) {
	file, err := parser.ParseFile(token.NewFileSet(), "../feature/coverage_test.go", nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	ceiling := ""
	ast.Inspect(file, func(node ast.Node) bool {
		value, ok := node.(*ast.ValueSpec)
		if ok && len(value.Names) == 1 && value.Names[0].Name == "unregistered" {
			if literal, ok := value.Values[0].(*ast.BasicLit); ok {
				ceiling = literal.Value
			}
		}
		return true
	})
	if ceiling == "" {
		t.Fatal("coverage ceiling not found")
	}
	if !strings.Contains(openSection(t, worklist(t)), "feature 레지스트리 미등록 "+ceiling+"건") {
		t.Fatal("canonical worklist disagrees with the AST coverage ceiling " + ceiling)
	}
}

func TestWebGapSummaryMatchesRecordedClosures(t *testing.T) {
	b, err := os.ReadFile("../../docs/dev/web-ui-gaps.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Gaps []struct {
			Status string `json:"status"`
		} `json:"gaps"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	closed := 0
	for _, gap := range doc.Gaps {
		if gap.Status == "closed" {
			closed++
		}
	}
	b, err = os.ReadFile("../../docs/dev/web-ui-worklist.md")
	if err != nil {
		t.Fatal(err)
	}
	summary := strings.SplitN(string(b), "\n## 단계 기록", 2)[0]
	if !strings.Contains(summary, fmt.Sprintf("닫힘 %d/%d", closed, len(doc.Gaps))) {
		t.Fatal("gap summary disagrees with recorded closures")
	}
}

func TestBacklogListsAllFourteenWithCompletionCriteria(t *testing.T) {
	doc := worklist(t)
	section := strings.SplitN(doc, "\n## 0a.", 2)
	if len(section) != 2 {
		t.Fatal("missing follow-up worklist")
	}
	section = strings.SplitN(section[1], "\n## 1.", 2)
	for n := 1; n <= 14; n++ {
		prefix := fmt.Sprintf("| %d |", n)
		found := false
		for _, line := range strings.Split(section[0], "\n") {
			if !strings.HasPrefix(line, prefix) {
				continue
			}
			cells := strings.Split(line, "|")
			if found || len(cells) != 6 || strings.TrimSpace(cells[3]) == "" || strings.TrimSpace(cells[4]) == "" {
				t.Fatalf("invalid or duplicate follow-up row %d", n)
			}
			found = true
		}
		if !found {
			t.Errorf("missing follow-up row %d", n)
		}
	}
	if !strings.Contains(section[0], "최종 공통 게이트:") {
		t.Fatal("missing final acceptance gate")
	}
}
