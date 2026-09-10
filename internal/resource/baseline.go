package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/filestore"
)

// The approved baseline of a regression environment.
//
// A regression test is only a regression test if what it runs against is fixed.
// The reuse check compares what a run would compose against what is already
// there, but both sides move with the run: change a prepared genesis on the
// server and the next run happily adopts it. A baseline is the third, unmoving
// party — what the inputs hashed to when a human approved them.
//
// Two rules make it worth having, and both are the point rather than details:
// a run may only READ it (a run that rewrote the baseline would re-approve
// whatever it found, which is the failure this prevents), and updating it is an
// explicit act by a person who has looked at what changed.
//
// It records content hashes and public identities only. A key's bytes and a
// password never appear, nor their hashes — a hash of a secret is still a
// secret's fingerprint, and this file is meant to be readable and diffable.

// BaselineFile is the name the baseline is stored under, beside the
// workspace-config whose environment it fixes.
const BaselineFile = "chainbench-baseline.json"

// Baseline is the approved fingerprint of an environment's prepared inputs.
type Baseline struct {
	// Version is the record format version.
	Version int `json:"version"`
	// ApprovedAt is when a person approved this baseline (RFC3339). It is
	// stamped by the caller, not read from a clock here, so the record stays
	// reproducible.
	ApprovedAt string `json:"approvedAt,omitempty"`
	// Note is why this baseline was approved, in the approver's words.
	Note string `json:"note,omitempty"`
	// Genesis is the content hash ("sha256:…") of the genesis the environment
	// runs. Empty when the environment has no fixed genesis.
	Genesis string `json:"genesis,omitempty"`
	// Configs are the content hashes of each node's config, keyed by the node
	// label ("node1"). A node absent here is not checked.
	Configs map[string]string `json:"configs,omitempty"`
	// Validators are the validator addresses (0x-hex) the environment's keys
	// derive — public identities, never key material.
	Validators []string `json:"validators,omitempty"`
}

// BaselinePathFor returns where the baseline for a workspace-config lives: next
// to that file, so an environment and the fingerprint of what it approved travel
// together.
func BaselinePathFor(workspaceConfigPath string) string {
	return filepath.Join(filepath.Dir(workspaceConfigPath), BaselineFile)
}

// LoadBaseline reads the baseline at path. A missing file is not an error: it
// means no baseline has been approved yet, and the caller decides whether that
// is allowed (it is, unless the caller demands one).
func LoadBaseline(path string) (*Baseline, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("baseline: read %s: %w", path, err)
	}
	var out Baseline
	if err := json.Unmarshal(b, &out); err != nil {
		return nil, fmt.Errorf("baseline: parse %s: %w", path, err)
	}
	return &out, nil
}

// SaveBaseline writes a baseline. It is called only by the explicit approve
// path — never by a run, which may only read.
//
// The write goes through the file interface (filestore.Local) rather than os
// directly: the baseline sits beside the workspace-config on this machine, and
// the state-ownership rule is that files are written through that interface.
func SaveBaseline(path string, b Baseline) error {
	if b.Version == 0 {
		b.Version = 1
	}
	raw, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return fmt.Errorf("baseline: encode: %w", err)
	}
	if err := (filestore.Local{}).Write(context.Background(), path, append(raw, '\n'), 0o644); err != nil {
		return fmt.Errorf("baseline: write %s: %w", path, err)
	}
	return nil
}

// Observed is what a run actually found, in the same shape as a Baseline, for
// comparing against one.
type Observed struct {
	Genesis    string
	Configs    map[string]string
	Validators []string
}

// Check reports how observed differs from the approved baseline. An empty
// result means the environment is exactly what was approved. Only fields the
// baseline records are checked: a baseline that fixes the genesis alone says
// nothing about configs, and silence there is not a failure.
func (b Baseline) Check(o Observed) []string {
	var diffs []string
	// A field the baseline records must be observed. Skipping the comparison
	// when the observation is missing turned "we could not look" into "it
	// matches" — an empty Observed compared clean against a fully populated
	// baseline. Not recorded is still not checked; recorded but unseen is a
	// failure to check, and says so.
	if b.Genesis != "" {
		switch {
		case o.Genesis == "":
			diffs = append(diffs, "genesis: approved "+shortHash(b.Genesis)+", but none was observed")
		case b.Genesis != o.Genesis:
			diffs = append(diffs, fmt.Sprintf("genesis: approved %s, found %s", shortHash(b.Genesis), shortHash(o.Genesis)))
		}
	}
	labels := make([]string, 0, len(b.Configs))
	for label := range b.Configs {
		labels = append(labels, label)
	}
	sort.Strings(labels)
	for _, label := range labels {
		want := b.Configs[label]
		if want == "" {
			continue // the baseline fixes nothing for this label
		}
		got, ok := o.Configs[label]
		switch {
		case !ok || got == "":
			diffs = append(diffs, label+" config: approved "+shortHash(want)+", but none was observed")
		case want != got:
			diffs = append(diffs, fmt.Sprintf("%s config: approved %s, found %s", label, shortHash(want), shortHash(got)))
		}
	}
	if len(b.Validators) > 0 {
		if len(o.Validators) == 0 {
			diffs = append(diffs, "validators: approved a set, but none was observed")
		} else if d := diffAddresses(b.Validators, o.Validators); d != "" {
			diffs = append(diffs, "validators: "+d)
		}
	}
	return diffs
}

// diffAddresses compares two address sets case-insensitively, naming what each
// side has that the other does not. Addresses are public, so naming them is
// what an operator needs to see.
func diffAddresses(approved, found []string) string {
	set := func(in []string) map[string]bool {
		m := make(map[string]bool, len(in))
		for _, a := range in {
			m[strings.ToLower(a)] = true
		}
		return m
	}
	a, f := set(approved), set(found)
	var missing, extra []string
	for x := range a {
		if !f[x] {
			missing = append(missing, x)
		}
	}
	for x := range f {
		if !a[x] {
			extra = append(extra, x)
		}
	}
	if len(missing) == 0 && len(extra) == 0 {
		return ""
	}
	sort.Strings(missing)
	sort.Strings(extra)
	var parts []string
	if len(missing) > 0 {
		parts = append(parts, "approved but not running: "+strings.Join(missing, ", "))
	}
	if len(extra) > 0 {
		parts = append(parts, "running but not approved: "+strings.Join(extra, ", "))
	}
	return strings.Join(parts, "; ")
}

// shortHash renders a content hash short enough to read but long enough to tell
// two apart.
func shortHash(h string) string {
	if i := strings.IndexByte(h, ':'); i >= 0 && len(h) > i+13 {
		return h[:i+13]
	}
	return h
}
