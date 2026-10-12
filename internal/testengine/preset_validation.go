package testengine

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/nodeconfig"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/dsl"
)

// ValidateChainPreset checks a composition declaration without creating files,
// resolving private credentials, or contacting a target. It uses the same
// lowering, topology, dialect and launch assembly as the execution engine.
// Target/binary access and resource allocation remain execution preconditions.
func ValidateChainPreset(raw []byte) error {
	if !json.Valid(raw) {
		return fmt.Errorf("preset: expected one JSON document")
	}
	env, err := dsl.ParseChainPreset(raw)
	if err != nil {
		return err
	}
	if env.SchemaVersion != "2" {
		return fmt.Errorf("preset: schemaVersion must be 2")
	}
	plugin, err := registry.Get(env.Chain)
	if err != nil {
		return err
	}
	caseRaw, err := json.Marshal(map[string]any{"schemaVersion": "2", "kind": "case", "id": "preset-validation", "chainPreset": json.RawMessage(raw), "steps": []map[string]any{{"expect": "blockNumber", "is": 0}}})
	if err != nil {
		return err
	}
	spec, err := dsl.Parse(caseRaw)
	if err != nil {
		return err
	}
	if err := Precheck([]dsl.Spec{spec}); err != nil {
		return err
	}
	if env.Attach != nil {
		return nil
	}
	if env.Blueprint != "" {
		return fmt.Errorf("preset: blueprint requires file validation before use")
	}
	topo, _, _, err := inlineTopologyOf(env.Chain, spec.Topology, spec.Chain.Binaries)
	if err != nil {
		return err
	}
	if topo == nil {
		bp, en, pn, syncMode, auto, err := topologyOf(spec.Topology)
		if err != nil {
			return err
		}
		if auto {
			return fmt.Errorf("preset: topology.bp=max requires server allocation validation")
		}
		if _, declared := spec.Topology["bp"]; !declared {
			bp = suiteDefaultValidators
		}
		// Validate counts without allocating one entry per requested node.
		// Assembly needs only one representative per role and every node with
		// an explicit override. Actual allocation belongs to execution preflight.
		if bp > math.MaxInt-en || bp+en > math.MaxInt-pn {
			return fmt.Errorf("topology: total node count overflows")
		}
		total := bp + en + pn
		sample := node.Topology{Chain: env.Chain}
		topo = &node.Topology{Chain: env.Chain}
		indices := map[int]bool{}
		offset := 0
		for _, group := range []struct {
			role  node.Role
			count int
		}{{node.RoleBP, bp}, {node.RoleEN, en}, {node.RolePN, pn}} {
			if group.count > 0 {
				sample.Nodes = append(sample.Nodes, node.Entry{Index: len(sample.Nodes) + 1, Role: string(group.role), SyncMode: syncMode})
				indices[offset+1] = true
			}
			offset += group.count
		}
		if err := sample.Validate(); err != nil {
			return err
		}
		for _, scopes := range []map[string]map[string]any{env.Launch, env.Config} {
			for scope := range scopes {
				if index := node.ScopeIndex(scope); index > 0 {
					if index > total {
						return fmt.Errorf("preset: scope %s names a node outside this topology", scope)
					}
					indices[index] = true
				}
			}
		}
		sorted := make([]int, 0, len(indices))
		for index := range indices {
			sorted = append(sorted, index)
		}
		sort.Ints(sorted)
		for _, index := range sorted {
			role := node.RoleBP
			if index > bp+en {
				role = node.RolePN
			} else if index > bp {
				role = node.RoleEN
			}
			topo.Nodes = append(topo.Nodes, node.Entry{Index: index, Role: string(role), SyncMode: syncMode})
		}
	}
	for _, entry := range topo.Nodes {
		if !plugin.Family().SupportsRole(entry.NodeRole()) {
			return fmt.Errorf("preset: unsupported role %s", entry.Role)
		}
	}
	availableScopes := map[string]bool{}
	for _, entry := range topo.Nodes {
		for _, scope := range node.ScopeFor(entry.NodeRole(), entry.Index) {
			availableScopes[scope] = true
		}
	}
	for scope, knobs := range env.Launch {
		if !availableScopes[scope] {
			return fmt.Errorf("launch.%s: scope does not select any node", scope)
		}
		if !node.ValidScope(scope) {
			return fmt.Errorf("launch.%s: invalid scope", scope)
		}
		for key, value := range knobs {
			if nodeconfig.IsPerNode(nodeconfig.OptionKey(key)) && node.ScopeIndex(scope) == 0 {
				return fmt.Errorf("launch.%s.%s: requires one node", scope, key)
			}
			switch v := value.(type) {
			case bool:
				d, err := nodeconfig.DialectFor(plugin.Manifest().Dialect)
				if err != nil {
					return err
				}
				if !d.IsBool(nodeconfig.OptionKey(key)) || !v {
					return fmt.Errorf("launch.%s.%s: boolean overrides must enable a boolean knob", scope, key)
				}
			case string, float64:
				d, err := nodeconfig.DialectFor(plugin.Manifest().Dialect)
				if err != nil {
					return err
				}
				if d.IsBool(nodeconfig.OptionKey(key)) {
					return fmt.Errorf("launch.%s.%s: expected boolean true", scope, key)
				}
			default:
				return fmt.Errorf("launch.%s.%s: expected scalar value", scope, key)
			}
		}
	}
	for scope := range env.Config {
		if !availableScopes[scope] {
			return fmt.Errorf("config.%s: scope does not select any node", scope)
		}
		if !node.ValidScope(scope) {
			return fmt.Errorf("config.%s: invalid scope", scope)
		}
	}
	for _, entry := range topo.Nodes {
		selected := plugin
		if entry.Binary != "" {
			if chain := spec.Chain.BinaryChains[entry.Binary]; chain != "" {
				selected, err = registry.Get(chain)
				if err != nil {
					return err
				}
			}
		}
		s := nodeconfig.Spec{Chain: nodeconfig.ChainOf(selected, entry.NodeRole()), Network: nodeconfig.NetworkOf(plugin, 0), Role: entry.NodeRole(), DataDir: "preset-validation", Ports: node.Endpoints{P2P: 30301, HTTP: 8545, WS: 8546}, SyncMode: entry.SyncMode}
		var overrides []nodeconfig.Override
		for _, scope := range node.ScopeFor(entry.NodeRole(), entry.Index) {
			knobs := env.Launch[scope]
			keys := make([]string, 0, len(knobs))
			for k := range knobs {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				overrides = append(overrides, nodeconfig.Override{Key: nodeconfig.OptionKey(k), Value: fmt.Sprint(knobs[k]), Layer: nodeconfig.LayerEnv})
			}
			for k, v := range env.Config[scope] {
				text, ok := v.(string)
				if !ok {
					return fmt.Errorf("config.%s.%s: expected string", scope, k)
				}
				if err := nodeconfig.ApplyConfigOverride(&s, k, text); err != nil {
					return err
				}
			}
		}
		if _, err := nodeconfig.Argv(s, overrides...); err != nil {
			return err
		}
	}
	return nil
}
