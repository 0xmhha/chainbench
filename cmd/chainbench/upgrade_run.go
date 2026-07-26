package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/pkg/consensus/poa"
	"github.com/0xmhha/chainbench/pkg/consensus/upgrade"
	"github.com/0xmhha/chainbench/pkg/core/driver"
	"github.com/0xmhha/chainbench/pkg/core/genesis"
	"github.com/0xmhha/chainbench/pkg/core/keys"
	"github.com/0xmhha/chainbench/pkg/core/node"
	"github.com/0xmhha/chainbench/pkg/core/registry"
	"github.com/0xmhha/chainbench/pkg/core/rpc"
)

// bigDec parses a decimal wei string into a *big.Int (0 on empty/invalid).
func bigDec(s string) *big.Int {
	n, ok := new(big.Int).SetString(strings.TrimSpace(s), 10)
	if !ok {
		return big.NewInt(0)
	}
	return n
}

// execRunner shells out to a binary, returning combined output. It is the live
// poa.Runner backing the genesis/deploy-governance/etcdInit steps.
func execRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

func newUpgradeRunCmd() *cobra.Command {
	var profilePath, presetDir, fromBinary, toBinary, template, dataDir string
	var waitFor int
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Launch and bootstrap the full concurrent handoff from a golden profile",
		Long: "Composes the handoff framework end to end: build the producer's base " +
			"genesis, merge the successor fork section, launch the mixed binaries " +
			"concurrently, wire a full peer mesh, and bootstrap governance + etcd on " +
			"the producer. Requires the built binaries, etcd, and a preset key set.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if profilePath == "" || template == "" || dataDir == "" {
				return fmt.Errorf("--profile, --template, and --data-dir are required")
			}
			ctx := cmd.Context()
			out := cmd.OutOrStdout()

			prof, err := upgrade.LoadProfile(profilePath)
			if err != nil {
				return err
			}
			preset, err := keys.LoadPreset(presetDir)
			if err != nil {
				return err
			}
			fromBin, err := resolveBinary(fromBinary, prof.Chains.From.Binary)
			if err != nil {
				return fmt.Errorf("from binary: %w", err)
			}
			toBin, err := resolveBinary(toBinary, prof.Chains.To.Binary)
			if err != nil {
				return fmt.Errorf("to binary: %w", err)
			}
			from, err := registry.Get(prof.Upgrade.From)
			if err != nil {
				return err
			}
			to, err := registry.Get(prof.Upgrade.To)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(dataDir, 0o755); err != nil {
				return err
			}
			order := prof.PlanOrderOrDefault()
			host := "127.0.0.1"

			// 1. wemix governance config (producer only) -> base genesis.
			producerAcct := prof.Producers.Members[0]
			prodPreset, ok := preset.Node(order[0])
			if !ok {
				return fmt.Errorf("preset has no node %d (producer)", order[0])
			}
			cfg := buildPoAConfig(prof, preset, producerAcct, prodPreset, host)
			cfgBytes, err := cfg.JSON()
			if err != nil {
				return err
			}
			configPath := filepath.Join(dataDir, from.Manifest().ID+"-config.json")
			if err := os.WriteFile(configPath, cfgBytes, 0o644); err != nil {
				return err
			}
			basePath := filepath.Join(dataDir, "base-genesis.json")
			if err := poa.GenerateGenesis(ctx, execRunner, fromBin, configPath, template, basePath); err != nil {
				return err
			}
			baseGenesis, err := forceForkPrereqs(basePath, prof.Upgrade.NetworkID)
			if err != nil {
				return err
			}

			// 2. plan (with per-node pubkeys for the mesh).
			in, err := prof.Inputs(baseGenesis)
			if err != nil {
				return err
			}
			in.NodePubkeys = make([]string, len(order))
			for i, presetNum := range order {
				nk, ok := preset.Node(presetNum)
				if !ok {
					return fmt.Errorf("preset has no node %d", presetNum)
				}
				in.NodePubkeys[i] = nk.PublicKey
			}
			plan, err := upgrade.BuildPlan(from, to, in)
			if err != nil {
				return err
			}

			// 3. compose launch options: key provisioning + account/RPC flags.
			pwPath := filepath.Join(dataDir, "password")
			if err := os.WriteFile(pwPath, []byte(preset.Password), 0o600); err != nil {
				return err
			}
			opts := upgrade.LaunchOptions{
				DataRoot:   dataDir,
				FromBinary: fromBin, ToBinary: toBin,
				FromFamily: from.Family(), ToFamily: to.Family(),
				Host:          host,
				ProvisionKeys: provisionKeysFn(prof, preset, order, plan, host, presetDir),
				ExtraArgs:     extraArgsFn(producerAcct, pwPath, from.Family().RPCNamespace(), to.Family().RPCNamespace()),
				WaitReady: func(ctx context.Context, eps []string) error {
					return upgrade.WaitEndpointsReady(ctx, eps, 30*time.Second)
				},
			}
			bootstrap := func(ctx context.Context, producer node.Node) error {
				ipc := filepath.Join(dataDir, fmt.Sprintf("node%d", producer.Index+1), filepath.Base(fromBin)+".ipc")
				if err := waitForIPC(ipc, 20*time.Second); err != nil {
					return err
				}
				ks := filepath.Join(dataDir, fmt.Sprintf("node%d", producer.Index+1), "keystore")
				ksFile, err := firstFile(ks)
				if err != nil {
					return err
				}
				if err := poa.DeployGovernance(ctx, execRunner, fromBin, ipc, configPath, ksFile, pwPath); err != nil {
					return err
				}
				return poa.EtcdInit(ctx, execRunner, fromBin, ipc)
			}

			// 4. run it.
			fmt.Fprintf(out, "handoff %s -> %s at %s block %s; %d nodes; launching...\n",
				plan.From.ID, plan.To.ID, plan.AtFork, in.ForkBlock, len(plan.Nodes))
			ns, err := upgrade.LaunchHandoff(ctx, driver.NewLocalDriver(), plan, opts, upgrade.DefaultPeerCaller(), bootstrap)
			if err != nil {
				return err
			}
			for _, n := range ns.Nodes {
				fmt.Fprintf(out, "  node%d  %s  pid=%d\n", n.Index+1, n.RPCURL, n.PID)
			}
			fmt.Fprintf(out, "governance deployed, etcd initialized, mesh wired.\n")

			if waitFor > 0 {
				return awaitHandoff(ctx, out, ns, prof.Upgrade.ForkBlock, producerAcct, waitFor)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&profilePath, "profile", "", "golden upgrade profile (profiles/*.yaml)")
	cmd.Flags().StringVar(&presetDir, "preset", "keys/preset", "preset key set directory")
	cmd.Flags().StringVar(&fromBinary, "from-binary", "", "from-chain (producer) binary path")
	cmd.Flags().StringVar(&toBinary, "to-binary", "", "to-chain (validator) binary path")
	cmd.Flags().StringVar(&template, "template", "", "wemix genesis template path")
	cmd.Flags().StringVar(&dataDir, "data-dir", "", "node data root")
	cmd.Flags().IntVar(&waitFor, "wait", 0, "seconds to poll for the post-fork handoff (0=don't wait)")
	return cmd
}

