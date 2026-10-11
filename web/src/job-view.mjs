const terminal = ['succeeded', 'failed', 'cancelled', 'interrupted']

// jobSummary reads what a job detail shows first: whether it is still active,
// the phase that failed and why, the changes it already made, and whether its
// machines must be checked because the outcome on them is unknown.
export function jobSummary(job) {
 const failedPhase = (job?.phases ?? []).find(p => p.state === 'failed') ?? null
 return {
  active: !terminal.includes(job?.state),
  failedPhase,
  failure: job?.error?.message ?? '',
  effects: job?.partialEffects ?? [],
  unresolved: job?.unresolvedResources ?? [],
  needsCheck: job?.state === 'interrupted' || (terminal.includes(job?.state) && job?.nodeDisposition === 'unknown'),
 }
}
