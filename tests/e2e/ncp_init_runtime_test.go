//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// ncpInitStorageScript reuses the handoff and Web storage policy for ownership,
// free-space preflight and reference-aware disposal. Runtime paths travel
// on stdin so the inventory does not mistake this command for a live node.
const ncpInitStorageScript = `import json,sys
from pathlib import Path
sys.path.insert(0,"../../tests/webui")
from runtime_storage import create_runtime,reclaim_after_shutdown
r=json.load(sys.stdin)
if r["operation"]=="create":
 print(json.dumps({"runtime":str(create_runtime(prefix="ncpi-"))}))
else:
 print(json.dumps(reclaim_after_shutdown(Path(r["runtime"]),Path(r["runtime"])/"reports")))
`

func ncpInitStorage(ctx context.Context, operation, root string) ([]byte, error) {
	body, err := json.Marshal(map[string]string{"operation": operation, "runtime": root})
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, "python3", "-B", "-c", ncpInitStorageScript)
	cmd.Stdin = bytes.NewReader(body)
	return cmd.CombinedOutput()
}

func ncpInitRuntime(t *testing.T) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	body, err := ncpInitStorage(ctx, "create", "")
	if err != nil {
		t.Fatalf("create bounded init runtime: %v %s", err, body)
	}
	var result struct {
		Runtime string `json:"runtime"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.Runtime == "" {
		t.Fatalf("invalid owned runtime: %v %s", err, body)
	}
	root := result.Runtime
	if err := os.MkdirAll(filepath.Join(root, "reports"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Log("owned NCP init runtime:", root)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		body, err := ncpInitStorage(ctx, "reclaim", root)
		if err != nil {
			t.Error(fmt.Errorf("preserve init runtime after reclamation refusal: %w: %s", err, body))
		}
	})
	return root
}
