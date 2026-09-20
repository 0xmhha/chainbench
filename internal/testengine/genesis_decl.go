package testengine

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"sort"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/consensus/upgrade"
	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/dsl"
)

// The genesis a declaration asks for: overlays, and a fork it schedules itself.
//
// An overlay is written to a file because the genesis step reads overlays from
// files, and its name carries the content's digest so two declarations asking
// for different overlays in one workspace do not overwrite each other.

func hardforkSets(forks map[string]int) []string {
	names := make([]string, 0, len(forks))
	for name := range forks {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]string, 0, len(names))
	for _, name := range names {
		out = append(out, fmt.Sprintf("%sBlock=%d", name, forks[name]))
	}
	return out
}

// writeOverlay puts a declared genesis overlay where the genesis step reads
// overlays from: a file under the workspace, written through the file seam
// like everything else the workspace holds. No overlay writes nothing and
// returns no path.
func writeOverlay(ctx context.Context, dataDir string, overlay map[string]any, provides []string, haltsAt int64) (string, error) {
	if len(overlay) == 0 && len(provides) == 0 && haltsAt == 0 {
		return "", nil
	}
	// The same {capabilities, genesis} document `chain up --genesis-overlay`
	// takes. Writing only the genesis half is why a DSL env had no way to make
	// its network advertise anything, and why cases that needed a capability
	// declared it as a requirement instead and skipped for good.
	doc := map[string]any{"genesis": overlay}
	if len(provides) > 0 {
		doc["capabilities"] = provides
	}
	if haltsAt > 0 {
		doc["haltsAt"] = haltsAt
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("render genesis overlay: %w", err)
	}
	sum := sha256.Sum256(b)
	path := filepath.Join(dataDir, fmt.Sprintf("%s-%s.json", overlayFilePrefix, hex.EncodeToString(sum[:8])))
	if err := (filestore.Local{}).Write(ctx, path, b, 0o644); err != nil {
		return "", fmt.Errorf("write genesis overlay: %w", err)
	}
	return path, nil
}

// forkOf reads the fork a declaration schedules, taking from the preset what the
// case did not say.
//
// The preset decides the fork and the block; a case may repeat them and is held
// to the repetition (see checkDeclaredFork). Here the two are folded into the
// one instruction the genesis step acts on, along with which file the case
// wants the fork carried in — empty meaning the genesis, which is the step's
// own default.
func forkOf(u *dsl.UpgradeV2) (*chainsetup.GenesisFork, error) {
	name, at := u.Fork, int64(0)
	if u.At != nil {
		at = *u.At
	}
	// The preset is read only for what the case left out. A declaration that
	// says both says everything, and a restart has no preset to read at all.
	if u.Fork == "" || u.At == nil {
		prof, err := upgrade.LoadProfile(upgradePresetPath(u))
		if err != nil {
			return nil, fmt.Errorf("upgrade preset: %w", err)
		}
		if name == "" {
			name = prof.Upgrade.AtFork
		}
		if u.At == nil {
			at = prof.Upgrade.ForkBlock
		}
	}
	if name == "" {
		return nil, fmt.Errorf("upgrade: neither the case nor its preset names a fork")
	}
	to := u.To
	if to == "" {
		to = dsl.BinaryTo
	}
	return &chainsetup.GenesisFork{
		Name: name, At: at, Binary: to,
		Carrier: chainsetup.ForkCarrier(u.Carry),
		Restart: u.Style == dsl.UpgradeRestart,
	}, nil
}

// hardforkPresetDir is where a named hardfork preset lives.
const hardforkPresetDir = "presets/hardfork"

// upgradePresetPath is the preset file this declaration names, under
// presets/hardfork.
func upgradePresetPath(u *dsl.UpgradeV2) string {
	return filepath.Join(hardforkPresetDir, u.Preset+".yaml")
}

// checkDeclaredFork holds a case to what it said about the fork.
//
// The preset decides which fork and which block; a case may repeat them, and a
// repetition that disagrees is the case testing something other than what it
// claims. Saying nothing is fine — the preset answers.
func checkDeclaredFork(u *dsl.UpgradeV2, presetPath string) error {
	if u.Fork == "" && u.At == nil {
		return nil
	}
	// A restart names no preset, so there is nothing to hold it to.
	if u.Style == dsl.UpgradeRestart {
		return nil
	}
	prof, err := upgrade.LoadProfile(presetPath)
	if err != nil {
		return fmt.Errorf("upgrade preset: %w", err)
	}
	if u.Fork != "" && u.Fork != prof.Upgrade.AtFork {
		return fmt.Errorf("the case says it tests the %q fork and %s schedules %q", u.Fork, presetPath, prof.Upgrade.AtFork)
	}
	// The block is NOT held to the preset, and the fork's name is.
	//
	// They are different kinds of fact. The name says which change is under
	// test, and a case wrong about that reports a pass for a fork it never
	// exercised — the worst failure there is. The block is a schedule this run
	// chooses: how far in it puts the fork. A case that has to act while the
	// pre-fork build is still sealing needs the fork far enough out to get the
	// work done, and the preset's height is the handoff environment's, not
	// every case's.
	//
	// Measured: with the preset's block 20 the chain reaches the fork and stops
	// during bring-up, so a case sending a transaction beforehand submits it to
	// a network that seals nothing and waits out its receipt.
	return nil
}

// writeOverlays renders one overlay file per binary, the same way the network's
// own overlay is rendered, and returns where each landed.
func writeOverlays(ctx context.Context, dataDir string, per map[string]map[string]any) (map[string]string, error) {
	if len(per) == 0 {
		return nil, nil
	}
	out := make(map[string]string, len(per))
	for _, name := range slices.Sorted(maps.Keys(per)) {
		path, err := writeOverlay(ctx, dataDir, per[name], nil, 0)
		if err != nil {
			return nil, fmt.Errorf("binary %s: %w", name, err)
		}
		out[name] = path
	}
	return out, nil
}

// sameComposition checks that every spec in a suite declares the SAME network,
// not merely the same chain.
//
// One run composes one network, and it composes it from the first spec. A spec
// further down the list that declares a different genesis, topology or binary
// does not get the network it asked for — it runs against the first spec's, and
// its assertions are answered by the wrong chain. That failure is silent, which
// is the worst kind: six genesis-string cases each declaring their own
// authorizedAccounts would all be answered by the first one's genesis and five
// of them would report a wrong count as a real result.
//
// So the disagreement is refused here, before anything is allocated, and the
