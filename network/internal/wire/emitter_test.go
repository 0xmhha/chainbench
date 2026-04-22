package wire

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/network/internal/types"
)

func fixedClock(t time.Time) func() time.Time {
	return func() time.Time { return t }
}

func decodeLine(t *testing.T, line []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(line, &m); err != nil {
		t.Fatalf("line not valid JSON: %v (%q)", err, line)
	}
	return m
}

func TestEmitter_EmitEvent_WritesValidNDJSON(t *testing.T) {
	var buf bytes.Buffer
	e := NewEmitter(&buf)
	e.clock = fixedClock(time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC))
	if err := e.EmitEvent(types.EventName("chain.block"), map[string]any{"height": 42}); err != nil {
		t.Fatalf("emit: %v", err)
	}
	m := decodeLine(t, bytes.TrimSpace(buf.Bytes()))
	if m["type"] != "event" {
		t.Errorf("type: got %v, want event", m["type"])
	}
	if m["name"] != "chain.block" {
		t.Errorf("name: got %v", m["name"])
	}
	if m["ts"] != "2026-04-22T10:00:00Z" {
		t.Errorf("ts: got %v", m["ts"])
	}
	data, ok := m["data"].(map[string]any)
	if !ok || data["height"].(float64) != 42 {
		t.Errorf("data: got %v", m["data"])
	}
}

func TestEmitter_EmitProgress(t *testing.T) {
	var buf bytes.Buffer
	e := NewEmitter(&buf)
	if err := e.EmitProgress("init", 2, 4); err != nil {
		t.Fatalf("emit: %v", err)
	}
	m := decodeLine(t, bytes.TrimSpace(buf.Bytes()))
	if m["type"] != "progress" || m["step"] != "init" {
		t.Errorf("progress mismatch: %v", m)
	}
	if int(m["done"].(float64)) != 2 || int(m["total"].(float64)) != 4 {
		t.Errorf("done/total mismatch: %v", m)
	}
}

func TestEmitter_EmitResult_OK(t *testing.T) {
	var buf bytes.Buffer
	e := NewEmitter(&buf)
	if err := e.EmitResult(true, map[string]any{"blockNumber": "0x2a"}); err != nil {
		t.Fatalf("emit: %v", err)
	}
	m := decodeLine(t, bytes.TrimSpace(buf.Bytes()))
	if m["type"] != "result" || m["ok"] != true {
		t.Errorf("result mismatch: %v", m)
	}
	if _, has := m["error"]; has {
		t.Errorf("ok:true result must not have error field: %v", m)
	}
}

func TestEmitter_EmitResultError(t *testing.T) {
	var buf bytes.Buffer
	e := NewEmitter(&buf)
	if err := e.EmitResultError(types.ResultErrorCode("NOT_SUPPORTED"), "no process cap"); err != nil {
		t.Fatalf("emit: %v", err)
	}
	m := decodeLine(t, bytes.TrimSpace(buf.Bytes()))
	if m["type"] != "result" || m["ok"] != false {
		t.Errorf("error result: %v", m)
	}
	errObj, ok := m["error"].(map[string]any)
	if !ok {
		t.Fatalf("error obj missing: %v", m)
	}
	if errObj["code"] != "NOT_SUPPORTED" || errObj["message"] != "no process cap" {
		t.Errorf("error fields: %v", errObj)
	}
	if _, has := m["data"]; has {
		t.Errorf("ok:false result must not have data field: %v", m)
	}
}

func TestEmitter_ResultTerminator(t *testing.T) {
	var buf bytes.Buffer
	e := NewEmitter(&buf)
	if err := e.EmitResult(true, nil); err != nil {
		t.Fatalf("first result: %v", err)
	}
	if err := e.EmitEvent(types.EventName("chain.block"), nil); !errors.Is(err, ErrStreamClosed) {
		t.Errorf("event after result: got %v, want ErrStreamClosed", err)
	}
	if err := e.EmitProgress("x", 0, 1); !errors.Is(err, ErrStreamClosed) {
		t.Errorf("progress after result: got %v, want ErrStreamClosed", err)
	}
	if err := e.EmitResult(true, nil); !errors.Is(err, ErrStreamClosed) {
		t.Errorf("second result: got %v, want ErrStreamClosed", err)
	}
}

func TestEmitter_ConcurrentEmits_AllLinesValidJSON(t *testing.T) {
	var buf bytes.Buffer
	e := NewEmitter(&buf)
	const goroutines = 100
	const perGoroutine = 10
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				_ = e.EmitEvent(types.EventName("chain.block"), map[string]any{"g": n, "j": j})
			}
		}(i)
	}
	wg.Wait()
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if got, want := len(lines), goroutines*perGoroutine; got != want {
		t.Fatalf("line count: got %d, want %d", got, want)
	}
	for i, line := range lines {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("line %d not valid JSON: %v (%q)", i, err, line)
		}
	}
}
