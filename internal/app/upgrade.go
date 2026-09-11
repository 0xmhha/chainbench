package app

import (
	"context"
	"fmt"
	"github.com/0xmhha/chainbench/internal/consensus/poa"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/resource"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/0xmhha/chainbench/internal/consensus/upgrade"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// defaultEtcdFormTimeout bounds the wait for the producer's etcd cluster to
// form when the caller does not set one.
const defaultEtcdFormTimeout = 60 * time.Second

// UpgradeRunIn shapes a profile-based consensus handoff (the CLI `upgrade run`).
type UpgradeRunIn struct {
	ProfilePath    string
	PresetDir      string
	FromBinary     string
	ToBinary       string
	Template       string
	GenesisOverlay string
	DataDir        string
	// EtcdTimeout bounds the etcd-cluster-form wait; zero uses the default.
	EtcdTimeout time.Duration
	// AwaitFork, when above zero, polls that long for the successor to take
	// over after the fork block, so a caller can ask for the handoff to be
	// confirmed rather than merely started.
	AwaitFork time.Duration
	// Server names where the handoff's data plane lives, from the server set.
	// Empty runs it on this machine, which is what it has always done.
	//
	// The sequence itself never needed a branch for this: upgrade.HandoffInputs
	// has taken Host, Files, Driver and Exec since the family rework, so a
	// remote handoff was one resolution away. Nothing resolved it, so the CLI
	// had no target flags and every handoff was local by construction — the
	// last piece of G2.
	Server ServerRef
	// WorkspaceConfigPath owns the target's data root, and is required with
	// Server: a remote server cannot resolve without knowing where its data
	// lives.
	WorkspaceConfigPath string
	// Docker translates this tool's dials through the localmap beside the server
	// set, for a fleet of containers standing in for servers.
	Docker bool
}

// UpgradeRunOut is the handoff result.
type UpgradeRunOut struct {
	Nodes      node.NodeSet
	Governance string
	Cluster    string
	// Plan describes what was composed: the chains, the fork, the node count.
	Plan UpgradePlanOut
	// Confirmed is what the successor reported once it took over, when the
	// caller asked to wait for it.
	Confirmed string
}

// UpgradePlanOut is the shape of a composed handoff.
type UpgradePlanOut struct {
	From      string `json:"from"`
	To        string `json:"to"`
	AtFork    string `json:"atFork"`
	ForkBlock int64  `json:"forkBlock"`
	Nodes     int    `json:"nodes"`
}

// UpgradeRun performs a profile-based consensus handoff (go-wemix -> go-wbft at
// a fork), the same sequence the CLI `upgrade run` drives, so the MCP surface
// reaches it too. It wraps upgrade.NewHandoff and the shared Handoff.Run.
func UpgradeRun(ctx context.Context, d Deps, in UpgradeRunIn) (UpgradeRunOut, error) {
	// The binaries resolve here, not in the sequence: a profile names each by
	// its command name, and a caller may point at a build instead. Doing it
	// here means every surface accepts both spellings.
	prof, err := upgrade.LoadProfile(in.ProfilePath)
	if err != nil {
		return UpgradeRunOut{}, err
	}
	// The target resolves first, because it decides WHERE a binary path means
	// something. A remote handoff names paths on the server; looking those up in
	// the operator's own PATH is how the first remote run failed, reporting a
	// binary "not found" that was sitting on the target all along.
	var acc *resource.Access
	if in.Server.Name != "" {
		acc, err = openHandoffTarget(d, in)
		if err != nil {
			return UpgradeRunOut{}, err
		}
	}
	fromBin, err := resolveBinaryOn(ctx, acc, in.FromBinary, prof.Chains.From.Binary)
	if err != nil {
		return UpgradeRunOut{}, fmt.Errorf("from binary: %w", err)
	}
	toBin, err := resolveBinaryOn(ctx, acc, in.ToBinary, prof.Chains.To.Binary)
	if err != nil {
		return UpgradeRunOut{}, fmt.Errorf("to binary: %w", err)
	}
	hi := upgrade.HandoffInputs{
		ProfilePath:    in.ProfilePath,
		PresetDir:      in.PresetDir,
		FromBinary:     fromBin,
		ToBinary:       toBin,
		Template:       in.Template,
		GenesisOverlay: in.GenesisOverlay,
		DataDir:        in.DataDir,
	}
	// A named server moves the data plane. The profile, template, preset and
	// overlay stay local — they are the operator's inputs — while the genesis,
	// configs, keystores and the node processes go through the target's own
	// boundaries. That split is already what HandoffInputs documents; this is
	// the resolution that was missing.
	if acc != nil {
		hi.Host = acc.Spec.Host
		hi.DataDir = acc.DataRoot
		hi.Files = acc.Files
		hi.Driver = acc.Driver
		ex, err := handoffExec(acc)
		if err != nil {
			return UpgradeRunOut{}, err
		}
		hi.Exec = ex
	}
	h, err := upgrade.NewHandoff(hi)
	if err != nil {
		return UpgradeRunOut{}, err
	}
	timeout := in.EtcdTimeout
	if timeout <= 0 {
		timeout = defaultEtcdFormTimeout
	}
	ns, info, err := h.Run(ctx, timeout)
	if err != nil {
		return UpgradeRunOut{Nodes: ns}, err
	}
	out := UpgradeRunOut{
		Nodes: ns, Governance: info.Governance, Cluster: info.Cluster(),
		Plan: UpgradePlanOut{
			From: h.Plan.From.ID, To: h.Plan.To.ID, AtFork: h.Plan.AtFork,
			ForkBlock: h.ForkBlock(), Nodes: len(h.Plan.Nodes),
		},
	}
	if in.AwaitFork > 0 {
		detail, err := h.AwaitFork(ctx, ns, in.AwaitFork)
		if err != nil {
			return out, err
		}
		out.Confirmed = detail
	}
	return out, nil
}

// ResolveBinary returns the executable for a launch: the explicit path when
// given, otherwise the chain's own command name looked up on PATH.
//
// Every surface accepts both spellings, so both are read here. A surface that
// resolved this itself would be one that could accept a name the other
// rejects.
// resolveBinaryOn resolves a node binary for the machine that will run it.
//
// Locally that is a PATH lookup, which is what it has always been. On a target
// it is a probe of the target's own filesystem: the path names a file over
// there, so the check has to happen over there too. An absent binary is refused
// here rather than at launch, so the message names the file and the server
// instead of surfacing as a node that never came up.
func resolveBinaryOn(ctx context.Context, acc *resource.Access, explicit, chainBinary string) (string, error) {
	if acc == nil {
		return ResolveBinary(explicit, chainBinary)
	}
	name := explicit
	if name == "" {
		name = chainBinary
	}
	if !path.IsAbs(name) {
		return "", fmt.Errorf("binary %q must be an absolute path on server %q — a bare name would be looked up on this machine, not there",
			name, acc.Spec.Server)
	}
	ok, err := acc.Files.Exists(ctx, name)
	if err != nil {
		return "", fmt.Errorf("probing %s on server %q: %w", name, acc.Spec.Server, err)
	}
	if !ok {
		return "", fmt.Errorf("binary %s is not on server %q (upload it, e.g. `chainbench file upload --purpose bin`)", name, acc.Spec.Server)
	}
	return name, nil
}

// ResolveBinary finds a node binary on THIS machine.
func ResolveBinary(explicit, chainBinary string) (string, error) {
	name := explicit
	if name == "" {
		name = chainBinary
	}
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("cannot find node binary %q: %w (build it or pass an explicit path)", name, err)
	}
	return path, nil
}

