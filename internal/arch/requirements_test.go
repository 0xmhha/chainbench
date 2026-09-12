package arch

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
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

// declaredTests returns every Test function name in the module, found with go
// vet's own view of the tree rather than a hand-kept list of directories.
func declaredTests(t *testing.T) map[string]bool {
	t.Helper()
	out, err := exec.Command("grep", "-rho", "--include=*_test.go", `^func Test[A-Za-z_0-9]*`, moduleRoot).Output()
	if err != nil {
		t.Fatalf("scan for test functions: %v", err)
	}
	set := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		if name := strings.TrimPrefix(strings.TrimSpace(line), "func "); name != "" {
			set[name] = true
		}
	}
	if len(set) == 0 {
		t.Fatal("found no test functions at all, so the scan is wrong rather than the module empty")
	}
	return set
}
