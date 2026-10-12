package app

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/0xmhha/chainbench/internal/core/node"
	"github.com/0xmhha/chainbench/internal/resource"
)

// WebReconciliation is what a read-only check found on the machines of an
// interrupted job. Outcome is absent when no node of its network exists,
// present when every recorded node was checked and some exist, and
// unverified when any check could not be made.
type WebReconciliation struct {
	CheckedAt time.Time      `json:"checkedAt"`
	Outcome   string         `json:"outcome"`
	Nodes     []WebNodeCheck `json:"nodes"`
	Message   string         `json:"message,omitempty"`
}

// WebNodeCheck is one recorded node's observed process and data directory:
// running, stopped or unknown; present, absent or unknown.
type WebNodeCheck struct {
	NodeID  string `json:"nodeId"`
	Process string `json:"process"`
	Data    string `json:"data"`
}

// webJobReconciler checks an accepted plan's machines without changing them.
type webJobReconciler interface {
	Reconcile(context.Context, DeploymentActor, WebPreparedJob) (WebReconciliation, error)
}

// Reconcile checks the actual state behind an interrupted or unresolved job,
// as its initiator or an administrator, and records the result. Confirmed
// absence releases the job's claims; existing nodes are kept as a retained
// network; anything unchecked stays unresolved and locked.
func (s *WebJobs) Reconcile(ctx context.Context, a DeploymentActor, id string) (WebJob, error) {
	if err := s.allowed(a); err != nil {
		return WebJob{}, err
	}
	s.mu.Lock()
	job, ok := s.state.Jobs[id]
	prepared := s.state.Plans[job.PlanID].Prepared
	s.mu.Unlock()
	if !ok {
		return WebJob{}, ErrDeploymentNotFound
	}
	if job.ActorID != a.ID && a.Role != "admin" {
		return WebJob{}, ErrDeploymentForbidden
	}
	if !terminalWebJob(job.State) || job.NodeDisposition != "unknown" && job.State != "interrupted" {
		return WebJob{}, errors.New("only an interrupted or unresolved finished job is reconciled")
	}
	r, ok := s.engine.(webJobReconciler)
	if !ok {
		return WebJob{}, errors.New("this server cannot check the job's machines")
	}
	result, err := r.Reconcile(ctx, a, prepared)
	if err != nil {
		result = WebReconciliation{Outcome: "unverified", Message: s.sanitized(err.Error()), Nodes: []WebNodeCheck{}}
	}
	result.CheckedAt = time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	if err = s.commit(a.ID, id, "job.reconciled", func(next *webJobState) {
		v := next.Jobs[id]
		v.Reconciliation = &result
		switch result.Outcome {
		case "absent":
			v.NodeDisposition, v.UnresolvedResources = "cleaned", []string{}
		case "present":
			v.NodeDisposition, v.UnresolvedResources = "retained", []string{}
		default:
			v.NodeDisposition = "unknown"
		}
		next.Jobs[id] = v
	}); err != nil {
		return WebJob{}, err
	}
	return detachedWebJob(s.state.Jobs[id]), nil
}

