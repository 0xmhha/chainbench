package testengine_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/0xmhha/chainbench/internal/app"
	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type quorumBlock struct {
	Number                   uint64
	Hash, ParentHash, Author string
	Committers, Validators   []string
}
type quorumProof struct {
	N, Faults, Quorum, LiveCount int
	Validators, Stopped          []string
	Healthy, Fault, Recovery     []quorumBlock
	HaltStart, HaltEnd           uint64
	HaltMillis                   int64
	HealthyStatusCommitted       map[string]int
	HaltHeads                    map[string][2]uint64
}

func verifyQuorumBoundary(p quorumProof) error {
	if p.N < 3 || p.Faults <= 0 || p.Faults >= p.N || len(p.Validators) != p.N || len(p.Stopped) != p.Faults || p.Quorum != p.N-(p.N-1)/3 || p.LiveCount != p.N-p.Faults || len(p.Healthy) < 3 || len(p.Recovery) < 3 {
		return fmt.Errorf("wrong actual quorum context")
	}
	allowed, dead := quorumAddressSet(p.Validators), quorumAddressSet(p.Stopped)
	if len(allowed) != p.N || allowed[""] || len(dead) != p.Faults {
		return fmt.Errorf("invalid address context")
	}
	for key := range dead {
		if !allowed[key] {
			return fmt.Errorf("stopped node outside validators")
		}
	}
	check := func(rows []quorumBlock, fault bool) error {
		for i, row := range rows {
			current := quorumAddressSet(row.Validators)
			if len(current) != p.N || row.Hash == "" || !allowed[strings.ToLower(row.Author)] || len(row.Committers) < p.Quorum || i > 0 && (row.Number != rows[i-1].Number+1 || row.ParentHash != rows[i-1].Hash) {
				return fmt.Errorf("invalid committed block context")
			}
			for key := range current {
				if !allowed[key] {
					return fmt.Errorf("validator membership changed")
				}
			}
			seen := map[string]bool{}
			for _, key := range row.Committers {
				key = strings.ToLower(key)
				if !allowed[key] || seen[key] || fault && dead[key] {
					return fmt.Errorf("invalid/dead/duplicate committer")
				}
				seen[key] = true
			}
			if fault && p.LiveCount == p.Quorum && len(row.Committers) != p.Quorum {
				return fmt.Errorf("not exact surviving quorum")
			}
		}
		return nil
	}
	if err := check(p.Healthy, false); err != nil {
		return err
	}
	if err := check(p.Recovery, false); err != nil {
		return err
	}
	counts := map[string]int{}
	for _, row := range p.Healthy {
		for _, key := range row.Committers {
			counts[strings.ToLower(key)]++
		}
	}
	observed := map[string]int{}
	for key, n := range p.HealthyStatusCommitted {
		key = strings.ToLower(key)
		if !allowed[key] || n < 0 {
			return fmt.Errorf("invalid status signer")
		}
		if n > 0 {
			observed[key] = n
		}
	}
	if len(observed) != len(counts) {
		return fmt.Errorf("missing healthy status")
	}
	for key, n := range counts {
		if observed[key] != n {
			return fmt.Errorf("healthy status differs")
		}
	}
	if p.LiveCount >= p.Quorum {
		if len(p.Fault) < 10 {
			return fmt.Errorf("need ten blocks during fault")
		}
		return check(p.Fault, true)
	}
	if len(p.Fault) != 0 || p.HaltStart != p.HaltEnd || p.HaltMillis < 8000 || len(p.HaltHeads) != p.LiveCount {
		return fmt.Errorf("missing actual halt window")
	}
	for key, heads := range p.HaltHeads {
		key = strings.ToLower(key)
		if !allowed[key] || dead[key] || heads[0] != heads[1] {
			return fmt.Errorf("survivor grew during quorum deficit")
		}
	}
	return nil
}

