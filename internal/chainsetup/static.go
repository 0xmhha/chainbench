package chainsetup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/keys"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/pipeline/setup"
	"github.com/0xmhha/chainbench/internal/core/place"
	"github.com/0xmhha/chainbench/internal/core/procman"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/engine"
)

// Local port layout for a bring-up. The steps clear the derived reservations:
// etcd is p2p+1, ws is http+1 and auth is http+2, so p2p_step >= 2 and
// rpc_step >= 3 keep co-located nodes from colliding.
const (
	defaultP2PBase = 30300
	defaultRPCBase = 8600
	defaultStep    = 10
	// defaultPortBand caps how many nodes one host may take.
	defaultPortBand = 50
)

// healthPollInterval is how often the health gate re-reads the head.
const healthPollInterval = time.Second

// Options are the inputs a bring-up needs. Everything a case can vary is here,
// so the CLI stays a thin translation of flags.
type Options struct {
	// Chain is the chain id for a static case.
	Chain string
	// Binary is the node executable.
	Binary string
	// KeysDir holds the preset.
	KeysDir string
	// DataDir is the network's data root.
	DataDir string
	// Validators is the block-producing node count.
	Validators int
	// HealthTimeout bounds the health gate.
	HealthTimeout time.Duration
	// StopAfter ends the run once that step completes.
	StopAfter string
}

// RunStatic brings up a static-bootstrap chain (wbft, stablenet) step by step,
// reporting each. It uses the same components the engine does, called one phase
// at a time so a failure names the phase rather than the whole bring-up.
func RunStatic(ctx context.Context, c Case, o Options, report Reporter) (Run, error) {
	if err := validateStopAfter(c, o.StopAfter); err != nil {
		return Run{Case: c}, err
	}
	if o.Binary == "" || o.DataDir == "" {
		return Run{Case: c}, fmt.Errorf("chainsetup: binary and data dir are required")
	}
	if o.KeysDir == "" {
		o.KeysDir = "keys/preset"
	}
	if o.Validators <= 0 {
		o.Validators = 4
	}
	if o.HealthTimeout <= 0 {
		o.HealthTimeout = 90 * time.Second
	}

	t := newTracker(report, o.StopAfter)
	run := Run{Case: c, DataDir: o.DataDir}

	var (
		plugin registry.ChainPlugin
		preset keys.Preset
		places []place.NodePlacement
		gen    []byte
		plan   setup.Plan
		specs  []driver.NodeSpec
		launch engine.LocalLauncher
		nodes  node.NodeSet
	)
	reqs := validatorReqs(o.Validators)

	t.do(c.Steps[0], func() (string, error) {
		p, err := registry.Get(o.Chain)
		if err != nil {
			return "", err
		}
		plugin = p
		m := p.Manifest()
		return fmt.Sprintf("%s: family %s, chain id %d, bootstrap %s", m.ID, m.ConsensusFamily, m.ChainID, m.Bootstrap.Type), nil
	})

	t.do(c.Steps[1], func() (string, error) {
		fi, err := os.Stat(o.Binary)
		if err != nil {
			return "", fmt.Errorf("binary %q: %w", o.Binary, err)
		}
		if fi.IsDir() {
			return "", fmt.Errorf("binary %q is a directory", o.Binary)
		}
		return o.Binary, nil
	})

	t.do(c.Steps[2], func() (string, error) {
		p, err := keys.LoadPreset(o.KeysDir)
		if err != nil {
			return "", err
		}
		preset = p
		return fmt.Sprintf("%d node identities, %d validators from %s", len(preset.Nodes), len(preset.Validators), o.KeysDir), nil
	})

	t.do(c.Steps[3], func() (string, error) {
		alloc := place.New(place.Config{
			P2PBase: defaultP2PBase, P2PStep: defaultStep,
			RPCBase: defaultRPCBase, RPCStep: defaultStep,
		})
		ps, err := alloc.Allocate(reqs, place.LocalStepped, place.Capacity{
			MinValidators: 1, PortBandSize: defaultPortBand,
		})
		if err != nil {
			return "", err
		}
		places = ps
		return fmt.Sprintf("%d node(s); node1 p2p=%d http=%d", len(ps), ps[0].Ports.P2P, ps[0].Ports.HTTP), nil
	})

	t.do(c.Steps[4], func() (string, error) {
		src := engine.PresetGenesisSource{KeysDir: o.KeysDir}
		b, err := src.Genesis(ctx, plugin, o.Validators)
		if err != nil {
			return "", err
		}
		gen = b
		return fmt.Sprintf("%d bytes, %d validator(s) substituted", len(b), o.Validators), nil
	})

	t.do(c.Steps[5], func() (string, error) {
		placed := make([]engine.PlacedNode, len(reqs))
		for i := range reqs {
			placed[i] = engine.PlacedNode{Req: reqs[i], Placement: places[i]}
		}
		p, err := engine.AssemblePlan(plugin, placed, gen, o.DataDir, plugin.Manifest().Capabilities)
		if err != nil {
			return "", err
		}
		plan = p
		return fmt.Sprintf("data root %s, %d node spec(s)", plan.DataRoot, len(plan.Nodes)), nil
	})

	t.do(c.Steps[6], func() (string, error) {
		launch = engine.LocalLauncher{Plugin: plugin, Binary: o.Binary, KeysDir: o.KeysDir}
		s, err := launch.Arm(plan)
		if err != nil {
			return "", err
		}
		specs = s
		return fmt.Sprintf("%d spec(s) armed; node1 config %s", len(s), filepath.Base(s[0].ConfigPath)), nil
	})

	t.do(c.Steps[7], func() (string, error) {
		if err := launch.Materialize(ctx, plan, specs); err != nil {
			return "", err
		}
		return fmt.Sprintf("genesis + %d config(s) under %s", len(specs), plan.DataRoot), nil
	})

	procs := procman.New()
	t.do(c.Steps[8], func() (string, error) {
		res, err := launch.InitAndLaunch(ctx, plan, specs)
		for _, p := range res.Procs {
			procs.TrackProc(p)
		}
		nodes = res.Nodes
		if err != nil {
			return "", err
		}
		for _, n := range nodes.Nodes {
			run.Nodes = append(run.Nodes, n.RPCURL)
		}
		return fmt.Sprintf("%d node(s) up; node1 %s", len(nodes.Nodes), nodes.Nodes[0].RPCURL), nil
	})

	t.do(c.Steps[9], func() (string, error) {
		head, err := waitHead(ctx, nodes.Nodes[0].RPCURL, 1, o.HealthTimeout)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("head %d on %s", head, nodes.Nodes[0].RPCURL), nil
	})

	run.Results = t.results
	if err := saveState(o.DataDir, nodes); err != nil && !t.halted() {
		return run, err
	}
	return run, nil
}

