package chainsetup_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/chainsetup"
)

// writeRecord puts a composition record on disk exactly as given, so a test can
// present the shape an older build would have written.
func writeRecord(t *testing.T, dir string, record map[string]any) {
	t.Helper()
	b, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "chain-record.json"), b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestOpen_RefusesARecordFromAnotherFormat is why the record carries a version.
//
// It had none. A field could be renamed, removed, or given a different meaning
// and an older record still decoded: the missing field read as a zero, and a
// zero is a legitimate value for most of them, so the composition came up
// describing itself wrongly rather than refusing. The failure then showed as
// behaviour — a step doing the wrong thing — with nothing pointing at the
// record.
//
// There is no migration on purpose. This track keeps one current shape, so a
// record from another version is a workspace to compose again, and the message
// says so.
func TestOpen_RefusesARecordFromAnotherFormat(t *testing.T) {
	for _, tc := range []struct {
		name   string
		record map[string]any
	}{
		{"no version at all (every record written before this field)", map[string]any{
			"chain": "stablenet", "steps": map[string]any{},
		}},
		{"an older version", map[string]any{
			"formatVersion": chainsetup.StateFormatVersion - 1, "chain": "stablenet", "steps": map[string]any{},
		}},
		{"a newer version, written by a build this one does not know", map[string]any{
			"formatVersion": chainsetup.StateFormatVersion + 1, "chain": "stablenet", "steps": map[string]any{},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeRecord(t, dir, tc.record)

			_, err := chainsetup.Open(dir, nil)
			if err == nil {
				t.Fatal("a record from another format must be refused")
			}
			// The message has to say what to do about it, not only that
			// something is wrong.
			for _, want := range []string{"format", "chain new"} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not mention %q", err, want)
				}
			}
		})
	}
}

// TestOpen_AcceptsAFreshWorkspaceAndStampsIt: a directory with no record is
// where a composition starts, so it opens, and the version is on it from the
// first save rather than from whenever a field happens to be set.
func TestOpen_AcceptsAFreshWorkspaceAndStampsIt(t *testing.T) {
	dir := t.TempDir()
	ws, err := chainsetup.Open(dir, nil)
	if err != nil {
		t.Fatalf("a fresh workspace must open: %v", err)
	}
	if err := ws.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(dir, "chain-record.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		FormatVersion int `json:"formatVersion"`
	}
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got.FormatVersion != chainsetup.StateFormatVersion {
		t.Errorf("record written at format %d, want %d", got.FormatVersion, chainsetup.StateFormatVersion)
	}

	// And it re-opens, which is the other half: a record this build wrote must
	// be one it reads.
	if _, err := chainsetup.Open(dir, nil); err != nil {
		t.Errorf("a record this build wrote must re-open: %v", err)
	}
}

// TestRecord_KeepsTheStatePath is the field's whole guarantee for now.
//
// Nothing reads it yet, which is the risk a field like this carries: it can be
// added, look right, and turn out not to survive a save when the code that
// needs it arrives. So the round trip is checked at the commit that adds it,
// not at the one that starts depending on it.
func TestRecord_KeepsTheStatePath(t *testing.T) {
	dir := t.TempDir()
	ws, err := chainsetup.Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	const path = "Composition/Composing/BuildingGenesis/GenesisFromTemplate"
	ws.SetStatePath(path)
	if err := ws.Save(); err != nil {
		t.Fatal(err)
	}

	back, err := chainsetup.Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got := back.State().StatePath; got != path {
		t.Errorf("the record came back at %q, want %q", got, path)
	}
}

// TestRecord_OmitsAnEmptyStatePath: a workspace no machine has walked should
// not carry an empty key, because a reader meeting "statePath": "" cannot tell
// it from a machine that reported nowhere.
func TestRecord_OmitsAnEmptyStatePath(t *testing.T) {
	dir := t.TempDir()
	ws, err := chainsetup.Open(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := ws.Save(); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, "chain-record.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "statePath") {
		t.Errorf("an unwalked workspace records a state path:\n%s", b)
	}
}
