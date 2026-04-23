package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempLog(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "node.log")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestTailFile_ReturnsLastNLines(t *testing.T) {
	path := writeTempLog(t, "a\nb\nc\nd\ne\n")
	got, err := TailFile(path, 3)
	if err != nil {
		t.Fatalf("tail: %v", err)
	}
	want := []string{"c", "d", "e"}
	if len(got) != 3 {
		t.Fatalf("len: got %d, want 3 (%v)", len(got), got)
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("line %d: got %q, want %q", i, got[i], w)
		}
	}
}

func TestTailFile_FewerLinesThanN(t *testing.T) {
	path := writeTempLog(t, "a\nb\nc\n")
	got, err := TailFile(path, 10)
	if err != nil {
		t.Fatalf("tail: %v", err)
	}
	want := []string{"a", "b", "c"}
	if len(got) != 3 {
		t.Fatalf("len: got %d", len(got))
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("line %d: got %q, want %q", i, got[i], w)
		}
	}
}

func TestTailFile_EmptyFile(t *testing.T) {
	path := writeTempLog(t, "")
	got, err := TailFile(path, 5)
	if err != nil {
		t.Fatalf("tail: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("empty: got %v, want []", got)
	}
}

func TestTailFile_SingleLineNoTrailingNewline(t *testing.T) {
	path := writeTempLog(t, "only-one")
	got, err := TailFile(path, 3)
	if err != nil {
		t.Fatalf("tail: %v", err)
	}
	if len(got) != 1 || got[0] != "only-one" {
		t.Errorf("got %v", got)
	}
}

func TestTailFile_NEqualOne(t *testing.T) {
	path := writeTempLog(t, "a\nb\nc\n")
	got, err := TailFile(path, 1)
	if err != nil {
		t.Fatalf("tail: %v", err)
	}
	if len(got) != 1 || got[0] != "c" {
		t.Errorf("got %v, want [c]", got)
	}
}

func TestTailFile_MissingFile(t *testing.T) {
	_, err := TailFile("/no/such/file.log", 10)
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestTailFile_LargeLine(t *testing.T) {
	// A single line larger than default bufio.Scanner buffer (64 KiB).
	big := strings.Repeat("x", 200*1024)
	path := writeTempLog(t, big+"\n")
	got, err := TailFile(path, 1)
	if err != nil {
		t.Fatalf("tail: %v", err)
	}
	if len(got) != 1 || len(got[0]) != 200*1024 {
		t.Errorf("large line not preserved: got len=%d", len(got[0]))
	}
}
