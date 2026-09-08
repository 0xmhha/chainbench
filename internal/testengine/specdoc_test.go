package testengine_test

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// specDoc is the migration record. It is parsed rather than duplicated, the way
// the architecture tests read layers.md: a list of blocked cases that lives only
// in prose goes stale silently, and this one did.
const specDoc = "../../tests/tc/SPECS.md"

// gapSection is where a case is recorded as NOT migrated. Everything under one
// of these headings names cases blocked on a grammar gap.
var gapHeadings = []string{
	"## 이관하지 않은 것과 그 이유",
	"### gas-policy 잔여",
	"### accounts 잔여",
	"### hardfork 잔여",
	"### system-contracts 잔여",
}

// leadingNames matches the ids a row opens with. The table's convention is that
// a row names its cases first, separated by "·", and then explains; reading the
// whole cell instead picks up every backticked word in the prose, which is how
// an earlier version counted `logs` and `call` as legacy cases.
var leadingNames = regexp.MustCompile("^\\s*(?:~~)?(?:`[a-z][a-z0-9-]{3,}`(?:~~)?\\s*(?:·\\s*(?:~~)?)?)+")

var (
	// caseName picks one backticked id out of the leading run.
	caseName = regexp.MustCompile("`([a-z][a-z0-9-]{3,})`")
	// migrated marks a row struck through, which is how the document says a
	// case that was once blocked has since been written.
	migrated = regexp.MustCompile(`~~[^~]*~~`)
)

// filePrefix matches the "NN-" / "NNb-" ordering prefix a spec file carries
// under tests/tc, which mirrors the legacy suite's numbering. The document
// names specs by id, so the prefix is stripped before matching.
var filePrefix = regexp.MustCompile(`^[0-9]+[a-z]?-`)

// specNameFromFile turns a spec file name into the id the document uses.
func specNameFromFile(name string) string {
	return filePrefix.ReplaceAllString(strings.TrimSuffix(name, ".json"), "")
}

// TestSpecDoc_BlockedCasesHaveNoSpec keeps the migration record honest.
//
// Measured 2026-09-07: the document listed nineteen cases as blocked on grammar
// gaps, and fifteen of them had specs that validate. The primitives it named as
// missing — wsOpen for opening a subscription before the log that fills it,
// sendRawTampered for a corrupted double signature, sendSetCode for EIP-7702,
// newAccount for a fresh local key, methodPresent for "the method exists even
// if it errors", callError for a call expected to revert, derive op:"quorum"
// for ceil(2n/3) — had all been added since the notes were written.
//
// That is worse than an incomplete document. Someone planning work from it
// would re-implement finished cases, and someone auditing coverage would
// believe it thinner than it is. So the claim is now checked: a case the
// document says is blocked must not have a spec, and a case that has one must
// be struck through with what closed the gap.
func TestSpecDoc_BlockedCasesHaveNoSpec(t *testing.T) {
	raw, err := os.ReadFile(specDoc)
	if err != nil {
		t.Fatal(err)
	}
	specs := map[string]string{}
	if err := filepath.WalkDir("../../tests/tc", func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".json") {
			specs[specNameFromFile(d.Name())] = p
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if len(specs) == 0 {
		t.Fatal("no specs were found, so this test asserts nothing")
	}

	var (
		inGap   bool
		claimed int
	)
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "#") {
			inGap = false
			for _, h := range gapHeadings {
				if strings.HasPrefix(line, h) {
					inGap = true
				}
			}
			continue
		}
		if !inGap || !strings.HasPrefix(line, "|") {
			continue
		}
		cell := strings.SplitN(strings.TrimPrefix(line, "|"), "|", 2)[0]
		// A struck-through row is the document saying "this one landed", which
		// is exactly when a spec is expected.
		lead := leadingNames.FindString(cell)
		for _, m := range caseName.FindAllStringSubmatch(migrated.ReplaceAllString(lead, ""), -1) {
			name := m[1]
			claimed++
			if p, ok := specs[name]; ok {
				t.Errorf("%s is listed as blocked on a grammar gap, but %s exists and validates — strike the row through and say what closed the gap", name, p)
			}
		}
	}
	if claimed == 0 {
		t.Fatal("no blocked case was read out of the document, so the parse is wrong rather than the document clean")
	}
	t.Logf("%d cases are recorded as blocked, and none of them has a spec", claimed)
}
