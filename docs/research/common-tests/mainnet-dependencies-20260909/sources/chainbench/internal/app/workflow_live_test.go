package app_test

// The whole-workflow gate (worklist V6.2): one call takes DSL inputs, sets a
// chain up on a docker server, runs the specs against it, collects the
// session, and tears the network down.
//
//	CHAINBENCH_DOCKER_SERVERS=$PWD/env/docker/build go test ./internal/app -run Live_RunSuite -v
//
// The server set must carry the chain binary at /data/chainbench/bin/gstable.

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/testsupport"

	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/resource"
)

func TestLive_RunSuiteSetsUpRunsAndReports(t *testing.T) {
	build := testsupport.ServersBuildDir(t)

	spec := filepath.Join("..", "..", "tests", "specs", "consensus", "wbft-seals-quorum.json")
	if _, err := os.Stat(spec); err != nil {
		t.Fatalf("spec fixture missing: %v", err)
	}

	out, err := app.RunSuite(context.Background(), app.Deps{}, app.RunSuiteIn{
		SpecPaths:  []string{spec},
		DataDir:    t.TempDir(),
		Chain:      "stablenet",
		Binary:     "/data/chainbench/bin/gstable",
		Validators: 4,
		Server:     resource.ServerRef{SetPath: filepath.Join(build, "server-set.yaml"), Name: "server1"},
		Docker:     true,
		KeysDir:    filepath.Join("..", "..", "keys", "preset"),
		Caps:       []string{"consensus"},
		WaitBlocks: 2,
	})
	if err != nil {
		t.Fatalf("run suite: %v (setup steps: %v)", err, out.SetupSteps)
	}
	if len(out.SetupSteps) == 0 {
		t.Error("no setup steps recorded")
	}
	if len(out.Endpoints) != 4 {
		t.Errorf("endpoints = %v, want 4", out.Endpoints)
	}
	if out.SessionRoot == "" {
		t.Fatal("no session root — the tests never ran")
	}
	if out.Summary.Summary.Pass != 1 || out.Summary.Failed() {
		t.Errorf("summary = %+v, want the one spec passed", out.Summary.Summary)
	}
	if !out.Stopped {
		t.Error("the network was not torn down")
	}
}
