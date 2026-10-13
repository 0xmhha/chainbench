package resourcecmd_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInspectReadsTargetAndFilesWithoutChangingThem(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "input")
	if err := os.WriteFile(path, []byte("retained input"), 0600); err != nil {
		t.Fatal(err)
	}
	request := filepath.Join(t.TempDir(), "request.json")
	raw, _ := json.Marshal(map[string]any{"target": map[string]any{"dataRoot": dir}, "inspect": true, "checksums": []string{path}, "exists": []string{filepath.Join(dir, "absent")}})
	if err := os.WriteFile(request, raw, 0600); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "resource", "inspect", "--request", request, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Target    struct{ HostIdentity, DataPath string }
		Checksums map[string]string
		Exists    map[string]bool
	}
	if err = json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(out, err)
	}
	physical, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Target.HostIdentity == "" || doc.Target.DataPath != physical || doc.Checksums[path] == "" || doc.Exists[filepath.Join(dir, "absent")] {
		t.Fatal(out)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "retained input" {
		t.Fatal("inspection changed input", err)
	}
	if _, err = os.Stat(filepath.Join(dir, "absent")); !os.IsNotExist(err) {
		t.Fatal("inspection created an absent path")
	}
}
