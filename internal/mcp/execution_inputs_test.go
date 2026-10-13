package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/app"
)

func TestExecutionToolsExposeCLIInputs(t *testing.T) {
	for _, tc := range []struct {
		name   string
		inputs []string
	}{
		{"chainbench_run", []string{"binaryFor", "readOnlyKeys", "suite", "noNodeControl", "serverLogins", "workspaceConfig", "artifactRoot"}},
		{"chainbench_chain_up", []string{"stage", "chainPreset", "serverLogins"}},
		{"chainbench_chain_observe", []string{"serverLogins"}},
		{"chainbench_file_upload", []string{"sha256", "mode", "serverLogins", "localPaths", "workspaceConfig"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tool, ok := Default("inputs", "test").tools[tc.name]
			if !ok {
				t.Fatal("missing tool")
			}
			props := tool.InputSchema["properties"].(map[string]any)
			for _, name := range tc.inputs {
				if props[name] == nil {
					t.Errorf("missing input %s", name)
				}
			}
		})
	}
}

func TestRunToolPassesNamedBinaryBindingsToPlan(t *testing.T) {
	keys, err := filepath.Abs("../../presets/keys")
	if err != nil {
		t.Fatal(err)
	}
	args := map[string]any{"plan": true, "suite": true, "dataDir": t.TempDir(), "keysDir": keys, "spec": `{"schemaVersion":"2","kind":"case","id":"c","chainPreset":{"schemaVersion":"2","kind":"chain-preset","id":"p","chain":"stablenet","topology":{"bp":1}},"steps":[{"expect":"blockNumber","is":0}]}`, "binaryFor": map[string]any{"default": "/accepted/gstable"}}
	out, err := runTool().Handler(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	var plans []app.SuitePlan
	if err = json.Unmarshal([]byte(out), &plans); err != nil {
		t.Fatal(err)
	}
	if len(plans) != 1 || plans[0].Layout.Binary != "/accepted/gstable" {
		t.Fatalf("binding ignored: %s", out)
	}
	args["binaryFor"] = map[string]any{"undeclared": "/accepted/other"}
	if _, err = runTool().Handler(context.Background(), args); err == nil {
		t.Fatal("undeclared binding accepted")
	}
}

func TestUploadToolChecksDigestAndSetsMode(t *testing.T) {
	tool, ok := Default("upload", "test").tools["chainbench_file_upload"]
	if !ok {
		t.Fatal("missing upload tool")
	}
	dir := t.TempDir()
	root := filepath.Join(dir, "data")
	set := filepath.Join(dir, "servers.yaml")
	config := filepath.Join(dir, "workspace.yaml")
	for path, content := range map[string]string{set: "version: 2\npool:\n  hosts:\n    - {name: here, addr: 127.0.0.1}\n", config: "version: 1\ndataRoot: " + root + "\npaths: {binaries: binaries, configs: configs, genesis: genesis, keystore: keystore, keyrings: keyrings, nodes: nodes, runtime: runtime, logs: logs}\ncontrol: {artifactRoot: out}\ninputs: {mode: generated}\nexecution: {chain: fresh}\n"} {
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	src := filepath.Join(dir, "asset")
	content := []byte("accepted bytes")
	if err := os.WriteFile(src, content, 0600); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(root, "asset")
	args := map[string]any{"serverSet": set, "server": "here", "workspaceConfig": config, "serverLogins": set, "localPaths": []any{src}, "remote": dst, "sha256": strings.Repeat("0", 64), "mode": "0755"}
	if _, err := tool.Handler(context.Background(), args); err == nil || !strings.Contains(err.Error(), "sha256") {
		t.Fatalf("wrong digest: %v", err)
	}
	if _, err := os.Stat(dst); !os.IsNotExist(err) {
		t.Fatal("rejected bytes were written")
	}
	sum := sha256.Sum256(content)
	args["sha256"] = hex.EncodeToString(sum[:])
	if _, err := tool.Handler(context.Background(), args); err != nil {
		t.Fatal(err)
	}
	s, err := os.Stat(dst)
	if err != nil || s.Mode().Perm() != 0755 {
		t.Fatalf("mode: %v %v", s, err)
	}
	b, err := os.ReadFile(dst)
	if err != nil || string(b) != string(content) {
		t.Fatal("written bytes differ", err)
	}
	args["forceUpload"] = true
	args["mode"] = "4755"
	if _, err := tool.Handler(context.Background(), args); err == nil {
		t.Fatal("special mode accepted")
	}
}

func TestAttachInputPreservesAcceptedKeysAndControlPolicy(t *testing.T) {
	args := map[string]any{"keysDir": "/accepted/keys", "readOnlyKeys": true, "noNodeControl": true}
	in := attachInput(args, app.Start{}, "stablenet", []string{"http://localhost:8545"}, "/recorded/workspace", "/reports", [][]byte{[]byte("case")})
	if !in.ReadOnlyKeys || !in.NoNodeControl || in.KeysDir != "/accepted/keys" || in.DataDir != "/recorded/workspace" || in.ArtifactRoot != "/reports" {
		t.Fatalf("attach policy lost: %+v", in)
	}
	in = attachInput(map[string]any{}, app.Start{}, "", nil, "", "", nil)
	if in.ReadOnlyKeys || in.NoNodeControl || in.KeysDir != "" {
		t.Fatal("omitted input overrides declaration")
	}
}

func TestRunToolSequencesDefinitionsUnlessSuiteRequested(t *testing.T) {
	keys, err := filepath.Abs("../../presets/keys")
	if err != nil {
		t.Fatal(err)
	}
	for _, suite := range []bool{false, true} {
		spec := `{"schemaVersion":"2","kind":"case","id":"c","chainPreset":{"schemaVersion":"2","kind":"chain-preset","id":"p","chain":"stablenet","topology":{"bp":1}},"steps":[{"expect":"blockNumber","is":0}]}`
		args := map[string]any{"suite": suite, "specs": []any{spec, spec}, "keysDir": keys, "binary": "/nonexistent/accepted-binary", "dataDir": t.TempDir(), "artifactRoot": t.TempDir()}
		out, err := runTool().Handler(context.Background(), args)
		if err == nil {
			t.Fatal("missing binary ran successfully")
		}
		if !suite {
			var results app.RunSuitesOut
			if json.Unmarshal([]byte(out), &results) != nil || len(results.Runs) != 2 || !results.Failed() {
				t.Fatalf("sequential results were lost: %s %v", out, err)
			}
		} else if strings.Contains(out, `"runs"`) {
			t.Fatal("suite was sequenced")
		}
	}
}
