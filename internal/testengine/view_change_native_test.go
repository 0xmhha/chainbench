package testengine_test

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/0xmhha/chainbench/internal/chainsetup"
	"github.com/0xmhha/chainbench/internal/chainsetup/verb"
	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/rpc"
	"github.com/0xmhha/chainbench/internal/dsl"
	"github.com/0xmhha/chainbench/internal/testengine"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"
)

type viewHeader struct {
	Number, Timestamp, Round, PrevRound uint64
	Hash, ParentHash, Miner             string
}
type viewProof struct {
	Validators                       []string
	Before, Changed, Successor       viewHeader
	StoppedAddress, RestartedAddress string
	RequestTimeout, BlockPeriod      uint64
	OldPID, NewPID, TargetIndex      int
	RoundChangeNodes                 int
	CommitLogDelayMillis             int64
	AfterRestart                     []viewHeader
}

func verifyViewChange(p viewProof) error {
	if len(p.Validators) < 4 || len(p.AfterRestart) != 6 || p.RequestTimeout == 0 || p.BlockPeriod == 0 || p.OldPID <= 0 || p.NewPID <= 0 || p.NewPID == p.OldPID || p.TargetIndex <= 0 || p.RoundChangeNodes < 2 || !sameAddress(p.StoppedAddress, p.RestartedAddress) {
		return fmt.Errorf("missing recovery context")
	}
	allowed := map[string]bool{}
	for _, addr := range p.Validators {
		key := strings.ToLower(addr)
		if key == "" || allowed[key] {
			return fmt.Errorf("invalid validator set")
		}
		allowed[key] = true
	}
	if !allowed[strings.ToLower(p.Changed.Miner)] {
		return fmt.Errorf("new proposer outside validator set")
	}
	index := -1
	for i, addr := range p.Validators {
		if sameAddress(addr, p.Before.Miner) {
			index = i
		}
	}
	if index < 0 || !sameAddress(p.StoppedAddress, p.Validators[(index+1)%len(p.Validators)]) {
		return fmt.Errorf("stopped node is not the observed round-zero next proposer")
	}
	if p.Changed.Number != p.Before.Number+1 || p.Changed.ParentHash != p.Before.Hash || p.Changed.Round == 0 || sameAddress(p.Changed.Miner, p.StoppedAddress) || p.Changed.Timestamp < p.Before.Timestamp+p.BlockPeriod || p.CommitLogDelayMillis < int64(p.RequestTimeout)*1000 || p.CommitLogDelayMillis <= int64(p.BlockPeriod)*1000 {
		return fmt.Errorf("no timeout/new proposer round change")
	}
	if p.Successor.Number != p.Changed.Number+1 || p.Successor.ParentHash != p.Changed.Hash || p.Successor.PrevRound != p.Changed.Round {
		return fmt.Errorf("no matching PrevRound trace")
	}
	for i, h := range p.AfterRestart {
		if h.Hash == "" || i > 0 && (h.Number != p.AfterRestart[i-1].Number+1 || h.ParentHash != p.AfterRestart[i-1].Hash) {
			return fmt.Errorf("not five consecutive post-restart links")
		}
	}
	return nil
}

func TestViewChangeRejectsPresenceOnlyRecovery(t *testing.T) {
	for _, name := range []string{"valid", "wrong target", "no round change", "no previous round trace", "no timeout delay", "same proposer", "same pid", "short chain", "wrong height", "wrong parent", "no propagated log", "unknown new proposer", "wrong restarted identity"} {
		t.Run(name, func(t *testing.T) {
			p := viewProof{Validators: []string{"v0", "v1", "v2", "v3"}, Before: viewHeader{Number: 10, Timestamp: 100, Miner: "v0", Hash: "10"}, Changed: viewHeader{Number: 11, Timestamp: 105, Miner: "v2", Hash: "11", ParentHash: "10", Round: 1}, Successor: viewHeader{Number: 12, Timestamp: 107, PrevRound: 1, Hash: "12", ParentHash: "11"}, StoppedAddress: "v1", RestartedAddress: "v1", RequestTimeout: 3, BlockPeriod: 2, OldPID: 123, NewPID: 456, TargetIndex: 2, RoundChangeNodes: 2, CommitLogDelayMillis: 5000}
			for n := uint64(15); n <= 20; n++ {
				p.AfterRestart = append(p.AfterRestart, viewHeader{Number: n, Hash: fmt.Sprint(n), ParentHash: fmt.Sprint(n - 1)})
			}
			switch name {
			case "unknown new proposer":
				p.Changed.Miner = "stranger"
			case "wrong restarted identity":
				p.RestartedAddress = "v3"
			case "wrong target":
				p.StoppedAddress = "v3"
			case "no round change":
				p.Changed.Round = 0
			case "no previous round trace":
				p.Successor.PrevRound = 0
			case "no timeout delay":
				p.Changed.Timestamp = 102
				p.CommitLogDelayMillis = 2000
			case "same proposer":
				p.Changed.Miner = "v1"
			case "same pid":
				p.NewPID = p.OldPID
			case "short chain":
				p.AfterRestart = p.AfterRestart[:4]
			case "wrong height":
				p.AfterRestart[1].Number++
			case "wrong parent":
				p.AfterRestart[1].ParentHash = "unrelated"
			case "no propagated log":
				p.RoundChangeNodes = 1
			}
			err := verifyViewChange(p)
			if name == "valid" && err != nil {
				t.Fatal(err)
			}
			if name != "valid" && err == nil {
				t.Fatal("presence-only recovery accepted invalid view-change proof")
			}
		})
	}
}

