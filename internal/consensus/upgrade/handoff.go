package upgrade

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/consensus/poa"
	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/genesis"
	"github.com/0xmhha/chainbench/internal/core/inspector"
	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/keyring/store"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/rpc"
)

// Timing of a live handoff.
const (
	// ipcWait is how long the producer's IPC socket has to appear before a
	// bootstrap step is attempted over it.
	ipcWait = 30 * time.Second
	// producingWait is how long the producer has to start sealing before the
	// governance deploy sends a transaction into it.
	producingWait = 60 * time.Second
	// selfWait is how long the producer has to recognise itself in the
	// governance member list before etcd is initialized.
	selfWait = 60 * time.Second
	// readyWait bounds the wait for every node's RPC before the mesh is wired.
	readyWait = 30 * time.Second
	// forkPoll is how often the successor's head is read while waiting for the
	// fork.
	forkPoll = time.Second
	// postForkBlocks is the FLOOR on how far past the fork the handoff is
	// observed. One block proves the boundary was crossed; several prove the
	// successor kept producing. The window itself scales with the successor set
	// (see postForkWindow), because a fixed ten blocks can show at most ten
	// validators however many there are — a fifteen-validator run reported
	// "10 of 15" and could not have reported more.
	postForkBlocks = 10
)

// handoffBalance funds the producer and every validator in the governance
// alloc: 10^27 wei, enough that staking and gas never bound a test.
const handoffBalance = "1000000000000000000000000000"

// HandoffInputs is what a live gwemix -> gwbft handoff needs from outside:
// the golden profile, the key preset, the two binaries, go-wemix's own genesis
// template, and where the network's files go. The seams (Exec, Files, Driver,
// Peers) are how a caller runs the same sequence against a fake or a remote
// target; nil takes the local default.
type HandoffInputs struct {
	// ProfilePath is the golden upgrade profile (profiles/*.yaml).
	ProfilePath string
	// PresetDir holds the key preset the nodes' identities come from.
	PresetDir string
	// FromBinary produces blocks up to the fork; ToBinary takes over after it.
	FromBinary, ToBinary string
	// Template is go-wemix's OWN genesis template (not chainbench's
	// substitution template, which carries placeholders the binary rejects).
	Template string
	// GenesisOverlay optionally deep-merges extra genesis fields, in the same
	// {"genesis":{...}} file shape `setup --genesis-overlay` takes.
	GenesisOverlay string
	// DataDir is the network's data root.
	DataDir string
	// Host is the address nodes bind and advertise; empty is this resource.
	Host string
	// Exec runs a binary; nil uses os/exec.
	Exec poa.Runner
	// Files is where this run's artifacts are written; nil is the local
	// filesystem. The preset is always read from this machine — it is the
	// operator's — while what the nodes need lands through Files, which is what
	// lets the same sequence place a keystore on a remote node.
	Files filestore.Store
	// Driver launches the nodes; nil is the local process.
	Driver process.Driver
	// Peers wires the mesh; nil uses JSON-RPC admin_addPeer.
	Peers PeerCaller
	// MultiMachine says the placement spans more than one server, which needs
	// Machine to resolve a file store and a driver per node. Set without it, the
	// run is refused rather than silently collapsed onto one machine — the shape
	// measured before Machine existed, where five nodes placed across five servers
	// all landed on the first.
	MultiMachine bool
	// DialURL turns a node's own address into the one this tool reaches it at.
	DialURL func(host string, port int) (string, error)
	// Machine resolves which machine node i runs on — its file store and its
	// driver. Nil uses Files and Driver for every node, which is one machine.
	Machine func(index int) (filestore.Store, process.Driver, error)
	// Placement, when set, is where the resource module put this handoff's nodes:
	// which server each one runs on and which port band it got. It wins over the
	// profile's port bases, because a server set steps ports by SLOT rather than
	// by node index — 15 producers and 15 successors over 15 servers is two nodes
	// per server, and no base-plus-index arithmetic produces that.
	Placement *node.Map
}

func (in HandoffInputs) host() string {
	if in.Host == "" {
		return "127.0.0.1"
	}
	return in.Host
}

func (in HandoffInputs) exec() poa.Runner {
	if in.Exec == nil {
		return poa.ExecRunner
	}
	return in.Exec
}

func (in HandoffInputs) files() filestore.Store {
	if in.Files == nil {
		return filestore.Local{}
	}
	return in.Files
}

func (in HandoffInputs) driver() process.Driver {
	if in.Driver == nil {
		return process.NewLocalDriver()
	}
	return in.Driver
}

// Handoff is one gwemix -> gwbft handoff being brought up: the producer and
// the successors run different binaries concurrently from genesis and the
// chain forks to the successor at a block. It holds what each step hands to
// the next, so a caller can run the steps in order and report each — the
// `chain up` case runner and `upgrade run` both do exactly that, and this is
// the one body they share.
//
// The order is: WriteConfig, BaseGenesis, ComposePlan, ApplyOverlay, Launch,
// WireMesh, DeployGovernance, EtcdInit, VerifyEtcd, AwaitFork. Each step
// assumes the ones before it ran.
type Handoff struct {
	in HandoffInputs
	// producerKeystore is the path this run shipped the producer's keystore to.
	// It is recorded rather than re-found, because the destination may be on
	// another machine and the file interface cannot list a directory.
	producerKeystore string
	// Profile is the loaded golden profile.
	Profile Profile
	// Preset is the loaded key preset.
	Preset keyring.Preset
	// From and To are the two chains.
	From, To registry.ChainPlugin
	// Plan is the composed handoff plan, set by ComposePlan.
	Plan Plan

	order      []int
	configPath string
	pwPath     string
}

