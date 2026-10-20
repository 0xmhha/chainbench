//go:build e2e

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNativeHandoffRuntimeStoragePreservesActiveDataAndEvidence(t *testing.T) {
	if os.Getenv("CHAINBENCH_HANDOFF_STORAGE_PROOF") != "1" {
		t.Skip("set CHAINBENCH_HANDOFF_STORAGE_PROOF=1 for the real storage proof")
	}
	source := filepath.Join(t.TempDir(), "case.json")
	if err := os.WriteFile(source, []byte("{\"kind\":\"storage-proof\"}\n"), 0400); err != nil {
		t.Fatal(err)
	}
	root := createHandoffRuntime(t, source)
	w := filepath.Join(root, "w")
	databases := []string{filepath.Join(w, "node1", "geth", "chaindata"), filepath.Join(w, "node1", "etcd")}
	for _, directory := range databases {
		if err := os.MkdirAll(directory, 0700); err != nil {
			t.Fatal(err)
		}
		f, err := os.OpenFile(filepath.Join(directory, "disposable"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		chunk := make([]byte, 1024*1024)
		for i := 0; i < 32; i++ {
			if _, err := f.Write(chunk); err != nil {
				f.Close()
				t.Fatal(err)
			}
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}
	}
	retained := map[string]struct {
		body string
		mode os.FileMode
	}{
		filepath.Join(w, "config_node1.toml"): {"[Node]\nName = \"owned-proof\"\n", 0640},
		filepath.Join(w, "logs", "node1.log"): {"public retained log\n", 0600},
	}
	timestamp := time.Unix(1700000000, 0)
	for path, value := range retained {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(value.body), value.mode); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, timestamp, timestamp); err != nil {
			t.Fatal(err)
		}
	}
	// A real process holds a log open. Its argv and open file both belong to
	// this fixture. No chain binary or maintenance component is mocked.
	logPath := filepath.Join(w, "logs", "node1.log")
	reader := exec.Command("/usr/bin/tail", "-f", logPath)
	if err := reader.Start(); err != nil {
		t.Fatal(err)
	}
	readerStopped := false
	t.Cleanup(func() {
		if !readerStopped {
			_ = reader.Process.Kill()
			_ = reader.Wait()
		}
	})
	err := closeHandoffRuntime(root)
	if err == nil || !strings.Contains(err.Error(), "runtime remains active after shutdown") {
		t.Fatalf("active inventory did not refuse reclamation: %v", err)
	}
	for _, directory := range databases {
		if _, err := os.Stat(filepath.Join(directory, "disposable")); err != nil {
			t.Fatal("active DB removed", err)
		}
	}
	if err := reader.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = reader.Wait()
	readerStopped = true
	if err := closeHandoffRuntime(root); err != nil {
		t.Fatal(err)
	}
	for _, directory := range databases {
		if _, err := os.Stat(directory); !os.IsNotExist(err) {
			t.Fatal("stopped DB not reclaimed", directory, err)
		}
	}
	files := map[string]string{}
	for path, value := range retained {
		body, err := os.ReadFile(path)
		if err != nil || string(body) != value.body {
			t.Fatal("evidence content changed", path, err)
		}
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != value.mode || !info.ModTime().Equal(timestamp) {
			t.Fatal("evidence permissions or time changed", path, err)
		}
		sum := sha256.Sum256(body)
		files[path] = hex.EncodeToString(sum[:])
	}
	input := filepath.Join(root, "case.json")
	info, err := os.Stat(input)
	if err != nil || info.Mode().Perm() != 0400 {
		t.Fatal("input permissions changed", err)
	}
	body, err := os.ReadFile(input)
	if err != nil || string(body) != "{\"kind\":\"storage-proof\"}\n" {
		t.Fatal("input content changed", err)
	}
	sum := sha256.Sum256(body)
	files[input] = hex.EncodeToString(sum[:])
	sources := map[string]string{}
	for _, path := range []string{"handoff_runtime_test.go", "handoff_cleanup_test.go", "handoff_cleanup_contract_test.go", "handoff_storage_native_test.go", "../../tests/webui/runtime_storage.py", "../../scripts/webui_storage.py"} {
		body, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		abs, err := filepath.Abs(path)
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(body)
		sources[abs] = hex.EncodeToString(sum[:])
	}
	proof := map[string]any{"runtime": root, "activeReclamationRefused": true, "disposableBytes": 64 * 1024 * 1024, "remainingDatabases": 0, "retainedFiles": files, "sources": sources}
	encoded, err := json.MarshalIndent(proof, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "reports", "storage-proof.json")
	if err := os.WriteFile(path, encoded, 0600); err != nil {
		t.Fatal(err)
	}
	t.Log("handoff storage proof:", path)
}