// validatorReqs builds n validator placement requests.
func validatorReqs(n int) []place.NodeReq {
	reqs := make([]place.NodeReq, n)
	for i := range reqs {
		reqs[i] = place.NodeReq{Name: fmt.Sprintf("node%d", i+1), Role: node.RoleValidator}
	}
	return reqs
}

// waitHead polls url until the head reaches target, returning the head it saw.
// A timeout reports the last observation, because "stuck at 0" and "stuck at 10"
// are different failures.
func waitHead(ctx context.Context, url string, target uint64, timeout time.Duration) (uint64, error) {
	c := rpc.Dial(url)
	deadline := time.Now().Add(timeout)
	var last uint64
	var lastErr error
	for time.Now().Before(deadline) {
		h, err := c.BlockNumber(ctx)
		if err == nil {
			last, lastErr = h, nil
			if h >= target {
				return h, nil
			}
		} else {
			lastErr = err
		}
		select {
		case <-ctx.Done():
			return last, ctx.Err()
		case <-time.After(healthPollInterval):
		}
	}
	if lastErr != nil {
		return last, fmt.Errorf("head never reached %d within %s (last RPC error: %v)", target, timeout, lastErr)
	}
	return last, fmt.Errorf("head stalled at %d, never reached %d within %s", last, target, timeout)
}

// saveState writes the node set so `chain status` and `chain down` can find the
// network after the bring-up command exits.
func saveState(dataDir string, ns node.NodeSet) error {
	if len(ns.Nodes) == 0 {
		return nil
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	return writeNodeSet(filepath.Join(dataDir, stateFile), ns)
}
