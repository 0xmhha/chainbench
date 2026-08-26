package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/0xmhha/chainbench/internal/core/keyring/store"
)

// Layout names for the .chainbench/<session>/ tree (design §D-1). Kept as
// constants so the layout has a single source of truth.
const (
	dirKeys         = "keys"
	dirEnvironments = "environments"
	dirTests        = "tests"
	dirLogs         = "logs"
	dirChainstate   = "chainstate"
	dirNodes        = "nodes"

	fileSession = "session.json"
	fileEnv     = "env.json"
	fileEnvRef  = "env-ref"

	envIDPrefix = "env-"
	// fpShortLen is how many fingerprint hex chars go into an env-id folder
	// name, bounding path length (design L5).
	fpShortLen = 12

	// sessionIDLayout formats the start time as YYYYMMDD-HHMMSS; the id is this
	// with a "UTC-" prefix.
	sessionIDLayout = "20060102-150405"

	dirPerm     fs.FileMode = 0o755
	keysDirPerm fs.FileMode = 0o700
)

// SessionID returns the deterministic session id for a start time:
// "UTC-YYYYMMDD-HHMMSS" in UTC.
func SessionID(startedAt time.Time) string {
	return "UTC-" + startedAt.UTC().Format(sessionIDLayout)
}

// EnvID returns the environment folder id for a fingerprint: "env-" plus the
// first fpShortLen hex chars (or the whole value if shorter).
func EnvID(fp Fingerprint) string {
	s := string(fp)
	if len(s) > fpShortLen {
		s = s[:fpShortLen]
	}
	return envIDPrefix + s
}

// sess is the concrete Session: the single owner of one command's artifact tree.
type sess struct {
	id        string
	root      string
	command   string
	startedAt time.Time
	keys      *store.KeySet

	mu    sync.Mutex
	envs  map[Fingerprint]*env
	tests []*record
}

// New creates the .chainbench/<session>/ tree under baseDir and returns the
// session, with its keyring rooted in the session's own keys/ directory.
// startedAt is injected (not read from the clock) for reproducibility. A
// disk-write failure aborts session start.
//
// The session decides where key material lands because it owns the artifact
// layout (design §3.1, single ownership); a caller never reconstructs that path.
func New(baseDir, command string, startedAt time.Time) (Session, error) {
	return newSession(baseDir, command, startedAt, store.NewKeySet)
}

// newSession creates the session tree and binds its keyring. newKeys receives
// the session's keys/ path so the ring can be rooted there without the layout
// leaking to callers.
func newSession(baseDir, command string, startedAt time.Time, newKeys func(keysDir string) *store.KeySet) (Session, error) {
	id := SessionID(startedAt)
	root := filepath.Join(baseDir, id)
	dirs := []struct {
		path string
		perm fs.FileMode
	}{
		{root, dirPerm},
		{filepath.Join(root, dirKeys), keysDirPerm},
		{filepath.Join(root, dirEnvironments), dirPerm},
		{filepath.Join(root, dirTests), dirPerm},
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d.path, d.perm); err != nil {
			return nil, fmt.Errorf("session: create %s: %w", d.path, err)
		}
	}
	return &sess{
		id:        id,
		root:      root,
		command:   command,
		startedAt: startedAt,
		keys:      newKeys(filepath.Join(root, dirKeys)),
		envs:      make(map[Fingerprint]*env),
	}, nil
}

func (s *sess) ID() string          { return s.id }
func (s *sess) Root() string        { return s.root }
func (s *sess) Keys() *store.KeySet { return s.keys }

// Environment returns an existing environment for fp, or ok=false when none
// exists yet.
func (s *sess) Environment(fp Fingerprint) (Environment, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.envs[fp]
	if !ok {
		return nil, false
	}
	return e, true
}

// NewEnvironment creates the environment tree for fp and registers it.
func (s *sess) NewEnvironment(fp Fingerprint) (Environment, error) {
	id := EnvID(fp)
	dir := filepath.Join(s.root, dirEnvironments, id)
	for _, sub := range []string{"", dirLogs, dirChainstate, dirNodes} {
		if err := os.MkdirAll(filepath.Join(dir, sub), dirPerm); err != nil {
			return nil, fmt.Errorf("session: create env %s: %w", id, err)
		}
	}
	e := &env{id: id, dir: dir, fp: fp, dataPath: dir}
	s.mu.Lock()
	s.envs[fp] = e
	s.mu.Unlock()
	return e, nil
}

// Test creates the record folder for one test (tests/<NNN>_<id>) and returns it.
func (s *sess) Test(seq int, id string) TestRecord {
	dir := filepath.Join(s.root, dirTests, fmt.Sprintf("%03d_%s", seq, id))
	_ = os.MkdirAll(dir, dirPerm)
	r := &record{dir: dir, seq: seq, id: id}
	s.mu.Lock()
	s.tests = append(s.tests, r)
	s.mu.Unlock()
	return r
}

// sessionDoc is the session.json schema.
type sessionDoc struct {
	ID        string       `json:"id"`
	Command   string       `json:"command"`
	StartedAt string       `json:"startedAt"`
	Tests     []testEntry  `json:"tests"`
	Summary   summaryCount `json:"summary"`
}

type testEntry struct {
	Seq    int    `json:"seq"`
	ID     string `json:"id"`
	Env    string `json:"env,omitempty"`
	Status string `json:"status"`
}

type summaryCount struct {
	Pass    int `json:"pass"`
	Fail    int `json:"fail"`
	Blocked int `json:"blocked"`
	Skip    int `json:"skip"`
}

// Save writes session.json. It returns any deferred artifact-write error so
// record write failures are surfaced rather than lost.
func (s *sess) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	doc := sessionDoc{
		ID:        s.id,
		Command:   s.command,
		StartedAt: s.startedAt.UTC().Format(time.RFC3339),
		Tests:     make([]testEntry, 0, len(s.tests)),
	}
	var writeErrs []error
	for _, r := range s.tests {
		doc.Tests = append(doc.Tests, testEntry{Seq: r.seq, ID: r.id, Env: r.envRef, Status: string(r.status)})
		switch r.status {
		case StatusPass:
			doc.Summary.Pass++
		case StatusFail:
			doc.Summary.Fail++
		case StatusBlocked:
			doc.Summary.Blocked++
		case StatusSkip:
			doc.Summary.Skip++
		}
		writeErrs = append(writeErrs, r.errs...)
	}
	if err := writeJSON(filepath.Join(s.root, fileSession), doc); err != nil {
		return err
	}
	return errors.Join(writeErrs...)
}

// writeJSON marshals v indented and writes it to path atomically enough for
// artifacts (single Save per file).
func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("session: marshal %s: %w", filepath.Base(path), err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		return fmt.Errorf("session: write %s: %w", filepath.Base(path), err)
	}
	return nil
}