// buildPoAConfig assembles the wemix governance config: a single producer member
// (its unlockable account + devp2p id), the governance env from the profile, and
// an alloc funding the producer and every validator.
func buildPoAConfig(prof upgrade.Profile, preset keys.Preset, producerAcct string, prod keys.NodeKey, host string) poa.Config {
	g := prof.Producers.Governance
	env := poa.Env{
		BallotDurationMin: g.BallotDurationMin, BallotDurationMax: g.BallotDurationMax,
		StakingMin: bigDec(g.StakingMin), StakingMax: bigDec(g.StakingMax),
		MaxIdleBlockInterval: g.MaxIdleBlockInterval, BlockCreationTime: g.BlockCreationTime,
		BlockRewardAmount: bigDec(g.BlockRewardAmount), MaxPriorityFeePerGas: bigDec(g.MaxPriorityFeePerGas),
		RewardDistribution: g.RewardDistribution, MaxBaseFee: bigDec(g.MaxBaseFee),
		BlockGasLimit: g.BlockGasLimit, BaseFeeMaxChangeRate: g.BaseFeeMaxChangeRate,
		GasTargetPercentage: g.GasTargetPercentage,
	}
	bal := bigDec("1000000000000000000000000000")
	accounts := []poa.Account{{Addr: producerAcct, Balance: bal}}
	for _, v := range preset.Validators {
		accounts = append(accounts, poa.Account{Addr: v, Balance: bal})
	}
	return poa.Config{
		ExtraData: "chainbench handoff", Staker: producerAcct, Ecosystem: producerAcct,
		Maintenance: producerAcct, FeeCollector: producerAcct, Env: env,
		Members: []poa.Member{{
			Addr: producerAcct, Stake: bigDec(prof.Producers.Stake), Name: "producer",
			ID: "0x" + prod.PublicKey, IP: host, Port: prof.Ports.BaseP2P, Bootnode: true,
		}},
		Accounts: accounts,
	}
}

