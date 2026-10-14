//go:build e2e

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestHandoffCleanupPreservesDataWhenTerminationIsUnproven(t *testing.T) {
	root := t.TempDir()
	database := filepath.Join(root, "node1", "geth", "chaindata")
	if err := os.MkdirAll(database, 0700); err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(database, "evidence")
	if err := os.WriteFile(data, []byte("still in use"), 0600); err != nil {
		t.Fatal(err)
	}
	stopErr := errors.New("recorded process still alive")
	reclaimed := false
	err := finishHandoffWorkspace(t.Context(), root, func(context.Context, string) error { return stopErr }, func(context.Context, string) error { reclaimed = true; return os.RemoveAll(database) })
	if !errors.Is(err, stopErr) || reclaimed {
		t.Fatal("unproven termination allowed data reclamation", err, reclaimed)
	}
	if body, err := os.ReadFile(data); err != nil || string(body) != "still in use" {
		t.Fatal("active node evidence lost", err)
	}
}

func TestHandoffCleanupOrdersTerminationBeforeReclamationAndReportsFailure(t *testing.T) {
	var calls []string
	reclaimErr := errors.New("open file inventory unavailable")
	err := finishHandoffWorkspace(t.Context(), "owned", func(context.Context, string) error { calls = append(calls, "stop"); return nil }, func(context.Context, string) error { calls = append(calls, "reclaim"); return reclaimErr })
	if !errors.Is(err, reclaimErr) || !reflect.DeepEqual(calls, []string{"stop", "reclaim"}) {
		t.Fatal(err, calls)
	}
}

func TestHandoffStopRefusesARecordlessProcessLedger(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "process.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := stopHandoffWorkspace(t.Context(), root); err == nil {
		t.Fatal("an unknown process ledger was interpreted as never launched")
	}
}

func TestHandoffStopRefusesACorruptChainRecord(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "chain-record.json"), []byte("not JSON"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := stopHandoffWorkspace(t.Context(), root); err == nil {
		t.Fatal("a corrupt record was interpreted as stopped")
	}
}

func TestHandoffStopAllowsAnEmptyNeverLaunchedWorkspace(t *testing.T) {
	if err := stopHandoffWorkspace(t.Context(), t.TempDir()); err != nil {
		t.Fatal(err)
	}
}
