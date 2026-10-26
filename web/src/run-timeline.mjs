// Pure helpers behind the run timeline: which tests failed and how, where each
// one sits in time, and which log span answers "what happened then".

const FAILED = ['failed', 'blocked']
// session.json counts verdicts as pass/fail/blocked/skip; the bar speaks the
// run states the tests themselves carry.
const VERDICTS = [['fail', 'failed'], ['blocked', 'blocked'], ['pass', 'succeeded'], ['skip', 'skipped']]

const at = value => (value ? Date.parse(value) : NaN)

// testRows turns a run detail's tests (already read from the engine's result
// files by the server) into one row per test, failed and blocked tests first,
// each carrying the assertions and steps that broke.
export function testRows(run) {
 const rows = (run?.tests ?? []).map(test => {
  const failedAsserts = (test.assertions ?? []).filter(a => a.passed === false).map(a => ({ assert: a.assert, on: a.nodeId ?? '', compare: a.compare ?? '', source: a.detail ?? '', expected: a.expected, actual: a.actual ?? '' }))
  const stepErrors = (test.steps ?? []).filter(s => s.error).map(s => ({ index: s.index, type: s.type, on: s.nodeId ?? '', error: s.error }))
  const start = at(test.startedAt), end = at(test.finishedAt)
  return {
   seq: test.seq,
   id: test.id,
   status: test.state,
   startedAt: test.startedAt ?? null,
   finishedAt: test.finishedAt ?? null,
   durationMs: Number.isFinite(start) && Number.isFinite(end) ? end - start : null,
   reason: test.reason ?? '',
   failedAsserts,
   stepErrors,
   nodes: [...new Set([...failedAsserts, ...stepErrors].map(x => x.on).filter(Boolean))].sort()
  }
 })
 const rank = row => (FAILED.includes(row.status) ? 0 : 1)
 return rows.sort((a, b) => rank(a) - rank(b) || a.seq - b.seq)
}

export function verdictSegments(counts) {
 const total = VERDICTS.reduce((sum, [key]) => sum + (counts?.[key] ?? 0), 0)
 if (!total) return []
 return VERDICTS.filter(([key]) => (counts?.[key] ?? 0) > 0).map(([key, status]) => ({ status, count: counts[key], percent: Math.round((counts[key] / total) * 1000) / 10 }))
}

// placeOnAxis gives each timed row its left offset and width as percentages of
// the from–to axis. Untimed rows keep left and width null.
export function placeOnAxis(rows, from, to) {
 const start = at(from), span = at(to) - start
 return rows.map(row => {
  const s = at(row.startedAt), e = at(row.finishedAt)
  if (!Number.isFinite(s) || !Number.isFinite(e) || !(span > 0)) return { ...row, left: null, width: null }
  const left = Math.max(0, Math.min(100, ((s - start) / span) * 100))
  const width = Math.max(0.5, Math.min(100 - left, ((e - s) / span) * 100))
  return { ...row, left: Math.round(left * 10) / 10, width: Math.round(width * 10) / 10 }
 })
}

// testLogSpan is the log window for one test: its own span widened by pad
// seconds, or, for a record without test times, no window of its own.
export function testLogSpan(row, padSeconds) {
 const s = at(row?.startedAt), e = at(row?.finishedAt)
 if (!Number.isFinite(s) || !Number.isFinite(e)) return { from: null, to: null, exact: false }
 return { from: new Date(s - padSeconds * 1000).toISOString(), to: new Date(e + padSeconds * 1000).toISOString(), exact: true }
}

// phaseSegments turns job phases into shares of the timed total; a phase that
// has not run (or has no times) gets share 0 and timed false.
export function phaseSegments(phases) {
 const timed = (phases ?? []).map(p => {
  const d = at(p.finishedAt) - at(p.startedAt)
  return { ...p, durationMs: Number.isFinite(d) && d >= 0 ? d : null }
 })
 const total = timed.reduce((sum, p) => sum + (p.durationMs ?? 0), 0)
 return timed.map(p => ({ ...p, timed: p.durationMs !== null, share: total && p.durationMs !== null ? Math.round((p.durationMs / total) * 1000) / 10 : 0 }))
}

const INTEGER = /^-?\d+$/

// assertDiff is actual minus expected when both are integers, signed, exact
// for any size; otherwise ''. Two long numbers then differ at a glance.
export function assertDiff(expected, actual) {
 const e = String(expected ?? ''), a = String(actual ?? '')
 if (!INTEGER.test(e) || !INTEGER.test(a)) return ''
 const d = BigInt(a) - BigInt(e)
 return d === 0n ? '' : d > 0n ? `+${d}` : String(d)
}
