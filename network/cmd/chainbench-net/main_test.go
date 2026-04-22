package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestVersionCommand_PrintsSemver(t *testing.T) {
	var buf bytes.Buffer
	cmd := newRootCmd()
	cmd.SetOut(&buf)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}

	out := buf.String()
	if !strings.HasPrefix(out, "chainbench-net ") {
		t.Fatalf("want prefix %q, got %q", "chainbench-net ", out)
	}
	if !strings.Contains(out, "\n") {
		t.Fatalf("want trailing newline, got %q", out)
	}
}
