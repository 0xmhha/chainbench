package suitecmd_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/mcp"
)

func TestNativeSuiteMatrixCLIAndMCP(t *testing.T) {
	binaries := map[string]string{}
	for chain, env := range map[string]string{"stablenet": "GSTABLE_BIN", "wbft": "GWBFT_BIN", "wemix": "GWEMIX_BIN"} {
		if os.Getenv(env) == "" {
			t.Skip("native matrix proof requires " + env)
		}
		binary, err := filepath.Abs(os.Getenv(env))
		if err != nil {
			t.Fatal(err)
		}
		binaries[chain] = binary
	}
	rootBase := os.Getenv("WEBUI_RUNTIME_ROOT")
	if rootBase == "" {
		rootBase = t.TempDir()
	}
	root, err := os.MkdirTemp(rootBase, "matrix-native-")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if t.Failed() {
			t.Log("failed matrix runtime retained:", root)
		}
	}()
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Minute)
	defer cancel()
	chains := []string{"stablenet", "wbft", "wemix"}
	workspaces := []string{filepath.Join(root, "cli-workspace"), filepath.Join(root, "mcp-workspace")}
	cleanupWorkspace := func(workspace string) {
		cleanupCtx, stop := context.WithTimeout(context.Background(), 2*time.Minute)
		defer stop()
		for index := range chains {
			dir := filepath.Join(workspace, "matrix", fmt.Sprintf("preset-%03d", index+1))
			if _, err := os.Stat(dir); os.IsNotExist(err) {
				continue
			}
			_, _ = app.ChainStop(cleanupCtx, app.Deps{}, app.ChainStopIn{DataDir: dir})
			preserveMatrixDiagnostics(t, dir, filepath.Join(root, "diagnostics", filepath.Base(workspace), fmt.Sprintf("preset-%03d", index+1)))
			if _, err := app.ChainRm(cleanupCtx, app.Deps{}, app.ChainRmIn{DataDir: dir}); err != nil {
				t.Error("matrix runtime cleanup", err)
			}
		}
	}
	t.Cleanup(func() {
		for _, workspace := range workspaces {
			cleanupWorkspace(workspace)
		}
	})
	presets := []string{}
	for _, chain := range chains {
		env := map[string]string{"stablenet": "GSTABLE_BIN", "wbft": "GWBFT_BIN", "wemix": "GWEMIX_BIN"}[chain]
		logical := map[string]string{"stablenet": "gstable", "wbft": "gwbft", "wemix": "gwemix"}[chain]
		raw, _ := json.Marshal(map[string]any{"schemaVersion": "2", "kind": "chain-preset", "id": "native-" + chain, "chain": chain, "topology": map[string]any{"bp": 4}, "binaries": map[string]any{"default": "${" + env + ":-" + logical + "}"}, "values": map[string]any{"chainId": 77773}})
		path := filepath.Join(root, chain+".json")
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		presets = append(presets, path)
	}
	paths, raws := []string{}, []string{}
	for index, step := range []map[string]any{{"expect": "chainId", "is": 77773}, {"expect": "blockNumber", "compare": "GreaterOrEqual", "is": 1}} {
		raw, _ := json.Marshal(map[string]any{"schemaVersion": "2", "kind": "case", "id": fmt.Sprintf("shared-%d", index+1), "requires": []string{"rpc"}, "chainPreset": "original-reference", "steps": []any{step}})
		path := filepath.Join(root, fmt.Sprintf("case-%d.json", index+1))
		if err := os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
		raws = append(raws, string(raw))
	}
	requireMatrixDisk(t, root)
	args := []string{"run", "--json", "--no-skips", "--chain-id", "77773", "--wait-blocks", "1", "--read-only-keys", "--keys", repoKeys(t), "--workspace-dir", workspaces[0], "--artifact-root", filepath.Join(root, "cli-artifacts")}
	for _, preset := range presets {
		args = append(args, "--chain-presets", preset)
	}
	stdout, progress, err := runSplit(t, append(args, paths...)...)
	if err != nil {
		t.Fatal("native CLI matrix", err, stdout, progress)
	}
	cli := requireNativeMatrixVerdicts(t, []byte(stdout), false)
	if err := os.WriteFile(filepath.Join(root, "cli-report.json"), []byte(stdout), 0600); err != nil {
		t.Fatal(err)
	}
	t.Log("native CLI matrix passed all three chains")
	cleanupWorkspace(workspaces[0])
	requireMatrixDisk(t, root)
	// An intentional ID mismatch fails the middle target; the last chain still
	// runs, and all verdicts retain their selected declaration and own session.
	presetRaw, err := os.ReadFile(presets[1])
	if err != nil {
		t.Fatal(err)
	}
	var badPreset map[string]any
	_ = json.Unmarshal(presetRaw, &badPreset)
	badPreset["values"].(map[string]any)["chainId"] = 77774
	modified, _ := json.Marshal(badPreset)
	mcpPresets := append([]string(nil), presets...)
	mcpPresets[1] = filepath.Join(root, "wbft-mismatch.json")
	if err := os.WriteFile(mcpPresets[1], modified, 0600); err != nil {
		t.Fatal(err)
	}
	request, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "chainbench_run", "arguments": map[string]any{"specs": raws, "chainPresets": mcpPresets, "dataDir": workspaces[1], "artifactRoot": filepath.Join(root, "mcp-artifacts"), "chainId": 77773, "waitBlocks": 1, "readOnlyKeys": true, "keysDir": repoKeys(t), "noSkips": true}}})
	response := mcp.Default("matrix", "native").Handle(ctx, request)
	var envelope struct {
		Result struct {
			Content []struct{ Text string }
			IsError bool
		}
		Error any
	}
	if err := json.Unmarshal(response, &envelope); err != nil || envelope.Error != nil || !envelope.Result.IsError || len(envelope.Result.Content) != 1 {
		t.Fatal("native MCP matrix", err, string(response))
	}
	mcpReport := requireNativeMatrixVerdicts(t, []byte(envelope.Result.Content[0].Text), true)
	if err := os.WriteFile(filepath.Join(root, "mcp-report.json"), []byte(envelope.Result.Content[0].Text), 0600); err != nil {
		t.Fatal(err)
	}
	sources := map[string]string{}
	for _, pattern := range []string{"matrix*.go", "run.go", "../../../internal/app/runmatrix*.go", "../../../internal/app/workflow.go", "../../../internal/app/start.go", "../../../internal/app/runsuites.go", "../../../internal/dsl/*.go", "../../../internal/core/session/*.go", "../../../internal/core/report/*.go", "../../../internal/testengine/engine_impl.go", "../../../internal/testengine/summary.go", "../../../internal/testengine/suite.go", "../../../internal/testengine/runmachine.go", "../../../internal/mcp/run*.go", "../../../internal/mcp/chain_id_input*.go", "../../../internal/app/preset_provenance_test.go", "../../../internal/app/testcase.go", "../../../internal/mcp/server.go", "../../../internal/feature/test.go"} {
		files, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range files {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(raw)
			sources[path] = hex.EncodeToString(sum[:])
		}
	}
	binaryHashes := map[string]string{}
	for chain, path := range binaries {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(raw)
		binaryHashes[chain] = hex.EncodeToString(sum[:])
	}
	proof, _ := json.MarshalIndent(map[string]any{"cli": cli, "mcp": mcpReport, "binaries": binaries, "binarySHA256": binaryHashes, "sources": sources, "intentionalFailure": "WBFT preset declares 77774 while the native runtime uses 77773; the later WEMIX target still runs"}, "", "  ")
	if err := os.WriteFile(filepath.Join(root, "proof.json"), proof, 0600); err != nil {
		t.Fatal(err)
	}
	t.Log("native matrix proof:", filepath.Join(root, "proof.json"))
}

