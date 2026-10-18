package testengine_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/registry"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
)

// TestNativeLegacySynchronizationSpread observes the public BP4 EN1 case.
// Independent RPC observations retain actual live heads and a common block,
// rather than accepting a successful session alone.
func TestNativeLegacySynchronizationSpread(t *testing.T) {
	families := []struct{ chain, variable string }{{"stablenet", "GSTABLE_BIN"}, {"wbft", "GWBFT_BIN"}, {"wemix", "GWEMIX_BIN"}}
	for _, f := range families {
		if os.Getenv(f.variable) == "" {
			t.Skip("native legacy synchronization proof requires " + f.variable)
		}
	}
	for _, f := range families {
		t.Run(f.chain, func(t *testing.T) {
			root := failureRuntime(t)
			ctx, cancel := context.WithTimeout(t.Context(), 6*time.Minute)
			defer cancel()
			path := filepath.Join(repoRoot(t), "tests/tc/common/node/CT-NODE-009-head-hash-agreement.json")
			raws, err := dsl.ReadFilesWithChainPreset([]string{path}, filepath.Join(repoRoot(t), "presets/chain", f.chain+"-bp4-en1.json"))
			if err != nil {
				t.Fatal(err)
			}
			if err = os.WriteFile(filepath.Join(root, "case.json"), raws[0], 0600); err != nil {
				t.Fatal(err)
			}
			proof := map[string]any{"chain": f.chain, "binary": os.Getenv(f.variable), "scope": "One public basic-consensus case on a prepared BP4 EN1 workspace, with every endpoint's actual validators ready; original basic/sync spread restored at both checkpoints. Not cold-start timing or full legacy/L1/Web acceptance.", "all14Accepted": false}
			defer func() {
				body, err := json.MarshalIndent(proof, "", "  ")
				if err != nil {
					t.Error(err)
					return
				}
				if err = os.WriteFile(filepath.Join(root, "reports/legacy-sync-proof.json"), body, 0600); err != nil {
					t.Error(err)
				}
			}()
			var spec struct {
				ChainPreset json.RawMessage `json:"chainPreset"`
			}
			if err = json.Unmarshal(raws[0], &spec); err != nil {
				t.Fatal(err)
			}
			workspace := filepath.Join(root, "w")
			up, _, err := testengine.PlanChainPreset(ctx, spec.ChainPreset, testengine.RunSuiteIn{DataDir: workspace, Binary: os.Getenv(f.variable), KeysDir: filepath.Join(repoRoot(t), "presets/keys"), ReadOnlyKeys: true})
			if err != nil {
				t.Fatal(err)
			}
			if _, err = verb.ChainUp(ctx, chainsetup.Deps{}, up); err != nil {
				proof["setupError"] = err.Error()
				t.Fatal(err)
			}
			prepared, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: workspace})
			if err != nil || len(prepared.Nodes.Nodes) != 5 {
				t.Fatal("missing prepared five-node network", err)
			}
			plugin, err := registry.Get(f.chain)
			if err != nil {
				t.Fatal(err)
			}
			readyCtx, readyCancel := context.WithTimeout(ctx, time.Minute)
			readiness, readyErr := waitLegacySyncValidators(readyCtx, prepared.Nodes.Nodes, 4, time.Second, func(callCtx context.Context, n node.Node) (string, []string, error) {
				probeCtx, probeCancel := context.WithTimeout(callCtx, 5*time.Second)
				defer probeCancel()
				return registry.RunningValidators(probeCtx, plugin, rpc.Dial(n.RPCURL))
			})
			readyCancel()
			proof["readiness"] = readiness
			if readyErr != nil {
				proof["readinessError"] = readyErr.Error()
				t.Fatal("every endpoint must report the actual consistent validator set", readyErr)
			}
			sessionRoot, runErr := testengine.AttachWorkspaceRun(ctx, chainsetup.Deps{}, testengine.AttachWorkspaceIn{DataDir: workspace, Chain: f.chain, ArtifactRoot: filepath.Join(root, "reports"), Specs: raws, ReadOnlyKeys: true, NoNodeControl: true})
			proof["session"] = sessionRoot
			if runErr != nil {
				proof["runError"] = runErr.Error()
			}
			summary, summaryErr := testengine.ReadSessionSummary(sessionRoot)
			proof["summary"] = summary
			if summaryErr != nil {
				t.Fatal(summaryErr)
			}
			if runErr != nil || summary.Summary.Pass != 1 || summary.Failed() || summary.Summary.Skip != 0 {
				t.Fatalf("public synchronization case: %v %+v", runErr, summary)
			}
			assertPath := filepath.Join(sessionRoot, "tests/001_basic-consensus/assert.json")
			body, err := os.ReadFile(assertPath)
			if err != nil {
				t.Fatal(err)
			}
			var assertions []session.AssertResult
			if err = json.Unmarshal(body, &assertions); err != nil {
				t.Fatal(err)
			}
			spreads, hashes := 0, 0
			for _, assertion := range assertions {
				if !assertion.Pass {
					t.Fatal("failed synchronization assertion", assertion)
				}
				if assertion.Assert == "derive" && assertion.Provenance["op"] == "diff" {
					spreads++
				}
				if assertion.Assert == "sameBlockHash" {
					hashes++
				}
			}
			if len(assertions) != 13 || spreads != 10 || hashes != 2 {
				t.Fatal("both five-node spread and hash checkpoints required", len(assertions), spreads, hashes)
			}
			proof["assertions"] = assertPath
			state, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: filepath.Join(root, "w")})
			if err != nil || len(state.Nodes.Nodes) != 5 {
				t.Fatal("missing actual five-node network", err)
			}
			minimum, maximum := ^uint64(0), uint64(0)
			producers, endpoints := 0, 0
			observations := []map[string]any{}
			for _, n := range state.Nodes.Nodes {
				switch n.Role {
				case node.RoleBP:
					producers++
				case node.RoleEN:
					endpoints++
				}
				head, err := rpc.Dial(n.RPCURL).BlockNumber(ctx)
				if err != nil || head < 20 {
					t.Fatal("live node did not reach the second checkpoint", n.Index, head, err)
				}
				minimum = min(minimum, head)
				maximum = max(maximum, head)
				observations = append(observations, map[string]any{"node": n.Index, "role": n.Role, "rpc": n.RPCURL, "height": head})
			}
			proof["liveNodes"] = observations
			proof["minimum"], proof["maximum"] = minimum, maximum
			if producers != 4 || endpoints != 1 || maximum-minimum > 2 {
				t.Fatal("independent live height spread or role mismatch", producers, endpoints, minimum, maximum)
			}
			var want string
			for i, n := range state.Nodes.Nodes {
				var block struct{ Number, Hash string }
				tag := fmt.Sprintf("0x%x", minimum)
				if err = rpc.Dial(n.RPCURL).Call(ctx, "eth_getBlockByNumber", &block, tag, false); err != nil || block.Number != tag || len(block.Hash) != 66 {
					t.Fatal("missing actual common block", n.Index, block, err)
				}
				observations[i]["commonBlock"] = block
				if want != "" && block.Hash != want {
					t.Fatal("independent common block differs", n.Index, block.Hash, want)
				}
				want = block.Hash
			}
			proof["liveSpreadAndHashVerified"] = true
		})
	}
}

