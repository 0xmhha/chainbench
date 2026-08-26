package testkit

import (
	"os"
	"testing"
)

// EnvDockerFleet names the generated docker-server directory a gated live
// test runs against. It is a TEST GATE, not a product setting: unset means
// "no virtual servers here, skip", which is how CI stays green without them.
const EnvDockerFleet = "CHAINBENCH_DOCKER_FLEET"

// FleetBuildDir returns the docker servers' build directory, skipping the
// test when the gate is unset. Named once so no suite spells the variable
// itself or invents its own skip message.
func FleetBuildDir(t *testing.T) string {
	t.Helper()
	build := os.Getenv(EnvDockerFleet)
	if build == "" {
		t.Skip("set " + EnvDockerFleet + "=<repo>/env/docker/build with the docker servers running (env/docker/gen-env.sh)")
	}
	return build
}
