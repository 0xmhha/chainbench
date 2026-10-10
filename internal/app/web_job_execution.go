package app

import (
	"context"
	"errors"
	"time"
)

func (s *WebJobs) sanitized(text string) string {
	if s.redact != nil {
		return s.redact(text)
	}
	return RedactWebText(text)
}

func (s *WebJobs) progress(id string, phase WebJobPhase) error {
	phase.Message = s.sanitized(phase.Message)
	s.mu.Lock()
	defer s.mu.Unlock()
	job := s.state.Jobs[id]
	if job.State == "cancelling" || terminalWebJob(job.State) {
		return context.Canceled
	}
	allowed := false
	for _, name := range s.state.Plans[job.PlanID].Prepared.Phases {
		if name == phase.Name {
			allowed = true
		}
	}
	if !allowed || (phase.State != "running" && phase.State != "succeeded" && phase.State != "failed") {
		return errors.New("invalid engine phase")
	}
	return s.commit(job.ActorID, id, "job.phase", func(next *webJobState) {
		v := next.Jobs[id]
		for i, old := range v.Phases {
			if old.Name == phase.Name {
				v.Phases[i] = phase
				next.Jobs[id] = v
				return
			}
		}
		v.Phases = append(v.Phases, phase)
		next.Jobs[id] = v
	})
}

func (s *WebJobs) execute(ctx context.Context, actor DeploymentActor, id string, prepared WebPreparedJob) {
	s.mu.Lock()
	job := s.state.Jobs[id]
	err := s.commit(actor.ID, id, "job.running", func(next *webJobState) {
		v := next.Jobs[id]
		if v.State == "accepted" {
			v.State = "running"
		}
		next.Jobs[id] = v
	})
	s.mu.Unlock()
	if err != nil {
		return
	} // Keep the durable claim held; never execute unrecorded work.
	var result WebJobResult
	if job.State == "cancelling" || ctx.Err() != nil {
		err = context.Canceled
	} else if err = s.allowed(actor); err == nil {
		result, err = s.engine.Execute(ctx, actor, prepared, func(p WebJobPhase) error { return s.progress(id, p) })
	}
	// Clean only when requested and still authorized. Revocation always retains.
	s.mu.Lock()
	job = s.state.Jobs[id]
	clean := job.Retention == "cleanup" && (job.CancelReason == "" || job.CancelReason == "user")
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 2*time.Minute)
	if clean {
		if old := s.cancels[id]; old != nil {
			old()
		}
		s.cancels[id] = cleanupCancel
	}
	s.mu.Unlock()
	if clean && s.allowed(actor) == nil {
		cleaned, cleanErr := s.engine.Cleanup(cleanupCtx, actor, prepared)
		result.PartialEffects = append(result.PartialEffects, cleaned.PartialEffects...)
		result.UnresolvedResources = append(result.UnresolvedResources, cleaned.UnresolvedResources...)
		result.NodeDisposition = cleaned.NodeDisposition
		if cleanErr != nil {
			result.NodeDisposition = "cleanup_failed"
			err = errors.Join(err, cleanErr)
		}
	}
	cleanupCancel()
	for i, effect := range result.PartialEffects {
		result.PartialEffects[i] = s.sanitized(effect)
	}
	for i, resource := range result.UnresolvedResources {
		result.UnresolvedResources[i] = s.sanitized(resource)
	}
	message := ""
	if err != nil {
		message = s.sanitized(err.Error())
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	job = s.state.Jobs[id]
	terminal := "succeeded"
	if err != nil {
		terminal = "failed"
	}
	if job.CancelReason != "" {
		terminal = "cancelled"
	}
	if job.CancelReason == "credential_revoked" || job.CancelReason == "account_revoked" {
		result.NodeDisposition = "retained"
	}
	if result.NodeDisposition == "" {
		result.NodeDisposition = "unknown"
	}
	if err = s.commit(actor.ID, id, "job."+terminal, func(next *webJobState) {
		v := next.Jobs[id]
		v.State, v.NodeDisposition = terminal, result.NodeDisposition
		for i, phase := range v.Phases {
			if phase.State == "running" {
				v.Phases[i].State = terminal
				v.Phases[i].FinishedAt = time.Now().UTC()
			}
		}
		v.RunIDs = append(v.RunIDs, result.RunIDs...)
		v.PartialEffects = append(v.PartialEffects, result.PartialEffects...)
		v.UnresolvedResources = append(v.UnresolvedResources, result.UnresolvedResources...)
		if message != "" {
			v.Error = map[string]string{"code": "engine_error", "message": message}
		}
		next.Jobs[id] = v
	}); err == nil {
		if cancel := s.cancels[id]; cancel != nil {
			cancel()
		}
		delete(s.cancels, id)
	}
}

// Cancel is limited to the initiator or administrator. Logout does not call it.
func (s *WebJobs) Cancel(a DeploymentActor, id string) (WebJob, error) {
	if err := s.allowed(a); err != nil {
		return WebJob{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	job, ok := s.state.Jobs[id]
	if !ok {
		return WebJob{}, ErrDeploymentNotFound
	}
	if job.ActorID != a.ID && a.Role != "admin" {
		return WebJob{}, ErrDeploymentForbidden
	}
	if terminalWebJob(job.State) {
		return detachedWebJob(job), nil
	}
	if err := s.cancelLocked(a.ID, id, "user"); err != nil {
		return WebJob{}, err
	}
	return detachedWebJob(s.state.Jobs[id]), nil
}

func (s *WebJobs) cancelLocked(actor, id, reason string) error {
	if err := s.commit(actor, id, "job.cancel_requested", func(next *webJobState) {
		job := next.Jobs[id]
		job.State = "cancelling"
		// Ordinary cancellation must never downgrade an earlier revocation.
		if job.CancelReason == "" || reason != "user" {
			job.CancelReason = reason
		}
		if reason != "user" {
			job.Retention = "retain"
		}
		next.Jobs[id] = job
	}); err != nil {
		return err
	}
	if cancel := s.cancels[id]; cancel != nil {
		cancel()
	}
	return nil
}

// RevokeActor cancels active work without attempting remote cleanup.
func (s *WebJobs) RevokeActor(id string) error { return s.revoke(id, "", "account_revoked") }

// RevokeCredential cancels only the owner's jobs using this credential binding.
func (s *WebJobs) RevokeCredential(owner, id string) error {
	return s.revoke(owner, id, "credential_revoked")
}
func (s *WebJobs) revoke(owner, credential, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, job := range s.state.Jobs {
		if job.ActorID != owner || terminalWebJob(job.State) {
			continue
		}
		uses := credential == ""
		input := s.state.Plans[job.PlanID].Input
		for _, ref := range input.CredentialBindings {
			if ref == credential {
				uses = true
			}
		}
		for _, ref := range input.AccountBindings {
			if ref == credential {
				uses = true
			}
		}
		if uses {
			if err := s.cancelLocked("system", id, reason); err != nil {
				return err
			}
		}
	}
	return nil
}
