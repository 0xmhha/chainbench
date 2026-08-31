package collector

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
	"time"
)

func TestBus_PublishDelivers(t *testing.T) {
	b := NewBus()
	defer b.Close()
	sub := b.Subscribe()

	b.Publish(Event{Phase: PhaseSetup, Kind: KindInfo, Message: "hello"})

	select {
	case e := <-sub:
		if e.Message != "hello" || e.Phase != PhaseSetup {
			t.Errorf("got %+v", e)
		}
		if e.Time.IsZero() {
			t.Error("bus should stamp zero Time")
		}
	case <-time.After(time.Second):
		t.Fatal("no event delivered")
	}
}

func TestBus_SubscribeAfterEventMissesIt(t *testing.T) {
	b := NewBus()
	defer b.Close()
	b.Publish(Event{Message: "early"})
	sub := b.Subscribe()
	select {
	case e := <-sub:
		t.Fatalf("late subscriber should not see prior event, got %q", e.Message)
	case <-time.After(50 * time.Millisecond):
		// expected: no delivery
	}
}

func TestBus_CloseClosesSubscribers(t *testing.T) {
	b := NewBus()
	sub := b.Subscribe()
	b.Close()
	if _, ok := <-sub; ok {
		t.Error("subscriber channel should be closed after Close")
	}
	// Publish after close is a no-op, not a panic.
	b.Publish(Event{Message: "x"})
}

func TestMemStore_SaveGetList(t *testing.T) {
	s := NewMemStore()
	t0 := time.Unix(100, 0)
	t1 := time.Unix(200, 0)
	_ = s.SaveRun(RunRecord{ID: "b", Phase: PhaseVerify, StartedAt: t1, Status: RunRunning})
	_ = s.SaveRun(RunRecord{ID: "a", Phase: PhaseSetup, StartedAt: t0, Status: RunSucceeded})

	r, ok := s.GetRun("a")
	if !ok || r.Status != RunSucceeded {
		t.Fatalf("GetRun(a): %+v ok=%v", r, ok)
	}
	if _, ok := s.GetRun("missing"); ok {
		t.Error("GetRun(missing) should be false")
	}

	list := s.ListRuns()
	if len(list) != 2 || list[0].ID != "a" || list[1].ID != "b" {
		t.Errorf("ListRuns not sorted by StartedAt: %+v", list)
	}

	// Replace semantics.
	_ = s.SaveRun(RunRecord{ID: "a", Phase: PhaseSetup, StartedAt: t0, Status: RunFailed})
	if r, _ := s.GetRun("a"); r.Status != RunFailed {
		t.Errorf("SaveRun should replace: got %v", r.Status)
	}
}

func TestLogEvent_JSON(t *testing.T) {
	var buf bytes.Buffer
	l := NewLogger(&buf, slog.LevelInfo)
	LogEvent(l, Event{
		Phase: PhaseTest, Kind: KindResult, Node: 3,
		Message: "done", Fields: map[string]any{"blocks": 10},
	})
	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatalf("log line not JSON: %v (%s)", err, buf.String())
	}
	if rec["msg"] != "done" || rec["phase"] != "test" || rec["kind"] != "result" {
		t.Errorf("unexpected log record: %v", rec)
	}
	if rec["node"] != float64(3) || rec["blocks"] != float64(10) {
		t.Errorf("structured fields missing/wrong: %v", rec)
	}
}
