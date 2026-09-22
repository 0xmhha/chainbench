package testengine

import (
	"github.com/0xmhha/chainbench/internal/core/lifecycle"

	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/registry"
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

// forkOf reads the fork a declaration schedules.
//
// The declaration says both the fork's name and its block. It used to say
// neither and take them from a second document, a hardfork preset the case
// named — so standing up a handoff meant reading two files, and which of them
// answered depended on what the first one left out. The second document is gone
// and this reads one.
func forkOf(u *dsl.UpgradeV2) (*chainsetup.GenesisFork, error) {
	if u.Fork == "" {
		return nil, lifecycle.Mark(errIncomplete, fmt.Errorf("upgrade: no fork named — a hardfork declaration says which fork it crosses"))
	}
	if u.At == nil {
		return nil, lifecycle.Mark(errIncomplete, fmt.Errorf("upgrade: the %q fork has no block — a hardfork declaration says where it activates", u.Fork))
	}
	to := u.To
	if to == "" {
		to = dsl.BinaryTo
	}
	return &chainsetup.GenesisFork{
		Name: u.Fork, At: *u.At, Binary: to,
		Carrier: chainsetup.ForkCarrier(u.Carry),
		Restart: u.Style == dsl.UpgradeRestart,
	}, nil
}

// checkForkIsOneTheChainKnows refuses a fork the binary taking over has never
// heard of.
//
// This is what a second document used to be for, and it is a different check.
// Comparing the case against a preset only said the two agreed; two documents
// can agree and both be wrong, and only the two declarations that named a preset
// were compared at all. A chain-manifest lists the forks its build knows, so
// asking it catches a misspelling in any declaration — and a misspelling is
// silent otherwise, because the genesis takes whatever name it is given and
// writes <name>Block, and the chain simply never forks.
//
// The chain asked is the one the POST-fork binary runs, which is not always the
// network's: wemix does not know croissant and the wbft build that takes over
// from it does.
func checkForkIsOneTheChainKnows(u *dsl.UpgradeV2, networkChain string, binaryChains map[string]string) error {
	to := u.To
	if to == "" {
		to = dsl.BinaryTo
	}
	chainID := binaryChains[to]
	if chainID == "" {
		chainID = networkChain
	}
	p, err := registry.Get(chainID)
	if err != nil {
		// An external manifest is resolved later and is not in the registry.
		// Refusing here would reject a chain the run can perfectly well
		// compose, so an unknown id is left to the step that resolves it.
		return nil
	}
	known := p.Manifest().Genesis.Hardforks
	if slices.Contains(known, u.Fork) {
		return nil
	}
	return lifecycle.Mark(errUnknownName, fmt.Errorf("the declaration crosses the %q fork and %s does not know it (it knows %s)",
		u.Fork, chainID, strings.Join(known, ", ")))
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
