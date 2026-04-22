package state

import (
	"bytes"
	"os"
	"testing"
)

func TestParseProfile_ValidFixture(t *testing.T) {
	data, err := os.ReadFile("testdata/profile-default.yaml")
	if err != nil {
		t.Fatal(err)
	}
	p, err := ParseProfile(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if p.Name != "default" {
		t.Errorf("name: got %q, want default", p.Name)
	}
	if p.Chain.Binary != "gstable" {
		t.Errorf("binary: got %q", p.Chain.Binary)
	}
	if p.Chain.ChainID != 8283 {
		t.Errorf("chain_id: got %d, want 8283", p.Chain.ChainID)
	}
	if p.Chain.NetworkID != 8283 {
		t.Errorf("network_id: got %d", p.Chain.NetworkID)
	}
	if p.Nodes.Validators != 4 || p.Nodes.Endpoints != 1 {
		t.Errorf("nodes: got %+v", p.Nodes)
	}
	if p.Ports.BaseHTTP != 8501 || p.Ports.BaseWS != 9501 || p.Ports.BaseP2P != 30301 {
		t.Errorf("ports: got %+v", p.Ports)
	}
}

func TestParseProfile_Empty(t *testing.T) {
	p, err := ParseProfile(bytes.NewReader(nil))
	if err != nil {
		t.Fatalf("empty should parse to zero-value profile, got: %v", err)
	}
	if p.Name != "" || p.Chain.ChainID != 0 {
		t.Errorf("expected zero-value profile, got %+v", p)
	}
}

func TestParseProfile_MalformedYAML(t *testing.T) {
	_, err := ParseProfile(bytes.NewReader([]byte("chain: [this is not valid")))
	if err == nil {
		t.Fatal("expected error for malformed YAML")
	}
}

func TestReadProfileFile_MissingFile(t *testing.T) {
	_, err := ReadProfileFile("testdata/does-not-exist.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestReadProfileFile_Valid(t *testing.T) {
	p, err := ReadProfileFile("testdata/profile-default.yaml")
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if p.Chain.ChainID != 8283 {
		t.Errorf("chain_id: got %d", p.Chain.ChainID)
	}
}
