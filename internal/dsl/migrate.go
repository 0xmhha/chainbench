package dsl

import (
	"encoding/json"
	"fmt"
	"maps"
)

// MigrateV1 mechanically converts a v1 spec to the v2 grammar
// (dsl-v2-proposal §3.6). The conversion is a desugar inverse: parsing the
// result yields the same executable Spec (same lowered sequence, same env
// fields), which is the property the migrate tests pin.
func MigrateV1(raw []byte) ([]byte, error) {
	s, err := Parse(raw)
	if err != nil {
		return nil, err
	}
	if s.SchemaVersion != supportedSchemaVersion {
		return nil, fmt.Errorf("dsl: migrate: %s is not a v1 spec", s.ID)
	}

	env := map[string]any{
		"schemaVersion": schemaVersionV2,
		"kind":          KindChainPreset,
		"id":            s.ID + "-env",
		"chain":         s.Chain.Name,
	}
	if s.Chain.Binary != "" {
		env["binaries"] = map[string]string{"default": s.Chain.Binary}
	} else if len(s.Chain.Binaries) > 0 {
		env["binaries"] = s.Chain.Binaries
	}
	if s.Chain.Config != "" {
		return nil, fmt.Errorf("dsl: migrate: v1 chain.config file has no equivalent v2 declaration")
	}
	if len(s.Chain.GenesisOverlay) > 0 {
		env["genesis"] = map[string]any{"overlay": s.Chain.GenesisOverlay}
	}
	if len(s.Chain.BinaryChains) > 0 {
		refs := map[string]BinaryRefV2{}
		if s.Chain.Binary != "" {
			refs[BinaryDefault] = BinaryRefV2{Binary: s.Chain.Binary, Chain: s.Chain.BinaryChains[BinaryDefault]}
		} else {
			for name, binary := range s.Chain.Binaries {
				refs[name] = BinaryRefV2{Binary: binary, Chain: s.Chain.BinaryChains[name]}
			}
		}
		env["binaries"] = refs
	}
	if s.Chain.ManifestPath != "" {
		env["manifest"] = s.Chain.ManifestPath
	}
	if s.Chain.TemplatePath != "" {
		env["genesisTemplate"] = s.Chain.TemplatePath
	}
	if len(s.Chain.GenesisProvides) > 0 || s.Chain.GenesisHaltsAt != 0 || len(s.Chain.GenesisPerBinary) > 0 || s.Chain.GenesisExisting != "" {
		genesis, _ := env["genesis"].(map[string]any)
		if genesis == nil {
			genesis = map[string]any{}
		}
		if len(s.Chain.GenesisProvides) > 0 {
			genesis["provides"] = s.Chain.GenesisProvides
		}
		if s.Chain.GenesisHaltsAt != 0 {
			genesis["haltsAt"] = s.Chain.GenesisHaltsAt
		}
		if len(s.Chain.GenesisPerBinary) > 0 {
			sides := map[string]any{}
			for name, overlay := range s.Chain.GenesisPerBinary {
				sides[name] = map[string]any{"overlay": overlay}
			}
			genesis["perBinary"] = sides
		}
		if s.Chain.GenesisExisting != "" {
			genesis["mode"] = "existing"
			genesis["ref"] = s.Chain.GenesisExisting
		}
		env["genesis"] = genesis
	}
	if len(s.Topology) > 0 {
		env["topology"] = s.Topology
	}
	if len(s.Hardforks) > 0 {
		env["hardforks"] = s.Hardforks
	}
	if s.Placement != "" {
		env["target"] = s.Placement
	}
	if len(s.Requires) > 0 {
		env["capabilities"] = s.Requires
	}

	steps := make([]map[string]any, 0, len(s.Steps)+len(s.Assertions))
	for _, st := range s.Steps {
		name := ActionName(st)
		stmt := map[string]any{"do": name}
		maps.Copy(stmt, ArgsOf(st[name]))
		stmt["do"] = name // an arg named "do" must not clobber the head
		steps = append(steps, stmt)
	}
	for _, as := range s.Assertions {
		stmt := make(map[string]any, len(as)+1)
		for k, v := range as {
			switch k {
			case "assert":
				stmt["expect"] = v
			case "expected":
				stmt["is"] = v
			default:
				stmt[k] = v
			}
		}
		steps = append(steps, stmt)
	}

	out := map[string]any{
		"schemaVersion": schemaVersionV2,
		"kind":          KindCase,
		"id":            s.ID,
		"chainPreset":   env,
		"steps":         steps,
	}
	if s.ApplicableChains != "" {
		out["applicableChains"] = s.ApplicableChains
	}
	if len(s.Requires) > 0 {
		out["requires"] = s.Requires
	}
	if s.SkipsOn != nil {
		out["skipsOn"] = s.SkipsOn
	}
	if s.DefaultOn != "" {
		out["on"] = s.DefaultOn
	}
	if len(s.Timeouts) > 0 {
		out["timeouts"] = s.Timeouts
	}
	hooks := map[string]any{}
	if len(s.PreActions) > 0 {
		hooks["pre"] = hookStatements(s.PreActions)
	}
	if len(s.PostActions) > 0 {
		hooks["post"] = hookStatements(s.PostActions)
	}
	if len(hooks) > 0 {
		out["hooks"] = hooks
	}
	return json.MarshalIndent(out, "", "  ")
}

// hookStatements converts v1 action maps to v2 do statements.
func hookStatements(actions []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(actions))
	for _, a := range actions {
		name := ActionName(a)
		stmt := map[string]any{}
		maps.Copy(stmt, ArgsOf(a[name]))
		stmt["do"] = name
		out = append(out, stmt)
	}
	return out
}
