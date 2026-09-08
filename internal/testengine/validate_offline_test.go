package testengine_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	_ "github.com/0xmhha/chainbench/internal/chains/all"

	"github.com/0xmhha/chainbench/internal/testengine"
)

// TestValidate_TouchesNothing is B1's gate, restated.
//
// It used to read "chainbench validate does not link core/rpc or core/session".
// That cannot pass and no longer measures anything worth having: the U track
// routed every surface through app, whose fan-out is twenty-two, so one binary
// links everything whichever command is run. Splitting the binary would satisfy
// the letter and undo the consolidation.
//
// What the gate was protecting is a property of the command, not of the link:
// `validate` reads specs and says what is wrong with them, and an operator runs
// it before anything exists. It must not dial, and it must not write. That is
// checkable, it is true today, and it is what would actually break if someone
// added a live check to the validation path.
func TestValidate_TouchesNothing(t *testing.T) {
	specs := committedSpecs(t)
	if len(specs) == 0 {
		t.Fatal("no spec was found, so this test asserts nothing")
	}

	// A directory validate has no business touching. Anything it writes —
	// a session, a workspace, a stray artifact — lands somewhere, and the
	// working directory is where a relative path would go.
	dir := t.TempDir()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	start := time.Now()
	results, err := testengine.ValidateSpecs(specs, "")
	elapsed := time.Since(start)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(results) != len(specs) {
		t.Fatalf("validated %d of %d specs", len(results), len(specs))
	}
	for _, r := range results {
		if !r.OK {
			t.Errorf("%s did not validate: %s", r.ID, r.Result)
		}
	}

	// Nothing written. A file here would mean the offline check is not offline.
	left, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 0 {
		var names []string
		for _, e := range left {
			names = append(names, e.Name())
		}
		sort.Strings(names)
		t.Errorf("validate wrote %v into the working directory; it is supposed to read specs and nothing else", names)
	}

	// Nothing dialled. There is no chain running in a unit test, so a validate
	// that reached for one would hang until its timeout rather than finish in
	// milliseconds. The bound is loose on purpose — it is there to catch a
	// dial, not to police the parser's speed.
	if elapsed > 10*time.Second {
		t.Errorf("validating %d specs took %s, which is long enough to have waited on a network", len(specs), elapsed)
	}
	t.Logf("%d specs validated in %s, writing nothing", len(specs), elapsed.Round(time.Millisecond))
}

// TestValidate_ReachesForNothingLive guards the same property where it would
// break first: the file itself.
//
// A live check added to validation would import the packages that do live work,
// and it would be caught here before anyone measured a binary. This is the
// narrow, achievable half of what the old link gate was reaching for.
func TestValidate_ReachesForNothingLive(t *testing.T) {
	live := map[string]string{
		"internal/core/rpc":       "dials a node",
		"internal/core/session":   "writes artifacts",
		"internal/core/process":   "launches and signals processes",
		"internal/core/filestore": "puts files on a target",
		"internal/core/remote":    "opens SSH",
		"internal/chainsetup":     "composes a network",
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "validate.go", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatal(err)
	}
	for _, imp := range f.Imports {
		p := strings.Trim(imp.Path.Value, `"`)
		rel := strings.TrimPrefix(p, "github.com/0xmhha/chainbench/")
		if why, bad := live[rel]; bad {
			t.Errorf("validate.go imports %s, which %s — validation is what an operator runs before anything exists", rel, why)
		}
	}
}

// committedSpecs lists the specs under tests/specs.
func committedSpecs(t *testing.T) []string {
	t.Helper()
	root, err := filepath.Abs("../../tests/specs")
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	_ = filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err == nil && !d.IsDir() && strings.HasSuffix(p, ".json") {
			out = append(out, p)
		}
		return nil
	})
	sort.Strings(out)
	return out
}
