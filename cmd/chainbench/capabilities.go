package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/pkg/mcp/capability"
)

func newCapabilitiesCmd() *cobra.Command {
	var chain string
	cmd := &cobra.Command{
		Use:   "capabilities",
		Short: "List the capabilities chainbench exposes, grouped by version and chain",
		RunE: func(cmd *cobra.Command, _ []string) error {
			var caps []capability.Capability
			if chain != "" {
				caps = capability.For(chain)
			} else {
				caps = capability.All()
			}
			out := cmd.OutOrStdout()
			if len(caps) == 0 {
				fmt.Fprintln(out, "no capabilities registered")
				return nil
			}
			if chain != "" {
				fmt.Fprintf(out, "capabilities for chain %q (common + %s):\n", chain, chain)
			} else {
				fmt.Fprintln(out, "all registered capabilities:")
			}
			lastGroup := ""
			for _, c := range caps {
				group := c.Version + "." + c.Chain
				if group != lastGroup {
					fmt.Fprintf(out, "\n[%s]\n", group)
					lastGroup = group
				}
				fmt.Fprintf(out, "  %s\n      %s\n", c.Address(), c.Summary)
				if len(c.Params) > 0 {
					parts := make([]string, len(c.Params))
					for i, p := range c.Params {
						req := ""
						if p.Required {
							req = "*"
						}
						parts[i] = p.Name + req
					}
					fmt.Fprintf(out, "      params: %s\n", strings.Join(parts, ", "))
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&chain, "chain", "", "scope to a chain (shows common + that chain's capabilities)")
	return cmd
}
