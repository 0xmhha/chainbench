package chainsetup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/netmap"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/place"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/engine"
)

// RunWemix walks the standalone wemix bring-up, reporting each step.
//
// It composes the same pieces the engine and the step surfaces do rather than
// carrying its own copy of the procedure: the placement comes from netmap, the
// genesis from the binary via WemixGenesisSource, the phase order from the
// consensus family, and the bootstrap actions from WemixBootstrap. What this
// adds is the reporting — each phase called on its own, so a failure names the
// step instead of the whole bring-up.
func RunWemix(ctx context.Context, c Case, o Options, report Reporter) (Run, error) {
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
		preset keyring.Preset
		assign *netmap.Map
		art    engine.GenesisArtifacts
		plan   driver.Plan
		specs  []driver.NodeSpec
		launch engine.LocalLauncher
		phases []registry.Phase
		nodes  node.NodeSet
		boot   node.Node
	)
	roles := make([]node.Role, o.Validators)
	for i := range roles {
		roles[i] = node.RoleBP
	}

	t.do(c.Steps[0], func() (string, error) {
		p, err := registry.Get("wemix")
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
		p, err := store.LoadPreset(o.KeysDir)
		if err != nil {
			return "", err
		}
		preset = p
		return fmt.Sprintf("%d node identities from %s", len(preset.Nodes), o.KeysDir), nil
	})

	t.do(c.Steps[3], func() (string, error) {
		// The port span is the family's, not this file's: poa reserves the
		// etcd peer and client ports beside p2p, and a layout that forgot them
		// puts the next node's p2p where this node's etcd goes.
		res := plugin.Family().PortReservation()
		pool := netmap.Pool{
			Hosts: []netmap.Host{{Name: "local", Addr: "127.0.0.1"}},
			Slots: defaultPortBand,
			Ports: netmap.Bands{
				P2PBase: defaultP2PBase, P2PStep: defaultStep,
				RPCBase: defaultRPCBase, RPCStep: defaultStep,
			},
			Reservation: res,
		}
		m, err := netmap.Assign(pool, netmapRoleRequests(roles))
		if err != nil {
			return "", err
		}
		assign = m
		p := m.Placements()
		return fmt.Sprintf("%d node(s); node1 p2p=%d etcd=%d http=%d",
			len(p), p[0].Ports.P2P, p[0].Ports.Etcd, p[0].Ports.HTTP), nil
	})

	t.do(c.Steps[4], func() (string, error) {
		a, err := engine.BuildGenesis(ctx, plugin, engine.GenesisRequest{Validators: o.Validators, Nodes: assign},
			engine.GenesisConfig{KeysDir: o.KeysDir, Binary: o.Binary})
		if err != nil {
			return "", err
		}
		art = a
		return fmt.Sprintf("%d bytes, %d member(s), %d by-product file(s)", len(a.Genesis), o.Validators, len(a.Extra)), nil
	})

	t.do(c.Steps[5], func() (string, error) {
		placed := make([]engine.PlacedNode, 0, o.Validators)
		for i, p := range assign.Placements() {
			placed = append(placed, engine.PlacedNode{Req: place.NodeReq{Role: roles[i]}, Placement: p})
		}
		p, err := engine.AssemblePlan(plugin, placed, art.Genesis, o.DataDir, plugin.Manifest().Capabilities)
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
		// The governance config is a by-product of the genesis step and an
		// input to deploy-governance. It goes to the target beside the
		// genesis, because a deploy that rebuilt it could disagree with the
		// genesis the network actually started from.
		for name, content := range art.Extra {
			if err := o.files().Write(ctx, filepath.Join(plan.DataRoot, name), content, 0o644); err != nil {
				return "", fmt.Errorf("write %s: %w", name, err)
			}
		}
		return fmt.Sprintf("genesis + %d config(s) + %d by-product(s) under %s", len(specs), len(art.Extra), plan.DataRoot), nil
	})

	procs := process.New()
	bootstrap := engine.WemixBootstrap{Binary: o.Binary, KeysDir: o.KeysDir}

	t.do(c.Steps[8], func() (string, error) {
		phases = plugin.Family().BringUpPhases(roles)
		if len(phases) == 0 {
			return "", fmt.Errorf("the wemix family declared no bring-up phases")
		}
		res, err := launch.Launch(ctx, plan, phases[0].Nodes)
		for _, p := range res.Procs {
			procs.TrackProc(p)
		}
		if err != nil {
			return "", err
		}
		nodes = res.Nodes
		if len(nodes.Nodes) == 0 {
			return "", fmt.Errorf("the boot phase started no node")
		}
		boot = nodes.Nodes[0]
		run.Nodes = append(run.Nodes, boot.RPCURL)
		return fmt.Sprintf("node%d alone at %s", boot.Index, boot.RPCURL), nil
	})

	// The bootstrap actions are the family's, named in the same order the step
	// list documents. Running them by name rather than by hand is what keeps
	// this procedure and the engine's from drifting apart.
	for i, name := range []string{c.Steps[9].ID, c.Steps[10].ID, c.Steps[11].ID} {
		step := c.Steps[9+i]
		t.do(step, func() (string, error) {
			if err := bootstrap.Action(ctx, name, plan, boot); err != nil {
				return "", err
			}
			return fmt.Sprintf("%s on node%d", name, boot.Index), nil
		})
	}

	t.do(c.Steps[12], func() (string, error) {
		if len(phases) < 2 {
			return "no other nodes to start", nil
		}
		res, err := launch.Launch(ctx, plan, phases[1].Nodes)
		for _, p := range res.Procs {
			procs.TrackProc(p)
		}
		if err != nil {
			return "", err
		}
		nodes.Nodes = append(nodes.Nodes, res.Nodes.Nodes...)
		for _, n := range res.Nodes.Nodes {
			run.Nodes = append(run.Nodes, n.RPCURL)
		}
		return fmt.Sprintf("%d more node(s) up", len(res.Nodes.Nodes)), nil
	})

	t.do(c.Steps[13], func() (string, error) {
		if len(phases) < 2 {
			return "a lone producer is a cluster of one", nil
		}
		if err := bootstrap.Action(ctx, c.Steps[13].ID, plan, boot); err != nil {
			return "", err
		}
		return fmt.Sprintf("%d producer(s) in the cluster", len(roles)), nil
	})

	t.do(c.Steps[14], func() (string, error) {
		head, err := waitHead(ctx, boot.RPCURL, 1, o.HealthTimeout)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("head %d on %s", head, boot.RPCURL), nil
	})

	run.Results = t.results
	if err := saveState(ctx, o.files(), o.DataDir, nodes); err != nil && !t.halted() {
		return run, err
	}
	return run, nil
}

// netmapRoleRequests turns a role list into placement requests: only the role
// travels, since position comes from the order.
func netmapRoleRequests(roles []node.Role) []netmap.Request {
	out := make([]netmap.Request, 0, len(roles))
	for _, r := range roles {
		out = append(out, netmap.Request{Role: r})
	}
	return out
}