// NewHandoff loads the profile, the preset and the two chain plugins, and
// checks the inputs a live run cannot do without. It reads; it writes nothing.
func NewHandoff(in HandoffInputs) (*Handoff, error) {
	if in.ProfilePath == "" || in.Template == "" {
		return nil, fmt.Errorf("upgrade: a profile and a go-wemix genesis template are required")
	}
	if in.FromBinary == "" || in.ToBinary == "" {
		return nil, fmt.Errorf("upgrade: both the from and the to binary are required")
	}
	if in.DataDir == "" {
		return nil, fmt.Errorf("upgrade: a data dir is required")
	}
	if in.PresetDir == "" {
		in.PresetDir = "keys/preset"
	}
	prof, err := LoadProfile(in.ProfilePath)
	if err != nil {
		return nil, err
	}
	// With keys: the handoff writes each node's nodekey into its datadir, which
	// is the one thing in this flow that needs the secret rather than the
	// identity.
	preset, err := store.LoadPresetWithKeys(in.PresetDir)
	if err != nil {
		return nil, err
	}
	from, err := registry.Get(prof.Upgrade.From)
	if err != nil {
		return nil, err
	}
	to, err := registry.Get(prof.Upgrade.To)
	if err != nil {
		return nil, err
	}
	h := &Handoff{in: in, Profile: prof, Preset: preset, From: from, To: to, order: prof.PlanOrderOrDefault()}
	if len(h.order) == 0 {
		return nil, fmt.Errorf("upgrade: the profile plans no nodes")
	}
	if _, ok := preset.Node(h.order[0]); !ok {
		return nil, fmt.Errorf("upgrade: preset has no node %d (the producer)", h.order[0])
	}
	if len(prof.Producers.Members) == 0 {
		return nil, fmt.Errorf("upgrade: the profile names no producer member")
	}
	return h, nil
}

// Describe says what the handoff is, for a step report.
func (h *Handoff) Describe() string {
	p := h.Profile
	return fmt.Sprintf("%s -> %s at %s block %d; %d producer(s) + %d validator(s)",
		p.Upgrade.From, p.Upgrade.To, p.Upgrade.AtFork, p.Upgrade.ForkBlock, p.Roles.Producers, p.Roles.Validators)
}

// ForkBlock is the height the handoff happens at.
func (h *Handoff) ForkBlock() int64 { return h.Profile.Upgrade.ForkBlock }

// ProducerAccount is the from-chain miner: the account that must NOT have
// sealed the first post-fork block.
func (h *Handoff) ProducerAccount() string { return h.Profile.Producers.Members[0] }

// WriteConfig assembles the wemix governance config from the profile and
// writes it under the data dir, returning its path. The deploy step reads the
// same file back, so the genesis and the deploy cannot disagree.
func (h *Handoff) WriteConfig(ctx context.Context) (string, error) {
	// The boot producer's identity, not the first producer's. The genesis names
	// one node the sole initial etcd member and the poa family brings THAT node up
	// alone; with one producer the two are the same node and the difference is
	// invisible, but with fifteen the member was plan node 1 while the node coming
	// up alone was plan node 15 — so the node that had to seal was not a member,
	// fell back to ethash, and produced nothing.
	boot := h.bootIndex()
	prod, ok := h.Preset.Node(h.order[boot])
	if !ok {
		return "", fmt.Errorf("upgrade: preset has no node %d (the boot producer)", h.order[boot])
	}
	cfg := h.poaConfig(prod)
	b, err := cfg.JSON()
	if err != nil {
		return "", err
	}
	path := filepath.Join(h.in.DataDir, h.From.Manifest().ID+"-config.json")
	// Write creates the parents, so the data dir needs no separate mkdir.
	if err := h.bootFiles().Write(ctx, path, b, 0o644); err != nil {
		return "", err
	}
	h.configPath = path
	return path, nil
}

// BaseGenesis has the producer's binary generate its base genesis from the
// governance config and the template, returning the file's path.
func (h *Handoff) BaseGenesis(ctx context.Context) (string, error) {
	if h.configPath == "" {
		return "", fmt.Errorf("upgrade: base genesis needs the governance config written first")
	}
	// The template is the OPERATOR's file, and the binary that reads it runs on
	// the target — so it has to be shipped there first, exactly as the governance
	// config above is. Passing the local path straight through worked only
	// because a local run shares one filesystem; on a server the binary opened a
	// path that does not exist there and reported it as a missing genesis
	// template.
	tmpl, err := h.shipTemplate(ctx)
	if err != nil {
		return "", err
	}
	path := filepath.Join(h.in.DataDir, "base-genesis.json")
	if err := poa.GenerateGenesis(ctx, h.bootExec(), h.in.FromBinary, h.configPath, tmpl, path); err != nil {
		return "", fmt.Errorf("%w (is --template go-wemix's own wemix/scripts/genesis-template.json?)", err)
	}
	return path, nil
}