func sameAddress(a, b string) bool { return strings.EqualFold(a, b) }

func readViewHeader(ctx context.Context, c *rpc.Client, n uint64) (viewHeader, error) {
	tag := fmt.Sprintf("0x%x", n)
	var b struct{ Number, Timestamp, Miner, Hash, ParentHash string }
	if err := c.Call(ctx, "eth_getBlockByNumber", &b, tag, false); err != nil {
		return viewHeader{}, err
	}
	var e struct{ Round, PrevRound string }
	if err := c.Call(ctx, "istanbul_getWbftExtraInfo", &e, tag); err != nil {
		return viewHeader{}, err
	}
	values := []uint64{}
	for _, s := range []string{b.Number, b.Timestamp, e.Round, e.PrevRound} {
		v, err := strconv.ParseUint(strings.TrimPrefix(s, "0x"), 16, 64)
		if err != nil {
			return viewHeader{}, err
		}
		values = append(values, v)
	}
	if values[0] != n {
		return viewHeader{}, fmt.Errorf("wrong actual header number")
	}
	return viewHeader{Number: n, Timestamp: values[1], Round: values[2], PrevRound: values[3], Miner: b.Miner, Hash: b.Hash, ParentHash: b.ParentHash}, nil
}

func TestNativeProposerViewChangeAndFiveLinks(t *testing.T) {
	bin := os.Getenv("GWBFT_BIN")
	if bin == "" {
		t.Skip("set GWBFT_BIN for actual view change")
	}
	root := failureRuntime(t)
	ctx, cancel := context.WithTimeout(t.Context(), 6*time.Minute)
	defer cancel()
	data := filepath.Join(root, "w")
	b, err := os.ReadFile(filepath.Join(repoRoot(t), "presets/chain/wbft-bp4.json"))
	if err != nil {
		t.Fatal(err)
	}
	var env map[string]any
	if err = json.Unmarshal(b, &env); err != nil {
		t.Fatal(err)
	}
	env["id"] = "wbft-view-change"
	env["genesis"] = map[string]any{"overlay": map[string]any{"config": map[string]any{"croissant": map[string]any{"wBFT": map[string]any{"blockPeriodSeconds": 2, "requestTimeoutSeconds": 3}}}}}
	b, err = json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	envPath := filepath.Join(root, "env.json")
	if err = os.WriteFile(envPath, b, 0600); err != nil {
		t.Fatal(err)
	}
	casePath := filepath.Join(root, "case.json")
	if err = os.WriteFile(casePath, []byte(`{"schemaVersion":"2","kind":"case","id":"view-change-ready","chainPreset":"wbft-view-change","requires":["rpc"],"steps":[{"do":"waitBlock","target":12,"timeout":"90s"},{"expect":"blockNumber","compare":"GreaterOrEqual","is":"12"}]}`), 0600); err != nil {
		t.Fatal(err)
	}
	specs, err := dsl.ReadFilesWithChainPreset([]string{casePath}, envPath)
	if err != nil {
		t.Fatal(err)
	}
	out, runErr := testengine.RunSuite(ctx, chainsetup.Deps{}, testengine.RunSuiteIn{SpecContent: specs, DataDir: data, ArtifactRoot: filepath.Join(root, "reports"), Binary: bin, KeysDir: filepath.Join(repoRoot(t), "presets/keys"), ReadOnlyKeys: true, KeepUp: true, WaitBlocks: 2})
	p := viewProof{RequestTimeout: 3, BlockPeriod: 2}
	proof := map[string]any{"session": out.SessionRoot, "summary": out.Summary, "scope": "WBFT-003/004 managed local Croissant0 four validators, round-zero next proposer selected from actual header/order, stopped, round change, original node restarted, five post-restart links. Not complete original migration/Linux/mainnet/legacy/final Web acceptance."}
	defer func() {
		proof["view"] = p
		b, err := json.MarshalIndent(proof, "", "  ")
		if err != nil {
			t.Error(err)
			return
		}
		if err = os.WriteFile(filepath.Join(root, "reports/view-change-proof.json"), b, 0600); err != nil {
			t.Error(err)
		}
	}()
	if runErr != nil || out.Summary.Summary.Pass != 1 || out.Summary.Failed() || out.Summary.Summary.Skip != 0 {
		t.Fatal("not ready", runErr, out.Summary)
	}
	state, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: data})
	if err != nil || len(state.Nodes.Nodes) != 4 {
		t.Fatal("not four validators", err)
	}
	var actualConfig map[string]any
	var genesisPath string
	err = filepath.WalkDir(data, func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() && (d.Name() == "chaindata" || d.Name() == "lightchaindata" || d.Name() == "etcd") {
			return filepath.SkipDir
		}
		if d.IsDir() || d.Name() != "genesis.json" {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var g struct {
			Config struct {
				Croissant struct {
					WBFT map[string]any `json:"wBFT"`
				} `json:"croissant"`
			}
		}
		if err = json.Unmarshal(b, &g); err != nil {
			return err
		}
		actualConfig = g.Config.Croissant.WBFT
		genesisPath = path
		return fs.SkipAll
	})
	if err != nil || actualConfig == nil || actualConfig["blockPeriodSeconds"] != float64(2) || actualConfig["requestTimeoutSeconds"] != float64(3) || actualConfig["proposerPolicy"] != float64(0) {
		t.Fatal("wrong actual view-change genesis", actualConfig, err)
	}
	proof["genesisPath"] = genesisPath
	proof["wbftConfig"] = actualConfig
	nodeAddresses := map[string]node.Node{}
	for _, n := range state.Nodes.Nodes {
		var addr string
		if err = rpc.Dial(n.RPCURL).Call(ctx, "istanbul_nodeAddress", &addr); err != nil {
			t.Fatal(err)
		}
		nodeAddresses[strings.ToLower(addr)] = n
	}
	c := rpc.Dial(state.Nodes.Nodes[0].RPCURL)
	initial, err := c.BlockNumber(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for {
		head, err := c.BlockNumber(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if head > initial && head%10 != 0 {
			p.Before, err = readViewHeader(ctx, c, head)
			if err != nil {
				t.Fatal(err)
			}
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal(ctx.Err())
		case <-time.After(100 * time.Millisecond):
		}
	}
	if p.Before.Round != 0 {
		t.Fatal("selection head not healthy round0")
	}
	if err = c.Call(ctx, "istanbul_getValidators", &p.Validators, fmt.Sprintf("0x%x", p.Before.Number)); err != nil {
		t.Fatal(err)
	}
	index := -1
	for i, a := range p.Validators {
		if sameAddress(a, p.Before.Miner) {
			index = i
		}
	}
	if index < 0 {
		t.Fatal("proposer missing from actual set")
	}
	p.StoppedAddress = p.Validators[(index+1)%len(p.Validators)]
	target, ok := nodeAddresses[strings.ToLower(p.StoppedAddress)]
	if !ok {
		t.Fatal("next proposer not a managed node")
	}
	p.TargetIndex = target.Index
	p.OldPID = target.PID
	for _, n := range state.Nodes.Nodes {
		if n.Index != target.Index {
			c = rpc.Dial(n.RPCURL)
			break
		}
	}
	if err = verb.NodeStop(ctx, chainsetup.Deps{}, verb.NodeStopIn{DataDir: data, Index: target.Index}); err != nil {
		t.Fatal(err)
	}
	stopped, err := verb.NetworkStatus(ctx, chainsetup.Deps{}, verb.NetworkStatusIn{DataDir: data})
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range stopped.Nodes.Nodes {
		if n.Index == target.Index && n.PID != 0 {
			t.Fatal("target still recorded running")
		}
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
	waitHead(p.Before.Number + 2)
	p.Changed, err = readViewHeader(ctx, c, p.Before.Number+1)
	if err != nil {
		t.Fatal(err)
	}
	p.Successor, err = readViewHeader(ctx, c, p.Before.Number+2)
	if err != nil {
		t.Fatal(err)
	}
	started, err := verb.NodeStart(ctx, chainsetup.Deps{}, verb.NodeStartIn{DataDir: data, Index: target.Index})
	if err != nil {
		t.Fatal(err)
	}
	p.NewPID = started.Node.PID
	p.RestartedAddress, err = waitRestartIdentity(ctx, rpc.Dial(started.Node.RPCURL), p.StoppedAddress)
	if err != nil {
		t.Fatal(err)
	}
	anchor, err := c.BlockNumber(ctx)
	if err != nil {
		t.Fatal(err)
	}
	waitHead(anchor + 5)
	for n := anchor; n <= anchor+5; n++ {
		h, err := readViewHeader(ctx, c, n)
		if err != nil {
			t.Fatal(err)
		}
		p.AfterRestart = append(p.AfterRestart, h)
	}
	restarted := rpc.Dial(started.Node.RPCURL)
	for {
		head, err := restarted.BlockNumber(ctx)
		if err == nil && head >= anchor+5 {
			break
		}
		select {
		case <-ctx.Done():
			t.Fatal("restarted proposer did not resync", ctx.Err())
		case <-time.After(200 * time.Millisecond):
		}
	}
	pattern := regexp.MustCompile(fmt.Sprintf(`msg.sequence=%d(?:\s|$)`, p.Changed.Number))
	logs := map[string]any{}
	commitTimes := map[uint64]time.Time{}
	commitLogs := map[string]any{}
	stampPattern := regexp.MustCompile(`\[(\d{2}-\d{2}\|\d{2}:\d{2}:\d{2}\.\d{3})\]`)
	err = filepath.WalkDir(data, func(path string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() && (d.Name() == "chaindata" || d.Name() == "lightchaindata" || d.Name() == "etcd") {
			return filepath.SkipDir
		}
		if d.Name() == fmt.Sprintf("node%d.log", target.Index) {
			return nil
		}
		if d.IsDir() || !strings.HasPrefix(d.Name(), "node") || !strings.HasSuffix(d.Name(), ".log") {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		lines := []string{}
		for _, line := range strings.Split(string(b), "\n") {
			if strings.Contains(line, "WBFT: block proposal committed") {
				for _, h := range []viewHeader{p.Before, p.Changed} {
					heightPattern := regexp.MustCompile(fmt.Sprintf(`number=%d(?:\s|$)`, h.Number))
					if !heightPattern.MatchString(line) {
						continue
					}
					matches := stampPattern.FindStringSubmatch(line)
					if len(matches) != 2 {
						return fmt.Errorf("missing commit log time")
					}
					stamp, e := time.Parse("01-02|15:04:05.000", matches[1])
					if e != nil {
						return e
					}
					previous, ok := commitTimes[h.Number]
					if !ok || stamp.Before(previous) {
						commitTimes[h.Number] = stamp
						commitLogs[fmt.Sprint(h.Number)] = map[string]any{"path": path, "line": line, "timestamp": matches[1]}
					}
				}
			}

			if strings.Contains(line, "handle ROUND-CHANGE message") && pattern.MatchString(line) {
				lines = append(lines, line)
			}
		}
		if len(lines) > 0 {
			logs[d.Name()] = map[string]any{"path": path, "lines": lines}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	p.RoundChangeNodes = len(logs)
	beforeTime, beforeOK := commitTimes[p.Before.Number]
	changedTime, changedOK := commitTimes[p.Changed.Number]
	if !beforeOK || !changedOK {
		t.Fatal("actual generation commit times missing", commitLogs)
	}
	p.CommitLogDelayMillis = changedTime.Sub(beforeTime).Milliseconds()
	proof["commitLogs"] = commitLogs
	proof["roundChangeLogs"] = logs
	if err = verifyViewChange(p); err != nil {
		t.Fatal(err)
	}
}

func waitRestartIdentity(ctx context.Context, c *rpc.Client, expected string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	var last error
	for {
		var address string
		err := c.Call(ctx, "istanbul_nodeAddress", &address)
		if err == nil {
			if !sameAddress(address, expected) {
				return address, fmt.Errorf("restarted node identity differs")
			}
			return address, nil
		}
		last = err
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("restarted node RPC not ready: %w (last: %v)", ctx.Err(), last)
		case <-time.After(100 * time.Millisecond):
		}
	}
}
