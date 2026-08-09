package engine

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/0xmhha/chainbench/internal/core/collector"
)

func TestChainstateSink_WritesJSONL(t *testing.T) {
	path := filepath.Join(t.TempDir(), chainstateFile)
	s, err := newChainstateSink(path)
	if err != nil {
		t.Fatalf("newChainstateSink: %v", err)
	}
	s.write(collector.Chainstate{
		Heights:         map[string]uint64{"node1": 10},
		Peers:           map[string]int{"node1": 3},
		BPParticipation: map[string]int{"0xA": 4},
	})
	s.write(collector.Chainstate{
		Heights: map[string]uint64{"node1": 11},
		Forked:  true,
	})
	if err := s.close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	recs := readChainstate(t, path)
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2", len(recs))
	}
	if recs[0].Seq != 1 || recs[0].Heights["node1"] != 10 || recs[0].BPParticipation["0xA"] != 4 {
		t.Fatalf("record 0 = %+v", recs[0])
	}
	if recs[1].Seq != 2 || recs[1].Heights["node1"] != 11 || !recs[1].Forked {
		t.Fatalf("record 1 = %+v", recs[1])
	}
}

// TestChainstateSink_NilIsNoop guards the nil-sink convenience used when
// persistence is unavailable.
func TestChainstateSink_NilIsNoop(t *testing.T) {
	var s *chainstateSink
	s.write(collector.Chainstate{Heights: map[string]uint64{"node1": 1}})
	if err := s.close(); err != nil {
		t.Fatalf("nil close: %v", err)
	}
}

func readChainstate(t *testing.T, path string) []chainstateRecord {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = f.Close() }()
	var recs []chainstateRecord
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		var r chainstateRecord
		if err := json.Unmarshal(sc.Bytes(), &r); err != nil {
			t.Fatalf("unmarshal %q: %v", sc.Text(), err)
		}
		recs = append(recs, r)
	}
	return recs
}
