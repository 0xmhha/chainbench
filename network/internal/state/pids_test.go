package state

import (
	"bytes"
	"os"
	"testing"
)

func TestParsePIDs_ValidFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/pids-default.json")
	if err != nil {
		t.Fatal(err)
	}
	p, err := ParsePIDs(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if p.Profile != "default" {
		t.Errorf("profile: got %q", p.Profile)
	}
	if p.ChainID != "local-default-20260420084840" {
		t.Errorf("chain_id: got %q", p.ChainID)
	}
	if len(p.Nodes) != 2 {
		t.Fatalf("nodes count: got %d, want 2", len(p.Nodes))
	}
	n1, ok := p.Nodes["1"]
	if !ok {
		t.Fatal("node 1 missing")
	}
	if n1.Type != "validator" || n1.HTTPPort != 8501 || n1.WSPort != 9501 {
		t.Errorf("node 1: got %+v", n1)
	}
	n5, ok := p.Nodes["5"]
	if !ok {
		t.Fatal("node 5 missing")
	}
	if n5.Type != "endpoint" || n5.HTTPPort != 8505 {
		t.Errorf("node 5: got %+v", n5)
	}
}

func TestParsePIDs_MalformedJSON(t *testing.T) {
	_, err := ParsePIDs(bytes.NewReader([]byte(`{"chain_id":`)))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}

func TestParsePIDs_EmptyNodes(t *testing.T) {
	p, err := ParsePIDs(bytes.NewReader([]byte(`{"chain_id":"x","profile":"y","nodes":{}}`)))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(p.Nodes) != 0 {
		t.Errorf("nodes: got %d, want 0", len(p.Nodes))
	}
}

func TestReadPIDsFile_MissingFile(t *testing.T) {
	_, err := ReadPIDsFile("testdata/does-not-exist.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestReadPIDsFile_Valid(t *testing.T) {
	p, err := ReadPIDsFile("testdata/pids-default.json")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if len(p.Nodes) != 2 {
		t.Errorf("nodes: got %d", len(p.Nodes))
	}
}
