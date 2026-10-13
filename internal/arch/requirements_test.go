package arch

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"
)

// directionDoc owns the product requirements; §12 is the traceability table.
const directionDoc = "../../docs/dev/chainbench-system-direction.md"

// The requirements are the product's contract, and until now nothing connected
// them to the code. Grepping for R01..R19 across docs and internal found only the
// document that declares them — nineteen statements about what chainbench must do,
// with no way to tell whether any of them still held.
//
// Walking that table by hand found a real gap on the first pass (R10 preserved a
// config revision by overwriting the previous one), which is the argument for the
// walk being mechanical from here: a table nobody checks drifts the same way a
// checkbox nobody re-reads does, and this repository has now spent several rounds
// on exactly that failure.
//
// So the table names a test per requirement, and this ratchet holds the naming
// honest. It deliberately does NOT judge whether the test proves the requirement —
// no test can decide that, and pretending otherwise would be the vacuous check
// this work keeps removing. It checks the two things a machine can: every
// requirement names evidence, and every named test exists.

var requirementRow = regexp.MustCompile(`^\|\s*(R\d\d)\s*\|(.*)$`)

// requirementEvidence parses §12 into requirement id -> named test.
func requirementEvidence(t *testing.T) map[string]string {
	t.Helper()
	b, err := os.ReadFile(filepath.Clean(directionDoc))
	if err != nil {
		t.Fatalf("read %s: %v", directionDoc, err)
	}
	out := map[string]string{}
	for _, line := range strings.Split(string(b), "\n") {
		m := requirementRow.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		cells := strings.Split(m[2], "|")
		// id | requirement | section | work | evidence
		if len(cells) < 4 {
			t.Errorf("%s: row has %d cells after the id, want at least 4 — the evidence column is missing", m[1], len(cells))
			continue
		}
		out[m[1]] = strings.TrimSpace(strings.Trim(strings.TrimSpace(cells[3]), "`"))
	}
	if len(out) == 0 {
		t.Fatalf("%s has no R-numbered rows, so the parse is wrong rather than the table empty", directionDoc)
	}
	return out
}

// TestEveryRequirementNamesEvidence is the first half: a requirement with an empty
// evidence cell is one nobody has connected to the code, and it must say so by
// failing rather than by sitting in a table looking complete.
func TestEveryRequirementNamesEvidence(t *testing.T) {
	ev := requirementEvidence(t)
	if len(ev) < 19 {
		t.Errorf("parsed %d requirements; the table declared nineteen — a row was dropped or the format changed", len(ev))
	}
	for id, name := range ev {
		if name == "" {
			t.Errorf("%s names no evidence — give it a test, or say in the document why it cannot have one", id)
		}
	}
}

// TestEveryNamedTestExists is the half that makes the first one worth having. A
// named test that was renamed or deleted leaves the table looking answered while
// answering nothing, which is the exact shape of every stale record this work has
// had to correct.
func TestEveryNamedTestExists(t *testing.T) {
	ev := requirementEvidence(t)
	have := declaredTests(t)
	for id, name := range ev {
		if name == "" {
			continue // the other test reports this
		}
		if !have[name] {
			t.Errorf("%s names %s, which no test declares — it was renamed or removed, so the requirement is unheld", id, name)
		}
	}
}

// declaredTests parses Go declarations, including gated tests. Existence is
// distinct from execution: the sufficiency review records live runs separately.
func declaredTests(t *testing.T) map[string]bool {
	t.Helper()
	return declaredTestsAt(t, moduleRoot)
}

