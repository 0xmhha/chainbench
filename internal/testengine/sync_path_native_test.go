package testengine_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

func TestNativeSyncPathObservationsOnEveryChain(t *testing.T) {
	families := []struct{ chain, variable string }{{"stablenet", "GSTABLE_BIN"}, {"wbft", "GWBFT_BIN"}, {"wemix", "GWEMIX_BIN"}}
	for _, family := range families {
		if os.Getenv(family.variable) == "" {
			t.Skip("native sync proof requires " + family.variable)
		}
	}
	cases := []struct{ file, metric string }{
		{"node/CT-NODE-004-full-sync.json", "eth_downloader_headers_in"},
		{"node/CT-NODE-005-snap-sync.json", "eth_downloader_receipts_in"},
		{"node/CT-NODE-006-missing-block-catch-up.json", "eth_downloader_headers_in"},
		{"node/CT-NODE-007-live-block-receive.json", "eth_fetcher_block_broadcasts_in"},
		{"fee/CT-FEE-009-snap-receipt-gas-price.json", "eth_downloader_receipts_in"},
	}
	for _, family := range families {
		for _, c := range cases {
			t.Run(family.chain+"/"+filepath.Base(c.file), func(t *testing.T) {
				root := os.Getenv("WEBUI_RUNTIME_ROOT")
				if root == "" {
					root = t.TempDir()
				}
				fixture, err := os.MkdirTemp(root, "sync-path-native-")
				if err != nil {
					t.Fatal(err)
				}
				workspace := filepath.Join(fixture, "w")
				t.Cleanup(func() {
					ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
					defer cancel()
					if _, err := verb.ChainStop(ctx, chainsetup.Deps{}, verb.ChainStopIn{DataDir: workspace}); err != nil {
						t.Error(err)
						return
					}
					if _, err := verb.ChainRm(ctx, chainsetup.Deps{}, verb.ChainRmIn{DataDir: workspace}); err != nil {
						t.Error(err)
					}
				})
				casePath := filepath.Join(repoRoot(t), "tests/tc/common", c.file)
				raws, err := dsl.ReadFilesWithChainPreset([]string{casePath}, family.chain+"-bp4-en1")
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(t.Context(), 12*time.Minute)
				defer cancel()
				out, runErr := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{SpecContent: raws, DataDir: workspace, ArtifactRoot: filepath.Join(fixture, "reports"), Binary: os.Getenv(family.variable), KeysDir: filepath.Join(repoRoot(t), "presets/keys"), ReadOnlyKeys: true, KeepUp: true, WaitBlocks: 2})
				// Preserve node logs and public configuration before cleanup even when
				// the chain fails. Every case is an independent owned network.
				diagnostics := filepath.Join(fixture, "diagnostics")
				if err := os.MkdirAll(diagnostics, 0700); err != nil {
					t.Fatal(err)
				}
				// Capture the raw exporter responses while the owned nodes are still
				// alive. Diagnostics do not substitute for the case assertions.
				status, statusErr := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: workspace})
				if statusErr == nil {
					for _, n := range status.Nodes.Nodes {
						if n.Index != 5 || n.MetricsURL == "" {
							continue
						}
						for suffix, endpoint := range map[string]string{"prometheus.txt": n.MetricsURL, "expvar.json": strings.TrimSuffix(n.MetricsURL, "/prometheus")} {
							body, err := exec.CommandContext(ctx, "/usr/bin/curl", "-fsS", "--max-time", "5", endpoint).CombinedOutput()
							if err != nil {
								suffix += ".error.txt"
								body = append(body, []byte(err.Error())...)
							}
							if err := os.WriteFile(filepath.Join(diagnostics, "node5-metrics-"+suffix), body, 0600); err != nil {
								t.Fatal(err)
							}
						}
					}
				}
				for _, pattern := range []string{"chain-record.json", "process.json", "logs/*", "runs/*/genesis.json", "runs/*/config_node*.toml", "runs/*/launch-commands.txt"} {
					files, err := filepath.Glob(filepath.Join(workspace, pattern))
					if err != nil {
						t.Fatal(err)
					}
					for _, path := range files {
						info, err := os.Stat(path)
						if err != nil || !info.Mode().IsRegular() {
							continue
						}
						relative, err := filepath.Rel(workspace, path)
						if err != nil {
							t.Fatal(err)
						}
						destination := filepath.Join(diagnostics, relative)
						if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
							t.Fatal(err)
						}
						if err := exec.Command("/bin/cp", "-p", path, destination).Run(); err != nil {
							t.Fatal(err)
						}
					}
				}
				proof := map[string]any{"chain": family.chain, "case": c.file, "metric": c.metric, "session": out.SessionRoot, "summary": out.Summary, "diagnostics": diagnostics}
				if runErr != nil {
					proof["runError"] = runErr.Error()
				}
				sources := map[string]string{}
				for _, path := range []string{casePath, filepath.Join(repoRoot(t), "internal/testhelper/metric.go"), filepath.Join(repoRoot(t), "internal/core/collector/metrics.go"), filepath.Join(repoRoot(t), "internal/testengine/sync_path_native_test.go"), os.Getenv(family.variable)} {
					data, err := os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					sum := sha256.Sum256(data)
					sources[path] = hex.EncodeToString(sum[:])
				}
				proof["sources"] = sources
				files, _ := filepath.Glob(filepath.Join(out.SessionRoot, "tests", "*", "assert.json"))
				matches := []session.AssertResult{}
				for _, file := range files {
					data, err := os.ReadFile(file)
					if err != nil {
						t.Fatal(err)
					}
					var assertions []session.AssertResult
					if err := json.Unmarshal(data, &assertions); err != nil {
						t.Fatal(err)
					}
					for _, a := range assertions {
						if a.Assert == "metric" && a.Provenance["name"] == c.metric {
							matches = append(matches, a)
						}
					}
				}
				proof["metricAssertions"] = matches
				data, err := json.MarshalIndent(proof, "", "  ")
				if err != nil {
					t.Fatal(err)
				}
				proofPath := filepath.Join(fixture, "proof.json")
				if err := os.WriteFile(proofPath, data, 0600); err != nil {
					t.Fatal(err)
				}
				t.Log("native sync path proof:", proofPath)
				if runErr != nil || out.Summary.Summary.Pass != 1 || out.Summary.Failed() || out.Summary.Summary.Skip != 0 {
					t.Fatalf("sync path case failed: %v %+v", runErr, out.Summary)
				}
				if len(matches) != 1 || !matches[0].Pass {
					t.Fatal("the actual path counter was not asserted", matches)
				}
				samples, ok := matches[0].Provenance["metricSamples"].([]any)
				wantSamples := 1
				if c.metric == "eth_fetcher_block_broadcasts_in" {
					wantSamples = 2
				}
				if !ok || len(samples) != wantSamples {
					t.Fatal("counter origin missing", matches)
				}
				for i, value := range samples {
					sample, ok := value.(map[string]any)
					wanted := c.metric
					if i == 1 {
						wanted = "eth_fetcher_block_announces_in"
					}
					if !ok || sample["name"] != wanted || sample["node"] != float64(5) || sample["role"] != "en" || sample["source"] != "recorded-endpoint" || sample["status"] != "observed" {
						t.Fatal("wrong observed node or source", samples)
					}
					if pid, ok := sample["pid"].(float64); !ok || pid <= 0 {
						t.Fatal("counter has no current process identity", samples)
					}
				}
			})
			if t.Failed() {
				return
			}
		}
	}
}
