package upgrade

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/0xmhha/chainbench/internal/consensus/poa"
	"github.com/0xmhha/chainbench/internal/core/filestore"
	"github.com/0xmhha/chainbench/internal/core/genesis"
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
	// readyWait bounds the wait for every node's RPC before the mesh is wired.
	readyWait = 30 * time.Second
	// forkPoll is how often the successor's head is read while waiting for the
	// fork.
	forkPoll = time.Second
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
	preset, err := store.LoadPreset(in.PresetDir)
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
	prod, _ := h.Preset.Node(h.order[0])
	cfg := h.poaConfig(prod)
	b, err := cfg.JSON()
	if err != nil {
		return "", err
	}
	path := filepath.Join(h.in.DataDir, h.From.Manifest().ID+"-config.json")
	// Write creates the parents, so the data dir needs no separate mkdir.
	if err := h.in.files().Write(ctx, path, b, 0o644); err != nil {
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
	path := filepath.Join(h.in.DataDir, "base-genesis.json")
	if err := poa.GenerateGenesis(ctx, h.in.exec(), h.in.FromBinary, h.configPath, h.in.Template, path); err != nil {
		return "", fmt.Errorf("%w (is --template go-wemix's own wemix/scripts/genesis-template.json?)", err)
	}
	return path, nil
}

