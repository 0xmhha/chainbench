package main

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/pkg/chains/wemix/deploy"
	"github.com/0xmhha/chainbench/pkg/core/remote"
)

// newRemoteCmd is the remote closed-network deployment command group (the
// chainbench-native migration of the wemix4 SSH suite). Phase 2: `keys read`.
func newRemoteCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "remote",
		Short: "Remote closed-network deployment (wemix+etcd cluster over SSH)",
	}
	c.AddCommand(newRemoteKeysCmd(), newRemoteDeployCmd(), newRemoteBootstrapCmd(), newRemoteHandoffCmd())
	return c
}

func newRemoteHandoffCmd() *cobra.Command {
	var (
		clusterPath  string
		credsPath    string
		accountsPath string
		wait         int
	)
	cmd := &cobra.Command{
		Use:   "handoff",
		Short: "Wait for and confirm the wemix->wbft Croissant handoff",
		Long: "Polls a go-wbft validator's RPC over an SSH tunnel (closed network)\n" +
			"until the chain crosses the Croissant block and the next block is sealed\n" +
			"by a validator rather than a wemix producer — i.e. the hardfork handoff\n" +
			"completed and go-wbft is producing.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := deploy.LoadCluster(clusterPath)
			if err != nil {
				return err
			}
			cr, err := deploy.LoadCredentials(credsPath)
			if err != nil {
				return err
			}
			a, err := deploy.LoadAccounts(accountsPath)
			if err != nil {
				return err
			}
			hostKey, err := remote.ResolveHostKeyCallback(os.Getenv)
			if err != nil {
				return err
			}
			miner, err := deploy.WaitHandoff(cmd.Context(), c, cr, hostKey, a.ProducerAddrs(), time.Duration(wait)*time.Second, os.Getenv)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "handoff confirmed: block %d sealed by %s (go-wbft validator)\n", c.CroissantBlock+1, miner)
			return nil
		},
	}
	cmd.Flags().StringVar(&clusterPath, "cluster", "", "cluster config file (cluster.yaml)")
	cmd.Flags().StringVar(&credsPath, "credentials", "", "SSH credentials file (or CHAINBENCH_REMOTE_USER / CHAINBENCH_REMOTE_PASS)")
	cmd.Flags().StringVar(&accountsPath, "accounts", "", "accounts file (to read the producer addresses to exclude)")
	cmd.Flags().IntVar(&wait, "wait", 300, "seconds to poll for the handoff")
	_ = cmd.MarkFlagRequired("cluster")
	_ = cmd.MarkFlagRequired("accounts")
	return cmd
}

func newRemoteBootstrapCmd() *cobra.Command {
	var (
		clusterPath  string
		credsPath    string
		accountsPath string
	)
	cmd := &cobra.Command{
		Use:   "bootstrap",
		Short: "Deploy governance + initialize etcd on the boot producer",
		Long: "On the cluster's boot producer (first wemix_bp, already launched by\n" +
			"`remote deploy`), builds the wemix governance config from the accounts,\n" +
			"deploys the governance contracts, and initializes the embedded etcd\n" +
			"cluster. No external etcd — gwemix embeds one.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := deploy.LoadCluster(clusterPath)
			if err != nil {
				return err
			}
			cr, err := deploy.LoadCredentials(credsPath)
			if err != nil {
				return err
			}
			a, err := deploy.LoadAccounts(accountsPath)
			if err != nil {
				return err
			}
			cfg, err := deploy.BuildWemixConfig(c, a)
			if err != nil {
				return err
			}
			hostKey, err := remote.ResolveHostKeyCallback(os.Getenv)
			if err != nil {
				return err
			}
			if err := deploy.Bootstrap(cmd.Context(), c, cr, hostKey, cfg, os.Getenv); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "governance deployed, etcd initialized")
			return nil
		},
	}
	cmd.Flags().StringVar(&clusterPath, "cluster", "", "cluster config file (cluster.yaml)")
	cmd.Flags().StringVar(&credsPath, "credentials", "", "SSH credentials file (or CHAINBENCH_REMOTE_USER / CHAINBENCH_REMOTE_PASS)")
	cmd.Flags().StringVar(&accountsPath, "accounts", "", "accounts file (validator/producer material)")
	_ = cmd.MarkFlagRequired("cluster")
	_ = cmd.MarkFlagRequired("accounts")
	return cmd
}

