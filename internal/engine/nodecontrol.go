package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/pipeline/setup"
	"github.com/0xmhha/chainbench/internal/core/procman"
	"github.com/0xmhha/chainbench/internal/core/supervisor"
	"github.com/0xmhha/chainbench/internal/testspec"
)

// stopGrace is how long a single-node stop waits for a graceful exit before
// escalating to SIGKILL.
const stopGrace = 5 * time.Second

// NodeController launches a plan and remembers each node's arming, so a spec can
// later stop and restart one node without disturbing the rest of the network.
// It sits in front of a LocalLauncher: it satisfies the supervisor's launch seam
// and testspec.NodeControl at once, which is what lets a fault step reach the
// exact process the bring-up started.
//
// Attach mode wires none of this — chainbench does not own those processes, and
// the fault actions say so rather than pretending.
type NodeController struct {
	launcher LocalLauncher
	procs    *procman.Manager

	mu    sync.Mutex
	specs map[int]driver.NodeSpec // node index -> arming from the last launch
	pids  map[int]int             // node index -> current pid (0 when stopped)
}

// NewNodeController returns a controller over launcher, tracking processes in
// procs (the same manager the supervisor tears down with, so a node stopped and
// restarted mid-test is still accounted for at teardown).
func NewNodeController(launcher LocalLauncher, procs *procman.Manager) *NodeController {
	if procs == nil {
		procs = procman.New()
	}
	return &NodeController{
		launcher: launcher,
		procs:    procs,
		specs:    map[int]driver.NodeSpec{},
		pids:     map[int]int{},
	}
}

// Launch implements the supervisor launch seam, recording each node's arming and
// pid on the way through.
func (c *NodeController) Launch(ctx context.Context, plan setup.Plan) (supervisor.LaunchResult, error) {
	res, specs, err := c.launcher.LaunchArmed(ctx, plan)
	c.record(res, specs)
	return res, err
}

// record stores the arming and pid of every node the launch produced.
func (c *NodeController) record(res supervisor.LaunchResult, specs []driver.NodeSpec) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, s := range specs {
		c.specs[s.Index] = s
	}
	for _, n := range res.Nodes.Nodes {
		c.pids[n.Index] = n.PID
	}
}

// Stop terminates one node's process and verifies it is gone.
func (c *NodeController) Stop(_ context.Context, n node.Node) error {
	pid := c.pidFor(n)
	if pid <= 1 {
		return fmt.Errorf("engine: node%d has no tracked process to stop", n.Index)
	}
	if err := c.procs.StopOne(pid, stopGrace); err != nil {
		return err
	}
	c.mu.Lock()
	c.pids[n.Index] = 0
	c.mu.Unlock()
	return nil
}

// Start relaunches a stopped node with the arming its original launch used, so
// the restarted node rejoins with the same identity, ports, and datadir. The
// datadir is not re-initialized: the point of a restart test is that the node
// recovers from its existing chain data.
func (c *NodeController) Start(ctx context.Context, n node.Node) error {
	c.mu.Lock()
	spec, ok := c.specs[n.Index]
	running := c.pids[n.Index]
	c.mu.Unlock()
	if !ok {
		return fmt.Errorf("engine: node%d was not launched by this run, so it cannot be started", n.Index)
	}
	if running > 1 && procman.Alive(running) {
		return fmt.Errorf("engine: node%d is already running (pid %d)", n.Index, running)
	}

	d := c.launcher.Driver
	if d == nil {
		d = driver.NewLocalDriver()
	}
	h, err := d.Launch(ctx, spec)
	if err != nil {
		return fmt.Errorf("engine: restart node%d: %w", n.Index, err)
	}
	c.procs.TrackProc(procman.Proc{
		PID: h.PID, Label: fmt.Sprintf("node%d", n.Index),
		DataDir: spec.DataDir, Host: spec.Host,
	})
	c.mu.Lock()
	c.pids[n.Index] = h.PID
	c.mu.Unlock()
	return nil
}

// pidFor returns the controller's recorded pid for a node, falling back to the
// pid carried on the node record (a node table read back from env.json).
func (c *NodeController) pidFor(n node.Node) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	if pid, ok := c.pids[n.Index]; ok && pid > 1 {
		return pid
	}
	return n.PID
}

// NodeController satisfies the DSL's node-control seam.
var _ testspec.NodeControl = (*NodeController)(nil)