// forceForkPrereqs sets chainId and petersburgBlock on the base genesis, which
// the wemix template omits but go-wbft requires for the fork ordering.
func forceForkPrereqs(path string, networkID int64) ([]byte, error) {
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

// provisionKeysFn returns the ProvisionKeys hook: it places each node's node key
// in the binary-specific instance dir, writes static-nodes, and copies the
// producer's keystore.
func provisionKeysFn(prof upgrade.Profile, preset keys.Preset, order []int, plan upgrade.Plan, host, presetDir string) func(context.Context, driver.NodeSpec, bool) error {
	enodes := plan.Enodes(host)
	staticNodes, _ := json.MarshalIndent(enodes, "", "  ")
	return func(_ context.Context, spec driver.NodeSpec, producer bool) error {
		inst := prof.Chains.To.NodekeyDir
		if producer {
			inst = prof.Chains.From.NodekeyDir
		}
		presetNum := order[spec.Index]
		nk, ok := preset.Node(presetNum)
		if !ok {
			return fmt.Errorf("preset node %d missing", presetNum)
		}
		instDir := filepath.Join(spec.DataDir, inst)
		if err := os.MkdirAll(instDir, 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(instDir, "nodekey"), []byte(nk.Nodekey), 0o600); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(instDir, "static-nodes.json"), staticNodes, 0o644); err != nil {
			return err
		}
		if producer {
			src := filepath.Join(presetDir, fmt.Sprintf("node%d", presetNum), "keystore")
			dst := filepath.Join(spec.DataDir, "keystore")
			if err := copyDirFiles(src, dst); err != nil {
				return fmt.Errorf("copy keystore: %w", err)
			}
		}
		return nil
	}
}

// extraArgsFn returns the ExtraArgs hook: --http.api (admin is needed for the
// mesh's admin_addPeer) on every node, plus the producer's unlock/etherbase.
func extraArgsFn(producerAcct, pwPath, fromNS, toNS string) func(upgrade.NodeSpec, bool) []string {
	return func(_ upgrade.NodeSpec, producer bool) []string {
		if producer {
			return []string{
				"--nat", "none",
				"--http.api", "eth,net,web3," + fromNS + ",admin,miner,txpool,personal",
				"--miner.etherbase", producerAcct,
				"--unlock", producerAcct, "--password", pwPath,
			}
		}
		return []string{"--nat", "none", "--http.api", "eth,net,web3," + toNS + ",admin,miner,txpool"}
	}
}

func waitForIPC(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if fi, err := os.Stat(path); err == nil && fi.Mode()&os.ModeSocket != 0 {
			return nil
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("producer IPC never appeared at %s", path)
}

func firstFile(dir string) (string, error) {
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

func copyDirFiles(src, dst string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
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
		if err := os.WriteFile(filepath.Join(dst, e.Name()), b, 0o600); err != nil {
			return err
		}
	}
	return nil
}

// awaitHandoff polls a validator (not the producer, which cannot import
// post-fork blocks) until the head passes the fork and a non-producer sealed the
// first post-fork block.
func awaitHandoff(ctx context.Context, out interface{ Write([]byte) (int, error) }, ns node.NodeSet, forkBlock int64, producerAcct string, seconds int) error {
	var validator string
	for _, n := range ns.Nodes {
		if n.Index != 0 {
			validator = n.RPCURL
			break
		}
	}
	cli := rpc.Dial(validator)
	producer := strings.ToLower(producerAcct)
	for i := 0; i < seconds; i++ {
		head, err := cli.BlockNumber(ctx)
		if err == nil && head > uint64(forkBlock) {
			var blk struct {
				Miner string `json:"miner"`
			}
			_ = cli.Call(ctx, "eth_getBlockByNumber", &blk, fmt.Sprintf("0x%x", forkBlock+1), false)
			miner := strings.ToLower(blk.Miner)
			if miner != "" && miner != producer {
				fmt.Fprintf(out, "handoff confirmed: head=%d, block %d sealed by %s (go-wbft validator)\n", head, forkBlock+1, miner)
				return nil
			}
		}
		time.Sleep(time.Second)
	}
	return fmt.Errorf("handoff not observed within %ds", seconds)
}