// Reconcile observes each recorded node of the plan's network on its own
// machine with the actor's bindings: its process by recorded identity and
// its data directory. It starts, stops and removes nothing.
func (e *WebChainEngine) Reconcile(ctx context.Context, a DeploymentActor, prepared WebPreparedJob) (WebReconciliation, error) {
	out := WebReconciliation{Nodes: []WebNodeCheck{}}
	var p webChainPayload
	if err := json.Unmarshal(prepared.Payload, &p); err != nil {
		return out, err
	}
	if err := e.allowed(a); err != nil {
		return out, err
	}
	raw, err := os.ReadFile(filepath.Join(p.ControlDir, "chain-record.json"))
	if os.IsNotExist(err) {
		out.Outcome, out.Message = "absent", "no node was recorded for this network"
		return out, nil
	}
	if err != nil {
		return out, err
	}
	var state State
	if err = json.Unmarshal(raw, &state); err != nil {
		return out, err
	}
	lookup, err := e.documents.jobCredentialLookup(ctx, a, p.Set.DeploymentDocumentInput, p.Input.CredentialBindings)
	if err != nil {
		return out, err
	}
	opener := resource.Opener{Lookup: lookup}
	present, unknown := false, false
	for _, ns := range state.Nodes {
		check := WebNodeCheck{NodeID: string(ns.NodeLabel()), Process: "unknown", Data: "unknown"}
		spec := webNodeTarget(state, ns)
		access, err := opener.Open(spec)
		machine, inspectErr := opener.Inspect(ctx, spec)
		if err == nil && inspectErr == nil {
			// The node's data directory or the configuration a deploy stage
			// already wrote: either one is this job's, left on the target.
			check.Data = webNodeFiles(ctx, access.Files, ns)
			if observer := webObserverFor(access, machine); observer != nil {
				binary := state.Binary
				if ns.Binary != "" {
					binary = state.Binaries[ns.Binary]
				}
				observed := observeWebNodeVacancy(ctx, observer, binary, ns)
				if ns.PID > 0 {
					observed = observeWebNodeProcess(ctx, observer, binary, ns)
				}
				if observed.State == "running" || observed.State == "stopped" {
					check.Process = observed.State
				}
			}
		}
		if check.Process == "unknown" || check.Data == "unknown" {
			unknown = true
		}
		if check.Process == "running" || check.Data == "present" {
			present = true
		}
		out.Nodes = append(out.Nodes, check)
	}
	switch {
	case unknown:
		out.Outcome, out.Message = "unverified", "some nodes could not be checked; their resources stay locked"
	case present:
		out.Outcome = "present"
	default:
		out.Outcome = "absent"
	}
	return out, nil
}

// ReconcilesLocally reports whether a plan's machines are this server's own,
// so they can be checked at startup without anyone's SSH binding.
func (e *WebChainEngine) ReconcilesLocally(prepared WebPreparedJob) bool {
	var p webChainPayload
	if json.Unmarshal(prepared.Payload, &p) != nil || len(p.Targets) > 0 {
		return false
	}
	return p.Target.Transport == "local"
}

type webLocalReconciler interface {
	ReconcilesLocally(WebPreparedJob) bool
}

// ReconcileInterrupted checks, once after a restart, every interrupted job
// whose machines are local, as that job's initiator. Jobs on SSH machines
// wait for their initiator or an administrator to ask, because that check
// uses a personal binding.
func (s *WebJobs) ReconcileInterrupted(ctx context.Context) {
	local, ok := s.engine.(webLocalReconciler)
	if !ok {
		return
	}
	s.mu.Lock()
	type candidate struct{ id, actor string }
	var due []candidate
	for id, job := range s.state.Jobs {
		if job.State == "interrupted" && job.Reconciliation == nil && local.ReconcilesLocally(s.state.Plans[job.PlanID].Prepared) {
			due = append(due, candidate{id, job.ActorID})
		}
	}
	s.mu.Unlock()
	for _, c := range due {
		if ctx.Err() != nil {
			return
		}
		// The initiator's current account decides; a revoked one is skipped.
		_, _ = s.Reconcile(ctx, DeploymentActor{ID: c.actor, Role: "operator"}, c.id)
	}
}

// webNodeFiles reports whether anything a job placed for this node is on its
// machine: present when either its data directory or its configuration file
// exists, absent when neither does, unknown when either cannot be checked.
func webNodeFiles(ctx context.Context, files interface {
	Exists(context.Context, string) (bool, error)
}, ns node.Record) string {
	data := "absent"
	for _, path := range []string{ns.DataDir, ns.ConfigPath} {
		if path == "" {
			continue
		}
		exists, err := files.Exists(ctx, path)
		if err != nil {
			return "unknown"
		}
		if exists {
			data = "present"
		}
	}
	return data
}
