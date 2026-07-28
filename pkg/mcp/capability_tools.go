package mcp

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/0xmhha/chainbench/pkg/mcp/capability"
)

// RegisterCapabilities folds the server's built-in flat tools into the
// capability catalog (keeping their names) and adds the capability surface:
// generated tools for handler-backed capabilities (named
// chainbench.<version>.<chain>.<name>) plus a chainbench.capabilities discovery
// tool. Only registered capabilities are exposed, so discovery reflects exactly
// what is supported and grows as projects register more.
func (s *Server) RegisterCapabilities() {
	// 1. Catalog the pre-existing flat tools as common capabilities (discovery
	//    only — they keep their established chainbench_* names).
	for _, name := range s.order {
		if !strings.HasPrefix(name, "chainbench_") {
			continue
		}
		t := s.tools[name]
		capName := strings.TrimPrefix(name, "chainbench_")
		capability.RegisterFlat("v1", capability.CommonChain, capName, name, t.Description, paramsFromSchema(t.InputSchema))
	}

	// 2. Generate a tool for each handler-backed capability (project-supplied);
	//    flat ones (Tool set) already have a tool, so skip them.
	for _, c := range capability.All() {
		if c.Tool != "" || c.Handler == nil {
			continue
		}
		s.Register(capabilityTool(c))
	}

	// 3. Discovery.
	s.Register(capabilitiesDiscoveryTool())
}

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
		Name:        c.ToolName(),
		Description: fmt.Sprintf("[%s] %s", scope, c.Summary),
		InputSchema: schema,
		Handler:     func(ctx context.Context, args map[string]any) (string, error) { return h(ctx, args) },
	}
}

// capabilitiesDiscoveryTool returns the version -> chain -> capability tree so
// an agent can see what is available (optionally for one chain), with each
// capability's invocation name.
func capabilitiesDiscoveryTool() Tool {
	return Tool{
		Name:        "chainbench.capabilities",
		Description: "List the capabilities chainbench exposes, grouped by version and chain, with the tool name to call each. Args: chain (optional; returns common + that chain's features).",
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
			return formatCapabilities(caps, chain), nil
		},
	}
}

// formatCapabilities renders the capability tree (shared shape with the CLI).
func formatCapabilities(caps []capability.Capability, chain string) string {
	if len(caps) == 0 {
		return "no capabilities registered"
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
		fmt.Fprintf(&b, "  %s\n      %s\n", c.ToolName(), c.Summary)
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
	return b.String()
}

// paramsFromSchema converts a tool's JSON input schema into capability Params.
func paramsFromSchema(schema map[string]any) []capability.Param {
	props, _ := schema["properties"].(map[string]any)
	req := map[string]bool{}
	switch rs := schema["required"].(type) {
	case []string:
		for _, r := range rs {
			req[r] = true
		}
	case []any:
		for _, r := range rs {
			if s, ok := r.(string); ok {
				req[s] = true
			}
		}
	}
	out := make([]capability.Param, 0, len(props))
	for k, v := range props {
		typ, desc := "string", ""
		if m, ok := v.(map[string]any); ok {
			if t, ok := m["type"].(string); ok {
				typ = t
			}
			if d, ok := m["description"].(string); ok {
				desc = d
			}
		}
		out = append(out, capability.Param{Name: k, Type: typ, Desc: desc, Required: req[k]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
