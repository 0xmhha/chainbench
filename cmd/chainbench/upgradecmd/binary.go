package upgradecmd

import (
	"fmt"
	"os/exec"
)

// resolveBinary returns the executable path for a launch: the explicit path if
// given, otherwise the chain's binary looked up on PATH.
func resolveBinary(explicit, chainBinary string) (string, error) {
	name := explicit
	if name == "" {
		name = chainBinary
	}
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("cannot find node binary %q: %w (build it or pass --binary)", name, err)
	}
	return path, nil
}