// UpgradeGenesisOut is the merged handoff genesis and the plan that produced
// it: the from-chain's genesis carrying the successor's fork section.
type UpgradeGenesisOut struct {
	Genesis []byte
	Plan    UpgradePlanOut
	// Nodes describes each planned node, for a surface that lists them.
	Nodes []UpgradeNodeOut
}

// UpgradeNodeOut is one node of a planned handoff.
type UpgradeNodeOut struct {
	Index     int    `json:"index"`
	Chain     string `json:"chain"`
	Producer  bool   `json:"producer"`
	NetworkID int64  `json:"networkId"`
	P2P       int    `json:"p2p"`
	HTTP      int    `json:"http"`
	Etcd      int    `json:"etcd"`
}

// UpgradeGenesis builds the merged handoff genesis from a golden profile and
// the from-chain's base genesis.
//
// Reading the profile and resolving both chain plugins is one procedure, and it
// belongs here rather than in the surface that happens to expose it today: the
// merged genesis is what every node of the handoff boots from, so two readings
// of the same profile would be two networks.
func UpgradeGenesis(_ Deps, profilePath, fromGenesisPath string) (UpgradeGenesisOut, error) {
	p, err := upgrade.LoadProfile(profilePath)
	if err != nil {
		return UpgradeGenesisOut{}, err
	}
	base, err := os.ReadFile(fromGenesisPath)
	if err != nil {
		return UpgradeGenesisOut{}, fmt.Errorf("read from-genesis: %w", err)
	}
	in, err := p.Inputs(base)
	if err != nil {
		return UpgradeGenesisOut{}, err
	}
	from, err := Chain(Deps{}, p.Upgrade.From)
	if err != nil {
		return UpgradeGenesisOut{}, fmt.Errorf("from-chain %q: %w", p.Upgrade.From, err)
	}
	to, err := Chain(Deps{}, p.Upgrade.To)
	if err != nil {
		return UpgradeGenesisOut{}, fmt.Errorf("to-chain %q: %w", p.Upgrade.To, err)
	}
	plan, err := upgrade.BuildPlan(from, to, in)
	if err != nil {
		return UpgradeGenesisOut{}, err
	}
	out := UpgradeGenesisOut{
		Genesis: plan.Genesis,
		Plan: UpgradePlanOut{
			From: plan.From.ID, To: plan.To.ID, AtFork: plan.AtFork, Nodes: len(plan.Nodes),
		},
	}
	for _, n := range plan.Nodes {
		out.Nodes = append(out.Nodes, UpgradeNodeOut{
			Index: n.Index, Chain: n.Chain, Producer: n.Producer, NetworkID: n.NetworkID,
			P2P: n.Ports.P2P, HTTP: n.Ports.HTTP, Etcd: n.Ports.Etcd,
		})
	}
	return out, nil
}

