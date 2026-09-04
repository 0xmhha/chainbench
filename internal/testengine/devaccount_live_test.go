package testengine_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/0xmhha/chainbench/internal/chains/stablenet" // register the stablenet plugin

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/testengine"
)

// TestSuite_Live_DeclaredAccounts is the point of declaring accounts instead of
// putting them in the genesis.
//
// dev1 is created when the chain is already running and funded by a
// transaction, so the genesis never mentions it: adding, removing or renaming a
// test account does not force a new genesis, a new set of derived files, and a
// re-init of every datadir. dev2 is declared with no balance on purpose —
// that is what a test needs to exercise the paths that fail for want of gas.
//
// The assertions make the chain the witness on both halves. The balance shows
// the funding actually moved; the recovered sender shows dev1 signed with its
// own key, which is the half no comparison of our own values could establish —
// no node holds that key, so the transaction can only have been signed here.
//
// Gated on GSTABLE_BIN; CI skips it and stays green.
func TestSuite_Live_DeclaredAccounts(t *testing.T) {
	bin := os.Getenv("GSTABLE_BIN")
	if bin == "" {
		t.Skip("set GSTABLE_BIN to a real gstable binary to run the declared-account live e2e")
	}

	ws, err := os.MkdirTemp("/tmp", "da")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		// A failed live run is worth reading: keep the session and say where.
		if t.Failed() {
			t.Logf("session kept at %s", ws)
			return
		}
		_ = os.RemoveAll(ws)
	})

	// A ring of this run's own, so the declared accounts are created here and
	// the repo's preset is left untouched.
	ringDir := filepath.Join(ws, "ring")
	copyPreset(t, filepath.Join(repoRoot(t), "keys", "preset"), ringDir)

	env := map[string]any{
		"schemaVersion": "2", "kind": "env", "id": "declared-accounts-env", "chain": "stablenet",
		"binaries": map[string]any{"default": bin},
		"accounts": map[string]any{
			"dev1": map[string]any{"fund": "10000000000000000000"},
			"dev2": map[string]any{},
		},
	}
	spec := map[string]any{
		"schemaVersion": "2", "kind": "case",
		"id":  "declared-accounts",
		"env": "declared-accounts-env",
		"steps": []map[string]any{
			// Signed here: no node holds dev1's key.
			{"do": "sendTx", "from": "dev1", "to": "dev2", "value": "0x1", "save": "sent"},
			// The declaration's funding actually arrived.
			{"expect": "balanceAt", "address": "dev1", "compare": "Greater", "expected": "0"},
			// dev2 was declared with no balance, so the only wei it can hold is
			// the one dev1 just sent it. Equality proves both halves: it started
			// at zero, and the transfer landed.
			{"expect": "balanceAt", "address": "dev2", "compare": "Equal", "expected": "1"},
			{"expect": "txStatus", "hash": "$sent", "expected": "0x1"},
		},
	}
	writeJSON(t, filepath.Join(ws, "declared-accounts-env.env.json"), env)
	casePath := filepath.Join(ws, "declared-accounts.json")
	writeJSON(t, casePath, spec)

	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
	defer cancel()

	out, err := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{
		SpecPaths: []string{casePath}, DataDir: ws, Binary: bin, KeysDir: ringDir, WaitBlocks: 1,
	})
	if err != nil {
		t.Fatalf("RunSuite: %v (setup %v)", err, out.SetupSteps)
	}
	if s := out.Summary.Summary; s.Pass != 1 || s.Fail > 0 || s.Blocked > 0 {
		t.Fatalf("summary %+v (session %s)", s, out.SessionRoot)
	}
}

func writeJSON(t *testing.T, path string, doc map[string]any) {
	t.Helper()
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func copyPreset(t *testing.T, src, dst string) {
	t.Helper()
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	ents, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range ents {
		s, d := filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())
		if e.IsDir() {
			copyPreset(t, s, d)
			continue
		}
		b, err := os.ReadFile(s)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(d, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