// shipTemplate places the operator's genesis template where the producer's
// binary can read it, and returns that path.
func (h *Handoff) shipTemplate(ctx context.Context) (string, error) {
	b, err := os.ReadFile(h.in.Template)
	if err != nil {
		return "", fmt.Errorf("upgrade: genesis template: %w", err)
	}
	dst := filepath.Join(h.in.DataDir, "genesis-template.json")
	if err := h.bootFiles().Write(ctx, dst, b, 0o644); err != nil {
		return "", fmt.Errorf("upgrade: shipping the genesis template: %w", err)
	}
	return dst, nil
}

// ComposePlan lifts the successor's fork section onto the base genesis and
// builds the plan, with every node's devp2p pubkey so the mesh can be wired.
func (h *Handoff) ComposePlan(ctx context.Context, basePath string) error {
	base, err := h.forkPrereqs(ctx, basePath, h.Profile.Upgrade.NetworkID)
	if err != nil {
		return err
	}
	in, err := h.Profile.Inputs(base)
	if err != nil {
		return err
	}
	// A placement without a per-node machine would be decorative: the plan would
	// say node2 is on server2 while Files and Driver still point at one machine,
	// so every node would launch on that one and the run would look like it
	// worked. Refuse instead, naming what is missing.
	//
	// Wiring it is the remaining half of a multi-server handoff: LaunchOptions
	// carries one Files and Launch takes one Driver, where the composition path
	// resolves a machine per node (chainsetup.machineFor). The readiness and mesh
	// dials need the same treatment — they reach a node at its own address, which
	// is not the address this tool can dial under --docker.
	if h.in.MultiMachine && h.in.Machine == nil {
		return fmt.Errorf("upgrade: a placement across servers needs a machine per node, which this path does not resolve yet — " +
			"every node would launch on one server while the plan claimed otherwise. Run without --all-servers, or place the handoff on a single server with --server")
	}
	in.Placement = h.in.Placement
	in.DialURL = h.in.DialURL
	in.NodePubkeys = make([]string, len(h.order))
	for i, num := range h.order {
		nk, ok := h.Preset.Node(num)
		if !ok {
			return fmt.Errorf("upgrade: preset has no node %d", num)
		}
		in.NodePubkeys[i] = nk.PublicKey
	}
	plan, err := BuildPlan(h.From, h.To, in)
	if err != nil {
		return err
	}
	h.Plan = plan
	return nil
}

// ApplyOverlay deep-merges the optional genesis overlay into the plan's
// genesis. It reports what it did: nothing when no overlay was given or the
// file carries no genesis fragment.
func (h *Handoff) ApplyOverlay() (string, error) {
	if h.in.GenesisOverlay == "" {
		return "none", nil
	}
	raw, err := os.ReadFile(h.in.GenesisOverlay)
	if err != nil {
		return "", err
	}
	var ov struct {
		Genesis json.RawMessage `json:"genesis"`
	}
	if err := json.Unmarshal(raw, &ov); err != nil {
		return "", fmt.Errorf("upgrade: bad genesis overlay %q: %w", h.in.GenesisOverlay, err)
	}
	if len(ov.Genesis) == 0 {
		return "overlay has no genesis fragment", nil
	}
	merged, err := genesis.MergeOverride(h.Plan.Genesis, ov.Genesis)
	if err != nil {
		return "", fmt.Errorf("upgrade: apply genesis overlay: %w", err)
	}
	h.Plan.Genesis = merged
	return fmt.Sprintf("merged %s", h.in.GenesisOverlay), nil
}

// Launch writes the shared password file and starts every node — producers on
// the from binary, validators on the to binary, concurrently — returning the
// running set. Node identities are placed per node as each datadir comes up.
func (h *Handoff) Launch(ctx context.Context) (node.NodeSet, error) { return h.launch(ctx, nil) }

// LaunchPhase starts only the nodes at these 0-based positions, so a caller can
// bring the producer up alone and the rest after the bootstrap.
func (h *Handoff) LaunchPhase(ctx context.Context, only []int) (node.NodeSet, error) {
	return h.launch(ctx, only)
}

func (h *Handoff) launch(ctx context.Context, only []int) (node.NodeSet, error) {
	if len(h.Plan.Nodes) == 0 {
		return node.NodeSet{}, fmt.Errorf("upgrade: launch needs a composed plan")
	}
	if err := h.checkVacant(ctx, only); err != nil {
		return node.NodeSet{}, err
	}
	// The password file goes to every machine that will hold a node, not once.
	//
	// It is read by the binary at unlock time, so it has to be on that node's own
	// machine. One write was right while the network shared a filesystem; across a
	// server set the fifteenth producer died on "Failed to read password file"
	// while the first fourteen were fine. Machines are told apart by host — a
	// store is an interface value that need not be comparable.
	h.pwPath = filepath.Join(h.in.DataDir, "password")
	wrote := map[string]bool{}
	for i, n := range h.Plan.Nodes {
		if !selected(only, i) {
			continue
		}
		if wrote[n.Host] {
			continue
		}
		files, _, err := h.machineFiles(i)
		if err != nil {
			return node.NodeSet{}, err
		}
		if err := files.Write(ctx, h.pwPath, []byte(h.Preset.Password), 0o600); err != nil {
			return node.NodeSet{}, fmt.Errorf("upgrade: write password for node%d: %w", i+1, err)
		}
		wrote[n.Host] = true
	}
	opts := LaunchOptions{
		DataRoot:   h.in.DataDir,
		FromBinary: h.in.FromBinary, ToBinary: h.in.ToBinary,
		FromFamily: h.From.Family(), ToFamily: h.To.Family(),
		Host:          h.in.host(),
		ProvisionKeys: h.provisionKeys(),
		Overrides:     h.overrides(),
		Files:         h.in.Files,
		Machine:       h.in.Machine,
		Only:          only,
	}
	return Launch(ctx, h.in.driver(), h.Plan, opts)
}

