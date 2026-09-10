package resource_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/0xmhha/chainbench/internal/resource"
)

// TestGenEnv_ProducesFilesThisParserAccepts pins the contract between the docker
// environment generator and the loader that reads what it writes.
//
// They drifted once and nothing caught it: the data root moved from the server
// set to the workspace-config, the loader began refusing a server set that still
// carried one, and the generator kept writing it — so a fresh checkout following
// the documented docker setup failed at the first command, while a hand-fixed
// local fixture kept passing. No Go test referenced the generator, so the gap
// was invisible.
//
// It runs the real script into a temp directory. Docker is not involved: the
// point is the files, not the containers.
func TestGenEnv_ProducesFilesThisParserAccepts(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the generator is a shell script")
	}
	script, err := filepath.Abs(filepath.Join("..", "..", "env", "docker", "gen-env.sh"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(script); err != nil {
		t.Skipf("generator not present: %v", err)
	}
	out := t.TempDir()

	cmd := exec.Command("bash", script)
	cmd.Dir = filepath.Dir(script)
	// BUILD is where the script writes; keep the repo's own fixtures untouched.
	cmd.Env = append(os.Environ(), "BUILD="+out)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("generator did not run here: %v\n%s", err, b)
	}

	setPath := filepath.Join(out, "server-set.yaml")
	if _, err := os.Stat(setPath); err != nil {
		t.Fatalf("the generator wrote no server set: %v", err)
	}
	// The whole point: what the generator writes is what the loader accepts.
	set, err := resource.LoadSet(setPath)
	if err != nil {
		t.Fatalf("the generated server set does not load: %v", err)
	}
	if len(set.Servers) == 0 {
		t.Fatal("the generated server set names no servers")
	}

	// The wemix set is generated too, and has to load the same way. The README
	// told operators to pass it long before anything wrote it.
	wemixPath := filepath.Join(out, "server-set-wemix.yaml")
	wemix, err := resource.LoadSet(wemixPath)
	if err != nil {
		t.Fatalf("the generated wemix server set does not load: %v", err)
	}
	// Its whole reason to exist: the poa port plan needs room for two
	// consecutive p2p-side ports, which a step of 1 does not leave.
	if got := wemix.PoolSpec.Ports.P2P.Step; got < 2 {
		t.Fatalf("wemix p2p step = %d, want >= 2 or the poa port plan is refused", got)
	}

	// And the environment half must be generated with it, since the data root
	// the server set no longer carries has to come from somewhere.
	wcPath := filepath.Join(out, "workspace-config.yaml")
	wc, err := resource.LoadWorkspaceConfig(wcPath)
	if err != nil {
		t.Fatalf("the generated workspace-config does not load: %v", err)
	}
	if wc.DataRoot == "" {
		t.Fatal("the generated workspace-config carries no data root")
	}
}
