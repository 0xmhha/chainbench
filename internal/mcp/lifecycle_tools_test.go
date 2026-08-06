package mcp_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestStopTool launches a real short-lived process, records its PID in a
// nodeset, and verifies the stop tool terminates it. This exercises the real
// LocalDriver stop path without needing a chain binary.
func TestStopTool(t *testing.T) {
	proc := exec.Command("sleep", "30")
	if err := proc.Start(); err != nil {
		t.Skipf("cannot start sleep: %v", err)
	}
	pid := proc.Process.Pid
	// reap the process so it does not linger if the assertion below fails.
	t.Cleanup(func() { _ = proc.Process.Kill(); _, _ = proc.Process.Wait() })

	dir := t.TempDir()
	nsJSON := `{"chain":"stablenet","network":"local","nodes":[{"index":1,"role":"validator","rpc_url":"http://127.0.0.1:8501","pid":` +
		itoa(pid) + `}]}`
	if err := os.WriteFile(filepath.Join(dir, "nodeset.json"), []byte(nsJSON), 0o644); err != nil {
		t.Fatal(err)
	}

	text, isErr := callText(t, newServer(), "chainbench_stop", map[string]any{"data_dir": dir})
	if isErr || !strings.Contains(text, "stopped 1 node") {
		t.Fatalf("stop: err=%v text=%s", isErr, text)
	}
	// the process should now be gone: Wait returns promptly.
	done := make(chan error, 1)
	go func() { _, err := proc.Process.Wait(); done <- err }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Error("process still running after stop")
	}
}

// itoa avoids importing strconv just for one int.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
