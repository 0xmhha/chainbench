package testengine_test

import (
	"fmt"
	"strings"
	"testing"
)

type restartSignerProof struct {
	Number       uint64
	Hash, Author string
	Committers   []string
}
type restartWindowSample struct {
	Header  viewHeader
	Signers restartSignerProof
}
type restartWindowProof struct {
	Validators []string
	Restarted  string
	SyncHeight uint64
	Samples    []restartWindowSample
}

// verifyRestartWindow requires five individual healthy one-second links and
// actual commit participation, tied to each queried header rather than an API
// identity or an average interval.
func verifyRestartWindow(p restartWindowProof) error {
	if len(p.Samples) != 6 || len(p.Validators) < 4 || p.SyncHeight == 0 {
		return fmt.Errorf("missing recovery window/context")
	}
	allowed := map[string]bool{}
	for _, a := range p.Validators {
		key := strings.ToLower(a)
		if key == "" || allowed[key] {
			return fmt.Errorf("invalid validator set")
		}
		allowed[key] = true
	}
	if !allowed[strings.ToLower(p.Restarted)] {
		return fmt.Errorf("restarted identity not in validator set")
	}
	participated := false
	for i, s := range p.Samples {
		h := s.Header
		if h.Number <= p.SyncHeight || h.Hash == "" || h.Timestamp == 0 || h.Round != 0 || !allowed[strings.ToLower(h.Miner)] || i > 0 && (h.Number != p.Samples[i-1].Header.Number+1 || h.ParentHash != p.Samples[i-1].Header.Hash || h.Timestamp != p.Samples[i-1].Header.Timestamp+1) {
			return fmt.Errorf("not five healthy one-second post-sync links")
		}
		if s.Signers.Number != h.Number || s.Signers.Hash != h.Hash || !sameAddress(s.Signers.Author, h.Miner) {
			return fmt.Errorf("commit signers not bound to header")
		}
		seen := map[string]bool{}
		for _, a := range s.Signers.Committers {
			key := strings.ToLower(a)
			if !allowed[key] || seen[key] {
				return fmt.Errorf("invalid/duplicate committer")
			}
			seen[key] = true
			if sameAddress(a, p.Restarted) {
				participated = true
			}
		}
		if len(seen) < 2*len(allowed)/3+1 {
			return fmt.Errorf("not a commit quorum")
		}
	}
	if !participated {
		return fmt.Errorf("restarted validator never committed")
	}
	return nil
}

func restartWindowFixture() restartWindowProof {
	p := restartWindowProof{Validators: []string{"v0", "v1", "v2", "v3"}, Restarted: "v2", SyncHeight: 99}
	for n := uint64(100); n <= 105; n++ {
		h := viewHeader{Number: n, Timestamp: 1000 + n - 100, Hash: fmt.Sprint(n), ParentHash: fmt.Sprint(n - 1), Miner: "v0"}
		p.Samples = append(p.Samples, restartWindowSample{Header: h, Signers: restartSignerProof{Number: n, Hash: h.Hash, Author: h.Miner, Committers: []string{"v0", "v1", "v2", "v3"}}})
	}
	return p
}

func TestRestartWindowRejectsAverageAndIdentityOnlyProof(t *testing.T) {
	if err := verifyRestartWindow(restartWindowFixture()); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*restartWindowProof)
	}{
		{"average one but individual gaps wrong", func(p *restartWindowProof) { p.Samples[1].Header.Timestamp++ }},
		{"view change in recovery window", func(p *restartWindowProof) { p.Samples[2].Header.Round = 1 }},
		{"window before completed sync", func(p *restartWindowProof) { p.SyncHeight = 100 }},
		{"wrong restarted identity", func(p *restartWindowProof) { p.Restarted = "outsider" }},
		{"original identity never signs", func(p *restartWindowProof) {
			for i := range p.Samples {
				p.Samples[i].Signers.Committers = []string{"v0", "v1", "v3"}
			}
		}},
		{"wrong signer block", func(p *restartWindowProof) { p.Samples[2].Signers.Number++ }},
		{"wrong signer hash", func(p *restartWindowProof) { p.Samples[2].Signers.Hash = "other" }},
		{"wrong signer author", func(p *restartWindowProof) { p.Samples[2].Signers.Author = "v1" }},
		{"duplicate committers", func(p *restartWindowProof) { p.Samples[2].Signers.Committers = []string{"v0", "v0", "v2"} }},
		{"unknown committer", func(p *restartWindowProof) { p.Samples[2].Signers.Committers = []string{"v0", "v1", "outsider"} }},
		{"not a quorum", func(p *restartWindowProof) { p.Samples[2].Signers.Committers = []string{"v0", "v2"} }},
		{"short window", func(p *restartWindowProof) { p.Samples = p.Samples[:5] }},
		{"wrong link", func(p *restartWindowProof) { p.Samples[2].Header.ParentHash = "other" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := restartWindowFixture()
			tc.change(&p)
			if verifyRestartWindow(p) == nil {
				t.Fatal("consecutive links hid incorrect restart recovery")
			}
		})
	}
}
