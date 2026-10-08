package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/resource"
)

func TestWebControlBinaryAcceptsOnlyRegisteredOwnedExecutables(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "registered-source")
	bytes := []byte("verified binary fixture")
	if err := os.WriteFile(source, bytes, 0o700); err != nil {
		t.Fatal(err)
	}
	config := deploymentTestConfig()
	var declaration map[string]any
	if err := json.Unmarshal(config.Content, &declaration); err != nil {
		t.Fatal(err)
	}
	declaration["dataRoot"] = root
	content, err := json.Marshal(declaration)
	if err != nil {
		t.Fatal(err)
	}
	config.Content = content
	payload := webChainPayload{ExecutionBinary: source, Binary: ManifestBinaryEvidence{Chain: "wbft", SHA256: manifestHash(bytes)}, Config: DeploymentDocument{DeploymentDocumentInput: config}}

	wc, err := deploymentWorkspace(payload.Config.DeploymentDocumentInput)
	if err != nil {
		t.Fatal(err)
	}
	native, err := wc.Resolve(resource.PurposeBinaries, payload.Binary.SHA256+"/gwbft")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.MkdirAll(filepath.Dir(native), 0o700); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(native, bytes, 0o700); err != nil {
		t.Fatal(err)
	}
	state := State{Target: resource.Spec{DataRoot: root}}
	for _, binary := range []string{source, native} {
		state.Binary = binary
		got, err := bindWebControlBinary(context.Background(), state, payload, nil)
		if err != nil || got != binary {
			t.Fatalf("registered executable rejected: %s %s %v", binary, got, err)
		}
	}
	unrelated := filepath.Join(root, "unrelated")
	if err = os.WriteFile(unrelated, bytes, 0o700); err != nil {
		t.Fatal(err)
	}
	state.Binary = unrelated
	if _, err = bindWebControlBinary(context.Background(), state, payload, nil); !errors.Is(err, ErrDeploymentConflict) {
		t.Fatal("same bytes at undeclared executable path accepted", err)
	}
	state.Binary = native
	if err = os.WriteFile(native, []byte("substituted bytes"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err = bindWebControlBinary(context.Background(), state, payload, nil); !errors.Is(err, ErrDeploymentConflict) {
		t.Fatal("owned file replacement accepted", err)
	}
	if err = os.Remove(native); err != nil {
		t.Fatal(err)
	}
	if _, err = bindWebControlBinary(context.Background(), state, payload, nil); err == nil {
		t.Fatal("missing execution binary accepted")
	}
}
