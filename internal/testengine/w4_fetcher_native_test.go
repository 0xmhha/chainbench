package testengine_test

import (
	"context"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

// This diagnostic deliberately passes only for the complete unresolved W4
// witness. Its PASS is never a CT-NODE-007 acceptance or a defect resolution.
func TestNativeWemixW4FetcherCollision(t *testing.T) {
	bin := os.Getenv("GWEMIX_BIN")
	if bin == "" || os.Getenv("CHAINBENCH_W4_PROOF") != "1" {
		t.Skip("set GWEMIX_BIN and CHAINBENCH_W4_PROOF=1 for unresolved W4 diagnostic")
	}
	root := failureRuntime(t)
	ctx, cancel := context.WithTimeout(t.Context(), 4*time.Minute)
	defer cancel()
	workspace := filepath.Join(root, "w")
	p := w4Witness{}
	proof := map[string]any{"scope": "Diagnostic only: latest pinned WEMIX4538, original CT-NODE007 remains failing at its exact ETH receive metric; real EN peers and twenty additional blocks with matching canonical hash/stateRoot while raw Prometheus/expvar stay zero. Actual binary LES-before-ETH initialization. No chain source edits, no replacement counter, no issue resolution, no full87/Linux/mainnet/final Web acceptance."}
	defer func() {
		proof["witness"] = p
		b, e := json.MarshalIndent(proof, "", "  ")
		if e != nil {
			t.Error(e)
			return
		}
		if e = os.WriteFile(filepath.Join(root, "reports/w4-fetcher-proof.json"), b, 0600); e != nil {
			t.Error(e)
		}
	}()
	info, err := buildinfo.ReadFile(bin)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" {
			p.Revision = s.Value
		}
		if s.Key == "vcs.modified" && s.Value != "false" {
			t.Fatal("WEMIX binary built from modified source")
		}
	}
	b, err := os.ReadFile(bin)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256(b)
	p.BinarySHA = hex.EncodeToString(hash[:])
	trace := exec.CommandContext(ctx, bin, "version")
	trace.Env = append(os.Environ(), "GODEBUG=inittrace=1")
	traceBytes, err := trace.CombinedOutput()
	p.InitTrace = string(traceBytes)
	if err != nil {
		t.Fatal("actual initialization trace", err)
	}
	casePath := filepath.Join(repoRoot(t), "tests/tc/common/node/CT-NODE-007-live-block-receive.json")
	raws, err := dsl.ReadFilesWithChainPreset([]string{casePath}, "wemix-bp4-en1")
	if err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(casePath); err != nil {
		t.Fatal(err)
	} else if err = os.WriteFile(filepath.Join(root, "case.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	for i, b := range raws {
		if err = os.WriteFile(filepath.Join(root, fmt.Sprintf("resolved-%d.json", i)), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	out, runErr := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{SpecContent: raws, DataDir: workspace, ArtifactRoot: filepath.Join(root, "reports"), Binary: bin, KeysDir: filepath.Join(repoRoot(t), "presets/keys"), ReadOnlyKeys: true, KeepUp: true, WaitBlocks: 2})
	proof["session"], proof["summary"] = out.SessionRoot, out.Summary
	if runErr != nil {
		proof["runError"] = runErr.Error()
	}
	if !out.Summary.Failed() || out.Summary.Summary.Skip != 0 {
		t.Fatal("not the unresolved strict case failure", runErr, out.Summary)
	}
	files, err := filepath.Glob(filepath.Join(out.SessionRoot, "tests/*/assert.json"))
	if err != nil {
		t.Fatal(err)
	}
	var failed []session.AssertResult
	for _, file := range files {
		b, e := os.ReadFile(file)
		if e != nil {
			t.Fatal(e)
		}
		var values []session.AssertResult
		if e = json.Unmarshal(b, &values); e != nil {
			t.Fatal(e)
		}
		for _, v := range values {
			if !v.Pass {
				failed = append(failed, v)
			}
		}
	}
	proof["failedAssertions"] = failed
	if len(failed) != 1 || failed[0].Assert != "metric" || failed[0].Provenance["name"] != "eth_fetcher_block_broadcasts_in" {
		t.Fatal("failure is not the exact W4 metric", failed)
	}
	status, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: workspace})
	if err != nil || len(status.Nodes.Nodes) != 5 {
		t.Fatal("not five live recorded nodes", err)
	}
	proof["nodes"] = status.Nodes.Nodes
	var producer, en node.Node
	for _, n := range status.Nodes.Nodes {
		if n.Index == 1 {
			producer = n
		}
		if n.Index == 5 {
			en = n
		}
	}
	if en.Host != "127.0.0.1" || en.RPCURL == "" || en.MetricsURL == "" || producer.RPCURL == "" {
		t.Fatal("missing owned loopback endpoints")
	}
	p.Index, p.PID, p.Role = en.Index, en.PID, string(en.Role)
	c, source := rpc.Dial(en.RPCURL), rpc.Dial(producer.RPCURL)
	for i := 0; i < 2; i++ {
		if i > 0 {
			deadline := time.Now().Add(90 * time.Second)
			for {
				h, e := c.BlockNumber(ctx)
				if e != nil {
					t.Fatal(e)
				}
				if h >= p.Samples[0].Height+20 {
					break
				}
				if time.Now().After(deadline) {
					t.Fatal("EN received no twenty new blocks")
				}
				time.Sleep(time.Second)
			}
		}
		s := w4Sample{}
		s.Height, err = c.BlockNumber(ctx)
		if err != nil {
			t.Fatal(err)
		}
		deadline := time.Now().Add(10 * time.Second)
		for {
			h, e := source.BlockNumber(ctx)
			if e != nil {
				t.Fatal(e)
			}
			if h >= s.Height {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("producer cannot answer pinned EN height")
			}
			time.Sleep(100 * time.Millisecond)
		}
		tag := fmt.Sprintf("0x%x", s.Height)
		if err = c.Call(ctx, "net_peerCount", &s.PeerCount); err != nil {
			t.Fatal(err)
		}
		if err = c.Call(ctx, "eth_getBlockByNumber", &s.Reader, tag, false); err != nil {
			t.Fatal(err)
		}
		if err = source.Call(ctx, "eth_getBlockByNumber", &s.Source, tag, false); err != nil {
			t.Fatal(err)
		}
		s.Prometheus = w4Fetch(t, ctx, en.MetricsURL)
		s.Expvar = w4Fetch(t, ctx, strings.TrimSuffix(en.MetricsURL, "/prometheus"))
		p.Samples = append(p.Samples, s)
	}
	if err := verifyW4Collision(p); err != nil {
		t.Fatal(err)
	}
}

func w4Fetch(t *testing.T, ctx context.Context, endpoint string) string {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		t.Fatal(err)
	}
	client := http.Client{Timeout: 5 * time.Second}
	r, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Body.Close()
	if r.StatusCode != http.StatusOK {
		t.Fatal("exporter status", r.StatusCode)
	}
	b, err := io.ReadAll(io.LimitReader(r.Body, 4*1024*1024+1))
	if err != nil || len(b) > 4*1024*1024 {
		t.Fatal("exporter body", err)
	}
	return string(b)
}
