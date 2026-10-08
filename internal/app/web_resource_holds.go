package app

import (
	"path"
	"sort"
	"time"
)

// WebResourceConflict exposes ownership metadata without accepted private inputs.
type WebResourceConflict struct {
	JobID           string             `json:"jobId"`
	WorkspaceID     string             `json:"workspaceId"`
	ActorID         string             `json:"actorId"`
	Operation       string             `json:"operation"`
	State           string             `json:"state"`
	NodeDisposition string             `json:"nodeDisposition"`
	Resources       []WebResourceClaim `json:"resources"`
}

// PlanConflicts is an actor-bound snapshot. Start still checks exclusions
// atomically; a review is not a reservation and cannot authorize execution.
func (s *WebJobs) PlanConflicts(a DeploymentActor, planID string) ([]WebResourceConflict, error) {
	if err := s.allowed(a); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	plan, ok := s.state.Plans[planID]
	if !ok || plan.Public.ActorID != a.ID {
		return nil, ErrDeploymentNotFound
	}
	if !time.Now().Before(plan.Public.ExpiresAt) {
		return nil, ErrDeploymentConflict
	}
	return s.resourceConflicts(plan.Input.WorkspaceID, plan.Prepared.Claims), nil
}

// resourceConflict runs under the job-store lock. Terminal jobs may still own
// live nodes or partial files; verified cleanup is required for release.
func (s *WebJobs) resourceConflict(workspace string, claims []WebResourceClaim) bool {
	return len(s.resourceConflicts(workspace, claims)) > 0
}

func (s *WebJobs) resourceConflicts(workspace string, claims []WebResourceClaim) []WebResourceConflict {
	conflicts := []WebResourceConflict{}
	accepted := map[string]int{}
	for i, event := range s.state.Audit {
		if event.Operation == "job.accepted" {
			if _, exists := accepted[event.JobID]; !exists {
				accepted[event.JobID] = i
			}
		}
	}
	for _, job := range s.state.Jobs {
		terminal := terminalWebJob(job.State)
		if terminal && (job.WorkspaceID == workspace || verifiedWebCleanup(job)) {
			continue
		}
		resources := []WebResourceClaim{}
		for _, held := range s.state.Plans[job.PlanID].Prepared.Claims {
			if terminal && s.claimCleaned(job, held, accepted) {
				continue
			}
			for _, requested := range claims {
				if claimsOverlap(requested, held) {
					held.Ports = append([]int(nil), held.Ports...)
					resources = append(resources, held)
					break
				}
			}
		}
		if len(resources) > 0 {
			conflicts = append(conflicts, WebResourceConflict{JobID: job.ID, WorkspaceID: job.WorkspaceID, ActorID: job.ActorID, Operation: job.Operation, State: job.State, NodeDisposition: job.NodeDisposition, Resources: resources})
		}
	}
	sort.Slice(conflicts, func(i, j int) bool { return conflicts[i].JobID < conflicts[j].JobID })
	return conflicts
}

func verifiedWebCleanup(job WebJob) bool {
	return terminalWebJob(job.State) && job.NodeDisposition == "cleaned" && len(job.UnresolvedResources) == 0
}

func (s *WebJobs) claimCleaned(owner WebJob, held WebResourceClaim, accepted map[string]int) bool {
	ownerOrder, known := accepted[owner.ID]
	if !known {
		return false // A missing durable ordering record cannot prove release.
	}
	for _, cleanup := range s.state.Jobs {
		cleanupOrder, known := accepted[cleanup.ID]
		if !known || cleanupOrder <= ownerOrder || cleanup.WorkspaceID != owner.WorkspaceID || !verifiedWebCleanup(cleanup) {
			continue
		}
		for _, released := range s.state.Plans[cleanup.PlanID].Prepared.Claims {
			if claimCleanupCovers(held, released) {
				return true
			}
		}
	}
	return false
}

func claimCleanupCovers(held, released WebResourceClaim) bool {
	if held.HostIdentity != released.HostIdentity {
		return false
	}
	if held.DataPath != "" {
		// A shared port or an ancestor path does not establish that this owned
		// network was removed. Its exact resolved root must have been cleaned.
		return released.DataPath != "" && path.Clean(held.DataPath) == path.Clean(released.DataPath)
	}
	for _, port := range held.Ports {
		found := false
		for _, cleanedPort := range released.Ports {
			if port == cleanedPort {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return len(held.Ports) > 0
}
