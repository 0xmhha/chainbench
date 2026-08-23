package chainsetup

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/0xmhha/chainbench/internal/consensus/poa"
	"github.com/0xmhha/chainbench/internal/consensus/upgrade"
	"github.com/0xmhha/chainbench/internal/core/driver"
	"github.com/0xmhha/chainbench/internal/core/genesis"
	"github.com/0xmhha/chainbench/internal/core/keyring"
	"github.com/0xmhha/chainbench/internal/core/launchopt"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/provision"
	"github.com/0xmhha/chainbench/internal/core/registry"
)

// defaultExec shells out and returns combined output.
func defaultExec(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

// liveHandoff drives the real handoff through the upgrade and poa packages. It
// holds the state each step hands to the next, so the step sequence in
// RunHandoff stays readable.
type liveHandoff struct {
	profile upgrade.Profile
	preset  keyring.Preset
	from    registry.ChainPlugin
	to      registry.ChainPlugin
	plan    upgrade.Plan
	order   []int
	pwPath  string
	overlay bool
}

// NewLiveHandoff returns a HandoffDriver backed by the real binaries.
func NewLiveHandoff() HandoffDriver { return &liveHandoff{} }

func (h *liveHandoff) Prepare(_ context.Context, o HandoffOptions) (string, error) {
	if o.ProfilePath == "" || o.Template == "" {
		return "", fmt.Errorf("a profile and a go-wemix genesis template are required")
	}
	if o.FromBinary == "" || o.ToBinary == "" {
		return "", fmt.Errorf("both --from-binary and --to-binary are required")
	}
	prof, err := upgrade.LoadProfile(o.ProfilePath)
	if err != nil {
		return "", err
	}
	preset, err := keyring.LoadPreset(o.PresetDir)
	if err != nil {
		return "", err
	}
	from, err := registry.Get(prof.Upgrade.From)
	if err != nil {
		return "", err
	}
	to, err := registry.Get(prof.Upgrade.To)
	if err != nil {
		return "", err
	}
	h.profile, h.preset, h.from, h.to = prof, preset, from, to
	h.order = prof.PlanOrderOrDefault()
	return fmt.Sprintf("%s -> %s at %s block %d; %d producer(s) + %d validator(s)",
		prof.Upgrade.From, prof.Upgrade.To, prof.Upgrade.AtFork,
		prof.Upgrade.ForkBlock, prof.Roles.Producers, prof.Roles.Validators), nil
}

func (h *liveHandoff) Config(ctx context.Context, o HandoffOptions) (string, error) {
	prod, ok := h.preset.Node(h.order[0])
	if !ok {
		return "", fmt.Errorf("preset has no node %d (the producer)", h.order[0])
	}
	cfg := poaConfig(h.profile, h.preset, h.profile.Producers.Members[0], prod, defaultHost)
	b, err := cfg.JSON()
	if err != nil {
		return "", err
	}
	path := filepath.Join(o.DataDir, h.from.Manifest().ID+"-config.json")
	if err := o.files().Write(ctx, path, b, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func (h *liveHandoff) BaseGenesis(ctx context.Context, o HandoffOptions, configPath string) (string, error) {
	path := filepath.Join(o.DataDir, "base-genesis.json")
	if err := poa.GenerateGenesis(ctx, poa.Runner(o.Exec), o.FromBinary, configPath, o.Template, path); err != nil {
		return "", fmt.Errorf("%w (is --template go-wemix's own wemix/scripts/genesis-template.json?)", err)
	}
	return path, nil
}

func (h *liveHandoff) Plan(_ context.Context, o HandoffOptions, basePath string) (string, error) {
	base, err := forkPrereqs(basePath, h.profile.Upgrade.NetworkID)
	if err != nil {
		return "", err
	}
	in, err := h.profile.Inputs(base)
	if err != nil {
		return "", err
	}
	in.NodePubkeys = make([]string, len(h.order))
	for i, num := range h.order {
		nk, ok := h.preset.Node(num)
		if !ok {
			return "", fmt.Errorf("preset has no node %d", num)
		}
		in.NodePubkeys[i] = nk.PublicKey
	}
	plan, err := upgrade.BuildPlan(h.from, h.to, in)
	if err != nil {
		return "", err
	}
	h.plan = plan
	return fmt.Sprintf("%d node(s); fork section %q merged; preflight passed", len(plan.Nodes), plan.AtFork), nil
}

func (h *liveHandoff) Overlay(_ context.Context, o HandoffOptions) (string, error) {
	if o.GenesisOverlay == "" {
		return "none", nil
	}
	raw, err := os.ReadFile(o.GenesisOverlay)
	if err != nil {
		return "", err
	}
	var ov struct {
		Genesis json.RawMessage `json:"genesis"`
	}
	if err := json.Unmarshal(raw, &ov); err != nil {
		return "", fmt.Errorf("bad overlay %q: %w", o.GenesisOverlay, err)
	}
	if len(ov.Genesis) == 0 {
		return "overlay has no genesis fragment", nil
	}
	merged, err := genesis.MergeOverride(h.plan.Genesis, ov.Genesis)
	if err != nil {
		return "", err
	}
	h.plan.Genesis = merged
	h.overlay = true
	return fmt.Sprintf("merged %s", o.GenesisOverlay), nil
}

func (h *liveHandoff) Launch(ctx context.Context, o HandoffOptions) (node.NodeSet, error) {
	h.pwPath = filepath.Join(o.DataDir, "password")
	if err := o.files().Write(ctx, h.pwPath, []byte(h.preset.Password), 0o600); err != nil {
		return node.NodeSet{}, err
	}
	producerAcct := h.profile.Producers.Members[0]
	opts := upgrade.LaunchOptions{
		DataRoot:   o.DataDir,
		FromBinary: o.FromBinary, ToBinary: o.ToBinary,
		FromFamily: h.from.Family(), ToFamily: h.to.Family(),
		Host:          defaultHost,
		ProvisionKeys: h.provisionKeys(o),
		Overrides:     handoffOverrides(producerAcct, h.pwPath, h.from.Family().RPCNamespace(), h.to.Family().RPCNamespace()),
	}
	return upgrade.Launch(ctx, driver.NewLocalDriver(), h.plan, opts)
}

func (h *liveHandoff) WireMesh(ctx context.Context, ns node.NodeSet) (string, error) {
	endpoints := make([]string, len(ns.Nodes))
	for i, n := range ns.Nodes {
		endpoints[i] = n.RPCURL
	}
	if err := upgrade.WaitEndpointsReady(ctx, endpoints, 30*time.Second); err != nil {
		return "", fmt.Errorf("nodes not ready for mesh: %w", err)
	}
	if err := upgrade.WireMesh(ctx, upgrade.DefaultPeerCaller(), endpoints, h.plan.Enodes(defaultHost)); err != nil {
		return "", err
	}
	return fmt.Sprintf("%d endpoint(s) meshed", len(endpoints)), nil
}

func (h *liveHandoff) DeployGovernance(ctx context.Context, o HandoffOptions, producer node.Node) (string, error) {
	ipc := h.ProducerIPC(o, producer)
	if err := waitIPC(ipc, 30*time.Second); err != nil {
		return "", err
	}
	ksDir := filepath.Join(o.DataDir, fmt.Sprintf("node%d", producer.Index+1), "keystore")
	ksFile, err := firstEntry(ksDir)
	if err != nil {
		return "", fmt.Errorf("producer keystore: %w", err)
	}
	cfgPath := filepath.Join(o.DataDir, h.from.Manifest().ID+"-config.json")
	if err := poa.DeployGovernance(ctx, poa.Runner(o.Exec), o.FromBinary, ipc, cfgPath, ksFile, h.pwPath); err != nil {
		return "", err
	}
	return "deploy-governance returned success (effect is checked by verify-etcd)", nil
}

func (h *liveHandoff) EtcdInit(ctx context.Context, o HandoffOptions, producer node.Node) (string, error) {
	if err := poa.EtcdInit(ctx, poa.Runner(o.Exec), o.FromBinary, h.ProducerIPC(o, producer)); err != nil {
		return "", err
	}
	return "admin.etcdInit() returned without error (effect is checked by verify-etcd)", nil
}

func (h *liveHandoff) ProducerIPC(o HandoffOptions, producer node.Node) string {
	return producerIPCPath(o.DataDir, o.FromBinary, producer)
}

func (h *liveHandoff) ForkBlock() int64 { return h.profile.Upgrade.ForkBlock }

func (h *liveHandoff) ProducerAccount() string { return h.profile.Producers.Members[0] }

// defaultHost is the address local nodes bind and advertise.
const defaultHost = "127.0.0.1"

// provisionKeys places each node's identity: the nodekey in the binary-specific
// instance directory, the static-nodes list, and the producer's keystore.
func (h *liveHandoff) provisionKeys(o HandoffOptions) func(context.Context, driver.NodeSpec, bool) error {
	enodes := h.plan.Enodes(defaultHost)
	staticNodes, _ := json.MarshalIndent(enodes, "", "  ")
	return func(ctx context.Context, spec driver.NodeSpec, producer bool) error {
		inst := h.profile.Chains.To.NodekeyDir
		if producer {
			inst = h.profile.Chains.From.NodekeyDir
		}
		nk, ok := h.preset.Node(h.order[spec.Index])
		if !ok {
			return fmt.Errorf("preset node %d missing", h.order[spec.Index])
		}
		// The store creates the instance dir along with the first file in it.
		dir := filepath.Join(spec.DataDir, inst)
		if err := o.files().Write(ctx, filepath.Join(dir, "nodekey"), []byte(nk.Nodekey.Hex()), 0o600); err != nil {
			return err
		}
		if err := o.files().Write(ctx, filepath.Join(dir, "static-nodes.json"), staticNodes, 0o644); err != nil {
			return err
		}
		if !producer {
			return nil
		}
		src := filepath.Join(o.PresetDir, fmt.Sprintf("node%d", h.order[spec.Index]), "keystore")
		return copyFiles(ctx, o.files(), src, filepath.Join(spec.DataDir, "keystore"))
	}
}

// handoffOverrides adds the account and RPC-namespace knobs. admin is required
// on every node because the mesh is wired with admin_addPeer.
func handoffOverrides(producerAcct, pwPath, fromNS, toNS string) func(upgrade.NodeSpec, bool) []launchopt.Override {
	return func(_ upgrade.NodeSpec, producer bool) []launchopt.Override {
		if producer {
			return []launchopt.Override{
				{Key: launchopt.KeyNAT, Value: "none"},
				{Key: launchopt.KeyHTTPAPI, Value: "eth,net,web3," + fromNS + ",admin,miner,txpool,personal"},
				{Key: launchopt.KeyEtherbase, Value: producerAcct},
				{Key: launchopt.KeyUnlock, Value: producerAcct},
				{Key: launchopt.KeyPassword, Value: pwPath},
			}
		}
		return []launchopt.Override{
			{Key: launchopt.KeyNAT, Value: "none"},
			{Key: launchopt.KeyHTTPAPI, Value: "eth,net,web3," + toNS + ",admin,miner,txpool"},
		}
	}
}

// poaConfig assembles the wemix governance config from the profile.
func poaConfig(prof upgrade.Profile, preset keyring.Preset, producerAcct string, prod keyring.Entry, host string) poa.Config {
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
	bal := dec("1000000000000000000000000000")
	accounts := []poa.Account{{Addr: producerAcct, Balance: bal}}
	for _, v := range preset.NetworkFor(0).Validators {
		accounts = append(accounts, poa.Account{Addr: v, Balance: bal})
	}
	return poa.Config{
		ExtraData: "chainbench handoff", Staker: producerAcct, Ecosystem: producerAcct,
		Maintenance: producerAcct, FeeCollector: producerAcct, Env: env,
		Members: []poa.Member{{
			Addr: producerAcct, Stake: dec(prof.Producers.Stake), Name: "producer",
			ID: "0x" + prod.PublicKey, IP: host, Port: prof.Ports.BaseP2P, Bootnode: true,
		}},
		Accounts: accounts,
	}
}

// forkPrereqs sets chainId and petersburgBlock, which the wemix template omits
// but the successor requires for fork ordering.
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

// dec parses a decimal wei string (0 when empty or malformed).
func dec(s string) *big.Int {
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return big.NewInt(0)
	}
	return n
}

// waitIPC waits for a unix socket to appear.
func waitIPC(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fi, err := os.Stat(path); err == nil && fi.Mode()&os.ModeSocket != 0 {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("the producer IPC never appeared at %s", path)
}

// firstEntry returns the first file in dir.
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

// copyFiles copies the regular files of src into dst.
//
// src is read from this machine — the key preset is the operator's — while dst
// is written through the seam, because that side is the target. The asymmetry
// is the point: it is what lets the same call place a keystore on a remote
// node instead of beside the preset it came from.
func copyFiles(ctx context.Context, files provision.FileStore, src, dst string) error {
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
