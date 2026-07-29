package main

import (
	"fmt"
	"os"

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
	c.AddCommand(newRemoteKeysCmd())
	return c
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
