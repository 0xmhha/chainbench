package suitecmd_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/mcp"
)

func TestCLIMCPAndDSLResolveTheSameComposition(t *testing.T) {
	workspace := t.TempDir()
	cases := t.TempDir()
	keys := repoKeys(t)
	raws := []string{
		`{"schemaVersion":"2","kind":"case","id":"first","chainPreset":{"schemaVersion":"2","kind":"chain-preset","id":"p","chain":"stablenet","topology":{"bp":2,"en":1},"launch":{"en":{"nodiscover":true}}},"steps":[{"expect":"blockNumber","is":0}]}`,
		`{"schemaVersion":"2","kind":"case","id":"second","chainPreset":{"schemaVersion":"2","kind":"chain-preset","id":"p","chain":"stablenet","topology":{"bp":2,"en":1},"launch":{"en":{"nodiscover":true}}},"steps":[{"expect":"chainId","is":8283}]}`,
	}
	args := []string{"run", "--plan", "--json", "--workspace-dir", workspace, "--keys", keys, "--binary", "/selected/gstable"}
	var contents [][]byte
	for i, raw := range raws {
		path := filepath.Join(cases, []string{"first.json", "second.json"}[i])
		if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
			t.Fatal(err)
		}
		args = append(args, path)
		contents = append(contents, []byte(raw))
	}
	text, _, err := runSplit(t, args...)
	if err != nil {
		t.Fatal(err)
	}
	var cli []app.SuitePlan
	if err := json.Unmarshal([]byte(text), &cli); err != nil {
		t.Fatal(err, text)
	}
	request, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "chainbench_run", "arguments": map[string]any{"plan": true, "specs": raws, "dataDir": workspace, "keysDir": keys, "binary": "/selected/gstable"}}})
	if err != nil {
		t.Fatal(err)
	}
	server := mcp.Default("parity", "test")
	var response struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
		Error any `json:"error"`
	}
	if err := json.Unmarshal(server.Handle(context.Background(), request), &response); err != nil {
		t.Fatal(err)
	}
	if response.Error != nil || response.Result.IsError || len(response.Result.Content) != 1 {
		t.Fatalf("MCP refused: %+v", response)
	}
	var tools []app.SuitePlan
	if err := json.Unmarshal([]byte(response.Result.Content[0].Text), &tools); err != nil {
		t.Fatal(err)
	}
	dsl, err := app.PlanSuites(context.Background(), app.RunSuiteIn{DataDir: workspace, SpecContent: contents, KeysDir: keys, Binary: "/selected/gstable"})
	if err != nil {
		t.Fatal(err)
	}
	if len(cli) != 2 || len(tools) != 2 || len(dsl) != 2 {
		t.Fatal("a definition disappeared")
	}
	for i := range cli {
		cli[i].Spec = ""
		tools[i].Spec = ""
		dsl[i].Spec = ""
		a, _ := json.Marshal(cli[i])
		b, _ := json.Marshal(tools[i])
		c, _ := json.Marshal(dsl[i])
		if !bytes.Equal(a, b) || !bytes.Equal(a, c) {
			t.Fatalf("surface %d disagrees: CLI=%s MCP=%s DSL=%s", i, a, b, c)
		}
		if cli[i].Layout.BPCount != 2 || cli[i].Layout.ENCount != 1 || len(cli[i].Plan.Launch["en"]) != 1 {
			t.Fatal("declared role or option lost")
		}
	}
	if entries, err := os.ReadDir(workspace); err != nil || len(entries) != 0 {
		t.Fatal("planning touched target state")
	}
}