// machineFiles is node i's file store: its own machine's when one is resolved,
// the single store otherwise.
func (h *Handoff) machineFiles(i int) (filestore.Store, process.Driver, error) {
	if h.in.Machine == nil {
		return h.in.files(), h.in.driver(), nil
	}
	return h.in.Machine(i)
}

// bootIndex is the 0-based plan position of the node the bootstrap runs on.
//
// The poa family's boot node is the LAST producer (consensus/poa/poa.go: it comes
// up alone and forms the etcd cluster of one, and genesis names that same node the
// sole initial member). Producers are always plan nodes 1..P, so the last of them
// is P-1 — derivable from the profile before a plan exists, which the config and
// genesis steps need since they run first.
func (h *Handoff) bootIndex() int {
	p := h.Profile.Roles.Producers
	if p < 1 {
		return 0
	}
	return p - 1
}

// bootFiles is the file store of the machine the bootstrap runs on. The
// governance config, the genesis template and the base genesis are all read by a
// binary on that machine, so they have to land there — not on whichever machine
// the run was started from.
func (h *Handoff) bootFiles() filestore.Store {
	// No resolver means one machine, and Files is it. Asking the resolver anyway
	// would be harmless here but not for bootExec, so both take the same shape.
	if h.in.Machine == nil {
		return h.in.files()
	}
	files, _, err := h.machineFiles(h.bootIndex())
	if err != nil || files == nil {
		return h.in.files()
	}
	return files
}

// bootExec runs a command on the machine the bootstrap runs on.
//
// The single Exec points at the machine the run was started from, which is the
// boot node's machine only by coincidence. With the boot producer on another
// server, `admin.etcdInit()` and the governance deploy were issued where the node
// is not — and the IPC wait looked for its socket on the wrong disk, reporting a
// node that was running as one that never came up.
func (h *Handoff) bootExec() poa.Runner {
	// Without a resolver there is one machine and Exec is how to reach it — which
	// includes a caller that injected its own runner. Deriving one from the driver
	// regardless would step over that: the local driver satisfies Commander, so an
	// injected Exec was replaced by a shell runner that ran the real binary.
	if h.in.Machine == nil {
		return h.in.exec()
	}
	_, driver, err := h.machineFiles(h.bootIndex())
	if err != nil || driver == nil {
		return h.in.exec()
	}
	cmdr, ok := driver.(process.Commander)
	if !ok {
		// A local driver runs commands as this process does; poa.ExecRunner is
		// already that, and h.in.exec() resolves to it.
		return h.in.exec()
	}
	return poa.Runner(process.ShellRunner(cmdr))
}

// checkVacant refuses to launch onto ports something else is already holding.
//
// Without it the collision surfaces as silence: the node exits on "address
// already in use" and the handoff waits out its timeout on a socket that will
// never appear, reporting "the node's IPC socket never appeared within 30s" and
// naming neither the port nor the process. Measured here — a node left over
// from an earlier session held the producer's p2p port, and three runs in a row
// died that way before anyone looked at the port table.
//
// The composition path has checked this since A3; the handoff is the surface
// that was never wired to it.
func (h *Handoff) checkVacant(ctx context.Context, only []int) error {
	var addrs []inspector.Addr
	for i, n := range h.Plan.Nodes {
		if !selected(only, i) {
			continue
		}
		for purpose, port := range map[string]int{
			"p2p": n.Ports.P2P, "etcd": n.Ports.Etcd, "etcd-client": n.Ports.EtcdClient,
			"http": n.Ports.HTTP, "ws": n.Ports.WS, "auth": n.Ports.Auth, "metrics": n.Ports.Metrics,
		} {
			if port > 0 {
				addrs = append(addrs, inspector.Addr{Host: h.in.host(), Port: port, Node: n.Index + 1, Purpose: purpose})
			}
		}
	}
	busy := inspector.Ports(ctx, addrs, nil)
	if len(busy) == 0 {
		return nil
	}
	lines := make([]string, len(busy))
	for i, b := range busy {
		lines[i] = "  " + b.String()
	}
	sort.Strings(lines)
	return fmt.Errorf("upgrade: these ports are already in use, so the handoff would launch onto them and the node would exit silently:\n%s\nstop whatever holds them, or run the handoff on a different port base",
		strings.Join(lines, "\n"))
}

// WireMesh waits for every node's RPC and connects each to every other, so
// the successor validators can reach a quorum among themselves.
func (h *Handoff) WireMesh(ctx context.Context, ns node.NodeSet) error {
	endpoints := make([]string, len(ns.Nodes))
	for i, n := range ns.Nodes {
		endpoints[i] = n.RPCURL
	}
	if err := WaitEndpointsReady(ctx, endpoints, readyWait); err != nil {
		return fmt.Errorf("upgrade: nodes not ready for mesh: %w", err)
	}
	peers := h.in.Peers
	if peers == nil {
		peers = DefaultPeerCaller()
	}
	return WireMesh(ctx, peers, endpoints, h.Plan.Enodes(h.in.host()))
}