// ComposePlan lifts the successor's fork section onto the base genesis and
// builds the plan, with every node's devp2p pubkey so the mesh can be wired.
func (h *Handoff) ComposePlan(basePath string) error {
	base, err := forkPrereqs(basePath, h.Profile.Upgrade.NetworkID)
	if err != nil {
		return err
	}
	in, err := h.Profile.Inputs(base)
	if err != nil {
		return err
	}
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
	h.pwPath = filepath.Join(h.in.DataDir, "password")
	if err := h.in.files().Write(ctx, h.pwPath, []byte(h.Preset.Password), 0o600); err != nil {
		return node.NodeSet{}, err
	}
	opts := LaunchOptions{
		DataRoot:   h.in.DataDir,
		FromBinary: h.in.FromBinary, ToBinary: h.in.ToBinary,
		FromFamily: h.From.Family(), ToFamily: h.To.Family(),
		Host:          h.in.host(),
		ProvisionKeys: h.provisionKeys(),
		Overrides:     h.overrides(),
		Files:         h.in.Files,
		Only:          only,
	}
	return Launch(ctx, h.in.driver(), h.Plan, opts)
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
	if err := poa.WaitForIPC(ctx, ipc, ipcWait); err != nil {
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
	if err := poa.WaitProducing(ctx, h.in.exec(), h.in.FromBinary, ipc, producingWait); err != nil {
		return err
	}
	ksDir := node.Layout{Root: h.in.DataDir}.KeystoreDir(h.label(producer))
	ksFile, err := firstEntry(ksDir)
	if err != nil {
		return fmt.Errorf("upgrade: producer keystore: %w", err)
	}
	return poa.DeployGovernance(ctx, h.in.exec(), h.in.FromBinary, ipc, h.configPath, ksFile, h.pwPath)
}

// EtcdInit calls admin.etcdInit() on the producer. Its return says only that
// the call was made; VerifyEtcd says whether the cluster formed.
func (h *Handoff) EtcdInit(ctx context.Context, producer node.Node) error {
	return poa.EtcdInit(ctx, h.in.exec(), h.in.FromBinary, h.ProducerIPC(producer))
}

// VerifyEtcd polls the producer until its etcd cluster is non-empty, or the
// window passes. This is the step whose absence let a failed bootstrap report
// success: admin.etcdInit() exits 0 whether or not a cluster came up.
func (h *Handoff) VerifyEtcd(ctx context.Context, producer node.Node, timeout time.Duration) (poa.Info, error) {
	return poa.WaitEtcdCluster(ctx, h.in.exec(), h.in.FromBinary, h.ProducerIPC(producer), timeout, 0)
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
	if err := h.ComposePlan(basePath); err != nil {
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
	var target string
	for _, n := range ns.Nodes {
		if n.Index != 0 {
			target = n.RPCURL
			break
		}
	}
	if target == "" {
		return "", fmt.Errorf("upgrade: no successor validator to observe the handoff on")
	}
	forkBlock := h.ForkBlock()
	c := rpc.Dial(target)
	producer := strings.ToLower(h.ProducerAccount())
	deadline := time.Now().Add(timeout)
	var head uint64
	for time.Now().Before(deadline) {
		if hd, err := c.BlockNumber(ctx); err == nil {
			head = hd
			if hd > uint64(forkBlock) {
				var blk struct {
					Miner string `json:"miner"`
				}
				if err := c.Call(ctx, "eth_getBlockByNumber", &blk, fmt.Sprintf("0x%x", forkBlock+1), false); err == nil {
					miner := strings.ToLower(blk.Miner)
					if miner != "" && miner != producer {
						return fmt.Sprintf("head %d; block %d sealed by %s (successor)", hd, forkBlock+1, miner), nil
					}
				}
			}
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(forkPoll):
		}
	}
	return "", fmt.Errorf("upgrade: head stalled at %d, never crossed fork block %d within %s", head, forkBlock, timeout)
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
	files := h.in.files()
	return func(ctx context.Context, spec process.NodeSpec, producer bool) error {
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
		if err := copyFiles(ctx, files, src, filepath.Join(spec.DataDir, "keystore")); err != nil {
			return fmt.Errorf("upgrade: copy keystore: %w", err)
		}
		return nil
	}
}

// overrides are the account and RPC-namespace knobs: admin on every node,
// because the mesh is wired with admin_addPeer, and the producer's unlocked
// etherbase.
func (h *Handoff) overrides() func(NodeSpec, bool) []nodeconfig.Override {
	producerAcct := h.ProducerAccount()
	fromNS, toNS := h.From.Family().RPCNamespace(), h.To.Family().RPCNamespace()
	pwPath := h.pwPath
	return func(_ NodeSpec, producer bool) []nodeconfig.Override {
		if producer {
			return []nodeconfig.Override{
				{Key: nodeconfig.KeyNAT, Value: "none"},
				{Key: nodeconfig.KeyHTTPAPI, Value: "eth,net,web3," + fromNS + ",admin,miner,txpool,personal"},
				{Key: nodeconfig.KeyEtherbase, Value: producerAcct},
				{Key: nodeconfig.KeyUnlock, Value: producerAcct},
				{Key: nodeconfig.KeyPassword, Value: pwPath},
			}
		}
		return []nodeconfig.Override{
			{Key: nodeconfig.KeyNAT, Value: "none"},
			{Key: nodeconfig.KeyHTTPAPI, Value: "eth,net,web3," + toNS + ",admin,miner,txpool"},
		}
	}
}

// poaConfig assembles the wemix governance config: one producer member (its
// unlockable account and devp2p id), the governance env from the profile, and
// an alloc funding the producer and every validator.
func (h *Handoff) poaConfig(prod keyring.Entry) poa.Config {
	prof := h.Profile
	producerAcct := h.ProducerAccount()
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
	accounts := []poa.Account{{Addr: producerAcct, Balance: bal}}
	for _, v := range h.Preset.NetworkFor(0).Validators {
		accounts = append(accounts, poa.Account{Addr: v, Balance: bal})
	}
	return poa.Config{
		ExtraData: "chainbench handoff", Staker: producerAcct, Ecosystem: producerAcct,
		Maintenance: producerAcct, FeeCollector: producerAcct, Env: env,
		Members: []poa.Member{{
			Addr: producerAcct, Stake: dec(prof.Producers.Stake), Name: "producer",
			ID: "0x" + prod.PublicKey, IP: h.in.host(), Port: prof.Ports.BaseP2P, Bootnode: true,
		}},
		Accounts: accounts,
	}
}

// forkPrereqs sets chainId and petersburgBlock on the base genesis, which the
// wemix template omits but the successor requires for fork ordering.
func forkPrereqs(path string, networkID int64) ([]byte, error) {
	b, err := os.ReadFile(path)
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
func copyFiles(ctx context.Context, files filestore.Store, src, dst string) error {
	ents, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			return err
		}
		if err := files.Write(ctx, filepath.Join(dst, e.Name()), b, 0o600); err != nil {
			return err
		}
	}
	return nil
}
