package mcp

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// tool finds a registered tool by name.
func tool(t *testing.T, name string) Tool {
	t.Helper()
	for _, x := range keyringTools() {
		if x.Name == name {
			return x
		}
	}
	t.Fatalf("no tool named %q", name)
	return Tool{}
}

// call runs a tool and decodes its JSON result.
func call(t *testing.T, name string, args map[string]any, into any) {
	t.Helper()
	out, err := tool(t, name).Handler(context.Background(), args)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if into != nil {
		if err := json.Unmarshal([]byte(out), into); err != nil {
			t.Fatalf("%s did not return JSON: %v\n%s", name, err, out)
		}
	}
}

type ringResult struct {
	Keyring    string `json:"keyring"`
	Source     string `json:"source"`
	Validators int    `json:"validators"`
	Entries    []struct {
		Label      string `json:"label"`
		Address    string `json:"address"`
		BLSPubKey  string `json:"blsPublicKey"`
		Validator  bool   `json:"validator"`
		PrivateKey string `json:"privateKey"`
	} `json:"entries"`
}

// TestKeyringTools_DriveTheSameUseCases is the point of the MCP surface: an
// agent creates, extends and reads a ring through the same functions the CLI
// calls, so the two cannot answer differently.
func TestKeyringTools_DriveTheSameUseCases(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ring")

	var created ringResult
	call(t, "chainbench_keyring_new", map[string]any{
		"keyringDir": dir, "count": float64(3), "validators": float64(2), "withBls": true,
	}, &created)

	if len(created.Entries) != 3 || created.Validators != 2 {
		t.Fatalf("new: %d identities, %d validators", len(created.Entries), created.Validators)
	}
	if created.Keyring != dir || created.Source != "explicit" {
		t.Errorf("the result does not say which ring it used: %+v", created)
	}
	for _, e := range created.Entries {
		if e.BLSPubKey == "" {
			t.Errorf("%s has no BLS material despite withBls", e.Label)
		}
		if e.PrivateKey != "" {
			t.Errorf("%s leaked its private key to the agent", e.Label)
		}
	}

	// Adding does not promote: the defect this suite exists to catch.
	var extended ringResult
	call(t, "chainbench_keyring_add", map[string]any{
		"keyringDir": dir, "count": float64(2), "withBls": true,
	}, &extended)
	if len(extended.Entries) != 5 {
		t.Fatalf("add: %d identities, want 5", len(extended.Entries))
	}
	if extended.Validators != 2 {
		t.Errorf("add changed the validator set to %d, want 2", extended.Validators)
	}

	var listed ringResult
	call(t, "chainbench_keyring_list", map[string]any{"keyringDir": dir, "verify": true}, &listed)
	if len(listed.Entries) != 5 {
		t.Errorf("list: %d identities, want 5", len(listed.Entries))
	}
}

// TestKeyringTools_ShowNeverReturnsASecret keeps a private key out of an
// agent's transcript.
func TestKeyringTools_ShowNeverReturnsASecret(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "ring")
	call(t, "chainbench_keyring_new", map[string]any{"keyringDir": dir, "count": float64(1)}, nil)

	out, err := tool(t, "chainbench_keyring_show").Handler(context.Background(),
		map[string]any{"keyringDir": dir, "name": "node1"})
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	if strings.Contains(out, "privateKey") {
		t.Errorf("show returned a private key:\n%s", out)
	}
}

// TestKeyringTools_NoExportTool is a deliberate absence, not an omission: an
// agent's transcript is not a place for a secret, and the CLI's --yes guard has
// no equivalent where no human is at the keyboard.
func TestKeyringTools_NoExportTool(t *testing.T) {
	for _, x := range keyringTools() {
		if strings.Contains(x.Name, "export") {
			t.Fatalf("the MCP surface exposes %q", x.Name)
		}
	}
}

// TestKeyringTools_AreRegistered checks the group reaches the default server,
// since a tool nobody registered is a tool no agent can call.
func TestKeyringTools_AreRegistered(t *testing.T) {
	s := Default("chainbench", "test")
	for _, x := range keyringTools() {
		if _, ok := s.tools[x.Name]; !ok {
			t.Errorf("%s is not registered", x.Name)
		}
	}
}