func declaredTestsAt(t *testing.T, root string) map[string]bool {
	t.Helper()
	set := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".git", "node_modules", "chainbench-out", "vendor":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		aliases := map[string]bool{}
		for _, imp := range file.Imports {
			imported, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return err
			}
			if imported == "testing" {
				name := "testing"
				if imp.Name != nil {
					name = imp.Name.Name
				}
				aliases[name] = true
			}
		}
		for _, declaration := range file.Decls {
			function, ok := declaration.(*ast.FuncDecl)
			if !ok || function.Recv != nil || function.Body == nil || !strings.HasPrefix(function.Name.Name, "Test") {
				continue
			}
			suffix := strings.TrimPrefix(function.Name.Name, "Test")
			if suffix != "" {
				first, _ := utf8.DecodeRuneInString(suffix)
				if unicode.IsLower(first) {
					continue
				}
			}
			if function.Type.TypeParams != nil || function.Type.Params.NumFields() != 1 || function.Type.Results.NumFields() != 0 {
				continue
			}
			pointer, ok := function.Type.Params.List[0].Type.(*ast.StarExpr)
			if !ok {
				continue
			}
			selector, ok := pointer.X.(*ast.SelectorExpr)
			if !ok || selector.Sel.Name != "T" {
				continue
			}
			pkg, ok := selector.X.(*ast.Ident)
			if ok && aliases[pkg.Name] {
				set[function.Name.Name] = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("parse test declarations: %v", err)
	}
	if len(set) == 0 {
		t.Fatal("found no test declarations; the inventory cannot attest anything")
	}
	return set
}

// Formatting or source text must never change whether a real Go test exists.
func TestDeclaredTestsUseDeclarationsRatherThanText(t *testing.T) {
	root := t.TempDir()
	source := "package fixture\nimport tst \"testing\"\n" +
		"/*\nfunc TestCommentIsNotEvidence\n*/\n" +
		"var text = `\nfunc TestStringIsNotEvidence\n`\n" +
		"func\nTestFormatted(t *tst.T) {}\n" +
		"func TestOrdinary(t *tst.T) {}\n" +
		"func TestWrongSignature(x int) {}\n" +
		"type receiver struct{}\nfunc (receiver) TestMethod(t *tst.T) {}\n"
	if err := os.WriteFile(filepath.Join(root, "fixture_test.go"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	got := declaredTestsAt(t, root)
	if len(got) != 2 || !got["TestFormatted"] || !got["TestOrdinary"] {
		t.Fatalf("text was mistaken for test declarations: %v", got)
	}
}

// This holds the review's links and honest scope, not the sufficiency judgment.
func TestRequirementSufficiencyReviewKeepsEveryEvidenceLink(t *testing.T) {
	bytes, err := os.ReadFile("../../docs/dev/requirement-sufficiency-review.json")
	if err != nil {
		t.Fatal(err)
	}
	var review struct {
		Scope        string `json:"scope"`
		Final        string `json:"finalWeb14Acceptance"`
		Requirements []struct {
			ID        string   `json:"id"`
			Criterion string   `json:"criterion"`
			Tests     []string `json:"tests"`
			Limit     string   `json:"limit"`
		} `json:"requirements"`
	}
	if err := json.Unmarshal(bytes, &review); err != nil {
		t.Fatal(err)
	}
	if review.Scope == "" || review.Final == "" {
		t.Fatal("review must distinguish final acceptance")
	}
	have, seen := declaredTests(t), map[string]bool{}
	for _, row := range review.Requirements {
		if seen[row.ID] {
			t.Errorf("duplicate requirement %s", row.ID)
		}
		seen[row.ID] = true
		if row.Criterion == "" || row.Limit == "" || len(row.Tests) == 0 {
			t.Errorf("%s omits behavior, evidence, or its limits", row.ID)
		}
		for _, name := range row.Tests {
			if !have[name] {
				t.Errorf("%s evidence %s is not a Go test declaration", row.ID, name)
			}
		}
	}
	if len(seen) != 19 {
		t.Errorf("review contains %d requirements, want 19", len(seen))
	}
	for i := 1; i <= 19; i++ {
		if !seen[fmt.Sprintf("R%02d", i)] {
			t.Errorf("missing R%02d", i)
		}
	}
}
