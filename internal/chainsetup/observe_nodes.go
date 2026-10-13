package chainsetup

import (
	"context"

	"github.com/0xmhha/chainbench/internal/core/process"
)

// NodeObservation is what a node's machine shows for one recorded node. The
// record says what was launched; this says what is there now.
type NodeObservation struct {
	Index       int    `json:"index"`
	Label       string `json:"label"`
	Role        string `json:"role"`
	Server      string `json:"server,omitempty"`
	RecordedPID int    `json:"recordedPid"`
	process.LaunchVerdict
	// Executable is whether the node's binary can be launched on its machine:
	// ready, unavailable, or unknown when the machine could not be reached.
	Executable string `json:"executable"`
}

// ObserveNodes reads every recorded node's machine and changes nothing. A node
// with a pid is held to its recorded launch; one without is searched for, so a
// process the record lost shows up rather than being taken for stopped.
func (w *Workspace) ObserveNodes(ctx context.Context) []NodeObservation {
	out := make([]NodeObservation, 0, len(w.state.Nodes))
	for _, ns := range w.state.Nodes {
		obs := NodeObservation{Index: ns.Index, Label: string(ns.NodeLabel()), Role: ns.Role, Server: ns.Server, RecordedPID: ns.PID,
			LaunchVerdict: process.LaunchVerdict{State: process.OwnedUnknown, Reason: "machine_unavailable"}, Executable: "unknown"}
		t, err := w.machineFor(ns)
		if err != nil {
			out = append(out, obs)
			continue
		}
		binary := w.launchBinary(ns)
		table := process.ProcessTableOf(ctx, t.Driver)
		if ns.PID > 0 {
			obs.LaunchVerdict = process.ObserveLaunch(ctx, table, ns.PID, binary, ns.Args)
		} else {
			obs.LaunchVerdict = process.ObserveVacancy(ctx, table, binary, ns.Args, ns.DataDir, ns.ConfigPath)
		}
		obs.Executable = "ready"
		if err := t.VerifyExecutable(ctx, binary); err != nil {
			obs.Executable = "unavailable"
		}
		out = append(out, obs)
	}
	return out
}
