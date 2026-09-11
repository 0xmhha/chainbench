package upgrade_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/consensus/upgrade"
	"github.com/0xmhha/chainbench/internal/core/node"
)

// AwaitFork is the assertion the whole handoff exists to make: after the fork
// block, the successor set produces. It used to read ONE block and accept any
// sealer that was not the producer, so a third address passed and a successor
// that sealed once and stopped passed too. These tests hold the stronger claim,
// and each of them fails against the old behavior.

// forkChain serves eth_blockNumber and eth_getBlockByNumber for a chain whose
// post-fork blocks are sealed by the addresses given, cycling through them.
func forkChain(t *testing.T, head uint64, producer string, forkBlock uint64, postForkSealers []string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
			Params []any  `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("bad request: %v", err)
			return
		}
		reply := func(v any) {
			_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": v})
		}
		switch req.Method {
		case "eth_blockNumber":
			reply(fmt.Sprintf("0x%x", head))
		case "eth_getBlockByNumber":
			raw, _ := req.Params[0].(string)
			n, err := strconv.ParseUint(strings.TrimPrefix(raw, "0x"), 16, 64)
			if err != nil {
				t.Errorf("bad block number %q", raw)
				return
			}
			miner := producer
			if n > forkBlock && len(postForkSealers) > 0 {
				miner = postForkSealers[int(n-forkBlock-1)%len(postForkSealers)]
			}
			reply(map[string]any{"miner": miner, "number": raw})
		default:
			t.Errorf("unexpected method %q", req.Method)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// handoffAt builds a Handoff whose plan declares the given successor set and
// fork block, which is all AwaitFork reads.
func handoffAt(t *testing.T, forkBlock int64, successors []string) *upgrade.Handoff {
	t.Helper()
	h, err := upgrade.NewHandoff(handoffInputs(t))
	if err != nil {
		t.Fatalf("NewHandoff: %v", err)
	}
	h.Profile.Upgrade.ForkBlock = forkBlock
	h.Plan.Network.WbftValidators = successors
	// A plan with one producer and one successor, so AwaitFork picks its
	// observation target the way it does in a real run rather than falling back.
	h.Plan.Nodes = []upgrade.NodeSpec{
		{Index: 0, Producer: true},
		{Index: 1, Producer: false},
	}
	return h
}

func nodeSetAt(url string) node.NodeSet {
	return node.NodeSet{Nodes: []node.Node{
		{Index: 0, RPCURL: "http://producer.invalid"},
		{Index: 1, RPCURL: url},
	}}
}

const (
	testProducer = "0xf9593d358b373d354a348c00887b914b408f6984"
	valA         = "0xc17d493883eaa3b4cceb0f214b273392d562f9d8"
	valB         = "0x2493a84a8f83cb87fdcbe0bb3b2d313f69a58d3c"
	valC         = "0x8eb79036bc0f3aba136ef18b3a2fb8c1188939a6"
)

// TestAwaitFork_ConfirmsWhenTheSuccessorSetKeepsProducing is the passing case:
// every block after the fork is sealed by a declared validator, and more than
// one of them takes a turn.
func TestAwaitFork_ConfirmsWhenTheSuccessorSetKeepsProducing(t *testing.T) {
	successors := []string{valA, valB, valC}
	url := forkChain(t, 40, testProducer, 20, successors)
	h := handoffAt(t, 20, successors)

	detail, err := h.AwaitFork(context.Background(), nodeSetAt(url), 5*time.Second)
	if err != nil {
		t.Fatalf("AwaitFork: %v", err)
	}
	for _, want := range []string{"blocks 21-30", "successor set", "3 of 3 validator(s)"} {
		if !strings.Contains(detail, want) {
			t.Errorf("detail %q is missing %q", detail, want)
		}
	}
}

// TestAwaitFork_RefusesAnUndeclaredSealer is the case the old check let through.
// The sealer is not the producer, so "not the producer" was satisfied — but it is
// not in the successor set either, which means production did not move where the
// handoff says it moved.
func TestAwaitFork_RefusesAnUndeclaredSealer(t *testing.T) {
	const stranger = "0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	url := forkChain(t, 40, testProducer, 20, []string{stranger})
	h := handoffAt(t, 20, []string{valA, valB, valC})

	_, err := h.AwaitFork(context.Background(), nodeSetAt(url), 5*time.Second)
	if err == nil {
		t.Fatal("a sealer outside the successor set was accepted")
	}
	if !strings.Contains(err.Error(), stranger) || !strings.Contains(err.Error(), "not a declared successor") {
		t.Errorf("the refusal should name the sealer and why: %v", err)
	}
}

// TestAwaitFork_RefusesASingleSealThenTheProducerAgain covers the other half the
// old check missed: the successor seals the boundary block and the producer takes
// the next one back. Reading only block fork+1 called that a handoff.
func TestAwaitFork_RefusesASingleSealThenTheProducerAgain(t *testing.T) {
	// Block 21 -> valA, block 22 -> the producer, alternating.
	url := forkChain(t, 40, testProducer, 20, []string{valA, testProducer})
	h := handoffAt(t, 20, []string{valA, valB, valC})

	_, err := h.AwaitFork(context.Background(), nodeSetAt(url), 5*time.Second)
	if err == nil {
		t.Fatal("production returning to the producer after one block was accepted")
	}
	if !strings.Contains(err.Error(), "block 22") {
		t.Errorf("the refusal should name the block that went back: %v", err)
	}
}

// TestAwaitFork_RefusesAStalledHead keeps the timeout behavior: a chain that
// never reaches far enough past the fork is a failure that says how far it got.
func TestAwaitFork_RefusesAStalledHead(t *testing.T) {
	url := forkChain(t, 21, testProducer, 20, []string{valA})
	h := handoffAt(t, 20, []string{valA, valB, valC})

	_, err := h.AwaitFork(context.Background(), nodeSetAt(url), 1500*time.Millisecond)
	if err == nil {
		t.Fatal("a head one block past the fork was accepted as a confirmed handoff")
	}
	if !strings.Contains(err.Error(), "stalled at 21") {
		t.Errorf("the refusal should say how far the head got: %v", err)
	}
}

// TestAwaitFork_RefusesAPlanWithNoSuccessors guards the check itself: an empty
// successor set would make every sealer "declared" by vacuous truth.
func TestAwaitFork_RefusesAPlanWithNoSuccessors(t *testing.T) {
	url := forkChain(t, 40, testProducer, 20, []string{valA})
	h := handoffAt(t, 20, nil)

	_, err := h.AwaitFork(context.Background(), nodeSetAt(url), time.Second)
	if err == nil {
		t.Fatal("a plan naming no successors confirmed a handoff")
	}
	if !strings.Contains(err.Error(), "no successor validators") {
		t.Errorf("the refusal should name what is missing: %v", err)
	}
}

// valSet returns n distinct lowercase addresses, for the cases whose subject is
// the SIZE of the successor set rather than any particular member.
func valSet(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("0x%040x", i+1)
	}
	return out
}

// TestAwaitFork_WidensTheWindowWithTheSuccessorSet is why the window is no longer
// a constant. Ten blocks can show at most ten sealers, so a fifteen-validator
// handoff could only ever report "10 of 15" — the report was bounded by the
// window, not by what the chain did. The window now scales with the set, so a
// rotation across all fifteen is visible as all fifteen.
func TestAwaitFork_WidensTheWindowWithTheSuccessorSet(t *testing.T) {
	successors := valSet(15)
	url := forkChain(t, 60, testProducer, 20, successors)
	h := handoffAt(t, 20, successors)

	detail, err := h.AwaitFork(context.Background(), nodeSetAt(url), 5*time.Second)
	if err != nil {
		t.Fatalf("AwaitFork: %v", err)
	}
	// 2 x 15 validators = blocks 21-50, giving every member two turns.
	for _, want := range []string{"blocks 21-50", "15 of 15 validator(s)"} {
		if !strings.Contains(detail, want) {
			t.Errorf("detail %q is missing %q", detail, want)
		}
	}
}

// TestAwaitFork_RefusesOneValidatorSealingTheWholeWindow is the claim "every
// block belongs to the successor set" does not make. One member sealing all of
// them satisfies it while production has moved to a single node, not to the set —
// which is indistinguishable from the pre-fork situation with a different
// address. A quorum of the set has to take a turn.
func TestAwaitFork_RefusesOneValidatorSealingTheWholeWindow(t *testing.T) {
	successors := valSet(15)
	url := forkChain(t, 60, testProducer, 20, successors[:1])
	h := handoffAt(t, 20, successors)

	_, err := h.AwaitFork(context.Background(), nodeSetAt(url), 5*time.Second)
	if err == nil {
		t.Fatal("one validator sealing every block after the fork was accepted as the set producing")
	}
	for _, want := range []string{"only 1 of 15", "rotating across the set"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal should say how few sealed and why that is not enough: %v", err)
		}
	}
}

// TestAwaitFork_AcceptsAQuorumShortOfTheWholeSet pins the floor from the other
// side: requiring all n would fail on a single round change, which is normal wbft
// operation and not a failed handoff. A quorum of fifteen is eleven.
func TestAwaitFork_AcceptsAQuorumShortOfTheWholeSet(t *testing.T) {
	successors := valSet(15)
	url := forkChain(t, 60, testProducer, 20, successors[:11])
	h := handoffAt(t, 20, successors)

	detail, err := h.AwaitFork(context.Background(), nodeSetAt(url), 5*time.Second)
	if err != nil {
		t.Fatalf("eleven of fifteen sealing is a quorum and should confirm: %v", err)
	}
	if !strings.Contains(detail, "11 of 15 validator(s)") {
		t.Errorf("detail %q should report the eleven that took a turn", detail)
	}
}
