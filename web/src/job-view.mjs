import { testRows } from './run-timeline.mjs'

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

const FAILED_TESTS = ['fail', 'blocked']

// set reads a recorded time; the server writes an unset time as year 1.
const set = value => {
 const t = Date.parse(value ?? '')
 return Number.isFinite(t) && t > 0 ? t : null
}

const shown = value => (typeof value === 'string' ? value : JSON.stringify(value))

// failureHeadline is the one sentence the job detail opens with: the failing
// test, the node it broke on and the evidence, or else the failed phase, or
// what the job is doing or how it ended. Each test failure carries the place
// (phase, test, node, run) a reader jumps to next.
export function failureHeadline(job, summary, failingRuns) {
 const base = { tone: 'fail', title: '', reason: '', phase: summary?.failedPhase?.name ?? '', test: '', node: '', runId: '' }
 for (const run of failingRuns ?? []) {
  const row = testRows(run.summary).find(r => FAILED_TESTS.includes(r.status))
  if (!row) continue
  const a = row.failedAsserts[0], s = row.stepErrors[0]
  const node = a?.on || s?.on || ''
  const reason = a ? `${a.assert ?? a.id} · 기대 ${shown(a.expected)} · 실제 ${shown(a.actual)}` : s ? `#${s.index} ${s.type} · ${s.error}` : row.reason
  return { ...base, title: `${row.id} 테스트가 ${node ? node + '에서 ' : ''}실패했습니다`, reason, test: row.id, node, runId: run.id }
 }
 if (summary?.failedPhase) return { ...base, title: `${summary.failedPhase.name} 단계에서 실패했습니다`, reason: summary.failedPhase.message || summary.failure }
 switch (job?.state) {
  case 'succeeded': return { ...base, tone: 'ok', title: '작업이 성공했습니다' }
  case 'cancelled': return { ...base, tone: 'warn', title: '작업이 취소되었습니다', reason: [job.cancelReason, job.cancelMessage].filter(Boolean).join(' · ') || summary?.failure }
  case 'interrupted': return { ...base, tone: 'warn', title: '작업이 중단되었습니다', reason: summary?.failure || '서버가 작업을 끝까지 지켜보지 못했습니다. 노드의 실제 상태를 확인해야 합니다.' }
  case 'failed': return { ...base, title: '작업이 실패했습니다', reason: summary?.failure }
 }
 const current = (job?.phases ?? []).find(p => p.state === 'running')
 return { ...base, tone: 'active', title: current ? `${current.name} 단계를 실행하고 있습니다` : '작업을 기다리고 있습니다' }
}

// jobDuration runs from the job's creation to the latest phase end, or null
// when either end was never recorded.
export function jobDuration(job) {
 const start = set(job?.createdAt)
 const ends = (job?.phases ?? []).map(p => set(p.finishedAt)).filter(t => t !== null)
 return start !== null && ends.length ? Math.max(...ends) - start : null
}

// phaseRows is one row per phase with unset times cleared and its length;
// only the failed phase starts open.
export function phaseRows(phases) {
 return (phases ?? []).map(p => {
  const s = set(p.startedAt), e = set(p.finishedAt)
  return { ...p, startedAt: s === null ? null : p.startedAt, finishedAt: e === null ? null : p.finishedAt, durationMs: s !== null && e !== null && e >= s ? e - s : null, open: p.state === 'failed' }
 })
}

// The job detail URL keeps the chosen test and log node so a link opens the
// same evidence.
export function readJobFocus(search) {
 const q = new URLSearchParams(search ?? '')
 return { test: q.get('test') ?? '', node: q.get('node') ?? '' }
}

export function jobFocusSearch({ test, node }) {
 const q = new URLSearchParams()
 if (test) q.set('test', test)
 if (node) q.set('node', node)
 const text = q.toString()
 return text ? '?' + text : ''
}

// nodeInMessage is the one network node a failure message names, or '' when
// it names none or several. A node name only counts as a whole word.
export function nodeInMessage(message, nodes) {
 const words = new Set(String(message ?? '').split(/[^A-Za-z0-9_.-]+/).map(w => w.replace(/[.]+$/, '')))
 const named = (nodes ?? []).filter(n => words.has(n))
 return named.length === 1 ? named[0] : ''
}
