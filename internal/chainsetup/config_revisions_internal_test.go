package chainsetup

import (
	"testing"
	"time"
)

// TestConfigProvenance_ASwapKeepsTheRevisionItReplaced is R10's other half. The
// requirement is to preserve the fixture, the overrides, the resulting config and
// the node WITH ITS TIME. The writer replaced the node's entry, so a swapNode
// erased the config the node had been composed with — while the state's own
// comment called each write "a new revision" and kept exactly one.
//
// What a run gets asked afterwards is "which config did node N have, and since
// when". One entry with no timestamp answers neither half.
func TestConfigProvenance_ASwapKeepsTheRevisionItReplaced(t *testing.T) {
	at := time.Date(2026, 9, 12, 10, 0, 0, 0, time.UTC)
	w := &Workspace{now: func() time.Time { at = at.Add(time.Minute); return at }}

	w.addConfigProvenance(ConfigProvenance{Node: 1, Overrides: []string{"httpHost=10.0.0.1"}, Checksum: "sha256:aaa"})
	w.addConfigProvenance(ConfigProvenance{Node: 1, Fixture: "config-slow-peer", Overrides: []string{"httpHost=10.0.0.2"}, Checksum: "sha256:bbb"})

	got := w.state.ConfigProvenance
	if len(got) != 2 {
		t.Fatalf("entries = %d, want 2 — the swap replaced the compose revision instead of following it", len(got))
	}
	if got[0].Checksum != "sha256:aaa" || got[1].Checksum != "sha256:bbb" {
		t.Fatalf("revisions are out of order or lost: %+v", got)
	}
	if got[1].Fixture != "config-slow-peer" {
		t.Errorf("the swap's fixture name is not recorded: %+v", got[1])
	}
	for i, p := range got {
		if p.At == "" {
			t.Errorf("revision %d has no time, so two revisions of node1 cannot be ordered", i)
		}
	}
	if got[0].At == got[1].At {
		t.Errorf("both revisions carry %q; the clock was read once for two writes", got[0].At)
	}
}

// TestConfigProvenance_TheTimeIsTheInjectedClock keeps the stamp reproducible. A
// record that reaches for the wall clock cannot be asserted, and this one is.
func TestConfigProvenance_TheTimeIsTheInjectedClock(t *testing.T) {
	fixed := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	w := &Workspace{now: func() time.Time { return fixed }}
	w.addConfigProvenance(ConfigProvenance{Node: 1, Checksum: "sha256:aaa"})

	got := w.state.ConfigProvenance
	if len(got) != 1 {
		t.Fatalf("entries = %d, want 1", len(got))
	}
	if want := fixed.Format(time.RFC3339); got[0].At != want {
		t.Errorf("At = %q, want the injected clock's %q", got[0].At, want)
	}
}

// TestConfigProvenance_AnExplicitTimeIsKept lets a caller that already knows when
// a revision happened say so, instead of the record silently restamping it with
// "now" — which is how a replayed or reconstructed history loses its order.
func TestConfigProvenance_AnExplicitTimeIsKept(t *testing.T) {
	w := &Workspace{now: func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) }}
	w.addConfigProvenance(ConfigProvenance{Node: 1, Checksum: "sha256:aaa", At: "2025-05-05T05:05:05Z"})
	if got := w.state.ConfigProvenance[0].At; got != "2025-05-05T05:05:05Z" {
		t.Errorf("At = %q; an explicit time was overwritten by the clock", got)
	}
}
