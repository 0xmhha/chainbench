package testengine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTailFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "node.log")
	var b strings.Builder
	for i := 1; i <= 500; i++ {
		b.WriteString("line")
		b.WriteString(strings.Repeat("x", 0))
		b.WriteByte('\n')
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	tail, err := tailFile(path, failureLogTailLines)
	if err != nil {
		t.Fatalf("tailFile: %v", err)
	}
	lines := strings.Split(string(tail), "\n")
	if len(lines) != failureLogTailLines {
		t.Errorf("tail has %d lines, want %d", len(lines), failureLogTailLines)
	}
}

func TestTailFile_ShorterThanTail(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "node.log")
	if err := os.WriteFile(path, []byte("a\nb\nc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tail, err := tailFile(path, failureLogTailLines)
	if err != nil {
		t.Fatalf("tailFile: %v", err)
	}
	if string(tail) != "a\nb\nc" {
		t.Errorf("tail = %q, want a\\nb\\nc", string(tail))
	}
}

func TestTailFile_Missing(t *testing.T) {
	if _, err := tailFile(filepath.Join(t.TempDir(), "nope.log"), 10); err == nil {
		t.Fatal("want an error for a missing file")
	}
}