func quorumFixture(n, f int) quorumProof {
	p := quorumProof{N: n, Faults: f, Quorum: n - (n-1)/3, LiveCount: n - f, HaltStart: 10, HaltEnd: 10, HaltMillis: 8000, HealthyStatusCommitted: map[string]int{}}
	for i := 0; i < n; i++ {
		p.Validators = append(p.Validators, fmt.Sprint("v", i))
		if i >= n-f {
			p.Stopped = append(p.Stopped, fmt.Sprint("v", i))
		}
	}
	rows := func(start, count int, signers []string) []quorumBlock {
		out := []quorumBlock{}
		for i := 0; i < count; i++ {
			out = append(out, quorumBlock{Number: uint64(start + i), Hash: fmt.Sprint(start + i), ParentHash: fmt.Sprint(start + i - 1), Author: signers[0], Committers: append([]string(nil), signers...), Validators: append([]string(nil), p.Validators...)})
		}
		return out
	}
	p.HaltHeads = map[string][2]uint64{}
	for _, v := range p.Validators[:p.LiveCount] {
		p.HaltHeads[v] = [2]uint64{10, 10}
	}
	p.Healthy = rows(5, 3, p.Validators[:p.Quorum])
	for _, row := range p.Healthy {
		for _, v := range row.Committers {
			p.HealthyStatusCommitted[v]++
		}
	}
	if p.LiveCount >= p.Quorum {
		p.Fault = rows(11, 10, p.Validators[:p.LiveCount])
	}
	p.Recovery = rows(30, 3, p.Validators[:p.Quorum])
	return p
}

func TestQuorumBoundariesRejectWeakFaultEvidence(t *testing.T) {
	for _, name := range []string{"live7", "halt7", "halt3", "live6", "halt6", "short live", "dead signer", "unknown signer", "duplicate signer", "wrong quorum", "wrong live count", "wrong validator count", "halt grew", "short halt", "healthy3 insufficient", "wrong status"} {
		t.Run(name, func(t *testing.T) {
			p := quorumFixture(6, 1)
			valid := false
			switch name {
			case "live7":
				p = quorumFixture(7, 1)
				valid = true
			case "halt7":
				p = quorumFixture(7, 3)
				valid = true
			case "halt3":
				p = quorumFixture(3, 1)
				valid = true
			case "live6":
				valid = true
			case "halt6":
				p = quorumFixture(6, 2)
				valid = true
			case "short live":
				p.Fault = p.Fault[:3]
			case "dead signer":
				p.Fault[0].Committers[0] = p.Stopped[0]
			case "unknown signer":
				p.Fault[0].Committers[0] = "stranger"
			case "duplicate signer":
				p.Fault[0].Committers[1] = p.Fault[0].Committers[0]
			case "wrong quorum":
				p.Quorum = 4
			case "wrong live count":
				p.LiveCount = 6
			case "wrong validator count":
				p.Validators = p.Validators[:5]
			case "halt grew":
				p = quorumFixture(7, 3)
				p.HaltEnd++
			case "short halt":
				p = quorumFixture(6, 2)
				p.HaltMillis = 500
			case "healthy3 insufficient":
				p = quorumFixture(3, 1)
				p.Healthy[0].Committers = p.Healthy[0].Committers[:2]
			case "wrong status":
				p.HealthyStatusCommitted["v0"]--
			}
			err := verifyQuorumBoundary(p)
			if valid && err != nil {
				t.Fatal(err)
			}
			if !valid && err == nil {
				t.Fatal("weak fault proof accepted missing original quorum conditions")
			}
		})
	}
}

func quorumAddressSet(values []string) map[string]bool {
	out := map[string]bool{}
	for _, v := range values {
		out[strings.ToLower(v)] = true
	}
	return out
}

