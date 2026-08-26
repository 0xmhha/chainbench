package chainsetup

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// HandoffOptions are the inputs for the gwemix -> gwbft case.
type HandoffOptions struct {
	// ProfilePath is the golden upgrade profile.
	ProfilePath string
	// PresetDir holds the key preset.
	PresetDir string
	// FromBinary produces blocks up to the fork; ToBinary takes over after it.
	FromBinary, ToBinary string
	// Template is go-wemix's OWN genesis template (not chainbench's substitution
	// template, which carries __CHAIN_ID__ placeholders the binary rejects).
	Template string
	// GenesisOverlay optionally deep-merges extra genesis fields.
	GenesisOverlay string
	// DataDir is the network's data root.
	DataDir string
	// ForkTimeout bounds the wait for the handoff.
	ForkTimeout time.Duration
	// EtcdTimeout bounds the wait for the etcd cluster to form.
	EtcdTimeout time.Duration
	// StopAfter ends the run once that step completes.
	StopAfter string
	// Exec runs a binary; nil uses os/exec.
	Exec Runner
	// Files is where this run's artifacts are written. Nil is the local
	// filesystem. The boundary exists so the package stops owning the question of
	// where state lands — the rule is that a step describes what to write and
	// the target decides where (layers §5).
	Files filestore.Store
}

func (o HandoffOptions) files() filestore.Store {
	if o.Files == nil {
		return filestore.Local{}
	}
	return o.Files
}

// HandoffDriver performs the parts of the handoff that need the upgrade
// orchestration. It is an interface so the step sequence — which is what this
// package contributes — can be exercised without chain binaries.
type HandoffDriver interface {
	// Prepare loads the profile and preset and reports what it read.
	Prepare(ctx context.Context, o HandoffOptions) (string, error)
	// Config assembles and writes the governance config, returning its path.
	Config(ctx context.Context, o HandoffOptions) (string, error)
	// BaseGenesis generates the producer's base genesis, returning its path.
	BaseGenesis(ctx context.Context, o HandoffOptions, configPath string) (string, error)
	// Plan composes the handoff plan (fork section lifted and merged).
	Plan(ctx context.Context, o HandoffOptions, baseGenesis string) (string, error)
	// Overlay applies the optional genesis overlay.
	Overlay(ctx context.Context, o HandoffOptions) (string, error)
	// Launch starts every node and returns the running set.
	Launch(ctx context.Context, o HandoffOptions) (node.NodeSet, error)
	// WireMesh connects every node to every other.
	WireMesh(ctx context.Context, ns node.NodeSet) (string, error)
	// DeployGovernance deploys the governance contracts on the producer.
	DeployGovernance(ctx context.Context, o HandoffOptions, producer node.Node) (string, error)
	// EtcdInit calls admin.etcdInit() on the producer.
	EtcdInit(ctx context.Context, o HandoffOptions, producer node.Node) (string, error)
	// ProducerIPC returns the producer's IPC socket path.
	ProducerIPC(o HandoffOptions, producer node.Node) string
	// ForkBlock is the height the handoff happens at.
	ForkBlock() int64
	// ProducerAccount is the from-chain miner, used to tell who sealed a block.
	ProducerAccount() string
}