// DeployGovernance deploys the governance contracts on the producer over its
// IPC, signing with the producer's keystore.
func (h *Handoff) DeployGovernance(ctx context.Context, producer node.Node) error {
	ipc := h.ProducerIPC(producer)
	if err := poa.WaitForIPCOn(ctx, h.bootFiles(), ipc, ipcWait); err != nil {
		return err
	}
	// The deploy is a transaction and waits for its receipt, so the chain has
	// to be sealing before it runs. The IPC socket appears about a second into
	// start-up and says nothing about that — waiting on it alone left this
	// deploying into a chain that had not produced a block, and the etcd
	// cluster that follows then formed nothing.
	//
	// The composition path's executor does this too. That it has to be written
	// twice is the real defect: this bootstrap repeats poa.Bootstrap.Action's
	// sequence by hand because the two carry different plan types, so every
	// lesson learned there has to be carried here as well. The functions are at
	// least the family's own rather than copies of them.
	if err := poa.WaitProducing(ctx, h.bootExec(), h.in.FromBinary, ipc, producingWait); err != nil {
		return err
	}
	// The keystore was shipped by this run, so its path is a fact this run holds.
	// Re-finding it by listing the directory only worked when the directory was
	// on this machine; the file interface cannot list a remote one.
	ksFile := h.producerKeystore
	if ksFile == "" {
		ksDir := node.Layout{Root: h.in.DataDir}.KeystoreDir(h.label(producer))
		var err error
		if ksFile, err = firstEntry(ksDir); err != nil {
			return fmt.Errorf("upgrade: producer keystore: %w", err)
		}
	}
	return poa.DeployGovernance(ctx, h.bootExec(), h.in.FromBinary, ipc, h.configPath, ksFile, h.pwPath)
}

// EtcdInit calls admin.etcdInit() on the producer, once the producer knows which
// governance member it is — before that the call is refused. A refusal is now an
// error rather than a silent success; VerifyEtcd still says whether the cluster
// actually formed.
func (h *Handoff) EtcdInit(ctx context.Context, producer node.Node) error {
	ipc := h.ProducerIPC(producer)
	// The node has to have read the governance contract before it can be told
	// to form a cluster: until it recognises itself among the members,
	// admin.etcdInit() refuses. Measured here — one run in four deployed
	// governance, initialized nothing, and stalled with an empty cluster.
	if err := poa.WaitSelf(ctx, h.bootExec(), h.in.FromBinary, ipc, selfWait); err != nil {
		return err
	}
	return poa.EtcdInit(ctx, h.bootExec(), h.in.FromBinary, ipc)
}

// VerifyEtcd polls the producer until its etcd cluster is non-empty, or the
// window passes. This is the step whose absence let a failed bootstrap report
// success: admin.etcdInit() exits 0 whether or not a cluster came up.
func (h *Handoff) VerifyEtcd(ctx context.Context, producer node.Node, timeout time.Duration) (poa.Info, error) {
	return poa.WaitEtcdCluster(ctx, h.bootExec(), h.in.FromBinary, h.ProducerIPC(producer), timeout, 0)
}

// Run performs the whole handoff in order — write config, base genesis, compose
// the plan, apply the overlay, launch, wire the mesh, deploy governance, init
// and verify etcd — and returns the running node set and the verified cluster
// info. It is the one orchestration the CLI `upgrade run` and app.UpgradeRun
// share, so both drive the identical sequence. etcdTimeout bounds the wait for
// the producer's etcd cluster to form.
func (h *Handoff) Run(ctx context.Context, etcdTimeout time.Duration) (node.NodeSet, poa.Info, error) {
	if _, err := h.WriteConfig(ctx); err != nil {
		return node.NodeSet{}, poa.Info{}, err
	}
	basePath, err := h.BaseGenesis(ctx)
	if err != nil {
		return node.NodeSet{}, poa.Info{}, err
	}
	if err := h.ComposePlan(ctx, basePath); err != nil {
		return node.NodeSet{}, poa.Info{}, err
	}
	if _, err := h.ApplyOverlay(); err != nil {
		return node.NodeSet{}, poa.Info{}, err
	}
	// The producer comes up alone. A poa network's etcd cluster forms only
	// while it is: with the others already running, admin.etcdInit() returns
	// without error and creates nothing, and the producer then never seals.
	// The order is not this function's to invent — the consensus family
	// declares it, and the composition path has followed that declaration since
	// F3. Doing it by hand here is what left the handoff failing at
	// verify-etcd while the same network came up correctly through `chain up`.
	boot, rest := h.phases()
	ns, err := h.LaunchPhase(ctx, boot)
	if err != nil {
		return ns, poa.Info{}, err
	}
	if len(ns.Nodes) == 0 {
		return ns, poa.Info{}, fmt.Errorf("upgrade: launch produced no nodes")
	}
	producer := ns.Nodes[0]
	if err := h.DeployGovernance(ctx, producer); err != nil {
		return ns, poa.Info{}, err
	}
	if err := h.EtcdInit(ctx, producer); err != nil {
		return ns, poa.Info{}, err
	}
	info, err := h.VerifyEtcd(ctx, producer, etcdTimeout)
	if err != nil {
		return ns, poa.Info{}, err
	}
	if len(rest) > 0 {
		more, err := h.LaunchPhase(ctx, rest)
		if err != nil {
			return ns, poa.Info{}, err
		}
		ns.Nodes = append(ns.Nodes, more.Nodes...)
	}
	// The mesh is wired once everyone is up: admin_addPeer needs both ends.
	if err := h.WireMesh(ctx, ns); err != nil {
		return ns, poa.Info{}, err
	}
	return ns, info, nil
}

