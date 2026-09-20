// Package networkcmd is the CLI mirror of the MCP named-network registry tools
// (chainbench_network_* and chainbench_remote_rpc). It gives an operator the
// same reach an agent has: attach to an already-running network by URL, list
// and inspect the saved networks, and call a method on one by name — the
// registry was MCP-only before (WA5). Every subcommand is flag decoding plus one
// internal/app call, the same function the MCP tool binds, so the two surfaces
// cannot drift.
package networkcmd

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/app"
)

// New returns the `network` command group.
func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "network",
		Short: "Attach to, list, inspect, and call already-running networks by name",
	}
	cmd.AddCommand(newAttachCmd(), newListCmd(), newInfoCmd(), newDetachCmd(), newPeersCmd(), newRPCCmd())
	return cmd
}

// stateDirFlag binds the shared --state-dir flag (where the registry lives).
func stateDirFlag(cmd *cobra.Command, dst *string) {
	cmd.Flags().StringVar(dst, "state-dir", "", "directory holding the attached-network registry")
}

func newAttachCmd() *cobra.Command {
	var stateDir, rpc, override string
	cmd := &cobra.Command{
		Use:   "attach <name>",
		Short: "Probe an RPC endpoint and save it as a named attached network",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, argv []string) error {
			name := argv[0]
			if rpc == "" || stateDir == "" {
				return fmt.Errorf("network attach needs --rpc and --state-dir")
			}
			if !app.IsValidNetworkName(name) {
				return fmt.Errorf("invalid network name %q (must match [a-z0-9][a-z0-9_-]* and not be 'local')", name)
			}
			res, err := app.DetectNetwork(cmd.Context(), app.Deps{}, app.DetectOptions{RPCURL: rpc, Override: override})
			if err != nil {
				return err
			}
			ns := app.NodeSet{
				Chain: res.ChainType, Network: name,
				Nodes:        []app.Node{{Index: 1, Role: app.RoleEN, Host: hostOf(rpc), RPCURL: rpc}},
				Capabilities: []string{"rpc"},
			}
			if err := app.AttachNetwork(app.Deps{}, stateDir, ns); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "attached %q: chain_type=%s chain_id=%d\n", name, res.ChainType, res.ChainID)
			return nil
		},
	}
	stateDirFlag(cmd, &stateDir)
	cmd.Flags().StringVar(&rpc, "rpc", "", "RPC endpoint of the running network")
	cmd.Flags().StringVar(&override, "override", "", "force the chain type instead of detecting it")
	return cmd
}

func newListCmd() *cobra.Command {
	var stateDir string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List saved attached networks",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if stateDir == "" {
				return fmt.Errorf("network list needs --state-dir")
			}
			nets, err := app.Networks(app.Deps{}, stateDir)
			if err != nil {
				return err
			}
			if len(nets) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no attached networks")
				return nil
			}
			for _, ns := range nets {
				ep := ""
				if len(ns.Nodes) > 0 {
					ep = ns.Nodes[0].RPCURL
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", ns.Network, ns.Chain, ep)
			}
			return nil
		},
	}
	stateDirFlag(cmd, &stateDir)
	return cmd
}

func newInfoCmd() *cobra.Command {
	var stateDir string
	cmd := &cobra.Command{
		Use:   "info <name>",
		Short: "Show a saved attached network's nodes",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, argv []string) error {
			if stateDir == "" {
				return fmt.Errorf("network info needs --state-dir")
			}
			ns, err := app.Network(app.Deps{}, stateDir, argv[0])
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "network=%s chain_type=%s nodes=%d\n", ns.Network, ns.Chain, len(ns.Nodes))
			for _, n := range ns.Nodes {
				authed := ""
				if len(n.Auth) > 0 {
					authed = " (auth)"
				}
				fmt.Fprintf(w, "  node%d %s %s%s\n", n.Index, n.Role, n.RPCURL, authed)
			}
			return nil
		},
	}
	stateDirFlag(cmd, &stateDir)
	return cmd
}

func newDetachCmd() *cobra.Command {
	var stateDir string
	cmd := &cobra.Command{
		Use:   "detach <name>",
		Short: "Remove a saved attached network",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, argv []string) error {
			if stateDir == "" {
				return fmt.Errorf("network detach needs --state-dir")
			}
			if err := app.DetachNetwork(app.Deps{}, stateDir, argv[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "detached %q\n", argv[0])
			return nil
		},
	}
	stateDirFlag(cmd, &stateDir)
	return cmd
}

func newPeersCmd() *cobra.Command {
	var rpc string
	cmd := &cobra.Command{
		Use:   "peers",
		Short: "Report an endpoint's peer count",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if rpc == "" {
				return fmt.Errorf("network peers needs --rpc")
			}
			n, err := app.PeersOf(cmd.Context(), app.Deps{}, rpc)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%d\n", n)
			return nil
		},
	}
	cmd.Flags().StringVar(&rpc, "rpc", "", "RPC endpoint")
	return cmd
}

func newRPCCmd() *cobra.Command {
	var stateDir, method string
	var node int
	cmd := &cobra.Command{
		Use:   "rpc <name>",
		Short: "Call a JSON-RPC method on a saved network's node, using its stored auth",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, argv []string) error {
			if stateDir == "" || method == "" {
				return fmt.Errorf("network rpc needs --state-dir and --method")
			}
			ns, err := app.Network(app.Deps{}, stateDir, argv[0])
			if err != nil {
				return err
			}
			n, ok := app.NodeAt(ns, node)
			if !ok {
				return fmt.Errorf("network %q has no usable node", argv[0])
			}
			out, err := app.CallOnNode(cmd.Context(), app.Deps{}, n, method)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), strings.TrimSpace(string(out)))
			return nil
		},
	}
	stateDirFlag(cmd, &stateDir)
	cmd.Flags().StringVar(&method, "method", "", "JSON-RPC method to call")
	cmd.Flags().IntVar(&node, "node", 0, "1-based node index (default: the first usable node)")
	return cmd
}

// hostOf returns the host of an RPC URL, or "" if it cannot be parsed.
func hostOf(rpc string) string {
	if u, err := url.Parse(rpc); err == nil {
		return u.Hostname()
	}
	return ""
}
