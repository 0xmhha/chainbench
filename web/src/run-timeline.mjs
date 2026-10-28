// Pure helpers for reading a recorded run: which tests failed and how, and
// which log span answers "what happened then".

const FAILED = ['failed', 'blocked']

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

// testLogSpan is the log window for one test: its own span widened by pad
// seconds, or, for a record without test times, no window of its own.
export function testLogSpan(row, padSeconds) {
 const s = at(row?.startedAt), e = at(row?.finishedAt)
 if (!Number.isFinite(s) || !Number.isFinite(e)) return { from: null, to: null, exact: false }
 return { from: new Date(s - padSeconds * 1000).toISOString(), to: new Date(e + padSeconds * 1000).toISOString(), exact: true }
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
