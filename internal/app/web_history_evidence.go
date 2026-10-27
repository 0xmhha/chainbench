package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/0xmhha/chainbench/internal/core/process"
	"github.com/0xmhha/chainbench/internal/core/session"
)

// WebRunEvidence lists what a run's failed tests gathered when they failed:
// node health, the process ledger, each node's log and what could not be
// gathered. A run without an engine session has none.
type WebRunEvidence struct {
	RunID string            `json:"runId"`
	Tests []WebEvidenceTest `json:"tests"`
}

type WebEvidenceTest struct {
	Seq   int               `json:"seq"`
	ID    string            `json:"id"`
	Files []WebEvidenceFile `json:"files"`
}

// WebEvidenceFile is one evidence file. Kind is health, processes, log or
// problems; Bytes is its size as the engine wrote it.
type WebEvidenceFile struct {
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	Bytes int64  `json:"bytes"`
}

// WebEvidencePage is a run of lines of one evidence file after redaction,
// starting at line Offset; NextOffset is where the next page starts, or nil
// at the end.
type WebEvidencePage struct {
	Name       string   `json:"name"`
	Kind       string   `json:"kind"`
	Offset     int      `json:"offset"`
	TotalLines int      `json:"totalLines"`
	Lines      []string `json:"lines"`
	NextOffset *int     `json:"nextOffset"`
}

// Evidence is raw engine output, so it gets more than the response
// redaction: every secret field name the engine itself scrubs, in any case and
// inside escaped JSON in a log line, and every private key block, PGP included,
// closed or not. A block whose end never came is hidden to the end of the file.
var (
	evidenceSecretField  = regexp.MustCompile(`(?i)("(?:key|privatekey|private_key|savekey|password|passphrase|mnemonic|secret|accountkey|account_key|setuptoken)"\s*:\s*)"(?:\\.|[^"\\])*"`)
	evidenceEscapedField = regexp.MustCompile(`(?i)(\\"(?:key|privatekey|private_key|savekey|password|passphrase|mnemonic|secret|accountkey|account_key|setuptoken)\\"\s*:\s*)\\"(?:\\\\\\"|\\\\\\\\|[^"\\]|\\[^"])*\\"`)
	evidenceKeyBlock     = regexp.MustCompile(`(?is)-----\s*BEGIN [A-Z0-9 ]*PRIVATE KEY(?: BLOCK)?\s*-----.*?-----\s*END [A-Z0-9 ]*PRIVATE KEY(?: BLOCK)?\s*-----`)
	evidenceOpenKeyBlock = regexp.MustCompile(`(?is)-----\s*BEGIN [A-Z0-9 ]*PRIVATE KEY(?: BLOCK)?\s*-----.*$`)
)

// Evidence pages are bounded in bytes as well as lines: a line longer than
// webEvidenceLineBytes is cut, and a page stops before it passes
// webEvidencePageBytes.
const (
	webEvidenceLineBytes = 64 << 10
	webEvidencePageBytes = 1 << 20
	webEvidenceReaders   = 2
)

// ErrEvidenceBusy reports that every evidence read slot is taken.
var ErrEvidenceBusy = errors.New("evidence reads are busy; try again")

// evidenceSlots bounds how many evidence files are read at once: each read
// holds a whole file in memory while it is redacted.
func (s *WebHistory) evidenceSlots() chan struct{} {
	s.evidenceOnce.Do(func() { s.evidenceReads = make(chan struct{}, webEvidenceReaders) })
	return s.evidenceReads
}

// redactEvidence hides secrets in a whole evidence file. JSON evidence is laid
// out one value per line again afterwards, since the response redaction
// compacts it and a page counts lines.
func (s *WebHistory) redactEvidence(text string) string {
	text = s.redact(text)
	text = evidenceSecretField.ReplaceAllString(text, `${1}"[REDACTED]"`)
	text = evidenceEscapedField.ReplaceAllString(text, `${1}\"[REDACTED]\"`)
	text = evidenceKeyBlock.ReplaceAllString(text, "[REDACTED]")
	text = evidenceOpenKeyBlock.ReplaceAllString(text, "[REDACTED]")
	if json.Valid([]byte(text)) {
		var laid bytes.Buffer
		if json.Indent(&laid, []byte(text), "", "  ") == nil {
			text = laid.String()
		}
	}
	return text
}

// cutLine shortens a line past webEvidenceLineBytes and says how much went.
func cutLine(line string) string {
	if len(line) <= webEvidenceLineBytes {
		return line
	}
	return strings.ToValidUTF8(line[:webEvidenceLineBytes], "") + fmt.Sprintf(" … (%d bytes cut)", len(line)-webEvidenceLineBytes)
}

// ErrEvidenceQuery reports an evidence page outside what the file holds.
var ErrEvidenceQuery = errors.New("invalid evidence query")

func evidenceKind(name string) string {
	switch name {
	case "health.json":
		return "health"
	case "processes.json":
		return "processes"
	case "gather-problems.txt":
		return "problems"
	default:
		return "log"
	}
}

// evidenceReadable admits every active role, as archived observations do.
func evidenceReadable(a DeploymentActor) error {
	switch a.Role {
	case "admin", "operator", "viewer":
	default:
		return ErrDeploymentForbidden
	}
	if a.ID == "" {
		return ErrDeploymentForbidden
	}
	return nil
}

