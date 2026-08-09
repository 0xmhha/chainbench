package chainsetup

import (
	"context"
	"fmt"
	"os"

	"github.com/0xmhha/chainbench/internal/core/keys"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// RunWemix walks the standalone wemix bring-up. The framework performs the first
// three steps; the rest are modelled but not built, so the run reports exactly
// where support ends instead of failing with an unrelated error.
//
// The point is to make the gap legible: the pieces the missing steps would need
// (poa.Config, poa.GenerateGenesis, DeployGovernance, EtcdInit) all exist, but
// nothing orchestrates them outside the handoff CLI.
func RunWemix(ctx context.Context, c Case, o Options, report Reporter) (Run, error) {
	if err := validateStopAfter(c, o.StopAfter); err != nil {
		return Run{Case: c}, err
	}
	if o.KeysDir == "" {
		o.KeysDir = "keys/preset"
	}
	t := newTracker(report, o.StopAfter)
	run := Run{Case: c, DataDir: o.DataDir}

	t.do(c.Steps[0], func() (string, error) {
		p, err := registry.Get("wemix")
		if err != nil {
			return "", err
		}
		m := p.Manifest()
		return fmt.Sprintf("%s: family %s, chain id %d, bootstrap %s", m.ID, m.ConsensusFamily, m.ChainID, m.Bootstrap.Type), nil
	})

	t.do(c.Steps[1], func() (string, error) {
		if o.Binary == "" {
			return "", fmt.Errorf("--binary is required (go-wemix build/bin/gwemix)")
		}
		if _, err := os.Stat(o.Binary); err != nil {
			return "", fmt.Errorf("binary %q: %w", o.Binary, err)
		}
		return o.Binary, nil
	})

	t.do(c.Steps[2], func() (string, error) {
		p, err := keys.LoadPreset(o.KeysDir)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%d node identities from %s", len(p.Nodes), o.KeysDir), nil
	})

	// Everything past here is modelled but unbuilt; the tracker reports each as
	// TODO rather than pretending it ran.
	for _, s := range c.Steps[3:] {
		t.do(s, nil)
	}

	run.Results = t.results
	_ = ctx
	return run, nil
}
