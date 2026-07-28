package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/0xmhha/chainbench/pkg/mcp/capability"
)

// RegisterCapabilities adds the capability surface to a server: one tool per
// registered capability (named by its hierarchical address) plus a
// "chainbench.capabilities" discovery tool. Only capabilities that projects
// have registered (catalog entry + bound handler) are exposed, so the tool set
// reflects exactly what is supported — and grows as chains/projects register
// more (docs: layered capability model).
func (s *Server) RegisterCapabilities() {
	s.Register(capabilitiesDiscoveryTool())
	for _, c := range capability.All() {
		s.Register(capabilityTool(c))
	}
}

// toolName encodes a capability address as an MCP tool name: dots are kept as
// the hierarchy separator ("chainbench.v1.stablenet.governance.propose_mint").
func toolName(addr string) string { return "chainbench." + addr }

func capabilityTool(c capability.Capability) Tool {
	props := map[string]any{}
	var required []string
	for _, p := range c.Params {
		typ := p.Type
		if typ == "" {
			typ = "string"
		}
		entry := map[string]any{"type": typ}
		if p.Desc != "" {
			entry["description"] = p.Desc
		}
		props[p.Name] = entry
		if p.Required {
			required = append(required, p.Name)
		}
	}
	schema := map[string]any{"type": "object", "properties": props}
	if len(required) > 0 {
		schema["required"] = required
	}
	scope := "common (all chains)"
	if c.Chain != capability.CommonChain {
		scope = "chain: " + c.Chain
	}
	h := c.Handler
	return Tool{
		Name:        toolName(c.Address()),
		Description: fmt.Sprintf("[%s] %s", scope, c.Summary),
		InputSchema: schema,
		Handler:     func(ctx context.Context, args map[string]any) (string, error) { return h(ctx, args) },
	}
}

// capabilitiesDiscoveryTool returns the version -> chain -> capability tree so
// an agent can see what is available (optionally for one chain).
func capabilitiesDiscoveryTool() Tool {
	return Tool{
		Name:        "chainbench.capabilities",
		Description: "List the capabilities chainbench exposes, grouped by version and chain. Args: chain (optional; returns common + that chain's features).",
		InputSchema: map[string]any{
			"type":       "object",
			"properties": map[string]any{"chain": map[string]any{"type": "string", "description": "chain id to scope to"}},
		},
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			chain := argString(args, "chain", "")
			var caps []capability.Capability
			if chain != "" {
				caps = capability.For(chain)
			} else {
				caps = capability.All()
			}
			if len(caps) == 0 {
				return "no capabilities registered", nil
			}
			var b strings.Builder
			if chain != "" {
				fmt.Fprintf(&b, "capabilities for chain %q (common + %s):\n", chain, chain)
			} else {
				b.WriteString("all registered capabilities:\n")
			}
			lastGroup := ""
			for _, c := range caps {
				group := c.Version + "." + c.Chain
				if group != lastGroup {
					fmt.Fprintf(&b, "\n[%s]\n", group)
					lastGroup = group
				}
				fmt.Fprintf(&b, "  %s\n      %s\n", toolName(c.Address()), c.Summary)
				if len(c.Params) > 0 {
					parts := make([]string, len(c.Params))
					for i, p := range c.Params {
						req := ""
						if p.Required {
							req = "*"
						}
						parts[i] = p.Name + req
					}
					fmt.Fprintf(&b, "      params: %s\n", strings.Join(parts, ", "))
				}
			}
			return b.String(), nil
		},
	}
}
