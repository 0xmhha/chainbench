package app

import (
	"context"
	"fmt"
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
	KeysDir        string
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
	// AllServers spreads the handoff across every server in the set, drawing each
	// node's host and port band from the resource module instead of the profile's
	// port bases.
	//
	// It is what makes a handoff wider than one machine: the allocator hands out
	// slot 1 on every server, then slot 2 on every server at the next port band,
	// so 15 producers and 15 successors over 15 servers is one of each per server.
	AllServers bool
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
	//
	// The resolution itself lives with HandoffInputs now, so the other surface
	// that composes a handoff — a case declaring one in the DSL — gets the same
	// answer instead of a local-only one.
	hi := upgrade.HandoffInputs{
		ProfilePath:    in.ProfilePath,
		KeysDir:        in.KeysDir,
		Template:       in.Template,
		GenesisOverlay: in.GenesisOverlay,
		DataDir:        in.DataDir,
	}
	target := upgrade.Target{
		Server: in.Server, AllServers: in.AllServers,
		WorkspaceConfigPath: in.WorkspaceConfigPath, Docker: in.Docker,
		Env: d.Env, Report: d.Logf,
	}
	wc, err := target.Apply(&hi, prof.Roles.Producers+prof.Roles.Validators)
	if err != nil {
		return UpgradeRunOut{}, err
	}
	var acc *resource.Access
	if !target.Local() {
		acc = &resource.Access{Spec: resource.Spec{Server: hi.Host}, Files: hi.Files, Driver: hi.Driver}
	}
	if hi.FromBinary, err = resolveBinaryOn(ctx, acc, wc, in.FromBinary, prof.Chains.From.Binary); err != nil {
		return UpgradeRunOut{}, fmt.Errorf("from binary: %w", err)
	}
	if hi.ToBinary, err = resolveBinaryOn(ctx, acc, wc, in.ToBinary, prof.Chains.To.Binary); err != nil {
		return UpgradeRunOut{}, fmt.Errorf("to binary: %w", err)
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
func resolveBinaryOn(ctx context.Context, acc *resource.Access, wc *resource.WorkspaceConfig, explicit, chainBinary string) (string, error) {
	if acc == nil {
		return ResolveBinary(explicit, chainBinary)
	}
	name := explicit
	if name == "" {
		name = chainBinary
	}
	if !path.IsAbs(name) {
		// A bare name is a name on the TARGET, and the workspace-config is what
		// knows where names live there: dataRoot + paths.binaries, with
		// binaryAliases applied. Before this, a bare name was simply refused, so
		// every remote run had to spell out a path the config already described
		// and binaryAliases had no consumer at all.
		if wc == nil {
			return "", fmt.Errorf("binary %q must be an absolute path on server %q — without a workspace-config there is nothing that says where a bare name lives there",
				name, acc.Spec.Server)
		}
		resolved, err := wc.BinaryPath(name)
		if err != nil {
			return "", fmt.Errorf("resolving binary %q on server %q: %w", name, acc.Spec.Server, err)
		}
		name = resolved
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
