package chainsetup

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/process"

	"github.com/0xmhha/chainbench/internal/core/inspector"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/resource"
)

// The checks that run before a launch: is something already here, and can this
// network have the ports it planned.
//
// They exist because the failure they prevent is expensive and quiet. A node
// launched over a datadir another composition owns, or onto a port another
// process holds, fails after provisioning has already run — and the message
// names a port rather than the composition that took it.

func (w *Workspace) Preflight(ctx context.Context, binaryArg string) error {
	if err := w.allow("Preflight"); err != nil {
		return err
	}
	bin, err := w.binary(binaryArg)
	if err != nil {
		return err
	}
	if err := w.checkUnmanaged(ctx, bin); err != nil {
		return err
	}
	return w.checkVacant(ctx, registry.Phase{})
}

// checkUnmanaged asks the machine (through the driver's inspector) whether
// the binary about to be launched is already running OUTSIDE the run ledger.
// A pid the ledger knows is this workspace's and is handled per node; a pid
// it does not know belongs to someone — another workspace, an operator's
// hand-started node — and composing on top of it is refused by name.
func (w *Workspace) checkUnmanaged(ctx context.Context, bin string) error {
	name := filepath.Base(bin)
	return w.eachMachine(func(t *resource.Access, _ []node.Record) error {
		return w.checkUnmanagedOn(ctx, t, name)
	})
}

// checkUnmanagedOn is checkUnmanaged for one resource.
func (w *Workspace) checkUnmanagedOn(ctx context.Context, t *resource.Access, name string) error {
	insp, ok := t.Driver.(process.ProcessInspector)
	if !ok {
		return nil
	}
	pids, err := insp.FindBinary(ctx, name)
	if err != nil {
		return fmt.Errorf("chainsetup: process check: %w", err)
	}
	known := map[int]bool{}
	for _, p := range w.ledger.Recorded() {
		known[p.PID] = true
	}
	var strays []string
	for _, pid := range pids {
		if !known[pid] {
			strays = append(strays, strconv.Itoa(pid))
		}
	}
	if len(strays) > 0 {
		return ofKind(errLaunchOccupied,
			fmt.Errorf("chainsetup: %s is already running on the machine outside this workspace (pid %s) — stop it, or compose on a different server",
				name, strings.Join(strays, ", ")))
	}
	return nil
}

// checkVacant refuses to launch onto ports something is already listening on.
//
// Without it the collision is discovered by the node, which dies with "address
// already in use" partway through a bring-up, and the operator has to work out
// which of three situations they are in. This says which: a port held by a node
// this workspace recorded is its own leftover and `chain stop` clears it; anything
// else belongs to something this workspace did not start, and guessing would be
// worse than refusing.
func (w *Workspace) checkVacant(ctx context.Context, phase registry.Phase) error {
	var addrs []inspector.Addr
	for _, ns := range w.state.Nodes {
		if ns.PID > 0 || !phaseHasNode(phase, ns.Index) {
			continue
		}
		host := nodeHost(ns)
		for purpose, port := range map[string]int{
			"p2p": ns.P2P, "etcd": ns.Etcd, "etcd-client": ns.EtcdClient,
			"http": ns.HTTP, "ws": ns.WS, "auth": ns.Auth, "metrics": ns.Metrics,
		} {
			addrs = append(addrs, inspector.Addr{Host: host, Port: port, Node: ns.Index, Purpose: purpose})
		}
	}
	busy, err := w.scanPorts(ctx, addrs)
	if err != nil {
		return err
	}
	if len(busy) == 0 {
		return nil
	}
	mine := w.recordedLeftovers()
	lines := make([]string, 0, len(busy))
	var recoverable, byHand bool
	for _, b := range busy {
		who, ok := mine[portKey{host: b.Host, port: b.Port}]
		switch {
		case ok && who.pid > 0:
			recoverable = true
			lines = append(lines, fmt.Sprintf("  %s — this workspace's node%d (pid %d)", b, who.node, who.pid))
		case ok:
			// This workspace planned the address and never recorded a pid for
			// it, so whatever is listening is not something it started. Saying
			// "this workspace's node%d" claimed the opposite, and the operator
			// who believed it went looking in the wrong composition.
			byHand = true
			lines = append(lines, fmt.Sprintf("  %s — planned for this workspace's node%d, but it started nothing there: another composition holds it", b, who.node))
		default:
			byHand = true
			lines = append(lines, fmt.Sprintf("  %s — not started by this workspace", b))
		}
	}
	var hints []string
	if recoverable {
		hints = append(hints, "`chain stop --workspace-dir "+w.Dir()+"` stops the ones with a recorded pid")
	}
	if byHand {
		hints = append(hints, "the rest hold ports this workspace planned but cannot address — find and stop them by hand")
	}
	hint := strings.Join(hints, "; ")
	return ofKind(errLaunchPortBusy,
		fmt.Errorf("chainsetup: %d port(s) are already in use:\n%s\n%s", len(busy), strings.Join(lines, "\n"), hint))
}

