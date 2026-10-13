package testengine

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/core/report"
	"github.com/0xmhha/chainbench/internal/core/session"
	"github.com/0xmhha/chainbench/internal/dsl"
)

func TestEngineSerialExecutionContinuesAfterFailedVerdict(t *testing.T) {
	var active, maxActive atomic.Int32
	var executed []string
	var executedMu sync.Mutex
	clock := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	base := t.TempDir()
	eng := New(Deps{
		NewSession: func(_ context.Context, command string) (session.Session, error) {
			return session.New(base, command, clock)
		},
		Fingerprint: func(dsl.Spec) session.Fingerprint { return "same-environment" },
		BuildEnv: func(context.Context, session.Environment, dsl.Spec) (node.NodeSet, TeardownFunc, error) {
			return node.NodeSet{}, func(context.Context) error { return nil }, nil
		},
		RunSpec: func(_ context.Context, spec dsl.Spec, _ session.Environment, rec session.TestRecord) (session.TestStatus, error) {
			now := active.Add(1)
			defer active.Add(-1)
			for {
				old := maxActive.Load()
				if old >= now || maxActive.CompareAndSwap(old, now) {
					break
				}
			}
			executedMu.Lock()
			executed = append(executed, spec.ID)
			executedMu.Unlock()
			rec.Observation("execution.txt", []byte(spec.ID))
			time.Sleep(10 * time.Millisecond)
			if spec.ID == "middle-fails" {
				rec.Status(session.StatusFail)
				return session.StatusFail, nil
			}
			rec.Status(session.StatusPass)
			return session.StatusPass, nil
		},
	})
	root, err := eng.Run(context.Background(), [][]byte{attachSpec(t, "first", "1"), attachSpec(t, "middle-fails", "1"), attachSpec(t, "last", "1")})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(executed, []string{"first", "middle-fails", "last"}) || maxActive.Load() != 1 {
		t.Fatalf("execution order=%v, max concurrency=%d", executed, maxActive.Load())
	}
	rep, err := report.Build(root)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Summary.Pass != 2 || rep.Summary.Fail != 1 || len(rep.Tests) != 3 {
		t.Fatal(rep)
	}
	for i, test := range rep.Tests {
		if test.ID != executed[i] || test.Seq != i+1 {
			t.Fatal("report detached from execution order")
		}
		content, err := os.ReadFile(filepath.Join(root, test.Dir, "observations/execution.txt"))
		if err != nil || string(content) != test.ID {
			t.Fatalf("missing execution evidence: %s %v", test.ID, err)
		}
	}
}

func TestRapidRunsPreserveRootReportsAndEvidence(t *testing.T) {
	stub := newChainStub("0x9")
	srv := httptest.NewServer(stub)
	defer srv.Close()
	base := t.TempDir()
	clock := func() time.Time { return time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC) }
	eng, err := NewAttachEngine(AttachConfig{Chain: "stablenet", RPCURLs: []string{srv.URL}, ArtifactRoot: base, Clock: clock})
	if err != nil {
		t.Fatal(err)
	}
	first, err := eng.Run(context.Background(), [][]byte{attachSpec(t, "same-case", "5")})
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(first, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := eng.Run(context.Background(), [][]byte{attachSpec(t, "same-case", "10")})
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("rapid definitions overwrite each other's report")
	}
	after, err := os.ReadFile(filepath.Join(first, "report.json"))
	if err != nil || string(before) != string(after) {
		t.Fatal("first root report changed")
	}
	var a, b report.Report
	if err := json.Unmarshal(before, &a); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(second, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &b); err != nil {
		t.Fatal(err)
	}
	if a.Summary.Pass != 1 || b.Summary.Fail != 1 || a.Session == b.Session {
		t.Fatalf("distinct verdicts lost: %+v %+v", a, b)
	}
	for _, entry := range []struct {
		root   string
		report report.Report
	}{{first, a}, {second, b}} {
		for _, test := range entry.report.Tests {
			if len(test.Evidence) < 3 {
				t.Fatal("root report lacks evidence links")
			}
			for _, link := range test.Evidence {
				if _, err := os.Stat(filepath.Join(entry.root, link)); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
}