// phases asks the producer's consensus family how to order the bring-up, and
// renders its answer as the node positions this plan launches in each step.
//
// The family speaks in roles, so the plan's producers are offered as producers
// and everything else as endpoints; what comes back is which of them may start
// together.
func (h *Handoff) phases() (boot, rest []int) {
	roles := make([]node.Role, len(h.Plan.Nodes))
	for i, n := range h.Plan.Nodes {
		roles[i] = node.RoleEN
		if n.Producer {
			roles[i] = node.RoleBP
		}
	}
	for _, phase := range h.From.Family().BringUpPhases(roles) {
		positions := make([]int, 0, len(phase.Nodes))
		for _, oneBased := range phase.Nodes {
			positions = append(positions, oneBased-1)
		}
		if boot == nil {
			boot = positions
			continue
		}
		rest = append(rest, positions...)
	}
	return boot, rest
}

// ProducerIPC is the producer's console socket under the data root.
func (h *Handoff) ProducerIPC(producer node.Node) string {
	return node.Layout{Root: h.in.DataDir}.IPCPath(h.label(producer), h.in.FromBinary)
}

// AwaitFork waits until a successor validator seals the first post-fork
// block, and says which one did. It polls a validator rather than the
// producer: the producer cannot import post-fork blocks, so its head is not
// the handoff's evidence.
func (h *Handoff) AwaitFork(ctx context.Context, ns node.NodeSet, timeout time.Duration) (string, error) {
	// Observe from a SUCCESSOR, asked of the plan rather than guessed from the
	// index. "index != 0" worked while node 0 was the only producer; with fifteen
	// of them it picked producer number two, whose head stops at the fork by
	// design — so a completed handoff was reported as a chain stalled one block
	// short of it, while the successors were hundreds of blocks past.
	producer := make(map[int]bool, len(h.Plan.Nodes))
	for _, n := range h.Plan.Nodes {
		producer[n.Index] = n.Producer
	}
	isProducer := func(i int) bool {
		if len(producer) == 0 {
			// No plan to ask: producers come first, so position 0 is one.
			return i == 0
		}
		return producer[i]
	}
	var target string
	for _, n := range ns.Nodes {
		if !isProducer(n.Index) {
			target = n.RPCURL
			break
		}
	}
	if target == "" {
		return "", fmt.Errorf("upgrade: no successor validator to observe the handoff on")
	}
	// The successor set from the genesis, which is what "the successor produces"
	// has to mean. Checking only that the sealer is not the producer passes for
	// any third address, and it is the check this used to make.
	successors := map[string]bool{}
	for _, v := range h.Plan.Network.WbftValidators {
		successors[strings.ToLower(v)] = true
	}
	if len(successors) == 0 {
		return "", fmt.Errorf("upgrade: the plan names no successor validators, so a handoff cannot be confirmed against them")
	}
	forkBlock := uint64(h.ForkBlock())
	// Enough blocks past the fork to tell a handoff from a single lucky seal, and
	// to see production move between validators. One block only proves the
	// boundary was crossed; it does not prove the successor kept going.
	window := postForkWindow(len(successors))
	last := forkBlock + window
	c := rpc.Dial(target)
	deadline := time.Now().Add(timeout)
	var head uint64
	for time.Now().Before(deadline) {
		hd, err := c.BlockNumber(ctx)
		if err == nil {
			head = hd
			if hd >= last {
				return h.confirmPostFork(ctx, c, forkBlock, last, hd, successors)
			}
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(forkPoll):
		}
	}
	return "", fmt.Errorf("upgrade: head stalled at %d, never reached %d (fork %d + %d blocks) within %s",
		head, last, forkBlock, window, timeout)
}

// postForkWindow is how many blocks past the fork are read to confirm the
// handoff. wbft takes proposer turns round-robin, so a window as wide as the
// successor set gives every validator one turn and a window twice that absorbs
// a validator still catching up at the boundary without reading as a set that
// never rotated. It never drops below postForkBlocks, the floor that separates a
// handoff from one lucky seal.
func postForkWindow(validators int) uint64 {
	if w := uint64(2 * validators); w > postForkBlocks {
		return w
	}
	return postForkBlocks
}

// rotationFloor is how many DISTINCT validators must seal within the window for
// production to count as having moved to the set rather than to one member of
// it. It is wbft's own quorum, floor(2n/3)+1: below that many proposers the
// chain is not being run by the set it declared. Requiring all n would fail on a
// single round change, which is normal operation and not a failed handoff.
func rotationFloor(validators int) int {
	if validators <= 1 {
		return validators
	}
	return 2*validators/3 + 1
}

