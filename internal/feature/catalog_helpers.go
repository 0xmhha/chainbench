package feature

import (
	"context"

	"github.com/0xmhha/chainbench/internal/app"
)

type emptyInput struct{}

type chainInput struct {
	Chain string `cb:"chain,required" help:"Registered chain identifier"`
}

type addressInput struct {
	Address string `cb:"address,required" help:"System contract address"`
}

type capabilityNameInput struct {
	Name string `cb:"name,required" help:"Capability version.chain.name spelling"`
}

type capabilityRegistrationInput struct {
	Version     string      `cb:"version,required" help:"Capability version"`
	Chain       string      `cb:"chain,required" help:"Chain or common pseudo-chain"`
	Group       string      `cb:"group,required" help:"Capability group"`
	Name        string      `cb:"name,required" help:"Capability name"`
	Description string      `cb:"description" help:"Capability description"`
	Params      []app.Param `json:"params"`
}

type lookupResult[T any] struct {
	Value T    `json:"value"`
	Found bool `json:"found"`
}

func init() {
	Register(Registration{Name: "workspace.default", Stage: StageReport, ReadOnly: true, Summary: "Resolve the operator's default workspace directory"}, func(_ context.Context, deps app.Deps, _ emptyInput) (string, error) {
		return app.DefaultWorkspaceDir(deps)
	})
	Register(Registration{Name: "chain.list", Stage: StageReport, ReadOnly: true, Summary: "List registered chain identifiers"}, func(_ context.Context, deps app.Deps, _ emptyInput) ([]string, error) {
		return app.Chains(deps), nil
	})
	Register(Registration{Name: "chain.describe", Stage: StageReport, ReadOnly: true, Summary: "Describe a registered chain manifest"}, func(_ context.Context, deps app.Deps, in chainInput) (any, error) {
		plugin, err := app.Chain(deps, in.Chain)
		if err != nil {
			return nil, err
		}
		return plugin.Manifest(), nil
	})
	Register(Registration{Name: "capability.list", Stage: StageReport, ReadOnly: true, Summary: "List capabilities advertised by every chain"}, func(_ context.Context, deps app.Deps, _ emptyInput) ([]app.Descriptor, error) {
		capabilities := app.Capabilities(deps)
		out := make([]app.Descriptor, len(capabilities))
		for i, capability := range capabilities {
			out[i] = capability.Descriptor
		}
		return out, nil
	})
	Register(Registration{Name: "capability.for-chain", Stage: StageReport, ReadOnly: true, Summary: "List capabilities advertised by one chain"}, func(_ context.Context, deps app.Deps, in chainInput) ([]app.Descriptor, error) {
		capabilities := app.CapabilitiesFor(deps, in.Chain)
		out := make([]app.Descriptor, len(capabilities))
		for i, capability := range capabilities {
			out[i] = capability.Descriptor
		}
		return out, nil
	})
	Register(Registration{Name: "capability.address", Stage: StageReport, ReadOnly: true, Summary: "Find the system contract descriptor for an address"}, func(_ context.Context, deps app.Deps, in addressInput) (lookupResult[app.Descriptor], error) {
		value, found := app.CapabilityByAddress(deps, in.Address)
		return lookupResult[app.Descriptor]{Value: value, Found: found}, nil
	})
	Register(Registration{Name: "capability.lookup", Stage: StageReport, ReadOnly: true, Summary: "Find a capability by its full registry spelling"}, func(_ context.Context, deps app.Deps, in capabilityNameInput) (lookupResult[app.Descriptor], error) {
		value, found := app.CapabilityByName(deps, in.Name)
		return lookupResult[app.Descriptor]{Value: value.Descriptor, Found: found}, nil
	})
	Register(Registration{Name: "capability.register", Stage: StageCompose, Summary: "Register a surface-provided capability"}, func(_ context.Context, deps app.Deps, in capabilityRegistrationInput) (struct{}, error) {
		app.RegisterCapability(deps, in.Version, in.Chain, in.Group, in.Name, in.Description, in.Params)
		return struct{}{}, nil
	})
}