func requireNativeMatrixVerdicts(t *testing.T, raw []byte, middleFails bool) app.RunMatrixOut {
	t.Helper()
	var out app.RunMatrixOut
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err, string(raw))
	}
	wantPass, wantFail := 6, 0
	if middleFails {
		wantPass, wantFail = 4, 2
	}
	if len(out.Runs) != 3 || out.Summary.Pass != wantPass || out.Summary.Fail != wantFail || out.Summary.Blocked != 0 || out.Summary.Skip != 0 || out.SetupErrors != 0 {
		t.Fatal("wrong native matrix verdict", string(raw))
	}
	sessions := map[string]bool{}
	for index, row := range out.Runs {
		chain := []string{"stablenet", "wbft", "wemix"}[index]
		if row.PresetID != "native-"+chain || row.Chain != chain || !row.Attempted || !row.Stopped || row.CleanupFailed || row.Error != "" || len(row.Tests) != 2 || sessions[row.Session] {
			t.Fatalf("target provenance or lifecycle failed: %+v", row)
		}
		sessions[row.Session] = true
		for _, test := range row.Tests {
			want := "pass"
			if middleFails && index == 1 {
				want = "fail"
			}
			if test.PresetID != row.PresetID || test.Status != want {
				t.Fatal("per-test preset or verdict", test)
			}
		}
		for _, name := range []string{"session.json", "report.json"} {
			content, err := os.ReadFile(filepath.Join(row.Session, name))
			if err != nil {
				t.Fatal(err)
			}
			var record struct{ Tests []struct{ PresetID string } }
			if err := json.Unmarshal(content, &record); err != nil {
				t.Fatal(err)
			}
			for _, test := range record.Tests {
				if test.PresetID != row.PresetID {
					t.Fatal("persistent preset lost", name, string(content))
				}
			}
		}
		statuses, err := filepath.Glob(filepath.Join(row.Session, "tests", "*", "status.json"))
		if err != nil || len(statuses) != 2 {
			t.Fatal(err, statuses)
		}
		for _, path := range statuses {
			content, _ := os.ReadFile(path)
			var status struct{ PresetID string }
			if err := json.Unmarshal(content, &status); err != nil || status.PresetID != row.PresetID {
				t.Fatal("status preset lost", err, string(content))
			}
		}
	}
	return out
}

func preserveMatrixDiagnostics(t *testing.T, source, destination string) {
	t.Helper()
	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		name := entry.Name()
		keep := strings.HasSuffix(name, ".log") || name == "genesis.json" || name == "chain-record.json" || name == "process.json" || strings.HasPrefix(name, "config")
		if !keep {
			return nil
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		out := filepath.Join(destination, rel)
		if err := os.MkdirAll(filepath.Dir(out), 0700); err != nil {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if err := os.WriteFile(out, raw, info.Mode().Perm()); err != nil {
			return err
		}
		if owner, ok := info.Sys().(*syscall.Stat_t); ok {
			if err := os.Chown(out, -1, int(owner.Gid)); err != nil {
				return err
			}
		}
		return os.Chtimes(out, info.ModTime(), info.ModTime())
	})
	if err != nil {
		t.Error("preserve native matrix diagnostics", err)
	}
}

func requireMatrixDisk(t *testing.T, root string) {
	t.Helper()
	var info syscall.Statfs_t
	if err := syscall.Statfs(root, &info); err != nil {
		t.Fatal(err)
	}
	free := uint64(info.Bavail) * uint64(info.Bsize)
	if free < 20<<30 {
		t.Fatalf("native matrix needs 20 GiB free before starting each surface; available=%d", free)
	}
}