// TestLegacySyncReadinessWaitsForEveryEndpoint keeps initial failures visible.
func TestLegacySyncReadinessWaitsForEveryEndpoint(t *testing.T) {
	nodes := []node.Node{{Index: 1, RPCURL: "http://one"}, {Index: 2, RPCURL: "http://two"}}
	secondCalls := 0
	probe := func(_ context.Context, n node.Node) (string, []string, error) {
		if n.Index == 2 {
			secondCalls++
			if secondCalls == 1 {
				return "test validators", nil, fmt.Errorf("initializing")
			}
		}
		return "test validators", []string{"a", "b"}, nil
	}
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	observations, err := waitLegacySyncValidators(ctx, nodes, 2, time.Millisecond, probe)
	if err != nil || len(observations) != 4 || observations[1]["ready"] != false || observations[2]["ready"] != true || observations[3]["ready"] != true {
		t.Fatalf("all endpoints must become ready after preserving initial failure: %v %+v", err, observations)
	}
}

// TestLegacySyncReadinessRejectsIncompleteSets requires a bounded real result.
func TestLegacySyncReadinessRejectsIncompleteSets(t *testing.T) {
	nodes := []node.Node{{Index: 1, RPCURL: "http://one"}, {Index: 2, RPCURL: "http://two"}}
	for _, name := range []string{"missing", "duplicate", "short", "different sets", "RPC failure"} {
		t.Run(name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
			defer cancel()
			probe := func(_ context.Context, n node.Node) (string, []string, error) {
				switch name {
				case "missing":
					return "test validators", nil, nil
				case "duplicate":
					return "test validators", []string{"a", "a"}, nil
				case "short":
					return "test validators", []string{"a"}, nil
				case "RPC failure":
					return "test validators", nil, fmt.Errorf("unavailable")
				default:
					if n.Index == 1 {
						return "test validators", []string{"a", "b"}, nil
					}
					return "test validators", []string{"a", "c"}, nil
				}
			}
			observations, err := waitLegacySyncValidators(ctx, nodes, 2, time.Millisecond, probe)
			if !errors.Is(err, context.DeadlineExceeded) || len(observations) < 2 {
				t.Fatalf("unavailable or inconsistent data must not become ready: %v %+v", err, observations)
			}
		})
	}
}