func newRemoteDeployCmd() *cobra.Command {
	var (
		clusterPath string
		credsPath   string
		genesisFile string
		dryRun      bool
	)
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Provision and launch the wemix+etcd cluster over SSH",
		Long: "Builds the per-server launch plan from the cluster config (binary by\n" +
			"role, ports, node config), then provisions and launches each server over\n" +
			"SSH in launch order (endpoints/bootnodes before producers). Keys are read\n" +
			"from the servers, not shipped. --dry-run prints the plan without connecting.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := deploy.LoadCluster(clusterPath)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if dryRun {
				fmt.Fprint(out, deploy.Describe(c, deploy.BuildNodeSpecs(c, nil)))
				return nil
			}
			cr, err := deploy.LoadCredentials(credsPath)
			if err != nil {
				return err
			}
			hostKey, err := remote.ResolveHostKeyCallback(os.Getenv)
			if err != nil {
				return err
			}
			var genesis []byte
			if genesisFile != "" {
				if genesis, err = os.ReadFile(genesisFile); err != nil {
					return fmt.Errorf("read genesis: %w", err)
				}
			}
			nodes, err := deploy.Deploy(cmd.Context(), c, cr, hostKey, genesis, nil, os.Getenv)
			if err != nil {
				return err
			}
			for _, n := range nodes {
				fmt.Fprintf(out, "launched node %d  %s  pid=%d\n", n.Index, n.RPCURL, n.PID)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&clusterPath, "cluster", "", "cluster config file (cluster.yaml)")
	cmd.Flags().StringVar(&credsPath, "credentials", "", "SSH credentials file (or CHAINBENCH_REMOTE_USER / CHAINBENCH_REMOTE_PASS)")
	cmd.Flags().StringVar(&genesisFile, "genesis", "", "local genesis file to ship + init on each server (empty: use the server's genesis_file)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the deploy plan without connecting")
	_ = cmd.MarkFlagRequired("cluster")
	return cmd
}

func newRemoteKeysCmd() *cobra.Command {
	c := &cobra.Command{Use: "keys", Short: "Read key material from remote cluster servers"}
	c.AddCommand(newRemoteKeysReadCmd())
	return c
}

func newRemoteKeysReadCmd() *cobra.Command {
	var (
		clusterPath string
		credsPath   string
		server      int
		keystoreDir string
		accountsOut string
	)
	cmd := &cobra.Command{
		Use:   "read",
		Short: "Read each server's address/BLS keys (bootnode) and pull keystores",
		Long: "Reads validator identity from the remote servers over SSH: derives the\n" +
			"address + BLS public key/PoP via `bootnode -writeaddress`, and (with\n" +
			"--keystore-dir) pulls the coinbase/operator keystores locally. No external\n" +
			"etcd is needed (gwemix embeds it). SSH password comes from the credentials\n" +
			"file or CHAINBENCH_REMOTE_PASS. This automates wemix4's manual key read.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			c, err := deploy.LoadCluster(clusterPath)
			if err != nil {
				return err
			}
			cr, err := deploy.LoadCredentials(credsPath)
			if err != nil {
				return err
			}
			hostKey, err := remote.ResolveHostKeyCallback(os.Getenv)
			if err != nil {
				return err
			}

			targets := c.Validators() // validators carry the BLS keys
			if server > 0 {
				s, ok := c.ServerByIndex(server)
				if !ok {
					return fmt.Errorf("no server with index %d", server)
				}
				targets = []deploy.Server{s}
			}
			if len(targets) == 0 {
				return fmt.Errorf("no target servers (no wbft_bp validators in the cluster)")
			}

			out := cmd.OutOrStdout()
			var infos []deploy.NodeKeyInfo
			for _, s := range targets {
				info, err := deploy.ReadServerKeys(cmd.Context(), c, cr, hostKey, s, keystoreDir, os.Getenv)
				if err != nil {
					return err
				}
				infos = append(infos, info)
				fmt.Fprintf(out, "server %d (%s): addr=%s bls=%s\n", info.Server, s.Host, info.Address, short(info.BLSPubKey))
			}

			frag := deploy.FormatAccountsFragment(infos)
			if accountsOut != "" {
				if err := os.WriteFile(accountsOut, []byte(frag), 0o600); err != nil {
					return err
				}
				fmt.Fprintf(out, "wrote accounts fragment -> %s\n", accountsOut)
				return nil
			}
			fmt.Fprintf(out, "\n%s", frag)
			return nil
		},
	}
	cmd.Flags().StringVar(&clusterPath, "cluster", "", "cluster config file (cluster.yaml)")
	cmd.Flags().StringVar(&credsPath, "credentials", "", "SSH credentials file (or use CHAINBENCH_REMOTE_USER / CHAINBENCH_REMOTE_PASS)")
	cmd.Flags().IntVar(&server, "server", 0, "read only this server index (default: all wbft_bp validators)")
	cmd.Flags().StringVar(&keystoreDir, "keystore-dir", "", "local dir to pull keystores into (empty: skip the keystore pull)")
	cmd.Flags().StringVar(&accountsOut, "accounts-out", "", "write the accounts fragment here (default: stdout)")
	_ = cmd.MarkFlagRequired("cluster")
	return cmd
}

func short(s string) string {
	if len(s) > 14 {
		return s[:14] + "…"
	}
	return s
}
