package app

import "path"

// resourceConflict runs under the job-store lock. A terminal execution can still
// own live nodes or partially written files; only verified cleanup releases them.
func (s *WebJobs) resourceConflict(workspace string, claims []WebResourceClaim) bool {
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
		for _, held := range s.state.Plans[job.PlanID].Prepared.Claims {
			if terminal && s.claimCleaned(job, held, accepted) {
				continue
			}
			for _, requested := range claims {
				if claimsOverlap(requested, held) {
					return true
				}
			}
		}
	}
	return false
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
