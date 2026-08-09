package engine

import (
	"encoding/json"
	"os"

	"github.com/0xmhha/chainbench/internal/core/collector"
)

// chainstateFile is the session file that collects periodic chainstate samples,
// one JSON object per line (jsonl), so the dashboard can replay a completed run.
const chainstateFile = "chainstate.jsonl"

// chainstateRecord is one persisted chainstate sample. Seq is a 1-based sample
// counter giving the samples an order without depending on wall-clock time.
type chainstateRecord struct {
	Seq             int               `json:"seq"`
	Heights         map[string]uint64 `json:"heights,omitempty"`
	Peers           map[string]int    `json:"peers,omitempty"`
	BPParticipation map[string]int    `json:"bpParticipation,omitempty"`
	Forked          bool              `json:"forked"`
}

// chainstateSink appends chainstate samples as JSON lines to a file. It is used
// by a single goroutine at a time (the collection loop, then the stop path after
// that loop has exited), so it needs no locking.
type chainstateSink struct {
	f   *os.File
	enc *json.Encoder
	seq int
}

// newChainstateSink creates (truncating) the jsonl file at path.
func newChainstateSink(path string) (*chainstateSink, error) {
	f, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	return &chainstateSink{f: f, enc: json.NewEncoder(f)}, nil
}

// write appends one chainstate sample as a JSON line. A nil sink is a no-op so
// callers need not branch when persistence is unavailable.
func (s *chainstateSink) write(cs collector.Chainstate) {
	if s == nil {
		return
	}
	s.seq++
	_ = s.enc.Encode(chainstateRecord{
		Seq:             s.seq,
		Heights:         cs.Heights,
		Peers:           cs.Peers,
		BPParticipation: cs.BPParticipation,
		Forked:          cs.Forked,
	})
}

// close closes the underlying file. A nil sink is a no-op.
func (s *chainstateSink) close() error {
	if s == nil {
		return nil
	}
	return s.f.Close()
}