func TestNativeQuorumFaultBoundaries(t *testing.T) {
	bin := os.Getenv("GWBFT_BIN")
	if bin == "" {
		t.Skip("set GWBFT_BIN for actual quorum boundaries")
	}
	for _, profile := range []struct {
		id   string
		n, f int
	}{{"WBFT-007", 7, 1}, {"WBFT-008", 7, 3}, {"WBFT-011", 3, 1}, {"WBFT-012", 6, 1}, {"WBFT-013", 6, 2}} {
		t.Run(profile.id, func(t *testing.T) {
			root := failureRuntime(t)
			ctx, cancel := context.WithTimeout(t.Context(), 6*time.Minute)
			defer cancel()
			data, keyDir := filepath.Join(root, "w"), filepath.Join(root, "keyring")
			n := profile.n
			if _, err := app.KeyringNew(ctx, app.Deps{}, app.RingCreateIn{Ring: app.RingRef{Dir: keyDir}, Count: n, Validators: &n, WithBLS: true}); err != nil {
				t.Fatal(err)
			}
			env := map[string]any{"schemaVersion": "2", "kind": "chain-preset", "id": "quorum-" + profile.id, "chain": "wbft", "topology": map[string]any{"bp": n}, "keys": map[string]any{"nodekeys": map[string]any{"ref": keyDir, "source": "keyPreset"}}, "genesis": map[string]any{"set": map[string]any{"config.croissant.wBFT.targetValidators": n}}}
			b, err := json.Marshal(env)
			if err != nil {
				t.Fatal(err)
			}
			envPath := filepath.Join(root, "env.json")
			if err = os.WriteFile(envPath, b, 0600); err != nil {
				t.Fatal(err)
			}
			raw, err := json.Marshal(map[string]any{"schemaVersion": "2", "kind": "case", "id": "quorum-ready", "chainPreset": env["id"], "requires": []string{"rpc"}, "steps": []any{map[string]any{"do": "waitBlock", "target": 8, "timeout": "90s"}, map[string]any{"expect": "blockNumber", "compare": "GreaterOrEqual", "is": "8"}}})
			if err != nil {
				t.Fatal(err)
			}
			casePath := filepath.Join(root, "case.json")
			if err = os.WriteFile(casePath, raw, 0600); err != nil {
				t.Fatal(err)
			}
			specs, err := dsl.ReadFilesWithChainPreset([]string{casePath}, envPath)
			if err != nil {
				t.Fatal(err)
			}
			out, runErr := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{SpecContent: specs, DataDir: data, ArtifactRoot: filepath.Join(root, "reports"), Binary: bin, KeysDir: keyDir, ReadOnlyKeys: true, KeepUp: true, WaitBlocks: 2})
			p := quorumProof{N: n, Faults: profile.f, Quorum: n - (n-1)/3}
			proof := map[string]any{"criterion": profile.id, "session": out.SessionRoot, "summary": out.Summary, "keyringDir": keyDir, "keySource": "app keyring new, generated with BLS for exact validator count", "scope": "Managed current WBFT Croissant0 exact quorum/fault fixture; not original staged migration/mainnet/Linux/legacy/full87/final Web acceptance."}
			defer func() {
				proof["quorum"] = p
				b, err := json.MarshalIndent(proof, "", "  ")
				if err != nil {
					t.Error(err)
					return
				}
				if err = os.WriteFile(filepath.Join(root, "reports/quorum-boundary-proof.json"), b, 0600); err != nil {
					t.Error(err)
				}
			}()
			if runErr != nil || out.Summary.Summary.Pass != 1 || out.Summary.Failed() || out.Summary.Summary.Skip != 0 {
				t.Fatal("quorum fixture not ready", runErr, out.Summary)
			}
			state, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: data})
			if err != nil || len(state.Nodes.Nodes) != n {
				t.Fatal("wrong actual nodes", err)
			}
			c := rpc.Dial(state.Nodes.Nodes[0].RPCURL)
			if err = c.Call(ctx, "istanbul_getValidators", &p.Validators, "latest"); err != nil {
				t.Fatal(err)
			}
			byIndex := map[int]string{}
			oldPIDs := map[int]int{}
			for _, nd := range state.Nodes.Nodes {
				var addr string
				if err = rpc.Dial(nd.RPCURL).Call(ctx, "istanbul_nodeAddress", &addr); err != nil {
					t.Fatal(err)
				}
				byIndex[nd.Index] = addr
				oldPIDs[nd.Index] = nd.PID
			}
			waitHead := func(want uint64) {
				t.Helper()
				for {
					head, err := c.BlockNumber(ctx)
					if err != nil {
						t.Fatal(err)
					}
					if head >= want {
						return
					}
					select {
					case <-ctx.Done():
						t.Fatal(ctx.Err())
					case <-time.After(200 * time.Millisecond):
					}
				}
			}
			collect := func(first, last uint64) []quorumBlock {
				t.Helper()
				rows := []quorumBlock{}
				for number := first; number <= last; number++ {
					tag := fmt.Sprintf("0x%x", number)
					var row quorumBlock
					if err = c.Call(ctx, "istanbul_getCommitSignersFromBlock", &row, tag); err != nil {
						t.Fatal(err)
					}
					h, err := c.BlockByNumber(ctx, tag)
					if err != nil || row.Number != number || row.Hash != h.Hash {
						t.Fatal("commit signer/header mismatch", row, h, err)
					}
					var header struct{ ParentHash string }
					if err = c.Call(ctx, "eth_getBlockByNumber", &header, tag, false); err != nil {
						t.Fatal(err)
					}
					row.ParentHash = header.ParentHash
					if err = c.Call(ctx, "istanbul_getValidators", &row.Validators, tag); err != nil {
						t.Fatal(err)
					}
					rows = append(rows, row)
				}
				return rows
			}
			first, err := c.BlockNumber(ctx)
			if err != nil {
				t.Fatal(err)
			}
			waitHead(first + 2)
			p.Healthy = collect(first, first+2)
			var status struct {
				SealerActivity struct {
					Committed map[string]int `json:"committed"`
				} `json:"sealerActivity"`
			}
			if err = c.Call(ctx, "istanbul_status", &status, fmt.Sprintf("0x%x", first), fmt.Sprintf("0x%x", first+2)); err != nil {
				t.Fatal(err)
			}
			p.HealthyStatusCommitted = status.SealerActivity.Committed
			for index := n - profile.f + 1; index <= n; index++ {
				if err = verb.NodeStop(ctx, chainsetup.Deps{}, verb.NodeStopIn{DataDir: data, Index: index}); err != nil {
					t.Fatal(err)
				}
				p.Stopped = append(p.Stopped, byIndex[index])
			}
			stopped, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: data})
			if err != nil {
				t.Fatal(err)
			}
			p.LiveCount = len(stopped.Alive)
			for _, nd := range stopped.Nodes.Nodes {
				if nd.Index > n-profile.f && nd.PID != 0 {
					t.Fatal("fault node not stopped")
				}
			}
			// Drain an in-flight block after all process stops; the measured fault window
			// begins after this explicit one-second settle, not before the final stop.
			select {
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			case <-time.After(time.Second):
			}
			baseline, err := c.BlockNumber(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if p.LiveCount >= p.Quorum {
				waitHead(baseline + 10)
				p.Fault = collect(baseline+1, baseline+10)
			} else {
				p.HaltStart = baseline
				p.HaltHeads = map[string][2]uint64{}
				for _, nd := range stopped.Nodes.Nodes {
					if nd.PID == 0 {
						continue
					}
					head, err := rpc.Dial(nd.RPCURL).BlockNumber(ctx)
					if err != nil {
						t.Fatal(err)
					}
					p.HaltHeads[byIndex[nd.Index]] = [2]uint64{head, head}
				}
				began := time.Now()
				select {
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				case <-time.After(8 * time.Second):
				}
				p.HaltMillis = time.Since(began).Milliseconds()
				p.HaltEnd, err = c.BlockNumber(ctx)
				if err != nil {
					t.Fatal(err)
				}
				for _, nd := range stopped.Nodes.Nodes {
					if nd.PID == 0 {
						continue
					}
					head, err := rpc.Dial(nd.RPCURL).BlockNumber(ctx)
					if err != nil {
						t.Fatal(err)
					}
					pair := p.HaltHeads[byIndex[nd.Index]]
					pair[1] = head
					p.HaltHeads[byIndex[nd.Index]] = pair
				}
			}
			restarted := []node.Node{}
			newPIDs := map[int]int{}
			for index := n - profile.f + 1; index <= n; index++ {
				result, err := verb.NodeStart(ctx, chainsetup.Deps{}, verb.NodeStartIn{DataDir: data, Index: index})
				if err != nil {
					t.Fatal(err)
				}
				if result.Node.PID <= 0 || result.Node.PID == oldPIDs[index] {
					t.Fatal("node not actually restarted")
				}
				if _, err = waitRestartIdentity(ctx, rpc.Dial(result.Node.RPCURL), byIndex[index]); err != nil {
					t.Fatal(err)
				}
				restarted = append(restarted, result.Node)
				newPIDs[index] = result.Node.PID
			}
			recovered, err := c.BlockNumber(ctx)
			if err != nil {
				t.Fatal(err)
			}
			waitHead(recovered + 3)
			p.Recovery = collect(recovered+1, recovered+3)
			for _, nd := range restarted {
				for {
					head, err := rpc.Dial(nd.RPCURL).BlockNumber(ctx)
					if err == nil && head >= recovered+3 {
						break
					}
					select {
					case <-ctx.Done():
						t.Fatal("restarted node did not catch up")
					case <-time.After(200 * time.Millisecond):
					}
				}
			}
			proof["oldPIDs"] = oldPIDs
			proof["newPIDs"] = newPIDs
			proof["nodeAddresses"] = byIndex
			proof["recoveryTarget"] = recovered + 3
			if err = verifyQuorumBoundary(p); err != nil {
				t.Fatal(err)
			}
		})
	}
}
