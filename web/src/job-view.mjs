import { testRows } from './run-timeline.mjs'

// jobSummary reads what a job detail shows first: whether it is still active,
// the phase that failed and why, the changes it already made, and whether its
// machines must be checked because the outcome on them is unknown.
export function jobSummary(job) {
 const failedPhase = (job?.phases ?? []).find(p => p.state === 'failed') ?? null
 return {
  active: !job?.terminal,
  failedPhase,
  failure: job?.error?.message ?? '',
  effects: job?.partialEffects ?? [],
  unresolved: job?.unresolvedResources ?? [],
  needsCheck: Boolean(job?.needsCheck),
 }
}

const FAILED_TESTS = ['failed', 'blocked']

// set reads a recorded time; the server writes an unset time as year 1.
const set = value => {
 const t = Date.parse(value ?? '')
 return Number.isFinite(t) && t > 0 ? t : null
}

const shown = value => (typeof value === 'string' ? value : JSON.stringify(value))

// failureHeadline is the one sentence the job detail opens with: the failing
// test, the node it broke on and the evidence, or else the failed phase, or
// what the job is doing or how it ended. Each test failure carries the place
// (phase, test, node, run) a reader jumps to next. While the job's test runs
// are still being read (pending) a failed job claims nothing yet, so the first
// line never names a phase only to be replaced by the test a moment later.
export function failureHeadline(job, summary, failingRuns, pending = false) {
 const base = { tone: 'fail', title: '', reason: '', phase: summary?.failedPhase?.name ?? '', test: '', node: '', runId: '' }
 if (pending && !failingRuns?.length && job?.terminal) return { ...base, tone: 'pending', title: '테스트 판정을 읽는 중입니다', phase: '' }
 for (const run of failingRuns ?? []) {
  const failure = testFailure(run)
  if (failure) return { ...base, ...failure }
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

// testFailure is the headline of a run's first failing test: the test, the
// node it broke on and the evidence, or null when no test failed.
function testFailure(run) {
 const row = testRows(run).find(r => FAILED_TESTS.includes(r.status))
 if (!row) return null
 const a = row.failedAsserts[0], s = row.stepErrors[0]
 const node = a?.on || s?.on || ''
 const reason = a ? `${a.assert} · 기대 ${shown(a.expected)} · 실제 ${shown(a.actual)}` : s ? `#${s.index} ${s.type} · ${s.error}` : row.reason
 return { title: `${row.id} 테스트가 ${node ? node + '에서 ' : ''}실패했습니다`, reason, test: row.id, node, runId: run.id }
}

// runHeadline is the same first sentence for a recorded run read on its own
// (the history detail): the failing test, or how the run ended. The run is the
// page itself, so the headline does not name it again.
export function runHeadline(run) {
 const base = { tone: 'fail', title: '', reason: '', phase: '', test: '', node: '', runId: '' }
 const failure = testFailure(run)
 if (failure) return { ...base, ...failure, runId: '' }
 // A job's run records its phases and the engine's refusal instead of verdicts.
 const error = run?.summary?.error?.message ?? ''
 const failedPhase = (run?.summary?.phases ?? []).find(p => p.state === 'failed')
 if (failedPhase) return { ...base, title: `${failedPhase.name} 단계에서 실패했습니다`, reason: failedPhase.message || error, phase: failedPhase.name }
 switch (run?.state) {
  case 'succeeded': return { ...base, tone: 'ok', title: '실행이 성공했습니다' }
  case 'cancelled': return { ...base, tone: 'warn', title: '실행이 취소되었습니다', reason: error }
  case 'interrupted': return { ...base, tone: 'warn', title: '실행이 중단되었습니다', reason: error || '서버가 실행을 끝까지 지켜보지 못했습니다.' }
  case 'failed': return { ...base, title: '실행이 실패했습니다', reason: error }
 }
 return { ...base, tone: 'active', title: '실행 중입니다' }
}

// benchTest is the test a run is read on: the one asked for, else the first
// failing test, else the first test, so a passing run can be read too.
export function benchTest(rows, wanted) {
 return rows.find(r => r.id === wanted) ?? rows.find(r => FAILED_TESTS.includes(r.status)) ?? rows[0] ?? null
}

// jobDuration runs from the job's creation to the latest phase end, or null
// when either end was never recorded.
export function jobDuration(job) {
 const start = set(job?.createdAt)
 const ends = (job?.phases ?? []).map(p => set(p.finishedAt)).filter(t => t !== null)
 return start !== null && ends.length ? Math.max(...ends) - start : null
}

// jobElapsed is how long a running job has run by now (ms); a browser clock
// behind the server reads as zero, never as negative time.
export function jobElapsed(job, now) {
 const start = set(job?.createdAt)
 return start === null ? null : Math.max(0, now - start)
}

// phaseRows is one row per phase with unset times cleared and its length;
// only the failed phase starts open.
export function phaseRows(phases) {
 return (phases ?? []).map(p => {
  const s = set(p.startedAt), e = set(p.finishedAt)
  return { ...p, startedAt: s === null ? null : p.startedAt, finishedAt: e === null ? null : p.finishedAt, durationMs: s !== null && e !== null && e >= s ? e - s : null, open: p.state === 'failed' }
 })
}

// A job or history detail URL keeps the chosen run, test, log node and
// cursor moment, so a copied link opens the same evidence.
const FOCUS = ['run', 'test', 'node', 'at']
export function readFocus(search) {
 const q = new URLSearchParams(search ?? '')
 const out = Object.fromEntries(FOCUS.map(k => [k, q.get(k) ?? '']))
 return { ...out, at: Number.isFinite(Date.parse(out.at)) ? out.at : '' }
}

// focusSearch writes the focus over search, keeping the page's other
// parameters (the history search) as they were.
export function focusSearch(focus, search = '') {
 const q = new URLSearchParams(search)
 for (const k of FOCUS) if (focus[k]) q.set(k, focus[k]); else q.delete(k)
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