// RunHandoff executes the gwemix -> gwbft case step by step.
//
// It adds one step the existing CLI does not have: verify-etcd. Calling
// admin.etcdInit() and reporting success because the command exited 0 is how a
// failed bootstrap came to be reported as "etcd initialized" — the cluster can
// stay empty and the producer stalls a few blocks later, far from the cause.
func RunHandoff(ctx context.Context, c Case, o HandoffOptions, d HandoffDriver, report Reporter) (Run, error) {
	if err := validateStopAfter(c, o.StopAfter); err != nil {
		return Run{Case: c}, err
	}
	if o.DataDir == "" {
		return Run{Case: c}, fmt.Errorf("chainsetup: data dir is required")
	}
	if o.PresetDir == "" {
		o.PresetDir = "keys/preset"
	}
	if o.ForkTimeout <= 0 {
		o.ForkTimeout = 180 * time.Second
	}
	if o.EtcdTimeout <= 0 {
		o.EtcdTimeout = 60 * time.Second
	}
	if o.Exec == nil {
		o.Exec = defaultExec
	}
	// No mkdir here: a store Write creates the parents, and the first thing
	// this run writes is the governance config under DataDir. The binary that
	// writes base-genesis into the same directory runs after that step.

	t := newTracker(report, o.StopAfter)
	run := Run{Case: c, DataDir: o.DataDir}

	var (
		configPath  string
		baseGenesis string
		ns          node.NodeSet
		producer    node.Node
	)

	t.do(c.Steps[0], func() (string, error) { return d.Prepare(ctx, o) })
	t.do(c.Steps[1], func() (string, error) {
		return fmt.Sprintf("preset %s", o.PresetDir), nil
	})
	t.do(c.Steps[2], func() (string, error) {
		p, err := d.Config(ctx, o)
		configPath = p
		if err != nil {
			return "", err
		}
		return p, nil
	})
	t.do(c.Steps[3], func() (string, error) {
		p, err := d.BaseGenesis(ctx, o, configPath)
		baseGenesis = p
		if err != nil {
			return "", err
		}
		return p, nil
	})
	t.do(c.Steps[4], func() (string, error) { return d.Plan(ctx, o, baseGenesis) })
	t.do(c.Steps[5], func() (string, error) { return d.Overlay(ctx, o) })

	t.do(c.Steps[6], func() (string, error) {
		set, err := d.Launch(ctx, o)
		ns = set
		for _, n := range ns.Nodes {
			run.Nodes = append(run.Nodes, n.RPCURL)
		}
		if err != nil {
			return "", err
		}
		if len(ns.Nodes) == 0 {
			return "", fmt.Errorf("launch produced no nodes")
		}
		producer = ns.Nodes[0]
		_ = saveState(ctx, o.files(), o.DataDir, ns)
		return fmt.Sprintf("%d node(s); producer %s", len(ns.Nodes), producer.RPCURL), nil
	})

	t.do(c.Steps[7], func() (string, error) { return d.WireMesh(ctx, ns) })
	t.do(c.Steps[8], func() (string, error) { return d.DeployGovernance(ctx, o, producer) })
	t.do(c.Steps[9], func() (string, error) { return d.EtcdInit(ctx, o, producer) })

	t.do(c.Steps[10], func() (string, error) {
		info, err := WaitEtcdCluster(ctx, o.Exec, o.FromBinary, d.ProducerIPC(o, producer), o.EtcdTimeout, 0)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("governance %s, etcd cluster %q, miners %q", info.Governance, info.Cluster(), info.Miners), nil
	})

	t.do(c.Steps[11], func() (string, error) {
		return awaitFork(ctx, ns, d.ForkBlock(), d.ProducerAccount(), o.ForkTimeout)
	})

	run.Results = t.results
	return run, nil
}

// awaitFork waits until a successor validator seals the first post-fork block.
// It polls a validator rather than the producer: the producer cannot import
// post-fork blocks, so its head is not the handoff's evidence.
func awaitFork(ctx context.Context, ns node.NodeSet, forkBlock int64, producerAcct string, timeout time.Duration) (string, error) {
	var target string
	for _, n := range ns.Nodes {
		if n.Index != 0 {
			target = n.RPCURL
			break
		}
	}
	if target == "" {
		return "", fmt.Errorf("no successor validator to observe the handoff on")
	}
	c := rpc.Dial(target)
	producer := strings.ToLower(producerAcct)
	deadline := time.Now().Add(timeout)
	var head uint64
	for time.Now().Before(deadline) {
		h, err := c.BlockNumber(ctx)
		if err == nil {
			head = h
			if h > uint64(forkBlock) {
				var blk struct {
					Miner string `json:"miner"`
				}
				if err := c.Call(ctx, "eth_getBlockByNumber", &blk, fmt.Sprintf("0x%x", forkBlock+1), false); err == nil {
					miner := strings.ToLower(blk.Miner)
					if miner != "" && miner != producer {
						return fmt.Sprintf("head %d; block %d sealed by %s (successor)", h, forkBlock+1, miner), nil
					}
				}
			}
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(healthPollInterval):
		}
	}
	return "", fmt.Errorf("head stalled at %d, never crossed fork block %d within %s", head, forkBlock, timeout)
}

// producerIPCPath is the conventional IPC socket of node1 under a data root.
func producerIPCPath(dataDir, fromBinary string, producer node.Node) string {
	return filepath.Join(dataDir, fmt.Sprintf("node%d", producer.Index+1), filepath.Base(fromBinary)+".ipc")
}
