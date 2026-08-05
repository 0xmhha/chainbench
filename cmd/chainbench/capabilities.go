package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/0xmhha/chainbench/internal/core/capability"
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
				fmt.Fprintf(out, "  %s\n      %s\n", c.ToolName(), c.Summary)
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
	cmd.AddCommand(newCapabilityCallCmd())
	return cmd
}

// newCapabilityCallCmd invokes a handler-backed capability by its address.
func newCapabilityCallCmd() *cobra.Command {
	var argPairs []string
	cmd := &cobra.Command{
		Use:   "call <version.chain.name>",
		Short: "Invoke a capability by address, e.g. v1.stablenet.governance.propose_mint",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			address := args[0]
			c, ok := capability.Lookup(address)
			if !ok {
				// Distinguish a built-in flat tool from an unknown address.
				if d, cataloged := capability.Get(address); cataloged && d.Tool != "" {
					return fmt.Errorf("%q is the built-in tool %q; call it via its own command (e.g. `chainbench %s`)",
						address, d.Tool, strings.TrimPrefix(strings.ReplaceAll(d.Name, ".", " "), ""))
				}
				return fmt.Errorf("unknown capability %q; run `chainbench capabilities` to list", address)
			}
			callArgs, err := parseArgPairs(argPairs)
			if err != nil {
				return err
			}
			out, err := c.Handler(cmd.Context(), callArgs)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), out)
			return nil
		},
	}
	cmd.Flags().StringArrayVar(&argPairs, "arg", nil, "capability argument as name=value (repeatable)")
	return cmd
}

// parseArgPairs turns "name=value" flags into a capability argument map.
func parseArgPairs(pairs []string) (map[string]any, error) {
	out := map[string]any{}
	for _, p := range pairs {
		k, v, ok := strings.Cut(p, "=")
		if !ok || k == "" {
			return nil, fmt.Errorf("bad --arg %q (expected name=value)", p)
		}
		out[k] = v
	}
	return out, nil
}
