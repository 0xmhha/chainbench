package keys_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/0xmhha/chainbench/pkg/core/keys"
)

// presetDir resolves the repo's keys/preset directory relative to this test
// file, independent of the working directory.
func presetDir() string {
	_, file, _, _ := runtime.Caller(0)
	// this file: <repo>/pkg/core/keys/keys_test.go
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "keys", "preset")
}

func TestLoadPreset(t *testing.T) {
	p, err := keys.LoadPreset(presetDir())
	if err != nil {
		t.Fatalf("LoadPreset: %v", err)
	}
	if len(p.Validators) != 4 || len(p.BLSKeys) != 4 {
		t.Fatalf("validators=%d bls=%d, want 4/4", len(p.Validators), len(p.BLSKeys))
	}
	if p.ExtraData == "" || p.Password == "" {
		t.Errorf("extraData/password empty: %q / %q", p.ExtraData, p.Password)
	}
}

func TestTake(t *testing.T) {
	p, err := keys.LoadPreset(presetDir())
	if err != nil {
		t.Fatal(err)
	}
	two := p.Take(2)
	if len(two.Validators) != 2 || len(two.BLSKeys) != 2 {
		t.Errorf("Take(2): %d/%d", len(two.Validators), len(two.BLSKeys))
	}
	if two.ExtraData != p.ExtraData {
		t.Error("Take should preserve extraData")
	}
	if len(p.Take(0).Validators) != 4 || len(p.Take(99).Validators) != 4 {
		t.Error("Take(0)/Take(overflow) should return full set")
	}
}
