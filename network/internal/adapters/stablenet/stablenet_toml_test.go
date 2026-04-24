package stablenet_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/network/internal/adapters/spec"
	"github.com/0xmhha/chainbench/network/internal/adapters/stablenet"
)

func loadNodeTemplate(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "templates", "node.template.toml"))
	if err != nil {
		t.Fatalf("load node template: %v", err)
	}
	return data
}

func decodeTomlInput(t *testing.T, dir string) spec.TomlInput {
	t.Helper()
	var metadata spec.Metadata
	if err := json.Unmarshal(loadFixture(t, "metadata.json"), &metadata); err != nil {
		t.Fatalf("metadata decode: %v", err)
	}
	return spec.TomlInput{
		Metadata:      metadata,
		TemplateBytes: loadNodeTemplate(t),
		WriteDir:      dir,
		TotalNodes:    2,
		NumValidators: 2,
		BaseP2P:       30303,
		BaseHTTP:      8545,
		BaseWS:        8546,
		BaseAuth:      8551,
		BaseMetrics:   6060,
	}
}

func TestGenerateToml_Happy(t *testing.T) {
	dir := t.TempDir()
	in := decodeTomlInput(t, dir)

	if err := stablenet.New().GenerateToml(context.Background(), in); err != nil {
		t.Fatalf("GenerateToml: %v", err)
	}

	for i := 1; i <= 2; i++ {
		name := fmt.Sprintf("config_node%d.toml", i)
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		s := string(data)

		wantPort := fmt.Sprintf("HTTPPort = %d", 8545+i-1)
		if !strings.Contains(s, wantPort) {
			t.Errorf("%s missing %q", name, wantPort)
		}
		if !strings.Contains(s, "StaticNodes") {
			t.Errorf("%s missing StaticNodes", name)
		}
		// Both enodes must appear in every file (shared static-nodes block).
		for _, pub := range []string{"aaaaaaaaaaaa", "bbbbbbbbbbbb"} {
			wantEnode := fmt.Sprintf("enode://%s@127.0.0.1:", pub)
			if !strings.Contains(s, wantEnode) {
				t.Errorf("%s missing enode for %s", name, pub)
			}
		}
		// Validator => miner section present.
		if !strings.Contains(s, `[Eth.Miner]`) {
			t.Errorf("%s missing [Eth.Miner] (expected for validator)", name)
		}
		// Auth/metrics ports too.
		wantAuth := fmt.Sprintf("AuthPort = %d", 8551+i-1)
		if !strings.Contains(s, wantAuth) {
			t.Errorf("%s missing %q", name, wantAuth)
		}
		wantMetrics := fmt.Sprintf("Port = %d", 6060+i-1)
		if !strings.Contains(s, wantMetrics) {
			t.Errorf("%s missing %q", name, wantMetrics)
		}
		// P2P port on ListenAddr.
		wantP2P := fmt.Sprintf("ListenAddr = \":%d\"", 30303+i-1)
		if !strings.Contains(s, wantP2P) {
			t.Errorf("%s missing %q", name, wantP2P)
		}
		// ethstats url
		wantEthstats := fmt.Sprintf("node%d:local@localhost:3000", i)
		if !strings.Contains(s, wantEthstats) {
			t.Errorf("%s missing ethstats URL %q", name, wantEthstats)
		}
		// keystore dir
		wantKs := fmt.Sprintf("%s/keystores/node%d", dir, i)
		if !strings.Contains(s, wantKs) {
			t.Errorf("%s missing keystore dir %q", name, wantKs)
		}
	}
}

func TestGenerateToml_EndpointSkipsMinerSection(t *testing.T) {
	dir := t.TempDir()
	in := decodeTomlInput(t, dir)
	in.NumValidators = 1 // node1 validator, node2 endpoint

	if err := stablenet.New().GenerateToml(context.Background(), in); err != nil {
		t.Fatalf("GenerateToml: %v", err)
	}

	node1, err := os.ReadFile(filepath.Join(dir, "config_node1.toml"))
	if err != nil {
		t.Fatalf("read node1: %v", err)
	}
	node2, err := os.ReadFile(filepath.Join(dir, "config_node2.toml"))
	if err != nil {
		t.Fatalf("read node2: %v", err)
	}
	if !strings.Contains(string(node1), "[Eth.Miner]") {
		t.Errorf("node1 (validator) should have [Eth.Miner]")
	}
	if strings.Contains(string(node2), "[Eth.Miner]") {
		t.Errorf("node2 (endpoint) should NOT have [Eth.Miner]")
	}
}

func TestGenerateToml_TotalNodesExceedsMetadata(t *testing.T) {
	dir := t.TempDir()
	in := decodeTomlInput(t, dir)
	in.TotalNodes = 5 // metadata only has 2 nodes

	if err := stablenet.New().GenerateToml(context.Background(), in); err == nil {
		t.Fatal("expected error when TotalNodes exceeds metadata.nodes length")
	}
}

func TestGenerateToml_EmptyTemplate(t *testing.T) {
	dir := t.TempDir()
	in := decodeTomlInput(t, dir)
	in.TemplateBytes = nil

	if err := stablenet.New().GenerateToml(context.Background(), in); err == nil {
		t.Fatal("expected error for empty template")
	}
}

func TestGenerateToml_MissingWriteDir(t *testing.T) {
	in := decodeTomlInput(t, "")
	if err := stablenet.New().GenerateToml(context.Background(), in); err == nil {
		t.Fatal("expected error for missing WriteDir")
	}
}

func TestGenerateToml_InvalidTotalNodes(t *testing.T) {
	dir := t.TempDir()
	in := decodeTomlInput(t, dir)
	in.TotalNodes = 0

	if err := stablenet.New().GenerateToml(context.Background(), in); err == nil {
		t.Fatal("expected error for zero TotalNodes")
	}
}