// confirmPostFork reads every block from the fork boundary to last and requires
// that all of them were sealed by a declared successor validator.
//
// It also reports how many distinct validators sealed them, because a set that
// rotates is the property a multi-validator successor chain has and a single
// validator sealing everything does not — the latter is one node mining, which
// looks identical if only the boundary block is read.
func (h *Handoff) confirmPostFork(
	ctx context.Context, c *rpc.Client, forkBlock, last, head uint64, successors map[string]bool,
) (string, error) {
	sealers := map[string]int{}
	for n := forkBlock + 1; n <= last; n++ {
		var blk struct {
			Miner string `json:"miner"`
		}
		if err := c.Call(ctx, "eth_getBlockByNumber", &blk, fmt.Sprintf("0x%x", n), false); err != nil {
			return "", fmt.Errorf("upgrade: reading block %d after the fork: %w", n, err)
		}
		miner := strings.ToLower(blk.Miner)
		if miner == "" {
			return "", fmt.Errorf("upgrade: block %d names no sealer", n)
		}
		if !successors[miner] {
			return "", fmt.Errorf("upgrade: block %d after the fork was sealed by %s, which is not a declared successor validator — the handoff did not move production to the successor set",
				n, miner)
		}
		sealers[miner]++
	}
	// Every block belonging to the set is not yet the set producing: one validator
	// sealing all of them satisfies that check. The window is wide enough for each
	// member to take a turn, so require a quorum of them to have done so.
	if floor := rotationFloor(len(successors)); len(sealers) < floor {
		return "", fmt.Errorf("upgrade: blocks %d-%d were sealed by only %d of %d successor validator(s), fewer than the %d needed to show production rotating across the set",
			forkBlock+1, last, len(sealers), len(successors), floor)
	}
	return fmt.Sprintf("head %d; blocks %d-%d all sealed by the successor set, across %d of %d validator(s)",
		head, forkBlock+1, last, len(sealers), len(successors)), nil
}

// label is a launched node's directory name. The plan numbers nodes from
// zero and the layout from one.
func (h *Handoff) label(n node.Node) node.Label { return node.LabelFor(n.Index + 1) }

// provisionKeys places each node's identity as its datadir comes up: the
// nodekey in the binary-specific instance directory, the static-nodes list,
// and — for the producer — its keystore.
func (h *Handoff) provisionKeys() func(context.Context, process.NodeSpec, bool) error {
	enodes := h.Plan.Enodes(h.in.host())
	staticNodes, _ := json.MarshalIndent(enodes, "", "  ")
	return func(ctx context.Context, spec process.NodeSpec, producer bool) error {
		// This node's own store. Capturing one store for the whole network sent
		// every node's key material to one machine, which is where a placement
		// across servers quietly collapsed.
		files, _, err := h.machineFiles(spec.Index)
		if err != nil {
			return err
		}
		inst := h.Profile.Chains.To.NodekeyDir
		if producer {
			inst = h.Profile.Chains.From.NodekeyDir
		}
		num := h.order[spec.Index]
		nk, ok := h.Preset.Node(num)
		if !ok {
			return fmt.Errorf("upgrade: preset node %d missing", num)
		}
		dir := filepath.Join(spec.DataDir, inst)
		if err := files.Write(ctx, filepath.Join(dir, "nodekey"), []byte(nk.Nodekey.Hex()), 0o600); err != nil {
			return err
		}
		if err := files.Write(ctx, filepath.Join(dir, "static-nodes.json"), staticNodes, 0o644); err != nil {
			return err
		}
		if !producer {
			return nil
		}
		src := filepath.Join(h.in.PresetDir, fmt.Sprintf("node%d", num), "keystore")
		shipped, err := copyFiles(ctx, files, src, filepath.Join(spec.DataDir, "keystore"))
		if err == nil && producer && len(shipped) > 0 {
			h.producerKeystore = shipped[0]
		}
		if err != nil {
			return fmt.Errorf("upgrade: copy keystore: %w", err)
		}
		return nil
	}
}

// overrides are the account and RPC-namespace knobs: admin on every node,
// because the mesh is wired with admin_addPeer, and the producer's unlocked
// etherbase.
func (h *Handoff) overrides() func(NodeSpec, bool) []nodeconfig.Override {
	fromNS, toNS := h.From.Family().RPCNamespace(), h.To.Family().RPCNamespace()
	pwPath := h.pwPath
	return func(spec NodeSpec, producer bool) []nodeconfig.Override {
		if producer {
			// Each producer unlocks and seals with ITS OWN account.
			//
			// This used to be Producers.Members[0] for every producer, which is
			// invisible while there is one of them and fatal past that: the
			// fifteenth producer was told to unlock the first one's address and
			// exited with "no key for given address or file", because the only
			// keystore on its machine is its own.
			acct := h.producerAccountFor(spec.Index)
			return []nodeconfig.Override{
				{Key: nodeconfig.KeyNAT, Value: "none"},
				{Key: nodeconfig.KeyHTTPAPI, Value: "eth,net,web3," + fromNS + ",admin,miner,txpool,personal"},
				{Key: nodeconfig.KeyEtherbase, Value: acct},
				{Key: nodeconfig.KeyUnlock, Value: acct},
				{Key: nodeconfig.KeyPassword, Value: pwPath},
			}
		}
		return []nodeconfig.Override{
			{Key: nodeconfig.KeyNAT, Value: "none"},
			{Key: nodeconfig.KeyHTTPAPI, Value: "eth,net,web3," + toNS + ",admin,miner,txpool"},
		}
	}
}

