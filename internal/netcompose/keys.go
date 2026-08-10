package netcompose

import (
	"fmt"

	"github.com/0xmhha/chainbench/internal/core/keys"
)

// KeysOpts customizes the keys step.
type KeysOpts struct {
	// KeysDir is the preset directory to use (default keys/preset).
	KeysDir string
	// Validators bounds the active validator count; <=0 uses the whole preset.
	Validators int
}

// KeysResult reports the resolved identity set.
type KeysResult struct {
	KeysDir    string   `json:"keysDir"`
	Nodes      int      `json:"nodes"`
	Validators int      `json:"validators"`
	Addresses  []string `json:"addresses"`
}

// Keys resolves the preset key set (node identities + validator set) and records
// it on the workspace. It reads the committed preset rather than generating keys
// (wbft-family validator sets and BLS keys are baked into the preset); use
// `keys generate` to make a larger preset first. Resolution is target-agnostic —
// identities are read locally; a remote target ships them at the provision step.
func (w *Workspace) Keys(opts KeysOpts) (KeysResult, error) {
	if w.state.Chain == "" {
		return KeysResult{}, fmt.Errorf("netcompose: run `net new --chain <id>` first")
	}
	keysDir := opts.KeysDir
	if keysDir == "" {
		keysDir = "keys/preset"
	}
	preset, err := keys.LoadPreset(keysDir)
	if err != nil {
		return KeysResult{}, err
	}

	validators := len(preset.Validators)
	if opts.Validators > 0 && opts.Validators < validators {
		validators = opts.Validators
	}
	res := KeysResult{
		KeysDir:    keysDir,
		Nodes:      len(preset.Nodes),
		Validators: validators,
		Addresses:  append([]string(nil), preset.Validators[:validators]...),
	}

	w.state.KeysDir = keysDir
	w.state.Validators = validators
	w.markStep("keys", fmt.Sprintf("%d node identities, %d validator(s) from %s", res.Nodes, res.Validators, keysDir))
	return res, nil
}
