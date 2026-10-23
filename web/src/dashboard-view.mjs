// Pure helpers behind the dashboard: how the nodes split by state and which
// finished jobs need someone to look at them.

const ATTENTION = ['failed', 'interrupted']
const SHORT_ID = 8

export function nodeTally(nodes) {
 const all = nodes ?? []
 const running = all.filter(n => n?.state === 'running').length
 const stopped = all.filter(n => n?.state === 'stopped').length
 return { running, stopped, other: all.length - running - stopped, total: all.length }
}

// attentionJobs lists failed and interrupted jobs, newest first, each with the
// phase that failed and the reason the server gave.
export function attentionJobs(jobs, limit) {
 return (jobs ?? [])
  .filter(j => ATTENTION.includes(j.state))
  .sort((a, b) => Date.parse(b.createdAt) - Date.parse(a.createdAt))
  .slice(0, limit)
  .map(j => ({ ...j, failedPhase: (j.phases ?? []).find(p => p.state === 'failed') ?? null, reason: j.error?.message ?? '' }))
}

export function shortId(id) {
 return id && id.length > SHORT_ID + 4 ? id.slice(0, SHORT_ID) + '…' : id
}
