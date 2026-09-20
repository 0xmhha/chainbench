package testengine

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/0xmhha/chainbench/internal/dsl"
)

// Whether a composed network can be reused for the next spec.
//
// Two declarations share a network when they ask for the same one — same chain,
// same shape, same genesis. The key is computed from the request rather than
// from what was built, so a reuse decision can be made before composing.

// message names what differs so the caller can split the run.
func sameComposition(specs []dsl.Spec) error {
	if len(specs) < 2 {
		return nil
	}
	want := compositionKey(specs[0])
	var others []string
	for _, s := range specs[1:] {
		if compositionKey(s) != want {
			others = append(others, s.ID)
		}
	}
	if len(others) == 0 {
		return nil
	}
	return fmt.Errorf(
		"one run composes one network, from the first spec (%s); these declare a different one: %s. "+
			"run them separately, or give them the same env",
		specs[0].ID, strings.Join(others, ", "))
}

// compositionKey is what makes two specs the same network to compose: the
// binaries, genesis, config, topology, hardforks and placement. It deliberately
// mirrors the reuse fingerprint's inputs — a run that may share one network is
// exactly a run whose specs would fingerprint alike.
func compositionKey(s dsl.Spec) string {
	key := struct {
		Binary    string            `json:"binary"`
		Binaries  map[string]string `json:"binaries"`
		Config    string            `json:"config"`
		Genesis   map[string]any    `json:"genesis"`
		Topology  map[string]any    `json:"topology"`
		Hardforks map[string]int    `json:"hardforks"`
		Placement string            `json:"placement"`
	}{
		Binary: s.Chain.Binary, Binaries: s.Chain.Binaries, Config: s.Chain.Config,
		Genesis: s.Chain.GenesisOverlay, Topology: s.Topology,
		Hardforks: s.Hardforks, Placement: s.Placement,
	}
	b, err := json.Marshal(key)
	if err != nil {
		return fmt.Sprintf("composition-error:%v", err)
	}
	return string(b)
}

// sameChain checks that every parsed spec declares the chain the first one
// does: one suite composes one network.
func sameChain(specs []dsl.Spec) error {
	if len(specs) == 0 {
		return nil
	}
	want := specs[0].Chain.Name
	var others []string
	for _, s := range specs[1:] {
		if s.Chain.Name != want {
			others = append(others, s.ID+"="+s.Chain.Name)
		}
	}
	if len(others) > 0 {
		return fmt.Errorf("every spec in a suite must declare one chain; %s declares %s, but: %s",
			specs[0].ID, want, strings.Join(others, ", "))
	}
	return nil
}
