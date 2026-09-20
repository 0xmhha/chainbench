package app

import (
	"fmt"
	"os/exec"
)

// ResolveBinary finds a node binary on THIS machine.
//
// An explicit path wins; otherwise the chain's own binary name is looked up on
// PATH. The failure names the binary that was looked for, because "not found"
// on its own leaves a caller guessing whether the chain or the flag decided it.
func ResolveBinary(explicit, chainBinary string) (string, error) {
	name := explicit
	if name == "" {
		name = chainBinary
	}
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("cannot find node binary %q: %w (build it or pass an explicit path)", name, err)
	}
	return path, nil
}
