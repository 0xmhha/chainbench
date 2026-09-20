package health

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Sealer reads who sealed a block. It is a separate interface from Prober
// because a chain that does not name a sealer per block simply does not supply
// one, and the check then reports that it could not tell rather than guessing.
type Sealer interface {
	// HeadBlock returns the current height, its hash and the address that
	// sealed it.
	HeadBlock(ctx context.Context) (number uint64, hash, miner string, err error)
	// BlockMiner returns the address that sealed block number n.
	BlockMiner(ctx context.Context, n uint64) (string, error)
}

// Participation is which validators have actually sealed a block recently.
//
// It exists because "the chain is producing" was answered by asking ONE node
// whether its head grew, and a BFT network of four validators keeps producing
// with three. The fourth is then up, in sync, reporting the same head as
// everyone else, and unable to get a transaction mined — and the only symptom
// is a receipt that never arrives, thirty seconds later, naming nothing.
//
// Measured 2026-09-07 on TestE2E_StablenetProposalExpiry: four failing runs and
// five passing ones separated on exactly this. Every passing run had node1
// sealing between five and twelve blocks; every failing run had it sealing
// none, holding the case's transaction in a pool no other node ever saw.
//
// Waiting on this instead of on "the head moved" took that case from 5 failures
// in 16 runs to 0 in 16, and removing the wait again brought a failure back.
type Participation struct {
	// Window is how many recent blocks were examined.
	Window int
	// Sealed maps a validator's address (lower-case) to how many of those
	// blocks it sealed.
	Sealed map[string]int
	// Silent names the validators that sealed none of them, sorted. A network
	// with a silent validator is up but not formed.
	Silent []string
	// Known is false when the chain does not report a sealer per block, so
	// nothing here is a judgement about the network.
	Known bool
}

// Formed reports whether every validator sealed at least one block in the
// window. A chain that does not name sealers is not judged.
func (p Participation) Formed() bool { return !p.Known || len(p.Silent) == 0 }

// Describe says what was found, for a report or an error.
func (p Participation) Describe() string {
	switch {
	case !p.Known:
		return "sealer participation unknown (this chain does not name a sealer per block)"
	case len(p.Silent) == 0:
		return fmt.Sprintf("every validator sealed within the last %d blocks", p.Window)
	default:
		return fmt.Sprintf("%d of %d validators sealed nothing in the last %d blocks: %s",
			len(p.Silent), len(p.Sealed)+len(p.Silent), p.Window, strings.Join(p.Silent, ", "))
	}
}

// Participants examines the last window blocks and reports which of the
// network's validators sealed one.
//
// The window has to cover more than one round or a validator that simply has
// not had its turn yet looks silent. Callers size it from the validator count;
// twice that is the smallest honest window for a round-robin chain.
func Participants(ctx context.Context, s Sealer, validators []string, window int) (Participation, error) {
	p := Participation{Window: window, Sealed: map[string]int{}}
	if len(validators) == 0 || window <= 0 {
		return p, nil
	}
	head, _, miner, err := s.HeadBlock(ctx)
	if err != nil {
		return p, fmt.Errorf("health: participation: head: %w", err)
	}
	if miner == "" {
		// Not an error: some chains do not put a sealer in the header, and
		// saying "unknown" beats inventing a verdict.
		return p, nil
	}
	p.Known = true

	want := map[string]bool{}
	for _, v := range validators {
		want[strings.ToLower(v)] = true
	}
	count := func(addr string) {
		a := strings.ToLower(addr)
		if want[a] {
			p.Sealed[a]++
		}
	}
	count(miner)
	for i := 1; i < window && uint64(i) <= head; i++ {
		m, err := s.BlockMiner(ctx, head-uint64(i))
		if err != nil {
			// A block that cannot be read shrinks the window rather than
			// failing the check: the question is who HAS sealed, and a gap
			// can only make the answer more cautious.
			p.Window = i
			break
		}
		count(m)
	}
	for a := range want {
		if p.Sealed[a] == 0 {
			p.Silent = append(p.Silent, a)
		}
	}
	sort.Strings(p.Silent)
	return p, nil
}

// WindowFor is the smallest honest window for a network of n validators: two
// full rounds, so a validator that has merely not had its turn is not called
// silent.
func WindowFor(n int) int {
	if n < 1 {
		return 0
	}
	return 2 * n
}
