package process

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
)

// stopGrace is how long a single-node stop waits for a graceful exit before
// escalating to SIGKILL.
const stopGrace = 5 * time.Second

// Controller launches a plan and remembers each node's arming, so a spec can
// later stop and restart one node without disturbing the rest of the network.
// It sits in front of a Direct: it satisfies the Launcher's launch boundary
// and the DSL's node-control boundary at once, which is what lets a fault
// step reach the exact process the bring-up started.
//
// It remembers arming, not pids. The pid lives in the environment's node
// table — the node handed to Stop and Start carries it, and the node handed
// back carries the new one. A private pid map beside the table was the second
// record P2 set out to remove.
//
// Attach mode wires none of this — chainbench does not own those processes, and
// the fault actions say so rather than pretending.
type Controller struct {
	launcher Direct
	procs    *Manager

	mu    sync.Mutex
	specs map[int]NodeSpec // node index -> arming from the last launch
}

// NewController returns a controller over direct, tracking processes in
// procs (the same manager the launcher tears down with, so a node stopped and
// restarted mid-test is still accounted for at teardown).
func NewController(direct Direct, procs *Manager) *Controller {
	if procs == nil {
		procs = New()
	}
	return &Controller{
		launcher: direct,
		procs:    procs,
		specs:    map[int]NodeSpec{},
	}
}

// Launch implements the Launcher's launch boundary, recording each node's
// arming on the way through.
func (c *Controller) Launch(ctx context.Context, plan Plan, nodes []int) (Result, error) {
	res, specs, err := c.launcher.LaunchArmed(ctx, plan, nodes)
	c.record(specs)
	return res, err
}

// record stores the arming of every node the launch produced.
func (c *Controller) record(specs []NodeSpec) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, s := range specs {
		c.specs[s.Index] = s
	}
}

// Stop terminates one node's process, verifies it is gone, and returns the
// node with its pid cleared.
func (c *Controller) Stop(_ context.Context, n node.Node) (node.Node, error) {
	if n.PID <= 1 {
		return n, fmt.Errorf("launcher: node%d has no tracked process to stop", n.Index)
	}
	if err := c.procs.StopOne(n.PID, stopGrace); err != nil {
		return n, err
	}
	n.PID = 0
	return n, nil
}

// Start relaunches a stopped node with the arming its original launch used, so
// the restarted node rejoins with the same identity, ports, and datadir. The
// datadir is not re-initialized: the point of a restart test is that the node
// recovers from its existing chain data.
func (c *Controller) Start(ctx context.Context, n node.Node) (node.Node, error) {
	c.mu.Lock()
	spec, ok := c.specs[n.Index]
	c.mu.Unlock()
	if !ok {
		return n, fmt.Errorf("launcher: node%d was not launched by this run, so it cannot be started", n.Index)
	}
	if n.PID > 1 && Alive(n.PID) {
		return n, fmt.Errorf("launcher: node%d is already running (pid %d)", n.Index, n.PID)
	}

	d := c.launcher.Driver
	if d == nil {
		d = NewLocalDriver()
	}
	h, err := d.Launch(ctx, spec)
	if err != nil {
		return n, fmt.Errorf("launcher: restart node%d: %w", n.Index, err)
	}
	c.procs.TrackProc(Proc{
		PID: h.PID, Label: string(node.LabelFor(n.Index)),
		DataDir: spec.DataDir, Host: spec.Host,
	})
	return NodeOf(spec, h.PID), nil
}
