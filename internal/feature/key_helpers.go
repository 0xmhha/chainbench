package feature

import (
	"context"

	"github.com/0xmhha/chainbench/internal/app"
)

type saveKeyInput struct {
	Key     app.PrivateKey `json:"key"`
	Options app.SaveKeyIn  `json:"options"`
}

type deriveIdentityInput struct {
	Chain string         `cb:"chain,required" help:"Chain determining the derivation family"`
	Key   app.PrivateKey `json:"key"`
}

type validatorSetInput struct {
	Chain   string `cb:"chain,required" help:"Chain identifier"`
	KeysDir string `cb:"keys,required" help:"Validator key set directory"`
}

type generatedSetResult struct {
	Set      app.GenerateSetOut `json:"set"`
	Progress []string           `json:"progress"`
}

func init() {
	Register(Registration{Name: "key.save", Stage: StageCompose, Summary: "Save private key material in the selected format"}, func(ctx context.Context, deps app.Deps, in saveKeyInput) (string, error) {
		return app.SaveKey(ctx, deps, in.Key, in.Options)
	})
	Register(Registration{Name: "key.generate", Stage: StageCompose, Summary: "Generate private key material using system entropy"}, func(ctx context.Context, deps app.Deps, _ emptyInput) (app.PrivateKey, error) {
		return app.GenerateKey(ctx, deps)
	})
	Register(Registration{Name: "identity.derive", Stage: StageCompose, Summary: "Derive account and family-specific BLS identity from a private key"}, func(_ context.Context, deps app.Deps, in deriveIdentityInput) (app.IdentityOut, error) {
		return app.DeriveIdentity(deps, in.Chain, in.Key)
	})
	Register(Registration{Name: "validator-set.read", Stage: StageReport, ReadOnly: true, Summary: "Read the public validator declaration from a key set"}, func(_ context.Context, deps app.Deps, in validatorSetInput) (app.ValidatorSetOut, error) {
		return app.ValidatorSetOf(deps, in.Chain, in.KeysDir)
	})
	Register(Registration{Name: "key-set.generate", Stage: StageCompose, Summary: "Generate a preset key set and return its progress messages"}, func(_ context.Context, deps app.Deps, in app.GenerateSetIn) (generatedSetResult, error) {
		var progress []string
		set, err := app.GenerateSet(deps, in, func(message string) { progress = append(progress, message) })
		return generatedSetResult{Set: set, Progress: progress}, err
	})
}
