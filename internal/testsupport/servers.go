// Package testsupport holds cross-package test gates: helpers a _test.go in one
// package shares with a _test.go in another, which a package-local test file
// cannot. It carries no product code.
package testsupport

import (
	"os"
	"testing"
)

// EnvDockerServers names the generated docker-server directory a gated live
// test runs against. It is a TEST GATE, not a product setting: unset means
// "no virtual servers here, skip", which is how CI stays green without them.
const EnvDockerServers = "CHAINBENCH_DOCKER_SERVERS"

// ServersBuildDir returns the docker servers' build directory, skipping the
// test when the gate is unset. Named once so no suite spells the variable
// itself or invents its own skip message.
func ServersBuildDir(t *testing.T) string {
	t.Helper()
	build := os.Getenv(EnvDockerServers)
	if build == "" {
		t.Skip("set " + EnvDockerServers + "=<repo>/env/docker/build with the docker servers running (env/docker/gen-env.sh)")
	}
	return build
}
