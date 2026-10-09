package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/0xmhha/chainbench/internal/core/session"
)

type savedWebPlan struct {
	Public   WebPlan        `json:"public"`
	Input    WebPlanInput   `json:"input"`
	Prepared WebPreparedJob `json:"prepared"`
}
type webJobKey struct{ Body, JobID string }
type webJobAudit struct {
	ActorID, JobID, Operation string
	Time                      time.Time
}
type webJobState struct {
	Plans map[string]savedWebPlan `json:"plans"`
	Jobs  map[string]WebJob       `json:"jobs"`
	Keys  map[string]webJobKey    `json:"keys"`
	Audit []webJobAudit           `json:"audit"`
}

// WebJobs owns durable jobs and physical resource exclusion. HTTP contexts are
// used for planning only; accepted jobs have their own cancellable lifetime.
type WebJobs struct {
	mu        sync.Mutex
	feed      *webJobFeed
	files     *session.JobStore
	state     webJobState
	engine    WebJobEngine
	redact    func(string) string
	authorize func(DeploymentActor) error
	cancels   map[string]context.CancelFunc
	root      string
}

func OpenWebJobs(root string, engine WebJobEngine, redact func(string) string, authorize func(DeploymentActor) error) (*WebJobs, error) {
	files, err := session.OpenJobStore(root)
	if err != nil {
		return nil, err
	}
	s := &WebJobs{feed: newWebJobFeed(), files: files, engine: engine, redact: redact, authorize: authorize, cancels: map[string]context.CancelFunc{}, root: root, state: webJobState{Plans: map[string]savedWebPlan{}, Jobs: map[string]WebJob{}, Keys: map[string]webJobKey{}}}
	b, err := files.Read()
	if err == nil {
		if err = json.Unmarshal(b, &s.state); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	if s.state.Plans == nil || s.state.Jobs == nil || s.state.Keys == nil {
		return nil, errors.New("invalid job snapshot")
	}
	changed := false
	for id, plan := range s.state.Plans {
		if plan.Public.ID != id || plan.Public.ActorID == "" || plan.Public.TargetFingerprint != plan.Prepared.Fingerprint {
			return nil, errors.New("invalid plan snapshot")
		}
		if err := validatePrepared(plan.Prepared); err != nil {
			return nil, err
		}
	}
	for _, key := range s.state.Keys {
		if _, ok := s.state.Jobs[key.JobID]; !ok {
			return nil, errors.New("invalid idempotency snapshot")
		}
	}
	for id, job := range s.state.Jobs {
		if id != job.ID {
			return nil, errors.New("invalid job identity")
		}
		plan, exists := s.state.Plans[job.PlanID]
		if !exists || plan.Public.ActorID != job.ActorID || plan.Public.WorkspaceID != job.WorkspaceID || plan.Public.Operation != job.Operation {
			return nil, errors.New("invalid job plan reference")
		}
		if !terminalWebJob(job.State) && job.State != "accepted" && job.State != "running" && job.State != "cancelling" {
			return nil, errors.New("invalid job state")
		}
		if !terminalWebJob(job.State) {
			job.State, job.NodeDisposition = "interrupted", "unknown"
			job.Error = map[string]string{"code": "server_restarted", "message": "Server restarted; inspect retained resources before starting another operation."}
			job.UnresolvedResources = append(job.UnresolvedResources, s.state.Plans[job.PlanID].Public.Resources...)
			s.state.Jobs[id] = job
			changed = true
		}
	}
	if changed {
		if err = s.commit("system", "", "jobs.interrupted", func(*webJobState) {}); err != nil {
			return nil, err
		}
	}
	return s, nil
}

func (s *WebJobs) commit(actor, id, operation string, change func(*webJobState)) error {
	b, err := json.Marshal(s.state)
	if err != nil {
		return err
	}
	var next webJobState
	if err = json.Unmarshal(b, &next); err != nil {
		return err
	}
	change(&next)
	next.Audit = append(next.Audit, webJobAudit{actor, id, operation, time.Now().UTC()})
	b, err = json.Marshal(next)
	if err != nil {
		return err
	}
	var committed webJobState
	if err = json.Unmarshal(b, &committed); err != nil {
		return err
	}
	if err = s.files.Write(b); err != nil {
		return err
	}
	s.state = committed
	s.recordJobChange(id)
	return nil
}

func (s *WebJobs) allowed(a DeploymentActor) error {
	if !a.canEdit() {
		return ErrDeploymentForbidden
	}
	if s.authorize != nil {
		return s.authorize(a)
	}
	return nil
}

func validatePrepared(p WebPreparedJob) error {
	if len(p.Fingerprint) != 64 || len(p.Claims) == 0 || !json.Valid(p.Payload) {
		return errors.New("engine did not resolve an immutable resource plan")
	}
	if _, err := hex.DecodeString(p.Fingerprint); err != nil {
		return errors.New("invalid target fingerprint")
	}
	for _, c := range p.Claims {
		if c.HostIdentity == "" || (c.DataPath == "" && len(c.Ports) == 0 && c.Executable == "") {
			return errors.New("unresolved physical resource")
		}
		if c.Executable != "" && (c.DataPath != "" || len(c.Ports) != 0 || strings.ContainsAny(c.Executable, "/\x00")) {
			return errors.New("unresolved physical resource")
		}
		if c.DataPath != "" && (!strings.HasPrefix(c.DataPath, "/") || strings.Contains(c.DataPath, "\x00")) {
			return errors.New("absolute resource path required")
		}
		for _, port := range c.Ports {
			if port < 1 || port > 65535 {
				return errors.New("invalid claimed port")
			}
		}
	}
	return nil
}

// Plan returns a short-lived actor-bound review of a resolved engine operation.
func (s *WebJobs) Plan(ctx context.Context, a DeploymentActor, in WebPlanInput) (WebPlan, error) {
	if err := s.allowed(a); err != nil {
		return WebPlan{}, err
	}
	if s.engine == nil {
		return WebPlan{}, errors.New("execution engine unavailable")
	}
	if in.WorkspaceID == "" {
		return WebPlan{}, errors.New("workspace required")
	}
	if in.Retention == "" {
		in.Retention = "retain"
	}
	if in.Retention != "retain" && in.Retention != "cleanup" {
		return WebPlan{}, errors.New("invalid retention")
	}
	p, err := s.engine.Prepare(ctx, a, in)
	if err != nil {
		return WebPlan{}, err
	}
	if err = validatePrepared(p); err != nil {
		return WebPlan{}, err
	}
	plan := WebPlan{ID: deploymentID(), ActorID: a.ID, WorkspaceID: in.WorkspaceID, Operation: in.Operation, DocumentRefs: in.DocumentRefs, TargetFingerprint: p.Fingerprint, ExpiresAt: time.Now().UTC().Add(15 * time.Minute), Phases: p.Phases, Changes: p.Changes, RequiredAccess: p.RequiredAccess, Resources: []string{}, ChainSurfaces: []any{}, Validation: map[string]any{"valid": true, "errors": []any{}, "warnings": []any{}}}
	for _, c := range p.Claims {
		if c.Executable != "" {
			plan.Resources = append(plan.Resources, c.HostIdentity+":process:"+c.Executable)
			continue
		}
		plan.Resources = append(plan.Resources, c.HostIdentity+":"+c.DataPath)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	err = s.commit(a.ID, "", "plan.created", func(next *webJobState) { next.Plans[plan.ID] = savedWebPlan{plan, in, p} })
	return plan, err
}

func jobBody(planID string) string {
	b := sha256.Sum256([]byte(planID))
	return hex.EncodeToString(b[:])
}

// Start atomically verifies plan freshness, acquires claims and records acceptance.
// A repeated actor/key/body returns the same job, including after restart.
func (s *WebJobs) Start(ctx context.Context, a DeploymentActor, planID, key string) (WebJob, error) {
	if err := s.allowed(a); err != nil {
		return WebJob{}, err
	}
	if strings.TrimSpace(key) == "" || len(key) > 128 {
		return WebJob{}, errors.New("idempotency key required (maximum 128 bytes)")
	}
	if err := ctx.Err(); err != nil {
		return WebJob{}, err
	}
	keyHash, body := jobBody(a.ID+":"+key), jobBody(planID)
	s.mu.Lock()
	p, replay, err := s.acceptanceLocked(a, planID, keyHash, body)
	s.mu.Unlock()
	if err != nil || replay.ID != "" {
		return replay, err
	}
	// Target inspection may dial SSH or check large inputs. It must never hold
	// the durable store lock needed by progress, cancellation or other targets.
	current, err := s.engine.Prepare(ctx, a, p.Input)
	if err != nil {
		return WebJob{}, err
	}
	if err = validatePrepared(current); err != nil {
		return WebJob{}, err
	}
	if current.Fingerprint != p.Prepared.Fingerprint || !sameClaims(current.Claims, p.Prepared.Claims) {
		return WebJob{}, ErrDeploymentConflict
	}
	if err = s.allowed(a); err != nil {
		return WebJob{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err = ctx.Err(); err != nil {
		return WebJob{}, err
	}
	// Another request may have accepted this key while inspection was running.
	// Freshness and exclusions are checked again in the same acceptance lock.
	latest, replay, err := s.acceptanceLocked(a, planID, keyHash, body)
	if err != nil || replay.ID != "" {
		return replay, err
	}
	if latest.Prepared.Fingerprint != current.Fingerprint || !sameClaims(latest.Prepared.Claims, current.Claims) {
		return WebJob{}, ErrDeploymentConflict
	}
	if s.resourceConflict(p.Input.WorkspaceID, p.Prepared.Claims) {
		return WebJob{}, ErrDeploymentConflict
	}
	job := WebJob{ID: deploymentID(), ActorID: a.ID, WorkspaceID: p.Input.WorkspaceID, PlanID: planID, Operation: p.Input.Operation, State: "accepted", Retention: p.Input.Retention, NodeDisposition: "unknown", CreatedAt: time.Now().UTC(), Phases: []WebJobPhase{}, RunIDs: []string{}, PartialEffects: []string{}, UnresolvedResources: []string{}}
	if err = s.commit(a.ID, job.ID, "job.accepted", func(next *webJobState) { next.Jobs[job.ID] = job; next.Keys[keyHash] = webJobKey{body, job.ID} }); err != nil {
		return WebJob{}, err
	}
	jobCtx, cancel := context.WithCancel(context.Background())
	s.cancels[job.ID] = cancel
	go s.execute(jobCtx, a, job.ID, p.Prepared)
	return detachedWebJob(job), nil
}

// acceptanceLocked reads only durable metadata; caller owns s.mu.
func (s *WebJobs) acceptanceLocked(a DeploymentActor, planID, keyHash, body string) (savedWebPlan, WebJob, error) {
	if old, ok := s.state.Keys[keyHash]; ok {
		if old.Body != body {
			return savedWebPlan{}, WebJob{}, ErrDeploymentConflict
		}
		return savedWebPlan{}, detachedWebJob(s.state.Jobs[old.JobID]), nil
	}
	p, ok := s.state.Plans[planID]
	if !ok || p.Public.ActorID != a.ID {
		return savedWebPlan{}, WebJob{}, ErrDeploymentNotFound
	}
	if !time.Now().Before(p.Public.ExpiresAt) || s.engine == nil {
		return savedWebPlan{}, WebJob{}, ErrDeploymentConflict
	}
	return p, WebJob{}, nil
}

func sameClaims(a, b []WebResourceClaim) bool {
	x, _ := json.Marshal(a)
	y, _ := json.Marshal(b)
	return string(x) == string(y)
}
func detachedWebJob(job WebJob) WebJob {
	b, _ := json.Marshal(job)
	var out WebJob
	_ = json.Unmarshal(b, &out)
	return out
}

func (s *WebJobs) Get(id string) (WebJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.state.Jobs[id]
	if !ok {
		return WebJob{}, ErrDeploymentNotFound
	}
	return detachedWebJob(job), nil
}
func (s *WebJobs) List() []WebJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	jobs := []WebJob{}
	for _, job := range s.state.Jobs {
		jobs = append(jobs, detachedWebJob(job))
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].CreatedAt.After(jobs[j].CreatedAt) })
	return jobs
}