// TestLegacySyncReadinessHonorsCancellation refuses stale successful callbacks.
func TestLegacySyncReadinessHonorsCancellation(t *testing.T) {
	nodes := []node.Node{{Index: 1, RPCURL: "http://one"}}
	for _, before := range []bool{true, false} {
		t.Run(fmt.Sprint(before), func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			calls := 0
			probe := func(context.Context, node.Node) (string, []string, error) {
				calls++
				cancel()
				return "test validators", []string{"a"}, nil
			}
			if before {
				cancel()
			}
			_, err := waitLegacySyncValidators(ctx, nodes, 1, time.Millisecond, probe)
			if !errors.Is(err, context.Canceled) || before && calls != 0 {
				t.Fatal("cancellation must prevent acceptance", before, calls, err)
			}
		})
	}
}

// waitLegacySyncValidators records every probe and accepts one complete round.
// It never substitutes a producer's answer for an unavailable endpoint.
func waitLegacySyncValidators(ctx context.Context, nodes []node.Node, want int, poll time.Duration, probe func(context.Context, node.Node) (string, []string, error)) ([]map[string]any, error) {
	if want <= 0 || poll <= 0 || len(nodes) == 0 || probe == nil {
		return nil, fmt.Errorf("invalid validator readiness inputs")
	}
	indices := map[int]bool{}
	for _, n := range nodes {
		if n.Index <= 0 || indices[n.Index] || n.RPCURL == "" {
			return nil, fmt.Errorf("invalid readiness node identity")
		}
		indices[n.Index] = true
	}
	ticker := time.NewTicker(poll)
	defer ticker.Stop()
	observations := []map[string]any{}
	for round := 1; ; round++ {
		complete := true
		var expected map[string]bool
		for _, n := range nodes {
			if err := ctx.Err(); err != nil {
				return observations, err
			}
			source, vals, err := probe(ctx, n)
			set := map[string]bool{}
			ready := err == nil && source != "" && len(vals) == want
			for _, value := range vals {
				key := strings.ToLower(strings.TrimSpace(value))
				if key == "" || set[key] {
					ready = false
				}
				set[key] = true
			}
			if ready {
				if expected == nil {
					expected = set
				} else {
					for key := range expected {
						if !set[key] {
							ready = false
						}
					}
				}
			}
			observation := map[string]any{"node": n.Index, "source": source, "validators": append([]string(nil), vals...), "round": round, "ready": ready}
			if err != nil {
				observation["errorType"] = fmt.Sprintf("%T", err)
			}
			observations = append(observations, observation)
			complete = complete && ready
		}
		if err := ctx.Err(); err != nil {
			return observations, err
		}
		if complete {
			return observations, nil
		}
		select {
		case <-ctx.Done():
			return observations, ctx.Err()
		case <-ticker.C:
		}
	}
}
