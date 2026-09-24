package statemachine

import "time"

// defaultLogSize is how many messages the ring keeps.
//
// Twenty is the reference's number. It is meant to answer "what happened just
// before this went wrong", not to be a history: a composition is nine stages,
// so twenty covers the stage that failed and the ones around it.
const defaultLogSize = 20

// LogRec is one message's passage through the machine.
//
// Processed and Original differ when a parent handled what a leaf did not, and
// that difference is usually the answer to "why did nothing happen": the
// message went up past the state that was supposed to care. Processed is empty
// when nobody handled it at all, and Dest is empty when the message caused no
// move.
type LogRec struct {
	Time      time.Time
	What      What
	Processed StateName
	Original  StateName
	Dest      StateName
}

// ring is a fixed-size buffer that overwrites its oldest entry.
type ring struct {
	recs  []LogRec
	next  int
	wrote int
}

// newRing returns a ring holding at most size records.
func newRing(size int) *ring {
	return &ring{recs: make([]LogRec, size)}
}

// add writes one record, overwriting the oldest once the ring is full.
func (r *ring) add(rec LogRec) {
	r.recs[r.next] = rec
	r.next = (r.next + 1) % len(r.recs)
	r.wrote++
}

// all returns the records it still holds, oldest first.
func (r *ring) all() []LogRec {
	n := r.wrote
	if n > len(r.recs) {
		n = len(r.recs)
	}
	out := make([]LogRec, 0, n)
	start := (r.next - n + len(r.recs)) % len(r.recs)
	for i := 0; i < n; i++ {
		out = append(out, r.recs[(start+i)%len(r.recs)])
	}
	return out
}
