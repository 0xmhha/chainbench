package verb

import (
	"context"
	"github.com/0xmhha/chainbench/internal/chainsetup"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeModeWC writes a workspace-config with the given execution.chain value.
func writeModeWC(t *testing.T, chain string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "workspace-config.yaml")
	body := `version: 1
dataRoot: /data
paths: {binaries: bin, configs: configs, genesis: genesis, keystore: keystore, keyrings: keys, nodes: node, runtime: runtime, logs: logs}
control: {artifactRoot: ~/.chainbench}
inputs: {mode: generated}
execution: {chain: ` + chain + `}
`
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestUpChainMode(t *testing.T) {
	cases := []struct {
		name string
		in   chainsetup.ChainUpIn
		want string
	}{
		{"no config is fresh", chainsetup.ChainUpIn{}, "fresh"},
		{"fresh", chainsetup.ChainUpIn{WorkspaceConfigPath: writeModeWC(t, "fresh")}, "fresh"},
		{"reuse", chainsetup.ChainUpIn{WorkspaceConfigPath: writeModeWC(t, "reuse-if-matching")}, "reuse-if-matching"},
		{"attach", chainsetup.ChainUpIn{WorkspaceConfigPath: writeModeWC(t, "attach")}, "attach"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := upChainMode(tc.in)
			if err != nil {
				t.Fatalf("upChainMode: %v", err)
			}
			if string(got) != tc.want {
				t.Fatalf("mode = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestChainUp_AttachModeRefusesToCompose: execution.chain=attach must not compose
// or launch — up rejects it before touching the workspace.
func TestChainUp_AttachModeRefusesToCompose(t *testing.T) {
	_, err := ChainUp(context.Background(), chainsetup.Deps{}, chainsetup.ChainUpIn{
		DataDir:             t.TempDir(),
		Chain:               "stablenet",
		Binary:              "/bin/gwbft",
		WorkspaceConfigPath: writeModeWC(t, "attach"),
	})
	if err == nil {
		t.Fatal("expected attach mode to be refused by up")
	}
	if !strings.Contains(err.Error(), "attach does not compose") {
		t.Fatalf("error = %q, want the attach refusal", err)
	}
}