// evidenceSource is the artifact root and session reference behind a run, or
// false for a run with no engine session.
func (s *WebHistory) evidenceSource(runID string) (string, string, bool, error) {
	run, err := s.Get(runID)
	if err != nil {
		return "", "", false, err
	}
	if len(run.SessionRefs) == 0 {
		return "", "", false, nil
	}
	ref := run.SessionRefs[0]
	if trimmed, ok := strings.CutPrefix(ref, "web:"); ok {
		return s.webArtifactRoot, trimmed, true, nil
	}
	if s.artifactRoot == "" {
		return "", "", false, nil
	}
	return s.artifactRoot, ref, true, nil
}

// evidenceOf reads a run's evidence listing with where it came from.
func (s *WebHistory) evidenceOf(a DeploymentActor, runID string) (string, string, []session.EvidenceTest, error) {
	if err := evidenceReadable(a); err != nil {
		return "", "", nil, err
	}
	root, ref, ok, err := s.evidenceSource(runID)
	if err != nil || !ok {
		return "", "", nil, err
	}
	tests, err := session.ListEvidence(root, ref)
	if err != nil {
		return "", "", nil, ErrDeploymentNotFound
	}
	return root, ref, tests, nil
}

// Evidence lists the evidence files of a run's failed tests.
func (s *WebHistory) Evidence(a DeploymentActor, runID string) (WebRunEvidence, error) {
	out := WebRunEvidence{RunID: runID, Tests: []WebEvidenceTest{}}
	_, _, tests, err := s.evidenceOf(a, runID)
	if err != nil {
		return out, err
	}
	for _, test := range tests {
		seq, id, ok := evidenceTestName(test.Dir)
		if !ok {
			continue
		}
		files := []WebEvidenceFile{}
		for _, f := range test.Files {
			files = append(files, WebEvidenceFile{Name: f.Name, Kind: evidenceKind(f.Name), Bytes: f.Bytes})
		}
		out.Tests = append(out.Tests, WebEvidenceTest{Seq: seq, ID: id, Files: files})
	}
	return out, nil
}

// EvidencePage reads lines of one evidence file of the test numbered seq. The
// whole file is redacted before it is cut into lines, so a key block is hidden
// whatever page boundary falls inside it; the process ledger loses its command
// lines, binary paths and data directories before anything else.
func (s *WebHistory) EvidencePage(a DeploymentActor, runID string, seq int, name string, offset, limit int) (WebEvidencePage, error) {
	if limit == 0 {
		limit = 200
	}
	if offset < 0 || limit < 1 || limit > 1000 {
		return WebEvidencePage{}, ErrEvidenceQuery
	}
	root, ref, tests, err := s.evidenceOf(a, runID)
	if err != nil {
		return WebEvidencePage{}, err
	}
	dir := ""
	for _, test := range tests {
		if n, _, ok := evidenceTestName(test.Dir); !ok || n != seq {
			continue
		}
		for _, f := range test.Files {
			if f.Name == name {
				dir = test.Dir
			}
		}
	}
	if dir == "" {
		return WebEvidencePage{}, ErrDeploymentNotFound
	}
	select {
	case s.evidenceSlots() <- struct{}{}:
		defer func() { <-s.evidenceSlots() }()
	default:
		return WebEvidencePage{}, ErrEvidenceBusy
	}
	raw, err := session.ReadEvidence(root, ref, dir, name)
	if err != nil {
		return WebEvidencePage{}, ErrDeploymentNotFound
	}
	text := string(raw)
	if name == "processes.json" {
		if text, err = withoutCommandLines(raw); err != nil {
			return WebEvidencePage{}, err
		}
	}
	lines := strings.Split(strings.TrimSuffix(s.redactEvidence(text), "\n"), "\n")
	if offset > len(lines) {
		return WebEvidencePage{}, ErrEvidenceQuery
	}
	page := WebEvidencePage{Name: name, Kind: evidenceKind(name), Offset: offset, TotalLines: len(lines), Lines: []string{}}
	end, size := offset, 0
	for end < len(lines) && end < offset+limit {
		line := cutLine(lines[end])
		if size+len(line) > webEvidencePageBytes && end > offset {
			break
		}
		page.Lines = append(page.Lines, line)
		size += len(line)
		end++
	}
	if end < len(lines) {
		page.NextOffset = &end
	}
	return page, nil
}

// withoutCommandLines keeps from each recorded process only what identifies
// it: pid, label, host and revision. Command lines can carry key files or
// passwords; the domain model keeps them, binaries and data paths out.
func withoutCommandLines(raw []byte) (string, error) {
	var procs []process.Proc
	if err := json.Unmarshal(raw, &procs); err != nil {
		return "", errors.New("the process ledger could not be read")
	}
	type shown struct {
		PID      int    `json:"pid"`
		Label    string `json:"label"`
		Host     string `json:"host,omitempty"`
		Revision int    `json:"revision,omitempty"`
	}
	out := make([]shown, 0, len(procs))
	for _, p := range procs {
		out = append(out, shown{PID: p.PID, Label: p.Label, Host: p.Host, Revision: p.Revision})
	}
	b, err := json.MarshalIndent(out, "", "  ")
	return string(b), err
}

// evidenceTestName reads "tests/<seq>_<id>" back into its parts.
func evidenceTestName(dir string) (int, string, bool) {
	name, ok := strings.CutPrefix(dir, "tests/")
	if !ok {
		return 0, "", false
	}
	number, id, ok := strings.Cut(name, "_")
	seq, err := strconv.Atoi(number)
	if !ok || err != nil || id == "" {
		return 0, "", false
	}
	return seq, id, true
}
