package resource_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/resource"
)

func TestBaseline_RoundTripsAndPathsBesideTheConfig(t *testing.T) {
	dir := t.TempDir()
	wc := filepath.Join(dir, "workspace-config.yaml")
	path := resource.BaselinePathFor(wc)
	if filepath.Dir(path) != dir {
		t.Fatalf("baseline should sit beside the workspace-config: %s", path)
	}

	// No baseline yet is not an error — it means none has been approved.
	got, err := resource.LoadBaseline(path)
	if err != nil || got != nil {
		t.Fatalf("missing baseline should read as none: %v / %v", got, err)
	}

	want := resource.Baseline{
		Genesis:    "sha256:aaaa",
		Configs:    map[string]string{"node1": "sha256:bbbb"},
		Validators: []string{"0xAA"},
		Note:       "first approval",
	}
	if err := resource.SaveBaseline(path, want); err != nil {
		t.Fatalf("save: %v", err)
	}
	back, err := resource.LoadBaseline(path)
	if err != nil || back == nil {
		t.Fatalf("load: %v / %v", back, err)
	}
	if back.Genesis != want.Genesis || back.Configs["node1"] != want.Configs["node1"] || back.Version != 1 {
		t.Fatalf("round trip lost data: %+v", back)
	}

	// The record must carry no key material — it is meant to be diffable.
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"password", "privateKey", "nodekey"} {
		if strings.Contains(string(raw), forbidden) {
			t.Fatalf("baseline must not record %q: %s", forbidden, raw)
		}
	}
}

func TestBaseline_CheckDetectsDrift(t *testing.T) {
	b := resource.Baseline{
		Genesis:    "sha256:approved",
		Configs:    map[string]string{"node1": "sha256:cfg1"},
		Validators: []string{"0xaa", "0xbb"},
	}
	cases := []struct {
		name string
		obs  resource.Observed
		want string // substring expected in the diff, "" for no diff
	}{
		{"identical", resource.Observed{
			Genesis: "sha256:approved", Configs: map[string]string{"node1": "sha256:cfg1"},
			Validators: []string{"0xBB", "0xAA"}, // order and case do not matter
		}, ""},
		{"genesis edited", resource.Observed{
			Genesis: "sha256:other", Configs: map[string]string{"node1": "sha256:cfg1"},
			Validators: []string{"0xaa", "0xbb"},
		}, "genesis"},
		{"config edited", resource.Observed{
			Genesis: "sha256:approved", Configs: map[string]string{"node1": "sha256:changed"},
			Validators: []string{"0xaa", "0xbb"},
		}, "node1 config"},
		{"validator swapped", resource.Observed{
			Genesis: "sha256:approved", Configs: map[string]string{"node1": "sha256:cfg1"},
			Validators: []string{"0xaa", "0xcc"},
		}, "validators"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			diffs := b.Check(tc.obs)
			joined := strings.Join(diffs, " | ")
			if tc.want == "" {
				if len(diffs) != 0 {
					t.Fatalf("expected no drift, got %s", joined)
				}
				return
			}
			if !strings.Contains(joined, tc.want) {
				t.Fatalf("expected a %q drift, got %q", tc.want, joined)
			}
		})
	}
}

// TestBaseline_UnsetFieldsAreNotChecked: a baseline that fixes only the genesis
// says nothing about configs, and silence there is not a failure.
func TestBaseline_UnsetFieldsAreNotChecked(t *testing.T) {
	b := resource.Baseline{Genesis: "sha256:g"}
	diffs := b.Check(resource.Observed{
		Genesis: "sha256:g",
		Configs: map[string]string{"node1": "anything"},
	})
	if len(diffs) != 0 {
		t.Fatalf("unrecorded fields must not be checked: %v", diffs)
	}
}