// producerAccountFor is the account plan node i seals with: the i-th entry of the
// profile's producer members, which are listed in plan order. A profile with one
// producer gives every producer the same answer, which is what it used to do
// unconditionally.
func (h *Handoff) producerAccountFor(i int) string {
	members := h.Profile.Producers.Members
	if i >= 0 && i < len(members) {
		return members[i]
	}
	return h.ProducerAccount()
}

// poaConfig assembles the wemix governance config: one producer member (its
// unlockable account and devp2p id), the governance env from the profile, and
// an alloc funding the producer and every validator.
func (h *Handoff) poaConfig(prod keyring.Entry) poa.Config {
	prof := h.Profile
	g := prof.Producers.Governance
	env := poa.Env{
		BallotDurationMin: g.BallotDurationMin, BallotDurationMax: g.BallotDurationMax,
		StakingMin: dec(g.StakingMin), StakingMax: dec(g.StakingMax),
		MaxIdleBlockInterval: g.MaxIdleBlockInterval, BlockCreationTime: g.BlockCreationTime,
		BlockRewardAmount: dec(g.BlockRewardAmount), MaxPriorityFeePerGas: dec(g.MaxPriorityFeePerGas),
		RewardDistribution: g.RewardDistribution, MaxBaseFee: dec(g.MaxBaseFee),
		BlockGasLimit: g.BlockGasLimit, BaseFeeMaxChangeRate: g.BaseFeeMaxChangeRate,
		GasTargetPercentage: g.GasTargetPercentage,
	}
	bal := dec(handoffBalance)
	// Every producer and every successor is funded: a producer that cannot pay for
	// its own staking transaction is a member on paper only.
	accounts := make([]poa.Account, 0, len(prof.Producers.Members)+len(h.Preset.NetworkFor(0).Validators))
	for _, a := range prof.Producers.Members {
		accounts = append(accounts, poa.Account{Addr: a, Balance: bal})
	}
	for _, v := range h.Preset.NetworkFor(0).Validators {
		accounts = append(accounts, poa.Account{Addr: v, Balance: bal})
	}
	// The member is the BOOT producer, at the address and port it was actually
	// placed on. The profile's port bases are the single-host fallback, so reading
	// them here published the wrong endpoint the moment a placement decided
	// otherwise — the member's IP and port are what other nodes dial to find it.
	boot := h.bootIndex()
	bootAcct := h.producerAccountFor(boot)
	host, port := h.in.host(), prof.Ports.BaseP2P
	if h.in.Placement != nil {
		if pl, ok := h.in.Placement.Lookup(node.LabelFor(boot + 1)); ok {
			host, port = pl.Host, pl.Ports.P2P
		}
	}
	return poa.Config{
		ExtraData: "chainbench handoff", Staker: bootAcct, Ecosystem: bootAcct,
		Maintenance: bootAcct, FeeCollector: bootAcct, Env: env,
		Members: []poa.Member{{
			Addr: bootAcct, Stake: dec(prof.Producers.Stake), Name: "producer",
			ID: "0x" + prod.PublicKey, IP: host, Port: port, Bootnode: true,
		}},
		Accounts: accounts,
	}
}

// forkPrereqs sets chainId and petersburgBlock on the base genesis, which the
// wemix template omits but the successor requires for fork ordering.
// forkPrereqs reads the base genesis the producer's binary just wrote — through
// the same store it was written to, because on a remote target it is over there
// — and sets the two fields the wemix template omits but the successor needs for
// fork ordering.
func (h *Handoff) forkPrereqs(ctx context.Context, path string, networkID int64) ([]byte, error) {
	b, err := h.bootFiles().Read(ctx, path)
	if err != nil {
		return nil, err
	}
	b, err = genesis.SetConfigSection(b, "chainId", json.RawMessage(strconv.FormatInt(networkID, 10)))
	if err != nil {
		return nil, err
	}
	return genesis.SetConfigSection(b, "petersburgBlock", json.RawMessage("0"))
}

// dec parses a decimal wei string; empty or malformed is zero.
func dec(s string) *big.Int {
	n, ok := new(big.Int).SetString(strings.TrimSpace(s), 10)
	if !ok {
		return big.NewInt(0)
	}
	return n
}

// firstEntry returns the first regular file in dir.
func firstEntry(dir string) (string, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	for _, e := range ents {
		if !e.IsDir() {
			return filepath.Join(dir, e.Name()), nil
		}
	}
	return "", fmt.Errorf("no file in %s", dir)
}

// copyFiles copies the regular files of src into dst. src is read from this
// machine — the key preset is the operator's — while dst is written through
// the file seam, because that side is the target.
// It returns the destination paths it wrote, so a later step names a shipped
// file from the record rather than by listing the destination. Listing is the one
// thing the file interface does not offer, and re-deriving the name locally is
// what made the producer's keystore unfindable on a remote target.
func copyFiles(ctx context.Context, files filestore.Store, src, dst string) ([]string, error) {
	ents, err := os.ReadDir(src)
	if err != nil {
		return nil, err
	}
	var written []string
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			return nil, err
		}
		out := filepath.Join(dst, e.Name())
		if err := files.Write(ctx, out, b, 0o600); err != nil {
			return nil, err
		}
		written = append(written, out)
	}
	return written, nil
}
