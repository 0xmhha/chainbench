package session

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Artifact file names within a test record folder.
const (
	fileSpec       = "spec.json"
	fileSteps      = "steps.json"
	fileAssert     = "assert.json"
	fileStatus     = "status.json"
	filePostAction = "postaction.json"
)

// record is the concrete TestRecord: it accumulates one test's artifacts and
// writes them under tests/<NNN>_<id>/. Write errors are collected and surfaced
// by the owning session's Save rather than silently dropped.
type record struct {
	reason string
	dir    string
	seq    int
	id     string

	mu      sync.Mutex
	envRef  string
	status  TestStatus
	steps   []StepResult
	asserts []AssertResult
	posts   []PostResult
	errs    []error
}

func (r *record) Dir() string { return r.dir }

// SetEnvRef records which environment this test ran against.
func (r *record) SetEnvRef(envID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.envRef = envID
	r.capture(os.WriteFile(filepath.Join(r.dir, fileEnvRef), []byte(envID), 0o644))
}

// Spec stores the raw test definition as spec.json.
func (r *record) Spec(raw []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.capture(os.WriteFile(filepath.Join(r.dir, fileSpec), raw, 0o644))
}

// Step appends an executed step and rewrites steps.json.
func (r *record) Step(i int, res StepResult) {
	r.mu.Lock()
	defer r.mu.Unlock()
	res.Index = i
	r.steps = append(r.steps, res)
	r.capture(writeJSON(filepath.Join(r.dir, fileSteps), r.steps))
}

// Assert appends an assertion result and rewrites assert.json.
func (r *record) Assert(res AssertResult) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.asserts = append(r.asserts, res)
	r.capture(writeJSON(filepath.Join(r.dir, fileAssert), r.asserts))
}

// statusDoc is the status.json schema.
type statusDoc struct {
	ID     string `json:"id"`
	Seq    int    `json:"seq"`
	Result string `json:"result"`
	Env    string `json:"env,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// Status records the terminal verdict as status.json.
func (r *record) Status(s TestStatus) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.status = s
	r.writeStatus()
}

// Reason records why, and rewrites the status so the two travel together.
func (r *record) Reason(why string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reason = why
	r.writeStatus()
}

// writeStatus persists the verdict and its reason. The caller holds the lock.
func (r *record) writeStatus() {
	r.capture(writeJSON(filepath.Join(r.dir, fileStatus), statusDoc{
		ID:     r.id,
		Seq:    r.seq,
		Result: string(r.status),
		Env:    r.envRef,
		Reason: r.reason,
	}))
}

// PostAction appends a post-action outcome; failures here do not change the
// verdict but are still recorded.
func (r *record) PostAction(res PostResult) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.posts = append(r.posts, res)
	r.capture(writeJSON(filepath.Join(r.dir, filePostAction), r.posts))
}

// record collects a non-nil write error for later surfacing by session.Save.
func (r *record) capture(err error) {
	if err != nil {
		r.errs = append(r.errs, fmt.Errorf("record %s: %w", r.id, err))
	}
}
