package session

import (
	"errors"
	"io"
	"testing"
)

func TestObservationStoreAppendsRecordsAndConfinesNames(t *testing.T) {
	s, err := OpenObservationStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"a\n", "b\n"} {
		if err = s.Append("net-1", "logs/node1.jsonl", []byte(line)); err != nil {
			t.Fatal(err)
		}
	}
	r, err := s.Open("net-1", "logs/node1.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(r)
	_ = r.Close()
	if string(b) != "a\nb\n" {
		t.Fatalf("appended = %q", b)
	}
	if r, err = s.Open("net-1", "samples.jsonl"); err != nil {
		t.Fatal(err)
	} else if b, _ = io.ReadAll(r); len(b) != 0 {
		t.Fatalf("unwritten record = %q", b)
	}
	if cursor, err := s.ReadCursor("net-1"); err != nil || cursor != nil {
		t.Fatalf("first cursor = %q, %v", cursor, err)
	}
	if err = s.WriteCursor("net-1", []byte(`{"offsets":{}}`)); err != nil {
		t.Fatal(err)
	}
	if cursor, _ := s.ReadCursor("net-1"); string(cursor) != `{"offsets":{}}` {
		t.Fatalf("cursor = %q", cursor)
	}
	for _, bad := range [][2]string{{"..", "samples.jsonl"}, {"net/1", "samples.jsonl"}, {"net-1", "../x"}, {"net-1", "logs/../x"}, {"net-1", "other/x"}, {"net-1", ".hidden"}} {
		if err := s.Append(bad[0], bad[1], []byte("x\n")); !errors.Is(err, ErrObservationName) {
			t.Errorf("%q %q accepted: %v", bad[0], bad[1], err)
		}
	}
}