// openHandoffTarget resolves where a remote handoff's data plane lives.
//
// It mirrors openTransferTarget: the workspace-config owns the data root, and a
// remote server cannot resolve without it, so asking for a server without one is
// an error naming the missing file rather than a resolution failure further down.
func openHandoffTarget(d Deps, in UpgradeRunIn) (*resource.Access, error) {
	if in.WorkspaceConfigPath == "" {
		return nil, fmt.Errorf("upgrade run: --workspace-config is required with --server (it owns the target data root)")
	}
	wc, err := resource.LoadWorkspaceConfig(in.WorkspaceConfigPath)
	if err != nil {
		return nil, err
	}
	opener := resource.Opener{ServerSet: in.Server.SetPath, Docker: in.Docker, Env: d.Env, Report: d.Logf}
	return opener.Open(resource.Spec{Server: in.Server.Name, DataRoot: wc.DataRoot})
}

// handoffExec is the bootstrap runner for a remote target: the poa helpers take
// a binary and arguments, and a remote target takes one command line, so the
// target's own Commander does the quoting through process.ShellRunner — the same
// adapter the composition path uses for the poa phase actions.
func handoffExec(acc *resource.Access) (poa.Runner, error) {
	cmdr, ok := acc.Driver.(process.Commander)
	if !ok {
		return nil, fmt.Errorf("upgrade run: target %q cannot run commands, so governance and etcd cannot be bootstrapped on it", acc.Spec.Server)
	}
	return poa.Runner(process.ShellRunner(cmdr)), nil
}
