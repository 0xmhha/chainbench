package app

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// evidenceHistory is a history over one Web-owned session whose failed test
// gathered evidence: a process ledger with a command line, and a node log
// that printed a private key block.
func evidenceHistory(t *testing.T) (*WebHistory, string) {
	t.Helper()
	root := t.TempDir()
	s, err := session.New(webOwnedSessions(root), "test run", time.Date(2026, 10, 27, 1, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	r := s.Test(1, "fails")
	r.Status(session.StatusFail)
	procs, _ := json.Marshal([]process.Proc{{PID: 4242, Label: "node1", Binary: "/opt/gstable", Command: "/opt/gstable --password launch-secret-word", DataDir: "/data/node1", Host: "10.0.0.5"}})
	r.Observation("processes.json", procs)
	var log strings.Builder
	for i := 1; i <= 250; i++ {
		if i == 120 {
			log.WriteString("-----BEGIN PRIVATE KEY-----\nMIIBVAIBADANBgkqhkiG9w0BAQEFAASCAT4wggE6AgEAAkEA\n-----END PRIVATE KEY-----\n")
		}
		log.WriteString("INFO line " + strings.Repeat("x", i%3) + "\n")
	}
	r.Observation("node1.log", []byte(log.String()))
	// A node that died while printing a key leaves a block with no end.
	r.Observation("node2.log", []byte("INFO before\n-----BEGIN PRIVATE KEY-----\nMIIBVAIBADANBgkqhkiG9w0BAQEFAASCAT4wggE6AgEAAkEA\n"))
	if err = s.Save(); err != nil {
		t.Fatal(err)
	}
	history, err := OpenWebHistory(root, "", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = history.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	return history, "web:" + session.IDFor(webOwnedSessions(root), s.Root())
}

func TestWebRunEvidenceListsAFailedTestsFiles(t *testing.T) {
	history, ref := evidenceHistory(t)
	viewer := DeploymentActor{ID: "viewer", Role: "viewer"}
	got, err := history.Evidence(viewer, ref)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Tests) != 1 || got.Tests[0].Seq != 1 || got.Tests[0].ID != "fails" || len(got.Tests[0].Files) != 3 {
		t.Fatalf("evidence = %+v", got)
	}
	kinds := map[string]string{}
	for _, f := range got.Tests[0].Files {
		kinds[f.Name] = f.Kind
		if f.Bytes <= 0 {
			t.Fatalf("file without a size: %+v", f)
		}
	}
	if kinds["processes.json"] != "processes" || kinds["node1.log"] != "log" {
		t.Fatalf("kinds = %v", kinds)
	}
}

func TestWebRunEvidenceNeverShowsCommandLinesOrKeys(t *testing.T) {
	history, ref := evidenceHistory(t)
	viewer := DeploymentActor{ID: "viewer", Role: "viewer"}
	page, err := history.EvidencePage(viewer, ref, 1, "processes.json", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(page.Lines, "\n")
	for _, hidden := range []string{"launch-secret-word", "--password", "/opt/gstable", "/data/node1"} {
		if strings.Contains(text, hidden) {
			t.Fatalf("processes evidence shows %q:\n%s", hidden, text)
		}
	}
	for _, kept := range []string{`"pid": 4242`, `"label": "node1"`, `"host": "10.0.0.5"`} {
		if !strings.Contains(text, kept) {
			t.Fatalf("processes evidence lost %q:\n%s", kept, text)
		}
	}
	// Every page of the log, whatever its boundary, hides the key block.
	whole, err := history.EvidencePage(viewer, ref, 1, "node1.log", 0, 1000)
	if err != nil {
		t.Fatal(err)
	}
	for offset := 0; offset < whole.TotalLines; offset += 7 {
		page, err := history.EvidencePage(viewer, ref, 1, "node1.log", offset, 5)
		if err != nil {
			t.Fatal(err)
		}
		if joined := strings.Join(page.Lines, "\n"); strings.Contains(joined, "MIIBVAIBADANBgkqhkiG9w0BAQEFAASCAT4wggE6AgEAAkEA") {
			t.Fatalf("page at %d shows key material:\n%s", offset, joined)
		}
	}
}

func TestWebRunEvidenceHidesAKeyBlockThatNeverEnds(t *testing.T) {
	history, ref := evidenceHistory(t)
	page, err := history.EvidencePage(DeploymentActor{ID: "viewer", Role: "viewer"}, ref, 1, "node2.log", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(page.Lines, "\n")
	if strings.Contains(joined, "MIIBVAIBADANBgkqhkiG9w0BAQEFAASCAT4wggE6AgEAAkEA") || !strings.Contains(joined, "INFO before") {
		t.Fatalf("unterminated block page:\n%s", joined)
	}
}

func TestWebRunEvidencePagesByLine(t *testing.T) {
	history, ref := evidenceHistory(t)
	viewer := DeploymentActor{ID: "viewer", Role: "viewer"}
	first, err := history.EvidencePage(viewer, ref, 1, "node1.log", 0, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Lines) != 100 || first.Offset != 0 || first.NextOffset == nil || *first.NextOffset != 100 || first.TotalLines < 250 {
		t.Fatalf("first page = offset %d next %v total %d lines %d", first.Offset, first.NextOffset, first.TotalLines, len(first.Lines))
	}
	last, err := history.EvidencePage(viewer, ref, 1, "node1.log", first.TotalLines-10, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(last.Lines) != 10 || last.NextOffset != nil {
		t.Fatalf("last page = %d lines next %v", len(last.Lines), last.NextOffset)
	}
	for _, bad := range []struct {
		seq           int
		name          string
		offset, limit int
	}{{2, "node1.log", 0, 10}, {1, "status.json", 0, 10}, {1, "node1.log", -1, 10}, {1, "node1.log", 0, 1001}} {
		if _, err = history.EvidencePage(viewer, ref, bad.seq, bad.name, bad.offset, bad.limit); err == nil {
			t.Errorf("page %+v was served", bad)
		}
	}
	if _, err = history.Evidence(viewer, "web:no/such-session"); err == nil {
		t.Error("evidence for an unknown run was served")
	}
}

// TestWebRunEvidenceIsEmptyWithoutASession: a job's own run has no session, so no
// evidence; its session run carries it.
func TestWebRunEvidenceIsEmptyWithoutASession(t *testing.T) {
	history, _ := evidenceHistory(t)
	history.state.Captures["job-x"] = webHistoryCapture{Run: WebRun{ID: "job-x", SessionRefs: []string{}, Summary: map[string]any{"kind": "job"}}}
	got, err := history.Evidence(DeploymentActor{ID: "viewer", Role: "viewer"}, "job-x")
	if err != nil || len(got.Tests) != 0 {
		t.Fatalf("job run evidence = %+v %v", got, err)
	}
}

// productionEvidence is a history whose redaction is the deployment store's,
// as the dashboard runs it, over one failed test with the given evidence.
func productionEvidence(t *testing.T, files map[string]string) (*WebHistory, string) {
	t.Helper()
	root := t.TempDir()
	store, err := OpenDeploymentStore(root)
	if err != nil {
		t.Fatal(err)
	}
	s, err := session.New(webOwnedSessions(root), "test run", time.Date(2026, 10, 27, 1, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	r := s.Test(1, "fails")
	r.Status(session.StatusFail)
	for name, content := range files {
		r.Observation(name, []byte(content))
	}
	if err = s.Save(); err != nil {
		t.Fatal(err)
	}
	history, err := OpenWebHistory(root, "", nil, store.RedactWeb)
	if err != nil {
		t.Fatal(err)
	}
	if err = history.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	return history, "web:" + session.IDFor(webOwnedSessions(root), s.Root())
}

var evidenceViewer = DeploymentActor{ID: "viewer", Role: "viewer"}

// TestWebRunEvidencePagesJSONByLineUnderProductionRedaction: the dashboard's
// redaction compacts JSON; evidence JSON is laid out again, so paging still
// bounds what one response carries.
func TestWebRunEvidencePagesJSONByLineUnderProductionRedaction(t *testing.T) {
	health := `{"nodes":[` + strings.Repeat(`{"label":"node1","height":12,"peers":3},`, 50) + `{"label":"node2","height":11,"peers":3}]}`
	procs, _ := json.Marshal([]process.Proc{{PID: 1, Label: "node1"}, {PID: 2, Label: "node2"}})
	history, ref := productionEvidence(t, map[string]string{"health.json": health, "processes.json": string(procs)})
	for _, name := range []string{"health.json", "processes.json"} {
		page, err := history.EvidencePage(evidenceViewer, ref, 1, name, 0, 5)
		if err != nil {
			t.Fatal(err)
		}
		if page.TotalLines < 6 || len(page.Lines) != 5 || page.NextOffset == nil {
			t.Fatalf("%s pages as %d lines (%d shown)", name, page.TotalLines, len(page.Lines))
		}
	}
}

// TestWebRunEvidenceBoundsWhatOnePageCarries: a log written as one enormous
// line is cut, and a page stops before it grows past its byte budget.
func TestWebRunEvidenceBoundsWhatOnePageCarries(t *testing.T) {
	long := strings.Repeat("x", 3<<20)
	var many strings.Builder
	for i := 0; i < 1000; i++ {
		many.WriteString(strings.Repeat("y", 4000) + "\n")
	}
	history, ref := productionEvidence(t, map[string]string{"node1.log": long + "\n", "node2.log": many.String()})
	for _, name := range []string{"node1.log", "node2.log"} {
		page, err := history.EvidencePage(evidenceViewer, ref, 1, name, 0, 1000)
		if err != nil {
			t.Fatal(err)
		}
		size := 0
		for _, line := range page.Lines {
			size += len(line)
		}
		if size > webEvidencePageBytes+webEvidenceLineBytes {
			t.Fatalf("%s page carries %d bytes", name, size)
		}
		if name == "node1.log" && !strings.Contains(page.Lines[0], "bytes cut") {
			t.Fatalf("an enormous line was not marked as cut: %q", page.Lines[0][len(page.Lines[0])-40:])
		}
		if name == "node2.log" && (page.NextOffset == nil || len(page.Lines) >= 1000) {
			t.Fatalf("a page past its byte budget did not stop: %d lines", len(page.Lines))
		}
	}
}

// TestWebRunEvidenceHidesSecretFieldsInAnyCase: secret fields the engine names
// are hidden whatever their case, inside escaped JSON in a log line too, and a
// PGP key block is hidden like other private key blocks.
func TestWebRunEvidenceHidesSecretFieldsInAnyCase(t *testing.T) {
	log := strings.Join([]string{
		`INFO saved {"Mnemonic":"abandon ability able"}`,
		`INFO request {\"privateKey\":\"0xfeedface01\"}`,
		`INFO sign {"key":"abc\"defsecret"}`,
		`INFO account {"Key":"0xdeadbeef02","Secret":"s3cr3t-word"}`,
		`INFO nested {\"key\":\"ab\\\"cdnested\"}`,
		`----- BEGIN PRIVATE KEY -----`,
		`spacedheadersecret`,
		`----- END PRIVATE KEY -----`,
		`-----BEGIN PGP PRIVATE KEY BLOCK-----`,
		`lQOYBGWpgpblocksecret`,
		`-----END PGP PRIVATE KEY BLOCK-----`,
		`INFO after`,
	}, "\n")
	history, ref := productionEvidence(t, map[string]string{"node1.log": log + "\n"})
	page, err := history.EvidencePage(evidenceViewer, ref, 1, "node1.log", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(page.Lines, "\n")
	for _, secret := range []string{"abandon ability", "0xfeedface01", "defsecret", "0xdeadbeef02", "s3cr3t-word", "pgpblocksecret", "cdnested", "spacedheadersecret"} {
		if strings.Contains(joined, secret) {
			t.Errorf("evidence shows %q:\n%s", secret, joined)
		}
	}
	if !strings.Contains(joined, "INFO after") {
		t.Fatalf("redaction ate the rest of the log:\n%s", joined)
	}
}

// TestWebRunEvidenceRefusesWhenBusy: evidence reads take a bounded number of
// slots; a read beyond them is refused rather than queued without limit.
func TestWebRunEvidenceRefusesWhenBusy(t *testing.T) {
	history, ref := productionEvidence(t, map[string]string{"node1.log": "INFO one\n"})
	for i := 0; i < cap(history.evidenceSlots()); i++ {
		history.evidenceSlots() <- struct{}{}
	}
	if _, err := history.EvidencePage(evidenceViewer, ref, 1, "node1.log", 0, 0); !errors.Is(err, ErrEvidenceBusy) {
		t.Fatalf("a read past the slots = %v", err)
	}
	for i := 0; i < cap(history.evidenceSlots()); i++ {
		<-history.evidenceSlots()
	}
	if _, err := history.EvidencePage(evidenceViewer, ref, 1, "node1.log", 0, 0); err != nil {
		t.Fatal(err)
	}
}
