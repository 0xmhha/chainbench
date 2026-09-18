package process

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// fakeExec runs `sh -c <script>` instead of the real binary so Launch is
// testable without a node binary.
func fakeExec(script string) ExecFn {
	return func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sh", "-c", script)
	}
}

func TestLocalDriver_ProvisionWritesConfig(t *testing.T) {
	dir := t.TempDir()
	d := NewLocalDriver()
	spec := NodeSpec{
		Index:         1,
		DataDir:       filepath.Join(dir, "node1"),
		ConfigPath:    filepath.Join(dir, "config_node1.toml"),
		ConfigContent: []byte("[Eth]\n"),
	}
	if err := d.Provision(context.Background(), spec); err != nil {
		t.Fatalf("provision: %v", err)
	}
	if _, err := os.Stat(spec.DataDir); err != nil {
		t.Errorf("datadir not created: %v", err)
	}
	b, err := os.ReadFile(spec.ConfigPath)
	if err != nil || !strings.Contains(string(b), "[Eth]") {
		t.Errorf("config not written: %v (%s)", err, b)
	}
}

func TestLocalDriver_LaunchWritesLog(t *testing.T) {
	dir := t.TempDir()
	d := NewLocalDriverWithExec(fakeExec(`echo "node up"; exit 0`))
	spec := NodeSpec{
		Index:   1,
		Binary:  "gstable",
		DataDir: dir,
		LogPath: filepath.Join(dir, "logs", "node1.log"),
	}
	h, err := d.Launch(context.Background(), spec)
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	if h.PID == 0 {
		t.Error("expected non-zero pid")
	}
	// Give the reaper goroutine time to flush + close the log.
	deadline := time.Now().Add(2 * time.Second)
	var content []byte
	for time.Now().Before(deadline) {
		content, _ = os.ReadFile(spec.LogPath)
		if strings.Contains(string(content), "node up") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !strings.Contains(string(content), "node up") {
		t.Errorf("log not captured: %q", content)
	}
}

func TestOffset(t *testing.T) {
	base := node.Endpoints{P2P: 30301, HTTP: 8501, WS: 9501, Auth: 8551, Metrics: 6061}
	got := node.Offset(base, 2)
	want := node.Endpoints{P2P: 30303, HTTP: 8503, WS: 9503, Auth: 8553, Metrics: 6063}
	if got != want {
		t.Errorf("Offset: got %+v, want %+v", got, want)
	}
}
