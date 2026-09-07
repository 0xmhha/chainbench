package dsl_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/dsl"
)

// seedCorpus reads the committed specs so the fuzzer starts from documents that
// are actually valid. Random bytes explore the reject path; a real spec is what
// lets a mutation reach the accept path, which is where a crash would matter.
func seedCorpus(f *testing.F) {
	f.Helper()
	var n int
	_ = filepath.WalkDir("../../tests/specs", func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".json") || n >= 40 {
			return nil
		}
		b, rerr := os.ReadFile(p)
		if rerr == nil {
			f.Add(b)
			n++
		}
		return nil
	})
	for _, s := range []string{
		"", "{}", "[]", "null", "0", `{"schemaVersion":"1"}`, `{"schemaVersion":"2"}`,
		`{"schemaVersion":"2","kind":"env"}`, `{"schemaVersion":"2","kind":"case"}`,
		`{"schemaVersion":"1","id":"x","chain":{"name":"a","binary":"b"},"assertions":[]}`,
		`{"schemaVersion":` + strings.Repeat("9", 300) + `}`,
		"\x00\x01\x02", `{"a":` + strings.Repeat("[", 200) + strings.Repeat("]", 200) + "}",
	} {
		f.Add([]byte(s))
	}
}

// FuzzParse is B1-b: arbitrary input must not kill the grammar.
//
// A spec is a file a person writes and a tool generates, so malformed input is
// the normal case rather than an attack. What must never happen is a panic:
// `chainbench validate` exists to tell an author what is wrong with their
// document, and a parser that dies on a bad one cannot do the job it is for.
//
// It also asserts the property that makes a parse worth having: whatever comes
// back accepted is complete enough for the interpreter. A Spec that parsed but
// carries no id, no chain or no schema version would be accepted here and fail
// somewhere with no file name attached.
func FuzzParse(f *testing.F) {
	seedCorpus(f)
	f.Fuzz(func(t *testing.T, raw []byte) {
		s, err := dsl.Parse(raw)
		if err != nil {
			return
		}
		// The validator's own promise: these three are what every consumer
		// reads first, and a spec missing one addresses nothing.
		if s.ID == "" {
			t.Fatalf("a spec parsed with no id: %q", raw)
		}
		if s.Chain.Name == "" {
			t.Fatalf("spec %q parsed with no chain: %q", s.ID, raw)
		}
		if s.SchemaVersion == "" {
			t.Fatalf("spec %q parsed with no schema version: %q", s.ID, raw)
		}
	})
}

// FuzzMigrateV1 covers the other direction: the migration writes a document the
// parser then has to read. A conversion that emits something its own parser
// rejects would fail at whatever picks the file up next, with the migration
// already reported as successful.
func FuzzMigrateV1(f *testing.F) {
	seedCorpus(f)
	f.Fuzz(func(t *testing.T, raw []byte) {
		out, err := dsl.MigrateV1(raw)
		if err != nil {
			return
		}
		if !json.Valid(out) {
			t.Fatalf("migration emitted invalid JSON from %q:\n%s", raw, out)
		}
		if _, err := dsl.Parse(out); err != nil {
			t.Fatalf("migration emitted a document its own parser rejects: %v\nfrom %q\ngot:\n%s", err, raw, out)
		}
	})
}

// FuzzInlineEnv exercises the one parse path that takes a callback, so a lookup
// that misbehaves — returning junk, or the document itself — cannot take the
// parser down with it.
func FuzzInlineEnv(f *testing.F) {
	seedCorpus(f)
	f.Fuzz(func(t *testing.T, raw []byte) {
		// A lookup that hands back the caller's own bytes is the worst
		// plausible one: it is what a mis-wired resolver does, and it invites
		// unbounded recursion.
		_, _ = dsl.InlineEnv(raw, func(string) ([]byte, error) { return raw, nil })
		_, _ = dsl.InlineEnv(raw, func(string) ([]byte, error) { return []byte("{"), nil })
	})
}

// TestValidate_RefusesAStatementThatNamesNothing pins what the fuzzer found, as
// a case a reader can see without running a fuzzer.
//
// Each of these used to parse and then die at run time — after the network was
// composed and launched — with `unknown action ""` or `unknown assertion ""`
// and no file or index attached.
func TestValidate_RefusesAStatementThatNamesNothing(t *testing.T) {
	const head = `{"schemaVersion":"1","id":"x","chain":{"name":"a","binary":"b"},`
	for name, c := range map[string]struct{ doc, wants string }{
		"an assertion that names no check": {
			head + `"assertions":[{}]}`, "assertion 1"},
		"an assertion whose check is empty": {
			head + `"assertions":[{"assert":""}]}`, "assertion 1"},
		"a step that names no action": {
			head + `"steps":[{}],"assertions":[{"assert":"rpc"}]}`, "step 1 names no action"},
		"a step that names two": {
			head + `"steps":[{"waitBlock":1,"sendTx":{}}],"assertions":[{"assert":"rpc"}]}`, "one step is one action"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := dsl.Parse([]byte(c.doc))
			if err == nil {
				t.Fatalf("%s parsed, so it would have failed at run time instead", name)
			}
			if !strings.Contains(err.Error(), c.wants) {
				t.Errorf("error %q does not say %q, so it does not point at the statement", err, c.wants)
			}
		})
	}
}

// TestMigrate_EmitsWhatItsOwnParserReads is the property the fuzzer asserts,
// stated once against the committed specs. A migration reported as successful
// that produces an unreadable document fails at whatever picks the file up
// next, by which time the original is gone.
func TestMigrate_EmitsWhatItsOwnParserReads(t *testing.T) {
	var n int
	err := filepath.WalkDir("../../tests/specs", func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".json") {
			return nil
		}
		raw, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil
		}
		out, merr := dsl.MigrateV1(raw)
		if merr != nil {
			return nil // not a v1 spec, or not migratable; the fuzzer covers those
		}
		n++
		if _, perr := dsl.Parse(out); perr != nil {
			t.Errorf("%s migrated to a document the parser rejects: %v", p, perr)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatal("no spec was migrated, so this test asserts nothing")
	}
	t.Logf("%d committed specs migrate to documents the parser reads", n)
}