// scanPorts asks whether the plan's ports are taken, from where the
// answer is true. A local target asks this machine's kernel (inspector.Scan's
// bind probe). A remote target is asked ON the target through the driver's
// PortProber: probing from here lies in both directions — a loopback-bound
// listener on the server is invisible from outside, and a docker-published
// port is "open" from here even when nothing inside the container listens,
// because the publish forwarder itself accepts the connection (measured: an
// idle set reported every node port busy).
func (w *Workspace) scanPorts(ctx context.Context, addrs []inspector.Addr) ([]inspector.Addr, error) {
	if !w.state.Target.IsRemote() {
		return inspector.Ports(ctx, addrs, nil), nil
	}
	byHost := map[string][]int{}
	for _, a := range addrs {
		if a.Port > 0 {
			byHost[a.Host] = append(byHost[a.Host], a.Port)
		}
	}
	// Each host is probed BY ITS OWN machine (the probe lies from anywhere
	// else); the node table says which machine owns which address.
	proberFor := func(host string) (process.PortProber, error) {
		for _, ns := range w.state.Nodes {
			if nodeHost(ns) != host {
				continue
			}
			t, err := w.machineFor(ns)
			if err != nil {
				return nil, err
			}
			p, ok := t.Driver.(process.PortProber)
			if !ok {
				return nil, nil
			}
			return p, nil
		}
		return nil, nil
	}
	var busy []inspector.Addr
	for host, ports := range byHost {
		prober, err := proberFor(host)
		if err != nil {
			return nil, err
		}
		if prober == nil {
			// A machine whose driver cannot probe reports nothing rather
			// than guessing from the wrong side; the launch finds a
			// collision the old way ("address already in use").
			continue
		}
		open, err := prober.ProbePorts(ctx, host, ports)
		if err != nil {
			return nil, fmt.Errorf("chainsetup: port probe on %s: %w", host, err)
		}
		taken := map[int]bool{}
		for _, p := range open {
			taken[p] = true
		}
		for _, a := range addrs {
			if a.Host == host && taken[a.Port] {
				busy = append(busy, a)
			}
		}
	}
	return busy, nil
}

// recordedLeftovers maps a port to what this workspace knows about the node
// that owns it, so a collision with our own earlier run reads as that rather
// than as a stranger.
//
// Every node in the table counts, not only the ones with a pid. A workspace
// that lost its pids — the interrupted run, the run whose state file was
// removed while its nodes kept running — still owns the layout, and telling the
// operator that their own ports belong to somebody else is the least useful
// thing this check could say.
func (w *Workspace) recordedLeftovers() map[portKey]owner {
	out := map[portKey]owner{}
	for _, ns := range w.state.Nodes {
		o := owner{node: ns.Index, pid: ns.PID}
		host := nodeHost(ns)
		for _, port := range []int{ns.P2P, ns.Etcd, ns.EtcdClient, ns.HTTP, ns.WS, ns.Auth, ns.Metrics} {
			if port > 0 {
				out[portKey{host: host, port: port}] = o
			}
		}
	}
	return out
}

// portKey addresses a planned port the way the scan reports a busy one: by host
// and port, never by port alone.
//
// A port number is not an identity here. Spread across a server set, every
// server runs its slot-1 node on the same numbers — 8601, 30301 — so a map
// keyed on the number alone answers "who planned 8601?" with whichever node was
// written last, and a busy port on one server gets reported as a node on
// another. Same collapse as the running-node map had (MON-010), one file over.
type portKey struct {
	host string
	port int
}

// owner is what this workspace knows about the node that planned a port: which
// node it is, and whether a pid was ever recorded for it. The difference
// decides the remedy — a recorded pid can be stopped, and a missing one means
// the run that started it never got to write it down.
type owner struct {
	node int
	pid  int
}

// startPhase launches one phase's nodes, or every stopped node when the phase
// names none. A node already running is left alone: `chain restart` bounces one,
// and re-running `chain start` should not double-launch the rest.
