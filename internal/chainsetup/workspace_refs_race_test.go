package chainsetup

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/0xmhha/chainbench/internal/resource"
)

func TestWorkspaceConfigConcurrentReadersShareInitialization(t *testing.T) {
	for _, configured := range []bool{false, true} {
		w := &Workspace{}
		if configured {
			w.state.WorkspaceConfig = filepath.Join(t.TempDir(), "workspace.yaml")
			data := "version: 1\ndataRoot: /owned/root\npaths: {binaries: binaries, configs: configs, genesis: genesis, keystore: keystore, keyrings: keyrings, nodes: nodes, runtime: runtime, logs: logs}\ncontrol: {artifactRoot: out}\ninputs: {mode: generated}\nexecution: {chain: fresh}\n"
			if err := os.WriteFile(w.state.WorkspaceConfig, []byte(data), 0600); err != nil {
				t.Fatal(err)
			}
		}
		start := make(chan struct{})
		results := make(chan *resource.WorkspaceConfig, 64)
		var wg sync.WaitGroup
		for i := 0; i < cap(results); i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				c, err := w.wc()
				if err != nil {
					t.Errorf("load: %v", err)
				}
				results <- c
			}()
		}
		close(start)
		wg.Wait()
		close(results)
		var expected *resource.WorkspaceConfig
		for c := range results {
			if configured && (c == nil || c.DataRoot != "/owned/root") {
				t.Fatal("concurrent reader observed an incomplete configuration", c)
			}
			if expected == nil {
				expected = c
			}
			if c != expected {
				t.Fatal("readers received different configuration snapshots")
			}
		}
	}
}

func TestWorkspaceConfigLoadErrorRemainsARefusal(t *testing.T) {
	w := &Workspace{state: State{WorkspaceConfig: filepath.Join(t.TempDir(), "missing.yaml")}}
	for i := 0; i < 2; i++ {
		if config, err := w.wc(); err == nil || config != nil {
			t.Fatal("failed load became an unconfigured workspace", config, err)
		}
	}
}
