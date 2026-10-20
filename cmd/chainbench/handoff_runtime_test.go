//go:build e2e

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/app"
)

// The shared storage module owns the ledger, free-space preflight, open-file
// inventory and reference-aware reclamation. Paths arrive over stdin so the
// maintenance process's argv is not mistaken for a live node using the runtime.
const handoffStorageScript = `import json,sys
from pathlib import Path
sys.path.insert(0,"../../tests/webui")
from runtime_storage import create_runtime,reclaim_after_shutdown,validate_runtime
request=json.load(sys.stdin)
if request["operation"]=="create":
 print(json.dumps({"runtime":str(create_runtime(prefix="hf-"))}))
elif request["operation"]=="validate":
 print(json.dumps(str(validate_runtime(request["runtime"]))))
else:
 print(json.dumps(reclaim_after_shutdown(Path(request["runtime"]),Path(request["runtime"])/"reports")))
`

func handoffStorage(ctx context.Context, operation, runtime string) ([]byte, error) {
	body, err := json.Marshal(map[string]string{"operation": operation, "runtime": runtime})
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, "python3", "-B", "-c", handoffStorageScript)
	cmd.Stdin = bytes.NewReader(body)
	return cmd.CombinedOutput()
}

func createHandoffRuntime(t *testing.T, spec string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	body, err := handoffStorage(ctx, "create", "")
	if err != nil {
		t.Fatalf("create bounded handoff runtime: %v\n%s", err, body)
	}
	var result struct {
		Runtime string `json:"runtime"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.Runtime == "" {
		t.Fatalf("runtime ownership response: %v %s", err, body)
	}
	root := result.Runtime
	if err := os.MkdirAll(filepath.Join(root, "reports"), 0700); err != nil {
		t.Fatal(err)
	}
	// Preserve the public input before the test's TempDir cleanup removes it.
	if body, err := exec.Command("/bin/cp", "-p", spec, filepath.Join(root, "case.json")).CombinedOutput(); err != nil {
		t.Fatalf("preserve handoff input: %v %s", err, body)
	}
	t.Log("owned handoff runtime:", root)
	t.Cleanup(func() {
		if err := closeHandoffRuntime(root); err != nil {
			t.Error(err)
		}
	})
	return root
}

func stopHandoffWorkspace(ctx context.Context, workspace string) error {
	if _, err := os.Stat(filepath.Join(workspace, "chain-record.json")); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		// No chain record is safe only before any process ledger exists. A missing
		// record with a ledger cannot be interpreted as "nothing was launched".
		if _, err := os.Stat(filepath.Join(workspace, "process.json")); !os.IsNotExist(err) {
			if err != nil {
				return err
			}
			return fmt.Errorf("handoff: process ledger exists without a chain record; preserve %s", workspace)
		}
		return nil
	}
	_, err := app.ChainStop(ctx, app.Deps{}, app.ChainStopIn{DataDir: workspace})
	return err
}

func closeHandoffRuntime(root string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if body, err := handoffStorage(ctx, "validate", root); err != nil {
		return fmt.Errorf("handoff: runtime ownership unverified: %w: %s", err, body)
	}
	return finishHandoffWorkspace(ctx, filepath.Join(root, "w"), stopHandoffWorkspace, func(ctx context.Context, _ string) error {
		body, err := handoffStorage(ctx, "reclaim", root)
		if err != nil {
			return fmt.Errorf("handoff: preserve runtime after reclamation refusal: %w: %s", err, strings.TrimSpace(string(body)))
		}
		return nil
	})
}

func preserveHandoffRun(t *testing.T, root string, output string, runErr error) {
	t.Helper()
	if runErr != nil {
		output += "\nerror: " + runErr.Error() + "\n"
	}
	if err := os.WriteFile(filepath.Join(root, "run.log"), []byte(output), 0600); err != nil {
		t.Fatal(err)
	}
}
