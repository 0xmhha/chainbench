package poa

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"testing"
	"time"
)

func sampleMember(boot bool) Member {
	return Member{
		Addr: "0xabc", Stake: big.NewInt(1), Name: "n1",
		ID: "0x" + strings.Repeat("a", 128), IP: "127.0.0.1", Port: 30010, Bootnode: boot,
	}
}

func TestConfig_JSONAndValidate(t *testing.T) {
	c := Config{
		ExtraData: "x", Staker: "0xs", Ecosystem: "0xe", Maintenance: "0xm", FeeCollector: "0xf",
		Env:      Env{StakingMin: big.NewInt(5), StakingMax: big.NewInt(9), RewardDistribution: []int{4000, 1000, 2500, 2500}},
		Members:  []Member{sampleMember(true)},
		Accounts: []Account{{Addr: "0xabc", Balance: new(big.Int).Exp(big.NewInt(10), big.NewInt(27), nil)}},
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}
	b, err := c.JSON()
	if err != nil {
		t.Fatal(err)
	}
	// big.Int must serialize as a JSON number, not a string, for gwemix.
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if !strings.Contains(string(b), `"stakingMin": 5`) {
		t.Errorf("stakingMin should be a numeric field:\n%s", b)
	}
	if !strings.Contains(string(b), "1000000000000000000000000000") {
		t.Errorf("big account balance should serialize as a number:\n%s", b)
	}
}

func TestConfig_ValidateRejects(t *testing.T) {
	if err := (Config{}).Validate(); err == nil {
		t.Error("no members should error")
	}
	// two bootnodes
	c := Config{Members: []Member{sampleMember(true), sampleMember(true)}}
	if err := c.Validate(); err == nil || !strings.Contains(err.Error(), "exactly one bootnode") {
		t.Errorf("two bootnodes should error, got %v", err)
	}
	// short id
	bad := sampleMember(true)
	bad.ID = "0xdead"
	if err := (Config{Members: []Member{bad}}).Validate(); err == nil || !strings.Contains(err.Error(), "128-hex") {
		t.Errorf("short id should error, got %v", err)
	}
}

func TestBootstrapExec_Commands(t *testing.T) {
	var calls [][]string
	r := func(_ context.Context, name string, args ...string) ([]byte, error) {
		calls = append(calls, append([]string{name}, args...))
		return nil, nil
	}
	ctx := context.Background()
	if err := GenerateGenesis(ctx, r, "gwemix", "/c.json", "/t.json", "/g.json"); err != nil {
		t.Fatal(err)
	}
	if err := DeployGovernance(ctx, r, "gwemix", "/ipc", "/c.json", "/ks", "/pw"); err != nil {
		t.Fatal(err)
	}
	if err := EtcdInit(ctx, r, "gwemix", "/ipc"); err != nil {
		t.Fatal(err)
	}
	joined := func(i int) string { return strings.Join(calls[i], " ") }
	if !strings.Contains(joined(0), "wemix genesis --data /c.json --genesis /t.json --out /g.json") {
		t.Errorf("genesis cmd: %s", joined(0))
	}
	// deploy-governance must use the 2-arg form (config + keystore, NO lockAmount)
	dg := joined(1)
	if !strings.Contains(dg, "wemix deploy-governance --url /ipc --password /pw /c.json /ks") {
		t.Errorf("deploy-governance cmd: %s", dg)
	}
	if strings.HasSuffix(dg, "/ks 0") || strings.Contains(dg, "STAKING") {
		t.Errorf("deploy-governance must NOT pass a lockAmount (3-arg form is buggy): %s", dg)
	}
	if !strings.Contains(joined(2), "attach /ipc --exec admin.etcdInit()") {
		t.Errorf("etcdInit cmd: %s", joined(2))
	}
}

// TestEtcdInit_ARefusalIsNotSuccess.
//
// admin.etcdInit() refuses with "not running" while the node has not yet read
// the governance contract that tells it which member it is. The console prints
// that rather than failing the process, so a caller that only checks the
// process status is told the cluster was formed when nothing was. Measured: one
// handoff in four deployed governance, initialized nothing, and stalled with an
// empty cluster and no error anywhere.
func TestEtcdInit_ARefusalIsNotSuccess(t *testing.T) {
	run := func(context.Context, string, ...string) ([]byte, error) {
		return []byte("Error: not running\n\tat web3.js:6373:9(45)"), nil
	}
	err := EtcdInit(context.Background(), run, "gwemix", "/tmp/x.ipc")
	if err == nil {
		t.Fatal("a refusal printed to the console was reported as success")
	}
	if !strings.Contains(err.Error(), "not running") {
		t.Fatalf("the error does not carry what the node said: %v", err)
	}
}

// TestEtcdInit_SuccessIsQuiet: a working init prints null, which is not an
// error and must not be read as one.
func TestEtcdInit_SuccessIsQuiet(t *testing.T) {
	run := func(context.Context, string, ...string) ([]byte, error) { return []byte("null"), nil }
	if err := EtcdInit(context.Background(), run, "gwemix", "/tmp/x.ipc"); err != nil {
		t.Fatalf("a successful init was reported as a failure: %v", err)
	}
}

// TestWaitSelf_WaitsForTheNodeToFindItself: the window closes when the node's
// governance refresh lands, so the wait is on the state and not on a duration.
func TestWaitSelf_WaitsForTheNodeToFindItself(t *testing.T) {
	calls := 0
	run := func(context.Context, string, ...string) ([]byte, error) {
		calls++
		if calls < 2 {
			return []byte(`"{\"self\":{\"name\":\"\"},\"miners\":\"\"}"`), nil
		}
		return []byte(`"{\"self\":{\"name\":\"node1\"},\"miners\":\"node1/up\"}"`), nil
	}
	if err := WaitSelf(context.Background(), run, "gwemix", "/tmp/x.ipc", 10*time.Second); err != nil {
		t.Fatalf("WaitSelf: %v", err)
	}
	if calls < 2 {
		t.Fatalf("returned after %d call(s); it must keep asking until self is known", calls)
	}
}
