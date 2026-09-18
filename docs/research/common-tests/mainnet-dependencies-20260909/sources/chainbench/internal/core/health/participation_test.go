package health_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/health"
)

// chain is a sealer log: block n was sealed by seals[n].
type chain struct {
	seals    []string
	failFrom int // blocks below this index cannot be read (0 = all readable)
}

func (c chain) HeadBlock(ctx context.Context) (uint64, string, string, error) {
	n := uint64(len(c.seals) - 1)
	return n, "0xhead", c.seals[n], nil
}

func (c chain) BlockMiner(ctx context.Context, n uint64) (string, error) {
	if c.failFrom > 0 && int(n) < c.failFrom {
		return "", fmt.Errorf("block %d is gone", n)
	}
	return c.seals[n], nil
}

// roundRobin builds a chain where n validators take turns for rounds rounds.
func roundRobin(vals []string, rounds int) chain {
	var seals []string
	for i := 0; i < rounds*len(vals); i++ {
		seals = append(seals, vals[i%len(vals)])
	}
	return chain{seals: seals}
}

var vals = []string{"0xAAA", "0xBBB", "0xCCC", "0xDDD"}

// TestParticipants_FindsTheValidatorThatNeverSeals is the whole point.
//
// A four-validator BFT network keeps producing with three. The fourth is up, in
// sync, reporting the same head as everyone else, and cannot get a transaction
// mined — the only symptom being a receipt that never arrives half a minute
// later. Asking one node whether its head grew cannot see that; asking who
// sealed can.
func TestParticipants_FindsTheValidatorThatNeverSeals(t *testing.T) {
	c := roundRobin([]string{"0xAAA", "0xBBB", "0xCCC"}, 4) // 0xDDD never seals
	p, err := health.Participants(context.Background(), c, vals, health.WindowFor(len(vals)))
	if err != nil {
		t.Fatal(err)
	}
	if p.Formed() {
		t.Error("a network with a silent validator was reported formed")
	}
	if len(p.Silent) != 1 || p.Silent[0] != "0xddd" {
		t.Errorf("silent = %v, want just 0xddd", p.Silent)
	}
	if !strings.Contains(p.Describe(), "0xddd") {
		t.Errorf("description %q does not name the silent validator", p.Describe())
	}
}

// TestParticipants_AHealthyNetworkIsFormed: every validator taking its turn is
// the normal case and must not be reported as a problem.
func TestParticipants_AHealthyNetworkIsFormed(t *testing.T) {
	p, err := health.Participants(context.Background(), roundRobin(vals, 4), vals, health.WindowFor(len(vals)))
	if err != nil {
		t.Fatal(err)
	}
	if !p.Formed() {
		t.Errorf("a healthy network was reported unformed: %s", p.Describe())
	}
	for _, v := range vals {
		if p.Sealed[strings.ToLower(v)] == 0 {
			t.Errorf("%s sealed nothing in the window", v)
		}
	}
}

// TestParticipants_WindowCoversMoreThanOneRound: with a window of one round a
// validator that simply has not had its turn yet looks silent, and a readiness
// check that cries wolf is one people stop waiting for.
func TestParticipants_WindowCoversMoreThanOneRound(t *testing.T) {
	c := roundRobin(vals, 4)
	if p, _ := health.Participants(context.Background(), c, vals, health.WindowFor(len(vals))); !p.Formed() {
		t.Errorf("two rounds should cover every validator: %s", p.Describe())
	}
	if health.WindowFor(len(vals)) <= len(vals) {
		t.Errorf("WindowFor(%d) = %d, which is one round or less", len(vals), health.WindowFor(len(vals)))
	}
}

// TestParticipants_SaysNothingWhenTheChainNamesNoSealer: a chain without a
// sealer in its header gets no verdict rather than a guessed one.
func TestParticipants_SaysNothingWhenTheChainNamesNoSealer(t *testing.T) {
	c := chain{seals: []string{"", "", ""}}
	p, err := health.Participants(context.Background(), c, vals, 4)
	if err != nil {
		t.Fatal(err)
	}
	if p.Known {
		t.Error("a chain with no sealer field produced a verdict")
	}
	if !p.Formed() {
		t.Error("an unjudgeable chain was reported unformed")
	}
	if !strings.Contains(p.Describe(), "unknown") {
		t.Errorf("description %q does not say the answer is unknown", p.Describe())
	}
}

// TestParticipants_AnUnreadableBlockShrinksTheWindow: a gap in history makes the
// check more cautious, never less. Treating an unreadable block as "nobody
// sealed it" would invent a silent validator.
func TestParticipants_AnUnreadableBlockShrinksTheWindow(t *testing.T) {
	c := roundRobin(vals, 4)
	c.failFrom = len(c.seals) - 3 // only the newest three readable
	p, err := health.Participants(context.Background(), c, vals, health.WindowFor(len(vals)))
	if err != nil {
		t.Fatal(err)
	}
	if p.Window >= health.WindowFor(len(vals)) {
		t.Errorf("window = %d, want it shrunk below %d", p.Window, health.WindowFor(len(vals)))
	}
}
